# Technical Solution

> Last update: 2026-06-30

`alter0` 的技术方案按与需求清单一致的领域模型维护。后续新增或调整需求时，技术方案必须落到对应领域与子域，不再按时间顺序、任务编号或零散专题堆叠。

## 维护规则

- 需求归属先确定领域路径，例如 `runtime.orchestration.intent`、`runtime.execution.codex-exec`、`task.workspace.runtime`。
- 技术方案使用相同领域路径补充实现信息，包括包边界、核心对象、调用链路、接口契约、存储形态、错误处理、观测字段和测试策略。
- 一个能力只允许有一个主归属领域；跨领域影响通过“依赖与边界”记录，不复制成多个重复方案。
- 影响架构边界、数据结构、接口、执行链路、存储、部署或研发治理的需求变更，必须同步更新本文件。
- 用户可见行为变化继续同步 `README.md`，稳定需求口径同步 `docs/requirements.md`，实现细节同步 `docs/requirements-details/*.md`。

## 领域方案索引

| 领域 | 主包/模块 | 技术方案重点 |
| --- | --- | --- |
| Runtime & Orchestration | `internal/interfaces`、`internal/shared`、`internal/orchestration`、`internal/execution/domain`、`internal/scheduler` | 统一消息、意图路由、Runtime Resolver、CLI Runtime、调度触发、观测与健康检查 |
| Conversation & Session Experience | `internal/interfaces/web`、`internal/session`、Web static assets | Chat/Chat 会话、历史隔离、移动端视口、消息渲染 |
| Skill & Memory | `internal/runtime`、`internal/execution`、`internal/orchestration`、`docs/skills`、`.alter0/memory` | CLI Runtime 上下文注入、Skill 仓库、Memory Context、会话摘要、长期记忆、记忆维护任务 |
| Task, Terminal & Workspace | `internal/task`、`internal/tasksummary`、`internal/terminal`、`.alter0/workspaces` | 异步任务、日志流、心跳、产物交付、Terminal 会话、工作区隔离 |
| Control, Operations & Governance | `internal/control`、`internal/codex`、`cmd/alter0`、`scripts`、`docs/deployment` | 控制面配置、Model Provider、Claude Code provider profile、Codex Runtime、运行时重启、部署凭据、测试与 TDD 约束 |

## Runtime & Orchestration

### 包边界

- `internal/interfaces/*` 负责外部输入适配，只生成内部统一消息，不承载业务路由。
- `internal/shared/domain` 承载 `UnifiedMessage`、`OrchestrationResult` 等跨领域消息对象。
- `internal/orchestration/domain` 承载 `Intent`、`Command` 等编排领域模型。
- `internal/orchestration/application` 承载意图识别、命令路由、Agent 执行分发。
- `internal/execution/domain` 定义执行端口和运行时上下文契约。
- `internal/scheduler` 负责 Cron 配置、触发与回注编排。

### 调用链路

```text
CLI / Web / Cron
  -> UnifiedMessage
  -> Orchestrator
  -> CommandHandler | ExecutionPort
  -> RuntimeResolver
  -> Claude Code + provider profile | Codex Direct
  -> OrchestrationResult
  -> Interface response / Session persistence / Task handoff
```

### 技术约束

- 外部输入必须先归一为 `UnifiedMessage`，再进入编排层。
- 命令路由优先于复杂度评估和模型执行；显式 `alter0.execution.engine=codex` 的消息会在编排层绕过命令路由，使 Codex CLI 内置斜线输入作为直连 Codex 内容进入 `ExecutionPort`。
- Cron 触发不直接调用执行器，必须复用编排链路。
- Cron runs 接口通过 Session history 按 `trigger_type=cron` 与 `job_id` 查询触发会话，不另建独立运行记录存储。
- `ExecutionPort` 是 Agent 执行能力的稳定边界；具体执行由 `RuntimeResolver` 选择 CLI Runtime。
- `RuntimeResolver` 按优先级选择执行器：空执行器、`auto` 或显式 `alter0.execution.engine=codex` 进入 `Codex Direct`；显式 `alter0.execution.engine=claude` 或携带 `alter0.llm.provider_id` 时解析对应 Model Provider 并进入 `Claude Code + provider profile`；Provider 不可用时进入 `Codex Direct`；Claude 执行失败不自动回退。
- Claude Code 运行前注入 `CLAUDE.md`、provider profile 环境、Skill、Memory、MCP 和工作区事实；Codex Direct 运行前注入 `AGENTS.md`、独立 `CODEX_HOME`、Skill、Memory、MCP 和工作区事实。
- 运行过程统一归一为 `RuntimeTraceEvent`：`source` 仅允许为 `provider / adapter / alter0`，分别表示底层 SDK/CLI 直接提供、工程 adapter 从稳定协议字段转换、alter0 本地确定性生成。事件 `kind / role / lifecycle / status / blocks / action` 不允许通过自然语言正文、标题、关键词或语言模式推断；Claude Code 与 Codex Direct 后续接入新事件时必须先在 adapter 层落成确定映射，再暴露给 Chat / Terminal。
- trace、session、message、correlation 字段贯穿日志、指标、会话与任务。

### 验证策略

- 领域对象测试覆盖消息字段归一、路由结果和错误编码。
- 编排应用测试覆盖命令优先、自然语言分发、Cron 回注。
- 接口测试覆盖 CLI/Web 输入到 `UnifiedMessage` 的转换。
- Go 单测用例说明按 `docs/testing/unit-test-cases.md` 与各 Go 包路径下的 `TEST_CASES.md` 维护，并按 Runtime、Conversation、Skill、Task、Control 领域路径归档。

## Conversation & Session Experience

### 包边界

