from typing import Any, TypedDict


class NormalizedEvent(TypedDict, total=False):
    event_id: str
    source: str
    event_type: str
    occurred_at: str
    trace_id: str
    dedup_key: str
    project_id: str
    work_item_id: str
    subject: dict[str, Any]
    external_refs: dict[str, Any]
    raw_payload_id: str
