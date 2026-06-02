# Technical Solution

> Last update: 2026-06-01

`alter0` 的技术方案按与需求清单一致的领域模型维护。后续新增或调整需求时，技术方案必须落到对应领域与子域，不再按时间顺序、任务编号或零散专题堆叠。

## 维护规则

- 需求归属先确定领域路径，例如 `runtime.orchestration.intent`、`agent.execution.codex-exec`、`task.workspace.runtime`。
- 技术方案使用相同领域路径补充实现信息，包括包边界、核心对象、调用链路、接口契约、存储形态、错误处理、观测字段和测试策略。
- 一个能力只允许有一个主归属领域；跨领域影响通过“依赖与边界”记录，不复制成多个重复方案。
- 影响架构边界、数据结构、接口、执行链路、存储、部署或研发治理的需求变更，必须同步更新本文件。
- 用户可见行为变化继续同步 `README.md`，稳定需求口径同步 `docs/requirements.md`，实现细节同步 `docs/requirements-details/*.md`。

## 领域方案索引

| 领域 | 主包/模块 | 技术方案重点 |
| --- | --- | --- |
| Runtime & Orchestration | `internal/interfaces`、`internal/shared`、`internal/orchestration`、`internal/execution/domain`、`internal/scheduler` | 统一消息、意图路由、执行端口、调度触发、观测与健康检查 |
| Conversation & Session Experience | `internal/interfaces/web`、`internal/session`、Web static assets | Chat/Agent 会话、SSE、历史隔离、移动端视口、消息渲染 |
| Agent Capability & Memory | `internal/agent`、`internal/execution`、`internal/llm`、`internal/orchestration` | Agent Catalog、ReAct、工具执行、Skills/MCP、Memory Context、上下文压缩 |
| Task, Terminal & Workspace | `internal/task`、`internal/tasksummary`、`internal/terminal`、`.alter0/workspaces` | 异步任务、日志流、心跳、产物交付、Terminal 会话、工作区隔离 |
| Control, Operations & Governance | `internal/control`、`internal/llm`、`internal/codex`、`cmd/alter0`、`scripts`、`docs/deployment` | 控制面配置、模型 Provider、Codex 多账号、运行时重启、部署凭据、测试与 TDD 约束 |

## Runtime & Orchestration

### 包边界

- `internal/interfaces/*` 负责外部输入适配，只生成内部统一消息，不承载业务路由。
- `internal/shared/domain` 承载 `UnifiedMessage`、`OrchestrationResult` 等跨领域消息对象。
- `internal/orchestration/domain` 承载 `Intent`、`Command` 等编排领域模型。
- `internal/orchestration/application` 承载意图识别、命令路由、自然语言执行分发。
- `internal/execution/domain` 定义执行端口和运行时上下文契约。
- `internal/scheduler` 负责 Cron 配置、触发与回注编排。

### 调用链路

```text
CLI / Web / Cron
  -> UnifiedMessage
  -> Orchestrator
  -> CommandHandler | ExecutionPort
  -> OrchestrationResult
  -> Interface response / Session persistence / Task handoff
```

### 技术约束

- 外部输入必须先归一为 `UnifiedMessage`，再进入编排层。
- 命令路由优先于复杂度评估和模型执行；显式 `alter0.execution.engine=codex` 的消息会在编排层绕过命令路由，使 Codex CLI 内置斜线输入作为直连 Codex 内容进入 `ExecutionPort`。
- Cron 触发不直接调用执行器，必须复用编排链路。
- Cron runs 接口通过 Session history 按 `trigger_type=cron` 与 `job_id` 查询触发会话，不另建独立运行记录存储。
- `ExecutionPort` 是自然语言执行能力的稳定边界，具体实现可替换。
- trace、session、message、correlation 字段贯穿日志、指标、会话与任务。

### 验证策略

- 领域对象测试覆盖消息字段归一、路由结果和错误编码。
- 编排应用测试覆盖命令优先、自然语言分发、Cron 回注。
- 接口测试覆盖 CLI/Web 输入到 `UnifiedMessage` 的转换。
- Go 单测用例说明按 `docs/testing/unit-test-cases.md` 与各 Go 包路径下的 `TEST_CASES.md` 维护，并按 Runtime、Conversation、Agent、Task、Control 领域路径归档。

## Conversation & Session Experience

### 包边界

