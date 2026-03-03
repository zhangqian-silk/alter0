# Requirements

> Last update: 2026-03-03

状态说明：

1. `supported`：已在主干代码可用
2. `planned`：已确认方向，待排期
3. `ready`：需求已完成拆分与细化，进入可执行状态

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
| R-012 | Web 侧边栏交互优化 | ready | 点击侧边栏项后进入“信息展示模式”：主区域仅展示对应信息，不再弹出或保留对话框；侧边栏在展开态提供显式折叠按钮 |
| R-013 | 流式传输能力 | supported | 提供端到端流式响应：后端增量推送生成内容，前端实时渲染并可感知进行中/完成/失败状态 |
| R-014 | 移动端真机适配增强 | supported | 优化小屏与键盘场景（安全区、输入区跟随、遮罩关闭与手势交互），确保 iOS/Android 浏览器下稳定可用 |
| R-015 | 移动端会话创建与信息完整性 | planned | 确保移动端可稳定新建会话，并完整展示会话必要信息（标题、入口状态、空态提示） |
| R-016 | 会话级并发控制与全局限流 | supported | 支持多会话并发处理，同时保证同一会话顺序一致，并提供系统级并发上限、排队与超时降级能力 |
| R-017 | 会话短期记忆 | planned | 在单会话内维护可控窗口的上下文记忆，提升多轮对话连续性与指代解析能力 |
| R-018 | 跨会话长期记忆 | planned | 支持跨会话沉淀用户偏好与长期事实，并按范围检索注入上下文 |
| R-019 | 会话内容持久化 | planned | 将会话消息、元数据与状态持久化存储，支持重启恢复与历史查询 |
| R-020 | 上下文压缩 | planned | 对超长上下文进行分层压缩与摘要回写，在控制 token 成本的同时保留关键信息 |
| R-021 | Skills/MCP 标准化能力模型 | supported | 统一 Skills 与 MCP 的配置结构、启停状态、作用域与校验规则，形成可治理的标准化能力层 |
| R-022 | 用户配置 Skills 接入 Codex | planned | 将启用的用户 Skills 以标准协议注入 Codex 执行上下文，支持选择、排序与冲突处理 |
| R-023 | 用户配置 MCP 接入 Codex | planned | 将用户配置的 MCP Server 安全映射到 Codex 运行配置，支持按会话/请求启用与审计追踪 |
| R-024 | 跨会话持久化记忆分级管理（参考 L1/L2/L3 Cache） | planned | 参考计算机缓存分层实现记忆分级：L1 高优先/低容量，L2 平衡层，L3 大容量归档层；按命中率与重要性动态迁移并分级限额 |
| R-025 | 天级记忆与长期记忆（Markdown 统一存储） | planned | 支持天级记忆落盘与长期记忆沉淀，并对每日记忆做压缩归档；R-017~R-024 的记忆数据统一以 Markdown 格式存储 |
| R-026 | 强制要求上下文文件（如 SOUL.md） | planned | 支持独立上下文文件存储用户强制要求，启动时高优先级加载并对后续会话持续生效 |

### 2.1 需求细化（草案）

#### R-012 Web 侧边栏交互优化

1. 点击任一侧边栏入口后，主区域切换到对应信息面板，不显示对话框容器。
2. 侧边栏展开时显示“折叠”按钮；折叠后仅保留图标轨道。
3. 移动端在选择菜单项后自动折叠侧边栏，避免遮挡主内容。
4. 验收：连续切换 3 个侧边栏入口，页面不出现叠层对话框，且可随时展开/折叠。
##### Task Breakdown (auto-managed)

1. Scope split
   - Clarify boundaries under existing requirement description; do not create new requirement IDs.
2. Delivery plan
   - Split into M1/M2 milestones with concrete acceptance checks.
3. Status workflow
   - Keep lifecycle as planned -> ready -> supported; update this row only.
4. Traceability
   - Keep change notes and verification criteria inside this requirement section.

#### R-013 流式传输能力

1. 增加流式接口（建议 SSE），响应头为 `text/event-stream`，按片段持续推送。
2. 事件至少包含：`start`（开始）、`delta`（增量内容）、`done`（完成）、`error`（失败）。
3. 前端在流式过程中实时追加渲染，`done` 后收敛为完整消息；`error` 时展示可重试提示。
4. 保留非流式回退路径（旧接口或开关控制），确保兼容现有调用方。
5. 验收：一次正常流式响应可看到逐步输出；断流或异常时前端有明确失败状态。
##### Task Breakdown (auto-managed)

1. Scope split
   - Clarify boundaries under existing requirement description; do not create new requirement IDs.
