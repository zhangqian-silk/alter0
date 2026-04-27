# Unit Test Cases

> Last update: 2026-04-26

本文件维护 Go 单元测试的领域覆盖口径。测试文件按 DDD 边界放置在对应包路径下：领域规则放在 `domain`，用例编排放在 `application`，CLI/Web 入口契约放在 `interfaces`，外部适配和基础设施契约放在 `infrastructure`。

每个 Go 包路径必须维护本路径的 `TEST_CASES.md`，记录覆盖范围、边界和执行命令。中央文档用于跨领域索引，本地 `TEST_CASES.md` 用于包内维护。

## 路径级说明

当前 Go 包路径均已补齐 `TEST_CASES.md`：

- `cmd/alter0/TEST_CASES.md`
- `internal/agent/application/TEST_CASES.md`
- `internal/codex/application/TEST_CASES.md`
- `internal/codex/domain/TEST_CASES.md`
- `internal/codex/infrastructure/localfile/TEST_CASES.md`
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
- `server_workspace_service_test.go`

用例范围：

- 登录保护、Chat 页面、静态资源、消息 JSON/SSE、Agent 消息入口。
- Control API：Channel、Capability、Skill、MCP、Agent、Cron、Environment、Runtime、LLM Provider。
- Workspace service 注册表：`/api/control/workspace-services` 的注册、查询、删除，以及 `<session_short_hash>.alter0.cn` / `<service>.<session_short_hash>.alter0.cn` 命中后的前端构建分发和 HTTP 反向代理；`travel` 服务额外覆盖 `<session_short_hash>.travel.alter0.cn` 的公开只读、免登录与 API 阻断。

边界：

- Workspace service 网关只覆盖共享运行时内的 Host 路由、静态分发与 HTTP 代理契约，不验证真实 Nginx/DNS 或浏览器跨域链路。
- 浏览器真实交互与完整页面联调仍由 `internal/interfaces/web/e2e` Playwright 套件覆盖。

### `internal/codex/domain`

覆盖文件：

- `auth_test.go`

用例范围：

- `auth.json` 的 OAuth / API Key 两类快照解析。
- 账号名、邮箱、用户 ID、账号 ID、套餐、身份键与推荐名称推导。

边界：

- 额度查询、token 刷新、切换策略与登录会话编排由 `internal/codex/application` 覆盖。
- 文件持久化与登录工作目录由 `internal/codex/infrastructure/localfile` 覆盖。

### `internal/codex/application`

覆盖文件：

- `service_test.go`

用例范围：

- 托管账号导入、当前活动账号识别、账号切换与备份生成。
- OAuth 账号状态刷新、额度回写与当前账号标记。
- 独立 `codex login` 会话启动、日志收集、成功保存与失败收口。

边界：

- `auth.json` 结构解析由 `internal/codex/domain` 覆盖。
- 本地文件持久化契约由 `internal/codex/infrastructure/localfile` 覆盖。

### `internal/codex/infrastructure/localfile`

覆盖文件：

- `store_test.go`

用例范围：

- 托管账号记录的保存、读取与列表加载。
- 账号根目录、备份目录与登录会话目录初始化。

边界：

- 账号业务规则、切换策略与登录会话状态由 `internal/codex/application` 覆盖。

## Agent Capability & Memory

### `internal/execution/infrastructure`

覆盖文件：

- `codex_cli_processor_test.go`
- `hybrid_nl_processor_test.go`

用例范围：

- Codex CLI 同步与流式执行、错误、空输出、认证失败和运行态心跳。
- 原生 Codex Runtime 资产生成、Session 工作区解析、repo root 与 session repo clone 模式。
- 直连 Codex 路径解析 `thread.started.thread_id`，持久化 `.alter0/codex-runtime/thread.json`，后续同 Session 使用 `codex exec resume <thread_id> -` 续写。
- Hybrid NL 的 Agent/ReAct/Codex 执行源、工具调用、委派与 Memory 工具。

边界：

- Web SSE 协议、会话落库与前端展示由 `internal/interfaces/web` 和前端测试覆盖。
- Terminal 自身的 Codex thread 恢复由 `internal/terminal/application` 覆盖。

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

### `cmd/alter0`

覆盖文件：

- `builtin_skills_test.go`
- `main_test.go`
- `supervisor_client_test.go`

用例范围：

- 内置 Skill 注册：`memory`、`deploy-test-service`、`frontend-design` 等默认 Skill 的描述、guide 与 file-backed 路径。
- 启动阶段 file-backed Skill 文件校验，确保 `.alter0/skills/deploy-test-service/SKILL.md` 与 `docs/skills/frontend-design/SKILL.md` 可被当前仓库解析。
- 运行时 PATH、NO_PROXY、Web 登录密码环境变量与内部启动参数过滤。
- supervisor client 重启错误、探活地址归一、空响应错误，以及 `sync_remote_master` 仅丢弃 tracked 改动并保留 untracked 工作区内容。

边界：

- 本路径测试只覆盖启动命令、内置 Skill 装配与 runtime supervisor 边界，不启动真实长期服务。
- 部署脚本、systemd、Nginx 与宿主签名凭据由 `docs/deployment` 与脚本级验证维护。

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