- `internal/session/domain` 定义会话与消息数据结构。
- `internal/session/application` 负责会话持久化、历史查询和删除清理。
- `internal/interfaces/web` 负责 HTTP、SSE、Web 登录、页面路由和前端静态资源分发。
- `internal/interfaces/web/frontend` 负责 Web Shell 的 Vite + React 构建、legacy DOM shell 渲染和 `static/dist` 产物输出。
- `scripts/build_alter0_service.sh` 是服务二进制的统一构建入口：先在 `internal/interfaces/web/frontend` 执行 `npm run build`，校验 `static/dist/index.html` 中的哈希 JS/CSS 资产引用，再执行 `go build -o bin/alter0 ./cmd/alter0`。`start_alter0_service.sh`、`relaunch_service.sh` 与 `make build` 必须复用该入口，避免 Go 服务重建时嵌入过期前端产物。
- `internal/interfaces/web/frontend/src/shared/api/client.ts` 负责统一 JSON 请求封装、错误收敛与登录失效回调，避免新前端页面继续散落原生 `fetch`。
- `internal/interfaces/web/frontend/src/shared/session/sessionHash.ts` 负责运行页会话短标识生成、短 hash 判定与短 hash 到完整会话 id 的前端列表解析；`Chat / Agent Runtime / Terminal` 的运行页 URL 参数统一使用该入口把完整会话 id 派生为 8 位短 hash，左侧会话列表不展示短 hash，完整会话 id 与 Terminal `terminal_session_id` 仅保留在接口、持久化、Details 与工作区路径语义中。
- `internal/interfaces/web/frontend/src/shared/time/format.ts` 负责固定 `Asia/Shanghai` 的前端显示时区与标准时间格式，避免新旧页面时间口径漂移；管理页中需要分钟精度的额度重置、运行时间等时间戳也必须复用这里的共享格式器，而不是在页面组件里手写 UTC 文案。
- `internal/interfaces/web/frontend/src/shared/time/sessionListGroups.ts` 负责把运行页会话列表按最近时间分组为 `Today / Yesterday / Earlier`，避免 Chat、Agent Runtime 与 Terminal 各自漂移成不同的分组策略。
- `internal/interfaces/web/frontend/src/shared/viewport/mobileViewport.ts` 负责移动端断点、键盘偏移阈值与 viewport baseline 计算，避免 Chat、Terminal 与 route 页重复维护软键盘占位逻辑。
- `internal/interfaces/web/frontend/src/shared/debug/clickDiagnostics.ts` 负责显式开关控制的前端点击诊断。入口只在 `?debug_clicks=1`、`?debugClicks=true` 或 `localStorage["alter0.debug.clicks"]="on"` 命中时注册 capture 监听与 `PerformanceObserver`，输出 `[alter0:click]` 与 `[alter0:longtask]` 控制台记录，默认路径不注册全局事件监听。
- `internal/interfaces/web/frontend/index.html`、`static/dist/index.html` 与登录页模板统一以 `html[lang="en"]` 启动；`src/app/WorkbenchApp.tsx` 通过写回 `document.documentElement.lang` 统一驱动中英文壳层文案切换。`renderLoginPage` 继续直接输出服务端 HTML，但视觉与文案已对齐工作台基线：复用 `IBM Plex Sans + Sora` 字体组合、近白卡片表面与安全入口 copy。
- `internal/interfaces/web/frontend/src/app` 承载当前 Web Shell 的顶层壳层：`App.tsx` 只负责挂载 `WorkbenchApp`；`WorkbenchApp.tsx` 负责主导航、路由分派、语言切换与桌面/移动导航态，并让 Management 管理页在窄屏下通过 `data-route-mobile-head` 输出共享 `Menu` 入口；`routeState.ts` 负责 canonical path 路由解析和派发，工作台顶层路由只解析 `/chat`、`/agent-runtime`、`/terminal`、`/management`，旧管理子路径统一回退到默认 Chat。`PrimaryNav.tsx` 消费 `NAV_GROUPS` 渲染三条主工作流且不再输出 `Workspace` 可见标题，并在当前运行页注册 `runtimeSessionRail` 时直接渲染 `Sessions / New` 会话列表；会话列表在主导航内通过 `.nav-session-rail` 叠加更紧凑的列表项样式，条目主体只展示标题，`busy` 会话显示 loading，尾侧保留轻量删除动作。底部固定渲染 `Management` 工具入口，点击后进入 `/management`。`ReactManagedRouteBody.tsx` 仅把 `management` 暴露为普通管理 route body，Management 页消费 `MANAGEMENT_ROUTE_GROUPS` 在页内输出 `Agent Studio / Control / Settings` 分组切换，再以本地状态挂载对应管理分区 body，不改写浏览器 path；`WorkbenchContext.tsx` 暴露当前 `route / language / navigate`、移动端主导航状态与开关，以及运行页向主导航注册会话列表的 `setRuntimeSessionRail`。根壳层通过 `app-shell[data-workbench-route]` 输出当前路由，管理页额外通过 `data-route-family="management"` 暴露页族钩子，各子能力继续通过 `data-route` 暴露页级钩子。
- `internal/interfaces/web/frontend/src/features/conversation-runtime` 承载 `chat / agent-runtime` 的运行态：`ConversationRuntimeProvider.tsx` 负责会话创建/切换/删除、消息流、SSE 收口、任务轮询、文本草稿与图片附件草稿恢复、模型与能力项选择；空白 Chat 与 Agent Runtime 会话统一使用 `New` 占位标题，会话列表标题与新建按钮由 `RuntimeWorkspacePage` 注册给 `PrimaryNav`，直接复用 shell 里的 `Sessions / New` copy，与 Terminal 保持同一入口文案。`chat` 与 `agent-runtime` 都会在常规模型 Provider 列表外额外注入前端内置 `Codex` provider，选中后通过统一的 message metadata builder 把请求改写为 `alter0.execution.engine=codex`，不再附带普通 `alter0.llm.provider_id / alter0.llm.model` 组合；编排层据此绕过 alter0 命令路由，使 Web 对话框里的 Codex CLI 内置斜线输入原样进入 Codex CLI。`ConversationWorkspace.tsx` 会在 `selectedProviderId=alter0-codex / selectedModelId=codex` 且草稿以 `/` 开头时，在共享 `RuntimeComposer` 的输入辅助区渲染 Web 适用的 Codex CLI 斜线命令候选；候选按作用分组顺序与短动作标签展示，权限和 TUI-only 命令不进入该清单，点击只更新草稿文本并保持输入焦点，不改写消息 metadata 或执行链路。Agent Runtime 加载 `/api/agents` 后会再次过滤 `main / Alter0`，并在 Agent 列表异步加载完成后把尚未产生消息的空白 Agent 会话回填到当前默认专项 Agent，避免早期空 target 会话参与发送；Provider 还会把最近一次手动选择的 Agent id 持久化到浏览器侧，并在下一次新建空会话时优先回填该选择。Agent 记录现在同时携带结构化 `deliverables[]` 契约与 `session_profile_fields[]`，前端在 `Details > Deliverables` 里直接显示专项 Agent 的最终交付物要求，并在存在 `session_attribute_key` 绑定时用当前 Session Profile 实例属性回填 URL/路径类结果；运行时还会按当前 Agent 合成 `agent-skill-<agent_id>` 私有 Skill 选择项，标记为 `agent-private + locked`，同时只把控制面返回的公有 Skill 放入可选列表。`ConversationWorkspace.tsx` 在 Agent Runtime 欢迎区内直接输出可点选 Agent 卡片，并对内置常用 Agent 提供更短的首屏介绍文案；移动端同一选择器会从桌面网格切到单列列表，条目结构固定为左侧标识、中间名称与单行介绍、右侧当前选择状态。运行页会话列表与消息详情现通过 `/api/conversation-runtime/sessions` 专用读取接口从服务端 Session history 恢复，浏览器只继续保存未发送草稿、附件草稿和当前页局部 UI 状态；provider 会同时把当前活动会话快照与最近会话轻量列表写入 `sessionStorage`，刷新时先恢复本地最近视图，再在集合接口暂时缺席时按 `session_id` 补拉单会话详情；集合返回与本地恢复视图会按 `session_id` 合并，避免刚创建或最近活跃的服务端会话在短暂漏返回时从左侧列表瞬时消失；桌面端文本草稿的浏览器持久化改为延迟写回，输入热路径只更新内存态，避免每次按键都同步序列化整份草稿缓存；provider 进一步拆为 `workspace` 与 `composer` 两套独立 context，`ConversationWorkspace` 主体只消费 `workspace`，底部 `RuntimeComposer` 由独立子树消费 `composer`，从根上隔离草稿更新对会话列表、header、时间线和详情面的影响；provider 仍可在会话列表 view model 中派生 Agent 上下文，但共享 session card 不再渲染标题下方 Agent 标签。运行页本身输出 `data-conversation-view / data-conversation-session-pane / data-conversation-workspace / data-conversation-chat-screen / data-conversation-inspector` 等稳定锚点，不再通过 bridge 或 snapshot store 回写业务状态。
- `ConversationRuntimeProvider.tsx` 对 `chat` 路由固定归一到 `alter0-chat`：读取本地活动会话、最近快照、服务端集合、单会话详情和用户手动 focus 时都会把 Chat 会话规整成这一条长期会话，并停止写回 URL 会话参数；`agent-runtime` 保留多 Agent、多 session 列表、`session_id` 短 hash query 恢复和删除能力。
- `internal/interfaces/web/conversation_runtime_session_registry.go` 新增服务端 Conversation Runtime session registry，持久化在 `.alter0/conversation-runtime-sessions.json`：它记录 `session_id + route` 维度的最小恢复视图、最近配置和 `busy / ready / failed` 状态。`messageHandler / messageStreamHandler / agentMessageHandler / agentMessageStreamHandler` 在请求开始、完成、失败时都会更新这份 registry；`conversationRuntimeSessionCollectionHandler / conversationRuntimeSessionItemHandler` 则优先读取 registry，再与 Session history 聚合结果合并。这样即使 SSE 因浏览器刷新或客户端断链中断，只要服务端已经接受了请求，运行页列表和单会话详情仍能先返回稳定的服务端视图，而不是直接丢会话或返回 404。
- `ConversationRuntimeProvider.tsx` 的恢复判定同时识别两类未完成状态：本地或远端存在 `streaming / error / Thinking...` assistant，以及当前活动会话最后一条消息仍是 user。后者用于覆盖服务端已先持久化 user、但 assistant 结果尚未写入 Session history 的窗口期；恢复流程在要求稳定 assistant 时必须等到详情接口返回非占位 assistant、任务消息或失败态后才 upsert。
- `internal/orchestration/application/SessionPersistenceService` 将会话落库拆为请求开始与结果收口两段：`Handle / HandleStream` 进入下游执行前先追加本轮 `user` 记录，执行完成后只追加 assistant 记录及 route、错误码、`process_steps`。Session history 因此不依赖浏览器连接生命周期，也不会因为 SSE 连接提前结束而丢失已发送用户消息。
- `internal/interfaces/web/server.go` 的 `executionContextForMessage` 现在会对 `trigger_type=user + channel_type=web` 的会话消息统一使用 `context.WithoutCancel` 派生执行上下文；浏览器刷新或前端主动断开 SSE 只会结束当前 HTTP 连接，不再把 `Chat / Agent Runtime` 已接受请求连带取消。前端恢复链路继续依赖 Conversation Runtime registry 与 Session history 汇合视图补拉最终结果。
- `ConversationWorkspace.tsx` 的移动端发送手势链路继续复用共享 `RuntimeComposer` submit capture，但在 `chat / agent-runtime` 路由下会先检查当前聚焦的主 textarea，并在直达 `sendPrompt` 前主动 `blur()`；这样 `mobileViewportSync` 仍可沿既有 `focusout + VisualViewport resize` 规则逐步释放 `--keyboard-offset`，把发送动作与软键盘回弹收敛到同一条稳定链路。
- `ConversationWorkspace.tsx` 额外负责把 Conversation 会话态归一为共享 `statusTone`：当前 assistant 消息为 `streaming / queued / running / in_progress` 或仍有挂起任务时输出 `busy`，显式错误、失败、取消或 `message.error` 输出 `failed`，其余稳定态输出 `ready`；同一派生结果驱动会话列表项和 workspace header。会话列表只消费 `busy` 并渲染 loading，其他状态不渲染行内状态灯；header 可见层只保留信号本身，状态名称仅通过无障碍名称与悬浮提示暴露，避免头部长期固定显示 `Ready`。
- `ConversationRuntimeProvider.tsx` 的发送链路始终先向当前 Session 追加 `user` 消息和一条新的 assistant 占位消息，再把同一条占位消息作为本轮 SSE/Task 的唯一补丁目标；流式循环在收到 `delta` 时只累积内存中的完整输出，并按 50ms 合并窗口刷新 React 状态，避免 token 级事件持续触发 Markdown 与时间线重建；收到 `done / error` 或读流异常时立即清理待刷新计时器并同步收口最终状态。流式循环在收到 `done` 后立即封口，后续迟到的 `process / delta` 事件直接丢弃，避免已完成消息被重新打开成 `streaming` 或覆盖最终正文。
- `internal/interfaces/web/server.go` 新增 `conversationRuntimeSessionCollectionHandler / conversationRuntimeSessionItemHandler`：它们复用 `sessionapp.Service` 的 `ListSessions / ListMessages`，按 `trigger_type=user + channel_type=web` 过滤运行页可见会话，再从持久化消息 metadata 中恢复 `target_type / target_id / target_name / model_provider_id / model_id / tool_ids / skill_ids / mcp_ids` 与用户附件引用，输出直接对齐 `ConversationRuntimeProvider` 需要的会话摘要与详情结构。
- `internal/interfaces/web/frontend/src/features/shell/components/RuntimeTimelinePrimitives.tsx` 维护运行页时间线里的共享展示原语：`RuntimeAttachmentGallery` 统一渲染用户图片附件与 Terminal turn 附件缩略图，`RuntimeMarkdownShell / RuntimeMarkdownHTML` 统一渲染带复制按钮的 markdown 输出与只读富文本块。`ChatMessageRegion.tsx` 与 `ReactManagedTerminalRouteBody.tsx` 现在都直接消费这组原语，让 `runtime-attachment-*` 与 `runtime-markdown-*` 成为时间线层的稳定 DOM 契约，而不再在两个运行页里各写一份图片画廊、复制按钮和 markdown 包装壳；其中 Conversation 用户消息回显直接取原图资源，Terminal turn 继续渲染缩略图并通过共享 preview dialog 查看原图。`RuntimeTimeline.tsx` 同时为 prompt 与 markdown-shell block 提供统一 bubble wrapper，`Chat / Agent Runtime / Terminal` 的用户输入与助手输出都输出 `runtime-message / runtime-message-user / runtime-message-assistant / runtime-message-bubble` 契约；后续新增运行页只要呈现用户/助手消息，也必须接入这组 class 与 block model。用户消息按 prompt block 装配后进入右侧 `runtime-message-user-shell user-message-shell` 浅灰低对比紧凑气泡，`shell.css` 单独压低该气泡的纵向 padding 与 `.terminal-log-text` 行高；Conversation 与 Terminal 都不再向 prompt block 传入 `timeLabel`，因此正文区不渲染 `.terminal-log-time`。助手 `Process` 在正文流里默认只渲染 `Thinking / 已思考` 披露行，折叠态不再保留卡片背景、渐变和纵向时间线；展开后桌面端与移动端都使用当前消息或 Terminal turn 内的轻量内联面板，避免移动端固定底部面板遮挡输入区或脱离当前上下文。Conversation 步骤头通过序号槽与标题槽保持垂直居中，Terminal 步骤头继续保留展开图标、标题、耗时和状态的同线对齐。最终答复继续使用 terminal-style markdown shell，但 `shell.css` 将 markdown body 排在复制工具栏之前，让复制动作贴在正文下方，代码块则通过 `.assistant-message-shell .chat-md-pre` 单独落到浅灰内容块；`agent-process-answer`、`terminal-final-rendered` 等差异样式只作为变体 class 挂载在共享原语外层，不再把页面私有 wrapper 重新塞回 markdown HTML 字符串。
- `internal/interfaces/web/frontend/src/features/shell/components/RuntimeMarkdown.ts` 负责运行页 markdown 安全渲染前的输入归一化：除常规 `CRLF -> LF` 外，还需剔除零宽断行字符，并把“每字一行”的病态段落折回单段文本，避免历史/流式 `process_steps.detail` 因异常字符或错误换行写入而在真机上继续显示为逐字竖排。
- `internal/interfaces/web/frontend/src/features/shell/components/ReactManagedTerminalRouteBody.tsx` 需要对 Terminal step detail 做语义分流：`text / message / reasoning / plan / log` 等说明类 block 直接复用 `renderRuntimeMarkdownToHTML` + `RuntimeMarkdownHTML`，让历史步骤、轮询拉取详情和新触发步骤共享同一套归一化与换行约束；`terminal / diff / code` 等输出类 block 继续保留 `<pre><code>`，避免破坏 shell、diff 和代码阅读形态。无 detail 时的 `step.preview` fallback 也要沿用同样的类型分流，而不是统一塞进 `<pre>`。
- `internal/interfaces/web/frontend/src/features/shell/components/RuntimeTimeline.tsx` 在可折叠 Terminal step 场景下必须稳定输出三列步骤头结构：`.terminal-step-toggle-icon`、标题节点和 meta 节点按固定顺序进入 `grid-template-columns: 12px minmax(0, 1fr) auto`。不能只依赖 CSS 假定第一列存在，否则标题会落进 12px 图标列并在移动端退化成单字符窄列。
- `ConversationWorkspace.tsx` 必须把 Conversation 时间线装配结果按可见消息数组稳定 memoize，长历史初始只传入最新 32 条消息，顶部 `topContent` 渲染 `Load earlier messages / 加载更早消息` 控件，点击或滚到顶部后每次再扩展 32 条并按滚动高度差恢复阅读位置；`ChatMessageRegion.tsx` / `buildChatTimelineItems` 进一步以 `session + language + callback + message id + message signature` 缓存单条 `RuntimeTimelineItem`，并用消息对象引用缓存签名计算，使流式 patch 只重建当前变化的 assistant 消息，不重复生成稳定历史消息的 Markdown HTML 与 Process 树；`RuntimeTimeline.tsx` 继续通过 `memo` 只在 `items / topContent / emptyState / overlay` 真正变化时重渲染；`RuntimeWorkspacePage.tsx` 也要把 session pane、workspace header 与 workspace content 这些重节点 memo 成稳定 ReactElement。仅有 Composer 草稿变化时，不重复解析 Markdown、不重建 `Process` 树，也不重跑整条消息时间线。
- `internal/interfaces/web/frontend/public/legacy/chat-terminal.css` 及其镜像产物 `internal/interfaces/web/static/dist/legacy/chat-terminal.css` 继续承接 `terminal-*` 旧 DOM 契约皮肤；其中 `.terminal-log-text` 必须保持 `word-break: normal` 与 `overflow-wrap: break-word`，避免移动端 prompt bubble 在 shrink-to-fit 场景下被 `overflow-wrap: anywhere` 压成逐字断行。
- `internal/interfaces/web/frontend/src/styles/root.css` 与 `shell.css` 共同承担运行页横向边界：`html / body / #frontend-root` 禁止页面级横向滚动，`runtime-workspace-* / runtime-timeline / terminal-* / runtime-markdown-*` 主容器统一声明 `min-width: 0`、`max-width: 100%` 与 `box-sizing: border-box`；长路径、错误日志、inline code 与 markdown pre/code 只在内容块内换行或内部滚动，不能把移动端顶部操作行、消息卡片或 Composer 撑出视口。
- `internal/interfaces/web/frontend/public/legacy/chat-terminal.css` 中的 `.terminal-step-title` 与 `.terminal-step-richtext` 需要显式占满可用列宽，并声明 `min-width: 0`、`overflow-wrap: break-word` 与首尾段落 margin 修正，保证说明类步骤正文在窄屏下维持整列阅读，而不会被步骤容器缩成窄列。
- `internal/interfaces/web/frontend/public/legacy/chat-core.css` 继续承担 Conversation runtime 内容区与 `Process` 阅读层皮肤，其中 `agent-process-step`、`agent-process-step-head` 与 `agent-process-step-body` 必须显式声明 `min-width: 0`、正文 `width: 100%` 与 Markdown 子节点的整列换行约束，保证真机窄屏下长中文步骤说明不会塌缩成逐字竖排窄列。
- `internal/interfaces/web/frontend/src/styles/shell.css` 中的 `.runtime-timeline` 必须以 `min-height: 100% + align-content: start + grid-auto-rows: max-content` 维持顶部收口；这样在少量消息、短回复或折叠 `Process` 场景下，消息块与状态标签仍按内容高度自然堆叠，不会被满高 grid 轨道拉伸。
- `internal/interfaces/web/frontend/src/features/shell/components/RuntimeRouteHost.tsx`、`RuntimeWorkspacePage.tsx` 与 `RuntimeTimeline.tsx` 负责三条运行页的统一装配、页面与时间线实现：`WorkbenchApp.tsx` 对 `chat / agent-runtime / terminal` 统一挂载 `RuntimeRouteHost`，不再在 app 层分别拼 conversation provider 路径和 terminal 路径；`RuntimeWorkspacePage` 固定产出 `RuntimeWorkspaceShell + RuntimeWorkspaceHeader + RuntimeTimeline + RuntimeComposer` 这条链路，并把会话列表注册为 `WorkbenchSessionRail` 交给 `PrimaryNav` 渲染，工作区内部 session pane 仅保留隐藏 DOM 契约；`ConversationWorkspace.tsx` 与 `ReactManagedTerminalRouteBody.tsx` 只保留 controller hook 与 route wrapper；`RuntimeTimeline.tsx` 则把 attachments、markdown、prompt、process step 与 final output 收敛成同一组 block model，Chat message 与 Terminal turn 共用一套时间线容器、item 装配和 block renderer。
- `ScrollJumpStrip.tsx` 统一承载 `Chat / Agent Runtime / Terminal` 的四键阅读定位逻辑；`ConversationWorkspace.tsx` 与 `ReactManagedTerminalRouteBody.tsx` 都直接装配这一个组件，只在 selector、attribute 和 DOM namespace 上按运行页注入各自目标集合，样式层则统一收敛为同一套圆形触达按钮语言与触摸反馈。按钮判定不再只看滚动阈值，而是同时检查首条内容顶边和末条内容底边是否仍在视口外：`上一条 / 回到顶部` 仅在上方确有未读内容时保留，`下一条` 只要最后一条已进入视口就隐藏，`回到底部` 则只在最后一条底边仍在视口外时显示，避免尾部只剩空白时继续误亮。
- `RuntimeWorkspaceScreen.tsx` 负责把运行页消息滚动面与 overlay 控件分层：时间线继续放在 `.runtime-workspace-screen` 内独立滚动，`ScrollJumpStrip` 与 `terminal-jump-cluster` 则挂到外层 `.runtime-workspace-panel` 作为悬浮层，避免按钮组继续参与消息流高度计算，把空白会话或短消息场景错误撑成可滚动区域。
- `internal/interfaces/web/frontend/src/features/shell` 继续维护主导航、共享 copy、React 管理页和 route surface。`components/PrimaryNav.tsx` 负责路由高亮、导航折叠、tooltip、语言切换与当前运行页会话列表渲染，品牌区只渲染文字 `Alter0`，不再输出图形 logo 节点；`components/ReactManagedRouteBody.tsx` 负责把 `management` 分派到合并管理页，再由页内分组切换挂载 Profiles、Control 与 Settings 相关 React 页面。`RuntimeWorkspaceHeader.tsx` 负责共享 `Chat / Agent Runtime / Terminal` 的固定 workspace header，只保留会话标题、状态按钮和 `Details` 入口，并把差异内容交给各页传入。`RuntimeWorkspaceShell.tsx` 在窄屏下输出运行页顶部操作行：`Chat / Agent Runtime / Terminal` 都传入 `Menu / New`，不再传入 `mobileSessionButtonLabel`。`ReactManagedTerminalRouteBody.tsx` 继续保持旧版 `terminal-*` DOM class 契约作为布局皮肤基线，但会话列表、工作区容器、工作区头部与窄屏顶部操作行额外复用 `ConversationWorkspace` 的工作台语义类，确保 Terminal 与 Chat / Agent Runtime 使用同一套表面和头部节奏；Terminal 的会话列表输出 `role="list"` 语义、卡片式标题和尾侧更多按钮，运行状态保留在 workspace header 与 `Details` 摘要中，不再在列表项内额外挂独立状态徽标、元信息或短标识，详情面板则先复用共享紧凑摘要栅格，再由 Terminal 自己补充会话字段和公有 Skill 选择区；Terminal 的会话列表、详情轮询、step 展开、列表删除、Skill 选择与 Markdown 输出仍全部由 React state 直接维护；控制页、Sessions、Tasks 与 Memory 继续复用统一客户端和共享 surface 样式。
- `internal/interfaces/web/frontend/src/app/WorkbenchContext.tsx` 与 `WorkbenchApp.tsx` 统一维护移动端运行页面板状态：运行页会话列表直接挂在主导航抽屉内，`Chat / Agent Runtime / Terminal` 移动 workbar 都只暴露 `Menu` 抽屉入口；普通 `page-mode` 路由页新增的 `Menu` 入口也复用同一套状态切换与关闭路径，切路由、点遮罩或切会话时都通过同一条关闭链路收口，不再由各页面各自维护独立开关。
- `ReactManagedTerminalRouteBody.tsx` 的提交链路在进入 `submitInput` 时会立即设置 `submitting`，哪怕当前还需要先 `createSession()`；这样首次点击发送按钮就会同步切到 `Sending...` 禁用态，再串行完成会话创建、输入提交、active session 刷新和滚动收口，避免用户把首击感知成无效点击或在 session 创建窗口内重复提交。
- `features/shell/components/RuntimeWorkspaceFrame.tsx` 为运行页共享 `workspaceBodyRef`，`ConversationWorkspace.tsx` 与 `ReactManagedTerminalRouteBody.tsx` 在移动端通过 `ResizeObserver + VisualViewport resize` 持续测量 Composer 的静态 footprint 与键盘抬起后的额外位移量：`--runtime-composer-rest-inset` 仅表示输入区自身高度，`--runtime-composer-inset` 只表示相对静态文档流额外抬起的那一段；`shell.css` 与 `public/legacy/chat-terminal.css` 再用这些变量收口 `.conversation-chat-screen` / `.terminal-chat-screen` 与 jump controls 的可见高度，保证 Chat、Agent Runtime 与 Terminal 的最后一屏内容稳定停在输入区上沿，同时不会在输入框上方重复留下键盘高度对应的空白带。
- 上述 `--runtime-composer-inset` 同步不能只依赖键盘事件起点；当前实现会在 `VisualViewport resize/scroll`、Composer 自身 `ResizeObserver` 与 `transitionend` 上继续补测，并在下一帧和动画尾帧兜底重算，确保键盘收起或 Composer 回弹到底边后及时释放旧的底部占位，不在页面上遗留空白带。
- `ConversationWorkspace.tsx` 与 `ReactManagedTerminalRouteBody.tsx` 不会把 `--keyboard-offset` 透传给其他运行区控件：移动端顶部操作行与紧凑 workspace header 继续留在静态工作区流里，键盘开合期间只有 fixed Composer 自身跟随 `VisualViewport` 贴底移动；阅读定位条在主输入框聚焦期间直接停止渲染，待输入框失焦后再恢复；Composer 配置面板在 `760px` 及以下改为独立 fixed 浮层并限制最大宽高，避免技能、模型或工具面板重新挤压正文与输入区。
- `shell.css` 在 `@media (max-width: 760px)` 下为共享 `.runtime-composer-shell` 建立独立移动端叠层上下文，让 Chat / Agent Runtime / Terminal 的 Composer 始终压过消息阅读定位按钮和 Terminal 四键定位条；同时 `ConversationWorkspace.tsx` 与 `ReactManagedTerminalRouteBody.tsx` 会在移动端输入框聚焦时直接移除 overlay，避免右下角残留半截 jump control 漏到键盘上沿。
- `RuntimeComposer.tsx` 为三条运行页共享主 `textarea` 的移动端输入契约：默认写入 `autocomplete="off" / autocorrect="off" / autocapitalize="off" / spellcheck=false / enterKeyHint="send"`，把主输入明确声明为普通命令文本输入，避免 iOS 在软键盘上沿追加钥匙串、卡片或地址类系统输入助手并露出底部残留页面层。
- `shell.css` 在 `@media (max-width: 760px)` 下对 `.composer textarea` 与 `.runtime-composer-input` 显式声明 `font-size: 16px`，把 iOS Safari 聚焦输入框时的自动页面缩放风险收敛在样式契约内，避免重新打开浏览器后首次唤起输入法造成横向裁切或分辨率突变。
- `ConversationWorkspace.tsx` 与 `ReactManagedTerminalRouteBody.tsx` 的发送按钮在移动端同时监听 `pointerdown(touch) capture` 与 `touchstart capture`：前者补齐键盘保持打开时的首触指针提交链路，后者覆盖仅暴露触摸事件的浏览器；两者共用同一手势锁做单次触摸去重，并在触发后通过 `preventDefault()` 直接进入 `sendPrompt` / `submitInput`，避免首触先被浏览器消费为 textarea blur、键盘收起或焦点切换。
- `RuntimeWorkspaceShell.tsx` 对移动端工作区头部的 `Menu / 标题 / Session / New` 按钮统一安装 `pointerdown(touch) capture` 与 `touchstart capture` 首触处理，并按按钮维度维护一次性手势锁；这些动作在输入框聚焦、软键盘打开或浏览器可能吞掉合成 `click` 的场景下仍由首个触摸直接执行，后续同一触摸链路产生的 `click` 不再重复触发。
- `RouteBodyPrimitives.tsx` 的 `CopyValueButton` 只为短值保留 `data-copy-value` 调试属性，长输出复制值不进入 DOM 属性；Terminal 最终输出的复制按钮通过组件闭包持有 payload，点击时写入剪贴板，避免长日志在 DOM 中重复存储。
- `ReactManagedTerminalRouteBody.tsx` 对 Terminal timeline item 构建使用 `useMemo`，并将 turn / step 展开处理函数稳定化；只有 `turns`、展开态、step 详情、错误或语言文案变化时才重新解析 Terminal 输出 Markdown，Composer 草稿和滚动状态变化不再触发整段输出重建。
- `public/legacy/chat-terminal.css` 中的移动端 `.terminal-composer-shell` 不再声明 `transition: bottom ...`；Composer 位置完全由 `VisualViewport` 同步后的 `--keyboard-offset` 驱动，避免键盘回弹动画和 CSS 补间叠加造成卡顿。
- `shared/viewport/mobileViewport.ts` 在输入框 blur 但 `VisualViewport` 尚未恢复时，会把当前状态视为“键盘正在关闭”而不是“键盘已关闭”：沿用上一轮 `baselineHeight` 继续计算 `--keyboard-offset`，直到可视视口回到基线高度后才归零，避免 focusout 提前把 composer 和正文区闪回到底边。
- Terminal 的 `.terminal-jump-cluster` 在 `max-width: 760px` 下不再直接吃“当前遮挡量”，而是消费独立的 `--runtime-composer-rest-inset` 作为静态 Composer footprint；输入框聚焦且软键盘弹起时，`ReactManagedTerminalRouteBody.tsx` 直接隐藏该 overlay，键盘收起和 Composer 回弹完成后再恢复到输入区上沿之上。
- `ReactManagedTerminalRouteBody.tsx` 的 turn 跳转状态继续复用测量缓存，滚动中默认不重复读取全部 turn 位置；仅在缓存明显失真（例如缓存底边落在当前滚动位置之上或当前视口找不到可见 turn）时才重测一次。`下一条` 的显隐除了看可见 turn 结果，还要额外受“视口是否已经贴底”、“最后一条 turn 是否已经进入当前视口”以及“当前提交是否仍处于 turn 结构未稳定的瞬时窗口”约束；一旦最后一条 turn 已经可见，即使最后一条 turn 下方仍有未读内容，也不再保留伪 `下一条`，剩余下方阅读只由 `回到底部` 承接，而新对话刚触发的 submit 窗口则会在 `setSubmitting(true)` 时立即清空 `nextTurnID`，避免旧 turn 集合和新 turn 增长之间的一拍错判。四键按钮继续使用原有箭头字形，并通过 `user-select: none`、`-webkit-touch-callout: none` 与按钮级事件约束隔离出正文文本选区，避免 iOS 长按把跳转图标卷入消息选中范围。
- Web 壳层的品牌展示统一由前端源与服务端模板共同维护：`frontend/index.html` / `static/dist/index.html` 负责 `Alter0 Chat` 页签标题，`renderLoginPage` 负责 `Alter0 Login` 与 `Alter0 Console Login`，`legacyShellCopy.ts`、`PrimaryNav.tsx` 与 `ConversationWorkspace.tsx` 负责导航品牌位、会话列表标题和运行区 copy；这些展示文案调整不影响 `alter0.*` 事件名、存储 key、cookie 或元数据字段等运行契约。
`internal/interfaces/web/frontend/src/styles/shell.css` 维护当前 React 壳层与运行页样式：桌面端使用左侧固定主导航 + 右侧主面板的两层工作台；全局 shell 半径令牌收敛为 `14px / 12px / 10px`，主 shell 样式不再使用 `999px` 胶囊半径，PC 端上传、发送与详情类控件继续落到 8-14px 的低圆角矩形，阅读定位按钮作为独立触达控件允许保持圆形。`PrimaryNav` 在运行页下通过 `.nav-session-rail` 直接展示当前路由的会话列表，`Chat / Agent Runtime / Terminal` 的桌面主工作区统一保持单内容列，内部 `.runtime-workspace-session-pane.is-navigation-owned` 隐藏以避免重复会话栏；`1100px` 及以下时左侧主导航切为抽屉，三条运行页都通过 `Menu` 打开抽屉并展示当前运行页会话列表。`WorkbenchApp.tsx` 会把 `chat / agent-runtime / terminal` 都直接挂到共享 `workbench-pane-shell` 下，其中 `terminal` 不再额外挂载 `route-view / route-body`，因此三条运行页在切换时保持同一 runtime workspace 外壳与滚动边界。到 `760px` 及以下时，`PrimaryNav` 使用 `width: min(86vw, 336px)` 的近白全高抽屉、平面菜单、细分割线和轻量侧影，`.nav-session-rail` 去除边框、背景和固定重容器，列表按自然高度进入抽屉滚动；`RuntimeWorkspaceShell.tsx` 的移动头部不再只输出左右按钮，而是改为三段式单层 workbar：左侧 `Menu`，中间 `runtime-workspace-mobile-title` 负责渲染“状态信号 + 当前会话标题”并承担 `Details` 触发，右侧固定保留 `New`；`RuntimeWorkspaceHeader.tsx` 同步支持 `is-mobile-collapsed` 变体，在手机宽度下隐藏第二层 header 占高，但继续复用同一套详情面板状态、浮层渲染和状态信号语义。Terminal 在窄屏下继续保持 `workbench-main -> chat-pane -> terminal-view -> terminal-chat-screen` 的闭合满高链路，由 `terminal-chat-screen` 独立承担纵向滚动；运行页样式现在以 `runtime-workspace-*` 与 `runtime-composer-*` 作为共享主链路，再由兼容选择器同时覆盖既有 `terminal-*` 与 `conversation-*` DOM 钩子，避免 `ConversationWorkspace.tsx` 与 `ReactManagedTerminalRouteBody.tsx` 再各自拼装样式 class；会话列表进一步复用主导航的分组和激活态语言，条目本身收敛为紧凑两列结构：正文列只承载标题，`busy` 时在标题旁显示 `.runtime-session-loading`，尾列保留 28-30px 方形更多按钮，避免再生成额外 footer、状态灯、元信息、短 hash 或 Agent 标签。Agent Runtime 欢迎区的 Agent 选择器在桌面保留卡片式网格，在移动端切为单列列表卡片：条目左侧使用统一方形标识，中间用单行短介绍承载能力摘要，右侧用轻量圆形状态指示当前选中项；欢迎区说明文字宽度也同步放宽，避免上方小字在窄容器里提前换行。移动端抽屉交互在真机上优先稳定而不是强调复杂 motion，当前只保留一层轻量侧滑与遮罩淡入淡出，并通过关闭 backdrop 的原生 tap highlight / focus 外观避免手机浏览器在遮罩点击关闭时出现闪烁，`max-width: 760px` 再进一步压缩按钮与间距。共享 route surface 继续作为 `Control / Sessions / Tasks / Memory / Codex Accounts` 等页面的统一视觉基线；`.scroll-jump-strip` 保持共享阅读定位条样式基线；`.terminal-jump-cluster` 负责 Terminal 对话区四键，并在 `max-width: 760px` 下改为固定停靠在工作区右侧、输入区上沿之上，四个按钮统一保持圆形。`public/legacy/*.css` 仅承载兼容内容区样式、Terminal 细节和 route body 内容皮肤。
- `internal/interfaces/web/frontend/public/legacy/chat-routes.css` 继续承载 `Agent` 与部分 legacy route primitives 的类名皮肤，但视觉已对齐 shell 基线：`.agent-route-card / .agent-builder-form / .agent-builder-managed-item` 统一使用近白主表面、浅灰辅助层、低对比边框和浅蓝选中态，避免 legacy 类名页面继续漂移到独立视觉体系。
- `static/dist/legacy/*` 当前仅承载兼容样式资源，不再包含 `/chat` 启动所需脚本。`/chat` 页面只加载 `static/dist/index.html` 中的 React bundle；兼容层通过 `app-shell[data-workbench-route]`、`data-route` 与 `data-conversation-*` 等稳定钩子让样式与页面结构继续协同工作，而不再让 legacy 脚本回写业务状态。
- 前端静态资源处理展示、输入、缓存、轮询和视口状态；会话恢复阶段允许把残留 `streaming` 消息归一为失败态或任务态，但不改写后端领域事实。

