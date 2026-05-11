# 轻量服务器部署说明

本文档用于把云效工作流助手部署到一台轻量服务器上，供云效 webhook 直接回调测试。

## 1. 部署方式

当前阶段推荐：

```text
本地代码推送到 Git 仓库
-> 服务器 git pull
-> 服务器 docker compose up -d --build
```

暂时不需要完整 CI/CD，也不需要镜像仓库。

## 2. 服务器准备

以下命令以 Ubuntu / Debian 系统为例：

```bash
sudo apt update
sudo apt install -y git docker.io docker-compose-plugin
sudo systemctl enable --now docker
```

确认 Docker 可用：

```bash
docker --version
docker compose version
```

## 3. 拉取代码

```bash
git clone <你的仓库地址> yunxiao-workflow-agent
cd yunxiao-workflow-agent
```

如果已经拉取过：

```bash
cd yunxiao-workflow-agent
git pull
```

## 4. 配置环境变量

复制示例文件：

```bash
cp .env.example .env
```

编辑 `.env`：

```bash
vim .env
```

建议把三个 token 都改成强随机值：

```env
CODEUP_WEBHOOK_TOKEN=替换为强随机CodeupToken
PROJEX_WEBHOOK_SECRET=替换为强随机ProjexSecret
FLOW_WEBHOOK_TOKEN=替换为强随机FlowToken
```

可以用下面命令生成随机值：

```bash
openssl rand -hex 32
```

## 5. 启动服务

```bash
docker compose up -d --build
```

查看服务：

```bash
docker compose ps
```

查看日志：

```bash
docker compose logs -f ingress-service
docker compose logs -f orchestrator-service
```

## 6. 健康检查

如果服务器安全组已经开放 8080：

```bash
curl http://<服务器IP>:8080/healthz
```

预期返回：

```json
{"status":"ok"}
```

## 7. 云效 webhook 配置

临时测试可以直接使用 HTTP：

```text
Codeup:
http://<服务器IP>:8080/webhooks/yunxiao/codeup

Projex 工作项更新:
http://<服务器IP>:8080/webhooks/yunxiao/projex?event_type=work_item.updated

Projex 工作项状态变更:
http://<服务器IP>:8080/webhooks/yunxiao/projex?event_type=work_item.status_changed

Flow:
http://<服务器IP>:8080/webhooks/yunxiao/flow
```

对应鉴权：

```text
Codeup:
X-Codeup-Token = .env 中的 CODEUP_WEBHOOK_TOKEN

Projex:
X-Projex-Signature = .env 中的 PROJEX_WEBHOOK_SECRET

Flow:
Authorization = Bearer <.env 中的 FLOW_WEBHOOK_TOKEN>
```

## 8. 查看数据库

进入 PostgreSQL：

```bash
docker compose exec postgres psql -U yunxiao -d yunxiao_agent
```

查看原始事件：

```sql
SELECT id, source, event_type, external_event_id, received_at, trace_id
FROM raw_event
ORDER BY id DESC
LIMIT 10;
```

查看标准事件：

```sql
SELECT id, event_id, source, event_type, project_id, work_item_id,
       subject_type, subject_id, publish_status, created_at
FROM normalized_event
ORDER BY id DESC
LIMIT 10;
```

查看任务：

```sql
SELECT id, task_id, event_id, event_type, workflow_type, status,
       context_snapshot_id, error_message, created_at
FROM workflow_task
ORDER BY id DESC
LIMIT 10;
```

查看上下文快照：

```sql
SELECT id, snapshot_id, event_id, context_type, subject_type, subject_id,
       project_id, work_item_id, created_at
FROM context_snapshot
ORDER BY id DESC
LIMIT 10;
```

查看上下文 JSON：

```sql
SELECT jsonb_pretty(context_json)
FROM context_snapshot
ORDER BY id DESC
LIMIT 1;
```

## 9. 推荐后续升级

临时测试可以直接开放 `8080`。

后续建议加：

```text
域名
Nginx
HTTPS 证书
只对公网暴露 80 / 443
不直接暴露 PostgreSQL / RabbitMQ 管理端口
```

正式 webhook 地址建议变成：

```text
https://<你的域名>/webhooks/yunxiao/codeup
https://<你的域名>/webhooks/yunxiao/projex?event_type=work_item.updated
https://<你的域名>/webhooks/yunxiao/flow
```
