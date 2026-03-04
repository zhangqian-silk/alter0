# Requirements

> Last update: 2026-03-04

状态说明：

1. `supported`：已在主干代码可用
2. `planned`：已确认方向，待排期
3. `ready`：需求已完成拆分与细化，进入可执行状态

## 需求列表

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
| R-012 | Web 侧边栏交互优化 | supported | 点击侧边栏项后进入“信息展示模式”：主区域仅展示对应信息，不再弹出或保留对话框；侧边栏在展开态提供显式折叠按钮 |
| R-013 | 流式传输能力 | supported | 提供端到端流式响应：后端增量推送生成内容，前端实时渲染并可感知进行中/完成/失败状态 |
| R-014 | 移动端真机适配增强 | supported | 优化小屏与键盘场景（安全区、输入区跟随、遮罩关闭与手势交互），确保 iOS/Android 浏览器下稳定可用 |
| R-015 | 移动端会话创建与信息完整性 | supported | 确保移动端可稳定新建会话，并完整展示会话必要信息（标题、入口状态、空态提示） |
| R-016 | 会话级并发控制与全局限流 | supported | 支持多会话并发处理，同时保证同一会话顺序一致，并提供系统级并发上限、排队与超时降级能力 |
| R-017 | 会话短期记忆 | supported | 在单会话内维护可控窗口的上下文记忆，提升多轮对话连续性与指代解析能力 |
| R-018 | 跨会话长期记忆 | supported | 支持跨会话沉淀用户偏好与长期事实，并按用户/租户范围检索后按相关性注入上下文 |
| R-019 | 会话内容持久化 | supported | 持久化用户/助手消息主数据与路由结果，支持重启恢复、会话分页与按时间范围检索 |
| R-020 | 上下文压缩 | supported | 超长上下文触发分层压缩，结构化回写摘要与关键事实并保留消息引用关系，降低长会话 token 成本 |
| R-021 | Skills/MCP 标准化能力模型 | supported | 统一 Skills 与 MCP 的配置结构、启停状态、作用域与校验规则，形成可治理的标准化能力层 |
| R-022 | 用户配置 Skills 接入 Codex | supported | 将启用的用户 Skills 以标准协议注入 Codex 执行上下文，支持选择、排序与冲突处理 |
| R-023 | 用户配置 MCP 接入 Codex | supported | 将用户配置的 MCP Server 安全映射到 Codex 运行配置，支持按会话/请求启用与审计追踪 |
| R-024 | 跨会话持久化记忆分级管理（参考 L1/L2/L3 Cache） | supported | 参考计算机缓存分层实现记忆分级：L1 高优先/低容量，L2 平衡层，L3 大容量归档层；按命中率与重要性动态迁移并分级限额 |
| R-025 | 天级记忆与长期记忆（Markdown 统一存储） | supported | 支持天级记忆落盘与长期记忆沉淀，并对每日记忆做压缩归档；R-017~R-024 的记忆数据统一以 Markdown 格式存储 |
| R-026 | 强制要求上下文文件（如 SOUL.md） | supported | 支持独立上下文文件存储用户强制要求，启动与会话初始化高优先级加载，并在冲突场景下覆盖普通记忆 |
| R-027 | Agent Memory 模块与页面收敛 | supported | 前端移除 `Workspace` 与 `Configuration` 页面；在 `Agent` 下新增 `Memory` 模块，可视化长期记忆、天级记忆与持久化记忆（`SOUL.md`） |
| R-028 | Memory 模块说明文档持久化与可视化 | supported | 新增记忆体系说明文档并持久化纳入仓库；前端 `Agent -> Memory` 提供文档视图入口，支持稳定查看 `USER.md`、`AGENTS.md`、`MEMORY.md`、`memory/YYYY-MM-DD.md`、`SOUL.md` 的职责说明与映射关系 |
| R-029 | 新对话空白会话唯一性约束 | supported | 前端与会话创建链路不允许生成多个“空白会话”；当已存在空白会话时，`New Chat` 必须复用并聚焦该会话，而不是继续新建 |
| R-030 | 会话与异步任务映射模型 | supported | 建立 `session_id` 与 `task_id` 的标准映射，支持长耗时请求异步化执行（快速应答 + 后台任务 + 任务日志回读），避免对话链路阻塞与上下文膨胀 |
| R-031 | 任务摘要跨会话记忆与按需深检索 | supported | 默认仅注入最近 3-5 条任务摘要控制上下文体积；当用户询问更早历史时自动切换深检索，从全量任务摘要库召回并按需下钻任务详情 |
| R-032 | `.alter0` 任务历史存储规范与 Memory 查阅 | ready | 统一任务运行态数据在 `.alter0` 下的目录结构、留存策略与回链规则；前端 `Agent -> Memory` 新增任务历史查阅能力（摘要默认可见、日志按需下钻） |
| R-033 | Control 任务观测台与流式日志 | planned | 在 `Control` 页面新增任务观测台，集中展示任务状态、进度、日志、产物与触发来源标识，支持流式日志观测与任务控制（retry/cancel） |

## 需求细化（草案）

### R-012 Web 侧边栏交互优化

1. 点击任一侧边栏入口后，主区域切换到对应信息面板，不显示对话框容器。
2. 侧边栏展开时显示“折叠”按钮；折叠后仅保留图标轨道。
3. 移动端在选择菜单项后自动折叠侧边栏，避免遮挡主内容。
4. 验收：连续切换 3 个侧边栏入口，页面不出现叠层对话框，且可随时展开/折叠。

#### Task Breakdown (auto-managed)

1. Scope split
   - Clarify boundaries under existing requirement description; do not create new requirement IDs.
2. Delivery plan
   - Split into M1/M2 milestones with concrete acceptance checks.
3. Status workflow
   - Keep lifecycle as planned -> ready -> supported; update this row only.
4. Traceability
   - Keep change notes and verification criteria inside this requirement section.

#### Traceability

- 实现文件：`internal/interfaces/web/static/chat.html`、`internal/interfaces/web/static/assets/chat.js`、`internal/interfaces/web/static/assets/chat.css`
- 测试覆盖：`internal/interfaces/web/server_sidebar_test.go`
- 验证命令：`GOSUMDB=sum.golang.org GOTOOLCHAIN=auto go test ./...`
- 验证记录：
  - 2026-03-03：点击侧边栏入口后进入信息展示模式，仅显示 `route-view`，不保留对话框容器。
  - 2026-03-03：桌面端侧边栏展开态显示“收起”按钮，折叠后进入图标轨道，支持再次“展开”。
  - 2026-03-03：移动端选择任一菜单项后自动收起侧边栏；连续切换 `Channels -> Skills -> MCP` 未出现叠层对话框，可随时展开/折叠。

