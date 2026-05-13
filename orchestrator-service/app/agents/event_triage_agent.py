from typing import Any

from app.agents.base import AgentResult


def run(context_snapshot: dict[str, Any]) -> AgentResult:
    context = context_snapshot["context"]
    event = context.get("event", {})
    subject = event.get("subject") or {}
    artifact = {
        "summary": "事件已进入兜底 Agent 处理。",
        "event_id": event.get("event_id"),
        "event_type": event.get("event_type"),
        "source": event.get("source"),
        "subject": {
            "type": subject.get("type"),
            "id": subject.get("id"),
        },
        "project_id": event.get("project_id"),
        "work_item_id": event.get("work_item_id"),
        "recommendations": [
            "如果该事件需要自动处理，应在事件注册表中配置专用 agent。",
        ],
    }
    return AgentResult(artifact_type="event_triage_result", artifact=artifact)