- `internal/session/domain` 定义会话与消息数据结构。
- `internal/session/application` 负责会话持久化、历史查询和删除清理。
- `internal/interfaces/web` 负责 HTTP、SSE、Web 登录、页面路由和前端静态资源分发。
- `internal/interfaces/web/frontend` 负责 Web Shell 的 Vite + React 构建、legacy DOM shell 渲染和 `static/dist` 产物输出。
- `scripts/build_alter0_service.sh` 是服务二进制的统一构建入口：先在 `internal/interfaces/web/frontend` 执行 `npm run build`，校验 `static/dist/index.html` 中的哈希 JS/CSS 资产引用，再执行 `go build -o <target> ./cmd/alter0`。`start_alter0_service.sh`、`relaunch_service.sh`、`make build` 与 Runtime supervisor 的候选二进制构建必须复用该入口，supervisor 通过 `ALTER0_BUILD_OUTPUT` 指定候选输出路径，避免 Go 服务重建时嵌入过期前端产物。
- `internal/interfaces/web` 在输出主 Web Shell 与 `frontend_dist` workspace preview HTML 前，会扫描 `/assets/index-*.js|css` 引用，读取对应 JS/CSS 内容并注入 `?v=<content-hash>`；已有固定 `?v=` 会被服务端内容 hash 替换。页面继续使用 `no-cache`，构建资产继续使用长期 immutable 缓存，但重启、快进或预览刷新后只要资产内容变化，浏览器会命中新 URL，避免旧 bundle 因 immutable 缓存残留。
- `internal/interfaces/web/frontend/src/shared/api/client.ts` 负责统一 JSON 请求封装、错误收敛与登录失效回调，避免新前端页面继续散落原生 `fetch`。
- `internal/interfaces/web/frontend/src/shared/session/sessionHash.ts` 负责运行页会话短标识生成、短 hash 判定与短 hash 到完整会话 id 的前端列表解析；`Chat / Terminal` 的运行页 URL 参数统一使用该入口把完整会话 id 派生为 8 位短 hash，左侧会话列表不展示短 hash，完整会话 id 与 Terminal `terminal_session_id` 仅保留在接口、持久化、Details 与工作区路径语义中。
- `internal/interfaces/web/frontend/src/shared/time/format.ts` 负责固定 `Asia/Shanghai` 的前端显示时区与标准时间格式，避免新旧页面时间口径漂移；管理页中需要分钟精度的额度重置、运行时间等时间戳也必须复用这里的共享格式器，而不是在页面组件里手写 UTC 文案。
- `internal/interfaces/web/frontend/src/shared/time/sessionListGroups.ts` 负责把运行页会话列表按 `Pinned / Today / Yesterday / Earlier` 分组；调用方通过 `getPinned` 把置顶会话抽到独立首组，其余会话再按最近时间分组，避免 Chat 与 Terminal 漂移成不同的分组策略。
- `internal/interfaces/web/frontend/src/shared/viewport/mobileViewport.ts` 负责移动端断点、键盘偏移阈值、composer 贴底偏移与 viewport baseline 计算，避免 Chat、Terminal 与 route 页重复维护软键盘占位逻辑。初始同步、页面刷新或 WebView 恢复过程中，若没有输入框聚焦且上一帧没有键盘证据，状态机以 layout viewport 作为 baseline，不把短暂或持续偏小的 visual viewport 报告升级为键盘态；已有 baseline 且上一帧存在键盘证据或当前输入聚焦时，仍保留键盘诊断偏移。
- `internal/interfaces/web/frontend/src/shared/debug/clickDiagnostics.ts` 负责显式开关控制的前端点击诊断。入口只在 `?debug_clicks=1`、`?debugClicks=true` 或 `localStorage["alter0.debug.clicks"]="on"` 命中时注册 capture 监听与 `PerformanceObserver`，输出 `[alter0:click]` 与 `[alter0:longtask]` 控制台记录，默认路径不注册全局事件监听。
- `internal/interfaces/web/frontend/src/shared/visibility/usePageActivation.ts` 是 Chat / Terminal 的共享页面激活补偿入口，监听 `focus`、`visibilitychange`、`pageshow` 与 `online`，在页面可见时立即触发当前运行页的数据回源，并通过短 debounce 合并同一恢复阶段的重复事件。
- `internal/interfaces/web/frontend/index.html`、`static/dist/index.html` 与登录页模板统一以 `html[lang="en"]` 启动；`src/app/WorkbenchApp.tsx` 通过写回 `document.documentElement.lang` 统一驱动中英文壳层文案切换。`renderLoginPage` 继续直接输出服务端 HTML，但视觉与文案已对齐工作台基线：复用 `IBM Plex Sans + Sora` 字体组合、近白卡片表面与安全入口 copy。
- `internal/interfaces/web/frontend/src/features/conversation-runtime` 承载共享对话运行态 UI：`ConversationRuntimeProvider.tsx` 负责复用 Terminal session API 的数据模型和状态机，按 route 通过 `/api/chat/sessions` 或 `/api/terminal/sessions` 完成会话创建/切换/删除/置顶、input 输入提交、结果收口、文本草稿与附件草稿恢复、模型与能力项选择。`RuntimeRouteHost.tsx` 将 `/chat` 与 `/terminal` 都挂载到 `ConversationRuntimeProvider + ConversationWorkspace`，但通过 `key={route}` 和真实 `route` 保证 controller、API owner、active session、缓存和草稿命名空间彼此独立；`/terminal` 不再挂载 `ReactManagedTerminalRouteBody`。`ConversationWorkspace.tsx` 不再渲染 Skill 选择器、Deliverables、Session Profile 或独立 Skill 面板；历史会话的 `target_type=runtime` 仅作为会话摘要元数据保留。Chat 仍在常规模型 Provider 列表外额外注入前端内置 `Codex` provider，未持久化其他模型选择时默认选中 Codex；模型/Provider、Tools/MCP、Skills 与过程披露过滤按 route 写入独立 `localStorage` 配置，发送时随当前 owner input payload 提交有效 `model_provider_id / model_id / tool_ids / mcp_ids / skill_ids`。运行页会话列表与消息详情通过当前 route 的 session 接口恢复，浏览器继续保存当前 route 的未发送草稿、附件草稿和运行配置；Chat owner 与 Terminal owner 的会话列表互不包含。单会话详情若携带 `turns_paging.has_more_before`，Provider 会先按消息 id 合并当前分页片段，再使用 `turns_paging.next_before_turn_id / oldest_turn_id` 继续请求 `turn_before` 后台补齐更早 turns；无重叠片段按消息时间判断前置或追加，避免 page-activation、轮询或输入返回的短 turns 页覆盖已加载历史。Provider 将时间线/会话 workspace 上下文与 Composer 上下文分离，后台 `turn_before` 补页只更新消息集合；Composer 上下文只消费稳定草稿、附件和运行配置字段，避免未变化的输入区随历史合并刷新。Provider 的模块级运行态缓存保留完整已加载消息并按 24 小时 TTL 失效；完整会话消息快照、轻量会话信息快照、active session、文本草稿、附件草稿、运行配置与过程披露过滤均使用 route 独立 `sessionStorage / localStorage` key，旧合并快照只作为读取迁移 fallback。`hydrateRuntimeSession` 在无 `turn_before` 的普通详情恢复中先检查当前 route 缓存，只有 `serverBacked=true`、`messagesLoaded=true`、`turnsPaging.has_more_before=false` 且没有可恢复 assistant/user-only 状态时，才直接返回缓存会话并跳过重复详情请求；分页补历史、待恢复状态和未知分页边界仍继续回源。
- `ConversationWorkspace.tsx` 的长历史可见窗口使用 `timelineWindow + pendingHistoryScrollRestoreRef` 维护：按钮点击和滚动触顶都调用同一 `loadEarlierMessages`，先记录当前可见消息节点的 `data-message-id` 与相对容器 top 偏移，再记录 `scrollHeight / scrollTop` 作为兜底；扩展一个本地消息批次后优先按同一消息节点锚点恢复阅读坐标，并在下一帧二次校正，只有锚点不可测时才回退到高度差恢复。当前批次恢复完成前，同一会话的后续触顶 scroll 事件直接合并，避免连续事件把窗口一次性扩到全量、把阅读区强制带回顶部，或因 Markdown/图片高度回流导致可见内容向上突变。
- Chat 的 `ScrollJumpStrip` 只使用 `.runtime-message-user[data-message-id]` 作为跳转测量目标，保证 `上一条 / 下一条` 都按用户消息定位；assistant 消息、Thinking / Process 披露区和过程步骤不参与 Chat 阅读定位目标计算。Terminal 仍按自身 turn/item selector 维持终端输出定位。
- `internal/interfaces/web/frontend/src/features/shell/components/runtimeTraceEvents.ts` 定义前端统一运行事件模型：`RuntimeTraceEvent` 是 Chat / Terminal `Process` 展示、过滤与 meta 的唯一数据结构，并通过 `raw.ref / raw.type / raw.has_detail` 承载事件详情索引。`normalizeRuntimeTraceEvents` 只接收服务端 canonical event 数组，不再从 Terminal 专属步骤结构转换。`runtimeTraceEventVisibleByFilter` 负责按 `important_text / plan / reasoning / tools / commands / system` 过滤；`runtimeTraceEventDisclosureCategory / runtimeTraceEventDisclosureLabel` 负责给 Thinking / Process 事件行输出同源分类标签。默认过滤器为 `important_text`，由 `ConversationRuntimeProvider.tsx` 持有并持久化到浏览器本地配置，`ConversationWorkspace.tsx` 在模型与 Skill 配置面板中渲染勾选项，`ChatMessageRegion.tsx` 在装配消息时间线时只渲染命中过滤器的事件。
- `internal/interfaces/web/frontend/src/app` 承载当前 Web Shell 的顶层壳层：`App.tsx` 只负责挂载 `WorkbenchApp`；`WorkbenchApp.tsx` 负责主导航、路由分派、语言切换与桌面/移动导航态，并让 Settings 管理页在窄屏下通过 `data-route-mobile-head` 输出共享 `Menu` 入口；`routeState.ts` 负责 canonical path 路由解析和派发，工作台顶层运行路由只保留 `/chat` 与 `/terminal`，旧 `/chat` 映射到 Chat。`PrimaryNav.tsx` 消费 `NAV_GROUPS` 渲染三条主工作流且不再输出 `Workspace` 可见标题，并在当前运行页注册 `runtimeSessionRail` 时直接渲染 `Sessions / New` 会话列表；会话列表在主导航内通过 `.nav-session-rail` 叠加更紧凑的列表项样式，条目主体只展示标题，`busy` 会话显示 loading，尾侧保留轻量删除动作。样式层把主导航菜单约束收窄到直系 `.primary-nav.has-session-rail > .menu`，并让 `.nav-session-rail .runtime-session-list` 在桌面与移动端都固定使用独立 `overflow-y: scroll` 与 `scrollbar-gutter: stable both-edges`，避免嵌套列表继承主菜单 flex/滚动规则或在滚动阈值临界点改变 `Sessions / New` 位置。底部固定渲染 `Settings` 工具入口，点击后进入 `/settings`。`ReactManagedRouteBody.tsx` 仅把 `settings` 暴露为设置页 route body，Settings 页消费 `SETTINGS_ROUTE_GROUPS` 在页内输出 `Skills / Control / Settings` 分组切换，再以本地状态挂载对应设置分区 body，不改写浏览器 path；`RoutePageFrame` 在 Settings 路由输出 `.workbench-route-frame` flex 外壳，并让 `.route-body` 成为 `flex: 1` 的内部滚动容器；`.route-head` 与 `RuntimeWorkspaceHeader` 共同输出 `.workbench-title-head` / `.workbench-title-leading` 语义 class，Settings 使用 `.route-title-marker` 和单行 `h3` 对齐 Chat / Terminal 的主面板与 42px 级紧凑标题栏节奏，副标题只保留为 route copy 数据、不进入可见标题栏；`RuntimeWorkspaceShell` 的移动端边缘操作输出图标 SVG 与 `.sr-only` 文案，Settings 的 `data-route-mobile-head` 使用同规格菜单图标和居中 route 标题，CSS 在 `760px` 及以下把运行页移动 workbar 固定为 48px 级触控高度，将边缘图标入口收敛为 `border: 0`、`background: transparent`、`box-shadow: none` 的无边框工具按钮，并隐藏 Settings 的第二行 `.route-head`；每个分区入口复用 `NavIcon`、`abbr` 短标识和 `aria-current` 活动态，样式层通过 `.route-body[data-route="settings"]` 解除 legacy 自适应卡片栅格影响，桌面以左侧 sticky 索引 + 右侧内容流渲染，`760px` 及以下回落为双列入口栅格；`shell.css` 末尾通过 `Gemini-style flat workbench reset` 覆盖主工作区、Settings frame、设置分区、表格、详情面板、空态、代码块与移动端抽屉等旧表面样式，移除多余外层圆角、边框和厚阴影，只保留必要分割线、低对比选中态和 Composer 胶囊，同时对 Settings 的 frame、标题、正文区和分区索引声明 `animation: none`、`transition: none` 与 `transform: none`，避免 Settings 出现区别于 Chat / Terminal 的页面进入动效；设计稿与导出的 PNG 保存在 `docs/design/workbench-flat-redesign.html`、`docs/design/workbench-flat-redesign.png` 和 `docs/design/workbench-flat-redesign-*.png`；CSS contract test 固定该扁平工作台 reset 必须位于旧圆角表面之后生效，并覆盖 Settings 静态路由契约；`WorkbenchContext.tsx` 暴露当前 `route / language / navigate`、移动端主导航状态与开关，以及运行页向主导航注册会话列表的 `setRuntimeSessionRail`。`WorkbenchApp.tsx` 按 route 缓存最近一次有效 `runtimeSessionRail`，运行页切换时在新页面注册前先复用该 route 最近列表，避免公共 `Sessions / New` chrome 闪烁或回退到占位。根壳层通过 `app-shell[data-workbench-route]` 输出当前路由，设置页额外通过 `data-route-family="settings"` 暴露页族钩子，各子能力继续通过 `data-route` 暴露页级钩子。
- `ConversationRuntimeProvider.tsx` 接受 `chat / terminal` conversation route：读取本地活动会话、最近快照、服务端集合、单会话详情和用户手动 focus 时都会使用当前 route owner 的会话模型；历史 `chat` snapshot 与短 hash query 在加载时迁移为 Chat 会话，Terminal 则继续恢复 Terminal owner 会话。`session_id` query 只作为显式恢复输入和用户手动 focus 的输出；当 `/chat` 或 `/terminal` 没有显式 `session_id` 时，Provider 以当前 owner 的服务端集合与本地最近快照合并后的第一条会话作为当前会话，不让其他 route 的活动会话覆盖最新入口。会话列表在前端按 `pinned -> created_at` 归一排序；对尚未进入 Session history 的空白本地会话，pin 请求失败时仍更新本地会话快照，保证当前工作区即时反馈不被 history 缺口吞掉。
- `internal/session/application.Service` 在 Session summary 中输出 `last_active_at` 与 `pinned`。`SetSessionPinned` 持久化置顶 metadata，`TouchSession` 写入活跃时间 metadata，`CleanupInactiveSessions` 按固定阈值扫描会话并返回删除、置顶跳过和扫描统计；没有显式活跃时间的历史会话使用最后消息时间参与排序和清理判断。
- `internal/interfaces/web/maintenance.go` 负责系统维护任务：服务启动时创建每日记忆维护与会话清理调度循环；`RunMemoryMaintenance` 通过编排入口发送系统消息并注入 `memory-maintenance` Skill；`RunSessionCleanup` 调用 Session application 清理超过 7 天不活跃且未置顶的会话，并同步删除任务关联与 Session workspace。
- `ConversationRuntimeProvider.tsx` 的恢复判定识别本地或远端存在的未完成 assistant，以及当前活动会话最后一条消息仍是 user 的窗口期；恢复流程在要求稳定 assistant 时必须等到详情接口返回非占位 assistant 或失败态后才 upsert。
- `internal/orchestration/application/SessionPersistenceService` 将非 Terminal 入口的会话落库拆为请求开始与结果收口两段：`Handle` 进入下游执行前先追加本轮 `user` 记录，执行完成后追加 assistant 记录及 route、错误码和结构化过程。Terminal/Chat 当前运行页以 Terminal turn store 为恢复来源。
- `internal/interfaces/web/server_terminal.go` 对 Chat owner 与 Terminal owner 的输入使用服务端请求生命周期控制 runtime turn；浏览器刷新或前端主动断开请求只结束当前 HTTP 回传，不取消已进入 runtime 的执行，前端恢复链路依赖当前 owner 的 session 详情补拉最终结果。
- `ConversationWorkspace.tsx` 的移动端发送手势链路继续复用共享 `RuntimeComposer` submit capture，但在 `chat / chat` 路由下会先检查当前聚焦的主 textarea，并在直达 `sendPrompt` 前主动 `blur()`；发送动作与软键盘回弹依赖浏览器动态视口恢复和 workspace grid footer 回贴完成，不再由键盘偏移变量驱动主布局。
- `ConversationWorkspace.tsx` 额外负责把 Conversation 会话态归一为共享 `statusTone`：当前 assistant 消息为 `streaming / queued / running / in_progress` 时输出 `busy`，显式错误、失败、取消或 `message.error` 输出 `failed`，其余稳定态输出 `ready`；同一派生结果驱动会话列表项和 workspace header。会话列表只消费 `busy` 并渲染 loading，其他状态不渲染行内状态灯；header 可见层只保留信号本身，状态名称仅通过无障碍名称与悬浮提示暴露，避免头部长期固定显示 `Ready`。
- `ConversationRuntimeProvider.tsx` 的发送链路始终确保存在当前 route owner 的 runtime session，再调用对应 owner 的 input 接口，并用返回的 session 或后续详情恢复生成本轮 user / assistant 消息；前端不再创建本地 running process step，不再解析 SSE。收到 input 结果、错误或恢复详情后同步收口最终状态，迟到结果不得重新打开已完成消息或覆盖最终正文。
- `ConversationRuntimeProvider.tsx` 维护模块级 `conversationRuntimeCache`，专门覆盖同一 SPA 工作台内从 Chat / Terminal 切到其他路由再返回的个人单设备恢复场景。缓存按 route 保存会话列表、活动会话 id 和每个会话完整已加载消息或 turns，写入时复制消息、附件和 process events，读取时按 24 小时 TTL 校验；命中时作为 Provider 初始状态，随后当前 owner 的集合接口继续回源并沿现有合并规则覆盖。当前活动会话若已经带有 `has_more_before=false` 的完整稳定消息缓存，前台恢复、切回会话和普通详情恢复不会再次请求同一会话详情；未知分页状态、仍有更早页、运行中、失败恢复和 user-only 状态继续按单会话详情回源。Provider 同步写入当前 route 的 24 小时 `localStorage` 完整消息快照，用于刷新、重开或 `sessionStorage` 丢失后的首屏恢复；同时写入当前 route 只含会话元数据的轻量快照，用于完整消息快照超出浏览器配额或被清理后的会话列表兜底。本地快照不作为跨设备事实来源，删除、置顶、输入返回、列表刷新和详情刷新仍通过 React state 统一回写当前 route 缓存并等待服务端回源确认。
- `internal/interfaces/web/server_terminal.go` 的 session collection、item、pin、delete 与 input handlers 由 Chat owner 和独立 Terminal owner 复用；Chat 请求必须走 `/api/chat/sessions` 并落到 `chat` owner，Terminal 请求走 `/api/terminal/sessions` 并落到 `terminal` owner。session payload 的 `pinned` 字段作为前端恢复契约始终输出明确布尔值，取消置顶响应保留 `pinned:false`，避免浏览器本地旧快照把缺字段合并回置顶态。当前 Web Shell 的 `/terminal` path 使用共享 Conversation UI，但仍触发 Terminal owner 前端请求。前端将当前 owner 的 `turn.prompt` 映射为用户消息，将 `turn.final_output` 映射为 assistant 正文，将 `turn.runtime_trace_events` 映射为运行过程。
- `internal/interfaces/web/frontend/src/features/shell/components/RuntimeTimelinePrimitives.tsx` 只维护运行页时间线里的共享展示原语：`RuntimeAttachmentGallery` 统一渲染用户图片附件与 Terminal turn 附件缩略图。`RuntimeTimeline.tsx` 统一输出 `runtime-message / runtime-message-user / runtime-message-assistant / runtime-message-bubble` 契约，Conversation runtime 的 `Process` 消费 `RuntimeTraceEvent`，默认只渲染 `Thinking / 已思考` 披露行。`RuntimeProcessStepMeta.tsx` 是共享事件行 meta 渲染入口，统一输出类型标签、耗时与 `Ready / Busy / Failed / Interrupted` 状态样式；`RuntimeProcessDetailBlocks.tsx` 是共享过程详情渲染入口，直接消费 `RuntimeTraceEvent.blocks` 或 event detail API 返回的 `blocks`。Conversation runtime 通过 `ConversationRuntimeProvider.loadProcessEventDetail` 在步骤展开时按当前 owner、`raw.ref` 或事件 id 拉取 detail 并写回当前消息的 `processEvents`；`ReactManagedTerminalRouteBody.tsx` 作为兼容组件继续维护同类 detail cache，但不参与当前 `/terminal` Shell 挂载。组件统一输出 `.terminal-rich-block`，其中 `terminal / code / diff / tool_input` 与 JSON 类 `tool_output` 输出 `.terminal-rich-pre.terminal-step-content`，`text / markdown / thinking / tool_output(text) / error / log` 输出 `MessageMarkdownHTML` + `.terminal-step-content.terminal-step-richtext`，并统一保留标题、文件名与起始行号，避免任何过程详情首帧先按临时普通 Markdown 显示。`runtime-process-answer`、`terminal-final-rendered` 等差异样式只作为变体 class 挂载在 block 外层，不再把页面私有 wrapper 重新塞回 markdown HTML 字符串。
- `internal/interfaces/web/frontend/src/features/shell/components/MessageMarkdown.ts` 负责消息正文 markdown 安全渲染前的输入归一化：除常规 `CRLF -> LF` 外，还需剔除零宽断行字符，并把“每字一行”的病态段落折回单段文本，避免 provider 或 adapter 写入异常换行后在真机上继续显示为逐字竖排。
- `internal/interfaces/web/frontend/src/features/shell/components/ReactManagedTerminalRouteBody.tsx` 对 Terminal event detail 只负责拉取、缓存和展开时序，具体语义分流交给共享 `RuntimeProcessDetailBlocks`：`text / markdown / message / reasoning / plan / log` 等说明类 block 直接进入安全 Markdown 富文本，`terminal / diff / code / tool_input` 与 JSON 输出继续保留 `<pre><code>`，避免破坏 shell、diff 和代码阅读形态。带 `has_detail` 且未缓存 detail 的任意事件在展开时先请求 `/api/terminal/sessions/{session_id}/turns/{turn_id}/events/{event_id}`，detail 写入后再设置 expanded；无 detail 时的 `event.summary` fallback 继续沿用同样的类型分流，而不是统一塞进 `<pre>`，也不在首帧以 summary 形态抢先打开详情体。Terminal event 的状态 badge 只由事件头 meta 渲染，detail block header 只保留标题与文件位置，不再根据 `block.status` 输出重复状态。
- `internal/interfaces/web/frontend/src/features/shell/components/RuntimeTimeline.tsx` 在可折叠 Terminal step 场景下必须稳定输出三列步骤头结构：`.terminal-step-toggle-icon`、标题节点和 meta 节点按固定顺序进入 `grid-template-columns: 12px minmax(0, 1fr) auto`。不能只依赖 CSS 假定第一列存在，否则标题会落进 12px 图标列并在移动端退化成单字符窄列。
- `ConversationWorkspace.tsx` 必须把 Conversation 时间线装配结果按可见消息数组稳定 memoize，长历史初始只传入最新 32 条消息，顶部 `topContent` 渲染 `Load earlier messages / 加载更早消息` 控件，点击或滚到顶部后每次再扩展 32 条并按滚动高度差恢复阅读位置；服务端更早历史由 `ConversationRuntimeProvider` 基于 `turns_paging` 自动后台补页，workspace 顶部滚动只展开本地已加载批次，不触发会话详情刷新。Workspace 记录上一帧完整消息 id 与可见消息 id；当后台分页只是在当前会话前方 prepend 旧消息时，当前渲染窗口继续使用上一帧可见消息，不因为全量缓存变长而刷新短会话正文区。只有用户触发加载更早消息时，`visibleCount` 才扩展到本地缓存中的下一批历史。若新消息追加前当前可见窗口已经覆盖全部已加载消息，追加后 `visibleCount` 扩展到新的消息总数，避免发送后的本轮消息把原先可见历史挤出窗口。`ChatMessageRegion.tsx` / `buildChatTimelineItems` 进一步以 `session + language + callback + message id + message signature` 缓存单条 `RuntimeTimelineItem`，并用消息对象引用缓存签名计算，使结果 patch 只重建当前变化的 assistant 消息，不重复生成稳定历史消息的 Markdown HTML 与 Process 树；`RuntimeTimeline.tsx` 继续通过 `memo` 只在 `items / topContent / emptyState / overlay` 真正变化时重渲染；`RuntimeWorkspacePage.tsx` 也要把 session pane、workspace header 与 workspace content 这些重节点 memo 成稳定 ReactElement。仅有 Composer 草稿变化时，不重复解析 Markdown、不重建 `Process` 树，也不重跑整条消息时间线。
- `internal/interfaces/web/frontend/public/legacy/chat-terminal.css` 及其镜像产物 `internal/interfaces/web/static/dist/legacy/chat-terminal.css` 继续承接 `terminal-*` 旧 DOM 契约皮肤；其中 `.terminal-log-text` 必须保持 `word-break: normal` 与 `overflow-wrap: break-word`，避免移动端 prompt bubble 在 shrink-to-fit 场景下被 `overflow-wrap: anywhere` 压成逐字断行。
- `internal/interfaces/web/frontend/src/styles/root.css` 与 `shell.css` 共同承担运行页横向边界：`html / body / #frontend-root` 禁止页面级横向滚动，`runtime-workspace-* / runtime-timeline / terminal-* / message-markdown-*` 主容器统一声明 `min-width: 0`、`max-width: 100%` 与 `box-sizing: border-box`；长路径、错误日志、inline code、markdown pre/code 与 `.chat-md-table-wrap` 只在内容块内换行或内部滚动，不能把移动端顶部操作行、消息卡片或 Composer 撑出视口。
- `internal/interfaces/web/frontend/public/legacy/chat-terminal.css` 中的 `.terminal-step-title` 与 `.terminal-step-richtext` 需要显式占满可用列宽，并声明 `min-width: 0`、`overflow-wrap: break-word` 与首尾段落 margin 修正，保证说明类步骤正文在窄屏下维持整列阅读，而不会被步骤容器缩成窄列。
- `internal/interfaces/web/frontend/public/legacy/chat-core.css` 继续承担 Conversation runtime 内容区与 `Process` 阅读层皮肤，其中 `runtime-process-step`、`runtime-process-step-head` 与 `runtime-process-step-body` 必须显式声明 `min-width: 0`、正文 `width: 100%` 与 Markdown 子节点的整列换行约束，保证真机窄屏下长中文步骤说明不会塌缩成逐字竖排窄列。
- `internal/interfaces/web/frontend/src/styles/shell.css` 中的 `.runtime-timeline` 必须以 `min-height: 100% + align-content: start + grid-auto-rows: max-content` 维持顶部收口；移动端 Chat 再通过最终覆盖层把 `runtime-workspace-screen / runtime-timeline / runtime-thinking-shell` 固定为顶部流式布局，这样在少量消息、短回复、加载中的 `Thinking` 披露行或折叠 `Process` 场景下，消息块与状态标签仍按内容高度自然堆叠，不会被满高 grid 轨道居中或拉伸。
- `internal/interfaces/web/frontend/src/features/shell/components/RuntimeRouteHost.tsx`、`RuntimeWorkspacePage.tsx` 与 `RuntimeTimeline.tsx` 负责运行页的统一装配、页面与时间线实现：`WorkbenchApp.tsx` 对 `chat / terminal` 统一挂载 `RuntimeRouteHost`，`RuntimeRouteHost` 传入真实 route 并通过 `key={route}` 隔离 controller 实例，不再在 app 层分别拼 conversation provider 路径和 terminal 路径；`RuntimeWorkspacePage` 固定产出 `RuntimeWorkspaceShell + RuntimeWorkspaceHeader + RuntimeTimeline + RuntimeComposer` 这条链路，并把会话列表注册为 `WorkbenchSessionRail` 交给 `PrimaryNav` 渲染，工作区内部 session pane 仅保留隐藏 DOM 契约；移动端真实 `RuntimeComposer` 直接作为 workspace footer 渲染在 `.runtime-workspace-body` 内，不再通过 body-level portal 或 spacer 参与布局；`ConversationWorkspace.tsx` 保留共享 route wrapper，`ReactManagedTerminalRouteBody.tsx` 仅作为兼容组件与测试对象保留；`RuntimeTimeline.tsx` 则把 attachments、markdown、prompt、process step 与 final output 收敛成同一组 block model，Chat message 与 Terminal path 共用一套时间线容器、item 装配和 block renderer。
- `ScrollJumpStrip.tsx` 统一承载 `Chat / Terminal` 的四键阅读定位逻辑；`ConversationWorkspace.tsx` 与 `ReactManagedTerminalRouteBody.tsx` 都直接装配这一个组件，只在 selector、attribute 和 DOM namespace 上按运行页注入各自目标集合，样式层则统一收敛为同一套圆形触达按钮语言与触摸反馈。按钮判定不再只看滚动阈值，而是同时检查首条内容顶边和末条内容底边是否仍在视口外：`上一条 / 回到顶部` 仅在上方确有未读内容时保留，`下一条` 只要最后一条已进入视口就隐藏，`回到底部` 则只在最后一条底边仍在视口外时显示，避免尾部只剩空白时继续误亮。
- `RuntimeWorkspaceScreen.tsx` 负责把运行页消息滚动面与 overlay 控件分层：时间线继续放在 `.runtime-workspace-screen` 内独立滚动，`ScrollJumpStrip` 与 `terminal-jump-cluster` 则挂到外层 `.runtime-workspace-panel` 作为悬浮层，避免按钮组继续参与消息流高度计算，把空白会话或短消息场景错误撑成可滚动区域。
- `internal/interfaces/web/frontend/src/features/shell` 继续维护主导航、共享 copy、React 设置页和 route surface。`components/PrimaryNav.tsx` 负责路由高亮、导航折叠、tooltip、语言切换与当前运行页会话列表渲染，品牌区只渲染文字 `Alter0`，不再输出图形 logo 节点；`PrimaryNav` 固定持有运行页会话列表的 `Sessions` 标题与 `New` 按钮文案，`WorkbenchSessionRail` 类型只允许页面提供 `route / countLabel / body / onPrimaryAction` 与动作样式，避免页面切换时接管公共 chrome。`WorkbenchApp.tsx` 在当前 route 属于运行页但尚未收到对应 rail 数据时生成稳定 fallback rail，先保持主导航会话区与 `New` 占位项可见，真实 rail 注册后再更新数量、列表 body 和动作绑定；同时按 `route` 缓存最近一次有效 rail，切回已访问运行页时先复用该 route 的最近列表，不因旧页 cleanup 或新页首帧注册窗口回退到占位 rail。`components/ReactManagedRouteBody.tsx` 只负责把 `settings` 分派到 Settings 合并页，再由页内分组切换挂载 Runtime、Skills、Memory 与 Schedules；`terminal` 不再属于 React managed route body 集合。`RuntimeWorkspaceHeader.tsx` 负责共享 Conversation runtime 的固定 workspace header，只保留会话标题、状态按钮和 `Details` 入口。`RuntimeWorkspaceShell.tsx` 在窄屏下输出运行页顶部操作行：`Chat / Terminal` 都传入 `Menu / New`，不再传入 `mobileSessionButtonLabel`。`RuntimeWorkspacePage.tsx` 把当前运行页的 session rail 数据注册到 `WorkbenchContext`；从 `Chat / Terminal` 互相切换时主导航公共 chrome 保持原位，不出现一帧空 rail。`ReactManagedTerminalRouteBody.tsx` 继续保持旧版 `terminal-*` DOM class 契约作为兼容组件和测试对象；其会话列表、详情轮询、step 展开、列表删除、Skill 选择与 Markdown 输出仍由 React state 直接维护，但不再由当前 `/terminal` Shell 入口挂载。Settings 内的 Runtime、Skills、Memory 与 Schedules 继续复用 `.settings-route-content` 作用域下的统一客户端和共享 surface 样式。
- `internal/interfaces/web/frontend/src/app/WorkbenchContext.tsx` 与 `WorkbenchApp.tsx` 统一维护移动端运行页面板状态：运行页会话列表直接挂在主导航抽屉内，`Chat / Terminal` 移动 workbar 都只暴露 `Menu` 抽屉入口；普通 `page-mode` 路由页新增的 `Menu` 入口也复用同一套状态切换与关闭路径，切路由、点遮罩或切会话时都通过同一条关闭链路收口，不再由各页面各自维护独立开关。
- `ReactManagedTerminalRouteBody.tsx` 的提交链路在进入 `submitInput` 时会立即设置 `submitting`，哪怕当前还需要先 `createSession()`；这样首次点击发送按钮就会同步切到 `Sending...` 禁用态，再串行完成会话创建、输入提交、active session 刷新和滚动收口，避免用户把首击感知成无效点击或在 session 创建窗口内重复提交。
- `features/shell/components/RuntimeWorkspacePage.tsx` 与 `RuntimeWorkspaceShell.tsx` 为运行页共享会话列表注册、移动端 workbar、layout state 与 Composer 交互状态契约；`runtimeMobileLayout.ts` 负责把 `isMobileViewport / inputFocused / sessionPaneOpen / composerPanelOpen` 归一为 `desktop / mobile-rest / mobile-keyboard / mobile-session-drawer / mobile-composer-panel`。`RuntimeWorkspaceHeader.tsx` 为 `Chat / Terminal` 输出同一组标题、状态按钮与 `Details` 按钮元素，Terminal 只传入状态值和详情面板内容，不再传入 Terminal 专属 header kind、状态按钮属性或 details toggle 外观；`RuntimeComposer.tsx` 为三条运行页输出同一套 composer shell 与 form surface，`shell.css` 只允许 Terminal 在 meta、Codex 候选和工具按钮上追加领域样式，不再通过 `[data-runtime-view="terminal"] .runtime-composer-shell` 覆盖更深背景或更低 padding，也不再为 Terminal 额外渲染 form 外层的 status note；`ReactManagedTerminalRouteBody.tsx` 将失败、退出和附件错误提示写入 `RuntimeComposer.metaContent`，与 Skill/Chat 工具栏 meta 保持同一结构。`ConversationWorkspace.tsx` 使用 `useViewportScrollAnchor` 在移动端输入聚焦且消息区原本贴近底部时保持 `.runtime-workspace-screen` 的底部阅读距离；该 hook 监听 `visualViewport`、窗口 resize 与滚动容器 resize，只维护当前滚动容器的底部锚定，不接管触摸滚动、不写页面级 scroll，也不把键盘高度写入 workspace inset。
- 移动端键盘布局不再使用 `useRuntimeComposerViewportSync`、body-level Composer portal、fixed bottom 或 `runtime-composer-spacer`。`RuntimeWorkspacePage.tsx` 始终把共享 Composer 作为 workspace footer 传给 `RuntimeWorkspaceShell`；`runtimeKeyboardIsolation.css` 在 `shell.css` 之后加载，把 App Shell 改为 `height: var(--mobile-viewport-height, 100dvh)` 的动态视口容器，并在 `760px` 及以下把 workspace body 收敛为 `mobile header / minmax(0, 1fr) panel / Composer footer` 三行。grid 中间行用 `minmax(0, 1fr)` 自动扣除 header 与真实 Composer；正文 panel 只占据第二行，不再手写 `VisualViewport height - header - Composer` 的高度公式。运行页后方层不再消费键盘高度、fixed bottom 或 transform 改写高度。
- `ConversationWorkspace.tsx` 与 `ReactManagedTerminalRouteBody.tsx` 不会把键盘偏移变量透传给其他运行区控件：移动端顶部操作行、紧凑 workspace header 和底部 Composer 都留在 workspace grid 里，键盘开合期间只允许 App Shell 的动态视口高度改变；阅读定位条在主输入框聚焦期间直接停止渲染，待输入框失焦后再恢复；Composer 配置面板在 `760px` 及以下改为独立 fixed 浮层并限制最大宽高，避免技能、模型或工具面板重新挤压正文与输入区。
- `runtimeKeyboardIsolation.css` 为移动端运行页建立固定层级：workspace footer Composer 高于消息阅读定位按钮和 Terminal 四键定位条，主导航抽屉、session drawer 与遮罩高于 Composer。`RuntimeWorkspaceShell.tsx` 在 `mobile-session-drawer` 状态下 blur 当前活动输入元素，并把 root 与 workspace body 标记为 `data-runtime-composer-interactive="false"`；`RuntimeWorkspacePage.tsx` 保持 footer Composer 可见但不可交互，避免抽屉打开时输入框覆盖抽屉或键盘残留。`ConversationWorkspace.tsx` 与 `ReactManagedTerminalRouteBody.tsx` 会在移动端输入框聚焦时直接移除 overlay，避免右下角残留半截 jump control 漏到键盘上沿。
- `RuntimeComposer.tsx` 为三条运行页共享主 `textarea` 的移动端输入契约：默认写入 `autocomplete="off" / autocorrect="off" / autocapitalize="off" / spellcheck=false / enterKeyHint="send"`，把主输入明确声明为普通命令文本输入，避免 iOS 在软键盘上沿追加钥匙串、卡片或地址类系统输入助手并露出底部残留页面层。
- `shell.css` 在 `@media (max-width: 760px)` 下对 `.composer textarea` 与 `.runtime-composer-input` 显式声明 `font-size: 16px`，把 iOS Safari 聚焦输入框时的自动页面缩放风险收敛在样式契约内，避免重新打开浏览器后首次唤起输入法造成横向裁切或分辨率突变。
- `ConversationWorkspace.tsx` 与 `ReactManagedTerminalRouteBody.tsx` 的发送按钮在移动端同时监听 `pointerdown(touch) capture` 与 `touchstart capture`：前者补齐键盘保持打开时的首触指针提交链路，后者覆盖仅暴露触摸事件的浏览器；两者共用同一手势锁做单次触摸去重，并在触发后通过 `preventDefault()` 直接进入 `sendPrompt` / `submitInput`，避免首触先被浏览器消费为 textarea blur、键盘收起或焦点切换。
- `RuntimeWorkspaceShell.tsx` 对移动端工作区头部的 `Menu / 标题 / Session / New` 按钮统一安装 `pointerdown(touch) capture` 与 `touchstart capture` 首触处理，并按按钮维度维护一次性手势锁；这些动作在输入框聚焦、软键盘打开或浏览器可能吞掉合成 `click` 的场景下仍由首个触摸直接执行，后续同一触摸链路产生的 `click` 不再重复触发。
- `RouteBodyPrimitives.tsx` 的 `CopyValueButton` 只为短值保留 `data-copy-value` 调试属性，长输出复制值不进入 DOM 属性；Terminal 最终输出的复制按钮通过组件闭包持有 payload，点击时写入剪贴板，避免长日志在 DOM 中重复存储。
- `ReactManagedTerminalRouteBody.tsx` 对 Terminal timeline item 构建使用 `useMemo`，并将 turn / step 展开处理函数稳定化；只有 `turns`、展开态、step 详情、错误或语言文案变化时才重新解析 Terminal 输出 Markdown，Composer 草稿和滚动状态变化不再触发整段输出重建。
- `public/legacy/chat-terminal.css` 中的移动端 `.terminal-composer-shell` 不再声明 `transition: bottom ...`；主运行页 Composer 位置由 CSS 动态视口和 workspace grid 驱动，避免键盘回弹动画和 CSS 补间叠加造成卡顿。
- `shared/viewport/mobileViewport.ts` 仍保留移动端视口状态推导，用于诊断、面板高度上限与 `--keyboard-offset` 输出；Chat / Terminal 主运行页布局不依赖键盘 offset 驱动 workspace header、正文 panel 或 Composer 位移，键盘后的可见高度由 `--mobile-viewport-height` 承接。
- `runtimeKeyboardIsolation.css` 让 workspace grid 的中间行成为唯一正文高度来源，`.runtime-workspace-panel` 不再按 Composer 高度二次计算；`.runtime-workspace-screen` 保持固定 20px 阅读留白，不写入键盘高度、Composer 高度或额外滚动范围。Chat 加载态的时间线顶部收口规则也放在该后加载文件中，最终覆盖 `shell.css` 的满高基础布局，确保 `Thinking`、用户消息和进行中占位不被移动端隔离层再次拉开。Terminal 的 `.terminal-jump-cluster` 在输入框聚焦且软键盘弹起时由 `ReactManagedTerminalRouteBody.tsx` 直接隐藏，键盘收起后再恢复到输入区上沿之上。
- `ReactManagedTerminalRouteBody.tsx` 的 turn 跳转状态继续复用测量缓存，滚动中默认不重复读取全部 turn 位置；仅在缓存明显失真（例如缓存底边落在当前滚动位置之上或当前视口找不到可见 turn）时才重测一次。`下一条` 的显隐除了看可见 turn 结果，还要额外受“视口是否已经贴底”、“最后一条 turn 是否已经进入当前视口”以及“当前提交是否仍处于 turn 结构未稳定的瞬时窗口”约束；一旦最后一条 turn 已经可见，即使最后一条 turn 下方仍有未读内容，也不再保留伪 `下一条`，剩余下方阅读只由 `回到底部` 承接，而新对话刚触发的 submit 窗口则会在 `setSubmitting(true)` 时立即清空 `nextTurnID`，避免旧 turn 集合和新 turn 增长之间的一拍错判。四键按钮继续使用原有箭头字形，并通过 `user-select: none`、`-webkit-touch-callout: none` 与按钮级事件约束隔离出正文文本选区，避免 iOS 长按把跳转图标卷入消息选中范围。
- Web 壳层的品牌展示统一由前端源与服务端模板共同维护：`frontend/index.html` / `static/dist/index.html` 负责 `Alter0 Chat` 页签标题，`renderLoginPage` 负责 `Alter0 Login` 与 `Alter0 Console Login`，`legacyShellCopy.ts`、`PrimaryNav.tsx` 与 `ConversationWorkspace.tsx` 负责导航品牌位、会话列表标题和运行区 copy；这些展示文案调整不影响 `alter0.*` 事件名、存储 key、cookie 或元数据字段等运行契约。
`internal/interfaces/web/frontend/src/styles/shell.css` 维护当前 React 壳层与运行页样式：桌面端使用左侧固定主导航 + 右侧主面板的两层工作台；文件末尾的 `Gemini-style flat workbench reset` 是最终视觉基线，主工作区、Settings frame、设置分区、表格、详情面板、空态、代码块和移动端抽屉默认移除外层圆角、边框和厚阴影，仅保留必要分割线、低对比选中态和 Composer 胶囊。`PrimaryNav` 在运行页下通过 `.nav-session-rail` 直接展示当前路由的会话列表，`Chat / Terminal` 的桌面主工作区统一保持单内容列，内部 `.runtime-workspace-session-pane.is-navigation-owned` 隐藏以避免重复会话栏；`1100px` 及以下时左侧主导航切为抽屉，三条运行页都通过 `Menu` 打开抽屉并展示当前运行页会话列表。`WorkbenchApp.tsx` 会把 `chat / chat / terminal` 都直接挂到共享 `workbench-pane-shell` 下，其中 `terminal` 不再额外挂载 `route-view / route-body`，因此三条运行页在切换时保持同一 runtime workspace 外壳与滚动边界。到 `760px` 及以下时，`PrimaryNav` 使用 `width: min(86vw, 336px)` 的近白全高抽屉、平面菜单、细分割线和轻量侧影，`.nav-session-rail` 去除边框、背景和固定重容器，列表按自然高度进入抽屉滚动；`RuntimeWorkspaceShell.tsx` 的移动头部不再只输出左右按钮，而是改为三段式单层 workbar：左侧 `Menu`，中间 `runtime-workspace-mobile-title` 负责渲染“状态信号 + 当前会话标题”并承担 `Details` 触发，右侧固定保留 `New`；`RuntimeWorkspaceHeader.tsx` 同步支持 `is-mobile-collapsed` 变体，在手机宽度下隐藏第二层 header 占高，但继续复用同一套详情面板状态、浮层渲染和状态信号语义。Terminal 在窄屏下继续保持 `workbench-main -> chat-pane -> terminal-view -> terminal-chat-screen` 的闭合满高链路，由 `terminal-chat-screen` 独立承担纵向滚动；运行页样式现在以 `runtime-workspace-*` 与 `runtime-composer-*` 作为共享主链路，再由兼容选择器同时覆盖既有 `terminal-*` 与 `conversation-*` DOM 钩子，避免 `ConversationWorkspace.tsx` 与 `ReactManagedTerminalRouteBody.tsx` 再各自拼装样式 class；会话列表进一步复用主导航的分组和激活态语言，`.nav-session-rail` 使用固定头部行、`contain: layout paint` 布局隔离和内部列表 `overflow-anchor: none`，`.nav-session-rail-body`、`.runtime-session-list`、分组容器和列表项都保持 `min-width: 0` 与横向裁切约束，列表在桌面与移动端固定预留 `scrollbar-gutter: stable both-edges` 并独立 `overflow-y: scroll`，避免长标题、新增会话插入或滚动条临界出现把 `Sessions / New` 区块撑宽、触发滚动锚点补偿或造成不同运行页切换时纵向位置跳变；条目本身收敛为紧凑两列结构：正文列只承载标题，`busy` 时在标题旁显示 `.runtime-session-loading`，尾列保留轻量更多按钮，避免再生成额外 footer、状态灯、元信息、短 hash 或 Skill 标签。移动端抽屉交互在真机上优先稳定而不是强调复杂 motion，当前只保留一层轻量侧滑与遮罩淡入淡出，并通过关闭 backdrop 的原生 tap highlight / focus 外观避免手机浏览器在遮罩点击关闭时出现闪烁，`max-width: 760px` 再进一步压缩按钮与间距。共享 route surface 继续作为 `Control / Sessions / Tasks / Memory / Codex Runtime` 等页面的统一视觉基线；`.scroll-jump-strip` 保持共享阅读定位条样式基线；`.terminal-jump-cluster` 负责 Terminal 对话区四键，并在 `max-width: 760px` 下改为固定停靠在工作区右侧、输入区上沿之上。`public/legacy/*.css` 仅承载兼容内容区样式、Terminal 细节和 route body 内容皮肤。
- `internal/interfaces/web/frontend/public/legacy/chat-routes.css` 继续承载 `Skill` 与部分 legacy route primitives 的类名皮肤，但视觉已对齐 shell 基线：`.runtime-route-card / .runtime-builder-form / .runtime-builder-managed-item` 统一使用白底主表面、浅灰辅助层、必要分割线和低对比选中态，避免 legacy 类名页面继续漂移到独立视觉体系。
- `static/dist/legacy/*` 当前仅承载兼容样式资源，不再包含 `/chat` 启动所需脚本。`/chat` 页面只加载 `static/dist/index.html` 中的 React bundle；兼容层通过 `app-shell[data-workbench-route]`、`data-route` 与 `data-conversation-*` 等稳定钩子让样式与页面结构继续协同工作，而不再让 legacy 脚本回写业务状态。
- 前端静态资源处理展示、输入、缓存、轮询和视口状态；会话恢复阶段允许把残留 `streaming` 消息归一为失败态或任务态，但不改写后端领域事实。