### R-013 流式传输能力

1. 增加流式接口（建议 SSE），响应头为 `text/event-stream`，按片段持续推送。
2. 事件至少包含：`start`（开始）、`delta`（增量内容）、`done`（完成）、`error`（失败）。
3. 前端在流式过程中实时追加渲染，`done` 后收敛为完整消息；`error` 时展示可重试提示。
4. 保留非流式回退路径（旧接口或开关控制），确保兼容现有调用方。
5. 验收：一次正常流式响应可看到逐步输出；断流或异常时前端有明确失败状态。

#### Task Breakdown (auto-managed)

1. Scope split
   - Clarify boundaries under existing requirement description; do not create new requirement IDs.
2. Delivery plan
   - Split into M1/M2 milestones with concrete acceptance checks.
3. Status workflow
   - Keep lifecycle as planned -> ready -> supported; update this row only.
4. Traceability
   - Keep change notes and verification criteria inside this requirement section.

#### Traceability

- 实现文件：`internal/interfaces/web/server.go`、`internal/interfaces/web/server_message_test.go`、`internal/interfaces/web/static/assets/chat.js`、`internal/interfaces/web/static/assets/chat.css`
- 验证：`go test ./...`

### R-015 移动端会话创建与信息完整性

1. 在移动端视口中，`New Chat` 入口需始终可达（不被遮挡、不依赖桌面交互路径）。
2. 点击 `New Chat` 后必须立即创建会话并切换到新会话上下文。
3. 新建后至少展示会话基础信息：会话标题、空态文案、可发送输入区。
4. 当会话列表为空或加载失败时，给出明确提示，避免出现“空白但无反馈”。
5. 验收：在 iOS Safari 与 Android Chrome 下，连续 3 次新建会话均成功，且信息展示完整。

#### Traceability

- 实现文件：`internal/interfaces/web/static/chat.html`、`internal/interfaces/web/static/assets/chat.js`、`internal/interfaces/web/static/assets/chat.css`
- 测试覆盖：`internal/interfaces/web/server_mobile_session_test.go`
- 验证命令：`GOSUMDB=sum.golang.org GOTOOLCHAIN=auto go test ./...`
- 验证记录：
  - 2026-03-03：移动端头部新增常驻 `New Chat` 入口，未展开会话面板时仍可直接新建会话。
  - 2026-03-03：`New Chat` 连续触发后均即时创建新会话并切换上下文，展示会话标题、空态文案与输入区。
  - 2026-03-03：会话列表为空时展示“会话列表为空，点击 New Chat 创建会话。”；会话加载失败时展示“会话列表加载失败”错误提示。

### R-016 会话级并发控制与全局限流

1. 支持“多会话并发、同会话串行”模型：不同 `session_id` 可并发执行，同一 `session_id` 按入队顺序处理。
2. 增加全局并发上限（worker pool），超过上限的请求进入等待队列，并记录排队时长。
3. 队列提供超时与取消机制：超时返回明确错误码，并保留可观测日志字段（session_id、queue_wait_ms、timeout）。
4. 当系统进入高压状态时支持降级策略（拒绝新请求或快速失败），避免资源被耗尽。
5. 验收：并发压测下跨会话吞吐提升；同一会话无乱序响应；达到阈值时可观测到限流/超时事件。

#### Traceability

- 实现文件：`internal/orchestration/application/concurrent_service.go`、`internal/orchestration/application/service.go`、`cmd/alter0/main.go`、`internal/interfaces/web/server.go`、`internal/shared/infrastructure/observability/telemetry.go`
- 测试覆盖：`internal/orchestration/application/concurrent_service_test.go`
- 验证：`go test ./...`

### R-017 会话短期记忆

1. 在单个 `session_id` 维度维护短期记忆窗口（最近 N 轮消息 + 关键状态）。
2. 对引用关系（如“这个”“刚才那个方案”）提供会话内指代解析能力。
3. 支持短期记忆的 TTL 或轮次上限，避免上下文无限增长。
4. 验收：同一会话连续多轮追问时，模型能稳定引用前文关键内容。

#### Traceability

- 实现文件：`internal/orchestration/application/session_memory.go`、`cmd/alter0/main.go`
- 测试覆盖：`internal/orchestration/application/session_memory_test.go`、`internal/orchestration/application/service_test.go`
- 验证记录：`GOSUMDB=sum.golang.org GOTOOLCHAIN=auto go test ./...`

### R-018 跨会话长期记忆

1. 提供长期记忆实体模型（用户偏好、长期事实、约束规则等）并支持标签化管理。
2. 支持按用户/租户范围检索长期记忆并按相关性注入请求上下文。
3. 支持记忆更新策略（新增、覆盖、失效）与审计字段（来源会话、更新时间）。
4. 验收：新会话中可命中历史偏好并正确生效，且不污染无关用户数据。

#### Traceability

- 实现文件：`internal/orchestration/application/long_term_memory.go`、`internal/orchestration/application/service.go`
- 测试覆盖：`internal/orchestration/application/long_term_memory_test.go`、`internal/orchestration/application/service_test.go`
- 验证命令：`GOSUMDB=sum.golang.org GOTOOLCHAIN=auto go test ./...`
- 验证记录：
  - 2026-03-03：新增长期记忆实体模型，支持 `preference/fact/constraint` 分类、标签管理、审计字段（`source_session_id`、`updated_at`）与更新策略（`add/overwrite/invalidate`）。
  - 2026-03-03：同一用户在新会话请求中可按相关性命中历史偏好并注入 `[LONG TERM MEMORY]` 上下文。
  - 2026-03-03：跨用户写入后检索保持隔离，未出现租户内用户间记忆污染。

### R-019 会话内容持久化

1. 持久化消息主数据：`message_id`、`session_id`、角色、内容、时间戳、路由结果。
2. 支持服务重启后恢复历史会话，并可按会话分页查询。
3. 支持最小化索引能力（按 session_id、时间范围）以支撑检索与回放。
4. 验收：重启后会话内容不丢失，可按会话完整回放最近消息。

#### Traceability

