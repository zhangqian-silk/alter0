# Unit Test Cases

> Last update: 2026-04-16

本文件维护 Go 单元测试的领域覆盖口径。测试文件按 DDD 边界放置在对应包路径下：领域规则放在 `domain`，用例编排放在 `application`，CLI/Web 入口契约放在 `interfaces`，外部适配和基础设施契约放在 `infrastructure`。

每个 Go 包路径必须维护本路径的 `TEST_CASES.md`，记录覆盖范围、边界和执行命令。中央文档用于跨领域索引，本地 `TEST_CASES.md` 用于包内维护。

## 路径级说明

当前 Go 包路径均已补齐 `TEST_CASES.md`：

- `cmd/alter0/TEST_CASES.md`
- `internal/agent/application/TEST_CASES.md`
- `internal/control/application/TEST_CASES.md`
- `internal/control/domain/TEST_CASES.md`
- `internal/execution/application/TEST_CASES.md`
- `internal/execution/domain/TEST_CASES.md`
- `internal/execution/infrastructure/TEST_CASES.md`
- `internal/interfaces/cli/TEST_CASES.md`
- `internal/interfaces/web/TEST_CASES.md`
- `internal/llm/application/TEST_CASES.md`
- `internal/llm/domain/TEST_CASES.md`
- `internal/llm/infrastructure/TEST_CASES.md`
- `internal/orchestration/application/TEST_CASES.md`
- `internal/orchestration/domain/TEST_CASES.md`
- `internal/orchestration/infrastructure/TEST_CASES.md`
- `internal/product/application/TEST_CASES.md`
- `internal/product/domain/TEST_CASES.md`
- `internal/scheduler/application/TEST_CASES.md`
- `internal/scheduler/domain/TEST_CASES.md`
- `internal/session/application/TEST_CASES.md`
- `internal/session/domain/TEST_CASES.md`
- `internal/shared/application/TEST_CASES.md`
- `internal/shared/domain/TEST_CASES.md`
- `internal/shared/infrastructure/id/TEST_CASES.md`
- `internal/shared/infrastructure/observability/TEST_CASES.md`
- `internal/storage/infrastructure/localfile/TEST_CASES.md`
- `internal/task/application/TEST_CASES.md`
- `internal/task/domain/TEST_CASES.md`
- `internal/tasksummary/application/TEST_CASES.md`
- `internal/terminal/application/TEST_CASES.md`
- `internal/terminal/domain/TEST_CASES.md`

## Runtime & Orchestration

### `internal/orchestration/domain`

覆盖文件：

- `contract_test.go`

用例范围：

- `Intent` 稳定类型：`command` 与 `nl` 保持为编排层意图分流契约。
- `CommandHandler` 契约：命令处理器接收 `CommandRequest`，保留 `UnifiedMessage`、命令名、参数并返回 `CommandResult` 输出与元数据。

边界：

- `CommandRegistry` 是端口接口，具体注册、别名解析、冲突处理由 `internal/orchestration/infrastructure` 和 `internal/orchestration/application` 覆盖。
- 领域包不绑定具体命令实现，不测试 `/help`、`/echo` 等命令文案。

### `internal/interfaces/cli`

覆盖文件：

- `runner_test.go`

用例范围：

- CLI 启动后生成稳定 `session_id`，跳过空输入，非空输入按 `cli-default`、`channel_type=cli`、`trigger_type=user` 转换为 `UnifiedMessage`。
- `/quit` 与 `/exit` 结束当前 CLI 循环，不进入编排层。
- 编排成功时输出结果文本，并记录 CLI gateway 指标。
- 编排失败时输出错误与错误编码，保持循环可继续直到退出命令。

边界：

- CLI 测试只覆盖输入适配契约，不验证具体命令执行逻辑。
- 终端交互通过 pipe 注入 stdin/stdout，避免依赖真实交互环境。

### `internal/interfaces/web`

覆盖文件：

- `server_chat_test.go`
- `server_control_test.go`
- `server_preview_test.go`
- `server_preview_control_test.go`

用例范围：

- 登录保护、Chat 页面、静态资源、消息 JSON/SSE、Agent/Product 消息入口。
- Control API：Channel、Capability、Skill、MCP、Agent、Product、Draft、Cron、Environment、Runtime、LLM Provider。
- 会话前端预览注册表：`/api/control/previews` 的注册、查询、删除，以及 `<session_short_hash>.alter0.cn` 命中后对 `/chat`、`/assets/*` 的工作区构建分发。

边界：

- 会话前端预览只覆盖共享运行时内的路由与静态分发契约，不验证真实 Nginx/DNS 或浏览器跨域链路。
- 浏览器真实交互与完整页面联调仍由 `internal/interfaces/web/e2e` Playwright 套件覆盖。

## Agent Capability & Memory

### `internal/execution/domain`

覆盖文件：

- `runtime_heartbeat_test.go`

用例范围：

- `WithRuntimeHeartbeatReporter` 在 context 中注册心跳回调。
- `EmitRuntimeHeartbeat` 在存在回调且时间戳为空时自动补 UTC 时间。
- 已传入的显式时间戳保持不变。
- nil context、缺少回调或 nil 回调不产生 panic，也不发送心跳。

边界：

- 心跳持久化、任务超时续租和 Web 展示归属 Task 与接口层测试。
- 执行端口、Skill、MCP、Memory Context 注入由 `internal/execution/application` 既有测试覆盖。

## Product Domain

### `internal/product/domain`

覆盖文件：

- `product_test.go`
- `travel_guide_test.go`

用例范围：

