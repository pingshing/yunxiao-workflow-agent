import json
import time

import pika


def build_normalized_event_message(row: dict[str, object]) -> dict[str, object]:
    external_refs = row.get("external_refs_json") or {}
    event = {
        "event_id": row["event_id"],
        "source": row["source"],
        "event_type": row["event_type"],
        "trace_id": row["trace_id"],
        "dedup_key": row["dedup_key"],
        "subject": {
            "type": row["subject_type"],
            "id": row["subject_id"],
        },
        "raw_payload_id": f"raw_{row['raw_event_id']}",
    }
    if row.get("project_id"):
        event["project_id"] = row["project_id"]
    if row.get("work_item_id"):
        event["work_item_id"] = row["work_item_id"]
    if external_refs:
        event["external_refs"] = external_refs
    occurred_at = row.get("occurred_at")
    if occurred_at is not None:
        event["occurred_at"] = occurred_at.isoformat()
    return event


def publish_normalized_event(channel, exchange: str, row: dict[str, object]) -> None:
    body = json.dumps(build_normalized_event_message(row), ensure_ascii=False).encode("utf-8")
    channel.basic_publish(
        exchange=exchange,
        routing_key=str(row["event_type"]),
        body=body,
        properties=pika.BasicProperties(
            content_type="application/json",
            delivery_mode=2,
            timestamp=int(time.time()),
            message_id=str(row["event_id"]),
            headers={
                "x-trace-id": str(row["trace_id"]),
                "x-event-id": str(row["event_id"]),
                "x-event-type": str(row["event_type"]),
                "x-source": str(row["source"]),
            },
        ),
    )
