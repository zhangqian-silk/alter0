# Alter0

> Local-first Task Orchestration Kernel for Mature Coding Agents

Alter0 是一个面向开发者自部署的任务编排内核。它默认运行在单实例、本地优先模式下，负责在多通道输入与成熟 Agent 执行器之间建立稳定的任务层：输入先被归入任务，再由执行器完成生成与执行。

## Overview

与“对话优先”的 Agent 容器不同，Alter0 以 `Task` 作为一等对象，重点解决三类工程问题：

- 如何把连续输入稳定映射到正确任务，而不是让上下文漂移。
- 如何在不绑定单一模型的前提下复用成熟执行器能力。
- 如何在长期运行中保持可观测、可审计、可维护。

## Design Domains

Alter0 采用以下设计域划分：

| 设计域 | 目标 | 当前实现 |
| --- | --- | --- |
| 部署与运行模型 | 面向开发者自部署，降低接入和运维复杂度 | 单实例、本地优先；支持 `go run .`、systemd 模板与 Docker 镜像打包 |
| 任务智能编排 | 用 LLM 进行任务归属与生命周期决策 | 路由（existing/new）+ 结题（close/open）双决策链 |
| 执行与扩展架构 | 执行器可替换，能力可注入 | `codex` / `claude_code`；Channel/Skill 抽象 |
| 可观测与上下文治理 | 支持运维排障并控制上下文冗余 | Web Console + `/health` + JSONL 日志 + 任务记忆快照 |

## Runtime Architecture

```text
Channel (CLI / HTTP / Web)
  -> Gateway
  -> Orchestrator (Task Router + Task Store + Task Closer)
  -> Executor (codex / claude_code)
  -> Orchestrator
  -> Channel
```

任务选择优先级：

1. 强制任务（`/task use`）
2. 显式 `task_id`
3. LLM 路由到已有开放任务
4. 创建新任务

## Core Capabilities

Alter0 的能力体系围绕任务编排内核展开，而不是围绕单次对话展开。系统将 Task 作为一等对象管理，覆盖任务创建、归属、状态流转和关闭；对输入的处理由模型决策驱动，包含任务路由与结题判断两个环节，并通过可配置置信度阈值控制误判风险。

功能矩阵与执行队列请参考：[`docs/features.md`](./docs/features.md)。

文档同步策略：当 `docs/features.md` 发生变更时，必须在同一 PR 同步更新 `README.md` 与 `ARCHITECTURE.md`。可以执行：

```bash
make docs-sync-check
```

该检查默认对比 `origin/master...HEAD`，若只改了 `docs/features.md` 会直接失败，避免能力矩阵与总览文档漂移。

在执行层，Alter0 复用成熟 Agent CLI（`codex`、`claude_code`），自身聚焦编排与治理，不重建模型执行栈。接收器（CLI/HTTP/Web）与执行器通过稳定接口解耦，允许在不影响任务存储和路由策略的前提下独立扩展通道或替换执行后端。扩展能力以 Skill 为主入口，外部能力（如 MCP）可通过扩展层纳入执行链路。

在运行治理方面，系统提供基础可观测能力，包括 Web Console、`/health` 探针、执行阶段日志与命令审计日志。为控制上下文开销，任务记忆采用快照式压缩策略，在 prompt 组装时优先保留高价值上下文，尽量降低冗余信息带来的推理噪声。

## Quick Start

### Prerequisites

- Go `1.25.7`（与 `go.mod` 保持一致）
- 安装至少一种执行器，并确保在 `PATH` 可访问：
  - `codex`
  - `claude`

### Run

```bash
go run .
```

启动时会先执行 preflight（配置合法性、SQLite 目录可写、执行器二进制可用），任一检查失败会直接退出并输出错误。

默认端点：

- Web Console: `http://localhost:8080/`
- API: `POST http://localhost:8080/api/message`
- Health: `GET http://localhost:8080/health`

## Deployment Options

### systemd

仓库内置服务模板：`deploy/systemd/alter0.service`。

典型部署流程：

```bash
sudo useradd --system --home /opt/alter0 --shell /usr/sbin/nologin alter0
sudo install -d -o alter0 -g alter0 /opt/alter0
sudo cp alter0 /usr/local/bin/alter0
sudo cp -r config /opt/alter0/
sudo cp deploy/systemd/alter0.service /etc/systemd/system/alter0.service
sudo systemctl daemon-reload
sudo systemctl enable --now alter0
```

### Docker

仓库根目录提供 `Dockerfile` 与 `.dockerignore`：

```bash
docker build -t alter0:local .
docker run --rm -p 8080:8080 \
  -v $(pwd)/output:/opt/alter0/output \
  -v $(pwd)/config/config.json:/opt/alter0/config/config.json:ro \
  alter0:local
```

可通过 `make deploy-check` 校验部署资产是否齐全。

## Configuration

配置文件路径：[`config/config.json`](./config/config.json)

示例：

