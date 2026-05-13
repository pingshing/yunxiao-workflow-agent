# 云效工作流助手 Agent 指南

本文件用于帮助新的 Codex 会话快速接手这个仓库。

## 先读什么

进入仓库后，按下面顺序读取：

1. `docs/current-state.md`
2. `CONTEXT.md`
3. `README.md`
4. `docs/agents/issue-tracker.md`
5. `docs/agents/triage-labels.md`
6. `docs/agents/domain.md`
7. `docs/deploy-server.md`

原则：

- `docs/current-state.md` 只看当前进度、风险和下一步。
- `CONTEXT.md` 只看长期稳定术语和边界。
- `README.md` 看系统主链路和启动方式。
- `docs/agents/*.md` 看项目级操作约定。

## 项目概述

这是一个面向云效研发事件的工作流助手后端仓库。

当前主链路：

```text
云效 webhook
-> ingress-service 接收并鉴权
-> raw_event / normalized_event 持久化
-> RabbitMQ 发布标准事件
-> orchestrator-service 消费事件
-> 构建 context_snapshot
-> 运行规则型 agent
-> 持久化 agent_run / agent_artifact / agent_action_outbox
```

## 仓库结构

```text
/
├── README.md
├── CONTEXT.md
├── AGENTS.md
├── docs/
│   ├── current-state.md
│   ├── deploy-server.md
│   └── agents/
├── infra/db/migration/
├── ingress-service/
└── orchestrator-service/
```

关键目录：

- `infra/db/migration`：PostgreSQL schema 迁移
- `ingress-service`：Go 接入服务，负责 webhook 接入和标准化
- `orchestrator-service`：Python 编排服务，负责消费、上下文构建、agent 执行和落库

## 工作边界

当前阶段重点是后端主链路和 agent runtime 骨架：

- 已有真实 webhook 接入链路
- 已有标准事件路由
- 已有上下文快照持久化
- 已有规则型 agent 占位实现
- 已有 action outbox 持久化

当前还没有：

- 真实 OpenAI 调用
- 真实 LangGraph 编排
- 真实 Codeup / Flow / Projex API 详情补齐
- action 执行器回写云效

## 常用命令

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

查看数据库：

```bash
docker compose exec postgres psql -U yunxiao -d yunxiao_agent
```

Go 测试：

```bash
cd ingress-service
go test ./...
```

Python 测试：

```bash
cd orchestrator-service
python -m pytest
```

## 协作约定

- 优先更新 `docs/current-state.md` 来记录阶段性进度和下一步，而不是把临时进展塞进 `README.md`。
- 新增领域术语或系统对象时，先更新 `CONTEXT.md`。
- 如果变更了 issue 跟踪方式、triage 标签或上下文文档布局，更新 `docs/agents/*.md`。

## Agent skills

### Issue tracker

当前仓库默认使用 GitHub Issues 作为任务、PRD 和拆分 issue 的落地点。See `docs/agents/issue-tracker.md`.

### Triage labels

当前仓库使用一套固定的 canonical triage labels。See `docs/agents/triage-labels.md`.

### Domain docs

当前是单上下文仓库，根目录 `CONTEXT.md` 是领域术语入口，`docs/adr/` 预留给架构决策记录。See `docs/agents/domain.md`.
