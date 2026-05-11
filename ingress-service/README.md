# 云效接入服务

`yunxiao-ingress-service` 是云效工作流助手的接入服务，负责把外部研发事件稳定接入系统。

## 服务职责

- 接收云效 webhook 事件。
- 做基础 Token 校验。
- 保存原始事件到 `raw_event`。
- 把原始事件转换成内部 `NormalizedEvent`。
- 保存标准事件到 `normalized_event`。
- 把标准事件发布到 RabbitMQ。
- 通过 `trace_id` 串联原始事件、标准事件和后续处理链路。

## 不负责的事情

- 不调用 LLM。
- 不拉取 PR diff。
- 不分析流水线日志。
- 不做风险判断。
- 不拼装完整上下文。
- 不直接执行云效写操作。

## 目录结构

```text
cmd/server
  程序入口，负责装配配置、数据库、RabbitMQ 和 HTTP 服务。

internal/config
  读取环境变量配置。

internal/httpserver
  Gin HTTP 服务和 webhook 路由。

internal/model
  接入层事件模型。

internal/yunxiao/codeup
  Codeup 合并请求 webhook 适配。

internal/yunxiao/projex
  Projex 自动化规则 webhook 适配。

internal/yunxiao/flow
  Flow 流水线通知 webhook 适配。

internal/repository
  raw_event 和 normalized_event 的 PostgreSQL 持久化。

internal/mq
  RabbitMQ 发布能力。
```

## Webhook 入口

| 路径 | 用途 | 鉴权 |
|---|---|---|
| `POST /webhooks/yunxiao/codeup` | Codeup 合并请求 webhook | `X-Codeup-Token: <CODEUP_WEBHOOK_TOKEN>` |
| `POST /webhooks/yunxiao/projex?event_type=work_item.updated` | Projex 自动化规则 webhook | `X-Projex-Signature: <PROJEX_WEBHOOK_SECRET>` |
| `POST /webhooks/yunxiao/flow` | Flow 流水线通知 webhook | `Authorization: Bearer <FLOW_WEBHOOK_TOKEN>` |

## 处理流程

```text
云效 webhook
-> /webhooks/yunxiao/codeup 或 /webhooks/yunxiao/projex 或 /webhooks/yunxiao/flow
-> 按来源校验 Token / Secret
-> 保存 raw_event
-> 生成 NormalizedEvent
-> 保存 normalized_event
-> 发布 RabbitMQ
-> 更新 publish_status
```

## 配置项

| 环境变量 | 说明 | 默认值 |
|---|---|---|
| `HTTP_ADDR` | HTTP 监听地址 | `:8080` |
| `POSTGRES_DSN` | PostgreSQL 连接串 | `postgres://yunxiao:yunxiao@127.0.0.1:5432/yunxiao_agent?sslmode=disable` |
| `RABBITMQ_URL` | RabbitMQ 连接串 | `amqp://guest:guest@127.0.0.1:5672/` |
| `RABBITMQ_EXCHANGE` | RabbitMQ Exchange | `yunxiao.events` |
| `CODEUP_WEBHOOK_TOKEN` | Codeup Webhook Token | `dev-codeup-token` |
| `PROJEX_WEBHOOK_SECRET` | Projex Webhook Secret | `dev-projex-secret` |
| `FLOW_WEBHOOK_TOKEN` | Flow Webhook Bearer Token | `dev-flow-token` |

## 当前边界

当前版本已经加入 Codeup、Projex、Flow 三类 adapter，但仍处于第一版接入阶段：

- Codeup 当前优先支持合并请求事件，映射为 `pr.created` / `pr.updated` / `pr.merged` / `pr.closed`。
- Projex 当前依赖自动化规则 URL 中的 `event_type` 参数，例如 `work_item.updated` 或 `work_item.status_changed`。
- Flow 当前根据 `task.statusCode` 映射流水线事件，例如 `FAIL` -> `pipeline.failed`。
- 三类 adapter 只做 payload 标准化，不在接入层调用云效 API 拉详情。
