# Requirements

> Last update: 2026-03-02

状态说明：

1. `supported`：已在主干代码可用
2. `planned`：已确认方向，待排期

## 1. 当前已支持需求

| ID | 需求 | 状态 | 说明 |
| --- | --- | --- | --- |
| R-001 | CLI 消息接入 | supported | 启动后自动开启 CLI 交互输入，统一进入编排层 |
| R-002 | Web 消息接入 | supported | `POST /api/messages` |
| R-003 | 统一消息模型 | supported | `UnifiedMessage` 统一 channel/trigger/trace |
| R-004 | 命令路由执行 | supported | `/help` `/echo` `/time` |
| R-005 | 自然语言路由执行 | supported | `ExecutionPort` 执行 |
| R-006 | 基础可观测性 | supported | metrics + health + structured logs |
| R-007 | Channel 配置管理 | supported | `GET/PUT/DELETE /api/control/channels/*` |
| R-008 | Skill 配置管理 | supported | `GET/PUT/DELETE /api/control/skills/*` |
| R-009 | 定时任务管理 | supported | `GET/PUT/DELETE /api/control/cron/jobs/*` |
| R-010 | 定时任务触发执行 | supported | Scheduler 触发消息并复用编排链路 |
| R-011 | 本地文件存储 | supported | Control/Scheduler 支持 `json`/`markdown` 本地存储 |

## 2. 规划中需求

| ID | 需求 | 状态 | 目标 |
| --- | --- | --- | --- |
