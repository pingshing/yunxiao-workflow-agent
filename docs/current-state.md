# 当前状态

更新时间：2026-05-13

## 当前系统做到哪了

仓库已经不再只是 webhook 接入骨架，而是具备了完整的第一条 Agent runtime 主链路：

- `ingress-service` 可接收 Codeup、Projex、Flow webhook
- webhook 会落 `raw_event`
- 事件会被标准化为 `normalized_event`
- 标准事件会发布到 RabbitMQ
- `orchestrator-service` 会按 `event_type` 路由到不同工作流
- 编排层会创建 `workflow_task`
- 编排层会构建 `context_snapshot`
- 编排层会执行规则型 agent
- 结果会落 `agent_run`、`agent_artifact`、`agent_action_outbox`

## 最近代码现状

当前代码已经包含这些关键能力：

- 事件路由表：`orchestrator-service/app/event_registry.py`
- 上下文构建器：`orchestrator-service/app/context_builders.py`
- Agent 分发器：`orchestrator-service/app/agent_runner.py`
- 规则型 Agent：`orchestrator-service/app/agents/`
- Agent runtime 表迁移：`infra/db/migration/V4__agent_runtime_tables.sql`
- 失败重试相关迁移：`V2__raw_event_processing_status.sql`、`V3__normalized_event_publish_retry.sql`

## 当前已支持的工作流

- `work_item_sync`
- `pr_review`
- `repo_event`
- `pipeline_status`
- `pipeline_diagnosis`
- `delivery_summary`
- `event_triage` 兜底流程

## 当前仍是占位实现的部分

- PR 评审还没有拉取真实 diff、commit、评论上下文
- 流水线诊断还没有拉取真实失败 stage、job、日志
- 工作项同步还没有补齐 Projex 详情、评论、关联对象
- delivery summary 还没有聚合完整交付链路
- action outbox 只是落库，尚未有独立执行器真正回写云效
- 还没有真实 OpenAI / LangGraph 集成

## 对新会话最重要的判断

如果下一次对话是要继续开发，这个项目当前最合理的理解是：

- 重点已经从“接入 webhook”转向“把 agent runtime 主链路补完整”
- 现在最缺的不是新增事件类型，而是把已有工作流从 placeholder 补成可用闭环
- `pr_review` 和 `pipeline_diagnosis` 是最值得优先深化的两个场景

## 推荐下一步任务

1. 落地 action executor，消费 `agent_action_outbox` 并真正回写云效
2. 为 `pr_review` 接入 Codeup API，补齐 diff、commit、评论上下文
3. 为 `pipeline_diagnosis` 接入 Flow API，补齐失败 job 和日志摘要
4. 把 `repo_profile`、`review_rules` 这类仓库级规则文件格式固定下来
5. 增加端到端联调脚本或回放脚本，减少手工 webhook 测试成本

## 当前风险

- 一部分 Python 编排代码是新增能力，行为边界虽然清楚，但还需要更多集成测试覆盖
- 规则型 Agent 的输出格式已经存在，后续接入真实 LLM 时需要注意兼容演进
- `docs/current-state.md` 需要在每轮较大实现后持续维护，否则新会话仍然会因为进度漂移而产生误判

## 如果新会话要快速开工

建议优先这样做：

1. 读本文件
2. 读根目录 `CONTEXT.md`
3. 读 `README.md`
4. 读 `orchestrator-service/app/event_registry.py`
5. 读 `orchestrator-service/app/context_builders.py`
6. 读 `orchestrator-service/app/consumer.py`

## 联调入口

本地启动：

```bash
cp .env.example .env
docker compose up --build
```

健康检查：

```bash
curl http://localhost:8080/healthz
curl http://localhost:8081/healthz
```

查看最新上下文快照：

```sql
SELECT jsonb_pretty(context_json)
FROM context_snapshot
ORDER BY id DESC
LIMIT 1;
```