### 调用链路

```text
Web input
  -> Web handler
  -> UnifiedMessage
  -> Orchestrator / Agent execution
  -> StreamEvent | JSON response
  -> Session history
  -> Incremental UI patch
```

### 技术约束

- Chat 默认绑定 `main` Agent，Agent 页面按目标 Agent 隔离会话历史。
- 根路径 `/`、`/chat`、`/login`、`/logout` 是稳定 Web Shell 入口；页面、受保护预览工作区与 API 共享同一登录态校验，静态只读 host 保留匿名访问。
- `/chat` 固定分发 `static/dist/index.html`，静态资源统一从 `static/dist/assets` 与 `static/dist/legacy` 提供；兼容层仅保留 legacy CSS，不再通过 legacy JS 启动 `/chat` 运行时。
- `static/dist/index.html` 仅保留前端挂载容器、字体与 legacy 样式入口；React 在 `frontend-root` 内渲染当前 Web Shell 所需的 legacy DOM 节点，并在运行时追加 source-owned shell 样式，确保既有 `id`、`data-*` 与布局结构保持稳定。
- React 壳层中的可变状态不得清空主工作区稳定实例；涉及 route body、消息区与 runtime panel/sheet host 需通过状态边界、结构化 snapshot store 与局部 DOM 更新把 React rerender 限定在安全壳层。会话列表、消息列表与 runtime panel/sheet 原生 DOM 由 React 直接消费并渲染自身状态；`agent / terminal / memory / channels / skills / mcp / models / environments / cron-jobs / sessions / tasks` 十一类 route body 已由 React 托管，其中 terminal 也由 React 原生实现。
- `ChatView` 采用“滚动内容区 + 固定底部 Composer”结构：欢迎区和消息区共享主内容栅格并各自独立滚动，Composer 独占底部行，避免空态欢迎区、消息流和输入面板相互覆盖。
- `/chat` 与 `static/dist/legacy/*` 统一返回 `Cache-Control: no-cache`，保证桥接期 HTML 与固定文件名 runtime 资源总能拿到最新版本；`static/dist/assets/*` 基于 Vite 哈希文件名返回 `Cache-Control: public, max-age=31536000, immutable`。
- 开发态可通过 `ALTER0_WEB_FRONTEND_DEV_ORIGIN` 启用 Go -> Vite dev server 反向代理：`/chat` 直接转发到前端开发服务器，`/@vite/*`、`/@react-refresh`、`/src/*`、`/node_modules/*` 等运行时资源也由同一代理提供；Vite 侧再通过 `ALTER0_WEB_BACKEND_ORIGIN` 把 `/api`、登录和健康检查路径代理回 Go。
- SSE 连接只负责回传，前端断连不得取消已被 Web 层接受的 `Chat / Agent Runtime` 后端任务。
- 会话标题升级、空白会话唯一性、历史折叠和页面滚动状态属于 Conversation 子域。
- `GET /api/agents` 返回 Agent Runtime 可直接进入的专项 Agent；当前内置入口包括 `coding`、`writing` 与 `travel`，由 `agentCatalog.ListEntrypointAgents()` 统一输出并排除 `main / Alter0` 主助手。Agent 响应同时携带 `session_profile_fields`、`deliverables` 与 `completion_checks`：前端使用前两者构建 Session Profile 与 Deliverables 详情骨架，运行时使用 `completion_checks` 执行机器校验与失败修复；控制面 Agent 读写接口也沿用同一字段集，避免用户管理 Agent 在更新时丢失既有产物契约与校验规则。
- `travel` 的 display name 为 `Travel Agent`；内置配置同时启用 `deploy_test_service`，用于把额外生成的 HTML 旅游攻略发布到当前 Session 的公开只读子域名 `https://travel-<session_short_hash>.alter0.cn`。`travel` 不再依赖执行器内部的硬编码完成判定，而是通过 Agent 通用 `completion_checks` 声明两个必过检查：当前请求已经在 `travel` 会话工作区根目录生成或更新了对应的 `index.html`，且当前 Session 已发布公开只读 `travel` 服务并存在公开 URL。运行时不再自动生成、补写或代发 fallback HTML，但会在首轮收口缺页或缺发布时额外拉起一轮仅面向当前 Session 工作区的 Codex 修复任务；修复轮本身仍需产出真实页面与真实服务注册，不能伪造成功状态。生成规则层面要求该 HTML 采用 mobile-first 页面实现：默认结构先服务手机单列阅读、触控目标、紧凑章节节奏和横向溢出控制，desktop 作为 secondary viewport 通过更宽版心、支撑信息分组和层级增强渐进扩展；页面内容先输出分类推荐池、再输出最终 itinerary，并在 all route-related sections whenever feasible 渲染路线化结构。路线图或路线卡由真实 HTML/CSS 或 inline SVG 表达，不使用空占位图；总体路线、每日路线、交通指南、步行段、换乘段、轮渡/船行段和地图提示都应尽量包含 numbered stops、connected path、travel segments、预计步行/公交/轮渡耗时和地标提示，以便用户在手机上快速理解游览顺序。推荐池至少覆盖吃喝、景点、住宿三类，其中吃喝拆分小吃/早点/特色菜/特色饮品并兼顾老字号与大众点评高分项，景点按公园/博物馆/表演等类型分组，住宿按价格带或档位列出热门酒店，同时为各组推荐显式保留数据来源标签。核心三类只是默认骨架，运行时 prompt 与 Agent 私有 Skill 还要引导模型根据城市特征扩展夜游、集市、游船、温泉、滑雪、庙会等 city-specific categories，避免模板化遗漏当地强信号内容。`travel` Agent 的 12 轮工具预算由 Agent 配置自身提供，而不是执行器里的专项默认分支。HTML 交付的完成条件不只依赖 prompt：运行时会拒绝任何未通过 `complete` 收口、或未通过所声明 `completion_checks` 的完成结果，`guide_html_url` 也只会在真实发布存在时对外暴露。
- `internal/execution/application.Service` 在 `MemoryContext` 注入前先执行 `SessionProfileExtractor` 旁路：依据 `agent_id` 解析 Agent schema、读取已有 profile 属性、抽取本轮自然语言的结构化 patch，再把结果转成 `alter0.agent.instance_attr.*` metadata 交给统一 profile 渲染链路。
- `chat.js` 读取本地缓存时先归一残留 `streaming` 消息；无 `task_id` 的消息转失败态，带真实 `task_id` 的消息恢复到任务轮询链路。
- Web 流式网关把 `shareddomain.StreamEvent` 映射为 SSE：输出事件写成 `delta`，结构化步骤事件写成 `process`，最终 `done` 继续携带完整 `process_steps` 用于收口与持久化恢复。
- 流式连接异常收尾时优先保留已收到正文；若该轮 SSE 已收到 `start` 但未收到 `done`，`ConversationRuntimeProvider.tsx` 会把浏览器读流错误视为“可恢复中断”，先调用 `/api/conversation-runtime/sessions/{session_id}?route=...` 回补服务端已落库的最终消息，再决定是否展示失败文案，避免把同一条 Agent 请求通过 fallback 端点重新提交一遍。只有在没有可用正文且回补失败时才渲染带刷新提示的失败文案。
- 页面初始化会对当前活动会话执行一次轻量恢复：只要本地存在流式失败态助手消息，或者集合接口返回了不带完整消息的摘要会话，前端就会调用 `/api/conversation-runtime/sessions/{session_id}?route=...`，以服务端已持久化消息覆盖本地失败态。
- Agent 执行器在运行期把工具动作与观察收敛为结构化 `process_steps`，经 `ExecutionResult -> OrchestrationResult -> session.RouteResult` 透传到 SSE `done`、Task 结果和 `/api/sessions/{session_id}/messages`。
- Agent 执行器在工具循环期间直接发出 `process` 事件，前端按 `process_step.id` 原地更新步骤状态；历史旧消息仍保留基于文本标记的回退解析。
- `chat.js` 渲染 Agent 消息时优先读取 `process_steps`；仅在缺失结构化步骤时才回退到历史 `[agent] action / observation` 文本解析，以兼容旧会话。
- 移动端输入区以 `VisualViewport` 为有效视口来源，并按聚焦、键盘和页面可见性降频刷新。
- 移动端 App Shell 高度使用 `--mobile-viewport-height`，由 `chat.js` 同步 `VisualViewport` 计算值，避免浏览器工具栏变化导致底部留白或内容裁切。
- `shared/viewport/mobileViewportSync.ts` 作为根壳层共享 controller，除常规 `resize / visualViewport.resize / visualViewport.scroll / focusin / focusout` 外，还在 `visibilitychange(visible) / window focus / pageshow` 上强制补做一次同步；浏览器从后台回前台、标签页重新激活或 iOS WebView 恢复可见时，先清掉旧的 `--mobile-viewport-height / --keyboard-offset` 推导结果，再按当前真实视口重建底部占位。
- 移动端 Chat/Agent 在输入框与会话设置底部面板之间切换时，`chat.js` 需先归一底部交互层：打开移动端设置面板前先 blur 当前输入并重新同步 `--keyboard-offset`，主输入框重新聚焦时则先关闭设置面板，再执行键盘贴底和输入框对齐逻辑。
- 桌面宽屏下 React 壳层使用 `shell.css` 中的 `--shell-reading-width=960px` 统一约束欢迎区、消息列与 Composer；legacy `chat-core.css` 继续基于主工作区可用宽度推导 `--content-width`，并让 `.message-list`、`.msg`、`.composer` 消费该宽度变量，避免消息列与输入区在不同渲染路径上出现双重宽度口径。
- React 壳层在 `shell.css` 中把 `1100px` 及以下统一视为抽屉式导航工作台：`primary-nav` 改为贴左侧视口边缘的全高抽屉，并在当前运行页直接承载会话列表，避免导航和会话列形成双浮层；`760px` 及以下继续压缩按钮和内边距，保证真手机宽度下的可触达性。
- `ConversationWorkspace` 负责运行页头部、`Details` 面板、消息区与 Composer 的排版，`ConversationRuntimeProvider` 负责 `compact` 断点感知、SSE 收口、任务轮询和草稿恢复；其中桌面端输入性能约束由 provider 的延迟草稿落盘与 workspace 的时间线 memoization 共同保证。Go 侧源码测试与前端组件测试共同约束这组契约。
- `chat.js` 内所有前端时间展示统一走同一北京时间格式化器，固定 `timeZone=Asia/Shanghai`、`hourCycle=h23`；时间标签输出 `HH:mm`，绝对时间输出 `YYYY-MM-DD HH:mm:ss`；控制台与账户管理视图的分钟精度时间戳输出 `YYYY-MM-DD HH:mm`，同样由共享时间格式器负责。
- Cron 创建表单默认时区直接复用同一前端常量 `Asia/Shanghai`，不再依赖浏览器本地时区探测。
- `Chat / Agent Runtime / Terminal` 会话列表前端继续按 `hashSessionIDShort(session_id)` 生成 8 位短 hash，并作为运行页 URL query、Agent Session Profile 与预览域名使用的短标识口径；共享会话列表项不再展示短 hash，Terminal 不再把完整 `terminal_session_id` 填入 `shortHash` 字段。
- `src/app/routeState.ts` 负责运行页路由与会话 query 协调：路由只解析 canonical path，不再使用 hash fragment；`Agent Runtime / Terminal` 统一维护 `session_id` 多会话恢复参数，写入时通过 `sessionRouteToken` 把完整会话 id 收敛为 8 位短 hash。`ConversationRuntimeProvider.tsx` 对 `chat` 始终使用 `alter0-chat`，不写会话 query；对 `agent-runtime` 在初始化时读取 `session_id`，通过服务端列表和本地快照把短 hash 解析为完整会话，再在会话 focus、创建、删除和服务端恢复后回写短 hash URL。`ReactManagedTerminalRouteBody.tsx` 对 Terminal 采用同一策略。query 缺失或目标会话不存在时，运行页才回退到 `sessionStorage` 快照与服务端列表默认项。
- Markdown 渲染必须避免原始 HTML 透传；长路径、代码块和 diff 只在内容块内部滚动。`RuntimeMarkdown.ts` 作为 Web Shell 的共享安全 Markdown 渲染器，被 `ChatMessageRegion`、Terminal 步骤/最终输出、`RouteFieldRow` 的正文模式、Memory 文档、Task 请求/结果/日志/产物摘要、Control 描述、Agent/Codex 说明与 Session Profile 非等宽字段共同复用；机器标识类字段继续走纯文本或等宽展示，不进入 Markdown 解析。
- `ChatMessageRegion.tsx` 统一负责 Conversation runtime 的消息正文与尾部元信息；已完成的 Chat / Agent Runtime 助手消息不渲染尾部元信息，运行中/排队/失败等瞬时状态才渲染紧凑状态标签，且不再附带逐条时间，避免在每条回复后重复输出 route/source/status/time 标签。
- `ChatMessageRegion.tsx` 与 `ReactManagedTerminalRouteBody.tsx` 通过共享 `RuntimeTimeline` process block 输出统一的 `runtime-thinking-shell / runtime-thinking-toggle` 思考披露入口；入口只渲染 `Thinking / 已思考` 与步骤数量，不传入 duration meta，`shell.css` 以透明、无边框、无阴影和内容宽度收缩样式覆盖旧 Process 卡片皮肤。
- `shell.css` 通过 `.runtime-workspace-head.is-sticky`、`.workspace-header-status` 和 `.workspace-header-details` 维护三条运行页共享的固定 workspace header 视觉状态：标题区吸顶、状态按钮按 `ready / busy / failed / interrupted / exited` 输出统一颜色反馈，但可见层只保留信号本身，并通过 `inline-flex` 信号槽直接复用会话列表 `.runtime-session-signal` 的中心点、描边与波纹规格，`Details` 入口沿同一低圆角控制台按钮系统渲染；`.workspace-details-layer / .workspace-details-backdrop / .workspace-details-panel` 负责把详情面板挂到顶层浮层、限制最大可视区域、提供点击外部关闭与独立滚动容器，并通过更高层级、明确背景和 `dialog` 语义保证浮层稳定可见，`.workspace-details-content / .workspace-details-summary / .workspace-details-body` 则把首屏统一为紧凑摘要栅格、窄标签字段行与压缩复制控件，Conversation 与 Terminal 只在详情内容内部保留差异化组件。`ConversationRuntimeProvider.tsx` 分离 `inspectorOpen` 与 `inspectorTabOpen`：`inspectorOpen` 只控制 Composer 配置面板，`inspectorTabOpen` 控制当前 `Agent / Deliverables / Model / Tools / MCP / Skills / Session Profile` 内容区；`toggleInspector(tab)` 在当前 tab 上再次触发时只切换内容区展开状态，不影响 workspace `Details` 浮层，Agent Runtime 额外通过 `deliverables` tab 呈现结构化交付契约，通过 `session-profile` tab 呈现结构化实例属性。
- `shared/visibility/usePageActivation.ts` 提供运行页共享的 page-activation hook：统一监听 `visibilitychange` 与 `focus`，并用短时间去抖吸收 `visible + focus` 的连续触发。`ConversationRuntimeProvider.tsx` 与 `ReactManagedTerminalRouteBody.tsx` 都复用这条链路；Conversation 在页面重新变为前台可见时按当前路由补偿刷新会话列表、当前活动会话详情与 pending task 状态，`agent-runtime` 额外回源 `/api/agent/session-profile` 刷新 `Details > Session Profile`，Terminal 则立即刷新会话列表与当前活动会话详情。
- `shell.css` 额外维护共享 header 状态信号样式：`.runtime-session-signal` 及其 `ready / busy / failed / interrupted / exited` 变体负责 workspace header 的微型中心点、双层波纹脉冲和红黄绿状态令牌，并在 `prefers-reduced-motion` 下回退为静态信号；会话列表不再使用该状态灯，改用 `.runtime-session-loading` 表达处理中状态。
- `ConversationWorkspace.tsx` 与 `ReactManagedTerminalRouteBody.tsx` 都已经退化为 controller + route wrapper：统一由 `RuntimeWorkspacePage.tsx` 产出紧凑工作区头部、时间线、`Details` 面板、Composer，并把会话列表注册到左侧 `PrimaryNav`，页面只负责注入会话数据、变体 class 和路由专属交互；`ConversationWorkspace.tsx` 不再向移动 workbar 传入 `mobileSessionButtonLabel`，`chat` 路由也不再向 `RuntimeComposer` 注入 `Session` 工具按钮，`agent-runtime` 继续保留 Composer 内的会话设置入口；`ReactManagedTerminalRouteBody.tsx` 同样不再向移动 workbar 传入 `mobileSessionButtonLabel`。`RuntimeComposer.tsx` 固定输出单一圆角助手输入面板，DOM 仍由 `runtime-composer-body + runtime-composer-toolbar` 组成，textarea 透明无内边框并与工具栏共享白色 surface，工具栏左侧按路由收口为附件按钮或 `Session` 会话设置入口与附件按钮，运行态配置通过面板内部 tab 切换，右侧只保留深色提交动作；`shell.css` 通过共享 padding、min-height、居中宽度和移动端安全区规则保证主输入区拥有稳定可读宽度；`PrimaryNav.tsx` 维护纯文字品牌位、当前运行页会话列表与语言切换，左下角不再额外挂接账号信息区。Conversation 继续保留 `data-conversation-*` 钩子，Terminal 继续保留 `terminal-*` 钩子与布局变体，但公共 DOM 主契约已经收敛到 `runtime-workspace-* / runtime-composer-* / runtime-timeline-*`。
- `ConversationWorkspace.tsx` 的移动端输入框继续复用 Terminal 已验证过的首次聚焦链路：`onPointerDownCapture / onTouchStartCapture` 在首次触摸时调用 `focus({ preventScroll: true })`，`useLayoutEffect` 在输入框聚焦期间监听 `window.scroll` 与 `visualViewport.resize/scroll`，把页面锚定回 `scrollY = 0`，避免首次弹出软键盘时公共操作行丢失、页面整体上移或测试环境下出现首帧分辨率跳变。
- `shell.css` 在共享 runtime 作用域下继续叠加 workbench 精修：会话卡片增加左侧激活竖线与尾侧轻量操作，`Details` 面板、空态阅读区与 Composer 使用统一的浅色 surface、低圆角与阴影密度；首页 Composer 固定为单一圆角助手输入面板，textarea 去除内层边框和 resize，工具栏左侧对齐工作区工具、附件与可选 meta，右侧单独保留深色 icon submit；桌面端 form 使用 `width: min(100%, 860px)` 居中，移动端 form 回落为满宽并通过 `env(safe-area-inset-bottom)` 与 `--keyboard-offset` 贴住可见底边。`ConversationWorkspace.tsx` 在空态为 console panel 与 chat screen 追加 `is-empty` class，`shell.css` 以 `overflow: hidden + overscroll-behavior: none` 锁住空态滚动，并通过低对比网格与细弧线背景提升空态画布层次，避免窄屏空页把头部操作行顶离可视区。
- `shell.css` 在 `@media (max-width: 1100px)` 下对工作台性能做额外收敛：关闭 `body::before/after` 光晕层，移除 `primary-nav / chat-pane / mobile-backdrop / runtime-workspace-session-pane-backdrop / runtime-workspace-session-pane-shell / runtime-workspace-body` 的 `backdrop-filter`，把移动运行页与抽屉回落为静态浅色表面，减少真机滚动和抽屉切换时的整页合成开销。