### 调用链路

```text
Web input
  -> Web handler
  -> UnifiedMessage
  -> Orchestrator / Skill execution
  -> JSON response | Task result
  -> Session history
  -> UI result patch / session detail recovery
```

### 技术约束

- Chat 默认绑定 `main` Skill，Settings 页面按目标 Skill 隔离会话历史。
- 根路径 `/`、`/chat`、`/login`、`/logout` 是稳定 Web Shell 入口；页面、受保护预览工作区与 API 共享同一登录态校验，静态只读 host 保留匿名访问。
- `/chat` 固定分发 `static/dist/index.html`，静态资源统一从 `static/dist/assets` 与 `static/dist/legacy` 提供；兼容层仅保留 legacy CSS，不再通过 legacy JS 启动 `/chat` 运行时。
- `static/dist/index.html` 仅保留前端挂载容器、字体与 legacy 样式入口；React 在 `frontend-root` 内渲染当前 Web Shell 所需的 legacy DOM 节点，并在运行时追加 source-owned shell 样式，确保既有 `id`、`data-*` 与布局结构保持稳定。
- React 壳层中的可变状态不得清空主工作区稳定实例；涉及 route body、消息区与 runtime panel/sheet host 需通过状态边界、结构化 snapshot store 与局部 DOM 更新把 React rerender 限定在安全壳层。会话列表、消息列表与 runtime panel/sheet 原生 DOM 由 React 直接消费并渲染自身状态；当前 React 托管入口集中在 `chat / terminal / settings`，Settings 内再切分 Runtime、Skills、Memory 与 Schedules。
- `ChatView` 采用“滚动内容区 + 固定底部 Composer”结构：欢迎区和消息区共享主内容栅格并各自独立滚动，Composer 独占底部行，避免空态欢迎区、消息流和输入面板相互覆盖。
- `/chat` 与 `static/dist/legacy/*` 统一返回 `Cache-Control: no-cache`，保证桥接期 HTML 与固定文件名 runtime 资源总能拿到最新版本；`static/dist/assets/*` 基于 Vite 哈希文件名返回 `Cache-Control: public, max-age=31536000, immutable`。
- 开发态可通过 `ALTER0_WEB_FRONTEND_DEV_ORIGIN` 启用 Go -> Vite dev server 反向代理：`/chat` 直接转发到前端开发服务器，`/@vite/*`、`/@react-refresh`、`/src/*`、`/node_modules/*` 等运行时资源也由同一代理提供；Vite 侧再通过 `ALTER0_WEB_BACKEND_ORIGIN` 把 `/api`、登录和健康检查路径代理回 Go。
- Chat 请求只负责回传本轮结果，前端断连不得取消已被 Web 层接受的后端执行。
- 会话标题升级、空白会话唯一性、历史折叠和页面滚动状态属于 Conversation 子域。
- Skill 管理接口继续维护用户管理 Skill 的 `session_profile_fields`、`deliverables` 与 `completion_checks` 字段集，用于历史兼容和控制面编辑；服务启动时不再注册内置业务编排，Web 对话运行页也不再由 Chat 入口、Deliverables 面板或 Session Profile 面板驱动。
- `travel` 作为内置 Skill 提供旅游攻略规则，不再对应内置业务编排。选择 `travel` Skill 后，当前 Chat / Terminal 会话仍直接由 Claude Code CLI 或 Codex CLI 执行；HTML 攻略、路线卡、图片资产和 `travel` workspace service 发布要求由 Skill 文档与 `preview-publish` 共同约束。
- `internal/execution/application.Service` 不再执行 Session Profile 抽取旁路；会话上下文只由用户消息、选中 Skill、Memory、MCP、模型配置和运行时事实共同组成。
- 读取本地缓存时先归一残留 `streaming` 消息，消息通过当前 route owner 的 session 详情恢复。
- Chat Web 前端不再使用 Web 流式网关；`ConversationRuntimeProvider.tsx` 只通过 `POST /api/chat/sessions/{session_id}/input` 获取更新后的 runtime session，并通过 Chat session 详情恢复请求断开后的最终消息。
- 请求异常收尾时，`ConversationRuntimeProvider.tsx` 会先调用 `/api/chat/sessions/{session_id}` 回补服务端已落库的最终 turn，再决定是否展示失败文案，避免把同一条 Chat 请求通过 fallback 端点重新提交一遍。只有在没有可用正文且回补失败时才渲染带刷新提示的失败文案。
- 页面初始化会对当前活动会话执行一次轻量恢复：只要本地存在历史占位、失败态助手消息，或者集合接口返回了不带完整 turns 的摘要会话，前端就会调用 `/api/chat/sessions/{session_id}`，以服务端已持久化 turn 覆盖本地失败态。
- 运行时执行器在运行期把工具动作与观察收敛为结构化 `RuntimeTraceEvent`；Terminal 应用层在 turn summary 中输出 `turns[].runtime_trace_events` 后透传给前端，事件详情通过 `session_id / turn_id / event_id` 单独读取。
- 移动端输入区以 CSS 动态视口为有效高度来源，Composer 作为 workspace grid footer 参与同一高度闭合链路。
- 移动端 root 在窄屏下不再使用 fixed 定位隔离 document 滚动，也不锁定 `html / body / #frontend-root` 的 `overflow:hidden`；前端入口 viewport meta 声明 `viewport-fit=cover, interactive-widget=resizes-content`，让支持该属性的浏览器把虚拟键盘纳入内容视口 resize。App Shell 由 `runtimeKeyboardIsolation.css` 固定为 `height: var(--mobile-viewport-height, 100dvh)`，运行页和 workspace body 继承该可见视口高度链，不再使用 `calc(var(--mobile-viewport-height) + var(--keyboard-offset))` 参与键盘布局；Composer 是 workspace grid 的第三行 footer，不消费 fixed bottom 或 transform。正文滚动窗口由 workspace grid 中间行收敛到 Composer 上沿，`.runtime-workspace-panel` 不再自己计算可见高度，`.runtime-workspace-screen` 不再用额外 bottom inset 扩大可滚范围，避免浏览器工具栏或软键盘变化导致底部留白、内容裁切、页面级上移、顶部 header 位移或重复键盘补偿。
- `shared/viewport/mobileViewportSync.ts` 作为根壳层共享 controller，除常规 `resize / visualViewport.resize / visualViewport.scroll / focusin / focusout` 外，还在 `visibilitychange(visible) / window focus / pageshow` 上强制补做一次同步；浏览器从后台回前台、标签页重新激活或 iOS WebView 恢复可见时，重建视口诊断变量，主运行页键盘布局仍以 `--mobile-viewport-height` 和 workspace grid 为准。
- 移动端 Chat 在输入框与会话设置底部面板之间切换时，打开移动端设置面板前先 blur 当前输入，主输入框重新聚焦时则先关闭设置面板，再交给浏览器原生键盘和动态视口完成贴底。
- `runtimeMobileLayout.ts` 是移动端 overlay owner 的唯一判定入口：`mobile-primary-nav-drawer / mobile-session-drawer` 会让 workspace footer Composer 保持可见但不可交互，`mobile-composer-panel` 由 Composer 自身承载，`mobile-keyboard / mobile-rest` 保持 Composer 可交互。`WorkbenchApp.tsx` 在移动端打开主导航或会话抽屉前先释放当前 editable 焦点，并通过 body 顶层 `workbench-mobile-overlay-portal` 渲染主导航与遮罩；overlay 层级高于 workspace 内 Composer，不使用 `visibility: hidden` 或卸载 DOM 修正层级。`ConversationWorkspace.tsx` 与 `ReactManagedTerminalRouteBody.tsx` 只传入 `mobileNavOpen / mobileSessionPaneOpen / inputFocused / composerPanelOpen` 等事实状态。
- 移动端不安装 Composer viewport sync hook，不在 Composer 输入框首触时记录或回放页面级、workspace 级滚动锚点，也不调用 `window.scrollTo`。用户在消息区产生新的 `touchstart / pointerdown / touchmove / pointermove / wheel / scroll` 时，由当前滚动容器直接处理。
- 移动端不安装 `touchstart / touchmove` 滚动桥接，不阻止 `.runtime-workspace-screen` 的默认触摸滚动，也不通过脚本写入 `scrollTop` 模拟滚动；移动端键盘态消息区滚动必须继续走浏览器原生 `overflow` 滚动路径，避免破坏 iOS Safari 的惯性滚动。
- `runtimeKeyboardIsolation.css` 在移动端只在 overlay owner 要求不可交互时通过 `data-runtime-composer-interactive="false"` 禁用 workspace footer Composer 命中；常规键盘态下 Composer shell 和 form 保持原生命中，textarea、工具按钮、附件预览、发送按钮和配置面板不经过额外 pointer-events 复写。这样避免输入胶囊空白区参与后方消息滚动补丁，也避免把真实输入 surface 变成半透明覆盖层。
- `RuntimeWorkspaceShell.tsx` 与 `RuntimeComposer.tsx` 共用 `runtimeKeyboardDismissal.ts`，在移动 workbar 动作、抽屉/面板入口、Composer 工具栏、附件与发送按钮的 `pointerdown / touchstart` 捕获阶段先释放当前 editable 焦点；主输入框本身和正文滚动区不安装该释放逻辑，保留浏览器原生输入与消息滚动手势。
- 桌面宽屏下 React 壳层使用 `shell.css` 中的 `--shell-reading-width=960px` 统一约束欢迎区、消息列与 Composer；legacy `chat-core.css` 继续基于主工作区可用宽度推导 `--content-width`，并让 `.message-list`、`.msg`、`.composer` 消费该宽度变量，避免消息列与输入区在不同渲染路径上出现双重宽度口径。
- React 壳层在 `shell.css` 中把 `1100px` 及以下统一视为抽屉式导航工作台：`primary-nav` 改为贴左侧视口边缘的全高抽屉，并在当前运行页直接承载会话列表，避免导航和会话列形成双浮层；`760px` 及以下继续压缩按钮和内边距，保证真手机宽度下的可触达性。
- `ConversationWorkspace` 负责运行页头部、`Details` 面板、消息区与 Composer 的排版，`ConversationRuntimeProvider` 负责 `compact` 断点感知、Terminal session 输入收口和草稿恢复；其中桌面端输入性能约束由 provider 的延迟草稿落盘与 workspace 的时间线 memoization 共同保证。Go 侧源码测试与前端组件测试共同约束这组契约。
- `chat.js` 内所有前端时间展示统一走同一北京时间格式化器，固定 `timeZone=Asia/Shanghai`、`hourCycle=h23`；时间标签输出 `HH:mm`，绝对时间输出 `YYYY-MM-DD HH:mm:ss`；控制台与账户管理视图的分钟精度时间戳输出 `YYYY-MM-DD HH:mm`，同样由共享时间格式器负责。
- Cron 创建表单默认时区直接复用同一前端常量 `Asia/Shanghai`，不再依赖浏览器本地时区探测。
- `Chat / Terminal` 会话列表前端继续按 `hashSessionIDShort(session_id)` 生成 8 位短 hash，并作为运行页 URL query、预览域名映射和排障记录引用；共享会话列表项不再展示短 hash，Terminal 不再把完整 `terminal_session_id` 填入 `shortHash` 字段。
- `src/app/routeState.ts` 负责运行页路由与会话 query 协调：路由只解析 canonical path，不再使用 hash fragment；`/chat` 与 `/terminal` 统一识别 `session_id` 多会话恢复参数，写入时通过 `sessionRouteToken` 把完整会话 id 收敛为 8 位短 hash。主导航进入 `Chat` 时即使当前已经位于 `/chat`，也会删除旧 `session_id` 并派发路由同步事件；`ConversationRuntimeProvider.tsx` 监听该同步事件，在 query 缺失时切到当前会话列表第一项。`/chat` 兼容映射到 `/chat`，`ConversationRuntimeProvider.tsx` 在 Chat 初始化时读取 Chat owner session 集合并把本地旧 `chat` snapshot 合并到 Chat 列表；`/terminal` 初始化时读取 Terminal owner session 集合。Terminal query 缺失或目标会话不存在时，运行页才回退到 Terminal 独立 `sessionStorage` 快照与服务端列表默认项。
- Markdown 渲染必须避免原始 HTML 透传；长路径、代码块和 diff 只在内容块内部滚动。`MessageMarkdown.ts` 作为 Web Shell 的共享安全 Markdown 渲染器，被 `ChatMessageRegion`、Terminal 步骤/最终输出、`RouteFieldRow` 的正文模式、Memory 文档、Control 描述、Skill/Codex 说明与 Session Profile 非等宽字段共同复用；解析核心使用 `markdown-it`，禁用 raw HTML，开启自动链接与软换行，并通过 renderer rules 保持 `assistant-inline-image`、`chat-md-pre`、`chat-md-inline-code`、`chat-md-table-wrap` 与 `chat-md-table` 等既有 DOM class。渲染器在进入 `markdown-it` 前继续清理零宽字符、修正“每字一行”的异常段落、剥离任务列表 marker，并把危险 Markdown URL 降级为可读文本；列表解析保留原始行缩进，把缩进更深的同类或异类列表、引用与代码块保留在父级 `<li>` 内，避免把层级关系拍平成连续顶层条目；机器标识类字段继续走纯文本或等宽展示，不进入 Markdown 解析。`shell.css` 在 `[data-runtime-view="conversation"]` 作用域下维护 Chat 的无框阅读流与 Markdown 视觉节奏：工作区正文白底无框，助手消息透明输出，标题、段落、列表、链接、引用和代码块通过共享 class 做弱边界排版；Markdown 表格保留真实 table DOM 与列对齐，但视觉上只使用横向分割线，不使用卡片外框、圆角边界或表头灰底；表格使用 `width: 100%` 与 `min-width: 100%` 铺满消息宽度，普通单元格声明 `overflow-wrap: anywhere` 处理长中文和长说明，表格内 `a/code` 继续 `white-space: nowrap` 保持链接、URL 和代码可复制性，只有真实不可断内容超宽时才触发表格块内部滚动，不改变 `MessageMarkdownShell` 的 DOM 与安全解析边界。
- `ChatMessageRegion.tsx` 统一负责 Conversation runtime 的消息正文与尾部元信息；已完成的 Chat 助手消息不渲染尾部元信息，运行中/排队/失败等瞬时状态才渲染紧凑状态标签，且不再附带逐条时间，避免在每条回复后重复输出 route/source/status/time 标签。
- `ChatMessageRegion.tsx` 与 `ReactManagedTerminalRouteBody.tsx` 通过共享 `RuntimeTimeline` process block 输出统一的 `runtime-thinking-shell / runtime-thinking-toggle` 思考披露入口；入口只渲染 `Thinking / 已思考` 与步骤数量，不传入 duration meta，`shell.css` 以透明、无边框、无阴影和内容宽度收缩样式覆盖旧 Process 卡片皮肤。
- `shell.css` 通过 `.runtime-workspace-head.is-sticky`、`.workspace-header-status` 和 `.workspace-header-details` 维护运行页共享的固定 workspace header 视觉状态：标题区吸顶、状态按钮按 `ready / busy / failed / interrupted / exited` 输出统一颜色反馈，但可见层只保留信号本身，并通过 `inline-flex` 信号槽直接复用会话列表 `.runtime-session-signal` 的中心点、描边与波纹规格，`Details` 入口沿同一无边框工具按钮系统渲染；`.workspace-details-layer / .workspace-details-backdrop / .workspace-details-panel` 负责把详情面板挂到顶层浮层、限制最大可视区域、提供点击外部关闭与独立滚动容器，并通过更高层级、明确背景和 `dialog` 语义保证浮层稳定可见，`.workspace-details-content / .workspace-details-summary / .workspace-details-body` 则把首屏统一为紧凑摘要栅格、窄标签字段行与压缩复制控件，Conversation 与 Terminal 只在详情内容内部保留差异化组件。`ConversationRuntimeProvider.tsx` 分离 `inspectorOpen` 与 `inspectorTabOpen`：`inspectorOpen` 只控制 Composer 配置面板，`inspectorTabOpen` 控制当前 `Model / Tools / MCP / Skills` 内容区；`toggleInspector(tab)` 在当前 tab 上再次触发时只切换内容区展开状态，不影响 workspace `Details` 浮层。
- `shared/visibility/usePageActivation.ts` 提供运行页共享的 page-activation hook：统一监听 `visibilitychange`、`focus`、`pageshow` 与 `online`，并用短时间去抖吸收同一恢复阶段的连续触发。`ConversationRuntimeProvider.tsx` 与 `ReactManagedTerminalRouteBody.tsx` 都复用这条链路；Conversation 在页面重新变为前台可见时补偿刷新 Chat 会话列表与当前活动会话详情，页面隐藏时通过 `resolveChatSessionPollPlan` 停止会话恢复轮询；Terminal 则立即刷新会话列表与当前活动会话详情。
- `shell.css` 额外维护共享 header 状态信号样式：`.runtime-session-signal` 及其 `ready / busy / failed / interrupted / exited` 变体负责 workspace header 的微型中心点、双层波纹脉冲和红黄绿状态令牌，并在 `prefers-reduced-motion` 下回退为静态信号；会话列表不再使用该状态灯，改用 `.runtime-session-loading` 表达处理中状态。
- `ScrollJumpStrip.tsx` 负责 Chat 与 Terminal 的四键阅读定位条。目标计算以滚动容器内可见消息块或 Terminal turn 为单位缓存测量结果，并在滚动、窗口 resize、DOM 变更、展开折叠和 watch key 变化时重算。`上一条` 在当前最上方可见块尚未对齐时先把该块对齐到顶部偏移；若该块已处在目标偏移位置，下一次点击直接指向它前一块，保证连续上跳不会卡在同一内容块。
- `ConversationWorkspace.tsx` 在 Chat 时间线滚动容器上只监听 `scroll`：普通滚动到顶且本地仍有隐藏消息时展开下一批本地消息，已完全展开时不再调用 `refreshActiveSession()`，也不注册额外触摸历史加载监听。服务端更早 turns 由 `ConversationRuntimeProvider` 的 `turns_paging` 后台补页 effect 负责；Workspace 通过消息 id 顺序区分“旧消息前置”和“新消息追加”，后台前置旧消息只更新缓存与隐藏历史，不扩展当前可见窗口、不触发回底，只有当前会话尾部真实追加用户消息时才按发送语义贴到底部。
- `ConversationWorkspace.tsx` 作为当前 Conversation runtime route wrapper，由 `RuntimeWorkspacePage.tsx` 产出紧凑工作区头部、时间线、`Details` 面板、Composer，并把会话列表注册到左侧 `PrimaryNav`，页面只负责注入会话数据、变体 class 和路由专属交互；`ReactManagedTerminalRouteBody.tsx` 保留相同 framework 兼容形态，但不再由 `/terminal` Shell 入口挂载。`RuntimeWorkspacePage.tsx` 统一把会话条目尾侧渲染为三点更多按钮，展开 `role="menu"` 操作层后承载置顶、详情与删除菜单项，删除菜单项在调用路由删除函数前按 `deleteConfirmLabel` 触发浏览器确认弹窗；`ConversationWorkspace.tsx` 在 `New` 草稿/占位状态不向共享列表项传入 pin/details/delete 操作，避免虚拟会话显示三点菜单。`ConversationRuntimeProvider.tsx` 通过 `isBlankDraftSession` 识别尚未落库且无消息的空白草稿，`createSession` 在存在空白草稿时只调用 `focusSession` 回到该入口，确保 `/chat` 与 `/terminal` 兼容 path 不会堆出多个空白虚拟会话。`ConversationWorkspace.tsx` 不再向移动 workbar 传入 `mobileSessionButtonLabel`，`chat` 路由也不再向 `RuntimeComposer` 注入 `Session` 工具按钮，`chat` 继续保留 Composer 内的会话设置入口。`RuntimeComposer.tsx` 固定输出单一胶囊式助手输入面板，DOM 仍由 `runtime-composer-body + runtime-composer-toolbar` 组成，textarea 透明无内边框并与工具栏共享白色 surface，工具栏左侧按路由收口为附件按钮或 `Session` 会话设置入口与附件按钮，运行态配置通过面板内部 tab 切换，右侧只保留提交动作；`shell.css` 通过共享 padding、min-height、居中宽度和移动端安全区规则保证主输入区拥有稳定可读宽度；`PrimaryNav.tsx` 维护纯文字品牌位、当前运行页会话列表与语言切换，左下角不再额外挂接账号信息区。Conversation 继续保留 `data-conversation-*` 钩子，兼容 Terminal 组件继续保留 `terminal-*` 钩子与布局变体，但当前 Shell 公共 DOM 主契约已经收敛到 `runtime-workspace-* / runtime-composer-* / runtime-timeline-*`。
- `ConversationWorkspace.tsx` 的主输入框不在移动端 `pointerdown / touchstart` 捕获阶段取消默认行为，首次触摸输入框仍走浏览器原生软键盘手势；运行页不再对首触输入框调用 `focus({ preventScroll: true })`，也不安装 page scroll 锁或 workspace 背景滚动锁。兼容 Terminal 组件保留同类输入恢复能力，但当前 `/terminal` path 使用 Conversation runtime 输入链路和 Terminal owner。
- `shell.css` 在共享 runtime 作用域下继续叠加 workbench 精修：会话列表项保留左侧激活竖线与尾侧三点更多按钮，菜单浮层相对条目右侧定位，导航栏内动作列固定为单按钮宽度，避免移动抽屉里标题被多枚操作按钮压缩；`Pinned / 置顶` 分组与时间分组共用同一低噪音 group header 与列表项布局，保证置顶组插入后不改变 rail 宽度或 `Sessions / New` chrome 位置；`Details` 面板使用带标题栏、显式关闭按钮、轻量遮罩和紧凑字段行的浅色 surface，空态阅读区与 Composer 使用同一套浅色 surface，最终扁平 reset 不再抹掉 Details 面板的必要边界与圆角；会话态消息区在 Conversation 视图下退回白底无框正文面，避免在主工作区内继续出现嵌套边框、背景分界或卡片式助手回复；首页 Composer 固定为单一胶囊式助手输入面板，textarea 去除内层边框和 resize，工具栏左侧对齐工作区工具、附件与可选 meta，右侧单独保留 icon submit；桌面端 form 使用 `width: min(100%, 860px)` 居中，移动端 form 回落为满宽并通过 `env(safe-area-inset-bottom)` 贴住可见底边。`760px` 及以下的共享 `.runtime-composer-shell` 在运行页内使用 workspace grid footer，不使用 fixed bottom 或 `transform` 创建额外合成层，避免 iOS Safari 键盘动画期间复用旧阴影层造成灰色残影。`ConversationWorkspace.tsx` 在空态为 console panel 与 chat screen 追加 `is-empty` class，`shell.css` 以内部滚动容器承接空态滚动，并通过低对比网格与细弧线背景提升空态画布层次，避免窄屏空页把头部操作行顶离可视区。
- `shell.css` 在 `@media (max-width: 1100px)` 下对工作台性能做额外收敛：关闭 `body::before/after` 光晕层，移除 `primary-nav / chat-pane / mobile-backdrop / runtime-workspace-session-pane-backdrop / runtime-workspace-session-pane-shell / runtime-workspace-body` 的 `backdrop-filter`，把移动运行页与抽屉回落为静态浅色表面，减少真机滚动和抽屉切换时的整页合成开销。