- 实现文件：`internal/session/domain/message.go`、`internal/session/application/service.go`、`internal/orchestration/application/session_persistence_service.go`、`internal/storage/infrastructure/localfile/session_store.go`、`internal/interfaces/web/server.go`、`cmd/alter0/main.go`
- 测试覆盖：`internal/session/application/service_test.go`、`internal/orchestration/application/session_persistence_service_test.go`、`internal/storage/infrastructure/localfile/session_store_test.go`、`internal/interfaces/web/server_session_test.go`
- 新增接口：
  - `GET /api/sessions`：按会话维度分页查询，支持 `page`/`page_size`/`start_at`/`end_at`
  - `GET /api/sessions/{session_id}/messages`：按会话分页回放消息，支持时间范围过滤
- 验证命令：`GOSUMDB=sum.golang.org GOTOOLCHAIN=auto go test ./...`
- 验证记录：
  - 2026-03-03：消息链路新增持久化装饰器，落盘字段覆盖 `message_id/session_id/role/content/timestamp/route_result`。
  - 2026-03-03：服务重启后通过本地文件恢复会话历史，并可按会话分页查询与回放。
  - 2026-03-03：会话查询支持 `session_id + 时间范围` 索引检索，用于检索与历史重放。

### R-020 上下文压缩

1. 当上下文超过阈值时自动触发压缩（摘要 + 关键事实抽取）。
2. 压缩结果回写为结构化记忆片段，并保留原始消息引用关系。
3. 压缩策略需可配置（触发阈值、摘要长度、保留轮次）。
4. 验收：长会话下 token 消耗下降且回答质量不明显退化。

#### Traceability

- 实现文件：`internal/orchestration/application/session_memory.go`、`internal/orchestration/application/service.go`、`cmd/alter0/main.go`
- 测试覆盖：`internal/orchestration/application/session_memory_test.go`、`internal/orchestration/application/context_compression_acceptance_test.go`
- 运行参数：
  - `-context-compression-threshold`
  - `-context-compression-summary-tokens`
  - `-context-compression-retain-turns`
- 验证命令：`GOSUMDB=sum.golang.org GOTOOLCHAIN=auto go test ./...`
- 验证记录：
  - 2026-03-03：会话上下文超过阈值时自动触发压缩，生成结构化片段（`summary`、`key_facts`、`source_turn_refs`）并回写内存。
  - 2026-03-03：压缩片段保留原始消息引用关系（`user_message_id -> assistant_reply_ref`），多轮追问仍可解析“这个方案”等指代。
  - 2026-03-03：工程化验收 `TestContextCompressionAcceptanceTokenReductionAndQuality` 显示长会话提示词 token 估算从 `420` 降至 `275`（约 `34.52%`），质量评分退化不超过 1 个关键事实。

### R-021 Skills/MCP 标准化能力模型

1. 定义统一能力模型：`id`、`name`、`type`（skill/mcp）、`enabled`、`scope`、`metadata`。
2. 提供一致的配置校验与版本规则，避免不同能力类型出现结构漂移。
3. 支持标准化生命周期动作（启用、禁用、更新、删除）与变更审计。
4. 验收：同一套 API/存储约束可同时管理 Skills 与 MCP 能力对象。

#### Traceability

- 实现文件：`internal/control/domain/capability.go`、`internal/control/application/service.go`、`internal/storage/infrastructure/localfile/control_store.go`、`internal/interfaces/web/server.go`
- 前端展示：`internal/interfaces/web/static/assets/chat.js`
- 验证：`go test ./...`

### R-022 用户配置 Skills 接入 Codex

1. 根据用户配置筛选启用的 Skills，并按优先级注入 Codex 执行上下文。
2. 定义 Skills 注入协议（名称、描述、参数模板、约束）以避免自由文本拼接。
3. 提供冲突处理策略（同名 skill、重复能力、参数冲突）并可观测。
4. 验收：关闭某 Skill 后不再注入；开启后可在 Codex 执行结果中体现其能力影响。

#### Traceability

- 实现文件：`internal/execution/domain/skill_context.go`、`internal/execution/application/skill_context_resolver.go`、`internal/execution/application/service.go`、`internal/execution/infrastructure/codex_cli_processor.go`、`internal/orchestration/application/service.go`、`cmd/alter0/main.go`
- 测试覆盖：`internal/execution/application/service_test.go`、`internal/execution/infrastructure/codex_cli_processor_test.go`、`internal/orchestration/application/service_test.go`
- 协议约束：`alter0.skill-context/v1`，注入结构固定为 `protocol`、`skills[].name/description/parameter_template/constraints`、`resolved_parameters`、`conflicts`；通过 `alter0.skill_context` 元数据注入，禁止自由文本拼接。
- 验证命令：`GOSUMDB=sum.golang.org GOTOOLCHAIN=auto go test ./...`
- 验证记录：
  - 2026-03-03：仅注入 `enabled=true` 的 Skills，按 `skill.priority` 降序注入；关闭 Skill 后 `skills.injected_count` 与注入上下文立即收敛。
  - 2026-03-03：同名 Skill、重复能力、参数值冲突按优先级保留高优先级项，冲突明细写入 `skills.conflicts` 与 `skills.conflict_types`。
  - 2026-03-03：Codex 执行参数切换为结构化 JSON 载荷（`alter0.codex-exec/v1`），执行结果元数据可观测 `skills.injected_ids`、`skills.injected_count`、`skills.conflict_count`。

### R-023 用户配置 MCP 接入 Codex

1. 将用户配置的 MCP Server 映射到 Codex 可识别配置，支持 stdio 与 HTTP 两类传输。
2. 支持按会话或请求粒度启用 MCP，避免全局污染与越权调用。
3. 对 MCP 接入过程补齐安全控制：白名单、超时、失败隔离、访问审计。
4. 验收：用户新增 MCP 配置后，可在指定会话中被 Codex 正确调用，且日志可追踪。

#### Traceability

- 实现文件：`internal/execution/domain/mcp_context.go`、`internal/execution/application/mcp_context_resolver.go`、`internal/execution/application/service.go`、`internal/execution/infrastructure/codex_cli_processor.go`
- 测试覆盖：`internal/execution/application/service_test.go`、`internal/execution/infrastructure/codex_cli_processor_test.go`
- 协议约束：`alter0.mcp-context/v1`，注入结构固定为 `protocol`、`servers[].transport/command/args/url/headers/tool_whitelist/timeout_ms/failure_isolation`、`audit[]`；通过 `alter0.mcp_context` 元数据注入，执行载荷统一为 `alter0.codex-exec/v1`。
- 验证命令：`GOSUMDB=sum.golang.org GOTOOLCHAIN=auto go test ./...`
- 验证记录：
  - 2026-03-03：`mcp.transport` 支持 `stdio/http` 映射，分别输出 `command/args/env` 与 `url/headers` 到 Codex 执行载荷。
  - 2026-03-03：会话级启用通过 `alter0.mcp.session.enable` 绑定 `session_id`，请求级启用通过 `alter0.mcp.request.enable` 按次生效，跨会话不共享。
  - 2026-03-03：`mcp.tools.whitelist`、`mcp.timeout_ms`、`mcp.failure_isolation` 生效；异常配置按 Server 级隔离并写入 `mcp.audit`。
  - 2026-03-03：执行结果元数据新增 `mcp.injected_ids`、`mcp.injected_count`、`mcp.audit_count`、`mcp.audit`，日志输出 `mcp resolved` 支持访问审计追踪。

