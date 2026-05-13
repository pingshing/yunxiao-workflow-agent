# 云效工作流助手上下文

本文档只记录长期稳定的术语、系统对象和边界，不记录阶段性进度。

## 系统目标

云效工作流助手的目标是把云效研发事件接入统一后端主链路，构建标准化上下文，并驱动后续 Agent 分析、总结和回写动作。

当前系统重点不是直接生成高质量 AI 结果，而是先打通：

- 事件接入
- 事件标准化
- 事件路由
- 上下文构建
- Agent 运行记录
- 产物与动作持久化

## 服务边界

### `ingress-service`

接入服务，负责把外部 webhook 变成内部标准事件。

负责：

- 接收云效 webhook
- 鉴权
- 保存 `raw_event`
- 转换并保存 `normalized_event`
- 发布 RabbitMQ 消息

不负责：

- 调用 LLM
- 拉取完整 PR diff 或流水线日志
- 构建完整分析上下文
- 执行云效写操作

### `orchestrator-service`

编排服务，负责把标准事件转成可执行的 Agent 工作单元。

负责：

- 消费 `normalized_event`
- 根据 `event_type` 路由到 `workflow_type`
- 构建 `context_snapshot`
- 创建 `workflow_task`
- 创建并执行 `agent_run`
- 保存 `agent_artifact`
- 保存 `agent_action_outbox`

不负责：

- 直接暴露 webhook 接口
- 在接入层做事件标准化

## 领域对象

### `raw_event`

原始入站事件。

含义：

- 这是 webhook 原样或近原样保存的输入证据
- 用于审计、重放、排障

### `normalized_event`

标准化后的内部事件。

含义：

- 这是跨来源统一后的事件格式
- 后续所有编排逻辑都围绕它展开

### `workflow_task`

事件进入编排层后的任务记录。

含义：

- 表示某个 `event_id` 在某个 `workflow_type` 上的处理实例
- 主要承担幂等和处理状态跟踪职责

### `context_snapshot`

某次工作流执行时使用的上下文快照。

含义：

- 记录当时传给 Agent 的结构化输入
- 便于回放、审计、诊断

### `agent_run`

一次 Agent 执行记录。

含义：

- 表示某个 `workflow_task` 被某种 `agent_type` 实际执行了一次
- 记录状态、模型名、输入快照、错误信息和原始输出

### `agent_artifact`

Agent 产出的结构化结果。

含义：

- 用于保存摘要、诊断结果、评审结果等结果型输出

### `agent_action_outbox`

Agent 准备执行的外部动作队列。

含义：

- 保存评论、状态更新、外部回写等动作
- 当前阶段先落库，不代表已经真正执行

## 路由术语

### `event_type`

标准事件类型，如：

- `pr.created`
- `pr.updated`
- `pr.merged`
- `pipeline.failed`
- `work_item.updated`

### `workflow_type`

编排层工作流类别。当前包括：

- `work_item_sync`
- `pr_review`
- `repo_event`
- `pipeline_status`
- `pipeline_diagnosis`
- `delivery_summary`
- `event_triage`

### `context_type`

传给 Agent 的上下文结构类别。通常与 `workflow_type` 对应，但语义强调“输入快照形状”。

### `agent_type`

具体执行当前工作流的 Agent 类型。当前实现为规则型占位 Agent，例如：

- `work_item_agent`
- `pr_review_agent`
- `pipeline_diagnosis_agent`
- `pipeline_status_agent`
- `delivery_summary_agent`
- `repo_event_agent`
- `event_triage_agent`

## 当前系统事实

- 当前 Agent 执行器是规则型实现，不是真实 LLM 编排。
- 多种上下文构建器已经存在，但很多字段仍是 placeholder。
- `agent_action_outbox` 已经存在，但独立 action executor 尚未落地。
- `pr_review`、`pipeline_diagnosis`、`delivery_summary` 等场景已经有最小占位产物和动作结构。

## 常见误解

- `normalized_event` 不是最终上下文，它只是统一后的事件。
- `context_snapshot` 不是数据库镜像，它是某次 Agent 输入的快照。
- `agent_action_outbox` 不是执行成功记录，它只是待执行动作存储。
- 当前“Agent”更多表示工作流处理单元，而不是成熟的 LLM 智能体。