### 验证策略

- Web handler 测试覆盖会话创建、历史隔离、流式事件和取消语义。
- Web static delivery 测试覆盖主 Web Shell 与 `frontend_dist` workspace preview 的内容 hash asset version 注入，确保长期 immutable asset 缓存不会在服务重启或预览刷新后继续命中旧 bundle。
- 前端 E2E 覆盖 Chat、移动端输入、设置面板和长会话渲染。
- 前端组件测试需覆盖 React 工作台的稳定契约，至少校验 `WorkbenchApp` 的 canonical path 路由、语言切换、移动端导航收口、左侧主导航会话列表，以及 Conversation / Terminal workspace 的固定 header、消息区、Composer 和 `Details` 面板未被回归破坏；Conversation 消息区还需覆盖轻量 IM 气泡 DOM 与样式契约、长历史最新优先渲染、加载更早批次和 Chat 运行态缓存的未过期恢复、接口回源更新、过期失效。
- `legacyRouteLayoutStyles.test.ts` 需继续对 `chat-core.css` 的 `Process` 阅读契约做源码断言，至少覆盖步骤标题收缩、正文整列宽度和 `max-width: 760px` 下的移动端可读性约束。
- `ReactManagedTerminalRouteBody.test.tsx` 需覆盖三类终端步骤回归：步骤头保留独立 `.terminal-step-toggle-icon` 且展开前后状态与标题主列同时成立；命令/终端输出类 block 继续渲染 `<pre><code>`；说明类 block 在详情展开后必须落到 `.message-markdown-rendered`，并对零宽字符或“每字一行”病态内容完成可读性归一化。该组件测试还需覆盖 Terminal 运行态缓存：未过期缓存应在接口返回前恢复会话列表和最近 turns，接口响应后刷新为服务端数据；超过 TTL 的缓存不得参与首屏渲染。
- 图片输入链路的最小稳定测试面包括：前端文件选择与剪贴板图片读取限制、Composer 附件预览与移除、Web 消息接口对附件元数据的编码、`RuntimeResolverProcessor` 对图片 part 的构造与禁回退约束、OpenAI Responses / Chat Completions 适配层对视觉内容的序列化。
- `src/app/routeState.test.ts`、`src/app/WorkbenchApp.test.tsx`、`features/shell/legacyShellConfig.test.ts`、`features/shell/components/PrimaryNav.test.tsx`、`shellLayoutStyles.test.ts`、`legacyRouteLayoutStyles.test.ts` 与各 `ReactManaged*RouteBody.test.tsx` 共同覆盖路由解析、三入口主导航、Management 工具入口、Management 页族标记、语言切换、Conversation runtime 入口、Skill/Terminal/Memory/Control/Tasks/Sessions 页面取数与窄屏布局契约；Go 侧 `internal/interfaces/web/server_*_test.go` 继续通过源码与嵌入资产断言校验 `WorkbenchApp`、`ConversationRuntimeProvider`、`ConversationWorkspace`、`ReactManagedRouteBody`、共享样式和静态资源分发策略。
- 图片输入链路的最小稳定测试面包括：前端文件选择与剪贴板图片读取限制、发送 payload 与会话恢复预览资产的分离、Composer 附件预览与移除、AI markdown 图片渲染、Web 消息接口对附件元数据的编码、`RuntimeResolverProcessor` 对图片 part 的构造与禁回退约束、OpenAI Responses / Chat Completions 适配层对视觉内容的序列化。
- 回归测试优先覆盖空白会话重复、软键盘残留空白、整段列表重建、断流恢复与残留 `In Progress` 等高频问题。

