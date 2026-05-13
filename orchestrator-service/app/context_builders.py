from typing import Any

from app.event_registry import EventRoute, resolve_event_route
from app.models import NormalizedEvent


def build_context(event: NormalizedEvent, route: EventRoute | None = None) -> tuple[str, dict[str, Any], dict[str, Any]]:
    route = route or resolve_event_route(event["event_type"])
    event_type = event["event_type"]
    if route.context_type == "pr_review":
        return route.context_type, build_pr_review_context(event), pr_source_refs(event)
    if route.context_type == "repo_event":
        return route.context_type, build_repo_event_context(event), {"event": event["event_id"]}
    if route.context_type == "pipeline_diagnosis":
        return route.context_type, build_pipeline_context(event, route.context_type), pipeline_source_refs(event)
    if route.context_type == "pipeline_status":
        return route.context_type, build_pipeline_context(event, route.context_type), pipeline_source_refs(event)
    if route.context_type == "work_item_sync":
        return route.context_type, build_work_item_context(event), {"event": event["event_id"]}
    if route.context_type == "delivery_summary":
        return route.context_type, build_delivery_summary_context(event), delivery_source_refs(event)
    return route.context_type, build_event_triage_context(event, route), {"event": event["event_id"], "event_type": event_type}


def build_pr_review_context(event: NormalizedEvent) -> dict[str, Any]:
    external_refs = event.get("external_refs") or {}
    repo_id = external_refs.get("repo_id")
    return {
        "context_type": "pr_review",
        "event": event,
        "project": {
            "project_id": event.get("project_id"),
            "enabled_rules": ["default_pr_review"],
        },
        "repository": {
            "repo_id": repo_id,
            "name": external_refs.get("repo_name"),
            "default_branch": external_refs.get("target_branch"),
            "metadata_source": "webhook_or_yunxiao_api_placeholder",
        },
        "repo_profile": {
            "source": ".yunxiao-agent/repo-profile.yaml",
            "modules": [],
            "rules": {},
        },
        "pull_request": {
            "pr_id": (event.get("subject") or {}).get("id"),
            "source_branch": external_refs.get("source_branch"),
            "target_branch": external_refs.get("target_branch"),
        },
        "primary_work_item_id": event.get("work_item_id"),
        "work_items": [],
        "diff": {
            "summary": {
                "files_changed": 0,
                "additions": 0,
                "deletions": 0,
                "change_size": "unknown",
                "has_test_changes": False,
                "has_config_changes": False,
                "has_db_changes": False,
                "has_api_changes": False,
            },
            "changed_files": [],
            "important_hunks": [],
            "raw_diff_ref": None,
        },
        "commits": [],
        "review_knowledge": {
            "source": ".yunxiao-agent/review-rules.md",
            "sections": [],
        },
    }


def build_pipeline_context(event: NormalizedEvent, context_type: str) -> dict[str, Any]:
    external_refs = event.get("external_refs") or {}
    return {
        "context_type": context_type,
        "event": event,
        "project": {
            "project_id": event.get("project_id"),
            "enabled_rules": ["default_pipeline_diagnosis"],
        },
        "repository": {
            "repo_id": external_refs.get("repo_id"),
            "name": external_refs.get("repo_name"),
            "default_branch": external_refs.get("target_branch"),
            "metadata_source": "webhook_or_yunxiao_api_placeholder",
        },
        "pipeline": {
            "pipeline_id": external_refs.get("pipeline_id"),
            "name": external_refs.get("pipeline_name"),
            "type": "ci",
        },
        "pipeline_run": {
            "run_id": (event.get("subject") or {}).get("id"),
            "status": event.get("event_type"),
            "branch": external_refs.get("branch"),
            "commit_sha": external_refs.get("commit_sha"),
        },
        "failed_stage": {},
        "failed_job": {},
        "failure_logs": {
            "raw_log_ref": None,
            "log_excerpt": [],
            "error_signatures": [],
            "failure_category_hint": "unknown",
        },
        "related_commit": {
            "sha": external_refs.get("commit_sha"),
        },
        "related_pull_request": None,
        "work_items": [],
        "recent_changes": {},
        "repo_profile": {},
        "diagnosis_hints": [],
    }


