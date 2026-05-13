from typing import Any, Callable

from app.agents.base import AgentResult
from app.agents.event_triage_agent import run as run_event_triage_agent
from app.agents.rule_agents import (
    run_delivery_summary_agent,
    run_pipeline_diagnosis_agent,
    run_pipeline_status_agent,
    run_pr_review_agent,
    run_repo_event_agent,
    run_work_item_agent,
)

AGENT_HANDLERS: dict[str, Callable[[dict[str, Any]], AgentResult]] = {
    "event_triage_agent": run_event_triage_agent,
    "work_item_agent": run_work_item_agent,
    "pr_review_agent": run_pr_review_agent,
    "pipeline_diagnosis_agent": run_pipeline_diagnosis_agent,
    "pipeline_status_agent": run_pipeline_status_agent,
    "delivery_summary_agent": run_delivery_summary_agent,
    "repo_event_agent": run_repo_event_agent,
}


def run_agent(agent_type: str, context_snapshot: dict[str, Any]) -> AgentResult:
    handler = AGENT_HANDLERS.get(agent_type, run_event_triage_agent)
    return handler(context_snapshot)