## Skill & Memory

### 包边界

- `internal/runtime/application` 保留用户管理 Skill catalog 与历史兼容查询；内置业务编排 catalog 为空。
- `internal/control/domain` 与 `internal/control/application` 负责 Runtime Profile、Skill、MCP、Model Provider 与 Codex Runtime 的控制面配置。
- `internal/execution/application` 负责 Runtime Context 解析，包括 Skill、MCP、Memory、工作区事实和交付要求。
- `internal/execution/infrastructure` 负责 Claude Code 启动、Codex Direct 启动、运行日志解析、线程/会话状态持久化和错误收口。
- `internal/orchestration/application` 负责会话摘要、长期记忆、天级记忆、项目记忆和任务摘要召回。
- `docs/skills` 承载 file-backed Skill 仓库。

### 调用链路

```text
Natural language message
  -> Runtime Profile / Skill selection
  -> MemoryContext resolution
  -> Workspace runtime injection
  -> Claude Code + provider profile | Codex Direct
  -> Final response + Process/logs
  -> Session archive + memory maintenance inputs
```

### 技术约束

- 任务推理、工具调用和会话内压缩由 Claude Code 或 Codex CLI 自身完成；服务侧围绕运行时选择、上下文注入、日志解析、结果归档和记忆维护组织执行链路。
- `internal/execution/application` 在启动前为当前会话生成 Runtime Context：选中 Skill、Memory 摘要、MCP、工作区路径、仓库路径、附件路径、产物路径、可写边界和交付要求。
- Claude Code 路径在会话工作区生成 `.alter0/claude-runtime/`、`CLAUDE.md`、Skill 副本、Memory 注入摘要和 provider profile 环境。
- Codex Direct 路径在会话工作区生成 `.alter0/codex-runtime/`、`AGENTS.md`、独立 `codex-home/`、Skill 副本、Memory 注入摘要和 thread id。
- `internal/storage/infrastructure/localfile.SessionStore` 使用分文件布局持久化 Chat 历史：新 Chat 会话按 `session_id` 写入 `_default` 分组；历史 `alter0-chat` 归档日文件与旧 `chat` 分文件布局在读取时按消息身份去重合并到当前 Chat 会话模型。删除会话后保存全量快照会清理已不存在的会话文件。加载时扫描当前目录布局，并读取旧版 `.alter0/sessions.json` / `.alter0/sessions.md` 聚合文件；当多种布局同时存在时按消息身份去重合并，随后立即把合并结果重写为新的分文件布局并删除旧聚合文件，避免迁移中断造成历史缺失。
- Web 上传的会话附件经 `internal/interfaces/web/server.go` 与 `session_attachment_store.go` 规范化后统一写入 `alter0.user_input.attachments`；图片附件额外保留兼容性的 `alter0.user_input.image_attachments`。`/api/sessions/{session_id}/attachments` 现在支持“原文件 + 可选预览”模型：图片仍落原图与预览图，普通文件只落原文件并让 `preview_url` 回退到 `asset_url`。Chat / Terminal input 随后只携带 `id + asset_url + preview_url` 引用，服务端再解析出工作区内的原图路径写入元数据；前端渲染层再按场景分流，缩略位读取 `preview_url`，回显与预览弹层读取 `asset_url`。assistant 最终回复中的 markdown 外链图片则由 `internal/orchestration/application/session_output_image_assets.go` 在 SessionPersistenceService 中做结果后处理：仅对可下载的 `http(s)` 图片做抓取，写入同一 Session 附件目录，并把最终输出和结构化步骤里的图片地址改写为 `/api/sessions/{session_id}/attachments/{asset_id}/original`。Terminal 输入与 Control Task follow-up 输入都会复用同一附件目录与交付 URL。图片附件进入 Claude Code 或 Codex CLI 时由对应 runtime processor 解析元数据；显式 Codex Direct 由 `internal/execution/infrastructure/codex_cli_processor.go` 从同一图片 metadata 读取 `workspace_path` 并生成 Codex CLI `-i <path>` 参数；Terminal 侧普通文件则不进入多模态图片 part，而是在执行前写入 Terminal 工作区并通过 prompt 注入稳定路径，交给 Codex 读盘。带图请求不会在模型链失败后静默回退到 Codex CLI，避免把视觉请求错误降级为纯文本执行。
- Memory Files 注入需要携带路径、存在状态、可写性、内容摘要、召回片段和截断标记。
- Memory 上下文路径由服务配置内置解析，并统一供 `internal/execution/application.MemoryContextOptions`、Web Memory 聚合服务、会话摘要读取链路与系统维护链路使用；启动命令不再暴露记忆路径覆盖参数。
- Markdown 上下文主存结构固定为根级 `AGENTS.md`、`SOUL.md`、`memory/USER.md`、`memory/MEMORY.md`、`memory/daily/<YYYY-MM-DD>.md`、`memory/projects/<project>.md` 与 `memory/conversations/<conversation_id>/summary.md`。`AGENTS.md` 由 execution memory resolver 以 `root_instructions` selection 注入，定位为运行规则上下文；`SOUL.md` 由内置 mandatory context 路径解析为强约束上下文；其余文件承载事实型记忆。
- 持久记忆 Markdown 由 CLI Runtime 维护；服务侧会话记忆只保留在运行态和 `ConversationSummary` 中，用于恢复、召回和维护任务输入，不把每轮会话、压缩片段或任务摘要直接写入天级记忆或长期候选 Markdown。
- 用户可见 Markdown 不写入 confidence、source、status、sensitivity 等机器元数据；检索索引可作为派生文件重建。
- 用户显式记忆写入由当前 CLI runtime 完成；会话归档由服务生成 `ConversationSummary`；任务摘要保留在 Task 领域对象与存储中，不再由 `RuntimeMarkdownStore` 直接追加到 Daily/Long-term Markdown；长期整理由系统维护任务启动 CLI runtime 并加载 `memory-maintenance` Skill 完成。维护任务入口 prompt 固定要求读取当日/昨日天级记忆、对照长期记忆、只提升稳定事实/偏好/决策/流程/约束、禁止复制原始 transcript、日志、密钥和一次性任务细节，合并重复项并报告变更文件与跳过候选。该维护任务状态由 Web `maintenanceService` 记录，并通过 Settings 的 Schedules 内置任务展示和触发。
- Skill Memory Web 聚合接口只读返回 `AGENTS.md` root instructions、`SOUL.md` 强约束、长期记忆、天级记忆、项目记忆、会话摘要与说明文档；任务摘要刷新走 Task summary 子域接口。

