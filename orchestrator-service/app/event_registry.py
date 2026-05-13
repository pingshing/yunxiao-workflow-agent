from dataclasses import dataclass


@dataclass(frozen=True)
class EventRoute:
    workflow_type: str
    agent_type: str
    context_type: str


EVENT_ROUTES: dict[str, EventRoute] = {
    "work_item.created": EventRoute("work_item_sync", "work_item_agent", "work_item_sync"),
    "work_item.updated": EventRoute("work_item_sync", "work_item_agent", "work_item_sync"),
    "work_item.status_changed": EventRoute("work_item_sync", "work_item_agent", "work_item_sync"),
    "work_item.assignee_changed": EventRoute("work_item_sync", "work_item_agent", "work_item_sync"),
    "work_item.deleted": EventRoute("work_item_sync", "work_item_agent", "work_item_sync"),
    "pr.created": EventRoute("pr_review", "pr_review_agent", "pr_review"),
    "pr.updated": EventRoute("pr_review", "pr_review_agent", "pr_review"),
    "pr.merged": EventRoute("delivery_summary", "delivery_summary_agent", "delivery_summary"),
    "pr.closed": EventRoute("event_triage", "event_triage_agent", "event_triage"),
    "repo.pushed": EventRoute("repo_event", "repo_event_agent", "repo_event"),
    "repo.tag_pushed": EventRoute("repo_event", "repo_event_agent", "repo_event"),
    "repo.note_created": EventRoute("repo_event", "repo_event_agent", "repo_event"),
    "pipeline.started": EventRoute("pipeline_status", "pipeline_status_agent", "pipeline_status"),
    "pipeline.failed": EventRoute("pipeline_diagnosis", "pipeline_diagnosis_agent", "pipeline_diagnosis"),
    "pipeline.succeeded": EventRoute("delivery_summary", "delivery_summary_agent", "delivery_summary"),
    "pipeline.finished": EventRoute("pipeline_status", "pipeline_status_agent", "pipeline_status"),
    "pipeline.canceled": EventRoute("pipeline_status", "pipeline_status_agent", "pipeline_status"),
    "pipeline.skipped": EventRoute("pipeline_status", "pipeline_status_agent", "pipeline_status"),
}

DEFAULT_EVENT_ROUTE = EventRoute("event_triage", "event_triage_agent", "event_triage")


def resolve_event_route(event_type: str) -> EventRoute:
    return EVENT_ROUTES.get(event_type, DEFAULT_EVENT_ROUTE)