### R-024 跨会话持久化记忆分级管理（参考 CPU Cache）

1. 采用类缓存分层模型：`L1` 作为热记忆层（高价值、低延迟、最小容量），`L2` 作为平衡层（中等容量与时效），`L3` 作为归档层（大容量、低频访问）。
2. 每层独立配置约束：单条长度上限、层总容量上限、保留时长（TTL）、淘汰策略（优先 LRU，可扩展评分淘汰）。
3. 查询注入遵循缓存命中路径：优先命中 `L1`，再查 `L2/L3`；在 token 预算不足时按 `L1 > L2 > L3` 截断。
4. 支持“缓存迁移”机制：高频命中或被显式标记的重要记忆可晋升（`L3 -> L2 -> L1`）；长期未命中项可降级。
5. 支持写入策略配置（如 write-through / write-back 风格）：保证持久化一致性与性能平衡。
6. 验收：跨会话与重启后记忆可恢复；命中链路与晋升降级可观测；在相同 token 预算下高价值记忆命中率提升。

#### Traceability

- 实现文件：`internal/orchestration/application/long_term_memory.go`、`internal/orchestration/application/mandatory_context.go`、`cmd/alter0/main.go`
- 测试覆盖：`internal/orchestration/application/long_term_memory_test.go`、`internal/orchestration/application/tiered_long_term_memory_acceptance_test.go`、`internal/orchestration/application/service_test.go`
- 运行参数：
  - `-long-term-memory-path`
  - `-long-term-memory-write-policy`
  - `-long-term-memory-writeback-flush`
  - `-long-term-memory-token-budget`
- 验证命令：`GOSUMDB=sum.golang.org GOTOOLCHAIN=auto go test ./...`
- 验证记录：
  - 2026-03-03：长期记忆升级为 `L1/L2/L3` 分层模型，支持分层单条长度、总容量、TTL 与 `LRU/score` 淘汰策略。
  - 2026-03-03：查询注入固定命中链路 `L1 -> L2 -> L3`，并在 token 预算不足时按层级优先级截断，输出链路与预算元数据。
  - 2026-03-03：命中反馈支持高频晋升与长期未命中降级，显式 `memory_long_term_important=true` 可直接进入高优先层，晋升/降级计数可观测。
  - 2026-03-03：支持 `write_through/write_back` 写入策略与重启恢复，验收测试确认同等预算下高价值记忆命中率提升。

### R-025 天级记忆与长期记忆（Markdown 统一存储）

1. 支持天级记忆落盘：按日期切分存储（例如 `memory/YYYY-MM-DD.md`），记录当日会话沉淀与关键事实。
2. 支持长期记忆存储：将跨天稳定事实与偏好沉淀到长期记忆文件（例如 `MEMORY.md` 或 `memory/long-term/*.md`）。
3. 支持按天压缩：每日或定时将当日长对话压缩为摘要与关键事实，再回写到天级文件与长期记忆候选区。
4. 统一存储格式：包括 R-017~R-024 在内的记忆存储均使用 Markdown（`.md`），禁止引入独立二进制记忆格式。
5. 支持分级容量约束：按 L1/L2/L3 分别配置天级条目长度上限、层容量上限与保留时长。
6. 验收：连续多天运行后可按日期检索历史；当天压缩任务可稳定执行；长期记忆可跨会话命中且 Markdown 文件可直接审阅。

#### Traceability

- 实现文件：`internal/orchestration/application/session_memory.go`、`internal/orchestration/application/session_memory_daily_markdown.go`、`internal/orchestration/application/long_term_memory.go`、`internal/orchestration/application/memory_markdown_codec.go`、`cmd/alter0/main.go`
- 测试覆盖：`internal/orchestration/application/session_memory_test.go`、`internal/orchestration/application/long_term_memory_test.go`、`internal/orchestration/application/daily_long_term_memory_markdown_acceptance_test.go`
- 运行参数：
  - `-daily-memory-dir`
  - `-long-term-memory-path`
- 验证命令：`GOSUMDB=sum.golang.org GOTOOLCHAIN=auto go test ./internal/orchestration/application`
- 验证记录：
  - 2026-03-03：会话记忆新增天级 Markdown 落盘，按 `memory/YYYY-MM-DD.md` 记录 L1/L2/L3 分层内容、摘要与关键事实。
  - 2026-03-03：上下文压缩片段在当日自动回写到天级文件，并同步生成 `memory/long-term/YYYY-MM-DD.md` 长期记忆候选。
  - 2026-03-03：长期记忆持久化由 `.json` 升级为 `MEMORY.md` Markdown 存储，重启后可恢复并继续跨会话命中。
  - 2026-03-03：天级记忆支持分层单条长度、层容量与保留时长约束，超限或过期条目按层级策略自动清理。

### R-026 强制要求上下文文件（如 SOUL.md）

1. 支持独立的“强制要求”上下文文件（例如 `SOUL.md`），用于存储用户不可违背的长期指令与偏好。
2. 启动与会话初始化时优先加载该文件，并在请求构造时高优先级注入上下文。
3. 支持热更新：文件变更后可在后续请求中生效，并记录版本/更新时间用于审计。
4. 该文件与记忆体系隔离：不参与自动摘要压缩，不被 L1/L2/L3 淘汰策略清理。
5. 支持冲突处理规则：当强制要求与普通记忆冲突时，以强制要求文件为准并产生日志告警。
6. 验收：修改 `SOUL.md` 后新会话可立即体现强制要求；跨会话重启后仍稳定生效。

#### Traceability