### 验证策略

- Execution 应用测试覆盖 Runtime Resolver、Skill/MCP/Memory Context 注入、工作区文件生成和 Provider 选择错误收口。
- Infrastructure 测试覆盖 Claude Code 启动参数、Codex Direct 运行目录、thread 状态持久化、日志解析和错误收口。
- Memory 测试覆盖 Markdown 编解码、会话摘要、长期召回、系统维护整理输入和任务摘要深检索。

## Task, Terminal & Workspace

### 包边界

- `internal/task/domain` 定义任务状态、来源字段、摘要与执行元数据。
- `internal/task/application` 负责异步执行池、复杂度预判、任务生命周期和心跳续租。
- `internal/tasksummary/application` 负责任务摘要存储与重建；记忆 Markdown 写入由 Agent 维护，不由任务摘要模块直接落盘。
- `internal/terminal/domain` 定义 Terminal 会话态、turn 和 step。
- `internal/terminal/application` 负责 Terminal 会话持久化、恢复、输入续写和工作区分配。

### 调用链路

```text
High-complexity message
  -> Task acceptance
  -> Async executor
  -> Workspace
  -> Codex CLI / Skill execution
  -> Task logs + heartbeat + artifacts
  -> Session result summary
```

```text
Terminal input
  -> Terminal session store
  -> Codex CLI resume/start
  -> Turn/runtime event append
  -> Terminal view model
  -> Web polling / JSON response
```

### 技术约束

