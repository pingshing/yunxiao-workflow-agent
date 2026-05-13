from typing import Any

from app.agents.base import AgentAction, AgentResult


def run_work_item_agent(context_snapshot: dict[str, Any]) -> AgentResult:
    context = context_snapshot["context"]
    event = context.get("event", {})
    work_item = context.get("work_item", {})
    artifact = {
        "summary": "工作项事件已同步到 Agent 编排层。",
        "event_id": event.get("event_id"),
        "event_type": event.get("event_type"),
        "work_item": work_item,
        "findings": [],
        "recommendations": ["后续可接入 Projex API 补齐工作项详情、评论和关联对象。"],
    }
    return AgentResult(artifact_type="work_item_summary", artifact=artifact)


def run_pr_review_agent(context_snapshot: dict[str, Any]) -> AgentResult:
    context = context_snapshot["context"]
    event = context.get("event", {})
    pull_request = context.get("pull_request", {})
    artifact = {
        "summary": "PR 事件已进入评审 Agent，当前使用规则型占位输出。",
        "event_id": event.get("event_id"),
        "event_type": event.get("event_type"),
        "pull_request": pull_request,
        "findings": [],
        "recommendations": ["后续需要接入 Codeup API 获取真实 diff、commit 和评论上下文。"],
    }
    actions = []
    pr_id = pull_request.get("pr_id")
    if pr_id:
        actions.append(
            AgentAction(
                action_type="codeup.pr.comment",
                target_type="pull_request",
                target_id=str(pr_id),
                payload={
                    "body": "PR 已进入云效 Agent 编排层，后续将补充真实 diff 评审结果。",
                    "event_id": event.get("event_id"),
                },
            )
        )
    return AgentResult(artifact_type="pr_review_result", artifact=artifact, actions=actions)


def run_pipeline_diagnosis_agent(context_snapshot: dict[str, Any]) -> AgentResult:
    context = context_snapshot["context"]
    event = context.get("event", {})
    pipeline_run = context.get("pipeline_run", {})
    artifact = {
        "summary": "流水线失败事件已进入诊断 Agent，当前使用规则型占位输出。",
        "event_id": event.get("event_id"),
        "event_type": event.get("event_type"),
        "pipeline_run": pipeline_run,
        "failure_category": "unknown",
        "recommendations": ["后续需要接入 Flow API 获取失败 stage、job 和日志。"],
    }
    actions = []
    work_item_id = event.get("work_item_id")
    if work_item_id:
        actions.append(
            AgentAction(
                action_type="projex.work_item.comment",
                target_type="work_item",
                target_id=str(work_item_id),
                payload={
                    "body": "关联流水线失败事件已进入云效 Agent 诊断流程。",
                    "event_id": event.get("event_id"),
                },
            )
        )
    return AgentResult(artifact_type="pipeline_diagnosis_result", artifact=artifact, actions=actions)


def run_pipeline_status_agent(context_snapshot: dict[str, Any]) -> AgentResult:
    context = context_snapshot["context"]
    event = context.get("event", {})
    artifact = {
        "summary": "流水线状态事件已记录。",
        "event_id": event.get("event_id"),
        "event_type": event.get("event_type"),
        "pipeline_run": context.get("pipeline_run", {}),
        "requires_attention": event.get("event_type") == "pipeline.canceled",
    }
    return AgentResult(artifact_type="pipeline_status_result", artifact=artifact)


def run_delivery_summary_agent(context_snapshot: dict[str, Any]) -> AgentResult:
    context = context_snapshot["context"]
    event = context.get("event", {})
    artifact = {
        "summary": "交付闭环事件已进入总结 Agent。",
        "event_id": event.get("event_id"),
        "event_type": event.get("event_type"),
        "project_id": event.get("project_id"),
        "work_item_id": event.get("work_item_id"),
        "recommendations": ["后续需要关联历史 PR 评审、流水线结果和工作项状态生成完整交付总结。"],
    }
    actions = []
    work_item_id = event.get("work_item_id")
    if work_item_id:
        actions.append(
            AgentAction(
                action_type="projex.work_item.comment",
                target_type="work_item",
                target_id=str(work_item_id),
                payload={
                    "body": "交付闭环事件已由云效 Agent 处理，后续将补充完整交付总结。",
                    "event_id": event.get("event_id"),
                },
            )
        )
    return AgentResult(artifact_type="delivery_summary_result", artifact=artifact, actions=actions)


def run_repo_event_agent(context_snapshot: dict[str, Any]) -> AgentResult:
    context = context_snapshot["context"]
    event = context.get("event", {})
    artifact = {
        "summary": "仓库事件已进入 Agent 编排层。",
        "event_id": event.get("event_id"),
        "event_type": event.get("event_type"),
        "repository": context.get("repository", {}),
        "ref": context.get("ref", {}),
        "commit": context.get("commit", {}),
        "recommendations": ["如需代码级分析，建议优先关联 PR 事件。"],
    }
    return AgentResult(artifact_type="repo_event_summary", artifact=artifact)