- 实现文件：`internal/orchestration/application/mandatory_context.go`、`internal/orchestration/application/service.go`、`cmd/alter0/main.go`
- 测试覆盖：`internal/orchestration/application/mandatory_context_test.go`、`internal/orchestration/application/service_test.go`
- 验证命令：`GOSUMDB=sum.golang.org GOTOOLCHAIN=auto go test ./...`
- 验证记录：
  - 2026-03-03：启动参数新增 `-mandatory-context-file`，默认读取 `SOUL.md`，会话初始化时优先注入 `[MANDATORY CONTEXT]` 到请求构造。
  - 2026-03-03：上下文文件支持热更新，后续请求自动加载最新内容并输出 `mandatory_context.version`、`mandatory_context.updated_at`、`mandatory_context.loaded_at` 审计元数据。
  - 2026-03-03：与会话/长期记忆冲突时按键规则裁决，冲突记忆被过滤或覆盖，结果元数据记录 `mandatory_context.conflict_count` 并输出告警日志。
  - 2026-03-03：强制要求上下文独立于普通记忆注入链路，标记 `mandatory_context.isolated=true`，不进入记忆摘要与分层淘汰路径。

### R-027 Agent Memory 模块与页面收敛

1. 页面收敛：前端导航与路由中移除 `Workspace`、`Configuration` 两个页面入口，避免与 Agent 视图并行造成信息割裂。
2. Agent 模块扩展：在 `Agent` 下新增 `Memory` 页面入口，作为统一记忆可视化界面。
3. `Memory` 页面展示三类数据：
   - 长期记忆：展示跨会话长期记忆（含分层信息与更新时间）。
   - 天级记忆：按日期展示日记忆文件（如 `memory/YYYY-MM-DD.md`）及摘要。
   - 持久化记忆：展示 `SOUL.md` 当前生效内容与最近更新时间。
4. 交互要求：支持在同一页面完成三类记忆切换查看；移动端与桌面端均可达且不依赖隐藏入口。
5. 边界约束：`Memory` 页面默认只读展示，不在本需求内引入在线编辑能力。
6. 验收：侧边栏不再出现 `Workspace`、`Configuration`；`Agent -> Memory` 可稳定展示三类记忆数据，空态与加载失败态均有明确提示。

#### Task Breakdown (auto-managed)

1. Scope split
   - Remove `Workspace`/`Configuration` route and menu entries.
   - Introduce `Agent -> Memory` information architecture and rendering boundaries.
2. Delivery plan
   - M1: Navigation and routing convergence.
   - M2: Memory visualization for long-term, daily and `SOUL.md`.
3. Status workflow
   - Keep lifecycle as planned -> ready -> supported; update this row only.
4. Traceability
   - Keep implementation and verification records in this section after delivery.

### R-028 Memory 模块说明文档持久化与可视化

1. 新增记忆体系说明文档 `docs/memory/persistent-memory-module-spec.md`，作为项目内长期保留的产品与工程基线文档。
2. 文档明确五类核心文件职责与边界：`USER.md`（用户画像）、`AGENTS.md`（代理执行规范）、`MEMORY.md`（长期记忆）、`memory/YYYY-MM-DD.md`（天级记忆）、`SOUL.md`（强制上下文）。
3. 文档定义“持久化可见”原则：文档本身纳入仓库版本管理；运行时记忆数据继续落盘到 `.alter0/memory/*`，不并入业务代码提交。
4. `Agent -> Memory` 页面补充“说明文档”视图入口，支持稳定展示该文档全文，并可关联到上述文件类型的说明锚点。
5. 页面展示边界：本需求仅覆盖只读展示、结构化分段、空态与加载失败态，不引入在线编辑。
6. 验收：在同一页面可切换查看长期记忆、天级记忆、持久化记忆与说明文档四类视图，且说明文档内容与仓库文件保持一致。

#### Task Breakdown (auto-managed)

1. Scope split
   - Persist and version a dedicated memory module specification document.
   - Extend Memory module IA with a read-only `Documentation` view.
2. Delivery plan
   - M1: Finalize document schema, file responsibilities, and persistence boundaries.
   - M2: Align frontend Memory view contract with document visibility requirements.
3. Status workflow
   - Keep lifecycle as planned -> ready -> supported; update this row only.
4. Traceability
   - Keep implementation and verification records in this section after delivery.

#### Traceability

- 说明文档：`docs/memory/persistent-memory-module-spec.md`
- 关联需求：`R-017`、`R-018`、`R-025`、`R-026`、`R-027`
- 可视化目标：`Agent -> Memory`
- 实现文件：`internal/interfaces/web/agent_memory.go`、`internal/interfaces/web/server.go`、`internal/interfaces/web/static/assets/chat.js`
- 测试覆盖：`internal/interfaces/web/server_memory_test.go`
- 验证记录：
  - 2026-03-04：`Agent -> Memory` 已支持说明文档可视化与只读展示，`R-028` 状态回填为 `supported`。

### R-029 新对话空白会话唯一性约束

1. 定义“空白会话”：会话内没有任何用户消息与助手消息（`messages.length == 0`）。
2. `New Chat` 行为约束：当存在任意空白会话时，不得创建新会话；必须复用最近创建的空白会话并切换为活动会话。
3. 仅当当前不存在空白会话时，`New Chat` 才允许创建新的会话记录。
4. 会话删除或首条消息写入后，空白会话状态实时更新，后续 `New Chat` 按最新状态判定。
5. 页面刷新后规则保持一致：本地恢复会话列表后仍遵守“最多一个空白会话”的约束。
6. 验收：连续点击 `New Chat` 5 次，最多只保留 1 个空白会话；发送首条消息后再次点击 `New Chat` 才会生成下一空白会话。

#### Task Breakdown (auto-managed)

1. Scope split
   - Define blank-session detection and uniqueness contract.
   - Apply uniqueness rule to all New Chat entry points.
2. Delivery plan
   - M1: Frontend creation guard and focus reuse behavior.
   - M2: Recovery-path consistency after local persistence reload.
3. Status workflow
   - Keep lifecycle as planned -> ready -> supported; update this row only.
4. Traceability
   - Keep implementation and verification records in this section after delivery.

#### Traceability

- 目标模块：`internal/interfaces/web/static/assets/chat.js`
- 关联需求：`R-015`、`R-019`
- 实现文件：`internal/interfaces/web/static/assets/chat.js`
- 测试覆盖：`internal/interfaces/web/server_mobile_session_test.go`
- 验证口径：`New Chat` 连续触发、刷新恢复、首条消息后再创建
- 验证记录：
  - 2026-03-04：空白会话唯一性规则已生效，`R-029` 状态回填为 `supported`。

### R-030 会话与异步任务映射模型