- Task 需要保存来源字段，支持从任务回会话、从会话查任务，并支持按触发类型、通道、来源消息与结果消息过滤。
- 长任务通过心跳续租运行窗口，任务日志 SSE 观测与后台心跳分离。
- Control 任务交互式续写通过追加输入创建 follow-up Task，不直接改写原任务执行记录。
- Task 不再提供独立 Settings 页面；任务记录、摘要和来源回链只作为 Chat / Memory / Terminal 运行链路的内部数据与只读上下文使用。
- Web 不直接暴露本地文件路径，产物通过引用、下载或预览接口交付。
- Task 产物列表响应需要过滤本地 URI；下载和预览由任务接口按 artifact id 读取并输出安全响应头。
- Memory 任务视图读取 Task 与 task summary 数据，支持任务摘要重建；重建结果保留在 Task 存储与视图数据中，不直接写入记忆 Markdown，也不直接执行 retry/cancel。
- 工作区按 Chat、Task、Terminal 分层隔离，删除会话或 Terminal 时同步清理对应目录。
- 直连 Codex 的 Chat / Chat 会话在自身工作区下维护 `.alter0/codex-runtime/` 与 `.alter0/codex-runtime/codex-home/`；Chat 的 Codex thread id 写入 `.alter0/codex-runtime/thread.json`，Chat 的 Codex thread id 写入 `.alter0/codex-runtime/threads/<YYYY-MM-DD>.json`；Terminal 会话在 `.alter0/workspaces/terminal/sessions/<terminal_session_id>/codex-home/` 下维护独立 `CODEX_HOME`。
- Terminal 会话态与 turn/runtime event 执行态分离，历史 `running / starting` 需要兼容归一。
- Terminal 会话详情聚合 turn 摘要；event 明细按 `session_id / turn_id / event_id` 单独读取，避免会话列表一次性加载大块执行日志。
- Terminal 应用层在 `ListTurns` / 单会话详情中直接输出 `RuntimeTraceEvent`，并随 turn summary 输出 `runtime_trace_events`；Terminal 前端装配 turn timeline 时直接消费该结构，把事件类型与来源写入稳定 DOM metadata。详情按展开时通过 `raw.ref` 或事件 id 对应的 `event_id` 懒加载。
- Terminal session store 落盘只写 `runtime_events` 与 `next_event_id`。读取历史状态文件时，store 兼容解析旧 `steps` 与 `next_step_id`，恢复为同一套 runtime event 结构，并在同次读取后通过标准 session snapshot 写回新字段；迁移保留既有事件 id，不为旧事件重新分配 `event-*`，避免历史详情引用失效。
- `internal/terminal/application/service.go` 在 `InputWithAttachments` 中把 Terminal 附件规范化为 turn 附件、在工作区 `input-attachments/` 下写入本轮输入文件，并按类型拆分消费：图片继续通过 `codex exec -i <file>` 或 `codex resume -i <file>` 进入 Codex 视觉输入，普通文件则在同轮 prompt 中附上 `input-attachments/<turn_id>/<filename>` 形式的 workspace 相对路径，要求 Codex 按需直接读盘；turn 摘要与持久化快照同步保留附件元数据，供 Terminal 输入草稿与图片历史回显复用。Terminal 输入请求携带的 `SkillContext` 会在每轮执行前渲染到工作区 `.alter0/codex-runtime/skills.md`，并通过托管 `AGENTS.md` 指令要求 Codex 只应用本轮选择的 Skill；空选择也会写入“未选择”标记，避免旧轮次 Skill 指令残留。
- `internal/execution/infrastructure/codex_cli_processor.go` 的流式执行链路默认使用 `codex exec --json -` 从标准输入喂入 prompt，并直接消费 stdout JSONL 事件；若当前执行上下文已解析出可用 Codex thread id，则改用 `codex exec resume --json <thread_id> -` 续写同一线程。Chat 的可用 thread id 来自当前 Session 文件。带图消息显式进入 Codex Direct 时，执行器从 `alter0.user_input.image_attachments` 解码已落盘的 `workspace_path`，并在新线程与 resume 线程中都通过 `-i <path>` 传给 Codex CLI。不再依赖已移除的 `--progress-cursor` 参数，避免新版本 Codex CLI 在 Terminal、Chat 共用执行器下启动失败。解析 `agent_message` 时按 `item.channel` 过滤输出：空频道继续兼容旧版 Codex CLI，`final` 进入 `StreamEventTypeOutput` 与最终 `ExecutionResult.Output`，`commentary` 在 `item.completed` 时转为结构化过程事件，其他非最终频道不进入最终 assistant 正文。
- Terminal 应用层在 Codex CLI 返回远端 compact 失败时，仅把当前 turn 标记为 failed，保留当前会话的运行线程指针、`terminal_session_id`、原工作区与日志，并让后续输入继续 resume 同一 Codex CLI thread。
- `internal/interfaces/web/frontend/src/features/shell/components/ReactManagedTerminalRouteBody.tsx` 与 `composerImageAttachments.ts` 现在共同承载 Terminal 通用附件体验：图片仍走读取/缩放/预览链路，并通过共享剪贴板图片提取逻辑支持 textarea 直接粘贴；普通文件直接读成原始 data URL，并在输入条里显示文件条目。前端选中或粘贴附件后统一先调用 `POST /api/sessions/{session_id}/attachments` 把原文件写入当前 Session 工作区，再在草稿与提交体里优先保留稳定 `id + asset_url + preview_url?` 引用，避免 Terminal 首发会话把原始 `data_url` 长期留在浏览器状态里。Terminal 会话列表为空时通过 `terminal-new-placeholder` 展示与 Chat 一致的 active `New` 占位，`startNewSession` 在该状态下关闭移动抽屉并把焦点送回 Composer 输入框，不调用创建接口；占位项不传入 pin/details/delete 操作，workspace header 的 `Details` 也保持禁用；首次发送输入或添加附件才进入 `createSession` 并创建真实服务端 Terminal session，未显式传入 title 时服务端返回的真实会话默认标题同样是 `New`。Chat 的空白草稿也按同一虚拟会话规则处理，重复创建只聚焦既有草稿，直到首次发送后才进入真实 Session history。`codexSlashCommands.ts` 维护 Chat / Terminal 共用的 Web 适用 Codex CLI 斜线命令候选、查询与补全文本规则；命令表按 Codex 帮助中的作用分组顺序排列，并用短动作标签控制候选行密度，同时排除权限、TUI 显示、键位、剪贴板、登录退出和本地 CLI 会话管理类命令。Terminal 仅在活动会话 `shell` 明确包含 `codex` 时把候选挂到 `RuntimeComposer.inputAssistContent`，普通 shell 不渲染该辅助。共享 `ScrollJumpStrip` 组件负责 Chat / Terminal 三条运行页的四键阅读定位：按当前视口中的可见消息块或 turn 集合计算 `上一条 / 下一条` 目标，并在 turn 列表、消息折叠态、窗口尺寸或 DOM 结构变化后才失效位置测量缓存；滚动过程只复用缓存并重算可视区交集。组件同时监听 `document.selectionchange`，当当前滚动容器内存在有效文本选区时，立即收起四键 overlay 并禁用命中区，待选区清空后恢复。统一 `MessageMarkdownShell` 负责 Chat / Terminal 最终输出的静态 Markdown/text DOM：正文排在复制工具栏之前，复制 payload 保留在组件闭包中，不写入长 DOM 属性；该组件不设置 `contenteditable`、`inputmode`、caret、focus 编辑态、脚本 `touchstart` 选区、`Selection API` 强制选中、假选中 class 或浮动复制层，并通过 memoized markdown HTML 保证相同正文不会因父级无关重渲染被反复写入。`MessageMarkdownSyntaxFixture.ts` 提供运行时与测试共用的 Markdown 语法覆盖样例，`ConversationWorkspace.tsx` 仅在 Chat 页面显式携带 `markdown_demo=1` 时把它注入为临时覆盖当前时间线视图的非持久化 assistant 演示消息。`shell.css` 只对 `message-markdown-*` 正文声明普通 `user-select: text` 与 `-webkit-touch-callout: default`，不再依赖 Terminal 视图级 `!important` 选择补丁；阅读定位按钮容器默认不参与 pointer 命中，仅可见按钮恢复命中，避免空白 overlay 截获拖选或长按选中。Conversation 与 Terminal 运行页分别在 `ConversationWorkspace.tsx`、`ReactManagedTerminalRouteBody.tsx` 中按活动 `session_id` 记录初始定位状态：每个已有内容会话首次渲染时间线或 Terminal turn 列表时同步把滚动容器的 `scrollTop` 设为 `scrollHeight`，并在下一帧仅当用户尚未移动滚动位置时复核一次以覆盖首次布局高度变化；Conversation 会在同一活动会话消息数增加时再次把滚动容器贴到底部，用于发送后立即展示新消息；同一 `session_id` 的后续结果 patch、轮询刷新、Process 展开和草稿输入不再强制回底。Terminal 的前后台恢复和轮询详情刷新仍优先使用 `scrollRestoreSnapshotRef` 保持原阅读锚点。Chat 新空白会话在公有 Skill 目录加载完成后默认选中全部可用公有 Skill，历史会话通过 `effectiveChatSkillIDs` 保留仍然可用的选择结果；Terminal 新会话首次加载 Skill 列表时默认选中全部可用公有 Skill，仅排除 `memory`。`internal/interfaces/web/server_terminal.go` 直接把 Terminal `attachments[]` 转成 `terminalapp.InputRequest.Attachments`，并把 `skill_ids[]` 过滤为控制面启用且非私有的 `SkillContext`。
- `MessageMarkdownSyntaxFixture.ts` 的构建预览样例覆盖 ATX/Setext 标题、删除线、自动链接、嵌套列表、任务项、列表内引用与代码块、对齐表格和 raw HTML 转义；其中表格样例覆盖短字符、长中文、长 URL/代码和混合内容场景，用于同时验证短表格铺满、普通长文本单元格内换行与不可断内容内部滚动；折叠示例把 `<details>` / `<summary>` 标签放入 `html` fenced code block，折叠内容按普通 Markdown 继续进入同一渲染器；`/chat?markdown_demo=1` 只注入该非持久化样例，不写入会话历史。
- `ReactManagedTerminalRouteBody.tsx` 作为兼容组件通过 `usePageActivation` 维护 Terminal 的前后台恢复：页面隐藏时仅更新 `pageHidden` 供 `busy` 会话轮询降频使用；页面重新可见或重新获得焦点后，会先刷新会话列表，再在存在活动会话时捕获当前滚动锚点并回源活动会话详情，避免后台期间新增输出、标题变化或状态切换在恢复前台后继续停留在旧视图。`ready` 会话不挂载周期轮询定时器，空闲同步统一依靠 page-activation 补偿刷新。Terminal 前端同时在删除成功后把 `session_id` 写入本地删除屏蔽集合，后续列表刷新与单会话详情回源都会先过滤该集合，防止服务端短暂返回旧快照时把已删除会话重新补回左侧列表；共享 `RuntimeWorkspacePage.tsx` 的菜单项会吞掉 `mouse/touch` 动作手势，避免删除当前活动会话时把同一手势透传成下一条会话的选中动作。当前 Shell 的 `/terminal` path 不进入该恢复链路。
- `ReactManagedTerminalRouteBody.tsx` 维护模块级 `terminalRuntimeCache`，专门覆盖兼容组件自身从 Terminal 旧视图切到其他路由再返回的个人单设备恢复场景。缓存保存排序后的会话列表、活动会话 id 和每个会话完整已加载 turns，写入时复制 turn/runtime event/attachment 数据，不裁剪已加载历史，读取时按 24 小时 TTL 校验；命中时直接作为 `useTerminalRuntimeController` 初始状态并跳过空白 loading 首屏，随后原有 `/api/terminal/sessions` 与单会话详情请求继续回源并通过 `mergeSessionSnapshot` 合并覆盖。当前活动会话详情若返回 `turns_paging.has_more_before`，组件会用 `turns_paging.next_before_turn_id / oldest_turn_id` 自动带 `turn_before` 和 `turn_limit=20` 后台补拉更早页，分页、输入返回、轮询和 page-activation 结果都按 turn id 合并并刷新缓存时间；后台补页不得覆盖当前输入、详情面板、选择态或主动阅读滚动，同一 `session_id + turn_before` 边界已完成补拉后不会再次触发，只有分页边界推进后才继续下一页。缓存不写入 `localStorage`，不作为跨刷新或跨设备事实来源；删除、置顶、输入返回、列表刷新和详情刷新仍通过 React state 统一回写缓存。
- `internal/interfaces/web/server_terminal.go` 的单会话详情默认返回最新 `20` 个 turns，`turn_limit` 上限为 `160`，并按约 `256KiB` 的 turns 页预算从最新 turn 向前装载；`turn_before=<turn_id>` 以目标 turn 为右开边界返回更早一页，并在 `turns_paging` 中输出 `total / byte_limit / approx_bytes / has_more_before / has_more_after / oldest_turn_id / newest_turn_id / next_before_turn_id`。集合接口继续只返回会话摘要，避免左侧会话列表拉取大体积 turns。
- `ReactManagedTerminalRouteBody.tsx` 兼容组件负责把 Terminal 领域状态映射到共享状态信号：`ready` 直接输出绿色 `ready`，`busy` 输出黄色 `busy`，`exited / interrupted` 等非活跃态统一收敛为红色失败信号，再复用 `RuntimeWorkspacePage.tsx` 的共享会话条目与 `RuntimeWorkspaceHeader` 渲染链路；当前 `/terminal` Shell path 的状态信号由 Conversation runtime 的 `terminal` owner session 提供。
- Terminal 跨设备共享同一 Web 登录态下的服务端会话历史，不再按 browser client 分桶。
- 共享 `shell.css` 在真手机宽度下负责运行页工作区头部和移动 workbar 的收缩规则：Terminal 与 Chat/Chat 使用同一标题、状态信号和 `Details` 触发元素，长标题按共享规则截断或换行，避免横向溢出。
- `WorkbenchApp` 继续安装共享 `mobileViewportSync` controller 输出动态视口诊断变量，但 Chat / Terminal 主运行页布局只依赖 `--mobile-viewport-height` 驱动后方高度闭合；移动端 `root.css` 在 `1100px` 及以下解除 `html / body / #frontend-root` 的 fixed 页面锁和根层 `overflow:hidden`，改用动态视口最小高度。`runtimeKeyboardIsolation.css` 在 `shell.css` 之后把 App Shell 改为动态视口容器，直接用 `height: var(--mobile-viewport-height, 100dvh)` 控制高度。真手机宽度下，workspace body 只保留 header footprint、正文 panel 和真实 Composer footer；正文 panel 是唯一内部滚动区且只占据 grid 中间行，不再手写高度公式。运行页后方层不让 document 或普通流布局承接 iOS 聚焦滚动，也不再通过 transform 处理 offsetTop；命令候选、配置面板和阅读定位控件保持静态位置，左侧抽屉打开时先释放输入焦点，Composer 保持可见但不可交互，并由更高层级的遮罩和抽屉自然覆盖。