```json
{
  "agent": {"name": "Alter0"},
  "executor": {"name": "codex"},
  "task": {
    "routing_timeout_sec": 15,
    "close_timeout_sec": 10,
    "generation_timeout_sec": 120,
    "routing_confidence_threshold": 0.55,
    "close_confidence_threshold": 0.7,
    "cli_user_id": "local_user",
    "open_task_candidate_limit": 8
  },
  "runtime": {
    "maintenance": {
      "enabled": true,
      "task_memory_prune_interval_sec": 21600,
      "task_memory_prune_timeout_sec": 20,
      "task_memory_retention_days": 30,
      "task_memory_open_retention_days": 0
    },
    "queue": {
      "enabled": true,
      "workers": 2,
      "buffer": 128,
      "enqueue_timeout_sec": 3,
      "attempt_timeout_sec": 180,
      "max_retries": 1,
      "retry_delay_sec": 2
    },
    "shutdown": {
      "drain_timeout_sec": 5
    }
  },
  "security": {
    "admin_user_ids": ["local_user"]
  }
}
```

关键项：

- `executor.name`: 当前执行器（`codex` / `claude_code`）
- `task.routing_confidence_threshold`: 路由命中阈值
- `task.close_confidence_threshold`: 自动结题阈值
- `task.open_task_candidate_limit`: 路由候选任务上限
- `runtime.maintenance.*`: 维护任务开关、执行间隔、超时与闭合/开放任务记忆保留策略
- `runtime.maintenance.task_memory_open_retention_days`: 开放任务记忆清理天数（`0` 表示禁用）
- `runtime.queue.*`: 执行队列开关、并发、重试与超时策略
- `runtime.shutdown.drain_timeout_sec`: 统一停机排空等待时间（queue/scheduler/http）
- `security.admin_user_ids`: 管理命令授权用户

## Command Interface

通用命令：

- `/help`
- `/status`
- `/config`
- `/config get [key]`
- `/task list [open|closed|all]`
- `/task current`
- `/task use <task_id>`
- `/task new [title]`
- `/task close [task_id]`
- `/task memory [task_id]`
- `/task memory clear [task_id]`
- `/task stats`

`/status` 输出统一运行时快照（gateway/scheduler/task/queue[workers,in_flight]/executors/command_audit_tail/git），与 HTTP `GET /api/status` 对齐。

管理员命令：

- `/config set <key> <value>`
- `/executor [name]`

## HTTP API

### `POST /api/message`

Request:

```json
{"content":"...", "user_id":"...", "task_id":"optional"}
```

Response:

```json
{"task_id":"...", "response":"...", "closed":false, "decision":"existing|new"}
```

可选流式分块输出（`stream=1`）：

- 响应头：`Content-Type: application/x-ndjson`
- 输出格式：多行 JSON，每行为一个 event
- chunk event: `{"type":"chunk","index":1,"total":N,"chunk":"..."}`
- done event: `{"type":"done","task_id":"...","decision":"existing|new","closed":false,"total":N}`
- 可选参数：`chunk_size`（按 rune 分块，默认 1200，最大 4000）

Example:

```bash
curl -X POST "http://localhost:8080/api/message?stream=1&chunk_size=800" \
  -H "Content-Type: application/json" \
  -d "{\"content\":\"帮我继续当前任务\",\"user_id\":\"local_user\"}"
```

### `GET /api/status`

返回当前 HTTP 通道状态，并附带运行时快照（gateway/scheduler/task/queue[workers,in_flight]/executors/command_audit_tail/git）。

## Observability and Data Layout

运行产物默认写入 `output/`：

- `output/db/alter0.db`: SQLite 数据库（tasks/messages/user_state/task_memory）
- `output/logs/`: 应用日志
- `output/orchestrator/<date>/executor_*.jsonl`: 执行器阶段日志（router/gen/closer）
- `output/audit/<date>/command_permission.jsonl`: 命令权限审计日志（`/status` 与 `/api/status` 会聚合最近 tail）

## Local Backup and Restore

Alter0 提供本地脚本用于备份与恢复核心状态（SQLite + 运行配置）：

- 备份数据库与配置：

```bash
make backup
# or: ./scripts/backup-local.sh
```

备份文件输出到 `output/backups/alter0-backup-<UTC timestamp>.tar.gz`。

- 从备份恢复：

```bash
make restore BACKUP=output/backups/alter0-backup-<timestamp>.tar.gz
# or: ./scripts/restore-local.sh <archive>
```

恢复会覆盖当前 `output/db/alter0.db` 与 `config/config.json`。

## Extension Guide

### Add a New Channel

1. 实现 `types.Channel`
2. 在启动时注册到 Gateway

### Add a New Executor

1. 在执行器解析逻辑中新增名称映射
2. 增加安装检测和调用命令定义

### Add a New Skill

1. 实现 `types.Skill`
2. 注册到 Skill Manager
3. 在命令层或编排层接入

## Scope and Non-goals

### Deployment Boundary

- 服务定位为开发者自部署、自使用，不面向多租户或公共开放场景。
- 默认网络边界为本机或内网可信环境，当前不引入统一鉴权体系。

### Stage Goal

- 在单实例前提下，实现可连续运行（24x7）、可恢复、可维护的任务编排运行时。

### Non-goals

- 账号体系与身份管理
- API 鉴权与访问控制
- 限流、WAF、复杂 HTTP 抗压链路

### Current Technical Constraints

1. 默认运行模型为单实例、本地优先，不内建分布式协调。
2. HTTP 接口默认同步响应；可通过 `stream=1` 使用 NDJSON 分块输出，单请求超时为 60 秒。
3. Schema 版本升级时会触发表重建；升级前应备份 `output/db`。

## License

[MIT](./LICENSE)