2. Delivery plan
   - Split into M1/M2 milestones with concrete acceptance checks.
3. Status workflow
   - Keep lifecycle as planned -> ready -> supported; update this row only.
4. Traceability
   - Keep change notes and verification criteria inside this requirement section.

##### Traceability

- 实现文件：`internal/interfaces/web/server.go`、`internal/interfaces/web/server_message_test.go`、`internal/interfaces/web/static/assets/chat.js`、`internal/interfaces/web/static/assets/chat.css`
- 验证：`go test ./...`

#### R-015 移动端会话创建与信息完整性

1. 在移动端视口中，`New Chat` 入口需始终可达（不被遮挡、不依赖桌面交互路径）。
2. 点击 `New Chat` 后必须立即创建会话并切换到新会话上下文。
3. 新建后至少展示会话基础信息：会话标题、空态文案、可发送输入区。
4. 当会话列表为空或加载失败时，给出明确提示，避免出现“空白但无反馈”。
5. 验收：在 iOS Safari 与 Android Chrome 下，连续 3 次新建会话均成功，且信息展示完整。

#### R-016 会话级并发控制与全局限流

1. 支持“多会话并发、同会话串行”模型：不同 `session_id` 可并发执行，同一 `session_id` 按入队顺序处理。
2. 增加全局并发上限（worker pool），超过上限的请求进入等待队列，并记录排队时长。
3. 队列提供超时与取消机制：超时返回明确错误码，并保留可观测日志字段（session_id、queue_wait_ms、timeout）。
4. 当系统进入高压状态时支持降级策略（拒绝新请求或快速失败），避免资源被耗尽。
5. 验收：并发压测下跨会话吞吐提升；同一会话无乱序响应；达到阈值时可观测到限流/超时事件。

##### Traceability

- 实现文件：`internal/orchestration/application/concurrent_service.go`、`internal/orchestration/application/service.go`、`cmd/alter0/main.go`、`internal/interfaces/web/server.go`、`internal/shared/infrastructure/observability/telemetry.go`
- 测试覆盖：`internal/orchestration/application/concurrent_service_test.go`
- 验证：`go test ./...`

#### R-017 会话短期记忆

1. 在单个 `session_id` 维度维护短期记忆窗口（最近 N 轮消息 + 关键状态）。
2. 对引用关系（如“这个”“刚才那个方案”）提供会话内指代解析能力。
3. 支持短期记忆的 TTL 或轮次上限，避免上下文无限增长。
4. 验收：同一会话连续多轮追问时，模型能稳定引用前文关键内容。

#### R-018 跨会话长期记忆

1. 提供长期记忆实体模型（用户偏好、长期事实、约束规则等）并支持标签化管理。
2. 支持按用户/租户范围检索长期记忆并按相关性注入请求上下文。
3. 支持记忆更新策略（新增、覆盖、失效）与审计字段（来源会话、更新时间）。
4. 验收：新会话中可命中历史偏好并正确生效，且不污染无关用户数据。

#### R-019 会话内容持久化

1. 持久化消息主数据：`message_id`、`session_id`、角色、内容、时间戳、路由结果。
2. 支持服务重启后恢复历史会话，并可按会话分页查询。
3. 支持最小化索引能力（按 session_id、时间范围）以支撑检索与回放。
4. 验收：重启后会话内容不丢失，可按会话完整回放最近消息。

#### R-020 上下文压缩

1. 当上下文超过阈值时自动触发压缩（摘要 + 关键事实抽取）。
2. 压缩结果回写为结构化记忆片段，并保留原始消息引用关系。
3. 压缩策略需可配置（触发阈值、摘要长度、保留轮次）。
4. 验收：长会话下 token 消耗下降且回答质量不明显退化。

#### R-021 Skills/MCP 标准化能力模型

1. 定义统一能力模型：`id`、`name`、`type`（skill/mcp）、`enabled`、`scope`、`metadata`。
2. 提供一致的配置校验与版本规则，避免不同能力类型出现结构漂移。
3. 支持标准化生命周期动作（启用、禁用、更新、删除）与变更审计。
4. 验收：同一套 API/存储约束可同时管理 Skills 与 MCP 能力对象。

##### Traceability

- 实现文件：`internal/control/domain/capability.go`、`internal/control/application/service.go`、`internal/storage/infrastructure/localfile/control_store.go`、`internal/interfaces/web/server.go`
- 前端展示：`internal/interfaces/web/static/assets/chat.js`
- 验证：`go test ./...`

#### R-022 用户配置 Skills 接入 Codex