1. 建立统一映射关系：一个 `session` 可关联多个 `task`（`1:N`），每个任务必须持有 `session_id` 与 `source_message_id`。
2. 新增异步判定门槛：预计耗时超过阈值或属于产物型/多步骤流程时，消息路由转为后台任务执行。
3. 对话层快速应答：命中异步后立即返回 `task_id` 与初始状态（`queued/running`），不阻塞会话响应。
4. 任务状态机标准化：`queued -> running -> success/failed/canceled`，并支持进度、重试、超时、取消。
5. 任务日志与产物持久化：后台执行写入阶段日志、错误原因、产物引用；对话模块可按 `task_id` 回读并生成用户可读摘要。
6. 结果回流规则：任务完成后将摘要结果回写原 `session`，保留 `task_id` 与消息关联，支持“查任务进度/查任务结果”指令。
7. 验收：同一会话连续发起多个长任务时，会话可持续交互；每个任务可独立查询状态、日志和最终结果，且不会导致会话上下文异常膨胀。

#### Task Breakdown (auto-managed)

1. Scope split
   - Define session-task data model and id mapping contract.
   - Define async routing trigger and response contract.
2. Delivery plan
   - M1: Task lifecycle APIs and persistence model.
   - M2: Chat readback flow and completion writeback behavior.
3. Status workflow
   - Keep lifecycle as planned -> ready -> supported; update this row only.
4. Traceability
   - Keep implementation and verification records in this section after delivery.

#### 接口拆分（草案）

1. 任务创建与快速应答
   - `POST /api/tasks`
   - 入参：`session_id`、`source_message_id`、`task_type`、`input`、`idempotency_key`、`async_hint`
   - 出参：`task_id`、`status`、`accepted_at`、`estimated_wait_ms`
   - 约束：同一 `idempotency_key` 重复提交时返回已存在任务，不重复创建。
2. 任务查询与列表
   - `GET /api/tasks/{task_id}`
   - `GET /api/tasks?session_id=&status=&page=&page_size=`
   - 能力：返回状态机节点、进度百分比、开始/结束时间、错误摘要。
3. 任务日志与产物查询
   - `GET /api/tasks/{task_id}/logs?cursor=&limit=`
   - `GET /api/tasks/{task_id}/artifacts`
   - 能力：按游标增量读取日志，产物返回可审计引用（文件路径、PR 链接、报告链接）。
4. 任务控制动作
   - `POST /api/tasks/{task_id}/cancel`
   - `POST /api/tasks/{task_id}/retry`
   - 约束：仅 `queued/running` 可取消，仅 `failed/canceled` 可重试。
5. 会话回读接口
   - `GET /api/sessions/{session_id}/tasks?latest=true`
   - 对话请求命中异步判定时，原会话响应必须附带 `task_id` 与 `task_status`，前端据此跳转任务详情或轮询。

#### 数据模型拆分（草案）

- `Task`
  - 关键字段：`task_id`、`session_id`、`source_message_id`、`task_type`、`status`、`progress`、`retry_count`、`timeout_at`
- `TaskLog`
  - 关键字段：`task_id`、`seq`、`stage`、`level`、`message`、`created_at`
- `TaskArtifact`
  - 关键字段：`task_id`、`artifact_type`、`uri`、`summary`、`created_at`
- `TaskMessageLink`
  - 关键字段：`task_id`、`session_id`、`request_message_id`、`result_message_id`

#### Traceability

- 核心对象：`session_id`、`task_id`、`source_message_id`
- 关联需求：`R-016`、`R-019`、`R-029`
- 验证口径：异步判定、快速应答、任务查询、结果回流
- 实现文件：`internal/task/domain/task.go`、`internal/task/application/service.go`、`internal/interfaces/web/server.go`
- 测试覆盖：`internal/task/application/service_test.go`、`internal/interfaces/web/server_task_test.go`
- 新增接口：
  - `POST /api/tasks`
  - `GET /api/tasks?session_id=&status=&page=&page_size=`
  - `GET /api/tasks/{task_id}/logs?cursor=&limit=`
  - `GET /api/tasks/{task_id}/artifacts`
  - `POST /api/tasks/{task_id}/retry`
- 验证命令：
  - `GOTOOLCHAIN=auto GOSUMDB=sum.golang.org go test ./internal/task/... ./internal/interfaces/web/...`
  - `GOTOOLCHAIN=auto GOSUMDB=sum.golang.org go test ./...`
- 验证记录：
  - 2026-03-04：任务主模型补齐 `source_message_id/task_type/timeout_at/message_link`，同一 `session_id` 下保持 `1:N` 任务映射并支持 `idempotency_key` 去重。
  - 2026-03-04：异步任务状态机覆盖 `queued -> running -> success/failed/canceled`，补齐重试、超时、取消转移与阶段日志序号化持久化。
  - 2026-03-04：会话链路支持快速应答返回 `task_id/task_status`，任务完成后摘要回写原会话并保留 `task_id -> request_message_id/result_message_id` 回链。
  - 2026-03-04：任务回读接口支持按会话筛选、按 `task_id` 查询详情、游标日志读取、产物读取与失败任务重试。

### R-031 任务摘要跨会话记忆与按需深检索

1. 任务完成后必须产出结构化摘要并写入跨会话记忆层，摘要至少包含：`task_id`、`task_type`、`goal`、`result`、`status`、`finished_at`、`tags`。
2. 默认注入策略：新会话与普通问答仅注入最近 `3-5` 条高价值任务摘要，不注入详细执行日志。
3. 深检索触发策略：当用户问题包含“更早/历史/之前/上周/首个任务”等历史信号时，自动切换到全量摘要库检索，不受 `3-5` 条默认窗口限制。
4. 命中回填策略：深检索命中的历史摘要临时注入当前回答上下文，并在回复中标注可继续按 `task_id` 下钻详情。
5. 详情下钻边界：仅在用户明确请求细节时读取任务日志/产物，避免详细信息默认进入对话上下文。
6. 验收：默认模式下上下文体积稳定；历史追问可召回早期任务摘要；按 `task_id` 可继续查询完整任务日志与产物。

#### 触发词词表（草案）

- 中文：`更早`、`之前`、`历史`、`上次`、`上周`、`上个月`、`最早`、`第一条`、`当时`、`那次`。
- 英文：`earlier`、`previous`、`history`、`last time`、`last week`、`first task`、`older`。
- 时间表达：显式日期（如 `2026-03-01`）、相对时间（如 `三天前`、`两周前`）。
- 指代增强：当句子同时包含 `任务/需求/PR/执行` 与历史词时，强制启用深检索。