- `Product.Normalized`：产品 ID、slug、状态、可见性、来源、主 Agent、版本默认值、标签、产物类型、知识源与 worker agent 去重排序。
- `Product.Validate`：产品 ID 格式、名称、slug、状态、可见性、来源、版本、active Product 的主 Agent 约束。
- `ProductDraftGenerateInput`：草稿生成输入的名称、目标、列表字段、生成模式默认值与合法模式。
- `ProductDraft`：草稿 ID、Product、审核状态、主 Agent、worker matrix、发布目标的归一化与校验。
- `ProductAgentDraft` 与 `ProductWorkerDraft`：Agent ID、名称、角色、默认迭代次数和 worker 优先级。
- `TravelGuide`：travel product 默认绑定、城市、天数上限、内容、修订号、偏好列表归一化。
- `TravelGuideCreateInput` 与 `TravelGuideReviseInput`：创建/修订输入的城市、天数默认值、14 天上限、可选天数字段不反向修改原始输入。

边界：

- Product CRUD、内置/托管来源、发布链路和 Travel 服务编排由 `internal/product/application` 覆盖。
- HTML 页面、Workspace、公开 Product 接口由 `internal/interfaces/web` 覆盖。

## Task, Terminal & Workspace

### `internal/task/domain`

覆盖文件：

- `task_test.go`

用例范围：

- `TaskStatus` 与 `TaskLogLevel` 合法值、终态识别。
- `TaskSummary.IsZero` 与 `TaskSummary.Validate` 的必填字段、状态和完成时间。
- `Task.Validate` 的任务身份、会话、来源消息 fallback、状态、阶段、进度、队列字段、重试字段、超时字段、创建/更新时间和请求内容。
- 任务日志校验：序号、时间、级别、消息内容。
- 产物校验：artifact id 或 URI、名称、content type、大小和创建时间。
- 任务摘要与任务 ID 一致性。
- 任务-会话回链中的 task/session 一致性。

边界：

- 任务生命周期、复杂度分流、并发、心跳续租、retry/cancel、日志流和删除清理由 `internal/task/application` 与 Web 测试覆盖。
- Terminal 会话态与 turn/step 领域规则由 `internal/terminal/domain` 和 `internal/terminal/application` 既有测试覆盖。

## Conversation & Session Experience

### `internal/session/domain`

覆盖文件：

- `message_test.go`

用例范围：

- `MessageRole` 仅允许 `user` 与 `assistant`。
- `MessageSource` 允许空来源，也允许 `user/cron/system` 与 `cli/web/scheduler` 的稳定组合。
- `MessageSource` 拒绝未知 trigger 或 channel。
- `MessageRecord.Validate` 校验消息 ID、会话 ID、角色、内容、时间戳和来源字段。

边界：

- 会话持久化、历史查询、删除清理由 `internal/session/application` 与 Web handler 测试覆盖。
- 前端会话切换、草稿、滚动、移动端输入由 Playwright E2E 覆盖，见 `docs/testing/playwright.md`。

## Control, Operations & Governance

### `internal/llm/application`

覆盖文件：

- `model_config_service_test.go`

用例范围：

- `CreateDefaultConfig` 默认包含 OpenAI 与 OpenRouter 两类 Provider。
- 默认配置不启用 Provider，不设置默认 Provider，不持久化 API Key。
- OpenAI 默认 Provider 使用 OpenAI Compatible、`openai-responses` 与官方 base URL。
- OpenRouter 默认 Provider 使用 OpenRouter 类型、`openai-completions` 与 OpenRouter base URL。

边界：

- Provider 创建、更新、启停、默认项收敛和密钥保留语义由 `internal/llm/domain`、`internal/llm/infrastructure` 与 Web 控制面测试覆盖。

### `internal/shared/application`

覆盖文件：

- `contract_test.go`

用例范围：

- `IDGenerator` 端口保持 `NewID() string` 契约。
- `Telemetry` 端口保持 gateway、route、command、error、memory event 与 duration 观测契约。

边界：

- 具体随机 ID 生成由 `internal/shared/infrastructure/id` 覆盖。
- 具体 Prometheus metrics 输出由 `internal/shared/infrastructure/observability` 覆盖。

### `internal/shared/infrastructure/id`

覆盖文件：

- `random_test.go`

用例范围：

- `RandomIDGenerator.NewID` 输出 UTC 时间戳前缀。
- 正常随机源可用时追加 8 字节十六进制随机后缀。
- 随机源不可用时允许退回仅时间戳格式，调用方仍获得非空 ID。

边界：

- ID 全局唯一性不通过固定次数单测证明；依赖时间戳和随机熵组合，冲突治理由调用侧业务键或存储层处理。

### `internal/shared/infrastructure/observability`

覆盖文件：

- `telemetry_test.go`

用例范围：

- gateway、route、command、error、memory event、queue event 计数输出 Prometheus counter 行。
- route duration 与 queue wait 输出平均秒数。
- queue depth 与 worker in-flight 输出 gauge 行。
- 空样本场景仍输出 queue wait、queue depth、worker in-flight 的零值指标。
- 同一指标标签按字典序输出，便于稳定比对。

边界：

- 日志 JSON handler 只由 `NewLogger` 封装标准库 `slog`，不额外锁定格式细节。
- Prometheus 抓取、告警规则和部署接入归属部署/运维配置。

## 执行口径

- Go 单测默认执行：`GOCACHE=/tmp/alter0-go-build-cache go test ./...`。
- 覆盖率概览执行：`GOCACHE=/tmp/alter0-go-build-cache go test ./... -cover`。
- 当前受限运行环境中，Go 默认缓存目录 `/var/lib/alter0/.cache/go-build` 可能不可写；测试命令使用 `/tmp/alter0-go-build-cache` 作为临时构建缓存，不改变项目配置。
- Web E2E 执行口径见 `docs/testing/playwright.md`。