1. 根据用户配置筛选启用的 Skills，并按优先级注入 Codex 执行上下文。
2. 定义 Skills 注入协议（名称、描述、参数模板、约束）以避免自由文本拼接。
3. 提供冲突处理策略（同名 skill、重复能力、参数冲突）并可观测。
4. 验收：关闭某 Skill 后不再注入；开启后可在 Codex 执行结果中体现其能力影响。

#### R-023 用户配置 MCP 接入 Codex

1. 将用户配置的 MCP Server 映射到 Codex 可识别配置，支持 stdio 与 HTTP 两类传输。
2. 支持按会话或请求粒度启用 MCP，避免全局污染与越权调用。
3. 对 MCP 接入过程补齐安全控制：白名单、超时、失败隔离、访问审计。
4. 验收：用户新增 MCP 配置后，可在指定会话中被 Codex 正确调用，且日志可追踪。

#### R-024 跨会话持久化记忆分级管理（参考 CPU Cache）

1. 采用类缓存分层模型：`L1` 作为热记忆层（高价值、低延迟、最小容量），`L2` 作为平衡层（中等容量与时效），`L3` 作为归档层（大容量、低频访问）。
2. 每层独立配置约束：单条长度上限、层总容量上限、保留时长（TTL）、淘汰策略（优先 LRU，可扩展评分淘汰）。
3. 查询注入遵循缓存命中路径：优先命中 `L1`，再查 `L2/L3`；在 token 预算不足时按 `L1 > L2 > L3` 截断。
4. 支持“缓存迁移”机制：高频命中或被显式标记的重要记忆可晋升（`L3 -> L2 -> L1`）；长期未命中项可降级。
5. 支持写入策略配置（如 write-through / write-back 风格）：保证持久化一致性与性能平衡。
6. 验收：跨会话与重启后记忆可恢复；命中链路与晋升降级可观测；在相同 token 预算下高价值记忆命中率提升。

#### R-025 天级记忆与长期记忆（Markdown 统一存储）

1. 支持天级记忆落盘：按日期切分存储（例如 `memory/YYYY-MM-DD.md`），记录当日会话沉淀与关键事实。
2. 支持长期记忆存储：将跨天稳定事实与偏好沉淀到长期记忆文件（例如 `MEMORY.md` 或 `memory/long-term/*.md`）。
3. 支持按天压缩：每日或定时将当日长对话压缩为摘要与关键事实，再回写到天级文件与长期记忆候选区。
4. 统一存储格式：包括 R-017~R-024 在内的记忆存储均使用 Markdown（`.md`），禁止引入独立二进制记忆格式。
5. 支持分级容量约束：按 L1/L2/L3 分别配置天级条目长度上限、层容量上限与保留时长。
6. 验收：连续多天运行后可按日期检索历史；当天压缩任务可稳定执行；长期记忆可跨会话命中且 Markdown 文件可直接审阅。

#### R-026 强制要求上下文文件（如 SOUL.md）

1. 支持独立的“强制要求”上下文文件（例如 `SOUL.md`），用于存储用户不可违背的长期指令与偏好。
2. 启动与会话初始化时优先加载该文件，并在请求构造时高优先级注入上下文。
3. 支持热更新：文件变更后可在后续请求中生效，并记录版本/更新时间用于审计。
4. 该文件与记忆体系隔离：不参与自动摘要压缩，不被 L1/L2/L3 淘汰策略清理。
5. 支持冲突处理规则：当强制要求与普通记忆冲突时，以强制要求文件为准并产生日志告警。
6. 验收：修改 `SOUL.md` 后新会话可立即体现强制要求；跨会话重启后仍稳定生效。

#### R-014 移动端真机适配增强

1. 小屏安全区适配：移动端导航抽屉、会话面板、消息区与输入区统一纳入 `env(safe-area-inset-bottom)`，避免底部刘海/手势区遮挡。
2. 键盘跟随：基于 `VisualViewport` 实时计算软键盘占位高度并写入 `--keyboard-offset`，输入区在 iOS/Android 键盘弹起时持续可见。
3. 遮罩关闭与手势交互：支持点击遮罩、按下 `Escape`、以及在导航/会话抽屉内左滑手势关闭覆盖层。
4. 路由切换一致性：移动端点击侧边栏菜单后立即收起抽屉并进入目标信息页，防止面板残留遮挡主内容。
5. 验收：在移动端连续执行“打开抽屉 -> 切换页面 -> 输入框聚焦 -> 键盘收起”流程，页面无错位、无遮挡、可稳定关闭覆盖层。

##### Traceability

- 实现文件：`internal/interfaces/web/static/assets/chat.js`、`internal/interfaces/web/static/assets/chat.css`
- 验证：`go test ./...`
