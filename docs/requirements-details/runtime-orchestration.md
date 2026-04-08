# Runtime & Orchestration Requirements

> Last update: 2026-04-08

## 领域边界

Runtime & Orchestration 负责把所有触发源归一成稳定执行链路，并为上层 Chat、Agent、Task、Product、Terminal 提供统一消息、路由、执行端口、调度与观测底座。

## 核心对象

| 对象 | 职责 |
| --- | --- |
| `UnifiedMessage` | 承载消息身份、通道、触发类型、追踪字段与业务载荷 |
| `OrchestrationResult` | 承载路由、输出、错误编码与执行摘要 |
| `Intent` | 表达命令或自然语言意图 |
| `Command` / `CommandHandler` | 承载内置命令与命令处理逻辑 |
| `ExecutionPort` | 隔离自然语言执行实现，可对接 LLM、Agent、Codex CLI 或 Workflow |
| `Channel` | 表示 CLI、Web、Scheduler 等输入通道 |
| `SchedulerJob` | 表示可配置定时任务及其触发计划 |
| `TraceContext` | 贯穿 trace、session、message、correlation 维度 |

## 输入通道

### CLI

- 启动后默认开启 CLI 输入能力。
- CLI 输入必须转换为 `UnifiedMessage`，并进入同一编排层。
- `/quit` 与 `/exit` 退出 CLI 交互，不影响 Web 服务需求口径。

### Web

- Web 消息入口必须转换为 `UnifiedMessage`，并携带 Web channel 信息。
- 普通 JSON 消息和 SSE 流式消息复用同一编排语义。
- Web 登录、会话历史、移动端体验等入口侧规则由 Conversation & Session Experience 领域维护。

### Scheduler

- Scheduler 到点触发后生成 `trigger_type=cron`、`channel_type=scheduler` 的 `UnifiedMessage`。
- Cron 触发必须复用编排层，不允许绕过 Orchestration 直接调用执行器。
- Cron 触发记录需要保留 `job_id`、`job_name`、`fired_at` 与会话回链。

## 编排路由

### 意图识别

- 编排层先判断输入是命令还是自然语言。
- 命令请求必须优先于复杂度评估执行，避免 `/help` 等命令进入模型或任务分流。
- 自然语言请求进入 `ExecutionPort`，并按当前上下文决定同步、流式、Agent 或异步任务执行。

### 命令执行

- 稳定内置命令包括 `/help`、`/echo`、`/time` 与 `/now`。
- 命令输出必须进入统一 `OrchestrationResult`。
- 新增命令应在 Orchestration 领域注册，并补充命令行为测试。

### 自然语言执行

- 执行端口只表达执行契约，不绑定具体模型或外部进程。
- LLM、Agent、Codex CLI 和后续 Workflow 执行器都应通过端口或适配层接入。
- 执行错误需保留可诊断错误编码，供 Web、Task 与 Agent 收口使用。

## 调度能力

### Cron Job 配置

- Control 面提供 Cron Job 创建、更新、删除与列表查询。
- Cron Job 支持可视化周期字段、timezone、cron expression 与任务配置。
- 可视化字段与 cron expression 必须保持一致；无法表达时需返回明确错误。
- Cron Job 接口通过 `GET /api/control/cron/jobs`、`PUT /api/control/cron/jobs/{job_id}`、`DELETE /api/control/cron/jobs/{job_id}` 管理配置。

### 触发归档

- 每次 Cron 触发必须创建或关联可回查会话。
- 会话列表支持按 Cron 来源筛选。
- Cron Job 详情支持查看触发记录与会话回链。
- `GET /api/control/cron/jobs/{job_id}/runs` 按 `job_id`、分页参数和 `trigger_type=cron` 读取会话摘要，返回 `session_id`、触发时间与会话回链信息。

## 存储

### 本地存储

- 默认存储后端为 `.alter0` 本地文件。
- Control 配置与 Scheduler 状态优先使用 JSON。
- Memory 主存使用 Markdown，派生索引可按 Memory 领域策略重建。

### 可替换性

- 存储实现位于 infrastructure 层，不应反向污染 domain 与 application。
- 后续替换 SQLite、PostgreSQL 或配置中心时，不应改动核心业务服务代码契约。

## 观测与健康

### 日志与指标

- 结构化日志需包含 `trace_id`、`session_id`、`message_id`、`route` 等关键字段。
- `/metrics` 输出 Prometheus 文本格式指标。
- 关键指标覆盖输入消息数、路由请求数、命令请求数、错误数与路由耗时。

### 健康检查

- `/healthz` 表示进程活性。
- `/readyz` 表示当前实例已可对外承接请求。
- `/metrics` 提供 Prometheus 格式指标。
- Runtime 重启与候选二进制切换必须以 readyz 通过作为稳定切换条件。

## 依赖与边界

- Conversation 领域消费 Runtime 的消息与流式结果，不拥有编排路由规则。
- Agent 领域消费 ExecutionPort 与结构化上下文，不直接改写通道模型。
- Task 领域可承接复杂度分流后的后台执行，但触发源仍来自统一消息。
- Product 领域通过 Agent 与 Product Context 接入执行链，不绕过 Orchestration。

## 验收口径

- CLI、Web、Cron 三类输入都能进入统一编排链路。
- 命令请求不进入模型执行。
- Cron 触发可在会话与任务视图中追踪来源。
- 结构化日志、metrics、healthz、readyz 均可用。
- 存储实现替换不要求改动 domain 对象。