### 验证策略

- Task 应用测试覆盖复杂度分流、并发上限、心跳续租、来源字段和删除清理。
- Web 测试覆盖任务列表、详情、日志流、retry/cancel、产物下载/预览、Memory 任务摘要重建和会话回链。
- Terminal 应用测试覆盖创建、恢复、续写、删除、详情读取、runtime event 明细、旧状态迁移、状态归一和工作区分配。
- E2E 测试覆盖 Terminal 移动端输入、滚动、Process 折叠和跨设备历史口径。

## Control, Operations & Governance

### 包边界

- `internal/control` 管理 Channel、Capability、Skill、MCP、Runtime Profile、Model Provider 与运行时状态。
- `internal/control` 中的 Model Provider 配置负责生成 Claude Code provider profile 所需的 base URL、API Key、model、profile 名称与健康状态。
- `internal/codex/domain` 负责 `auth.json` 快照、身份识别与额度状态模型；`internal/codex/application` 负责读取当前活动 `auth.json`、刷新 quota，通过 Codex app-server 的 `model/list`、`config/read`、`config/batchWrite` 读取真实运行时能力并更新 `model` / `model_reasoning_effort`，以及编排独立 Codex 登录会话。`LoginSessionStartRequest.AuthMethod=device_auth` 会把命令参数收敛为 `codex login --device-auth`；登录进程使用 Store 分配的独立登录目录作为 `CODEX_HOME`，stdout/stderr 通过流式 capture 写回会话日志，并从输出中宽松解析 `verification_uri / verification_uri_complete / user_code / expires_in / interval`。
- `internal/interfaces/web/frontend` 中的 `ReactManagedCodexAccountsRouteBody` 作为兼容组件名承载 `Settings > Runtime` 页面：初始加载并行请求 `/api/control/codex/runtime` 与 `/api/control/llm/providers`，单一顶部面板展示当前 Codex 身份、邮箱、计划、认证模式、profile、hourly / weekly 剩余额度、reset 时间与 LLM Provider 注册提示；model / 思考深度作为同一面板内的一行 key-value 可编辑选择项，选择变更后直接调用 `/api/control/codex/runtime` 写回当前设置，并用返回的运行时状态更新同页视图。面板内的 Device Code Login 区块调用 `/api/control/codex/accounts/login-sessions` 创建 `device_auth` 会话，再轮询 `/api/control/codex/accounts/login-sessions/{session_id}` 展示验证链接、用户码、过期时间、轮询间隔和登录输出；会话成功后刷新 Runtime 状态。Claude Code Provider 区块由 `RuntimeProviderConsole` 承载，桌面端使用 `registry + editor` 双栏，窄屏下单列；registry 展示已注册 Provider 的名称、base URL、默认 model、模型数量、模型列表与启用/默认状态，editor 负责新建与编辑。表单提交 OpenAI-compatible Provider，默认 `api_type=openai-completions`；models 输入使用全宽四行 textarea，按换行或逗号拆分、裁剪并保序去重，全部写入启用模型列表，首个 model 写入 `default_model`。点击列表中的编辑会把当前 Provider 载入表单，后续提交调用 `PUT /api/control/llm/providers/{id}`；API key 输入为空时发送空字符串，让后端按既有语义保留原密钥。注册或更新成功后重新加载 Provider 状态，清空 base URL、API key 与 models，并按最新 Provider 列表把空表单名称推进到下一个未占用的 `Claude Code N`。Provider 列表刷新只会在表单仍为空或保持 `Claude Code N` 默认命名时更新默认名称，避免覆盖用户正在输入的自定义 Provider。页面不展示 Account ID、User ID、保存名称、多账号管理动作、导入/切换操作、独立配置区、独立就绪侧栏、运行时诊断面板或由 auth/config 文件存在性推导的 Ready/Status 文案。
- `internal/interfaces/web/maintenance.go` 仅作为系统维护任务执行器挂入 Scheduler；前端通过 Schedules 内置任务查看和触发维护，不再保留独立维护 route 或直接维护接口。会话置顶通过 `POST /api/sessions/{session_id}/pin` 更新。
- `cmd/alter0` 管理启动、supervisor、重启、内置配置和运行时 metadata。
- `scripts` 承载运行账户凭据、Node/Playwright 工具链和部署初始化脚本；Node 初始化同时覆盖 `internal/interfaces/web` 与 `internal/interfaces/web/frontend`。
- `docs/deployment` 承载 Nginx 与部署权限说明。

### 调用链路

```text
Control UI / API
  -> Config / Capability service
  -> Codex account service
  -> Local storage
  -> Runtime resolver
  -> Execution / Scheduler
```

```text
Runtime restart
  -> Web confirmation
  -> supervisor
  -> optional git fast-forward or structured discard-confirmation error
  -> build candidate binary
  -> readyz probe
  -> switch or rollback
```

### 技术约束

- Control 面只能管理运行时配置，不绕过编排层直接执行业务请求。
- Schedules 控制面只管理系统内置维护任务的状态、启停和手动触发；每日自动运行时间、7 天不活跃阈值、置顶保护和 queued/running 任务保护规则固定在服务端，不作为用户配置暴露。维护任务不可执行或后续资源清理失败时写入 `failed` 状态，不使用空运行成功作为兜底。
- Skill 与 MCP 专用接口需要复用统一 Capability 数据结构；Capability 审计记录生命周期动作，供控制面按类型查询。
- `cmd/alter0/builtin_skills.go` 负责注册内置 Skill，并在启动阶段校验所有 file-backed 内置 Skill 文件存在；当前公有内置集合包含 `memory`、`preview-publish`、`frontend-design`、`doc-coauthoring`、`fullstack-developer`、`code-reviewer`、`webapp-testing`、`find-skills`、`test-driven-development`、`ui-ux-pro-max`、`code-simplifier`、`code-review`、`brainstorming` 与 `travel`，`memory-maintenance` 作为系统维护专用私有 Skill 注册。标准 skill 使用源码目录下的 `docs/skills/<skill_id>/SKILL.md`；plugin-style 的 `code-simplifier` 与 `code-review` 继续保留目录内 `.claude-plugin/plugin.json` 元数据，并分别以 `docs/skills/code-simplifier/SKILL.md`、`docs/skills/code-review/commands/code-review.md` 作为 alter0 的 file-backed 注入入口。`preview-publish` 的内置 guide 与 file-backed skill 共同要求静态用户可见产物和完整测试服务先发布到 `https://<service>-<short_hash>.alter0.cn`，避免把服务端路径、本地 URL 或工作区内部路径暴露为用户可访问链接。执行层不再合成或自动写入私有 Skill，Chat 直接按会话选择的 Skill 注入 CLI 运行时。
- Runtime Context 物化器负责在 CLI runtime 准备阶段复制 file-backed Skill：按当前服务进程工作目录向上查找可读的 skill 文件，定位 `docs/skills/<skill_id>/` 根目录，把整个目录复制到当前会话工作区。Claude Code 路径写入 `.alter0/claude-runtime/skills/<skill_id>/`，Codex Direct 路径写入 `.alter0/codex-runtime/skills/<skill_id>/`，并将注入上下文中的 `file_path` 改写为工作区内副本。
- Codex Direct 的托管 `AGENTS.md` 由 `internal/codex/infrastructure/runtimeconfig` 生成，并固定包含工作区隔离、禁止把 `/srv/...`、`.alter0/workspaces/...`、`file://`、`localhost`、`127.0.0.1` 作为用户链接返回，以及静态产物、完整服务与后端路由统一走 `preview-publish` 的交付约束。
- 服务启动时不再注册任何内置业务编排；对应业务能力通过用户选择的 Skill 组合表达。
- Models 控制面需要保持空 API Key 语义、占位值过滤、禁用态恢复和默认 Provider 收敛；Runtime 页的快速注册入口只复用创建接口，不改变更新接口的密钥保留语义。
- 旧运行参数 registry、配置更新接口和 audit 视图已移除；用户可见运行时变更集中在 Codex Runtime 配置与服务重启控制。会话内上下文压缩由 CLI runtime 自身处理，不在服务侧暴露配置项。
- Codex Runtime 服务固定解析当前活动 `CODEX_HOME`，未显式设置时回退到 `$HOME/.codex`；当前服务运行账户的 `<active_codex_home>/auth.json` 与 `config.toml` 作为运行时认证和配置来源。
- Codex 运行时状态接口同时返回活动 `auth.json`、`config.toml`、当前 profile、活动 model、思考深度、配置来源、`model/list` 返回的可选 model 能力集，以及当前 `auth.json` 解析出的身份快照与实时刷新后的 quota 信息；更新运行时设置时，后端先通过 `config/read` 解析当前生效 key path，再调用 `config/batchWrite` 更新 `model` 与 `model_reasoning_effort`，并触发 `reloadUserConfig` 让当前运行时立即生效。登录会话接口复用 Codex account service，不把 device-code 状态写入长期账号存储；只有 `codex login` 成功产出登录目录下的 `auth.json` 后，才通过既有 `AddFromRaw` 保存账号快照并允许覆盖 `runtime-device` 记录。
- CLI Runtime 运行参数通过源码内置默认值、启动参数和运行账户配置解析，仍受 Provider、Claude Code 与 Codex CLI 实际能力约束。
- Runtime 重启必须由 supervisor 托管，候选实例通过 readyz 后才切换；Runtime 页打开重启弹窗时默认选择 `sync_remote_master=true`。当同步请求无 Git 已跟踪改动时直接拉取、快进、通过统一构建入口重建前端产物和候选二进制、再切换；当存在 Git 已跟踪改动时，`cmd/alter0` 返回 `RuntimeRestartError`，错误码固定为 `runtime_restart_discard_confirmation_required`，Web API 与 supervisor client 必须透传同一 JSON `code`，前端只在收到该错误码后进入二次确认。未确认时不得清理或回滚本地工作区内容。
- 共享 Web 运行时内置通用 workspace service 注册表 `.alter0/workspace-services.json`：控制面 `PUT /api/control/workspace-services/{session_id}` 注册默认 `web` 服务，`PUT /api/control/workspace-services/{session_id}/{service_id}` 注册附加服务。`frontend_dist` 默认校验 git 工作区和 `internal/interfaces/web/static/dist` 构建产物，并在 Host 命中 `<session_short_hash>.alter0.cn` 或 `<service>-<session_short_hash>.alter0.cn` 时优先分发 `/`、`/chat`、`/terminal`、`/assets/*` 与 `/legacy/*`；`travel` 服务是唯一前端静态例外，固定命中 `https://travel-<session_short_hash>.alter0.cn`，当注册路径根目录已存在 `index.html` 时直接把该目录作为静态攻略根目录公开分发，并继续对该 host 只返回静态 HTML/资源、直接阻断 `/api/*` 与其他工作台路由。`http` 服务既可反向代理到外部 upstream，也可由共享运行时按注册的 `start_command + workdir + port + health_path` 托管本地子进程。默认 `scripts/deploy_test_service.sh <session_id>` 会为 `web` 合成一条当前分支后端启动命令并注册给共享运行时，先构建前端产物，再让 `https://<session_short_hash>.alter0.cn` 整体代理到这份托管后端，从而让前端与 `/api/*` 保持同一版本；当 `service_id=travel` 且未显式传入 `--repo-path` 时，脚本默认回退到当前 Session 工作区根目录，直接发布已生成的静态攻略页。由于线上证书只覆盖 `alter0.cn` 与 `*.alter0.cn`，附加服务必须保持单级子域名格式，不能再生成 `<service>.<short_hash>.alter0.cn` 或 `<short_hash>.travel.alter0.cn` 这类二级嵌套 host。共享运行时自己的 `supervisor -> web child` 继续继承 `web_login_password` 作为主登录边界，托管 workspace service 子进程启动前会剥离 `ALTER0_WEB_LOGIN_PASSWORD` 并注入 `ALTER0_WEB_REUSE_GATEWAY_AUTH=1`，使预览后端只复用共享网关登录态，不再叠第二层鉴权。
- Web 登录态继续由 `server.go` 的 `authMiddleware + loginHandler` 统一管理；当请求 Host 命中主域或其预览子域时，登录 cookie 会把 `Domain` 收敛到根域 `alter0.cn`，使主域工作台与短哈希预览 host 共享同一登录会话，而不是各自维护孤立 cookie。交互页登录回跳通过 `loginNextForRequest` 归一化：`/` 与 `/chat` 只回跳到 `/chat`，`/terminal` 只回跳到 `/terminal`，其他 HTML 导航仍保留安全校验后的相对 Request URI；实际运行页的会话 query 由前端收敛为 8 位短 hash，避免会话级长 id 进入登录页和稳定页面 URL。
- systemd 基线统一 `HOME=/var/lib/alter0`，确保 Codex、gh、git signing、Node/Playwright 工具链使用同一运行账户上下文。
- 提交签名问题不得通过关闭签名绕过。
- 技术文档、需求文档和 README 更新按领域同步，避免需求与方案分离。

### 验证策略

- Control 测试覆盖 Channel、Capability、Skill、MCP、Runtime Profile、Codex Runtime、Schedules 内置任务、Capability 审计和服务重启请求。
- Web 接口测试覆盖会话置顶、7 天不活跃清理、置顶跳过、queued/running 任务保护、workspace 删除、维护执行器不可用、资源清理失败和 Runtime 重启确认错误码；前端组件测试覆盖 Schedules 内置任务、Runtime 重启默认远端同步、按需二次确认、Codex device-code 登录和 Runtime 页 Claude Code Provider Console 的多 Provider 连续注册、查看与编辑。
- Provider 测试覆盖创建、更新、缺失密钥恢复、默认项收敛、Claude Code profile 生成和 OpenRouter 字段。
- Runtime supervisor 测试覆盖候选版本通过统一前端感知构建入口构建、readyz 切换、失败回滚和 metadata 展示。
- 文档治理变更至少运行 Markdown 引用与空白检查；代码变更按 TDD 运行对应包或全量测试。
- Go 单测新增或调整时，同步维护 `docs/testing/unit-test-cases.md` 与对应 Go 包路径下 `TEST_CASES.md` 的覆盖范围和边界说明。

## 变更模板

后续需求或技术方案变更使用以下字段维护：

```markdown
## <Domain>

### <Subdomain / Capability>

- 需求路径：
- 主归属领域：
- 涉及包：
- 核心对象：
- 调用链路：
- 接口契约：
- 存储与迁移：
- 错误与降级：
- 观测字段：
- 测试策略：
- 依赖与边界：
```
