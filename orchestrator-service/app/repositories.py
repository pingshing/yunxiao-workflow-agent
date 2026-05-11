import json
import uuid
from datetime import datetime, timezone
from typing import Any

from psycopg import Connection
from psycopg.errors import UniqueViolation

from app.models import NormalizedEvent


def create_workflow_task(connection: Connection, event: NormalizedEvent, workflow_type: str) -> tuple[str, bool]:
    task_id = f"task_{uuid.uuid4()}"
    now = datetime.now(timezone.utc).replace(tzinfo=None)

    with connection.cursor() as cursor:
        try:
            cursor.execute(
                """
                INSERT INTO workflow_task
                  (task_id, event_id, event_type, workflow_type, status, retry_count, trace_id, created_at, updated_at)
                VALUES
                  (%s, %s, %s, %s, 'running', 0, %s, %s, %s)
                """,
                (
                    task_id,
                    event["event_id"],
                    event["event_type"],
                    workflow_type,
                    event["trace_id"],
                    now,
                    now,
                ),
            )
            return task_id, True
        except UniqueViolation:
            connection.rollback()
            with connection.cursor() as cursor:
                cursor.execute(
                    """
                    SELECT task_id FROM workflow_task
                    WHERE event_id = %s AND workflow_type = %s
                    """,
                    (event["event_id"], workflow_type),
                )
                row = cursor.fetchone()
                return row["task_id"], False


def save_context_snapshot(
    connection: Connection,
    event: NormalizedEvent,
    context_type: str,
    context: dict[str, Any],
    source_refs: dict[str, Any],
) -> str:
    snapshot_id = f"snap_{uuid.uuid4()}"
    subject = event.get("subject") or {}
    now = datetime.now(timezone.utc).replace(tzinfo=None)

    with connection.cursor() as cursor:
        cursor.execute(
            """
            INSERT INTO context_snapshot
              (snapshot_id, event_id, context_type, subject_type, subject_id, project_id, work_item_id,
               context_json, source_refs_json, created_at)
            VALUES
              (%s, %s, %s, %s, %s, %s, %s, %s::jsonb, %s::jsonb, %s)
            """,
            (
                snapshot_id,
                event["event_id"],
                context_type,
                subject.get("type", ""),
                subject.get("id", ""),
                event.get("project_id") or None,
                event.get("work_item_id") or None,
                json.dumps(context, ensure_ascii=False),
                json.dumps(source_refs, ensure_ascii=False),
                now,
            ),
        )
    return snapshot_id


def mark_task_succeeded(connection: Connection, task_id: str, snapshot_id: str) -> None:
    now = datetime.now(timezone.utc).replace(tzinfo=None)
    with connection.cursor() as cursor:
        cursor.execute(
            """
            UPDATE workflow_task
            SET status = 'succeeded', context_snapshot_id = %s, updated_at = %s
            WHERE task_id = %s
            """,
            (snapshot_id, now, task_id),
        )


def mark_task_failed(connection: Connection, task_id: str, error_message: str) -> None:
    now = datetime.now(timezone.utc).replace(tzinfo=None)
    with connection.cursor() as cursor:
        cursor.execute(
            """
            UPDATE workflow_task
            SET status = 'failed', error_message = %s, updated_at = %s
            WHERE task_id = %s
            """,
            (error_message[:4000], now, task_id),
        )