### 验证策略

- Web handler 测试覆盖会话创建、历史隔离、流式事件和取消语义。
- 前端 E2E 覆盖 Chat、Agent、移动端输入、设置面板和长会话渲染。
- 前端组件测试需覆盖 React 工作台的稳定契约，至少校验 `WorkbenchApp` 的 canonical path 路由、语言切换、移动端导航收口、左侧主导航会话列表，以及 Conversation / Terminal workspace 的固定 header、消息区、Composer 和 `Details` 面板未被回归破坏；Conversation 消息区还需覆盖轻量 IM 气泡 DOM 与样式契约、长历史最新优先渲染与加载更早批次。
- `legacyRouteLayoutStyles.test.ts` 需继续对 `chat-core.css` 的 `Process` 阅读契约做源码断言，至少覆盖步骤标题收缩、正文整列宽度和 `max-width: 760px` 下的移动端可读性约束。
- `ReactManagedTerminalRouteBody.test.tsx` 需覆盖三类终端步骤回归：步骤头保留独立 `.terminal-step-toggle-icon` 且展开前后状态与标题主列同时成立；命令/终端输出类 block 继续渲染 `<pre><code>`；说明类 block 在详情展开后必须落到 `.runtime-markdown-rendered`，并对零宽字符或“每字一行”病态内容完成可读性归一化。
- 图片输入链路的最小稳定测试面包括：前端文件选择与剪贴板图片读取限制、Composer 附件预览与移除、Web 消息接口对附件元数据的编码、`HybridNLProcessor` 对图片 part 的构造与禁回退约束、OpenAI Responses / Chat Completions 适配层对视觉内容的序列化。
- `src/app/routeState.test.ts`、`src/app/WorkbenchApp.test.tsx`、`features/shell/legacyShellConfig.test.ts`、`features/shell/components/PrimaryNav.test.tsx`、`shellLayoutStyles.test.ts`、`legacyRouteLayoutStyles.test.ts` 与各 `ReactManaged*RouteBody.test.tsx` 共同覆盖路由解析、三入口主导航、Management 工具入口、Management 页族标记、语言切换、Conversation runtime 入口、Agent/Terminal/Memory/Control/Tasks/Sessions 页面取数与窄屏布局契约；Go 侧 `internal/interfaces/web/server_*_test.go` 继续通过源码与嵌入资产断言校验 `WorkbenchApp`、`ConversationRuntimeProvider`、`ConversationWorkspace`、`ReactManagedRouteBody`、共享样式和静态资源分发策略。
- 图片输入链路的最小稳定测试面包括：前端文件选择与剪贴板图片读取限制、发送 payload 与会话恢复预览资产的分离、Composer 附件预览与移除、AI markdown 图片渲染、Web 消息接口对附件元数据的编码、`HybridNLProcessor` 对图片 part 的构造与禁回退约束、OpenAI Responses / Chat Completions 适配层对视觉内容的序列化。
- 回归测试优先覆盖空白会话重复、软键盘残留空白、整段列表重建、断流恢复与残留 `In Progress` 等高频问题。

