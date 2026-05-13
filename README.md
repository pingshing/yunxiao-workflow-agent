# 云效工作流助手

这个仓库是云效工作流助手的后端主链路实现。

当前版本已经具备：

- 云效 webhook 接入
- 事件标准化与持久化
- RabbitMQ 事件分发
- 编排层事件路由
- 上下文快照构建
- 规则型 Agent 执行
- Agent 产物与动作落库
- 基础失败重试能力

如果你是新的开发会话，建议先看：

- `AGENTS.md`
- `docs/current-state.md`
- `CONTEXT.md`

## 当前范围

当前实现覆盖 webhook 接入、事件分发和第一版 agent runtime 主链路：

- Go 接入服务接收 webhook 形式的事件。
- 接入服务把原始事件保存到 `raw_event`，把标准事件保存到 `normalized_event`。
- 接入服务把 `NormalizedEvent` 发布到 RabbitMQ。
- Python 编排服务消费 RabbitMQ 事件。
- 编排服务创建 `workflow_task` 记录，用于消费幂等。
- 编排服务根据 `event_type` 路由到不同 `workflow_type`。
- 编排服务构建上下文快照并保存到 `context_snapshot`。
- 编排服务执行规则型 agent，并保存 `agent_run`、`agent_artifact` 和 `agent_action_outbox`。

当前实现刻意不包含真实 OpenAI 调用、LangGraph 编排和完整云效 API 详情补齐。
当前重点是先固定后端主链路和运行时数据模型。

接入服务已经预留并实现 Codeup、Projex、Flow 三类 webhook adapter，用于把云效真实 payload 转成内部标准事件。

## 系统主链路

```text
云效 webhook
-> ingress-service 接收并鉴权
-> raw_event / normalized_event 持久化
-> RabbitMQ 发布标准事件
-> orchestrator-service 消费标准事件
-> event_type -> workflow_type 路由
-> 构建 context_snapshot
-> 运行规则型 agent
-> 持久化 agent_run / agent_artifact / agent_action_outbox
```

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
- action outbox 尚未接入独立执行器
- 还没有真实 LLM 工作流

## 本地启动

先准备环境变量：

```bash
cp .env.example .env
```

编辑 `.env`，替换三个 webhook token。

```bash
docker compose up --build
```

服务地址：

- 接入服务健康检查：`http://localhost:8080/healthz`
- 编排服务健康检查：`http://localhost:8081/healthz`
- RabbitMQ 管理后台：`http://localhost:15672`（`guest` / `guest`）

## Webhook 入口

| 路径 | 用途 |
|---|---|
| `POST /webhooks/yunxiao/codeup` | Codeup 合并请求事件 |
| `POST /webhooks/yunxiao/projex?event_type=work_item.updated` | Projex 工作项事件 |
| `POST /webhooks/yunxiao/flow` | Flow 流水线通知事件 |

## 云效联调方式

在云效中把 webhook 地址配置为当前接入服务暴露出的真实地址：

```text
Codeup 合并请求事件：
POST https://<你的域名>/webhooks/yunxiao/codeup

Projex 工作项更新事件：
POST https://<你的域名>/webhooks/yunxiao/projex?event_type=work_item.updated

Projex 工作项状态变更事件：
POST https://<你的域名>/webhooks/yunxiao/projex?event_type=work_item.status_changed

Flow 流水线通知事件：
POST https://<你的域名>/webhooks/yunxiao/flow
```

预期结果：

- `raw_event` 中保存原始 payload。
- `normalized_event` 中存在 `publish_status = published` 的标准事件。
- `workflow_task` 中生成一条 `pr_review` 任务。
- `context_snapshot` 中生成一份 `pr_review` 上下文 JSON。
- `agent_run` 中生成一条规则型 agent 执行记录。
- `agent_artifact` 中生成一条结构化产物。
- 如果规则允许，还会在 `agent_action_outbox` 中生成待执行动作。

## 查看数据库

```bash
docker compose exec postgres psql -U yunxiao -d yunxiao_agent
```

常用查询：

```sql
SELECT id, source, event_type, external_event_id, received_at, trace_id
FROM raw_event
ORDER BY id DESC
LIMIT 10;

SELECT id, event_id, source, event_type, project_id, work_item_id,
       subject_type, subject_id, publish_status, created_at
FROM normalized_event
ORDER BY id DESC
LIMIT 10;

SELECT id, task_id, event_id, event_type, workflow_type, status,
       context_snapshot_id, error_message, created_at
FROM workflow_task
ORDER BY id DESC
LIMIT 10;

SELECT jsonb_pretty(context_json)
FROM context_snapshot
ORDER BY id DESC
LIMIT 1;
```

## 服务器部署

轻量服务器部署步骤见：

[docs/deploy-server.md](docs/deploy-server.md)
