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


def create_agent_run(
    connection: Connection,
    task_id: str,
    event: NormalizedEvent,
    workflow_type: str,
    agent_type: str,
    input_snapshot_id: str,
    model: str | None = None,
) -> str:
    run_id = f"run_{uuid.uuid4()}"
    now = datetime.now(timezone.utc).replace(tzinfo=None)

    with connection.cursor() as cursor:
        cursor.execute(
            """
            INSERT INTO agent_run
              (run_id, task_id, event_id, workflow_type, agent_type, status, model,
               input_snapshot_id, started_at, created_at, updated_at)
            VALUES
              (%s, %s, %s, %s, %s, 'running', %s, %s, %s, %s, %s)
            """,
            (
                run_id,
                task_id,
                event["event_id"],
                workflow_type,
                agent_type,
                model,
                input_snapshot_id,
                now,
                now,
                now,
            ),
        )
    return run_id


def mark_agent_run_succeeded(connection: Connection, run_id: str, raw_output: dict[str, Any]) -> None:
    now = datetime.now(timezone.utc).replace(tzinfo=None)
    with connection.cursor() as cursor:
        cursor.execute(
            """
            UPDATE agent_run
            SET status = 'succeeded',
                finished_at = %s,
                raw_output_json = %s::jsonb,
                updated_at = %s
            WHERE run_id = %s
            """,
            (now, json.dumps(raw_output, ensure_ascii=False), now, run_id),
        )


def mark_agent_run_failed(connection: Connection, run_id: str, error_message: str) -> None:
    now = datetime.now(timezone.utc).replace(tzinfo=None)
    with connection.cursor() as cursor:
        cursor.execute(
            """
            UPDATE agent_run
            SET status = 'failed',
                finished_at = %s,
                error_message = %s,
                updated_at = %s
            WHERE run_id = %s
            """,
            (now, error_message[:4000], now, run_id),
        )


def save_agent_artifact(
    connection: Connection,
    run_id: str,
    event_id: str,
    artifact_type: str,
    artifact: dict[str, Any],
) -> str:
    artifact_id = f"artifact_{uuid.uuid4()}"
    now = datetime.now(timezone.utc).replace(tzinfo=None)
    with connection.cursor() as cursor:
        cursor.execute(
            """
            INSERT INTO agent_artifact
              (artifact_id, run_id, event_id, artifact_type, artifact_json, created_at)
            VALUES
              (%s, %s, %s, %s, %s::jsonb, %s)
            """,
            (
                artifact_id,
                run_id,
                event_id,
                artifact_type,
                json.dumps(artifact, ensure_ascii=False),
                now,
            ),
        )
    return artifact_id


def save_agent_actions(
    connection: Connection,
    run_id: str,
    event_id: str,
    actions: list[dict[str, Any]],
) -> list[str]:
    action_ids: list[str] = []
    now = datetime.now(timezone.utc).replace(tzinfo=None)
    with connection.cursor() as cursor:
        for action in actions:
            action_id = f"action_{uuid.uuid4()}"
            cursor.execute(
                """
                INSERT INTO agent_action_outbox
                  (action_id, run_id, event_id, action_type, target_type, target_id,
                   payload_json, status, attempts, created_at, updated_at)
                VALUES
                  (%s, %s, %s, %s, %s, %s, %s::jsonb, 'pending', 0, %s, %s)
                """,
                (
                    action_id,
                    run_id,
                    event_id,
                    action["action_type"],
                    action["target_type"],
                    action["target_id"],
                    json.dumps(action["payload"], ensure_ascii=False),
                    now,
                    now,
                ),
            )
            action_ids.append(action_id)
    return action_ids


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


def fetch_retry_candidates(connection: Connection, limit: int, max_attempts: int) -> list[dict[str, Any]]:
    with connection.cursor() as cursor:
        cursor.execute(
            """
            SELECT event_id, source, event_type, occurred_at, trace_id, dedup_key, project_id, work_item_id,
                   subject_type, subject_id, external_refs_json, raw_event_id, publish_status, publish_attempts
            FROM normalized_event
            WHERE publish_status IN ('pending', 'failed')
              AND publish_attempts < %s
            ORDER BY created_at ASC
            FOR UPDATE SKIP LOCKED
            LIMIT %s
            """,
            (max_attempts, limit),
        )
        return list(cursor.fetchall())


def mark_publish_retry_succeeded(connection: Connection, event_id: str) -> None:
    now = datetime.now(timezone.utc).replace(tzinfo=None)
    with connection.cursor() as cursor:
        cursor.execute(
            """
            UPDATE normalized_event
            SET publish_status = 'published',
                published_at = %s,
                last_publish_error = NULL,
                last_publish_attempt_at = %s
            WHERE event_id = %s
            """,
            (now, now, event_id),
        )


def mark_publish_retry_failed(connection: Connection, event_id: str, error_message: str, max_attempts: int) -> None:
    now = datetime.now(timezone.utc).replace(tzinfo=None)
    with connection.cursor() as cursor:
        cursor.execute(
            """
            UPDATE normalized_event
            SET publish_attempts = publish_attempts + 1,
                last_publish_error = %s,
                last_publish_attempt_at = %s,
                publish_status = 'failed'
            WHERE event_id = %s
            """,
            (error_message[:4000], now, event_id),
        )
