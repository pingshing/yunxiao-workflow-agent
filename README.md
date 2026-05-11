# 云效工作流助手

这个仓库是云效工作流助手的第一阶段后端实现骨架。

## 当前范围

当前实现覆盖前两层逻辑架构和事件分发链路：

- Go 接入服务接收 webhook 形式的事件。
- 接入服务把原始事件保存到 `raw_event`，把标准事件保存到 `normalized_event`。
- 接入服务把 `NormalizedEvent` 发布到 RabbitMQ。
- Python 编排服务消费 RabbitMQ 事件。
- 编排服务创建 `workflow_task` 记录，用于消费幂等。
- 编排服务构建占位版 `PRReviewContext` / `PipelineFailureContext`。
- 编排服务把汇聚结果保存到 `context_snapshot`。

当前实现刻意不包含 LangGraph、OpenAI 调用和真实云效 API Client。现在先固定后端主链路。
接入服务已经预留并实现 Codeup、Projex、Flow 三类 webhook adapter，用于把云效真实 payload 转成内部标准事件。

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