## Agent Capability & Memory

### 包边界

- `internal/agent/application` 负责内置 Agent 与用户 Agent Catalog 聚合。
- `internal/control/domain` 与 `internal/control/application` 负责 Agent Profile、Skill、MCP 的控制面配置。
- `internal/execution/application` 负责运行时上下文解析，包括 Skill、MCP、Memory 和 Agent Session Profile。
- `internal/execution/infrastructure` 负责 ReAct、Codex CLI、工具执行与模型适配实现。
- `internal/orchestration/application` 负责会话记忆、长期记忆、压缩和任务摘要召回。

### 调用链路

```text
Agent message
  -> Agent Catalog / Profile resolution
  -> Runtime context resolution
  -> ReAct loop
  -> ToolExecutor
  -> codex_exec | memory tools | delegate_agent | complete
  -> Final response + Process
```

### 技术约束

- Agent 负责理解与驱动，具体文件、仓库、Shell、页面产出统一通过 `codex_exec`。
- `codex_exec` 使用 stdin 传递执行指令，不通过命令行拼接长上下文。
- 存在可用 Provider 且进入 Agent / ReAct 链路时，Agent 自身吸收 Skill、MCP、Memory 与运行时上下文，只向 Codex 下发当前步骤的纯执行指令。
- `internal/llm/domain/react.go` 在工具面存在 `complete` 时禁止把普通 assistant 文本直接当作最终答案；模型若未调用 `complete` 就尝试结束，ReAct 循环会把该回复作为中间消息保留，并回注一条运行时纠偏提示，要求它继续使用工具直至显式收口。
- `internal/storage/infrastructure/localfile.SessionStore` 使用分文件布局持久化 Chat / Agent Runtime 历史：`Agent Runtime` 保存时按 `MessageRecord.Source.AgentID` 与 `SessionID` 分组写入 `.alter0/sessions/<agent_id>/<session_id>.json` 或 `.md`，缺少 Agent 来源时写入 `_default` 分组；`Chat` 的固定长期会话 `alter0-chat` 则按北京时间 05:00 的归档日边界写入 `.alter0/sessions/_default/alter0-chat/<YYYY-MM-DD>.json` 或 `.md`，05:00 前消息归入前一归档日，05:00 及之后归入当天归档日。删除会话后保存全量快照会清理已不存在的会话文件。加载时扫描该目录布局，并读取旧版 `.alter0/sessions.json` / `.alter0/sessions.md` 聚合文件；当两种布局同时存在时按消息身份去重合并，随后立即把合并结果重写为新的分文件布局并删除旧聚合文件，避免迁移中断造成历史缺失。
- Web 上传的会话附件经 `internal/interfaces/web/server.go` 与 `session_attachment_store.go` 规范化后统一写入 `alter0.user_input.attachments`；图片附件额外保留兼容性的 `alter0.user_input.image_attachments`。`/api/sessions/{session_id}/attachments` 现在支持“原文件 + 可选预览”模型：图片仍落原图与预览图，普通文件只落原文件并让 `preview_url` 回退到 `asset_url`。Conversation runtime 消息接口随后只携带 `id + asset_url + preview_url` 引用，服务端再解析出工作区内的原图路径写入元数据；前端渲染层再按场景分流，缩略位读取 `preview_url`，回显与预览弹层读取 `asset_url`。assistant 最终回复中的 markdown 外链图片则由 `internal/orchestration/application/session_output_image_assets.go` 在 SessionPersistenceService 中做结果后处理：仅对可下载的 `http(s)` 图片做抓取，写入同一 Session 附件目录，并把 `result.Output / result.ProcessSteps[].Detail` 里的图片地址改写为 `/api/sessions/{session_id}/attachments/{asset_id}/original`。`/api/messages`、`/api/agent/messages`、Terminal 输入与 Control Task follow-up 输入都会复用同一附件目录与交付 URL。`internal/execution/infrastructure/hybrid_nl_processor.go` 继续只把图片子集解码成 `llmdomain.Message.Parts`；Terminal 侧普通文件则不进入多模态图片 part，而是在执行前写入 Terminal 工作区并通过 prompt 注入稳定路径，交给 Codex 读盘。带图请求不进入异步 Task，也不会在模型链失败后静默回退到 Codex CLI，避免把视觉请求错误降级为纯文本执行。
- 不存在 Provider、Agent 初始化失败或请求直接进入 Terminal / 直连 Codex 时，`internal/execution/infrastructure` 需要为当前会话编译原生 Codex Runtime：独立 `CODEX_HOME/config.toml`、工作区 `AGENTS.md` 与 `.alter0/codex-runtime/*`，并把启用的 MCP Server 渲染为原生 `mcp_servers.*` 配置。`codex_cli_processor.go` 会解析 Codex JSONL 的 `thread.started.thread_id` 并写入当前 Session workspace；Agent Runtime 写入 `.alter0/codex-runtime/thread.json`，同一 Session 下次继续直连 Codex 时读取该文件并通过 `codex exec resume <thread_id> -` 从 stdin 续写 prompt；Chat 固定会话 `alter0-chat` 写入 `.alter0/codex-runtime/threads/<YYYY-MM-DD>.json`，归档日按北京时间 05:00 计算，新归档日文件不存在时直接启动新的 Codex thread。旧版 Chat `.alter0/codex-runtime/thread.json` 在读取时会按 `updated_at` 或文件 mtime 迁移到对应归档日文件，并移除旧文件。`internal/codex/infrastructure/runtimeconfig` 写入的托管 `AGENTS.md` 固定追加工作区范围约束，要求 Codex 只在当前工作区及其派生 repo clone/产物路径内执行，不得改动其他会话、服务或工作区外仓库，除非当前任务明确点名这些目标。
- Memory Files 注入需要携带路径、存在状态、可写性、内容快照和自动召回片段。
- `internal/orchestration/application.LongTermMemoryStore` 以 Markdown `PersistencePath` 作为长期记忆主存，并同步维护派生 JSON 索引；默认索引路径为同名 `.index.json`，也可通过 `LongTermMemoryOptions.IndexPath` 显式指定。索引记录 entry id、tenant/user scope、tier、kind、key/value、tags、状态、来源 Session、更新时间与关键词 tokens；索引不是事实源，重启时仍从 Markdown 主存恢复，后续写入或 `Flush()` 会重建索引。Snapshot 阶段会基于相关命中生成 `ActiveRecall`，把高相关条目压缩为短摘要、来源 entry id 与命中计数，并在 `[LONG TERM MEMORY]` 段落和结果 metadata 中标记本轮主动召回。
- `internal/llm/domain.Message` 允许同时携带纯文本 `Content` 与结构化 `Parts`；`internal/llm/infrastructure/openai_client.go` 在 `openai-responses` 与 `openai-completions` 两条路径上都要把用户图片 part 序列化为官方视觉输入结构，同时继续保留 assistant `tool_calls` 与后续 `tool` 消息的 `tool_call_id` 稳定配对；否则 Provider 会把该轮请求判定为非法工具消息序列或直接丢失图片输入。
- 私有 `AGENTS.md`、私有 Skill、Agent Session Profile 分别承担协作边界、可复用打法、会话画像与实例属性职责，不混写一次性任务细节；`skill_context_resolver` 对当前 Agent 私有 Skill 执行强制注入，即使请求 metadata 携带 `alter0.skills.exclude` 也不会移除该私有 Skill，前端锁定只是同一规则的交互表达。
- `internal/execution/application/agent_session_profile.go` 负责渲染和回写 `Agent Session Profile` 自动块：保留 `Notes` 人工区，自动维护 `Session Identity / Session Scope / Instance Attributes`。其中 `Instance Attributes` 统一合并历史已存在属性、请求 metadata 注入的增量属性，以及 `coding` 自动派生出的 `repository_path / branch / preview_subdomain` 等交付属性；`travel` 只保留会话事实字段，不在执行侧预写 `guide_html_url`。`internal/execution/infrastructure/session_profile_codex_extractor.go` 提供受限 Codex fallback：只接收 schema、已有属性和最新用户消息，返回 JSON patch，不直接写文件。`internal/interfaces/web/agent_session_profile.go` 提供只读聚合接口，读取同一路径下的实例属性并与 Agent 预设字段定义拼装成前端 `Details` 视图数据；其中 `guide_html_url` 仅在 `workspaceService.ResolveService(session_id, "travel")` 命中公开只读 `travel` 服务时动态补齐，否则即使 profile 文件里残留旧值也会被隐藏。前端 `ConversationRuntimeProvider.tsx` 在 agent 会话首次打开与每轮消息收口后都会重新请求该接口，避免 `Session Profile` 因本地缓存停留在旧的 `city / days / hotel_area` 值。
- `internal/agent/application/catalog.go` 会把 Agent 结构化 `deliverables[]` 与 `completion_checks[]` 分别通过 `alter0.agent.deliverables`、`alter0.agent.completion_checks` 注入执行 metadata；`internal/execution/infrastructure/hybrid_nl_processor.go` 在构造 Agent system prompt 时附加当前 delivery contract，并在 `complete`、Agent fallback Codex、direct Codex 成功返回后执行 `completion_checks` 驱动的通用产物校验。
- `internal/agent/application/builtin.go`、`internal/execution/infrastructure/hybrid_nl_processor.go`、`internal/execution/infrastructure/codex_cli_processor.go` 与 `internal/execution/infrastructure/codex_native_runtime.go` 统一维护工作区隔离提示：内置 Agent system prompt、Provider/ReAct system prompt、Codex fallback prelude、托管 `AGENTS.md` 与 `runtime.md` 均要求把具体执行限定在当前 Session 工作区及其专属 repo clone、附件和产物路径内，并明确禁止顺带修改其他 Session、无关服务或工作区外仓库。
- `internal/agent/application/catalog.go` 还负责统一 Agent tool 默认值：`normalizeRuntimeAgent` 会在内置 Agent 与托管 Agent 两侧都补齐 `search_memory`，同时保留显式工具顺序和去重结果，让 `/api/agents`、新建 Agent Runtime 会话和执行 metadata 的默认工具面保持一致。
- `internal/execution/infrastructure/hybrid_nl_processor.go` 统一承担 Agent 通用产物检查、专门修复轮与 `travel` 公开 URL 回填：执行器先按 `completion_checks` 逐项校验 Session 文件、workspace service、Session 属性等确定性产物；若首轮结果未通过且相应 check 声明了 `repair_instruction`，执行器会复用当前 Session 工作区再触发一次专门的 Codex 修复 prompt，只允许围绕失败的产物检查完成补齐。修复轮若未真实落盘页面、未真实写入服务注册表或未补齐所需属性，最终状态仍保持阻塞。执行器不再自动固化正文攻略为 fallback HTML，也不会在没有 Codex 实际执行结果的前提下替 Agent 伪造发布结果；只有当注册表中已存在真实发布的 `travel` 服务时，才会把该公开 URL 追加回最终结果。
- `search_memory`、`read_memory`、`write_memory` 只操作已解析进 `memory_context` 的记忆文件。
- Agent Memory Web 聚合接口只读返回长期记忆、天级记忆、强制上下文与说明文档；任务摘要刷新走 Task summary 子域接口。