#### 检索排序规则（草案）

1. 先按语义相关度排序（query 与 `goal/result/tags` 的匹配分）。
2. 再按业务价值加权：`success` 与关键失败（有明确原因/产物）优先于噪声记录。
3. 再按时间衰减：默认优先近 30 天；若用户给定时间范围，则以范围约束优先。
4. 最终去重：同一 `task_id` 多条摘要仅保留最高分版本。
5. 返回策略：默认返回 `top 5` 摘要；用户明确要求“全部”时可分页继续拉取。

#### 任务历史双轨记忆策略（草案）

1. 主数据轨（Task Log）：任务系统保存完整执行日志、阶段状态、错误信息与产物引用，作为审计与排障唯一真源。
2. 摘要轨（Task Summary Memory）：任务进入终态（`success/failed/canceled`）时自动生成结构化摘要，写入跨会话记忆与天级记忆。
3. 摘要写入位置：
   - 天级摘要：`memory/YYYY-MM-DD.md`
   - 长期候选：`memory/long-term/YYYY-MM-DD.md`
   - 摘要记录必须包含 `task_id`，用于回链主数据轨。
4. 默认检索路径：先检索摘要轨并返回精简结论；不直接加载任务日志细节。
5. 按需下钻路径：仅当用户明确要求“细节/日志/报错原因/完整过程”时，基于 `task_id` 回查主数据轨并补充回答。
6. 一致性保障：若摘要轨与主数据轨状态不一致，以主数据轨为准并触发摘要重建。

#### 误触发保护规则（草案）

1. 否定语义保护：出现“不要查历史/不看以前/只看当前”等否定表达时，禁止深检索。
2. 当前上下文优先：若问题明确锚定当前任务（如“这个任务现在进度”），即便包含历史词也保持默认注入模式。
3. 双信号门槛：仅当“历史词”与“任务域词（任务/需求/PR/执行）”同时出现，或用户明确指定时间范围时触发深检索。
4. 预算保护：当深检索命中结果过多时，先返回摘要分页提示，不一次性注入超量历史内容。
5. 回退机制：深检索无命中时，自动回退到默认 `3-5` 条注入并提示可换关键词重试。
6. 可观测指标：记录 `deep_retrieval_triggered`、`deep_retrieval_overridden`、`deep_retrieval_miss`，用于评估误触发率。

#### Task Breakdown (auto-managed)

1. Scope split
   - Define task summary schema for cross-session memory.
   - Define default injection and deep-retrieval trigger rules.
2. Delivery plan
   - M1: Summary writeback pipeline and default top-k injection.
   - M2: Query-intent deep retrieval and task-detail drill-down flow.
3. Status workflow
   - Keep lifecycle as planned -> ready -> supported; update this row only.
4. Traceability
   - Keep implementation and verification records in this section after delivery.

#### Traceability

- 实现文件：`internal/task/domain/task.go`、`internal/task/application/service.go`、`internal/tasksummary/application/store.go`、`internal/orchestration/application/service.go`、`internal/orchestration/application/task_summary_prompt.go`、`internal/shared/application/telemetry.go`、`internal/shared/infrastructure/observability/telemetry.go`、`cmd/alter0/main.go`
- 测试覆盖：`internal/tasksummary/application/store_test.go`、`internal/orchestration/application/service_test.go`、`internal/task/application/service_test.go`
- 验证命令：
  - `GOSUMDB=sum.golang.org GOTOOLCHAIN=auto go test ./internal/tasksummary/application ./internal/task/application ./internal/orchestration/application ./internal/scheduler/application ./cmd/alter0`
  - `GOSUMDB=sum.golang.org GOTOOLCHAIN=auto go test ./...`
- 验证记录：
  - 2026-03-04：任务进入终态后生成结构化摘要（`task_id/task_type/goal/result/status/finished_at/tags`），并写入跨会话任务摘要记忆层。
  - 2026-03-04：默认注入窗口固定最近 5 条任务摘要，不注入日志与产物详情，控制上下文体积。
  - 2026-03-04：历史追问命中后触发深检索并支持按需下钻；仅在明确“细节/日志/报错原因/完整过程”语义时注入日志与产物摘要。
  - 2026-03-04：误触发保护生效（否定语义、当前任务优先、双信号门槛、预算截断与 miss 回退），并输出 `deep_retrieval_triggered`、`deep_retrieval_overridden`、`deep_retrieval_miss` 指标。
- 核心对象：`task_summary`、`task_id`、`session_id`、`history_query_intent`
- 关联需求：`R-018`、`R-025`、`R-030`
- 验证口径：默认注入窗口、深检索命中率、误触发率、详情下钻可用性、摘要轨与主数据轨一致性

### R-032 `.alter0` 任务历史存储规范与 Memory 查阅

1. 任务历史运行态数据统一落盘到 `.alter0/`，默认不纳入仓库提交，避免业务代码与运行数据混杂。
2. 目录规范：
   - `.alter0/tasks/index.json`：任务索引（`task_id`、`session_id`、`source_message_id`、`status`、`created_at`、`finished_at`）
   - `.alter0/tasks/{task_id}/meta.json`：任务主信息与状态机快照
   - `.alter0/tasks/{task_id}/logs.jsonl`：任务阶段日志（append-only）
   - `.alter0/tasks/{task_id}/artifacts.json`：产物引用（PR/文件/报告）
   - `.alter0/memory/YYYY-MM-DD.md`：任务摘要天级记忆
   - `.alter0/memory/long-term/YYYY-MM-DD.md`：长期候选摘要
3. 回链规范：摘要记录必须包含 `task_id`，可从 Memory 视图一跳回查任务详情；任务详情页必须反向展示其摘要引用。
4. 留存策略：
   - `logs.jsonl` 按任务保留（默认 90 天，可配置）
   - `index.json` 与 `meta.json` 保留任务生命周期全量索引
   - 天级摘要长期可读，超期归档不删除 `task_id` 映射
5. 前端 `Agent -> Memory` 查阅规范：
   - 默认展示任务摘要列表（按时间倒序）
   - 支持筛选 `status/task_type/time_range`
   - 支持按 `task_id` 打开详情抽屉，按需加载 `logs/artifacts`
   - 默认不展开原始日志，避免信息过载
6. 空态与异常态：
   - 无任务历史时显示“暂无任务历史”
   - 日志缺失或文件损坏时显示可重建提示，不影响摘要列表展示
7. 验收：`Memory` 页面可查阅 `.alter0` 中任务摘要并下钻详情；同一 `task_id` 可在摘要与日志间双向跳转，且运行态数据不进入 Git 变更。

