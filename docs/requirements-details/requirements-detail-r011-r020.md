# Requirements Details (R-011 ~ R-020)

> Last update: 2026-03-30

## 需求细化（草案）

### R-011 本地文件存储

暂无细化内容。

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
3. 当前端流式会话在较长时间内没有新增 `delta` 时，服务端仍需持续发送 SSE 保活帧，避免浏览器、网关或代理因空闲超时提前断开连接。
4. 前端在流式过程中实时追加渲染，`done` 后收敛为完整消息；`error` 时展示可重试提示。
5. 保留非流式回退路径（旧接口或开关控制），确保兼容现有调用方。
6. 对已进入 Agent 执行链的流式请求，前端断连、页面切换、标签页隐藏或浏览器主动取消请求，只能终止当前回传连接，不能中断后端执行。
7. 验收：一次正常流式响应可看到逐步输出；长时间无增量时连接仍保持存活；断流或异常时前端有明确失败状态；Agent 流中途断开后，最终结果仍可在会话历史中回放。

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
- 验证记录：
  - 2026-03-31：`Chat / Agent / Product` 的 SSE 响应链路新增保活注释帧；Codex CLI 长时间静默执行阶段不会仅因消息流空闲而提前断开。
  - 2026-03-31：Agent 相关 SSE 请求改为与前端连接生命周期解耦；浏览器断开连接后，后端 Agent 执行继续完成并沿用原会话持久化链路。

### R-014 移动端真机适配增强

1. 小屏安全区适配：移动端导航抽屉、会话面板、消息区与输入区统一纳入 `env(safe-area-inset-bottom)`，避免底部刘海/手势区遮挡。
2. 键盘跟随：基于 `VisualViewport` 实时计算软键盘占位高度并写入 `--keyboard-offset`，同时同步移动端对话区有效可视高度；`Chat / Agent` 输入区在 iOS/Android 键盘弹起、收起和窗口尺寸变化时持续贴住可见底部。
3. 遮罩关闭与手势交互：支持点击遮罩、按下 `Escape`、以及在导航/会话抽屉内左滑手势关闭覆盖层。
4. 路由切换一致性：移动端点击侧边栏菜单后立即收起抽屉并进入目标信息页，防止面板残留遮挡主内容。
5. Chat 输入区收纳：移动端 `Chat`/`Agent` 输入区默认只保留消息输入，以及与发送主按钮同排的单一“会话设置”入口；`Provider / Model`、`Tools / MCP`、`Skills` 改为在同一弹层内按需展开，不再常显多组运行时信息、附注文案与字数计数。
6. 运行时文案一致性：`Chat` 输入区底部的 `Model`、`Tools / MCP`、`Skills` 与“会话设置”弹层在当前语言下需保持统一文案口径，不得出现中英混排或同一概念多套称呼并存。
7. 会话设置弹层布局稳定性：在窄宽度和长文案场景下，配置项标题、描述、Provider 标签与计数信息需支持截断或换行，不得发生重叠、覆盖或挤压到不可读。
8. Terminal 输入稳定性：移动端 `Terminal` 路由在输入框聚焦且键盘弹起期间，后台轮询刷新不得重建输入节点、清空草稿或把页面滚动位置重置到顶部；输入法每次确认词句后，只要输入框仍保持聚焦，前端继续延迟重绘。
9. 移动端功耗要求：`Chat / Agent / Terminal` 的前端轮询、`VisualViewport` 同步与本地持久化需根据页面可见性、输入聚焦、滚动活跃度自动降频；页面隐藏时不得继续维持高频刷新，恢复前台后需尽快补齐一次状态同步。
10. 验收：在移动端连续执行“打开抽屉 -> 切换页面 -> 输入框聚焦 -> 键盘收起”流程，页面无错位、无遮挡、可稳定关闭覆盖层；`Chat / Agent` 输入区在 `VisualViewport` 高度变化与窗口尺寸切换期间持续贴底、焦点输入框不被键盘遮挡；`Chat` 输入区不会因运行时信息常显而挤占主要可视空间，会话设置入口与发送按钮保持同排，文案不出现中英混排，配置弹层中的长标题与说明文案保持可读、不互相重叠；在 `Terminal` 输入过程中，轮询刷新与输入法候选确认均不触发回顶或输入抖动，失焦后再统一刷新视图；页面切到后台时移动端不再维持高频轮询。

