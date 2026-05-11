from typing import Any

from app.models import NormalizedEvent


def build_context(event: NormalizedEvent) -> tuple[str, dict[str, Any], dict[str, Any]]:
    event_type = event["event_type"]
    if event_type.startswith("pr."):
        return "pr_review", build_pr_review_context(event), pr_source_refs(event)
    if event_type == "pipeline.failed":
        return "pipeline_failure", build_pipeline_failure_context(event), pipeline_source_refs(event)
    if event_type.startswith("work_item."):
        return "work_item_sync", build_work_item_context(event), {"event": event["event_id"]}
    raise ValueError(f"unsupported event_type: {event_type}")


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


def build_pipeline_failure_context(event: NormalizedEvent) -> dict[str, Any]:
    external_refs = event.get("external_refs") or {}
    return {
        "context_type": "pipeline_failure",
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
            "status": "failed",
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


def build_work_item_context(event: NormalizedEvent) -> dict[str, Any]:
    return {
        "context_type": "work_item_sync",
        "event": event,
        "work_item": {
            "work_item_id": (event.get("subject") or {}).get("id"),
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