### 验证策略

- Agent Catalog 测试覆盖内置 Agent 与用户 Agent 聚合、保留 ID 冲突。
- Execution 应用测试覆盖 Skill/MCP/Memory Context 注入，并约束当前 Agent 私有 Skill 不被 exclude 过滤。
- Infrastructure 测试覆盖 `codex_exec` stdin、ReAct 迭代上限、Process 输出和工具错误收口。
- Memory 测试覆盖短期回填、长期召回、Markdown 编解码、压缩和任务摘要深检索。

## Task, Terminal & Workspace

### 包边界

- `internal/task/domain` 定义任务状态、来源字段、摘要与执行元数据。
- `internal/task/application` 负责异步执行池、复杂度预判、任务生命周期和心跳续租。
- `internal/tasksummary/application` 负责任务摘要存储和运行态 Markdown 记录。
- `internal/terminal/domain` 定义 Terminal 会话态、turn 和 step。
- `internal/terminal/application` 负责 Terminal 会话持久化、恢复、输入续写和工作区分配。

### 调用链路

```text
High-complexity message
  -> Task acceptance
  -> Async executor
  -> Workspace
  -> Codex CLI / Agent execution
  -> Task logs + heartbeat + artifacts
  -> Session result summary
```

```text
Terminal input
  -> Terminal session store
  -> Codex CLI resume/start
  -> Turn/step append
  -> Terminal view model
  -> Web polling / stream response
```