def build_pipeline_failure_context(event: NormalizedEvent) -> dict[str, Any]:
    return build_pipeline_context(event, "pipeline_diagnosis")


def build_work_item_context(event: NormalizedEvent) -> dict[str, Any]:
    return {
        "context_type": "work_item_sync",
        "event": event,
        "work_item": {
            "work_item_id": (event.get("subject") or {}).get("id"),
        },
    }


def build_repo_event_context(event: NormalizedEvent) -> dict[str, Any]:
    external_refs = event.get("external_refs") or {}
    return {
        "context_type": "repo_event",
        "event": event,
        "repository": {
            "name": external_refs.get("repo_name"),
            "url": external_refs.get("repository_url"),
        },
        "ref": {
            "type": (event.get("subject") or {}).get("type"),
            "name": (event.get("subject") or {}).get("id"),
            "full_ref": external_refs.get("ref"),
        },
        "commit": {
            "sha": external_refs.get("commit_sha"),
            "message": external_refs.get("commit_message"),
            "url": external_refs.get("commit_url"),
        },
    }


def build_delivery_summary_context(event: NormalizedEvent) -> dict[str, Any]:
    external_refs = event.get("external_refs") or {}
    return {
        "context_type": "delivery_summary",
        "event": event,
        "project": {
            "project_id": event.get("project_id"),
        },
        "work_item": {
            "work_item_id": event.get("work_item_id"),
        },
        "repository": {
            "repo_id": external_refs.get("repo_id"),
            "name": external_refs.get("repo_name"),
            "url": external_refs.get("repository_url"),
        },
        "delivery_signal": {
            "event_type": event.get("event_type"),
            "subject": event.get("subject") or {},
            "commit_sha": external_refs.get("commit_sha"),
        },
        "related_artifacts": [],
    }


def build_event_triage_context(event: NormalizedEvent, route: EventRoute) -> dict[str, Any]:
    return {
        "context_type": "event_triage",
        "event": event,
        "route": {
            "workflow_type": route.workflow_type,
            "agent_type": route.agent_type,
            "context_type": route.context_type,
        },
        "triage": {
            "reason": "没有匹配到专用上下文构建器，进入兜底 Agent。",
        },
    }


def pr_source_refs(event: NormalizedEvent) -> dict[str, Any]:
    external_refs = event.get("external_refs") or {}
    return {
        "event": event["event_id"],
        "repo_id": external_refs.get("repo_id"),
        "pull_request_id": (event.get("subject") or {}).get("id"),
        "repo_profile": ".yunxiao-agent/repo-profile.yaml",
        "review_rules": ".yunxiao-agent/review-rules.md",
    }


def pipeline_source_refs(event: NormalizedEvent) -> dict[str, Any]:
    external_refs = event.get("external_refs") or {}
    return {
        "event": event["event_id"],
        "repo_id": external_refs.get("repo_id"),
        "pipeline_id": external_refs.get("pipeline_id"),
        "pipeline_run_id": (event.get("subject") or {}).get("id"),
    }


def delivery_source_refs(event: NormalizedEvent) -> dict[str, Any]:
    external_refs = event.get("external_refs") or {}
    return {
        "event": event["event_id"],
        "work_item_id": event.get("work_item_id"),
        "pull_request_id": (event.get("subject") or {}).get("id") if event["event_type"].startswith("pr.") else None,
        "pipeline_run_id": (event.get("subject") or {}).get("id") if event["event_type"].startswith("pipeline.") else None,
        "commit_sha": external_refs.get("commit_sha"),
    }
