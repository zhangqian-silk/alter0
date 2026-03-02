# Architecture Design

## 1. 目标与原则

`alter0` 的架构目标是：在单机环境快速形成可运行闭环，并为后续扩展保留稳定边界。

核心原则：

1. 明确分层：`Interface -> Orchestration -> Execution`。
2. 统一消息：所有触发源都映射为 `UnifiedMessage`。
3. 控制面独立：配置管理不绕过编排执行链路。
4. 可观测优先：trace/session/message 全链路贯穿。

## 2. 系统分层

### 2.1 Data Plane（执行面）

1. `internal/interfaces/*`
- 负责接入外部输入（CLI/Web）。
- 统一封装请求为 `UnifiedMessage`。

2. `internal/orchestration/*`
- 负责意图识别与路由分发。
- 命令请求走 `CommandRegistry`，自然语言走 `ExecutionPort`。

3. `internal/execution/*`
- 负责自然语言执行能力。
- 当前为模板执行器实现，可替换为真实 LLM/Workflow 执行器。

4. `internal/scheduler/*`
- 负责定时任务触发。
- 定时任务生成 `TriggerType=cron` 的 `UnifiedMessage` 并回注编排层。

### 2.2 Control Plane（控制面）

1. `internal/control/*`
- 负责 `Channel` / `Skill` 配置管理。
- 通过存储接口解耦，默认使用本地文件存储。
- 存储格式按场景选择：Control 配置使用 `json`，Scheduler 状态使用 `json`。

2. `internal/interfaces/web`
- 暴露 Control API：
  - `/api/control/channels`
  - `/api/control/skills`
  - `/api/control/cron/jobs`

## 3. 关键领域对象

### 3.1 UnifiedMessage

统一消息结构位于 `internal/shared/domain/message.go`，关键字段：

1. 身份与追踪：`message_id`、`session_id`、`trace_id`、`correlation_id`
2. 通道维度：`channel_id`、`channel_type`
3. 触发类型：`trigger_type`（`user` / `cron` / `system`）
4. 业务载荷：`content`、`metadata`

### 3.2 OrchestrationResult

输出路由与结果摘要：

1. `route`：`command` / `nl`
2. `output`：执行结果
3. `error_code`：统一错误编码

## 4. 核心时序

### 4.1 用户消息（CLI/Web）

1. Adapter 接收输入并构造 `UnifiedMessage`（`trigger_type=user`）。
2. 调用 `orchestrator.Handle(...)`。
3. 编排层识别意图并路由：
- command -> CommandHandler
- nl -> ExecutionPort
4. 返回 `OrchestrationResult` 给调用方。

### 4.2 定时任务消息（Cron）

1. Scheduler 到点触发 Job。
2. 生成 `UnifiedMessage`（`trigger_type=cron`, `channel_type=scheduler`）。
3. 调用 `orchestrator.Handle(...)`，复用同一执行链路。
4. 记录日志与指标。

## 5. 可观测性设计

指标（Prometheus 文本输出）：

1. `alter0_gateway_messages_total{channel_type=...}`
2. `alter0_route_requests_total{route=...}`
3. `alter0_command_requests_total{command=...}`
4. `alter0_route_errors_total{route=...}`
5. `alter0_route_duration_seconds_avg{route=...}`

健康检查：

1. `/healthz`
2. `/readyz`

## 6. 扩展点

1. 新通道：新增 `interfaces/<channel>` 适配器并生成 `UnifiedMessage`。
2. 新技能：在 `control` 增加技能配置，再在 `execution/orchestration` 接入选择策略。
3. 新调度引擎：替换 `scheduler` 内部触发实现，不影响编排层接口。
4. 新存储：将 `control` 与 `scheduler` 的内存状态迁移到 DB/配置中心。
5. 新存储后端：通过 `storage` 模块装配不同实现，不改业务服务代码。

## 7. 当前边界与限制

1. 无鉴权与多租户。
2. 默认仅提供单机本地文件存储，不含分布式一致性能力。
3. Skill 配置已支持管理，尚未完全接入执行路径选择。