### 技术约束

- Task 需要保存来源字段，支持从任务回会话、从会话查任务，并支持按触发类型、通道、来源消息与结果消息过滤。
- 长任务通过心跳续租运行窗口，浏览器 SSE 保活与后台心跳分离。
- Control 任务交互式续写通过追加输入创建 follow-up Task，不直接改写原任务执行记录。
- Web 不直接暴露本地文件路径，产物通过引用、下载或预览接口交付。
- Task 产物列表响应需要过滤本地 URI；下载和预览由任务接口按 artifact id 读取并输出安全响应头。
- Memory 任务视图读取 Task 与 task summary 数据，支持任务摘要重建，但不直接执行 retry/cancel。
- 工作区按 Chat/Agent、Task、Terminal 分层隔离，删除会话或 Terminal 时同步清理对应目录。
- 直连 Codex 的 Chat / Agent 会话在自身工作区下维护 `.alter0/codex-runtime/` 与 `.alter0/codex-runtime/codex-home/`；Agent Runtime 的 Codex thread id 写入 `.alter0/codex-runtime/thread.json`，Chat 的 Codex thread id 写入 `.alter0/codex-runtime/threads/<YYYY-MM-DD>.json`；Terminal 会话在 `.alter0/workspaces/terminal/sessions/<terminal_session_id>/codex-home/` 下维护独立 `CODEX_HOME`。
- Terminal 会话态与 turn/step 执行态分离，历史 `running / starting` 需要兼容归一。
- Terminal 会话详情聚合 turn 摘要；step 明细按 `session_id / turn_id / step_id` 单独读取，避免会话列表一次性加载大块执行日志。
- `internal/terminal/application/service.go` 在 `InputWithAttachments` 中把 Terminal 附件规范化为 turn 附件、在工作区 `input-attachments/` 下写入本轮输入文件，并按类型拆分消费：图片继续通过 `codex exec -i <file>` 或 `codex resume -i <file>` 进入 Codex 视觉输入，普通文件则在同轮 prompt 中附上 `input-attachments/<turn_id>/<filename>` 形式的 workspace 相对路径，要求 Codex 按需直接读盘；turn 摘要与持久化快照同步保留附件元数据，供 Terminal 输入草稿与图片历史回显复用。Terminal 输入请求携带的 `SkillContext` 会在每轮执行前渲染到工作区 `.alter0/codex-runtime/skills.md`，并通过托管 `AGENTS.md` 指令要求 Codex 只应用本轮选择的 Skill；空选择也会写入“未选择”标记，避免旧轮次 Skill 指令残留。
- `internal/execution/infrastructure/codex_cli_processor.go` 的流式执行链路默认使用 `codex exec --json -` 从标准输入喂入 prompt，并直接消费 stdout JSONL 事件；若当前执行上下文已解析出可用 Codex thread id，则改用 `codex exec resume --json <thread_id> -` 续写同一线程。Chat 的可用 thread id 只来自当前北京时间 05:00 归档日文件，Agent Runtime 的可用 thread id 来自当前 Session 文件。不再依赖已移除的 `--progress-cursor` 参数，避免新版本 Codex CLI 在 Terminal、Chat、Agent 共用执行器下启动失败。
- Terminal 应用层在 Codex CLI 返回远端 compact 失败时，会把当前会话的运行线程指针复位为初始 `terminal_session_id`，保留原工作区与日志，并让后续输入自动走新线程而不是继续 resume 已失效 thread。
- `internal/interfaces/web/frontend/src/features/shell/components/ReactManagedTerminalRouteBody.tsx` 与 `composerImageAttachments.ts` 现在共同承载 Terminal 通用附件体验：图片仍走读取/缩放/预览链路，并通过共享剪贴板图片提取逻辑支持 textarea 直接粘贴；普通文件直接读成原始 data URL，并在输入条里显示文件条目。前端选中或粘贴附件后统一先调用 `POST /api/sessions/{session_id}/attachments` 把原文件写入当前 Session 工作区，再在草稿与提交体里优先保留稳定 `id + asset_url + preview_url?` 引用，避免 Terminal 首发会话把原始 `data_url` 长期留在浏览器状态里。`codexSlashCommands.ts` 维护 Chat / Agent / Terminal 共用的 Web 适用 Codex CLI 斜线命令候选、查询与补全文本规则；命令表按 Codex 帮助中的作用分组顺序排列，并用短动作标签控制候选行密度，同时排除权限、TUI 显示、键位、剪贴板、登录退出和本地 CLI 会话管理类命令。Terminal 仅在活动会话 `shell` 明确包含 `codex` 时把候选挂到 `RuntimeComposer.inputAssistContent`，普通 shell 不渲染该辅助。共享 `ScrollJumpStrip` 组件负责 Chat / Agent / Terminal 三条运行页的四键阅读定位：按当前视口中的可见消息块或 turn 集合计算 `上一条 / 下一条` 目标，并在 turn 列表、消息折叠态或窗口尺寸变化后才失效位置测量缓存；滚动过程只复用缓存并重算可视区交集。组件同时监听 `document.selectionchange`，当当前滚动容器内存在有效文本选区时，立即收起四键 overlay 并禁用命中区，待选区清空后恢复。`shell.css` 对 Terminal 输出正文、Markdown 容器和代码结果显式保留 `user-select: text`、`-webkit-user-select: text` 与 `-webkit-touch-callout: default`，并把 Terminal 滚动屏幕的 `touch-action` 还原为 `auto`；`RuntimeMarkdownShell` 将正文 DOM 排在复制工具栏之前，Terminal 最终输出保持普通静态 Markdown/text DOM，不设置 `contenteditable`、`inputmode`、caret、focus 编辑态、脚本 `touchstart` 选区、`Selection API` 强制选中、假选中 class 或浮动复制层。阅读定位按钮容器默认不参与 pointer 命中，仅可见按钮恢复命中，避免空白 overlay 截获拖选或长按选中。Conversation 与 Terminal 运行页分别在 `ConversationWorkspace.tsx`、`ReactManagedTerminalRouteBody.tsx` 中按活动 `session_id` 记录初始定位状态：每个已有内容会话首次渲染时间线或 Terminal turn 列表时同步把滚动容器的 `scrollTop` 设为 `scrollHeight`，并在下一帧仅当用户尚未移动滚动位置时复核一次以覆盖首次布局高度变化；Conversation 会在同一活动会话消息数增加时再次把滚动容器贴到底部，用于发送后立即展示新消息；同一 `session_id` 的后续流式 patch、轮询刷新、Process 展开和草稿输入不再强制回底。Terminal 的前后台恢复和轮询详情刷新仍优先使用 `scrollRestoreSnapshotRef` 保持原阅读锚点。公有 Skill 列表首次加载完成后，前端会在当前会话尚未做过手动选择时默认选中全部可用公有 Skill，仅排除 `default-nl` 与 `memory`；一旦用户显式调整，后续刷新只保留仍然可用的选择结果，不再自动补回。`internal/interfaces/web/server_terminal.go` 直接把 Terminal `attachments[]` 转成 `terminalapp.InputRequest.Attachments`，并把 `skill_ids[]` 过滤为控制面启用且非私有的 `SkillContext`；`controlTaskTerminalInputHandler` 则继续把 follow-up 输入编码进统一消息元数据后交给 Task 服务。
- `ReactManagedTerminalRouteBody.tsx` 通过 `usePageActivation` 维护 Terminal 的前后台恢复：页面隐藏时仅更新 `pageHidden` 供轮询降频使用；页面重新可见或重新获得焦点后，会先刷新会话列表，再在存在活动会话时捕获当前滚动锚点并回源活动会话详情，避免后台期间新增输出、标题变化或状态切换在恢复前台后继续停留在旧视图。Terminal 前端同时在删除成功后把 `session_id` 写入本地删除屏蔽集合，后续列表刷新与单会话详情回源都会先过滤该集合，防止服务端短暂返回旧快照时把已删除会话重新补回左侧列表；共享 `RuntimeWorkspacePage.tsx` 的尾侧删除按钮也会吞掉 `mouse/touch` 删除手势，避免删除当前活动会话时把同一手势透传成下一条会话的选中动作。
- `ReactManagedTerminalRouteBody.tsx` 负责把 Terminal 领域状态映射到共享状态信号：`ready` 直接输出绿色 `ready`，`busy` 输出黄色 `busy`，`exited / interrupted` 等非活跃态统一收敛为红色失败信号，再复用 `RuntimeWorkspacePage.tsx` 的共享会话条目与 header 渲染链路。
- Terminal 跨设备共享同一 Web 登录态下的服务端会话历史，不再按 browser client 分桶。
- `chat-terminal.css` 在真手机宽度下允许 Terminal 工作区头部切换为多行排布：标题最多两行，状态与操作工具栏按可用宽度换行，避免横向溢出。
- `WorkbenchApp` 在根壳层安装共享 `mobileViewportSync` controller，把 `VisualViewport` 变化稳定写入 `--mobile-viewport-height / --keyboard-offset`；移动端 App Shell 在键盘弹起期间保持基线高度，避免整个 workbench 被 `visualViewport` 收缩带着上移；Terminal 移动端 Composer 通过 `bottom: var(--keyboard-offset)` 贴住可见底边，而不是通过增大 footer padding 把输入区继续留在文档流里。
- Conversation runtime 的窄屏四行工作区网格仅作用于带 `data-conversation-view` 的运行页，避免 Terminal 复用共享 surface class 时被错误套用 `auto auto minmax(0, 1fr) auto` 布局，导致长历史输出把 Composer 挤出屏幕。