#### 后端接口拆解（MVP）

1. 任务摘要列表接口
   - `GET /api/memory/tasks?status=&task_type=&start_at=&end_at=&page=&page_size=`
   - 返回：`items[]`（摘要字段）+ `pagination`（分页游标/总数）
2. 任务详情接口
   - `GET /api/memory/tasks/{task_id}`
   - 返回：`meta`（状态机快照）+ `summary_refs`（摘要回链）
3. 任务日志接口（按需加载）
   - `GET /api/memory/tasks/{task_id}/logs?cursor=&limit=`
   - 返回：增量日志片段，支持游标续读
4. 任务产物接口（按需加载）
   - `GET /api/memory/tasks/{task_id}/artifacts`
   - 返回：产物列表（类型、链接、摘要、生成时间）
5. 任务摘要重建接口（运维/修复）
   - `POST /api/memory/tasks/{task_id}/rebuild-summary`
   - 用于处理摘要轨与主数据轨不一致场景

#### 前端字段清单（MVP）

1. 摘要列表项字段
   - `task_id`、`task_type`、`goal`、`result`、`status`、`finished_at`、`tags`
2. 详情抽屉基础字段
   - `task_id`、`session_id`、`source_message_id`、`status`、`progress`、`created_at`、`finished_at`、`retry_count`
3. 日志视图字段
   - `seq`、`stage`、`level`、`message`、`created_at`
4. 产物视图字段
   - `artifact_type`、`uri`、`summary`、`created_at`
5. 交互状态字段
   - `isLoading`、`isLogLoading`、`hasMoreLogs`、`errorCode`、`activeTaskId`

#### Task Breakdown (auto-managed)

1. Scope split
   - Define `.alter0` runtime task storage layout and retention rules.
   - Define Memory module read model for summary-first and drill-down.
2. Delivery plan
   - M1: Runtime persistence schema and task-summary linking.
   - M2: Frontend Memory task-history browse and detail loading behavior.
3. Status workflow
   - Keep lifecycle as planned -> ready -> supported; update this row only.
4. Traceability
   - Keep implementation and verification records in this section after delivery.

#### Traceability

- 规范文档：`docs/memory/runtime-task-memory-spec.md`
- 核心对象：`.alter0/tasks/*`、`.alter0/memory/*`、`task_id`
- 关联需求：`R-028`、`R-030`、`R-031`
- 验证口径：目录规范、摘要回链、前端可查阅、Git 隔离

### R-033 Control 任务观测台与流式日志

1. 入口位置：在 `Control` 一级导航新增 `Tasks` 页面，不放在 `Agent/Memory` 下。
2. 列表观测：任务列表支持按 `session_id`、`status`、`trigger_type`、`channel_type`、`time_range` 过滤，默认按最近更新时间倒序。
3. 详情观测：详情抽屉展示 `task_id`、`session_id`、状态机、进度、重试次数、错误信息、产物引用。
4. 来源标识：每个任务必须展示触发来源字段，至少包括 `trigger_type(user/cron/system)`、`channel_type(web/cli/scheduler)`、`channel_id`、`correlation_id`；若为定时任务还需展示 `job_id`、`job_name`、`fired_at`。
5. 流式日志：支持日志流式观测（SSE），首屏展示最近日志片段，断线后可按游标续读。
6. 控制动作：在详情页提供 `retry`、`cancel`，并明确状态约束（仅终态可重试、仅 `queued/running` 可取消）。
7. 回链能力：任务详情可跳转到对应会话消息；会话消息可回跳任务详情。
8. 验收：在同一页面可实时观察任务从 `queued -> running -> success/failed/canceled` 的状态变化与日志流，并可区分 user/cron/system 等来源，无需切换到 Memory 页面。

#### 接口拆分（草案）

1. 任务列表接口
   - `GET /api/control/tasks?session_id=&status=&trigger_type=&channel_type=&start_at=&end_at=&page=&page_size=`
2. 任务详情接口
   - `GET /api/control/tasks/{task_id}`（返回 `trigger_type`、`channel_type`、`channel_id`、`correlation_id`，cron 场景附带 `job_id/job_name/fired_at`）
3. 任务日志流接口（SSE）
   - `GET /api/control/tasks/{task_id}/logs/stream?cursor=`
4. 任务日志回补接口（断线续读）
   - `GET /api/control/tasks/{task_id}/logs?cursor=&limit=`
5. 任务控制接口
   - `POST /api/control/tasks/{task_id}/retry`
   - `POST /api/control/tasks/{task_id}/cancel`

#### Task Breakdown (auto-managed)

1. Scope split
   - Define Control task observability page information architecture and interaction model.
   - Define streaming log protocol, cursor resume, and status synchronization contract.
2. Delivery plan
   - M1: Control task list/detail/read model and API aggregation.
   - M2: SSE log stream, reconnect resume, and retry/cancel operation panel.
3. Status workflow
   - Keep lifecycle as planned -> ready -> supported; update this row only.
4. Traceability
   - Keep implementation and verification records in this section after delivery.

#### Traceability

- 核心对象：`task_id`、`session_id`、`trigger_type`、`channel_type`、`channel_id`、`correlation_id`、`status`、`progress`、`logs`、`artifacts`
- 依赖需求：`R-030`、`R-031`、`R-032`
- 验证口径：流式可观测性、状态一致性、断线续读、控制动作可用性

### R-014 移动端真机适配增强

1. 小屏安全区适配：移动端导航抽屉、会话面板、消息区与输入区统一纳入 `env(safe-area-inset-bottom)`，避免底部刘海/手势区遮挡。
2. 键盘跟随：基于 `VisualViewport` 实时计算软键盘占位高度并写入 `--keyboard-offset`，输入区在 iOS/Android 键盘弹起时持续可见。
3. 遮罩关闭与手势交互：支持点击遮罩、按下 `Escape`、以及在导航/会话抽屉内左滑手势关闭覆盖层。
4. 路由切换一致性：移动端点击侧边栏菜单后立即收起抽屉并进入目标信息页，防止面板残留遮挡主内容。
5. 验收：在移动端连续执行“打开抽屉 -> 切换页面 -> 输入框聚焦 -> 键盘收起”流程，页面无错位、无遮挡、可稳定关闭覆盖层。

#### Traceability

- 实现文件：`internal/interfaces/web/static/assets/chat.js`、`internal/interfaces/web/static/assets/chat.css`
- 验证：`go test ./...`