#### Traceability

- 实现文件：`internal/interfaces/web/static/assets/chat.js`、`internal/interfaces/web/static/assets/chat.css`、`internal/interfaces/web/static/assets/chat-terminal.css`
- 验证：`go test ./...`
- 验证记录：
  - 2026-03-30：移动端 `Chat`/`Agent` 输入区改为紧凑模式，运行时配置收敛为单一“会话设置”入口；默认隐藏附注文案与字数计数，避免小屏与软键盘场景下发送区占满屏幕。
  - 2026-03-30：Chat 发送按钮与“会话设置”入口收敛到同一行；运行时配置文案统一为当前语言口径，去除 `Tools / Skills` 等中英混排。
  - 2026-03-30：Chat 会话设置弹层补充窄宽度文本约束，标题按可用宽度截断、说明文案允许换行，避免配置项文案重叠。
  - 2026-03-30：移动端 `Chat / Agent` 输入区改为基于 `VisualViewport` 同步有效可视高度；软键盘弹起、收起和窗口尺寸变化期间，输入区持续贴住可见底部，焦点输入框不被键盘遮挡。
  - 2026-03-31：移动端 `Chat / Agent / Terminal` 前端轮询与 `VisualViewport` 同步改为按页面可见性、输入聚焦与滚动活跃度自动降频；隐藏页停止高频刷新，恢复前台后立即补刷新，降低耗电与发热。

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
4. 支持按 `session_id` 删除会话持久化历史；若存在 `.alter0/workspaces/sessions/<session_id>` 旧式会话工作区或其下任务工作区，删除时需同步清理，不残留孤儿目录。
5. 验收：重启后会话内容不丢失，可按会话完整回放最近消息；显式删除后相关历史不再保留，遗留会话工作区若存在也会被同步清理。

#### Traceability

- 实现文件：`internal/session/domain/message.go`、`internal/session/application/service.go`、`internal/orchestration/application/session_persistence_service.go`、`internal/storage/infrastructure/localfile/session_store.go`、`internal/interfaces/web/server.go`、`cmd/alter0/main.go`
- 测试覆盖：`internal/session/application/service_test.go`、`internal/orchestration/application/session_persistence_service_test.go`、`internal/storage/infrastructure/localfile/session_store_test.go`、`internal/interfaces/web/server_session_test.go`
- 新增接口：
  - `GET /api/sessions`：按会话维度分页查询，支持 `page`/`page_size`/`start_at`/`end_at`
  - `GET /api/sessions/{session_id}/messages`：按会话分页回放消息，支持时间范围过滤
  - `DELETE /api/sessions/{session_id}`：删除会话历史，并触发关联工作区清理
- 验证命令：`GOSUMDB=sum.golang.org GOTOOLCHAIN=auto go test ./...`
- 验证记录：
  - 2026-03-03：消息链路新增持久化装饰器，落盘字段覆盖 `message_id/session_id/role/content/timestamp/route_result`。
  - 2026-03-03：服务重启后通过本地文件恢复会话历史，并可按会话分页查询与回放。
  - 2026-03-03：会话查询支持 `session_id + 时间范围` 索引检索，用于检索与历史重放。
  - 2026-03-30：新增 `DELETE /api/sessions/{session_id}`，删除 Chat / Agent 会话时同步清理会话历史与 `.alter0/workspaces/sessions/<session_id>`。
  - 2026-03-30：Chat / Agent 默认执行目录恢复为 `.alter0/workspaces/sessions/<session_id>`；Async Task 默认执行目录恢复为 `.alter0/workspaces/sessions/<session_id>/tasks/<task_id>`，继续与会话删除联动清理。

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