### 验证策略

- Task 应用测试覆盖复杂度分流、并发上限、心跳续租、来源字段和删除清理。
- Web 测试覆盖任务列表、详情、日志流、retry/cancel、产物下载/预览、Memory 任务摘要重建和会话回链。
- Terminal 应用测试覆盖创建、恢复、续写、删除、详情读取、step 明细、状态归一和工作区分配。
- E2E 测试覆盖 Terminal 移动端输入、滚动、Process 折叠和跨设备历史口径。

## Control, Operations & Governance

### 包边界

- `internal/control` 管理 Channel、Capability、Skill、MCP、Agent Profile 和 Environment 配置。
- `internal/llm` 管理 Model Provider、上游 API type、OpenRouter 扩展与密钥状态。
- `internal/codex/domain` 负责 `auth.json` 快照、身份识别与额度状态模型；`internal/codex/application` 负责账号导入、状态刷新、独立登录会话、活动账号切换，以及通过 Codex app-server 的 `model/list`、`config/read`、`config/batchWrite` 读取真实运行时能力并更新 `model` / `model_reasoning_effort`；`internal/codex/infrastructure/localfile` 负责 `<active_codex_home>/alter0-accounts` 下的账号快照、备份与登录工作目录。
- `internal/interfaces/web/frontend` 中的 `ReactManagedCodexAccountsRouteBody` 负责 Codex Accounts 控制面的运行时概览、当前 Codex 管理区、账号列表、导入/登录操作侧栏与登录会话展示，并复用 `/api/control/codex/accounts*` 与 `/api/control/codex/runtime` 控制接口完成刷新、切换和运行时设置更新；概览区优先从当前账号状态抽取用户可读账号名、套餐与小时/周剩余额度，渲染为“主身份区 + 四项紧凑指标列”，其中额度卡使用带 `progressbar` 语义的进度条并展示后端返回的 `reset_at`，key/value 默认按同列对齐，活动路径不在概览区直出；运行时维护字段通过 `Runtime Details` 折叠区展开；当前 live `auth.json` 未匹配托管快照时，前端需回退展示 `active.live` 快照与 `active.quota`，并输出未托管提示，首次加载阶段则输出同构 skeleton 保持页面结构稳定；当前 Codex 管理区从后端返回的 `runtime.models` 构建 model 选择器，并按所选 model 的 `supported_reasoning_effort` 联动思考深度选择器，当前值不再在选择器下额外重复渲染；账号区按断点切换为桌面高密度行式列表、中屏全宽列表 + 双侧栏和窄屏单列紧凑列表，避免额度进度条、reset 时间、当前 model、思考深度与切换入口被横向滚动隐藏；账号状态文本和切换按钮采用扁平控制台样式，列表内部优先使用分隔线式布局而不是额外方框容器。
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
  -> Execution / Agent / Scheduler
```

```text
Environment restart
  -> Web confirmation
  -> supervisor
  -> optional git fast-forward
  -> build candidate binary
  -> readyz probe
  -> switch or rollback
```

### 技术约束

- Control 面只能管理运行时配置，不绕过编排层直接执行业务请求。
- Skill 与 MCP 专用接口需要复用统一 Capability 数据结构；Capability 审计记录生命周期动作，供控制面按类型查询。
- `cmd/alter0/builtin_skills.go` 负责注册内置 Skill，并在启动阶段校验所有 file-backed 内置 Skill 文件存在；当前内置集合包含 `deploy-test-service`、`frontend-design`、`artifact-preview`、`doc-coauthoring`、`fullstack-developer`、`code-reviewer`、`webapp-testing`、`find-skills`、`test-driven-development`、`ui-ux-pro-max`、`code-simplifier`、`code-review` 与 `brainstorming`。标准 skill 使用源码目录下的 `docs/skills/<skill_id>/SKILL.md`；plugin-style 的 `code-simplifier` 与 `code-review` 继续保留目录内 `.claude-plugin/plugin.json` 元数据，并分别以 `agents/code-simplifier.md`、`commands/code-review.md` 作为 alter0 的 file-backed 注入入口。
- `internal/codex/infrastructure/runtimeconfig` 负责在 Codex runtime 准备阶段物化 file-backed Skill：按当前服务进程工作目录向上查找可读的 skill 文件，定位 `docs/skills/<skill_id>/` 或 `docs/agents/<agent_id>/` 根目录，把整个目录复制到当前会话工作区 `.alter0/codex-runtime/skills/<skill_id>/`，并将注入给 Codex 的 `file_path` 改写为工作区内副本。Terminal Runtime 与 Agent/Codex Native Runtime 共用该物化逻辑，保证 Codex 的实际 cwd 不是源码仓库根时仍可稳定读取完整 `SKILL.md` 与同目录脚本、参考文件。
- `internal/agent/application/builtin.go` 中的 `coding` 内置 Agent 默认声明 `memory`、`deploy-test-service`、`frontend-design`、`artifact-preview`、`doc-coauthoring`、`fullstack-developer`、`code-reviewer`、`webapp-testing`、`find-skills`、`test-driven-development`、`ui-ux-pro-max`、`code-simplifier`、`code-review` 与 `brainstorming`；Agent Catalog 会把该列表写入运行时 metadata 的 `alter0.skills.include`，新建 Coding Agent 会话直接携带完整工程执行默认集，用户仍可通过会话级 Skill 面板调整后续消息的公有 Skill 选择。
- `internal/agent/application/builtin.go` 中的 `travel` 内置 Agent 默认声明 `memory`、`deploy-test-service`、`frontend-design`、`artifact-preview`、`doc-coauthoring`、`find-skills`、`ui-ux-pro-max` 与 `brainstorming`；Agent Catalog 会把该列表写入运行时 metadata 的 `alter0.skills.include`，新建 Travel Agent 会话直接携带旅游执行默认集，用户仍可通过会话级 Skill 面板调整后续消息的公有 Skill 选择。
- Models 控制面需要保持空 API Key 语义、占位值过滤、禁用态恢复和默认 Provider 收敛。
- Environment registry 按 Web & Queue、Async Tasks、Terminal、Session Memory、Persistent Memory、LLM 模块声明 key、类型、默认值、校验规则、敏感性与生效方式。
- Environment 配置更新写入 audit store，控制面按时间倒序读取变更记录。
- Codex Accounts 服务固定解析当前活动 `CODEX_HOME`，未显式设置时回退到 `$HOME/.codex`；托管账号写入 `<active_codex_home>/alter0-accounts`，活动账号切换只替换 `<active_codex_home>/auth.json`。
- Codex 运行时状态接口同时返回活动 `auth.json`、`config.toml`、CLI 命令、当前 profile、活动 model、思考深度、配置来源与 `model/list` 返回的可选 model 能力集；更新运行时设置时，后端先通过 `config/read` 解析当前生效 key path，再调用 `config/batchWrite` 更新 `model` 与 `model_reasoning_effort`，并触发 `reloadUserConfig` 让当前运行时立即生效。
- 独立登录会话通过临时 `CODEX_HOME` 执行 `codex login`，完成后再把新 `auth.json` 保存为托管账号，避免直接污染当前正在服务的运行时认证。
- 独立登录会话在执行 `codex login` 前需显式创建隔离 `CODEX_HOME`，并用覆盖语义注入环境变量，避免宿主进程已有 `CODEX_HOME` 影响登录结果落盘位置。
- LLM 运行参数 `llm_temperature`、`llm_max_tokens`、`llm_react_max_iterations` 通过 Environment 配置即时或重启后参与运行时解析，仍受 Provider 与模型能力约束。
- Runtime 重启必须由 supervisor 托管，候选实例通过 readyz 后才切换；当 `sync_remote_master=true` 时，只允许回滚 Git 已跟踪改动，不得清理未跟踪文件或目录。
- 共享 Web 运行时内置通用 workspace service 注册表 `.alter0/workspace-services.json`：控制面 `PUT /api/control/workspace-services/{session_id}` 注册默认 `web` 服务，`PUT /api/control/workspace-services/{session_id}/{service_id}` 注册附加服务。`frontend_dist` 默认校验 git 工作区和 `internal/interfaces/web/static/dist` 构建产物，并在 Host 命中 `<session_short_hash>.alter0.cn` 或 `<service>-<session_short_hash>.alter0.cn` 时优先分发 `/`、`/chat`、`/terminal`、`/assets/*` 与 `/legacy/*`；`travel` 服务是唯一前端静态例外，固定命中 `https://travel-<session_short_hash>.alter0.cn`，当注册路径根目录已存在 `index.html` 时直接把该目录作为静态攻略根目录公开分发，并继续对该 host 只返回静态 HTML/资源、直接阻断 `/api/*` 与其他工作台路由。`http` 服务既可反向代理到外部 upstream，也可由共享运行时按注册的 `start_command + workdir + port + health_path` 托管本地子进程。默认 `scripts/deploy_test_service.sh <session_id>` 会为 `web` 合成一条当前分支后端启动命令并注册给共享运行时，先构建前端产物，再让 `https://<session_short_hash>.alter0.cn` 整体代理到这份托管后端，从而让前端与 `/api/*` 保持同一版本；当 `service_id=travel` 且未显式传入 `--repo-path` 时，脚本默认回退到当前 Session 工作区根目录，直接发布已生成的静态攻略页。由于线上证书只覆盖 `alter0.cn` 与 `*.alter0.cn`，附加服务必须保持单级子域名格式，不能再生成 `<service>.<short_hash>.alter0.cn` 或 `<short_hash>.travel.alter0.cn` 这类二级嵌套 host。共享运行时自己的 `supervisor -> web child` 继续继承 `web_login_password` 作为主登录边界，托管 workspace service 子进程启动前会剥离 `ALTER0_WEB_LOGIN_PASSWORD` 并注入 `ALTER0_WEB_REUSE_GATEWAY_AUTH=1`，使预览后端只复用共享网关登录态，不再叠第二层鉴权。
- Web 登录态继续由 `server.go` 的 `authMiddleware + loginHandler` 统一管理；当请求 Host 命中主域或其预览子域时，登录 cookie 会把 `Domain` 收敛到根域 `alter0.cn`，使主域工作台与短哈希预览 host 共享同一登录会话，而不是各自维护孤立 cookie。交互页登录回跳通过 `loginNextForRequest` 归一化：`/` 与 `/chat` 只回跳到 `/chat`，`/terminal` 只回跳到 `/terminal`，其他 HTML 导航仍保留安全校验后的相对 Request URI；实际运行页的会话 query 由前端收敛为 8 位短 hash，避免会话级长 id 进入登录页和稳定页面 URL。
- systemd 基线统一 `HOME=/var/lib/alter0`，确保 Codex、gh、git signing、Node/Playwright 工具链使用同一运行账户上下文。
- 提交签名问题不得通过关闭签名绕过。
- 技术文档、需求文档和 README 更新按领域同步，避免需求与方案分离。

### 验证策略

- Control 测试覆盖 Channel、Capability、Skill、MCP、Agent、Environment、Codex Accounts 配置持久化、Capability 审计和 Environment audit。
- LLM 测试覆盖 Provider 创建、更新、缺失密钥恢复、默认项收敛和 OpenRouter 字段。
- Runtime supervisor 测试覆盖候选版本构建、readyz 切换、失败回滚和 metadata 展示。
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
