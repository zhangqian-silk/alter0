# Technical Solution

> Last update: 2026-04-21

`alter0` 的技术方案按与需求清单一致的领域模型维护。后续新增或调整需求时，技术方案必须落到对应领域与子域，不再按时间顺序、任务编号或零散专题堆叠。

## 维护规则

- 需求归属先确定领域路径，例如 `runtime.orchestration.intent`、`agent.execution.codex-exec`、`product.travel.workspace`。
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
| Product Domain | `internal/product`、`internal/interfaces/web/product_*`、`cmd/alter0/builtin_skills.go` | Product 目录、Draft Studio、Product 总 Agent、Workspace、Travel 产品域 |
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
- 命令路由优先于复杂度评估和模型执行。
- Cron 触发不直接调用执行器，必须复用编排链路。
- Cron runs 接口通过 Session history 按 `trigger_type=cron` 与 `job_id` 查询触发会话，不另建独立运行记录存储。
- `ExecutionPort` 是自然语言执行能力的稳定边界，具体实现可替换。
- trace、session、message、correlation 字段贯穿日志、指标、会话与任务。

### 验证策略

- 领域对象测试覆盖消息字段归一、路由结果和错误编码。
- 编排应用测试覆盖命令优先、自然语言分发、Cron 回注。
- 接口测试覆盖 CLI/Web 输入到 `UnifiedMessage` 的转换。
- Go 单测用例说明按 `docs/testing/unit-test-cases.md` 与各 Go 包路径下的 `TEST_CASES.md` 维护，并按 Runtime、Conversation、Agent、Task、Product、Control 领域路径归档。

## Conversation & Session Experience

### 包边界

- `internal/session/domain` 定义会话与消息数据结构。
- `internal/session/application` 负责会话持久化、历史查询和删除清理。
- `internal/interfaces/web` 负责 HTTP、SSE、Web 登录、页面路由和前端静态资源分发。
- `internal/interfaces/web/frontend` 负责 Web Shell 的 Vite + React 构建、legacy DOM shell 渲染和 `static/dist` 产物输出。
- `internal/interfaces/web/frontend/src/shared/api/client.ts` 负责统一 JSON 请求封装、错误收敛与登录失效回调，避免新前端页面继续散落原生 `fetch`。
- `internal/interfaces/web/frontend/src/shared/time/format.ts` 负责固定 `Asia/Shanghai` 的前端显示时区与标准时间格式，避免新旧页面时间口径漂移。
- `internal/interfaces/web/frontend/src/shared/viewport/mobileViewport.ts` 负责移动端断点、键盘偏移阈值与 viewport baseline 计算，避免 Chat、Terminal 与 route 页重复维护软键盘占位逻辑。
- `internal/interfaces/web/frontend/index.html`、`static/dist/index.html` 与登录页模板统一以 `html[lang="en"]` 启动；`src/app/WorkbenchApp.tsx` 通过写回 `document.documentElement.lang` 统一驱动中英文壳层文案切换。`renderLoginPage` 继续直接输出服务端 HTML，但视觉与文案已对齐工作台基线：复用 `IBM Plex Sans + Sora` 字体组合、近白卡片表面与安全入口 copy。
- `internal/interfaces/web/frontend/src/app` 承载当前 Web Shell 的顶层壳层：`App.tsx` 只负责挂载 `WorkbenchApp`；`WorkbenchApp.tsx` 负责主导航、路由分派、语言切换与桌面/移动导航态，并让普通 `page-mode` 路由页在窄屏下通过 `data-route-mobile-head` 输出共享 `Menu` 入口；`routeState.ts` 负责 hash 路由解析与派发；`WorkbenchContext.tsx` 暴露当前 `route / language / navigate` 以及移动端主导航状态与开关。根壳层通过 `app-shell[data-workbench-route]` 输出当前路由，控制页继续通过 `data-route` 暴露页级钩子。
- `internal/interfaces/web/frontend/src/features/conversation-runtime` 承载 `chat / agent-runtime` 的运行态：`ConversationRuntimeProvider.tsx` 负责会话创建/切换/删除、消息流、SSE 收口、任务轮询、草稿恢复、模型与能力项选择；`ConversationWorkspace.tsx` 通过 `features/shell/components/RuntimeWorkspaceFrame.tsx` 复用与 `Terminal` 相同的运行页骨架，把会话列、主时间线工作区、Composer 与 Inspector 作为 slot 注入，并在窄屏下通过独立 `data-conversation-mobile-header` 输出运行页自有的 `Menu / Sessions / New` 操作行、联动主导航与会话抽屉。当前实现让会话列、workspace body、chat screen 与 composer 同时挂载 `terminal-*` 与 `conversation-*` 复合 class，直接消费 Terminal 皮肤并保留 Conversation 级测试钩子；会话抽屉列表额外输出稳定 `role="list"` 语义，并把当前态 badge、标题、摘要、短 hash 与尾侧删除动作收敛到统一列表项骨架；`chat` 与 `agent-runtime` 空态都会抑制重复的 workspace summary row，把标题与说明完全交给欢迎区承接。运行页本身输出 `data-conversation-view / data-conversation-session-pane / data-conversation-workspace / data-conversation-chat-screen / data-conversation-inspector` 等稳定锚点，不再通过 bridge 或 snapshot store 回写业务状态。
- `internal/interfaces/web/frontend/src/features/shell` 继续维护主导航、共享 copy、React 管理页和 route surface。`components/PrimaryNav.tsx` 负责路由高亮、导航折叠、tooltip 与语言切换；`components/ReactManagedRouteBody.tsx` 负责把 `agent / terminal / products / memory / channels / skills / mcp / models / environments / cron-jobs / sessions / tasks / codex-accounts` 分派到各自 React 页面。`ReactManagedTerminalRouteBody.tsx` 继续保持旧版 `terminal-*` DOM class 契约作为布局皮肤基线，但会话列表、工作区容器、工作区头部与窄屏顶部操作行额外复用 `ConversationWorkspace` 的工作台语义类，确保 Terminal 与 Chat / Agent Runtime 使用同一套表面和头部节奏；Terminal 的会话抽屉列表与 Conversation 一样输出 `role="list"` 语义、当前态 badge、状态徽标、最近输出、短标识与尾侧删除动作；Terminal 的会话列表、详情轮询、step 展开、工作区关闭、列表删除与 Markdown 输出仍全部由 React state 直接维护，并在窄屏下通过 `WorkbenchContext` 输出 `Menu / Sessions / New` 顶部操作行；控制页、Sessions、Tasks、Memory 与 Products 继续复用统一客户端和共享 surface 样式。
- `internal/interfaces/web/frontend/src/app/WorkbenchContext.tsx` 与 `WorkbenchApp.tsx` 统一维护移动端运行页面板状态：主导航抽屉与运行页 `Sessions` 抽屉收敛到同一个 `mobilePanel` 状态源，`mobileNavOpen` 与 `mobileSessionPaneOpen` 只作为派生视图暴露给 `ConversationWorkspace.tsx`、`ReactManagedTerminalRouteBody.tsx` 和壳层 backdrop。这样 `Menu / Sessions` 在移动端始终互斥，切路由、点遮罩或切会话时都通过同一条关闭路径收口，不再由各运行页各自维护独立开关。
- `internal/interfaces/web/frontend/src/app/WorkbenchContext.tsx` 与 `WorkbenchApp.tsx` 统一维护移动端运行页面板状态：主导航抽屉与运行页 `Sessions` 抽屉收敛到同一个 `mobilePanel` 状态源，`mobileNavOpen` 与 `mobileSessionPaneOpen` 只作为派生视图暴露给 `ConversationWorkspace.tsx`、`ReactManagedTerminalRouteBody.tsx` 和壳层 backdrop。这样 `Menu / Sessions` 在移动端始终互斥，普通 `page-mode` 路由页新增的 `Menu` 入口也复用同一套状态切换与关闭路径，切路由、点遮罩或切会话时都通过同一条关闭链路收口，不再由各页面各自维护独立开关。
- `ReactManagedTerminalRouteBody.tsx` 的提交链路在进入 `submitInput` 时会立即设置 `submitting`，哪怕当前还需要先 `createSession()`；这样首次点击发送按钮就会同步切到 `Sending...` 禁用态，再串行完成会话创建、输入提交、active session 刷新和滚动收口，避免用户把首击感知成无效点击或在 session 创建窗口内重复提交。
- `features/shell/components/RuntimeWorkspaceFrame.tsx` 为运行页共享 `workspaceBodyRef`，`ConversationWorkspace.tsx` 与 `ReactManagedTerminalRouteBody.tsx` 在移动端通过 `ResizeObserver + VisualViewport resize` 持续测量 fixed Composer 与工作区底边的实际重叠量，并把结果写入 `--runtime-composer-inset`；`shell.css` 与 `public/legacy/chat-terminal.css` 再用这个变量收口 `.conversation-chat-screen` / `.terminal-chat-screen` 的可见高度，保证 Chat、Agent Runtime 与 Terminal 的最后一屏内容稳定停在输入区上沿。
- 上述 `--runtime-composer-inset` 同步不能只依赖键盘事件起点；当前实现会在 `VisualViewport resize/scroll`、Composer 自身 `ResizeObserver` 与 `transitionend` 上继续补测，并在下一帧和动画尾帧兜底重算，确保键盘收起或 Composer 回弹到底边后及时释放旧的底部占位，不在页面上遗留空白带。
- `ConversationWorkspace.tsx` 与 `ReactManagedTerminalRouteBody.tsx` 不会把 `--keyboard-offset` 透传给其他运行区控件：移动端 `Menu / Sessions / New` 操作行、紧凑 workspace header 和 Terminal 右侧四键定位条继续留在静态工作区流里，键盘开合期间只有 fixed Composer 自身跟随 `VisualViewport` 贴底移动。
- `ConversationWorkspace.tsx` 与 `ReactManagedTerminalRouteBody.tsx` 的发送按钮在移动端额外监听 `touchstart capture`，触发后直接进入 `sendPrompt` / `submitInput` 链路，并通过 `preventDefault()` 避免首触先被浏览器消费为 textarea blur 或键盘收起动作。
- `public/legacy/chat-terminal.css` 中的移动端 `.terminal-composer-shell` 不再声明 `transition: bottom ...`；Composer 位置完全由 `VisualViewport` 同步后的 `--keyboard-offset` 驱动，避免键盘回弹动画和 CSS 补间叠加造成卡顿。
- `shared/viewport/mobileViewport.ts` 在输入框 blur 但 `VisualViewport` 尚未恢复时，会把当前状态视为“键盘正在关闭”而不是“键盘已关闭”：沿用上一轮 `baselineHeight` 继续计算 `--keyboard-offset`，直到可视视口回到基线高度后才归零，避免 focusout 提前把 composer 和正文区闪回到底边。
- Terminal 的 `.terminal-jump-cluster` 在 `max-width: 760px` 下不再直接吃“当前遮挡量”，而是消费独立的 `--runtime-composer-rest-inset` 作为静态 Composer footprint；这样四键定位条不会跟随软键盘位移一起上移，但在键盘收起和 Composer 回弹完成后仍会稳定停在输入区上沿之上。
- Web 壳层的品牌展示统一由前端源与服务端模板共同维护：`frontend/index.html` / `static/dist/index.html` 负责 `Alter0 Chat` 页签标题，`renderLoginPage` 负责 `Alter0 Login` 与 `Alter0 Console Login`，`legacyShellCopy.ts`、`PrimaryNav.tsx` 与 `ConversationWorkspace.tsx` 负责导航品牌位、会话列标题和运行区 copy；这些展示文案调整不影响 `alter0.*` 事件名、存储 key、cookie 或元数据字段等运行契约。
- `internal/interfaces/web/frontend/src/styles/shell.css` 维护当前 React 壳层与运行页样式：桌面端使用左侧固定主导航 + 右侧主面板的两层工作台；Conversation workspace 在内部采用两列布局，`1100px` 及以下时切为带 `Menu / Sessions / New` 操作行的单主面板，并把会话列切为独立左侧抽屉；`WorkbenchApp.tsx` 会把 `chat / agent-runtime / terminal` 都直接挂到共享 `workbench-pane-shell` 下，其中 `terminal` 不再额外挂载 `route-view / route-body`，因此三条运行页在切换时保持同一 runtime workspace 外壳与滚动边界。Terminal 在窄屏下继续保持 `workbench-main -> chat-pane -> terminal-view -> terminal-chat-screen` 的闭合满高链路，由 `terminal-chat-screen` 独立承担纵向滚动；`ConversationWorkspace.tsx` 与 `ReactManagedTerminalRouteBody.tsx` 通过相同的 `terminal-* + conversation-*` 复合 class 共享会话栏、工作区容器、工作区头部、聊天滚动区、composer 和移动端操作行样式；移动端抽屉交互在真机上优先稳定而不是强调复杂 motion，当前只保留一层轻量侧滑与遮罩淡入淡出，并通过关闭 backdrop 的原生 tap highlight / focus 外观避免手机浏览器在遮罩点击关闭时出现闪烁；会话列表中的状态文本、短 hash 与删除入口采用克制的文本层级，不再使用额外胶囊装饰，`max-width: 760px` 再进一步压缩按钮与间距。共享 route surface 继续作为 `Control / Sessions / Tasks / Memory / Terminal / Codex Accounts` 等页面的统一视觉基线；`.scroll-jump-strip` 与 `.terminal-jump-cluster` 分别负责 Agent 运行态消息区和 Terminal 对话区的箭头四键；`public/legacy/*.css` 仅承载兼容内容区样式、Terminal 细节和 route body 内容皮肤。
- `internal/interfaces/web/frontend/public/legacy/chat-routes.css` 继续承载 `Agent`、`Products` 与部分 legacy route primitives 的类名皮肤，但视觉已对齐 shell 基线：`.agent-route-card / .agent-builder-form / .agent-builder-managed-item / .product-workspace-*` 统一使用近白主表面、浅灰辅助层、低对比边框和浅蓝选中态，避免 legacy 类名页面继续漂移到独立视觉体系。
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
- 根路径 `/`、`/chat`、`/login`、`/logout` 是稳定 Web Shell 入口；登录密码启用后页面和 API 共享同一登录态校验。
- `/chat` 固定分发 `static/dist/index.html`，静态资源统一从 `static/dist/assets` 与 `static/dist/legacy` 提供；兼容层仅保留 legacy CSS，不再通过 legacy JS 启动 `/chat` 运行时。
- `static/dist/index.html` 仅保留前端挂载容器、字体与 legacy 样式入口；React 在 `frontend-root` 内渲染当前 Web Shell 所需的 legacy DOM 节点，并在运行时追加 source-owned shell 样式，确保既有 `id`、`data-*` 与布局结构保持稳定。
- React 壳层中的可变状态不得清空主工作区稳定实例；涉及 route body、消息区与 runtime panel/sheet host 需通过状态边界、结构化 snapshot store 与局部 DOM 更新把 React rerender 限定在安全壳层。会话列表、消息列表与 runtime panel/sheet 原生 DOM 由 React 直接消费并渲染自身状态；`agent / terminal / products / memory / channels / skills / mcp / models / environments / cron-jobs / sessions / tasks` 十二类 route body 已由 React 托管，其中 terminal 也由 React 原生实现。
- `ChatView` 采用“滚动内容区 + 固定底部 Composer”结构：欢迎区和消息区共享主内容栅格并各自独立滚动，Composer 独占底部行，避免空态欢迎区、消息流和输入面板相互覆盖。
- `/chat` 与 `static/dist/legacy/*` 统一返回 `Cache-Control: no-cache`，保证桥接期 HTML 与固定文件名 runtime 资源总能拿到最新版本；`static/dist/assets/*` 基于 Vite 哈希文件名返回 `Cache-Control: public, max-age=31536000, immutable`。
- 开发态可通过 `ALTER0_WEB_FRONTEND_DEV_ORIGIN` 启用 Go -> Vite dev server 反向代理：`/chat` 直接转发到前端开发服务器，`/@vite/*`、`/@react-refresh`、`/src/*`、`/node_modules/*` 等运行时资源也由同一代理提供；Vite 侧再通过 `ALTER0_WEB_BACKEND_ORIGIN` 把 `/api`、登录和健康检查路径代理回 Go。
- SSE 连接只负责回传，前端断连不得取消已进入 Agent 执行链的后端任务。
- 会话标题升级、空白会话唯一性、历史折叠和页面滚动状态属于 Conversation 子域。
- `chat.js` 读取本地缓存时先归一残留 `streaming` 消息；无 `task_id` 的消息转失败态，带真实 `task_id` 的消息恢复到任务轮询链路。
- Web 流式网关把 `shareddomain.StreamEvent` 映射为 SSE：输出事件写成 `delta`，结构化步骤事件写成 `process`，最终 `done` 继续携带完整 `process_steps` 用于收口与持久化恢复。
- 流式连接异常收尾时优先保留已收到正文；只有在没有可用正文时才渲染带刷新提示的失败文案。
- 页面初始化会对当前活动会话执行一次轻量恢复：仅当本地存在流式失败态助手消息时，前端调用 `/api/sessions/{session_id}/messages`，以服务端已持久化消息覆盖本地失败态。
- Agent 执行器在运行期把工具动作与观察收敛为结构化 `process_steps`，经 `ExecutionResult -> OrchestrationResult -> session.RouteResult` 透传到 SSE `done`、Task 结果和 `/api/sessions/{session_id}/messages`。
- Agent 执行器在工具循环期间直接发出 `process` 事件，前端按 `process_step.id` 原地更新步骤状态；历史旧消息仍保留基于文本标记的回退解析。
- `chat.js` 渲染 Agent 消息时优先读取 `process_steps`；仅在缺失结构化步骤时才回退到历史 `[agent] action / observation` 文本解析，以兼容旧会话。
- 移动端输入区以 `VisualViewport` 为有效视口来源，并按聚焦、键盘和页面可见性降频刷新。
- 移动端 App Shell 高度使用 `--mobile-viewport-height`，由 `chat.js` 同步 `VisualViewport` 计算值，避免浏览器工具栏变化导致底部留白或内容裁切。
- 移动端 Chat/Agent 在输入框与会话设置底部面板之间切换时，`chat.js` 需先归一底部交互层：打开移动端设置面板前先 blur 当前输入并重新同步 `--keyboard-offset`，主输入框重新聚焦时则先关闭设置面板，再执行键盘贴底和输入框对齐逻辑。
- 桌面宽屏下 React 壳层使用 `shell.css` 中的 `--shell-reading-width=960px` 统一约束欢迎区、消息列与 Composer；legacy `chat-core.css` 继续基于主工作区可用宽度推导 `--content-width`，并让 `.message-list`、`.msg`、`.composer` 消费该宽度变量，避免消息列与输入区在不同渲染路径上出现双重宽度口径。
- React 壳层在 `shell.css` 中把 `1100px` 及以下统一视为抽屉式导航工作台：`primary-nav` 改为贴左侧视口边缘的全高抽屉，Conversation workspace 在同一断点把会话列收敛为正文上方区域，避免导航和会话列形成双浮层；`760px` 及以下继续压缩按钮和内边距，保证真手机宽度下的可触达性。
- `ConversationWorkspace` 负责运行页头部、会话列、Inspector、消息区与 Composer 的排版，`ConversationRuntimeProvider` 负责 `compact` 断点感知、SSE 收口、任务轮询和草稿恢复；Go 侧源码测试与前端组件测试共同约束这组契约。
- `chat.js` 内所有前端时间展示统一走同一北京时间格式化器，固定 `timeZone=Asia/Shanghai`、`hourCycle=h23`；时间标签输出 `HH:mm`，绝对时间输出 `YYYY-MM-DD HH:mm:ss`。
- Cron 创建表单默认时区直接复用同一前端常量 `Asia/Shanghai`，不再依赖浏览器本地时区探测。
- `agent-runtime` 会话列表前端按 `sha1(session_id)[:8]` 生成短 hash，展示在会话卡片内并通过统一 `data-copy-value` 复制链路写入剪贴板，保持与 Agent Session Profile / 预览域名使用的短标识一致。
- Markdown 渲染必须避免原始 HTML 透传；长路径、代码块和 diff 只在内容块内部滚动。
- `ChatMessageRegion.tsx` 统一负责 Conversation runtime 的消息正文与尾部元信息；已完成的 Chat 助手消息仅保留时间标签，运行中/排队/失败等瞬时状态才渲染状态胶囊，避免在每条回复后重复输出 route/source/status 胶囊。
- `shell.css` 仅在 `[data-conversation-view]` 作用域内对 Conversation runtime 头部的 `.conversation-workspace-actions .terminal-inline-button` 单独施加更小的 `min-height`、字号与水平内边距，使 `Model / Tools / MCP` 维持紧凑工作台按钮密度，同时不影响 Terminal 工作区沿用的 `terminal-inline-button` 工具栏尺寸。
- `ConversationWorkspace.tsx` 对 `chat / agent-runtime` 路由统一启用紧凑工作区头部：`terminal-workspace-head / row / copy` 与 `conversation-workspace-head / row / copy` 同时追加 `is-compact` class，仅保留会话标题与配置按钮；移动端紧凑头部分别改用 `runtimeModelShort / runtimeToolsShort` 与 `runtimeModelShort / runtimeAgent`，并继续维持与 Terminal 对齐的单行标题 + 操作按钮结构，只移除冗余摘要，不再把标题整块折掉。Composer 同步切换为 Terminal 风格的 `textarea + meta + icon submit` 结构，但仍保留 Conversation runtime 自己的 `data-conversation-*` 钩子。对应样式由 `shell.css` 与 `chat-terminal.css` 共同负责四段 grid 轨道、较小标题字号、单行截断、按钮不换行和头部内边距收敛。
- `ConversationWorkspace.tsx` 的移动端输入框继续复用 Terminal 已验证过的首次聚焦链路：`onPointerDownCapture / onTouchStartCapture` 在首次触摸时调用 `focus({ preventScroll: true })`，`useLayoutEffect` 在输入框聚焦期间监听 `window.scroll` 与 `visualViewport.resize/scroll`，把页面锚定回 `scrollY = 0`，避免首次弹出软键盘时公共操作行丢失、页面整体上移或测试环境下出现首帧分辨率跳变。
- `shell.css` 在 `.terminal-runtime-view` 作用域下继续叠加 runtime 专属视觉修正：会话卡片、Inspector、空态阅读区与 Composer 使用统一的浅色渐变 surface、圆角与阴影密度；首页 Composer 保持单一紧凑输入面，桌面与移动端分别收紧输入高度、外层留白和圆形 icon submit 尺寸，计数 meta 与 icon submit 收敛到同一工具行，而不直接照搬 Terminal footer slab 的材质。`ConversationWorkspace.tsx` 在空态为 console panel 与 chat screen 追加 `is-empty` class，`shell.css` 以 `overflow: hidden + overscroll-behavior: none` 锁住空态滚动，避免窄屏空页把头部操作行顶离可视区。
- `shell.css` 在 `@media (max-width: 1100px)` 下对工作台性能做额外收敛：关闭 `body::before/after` 光晕层，移除 `primary-nav / session-pane / chat-pane / mobile-backdrop / conversation-session-pane-backdrop / conversation-session-pane-shell / conversation-workspace-body` 的 `backdrop-filter`，把移动运行页与抽屉回落为静态浅色表面，减少真机滚动和抽屉切换时的整页合成开销。

### 验证策略

- Web handler 测试覆盖会话创建、历史隔离、流式事件和取消语义。
- 前端 E2E 覆盖 Chat、Agent、移动端输入、设置面板和长会话渲染。
- 前端组件测试需覆盖 React 工作台的稳定契约，至少校验 `WorkbenchApp` 的 hash 路由、语言切换、移动端导航收口，以及 Conversation workspace 的会话列、消息区、Composer 和 Inspector 未被回归破坏。
- `src/app/routeState.test.ts`、`src/app/WorkbenchApp.test.tsx`、`shellLayoutStyles.test.ts`、`legacyRouteLayoutStyles.test.ts` 与各 `ReactManaged*RouteBody.test.tsx` 共同覆盖路由解析、主导航状态、语言切换、Conversation runtime 入口、Agent/Terminal/Memory/Control/Tasks/Sessions/Products 页面取数与窄屏布局契约；Go 侧 `internal/interfaces/web/server_*_test.go` 继续通过源码与嵌入资产断言校验 `WorkbenchApp`、`ConversationRuntimeProvider`、`ConversationWorkspace`、`ReactManagedRouteBody`、共享样式和静态资源分发策略。
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
- 存在可用 Provider 且进入 Agent / ReAct 链路时，Agent 自身吸收 Skill、MCP、Memory、Product 与运行时上下文，只向 Codex 下发当前步骤的纯执行指令。
- 不存在 Provider、Agent 初始化失败或请求直接进入 Terminal / 直连 Codex 时，`internal/execution/infrastructure` 需要为当前会话编译原生 Codex Runtime：独立 `CODEX_HOME/config.toml`、工作区 `AGENTS.md` 与 `.alter0/codex-runtime/*`，并把启用的 MCP Server 渲染为原生 `mcp_servers.*` 配置。
- Memory Files 注入需要携带路径、存在状态、可写性、内容快照和自动召回片段。
- `internal/llm/infrastructure` 的 `openai-completions` 适配层需要把 assistant 历史消息中的 `tool_calls` 一并序列化，并与后续 `tool` 消息的 `tool_call_id` 保持稳定配对；否则 Provider 会把该轮请求判定为非法工具消息序列。
- 私有 `AGENTS.md`、私有 Skill、Agent Session Profile 分别承担协作边界、可复用打法、会话画像职责，不混写一次性任务细节。
- `search_memory`、`read_memory`、`write_memory` 只操作已解析进 `memory_context` 的记忆文件。
- Agent Memory Web 聚合接口只读返回长期记忆、天级记忆、强制上下文与说明文档；任务摘要刷新走 Task summary 子域接口。

### 验证策略

- Agent Catalog 测试覆盖内置 Agent 与用户 Agent 聚合、保留 ID 冲突。
- Execution 应用测试覆盖 Skill/MCP/Memory Context 注入。
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
- 直连 Codex 的 Chat / Agent / Product 会话在自身工作区下维护 `.alter0/codex-runtime/` 与 `.alter0/codex-runtime/codex-home/`；Terminal 会话在 `.alter0/workspaces/terminal/sessions/<terminal_session_id>/codex-home/` 下维护独立 `CODEX_HOME`。
- Terminal 会话态与 turn/step 执行态分离，历史 `running / starting` 需要兼容归一。
- Terminal 会话详情聚合 turn 摘要；step 明细按 `session_id / turn_id / step_id` 单独读取，避免会话列表一次性加载大块执行日志。
- Terminal 应用层在 Codex CLI 返回远端 compact 失败时，会把当前会话的运行线程指针复位为初始 `terminal_session_id`，保留原工作区与日志，并让后续输入自动走新线程而不是继续 resume 已失效 thread。
- Terminal 跨设备共享同一 Web 登录态下的服务端会话历史，不再按 browser client 分桶。
- `chat-terminal.css` 在真手机宽度下允许 Terminal 工作区头部切换为多行排布：标题最多两行，状态与操作工具栏按可用宽度换行，避免横向溢出。
- `WorkbenchApp` 在根壳层安装共享 `mobileViewportSync` controller，把 `VisualViewport` 变化稳定写入 `--mobile-viewport-height / --keyboard-offset`；移动端 App Shell 在键盘弹起期间保持基线高度，避免整个 workbench 被 `visualViewport` 收缩带着上移；Terminal 移动端 Composer 通过 `bottom: var(--keyboard-offset)` 贴住可见底边，而不是通过增大 footer padding 把输入区继续留在文档流里。
- Conversation runtime 的窄屏四行工作区网格仅作用于带 `data-conversation-view` 的运行页，避免 Terminal 复用共享 surface class 时被错误套用 `auto auto minmax(0, 1fr) auto` 布局，导致长历史输出把 Composer 挤出屏幕。

### 验证策略

- Task 应用测试覆盖复杂度分流、并发上限、心跳续租、来源字段和删除清理。
- Web 测试覆盖任务列表、详情、日志流、retry/cancel、产物下载/预览、Memory 任务摘要重建和会话回链。
- Terminal 应用测试覆盖创建、恢复、续写、关闭、删除、详情读取、step 明细、状态归一和工作区分配。
- E2E 测试覆盖 Terminal 移动端输入、滚动、Process 折叠和跨设备历史口径。

## Product Domain

### 包边界

- `internal/product/domain` 定义 Product、Draft、TravelGuide 等领域对象。
- `internal/product/application` 负责内置 Product、托管 Product、草稿生成、发布和 Travel 领域服务。
- `internal/interfaces/web/product_*` 负责 Product 控制面、公开入口、Workspace 和独立 HTML 页面。
- `cmd/alter0/builtin_skills.go` 提供内置 Product 相关 Skill 初始化。

### 调用链路

```text
Product request
  -> Product discovery / public product lookup
  -> master_agent_id binding
  -> Product Context injection
  -> Agent execution
  -> Product artifact / workspace space
  -> Public result / HTML page
```

### 技术约束

- Product 是一等领域对象，用户管理 Product 与内置 Product 的写权限必须区分。
- 新草稿默认采用单主 Agent，supporting agents 只作为兼容字段保留。
- 发布链路必须把主 Agent 归一到当前稳定 Agent 执行模型，并补齐工具、Skill 与 Memory Files。
- Product 总 Agent 是已发布 Product 的唯一默认执行入口。
- `main` Agent 跨 Product 调度只读取运行态允许公开的摘要，不暴露草稿、凭据或审核记录。
- `travel` 领域规则沉淀在 `travel-master` system prompt 与私有 Skill，不反向污染通用 Agent 规则。

### 验证策略

- Product domain/application 测试覆盖 Product CRUD、内置/用户来源、版本、草稿发布和冲突提示。
- Web 测试覆盖 Product 公共目录、控制面详情、Workspace、Space HTML 和 Product 消息入口。
- Travel 测试覆盖攻略创建、修订、Workspace Chat、Agent fallback 与结构化字段完整性。

## Control, Operations & Governance

### 包边界

- `internal/control` 管理 Channel、Capability、Skill、MCP、Agent Profile 和 Environment 配置。
- `internal/llm` 管理 Model Provider、上游 API type、OpenRouter 扩展与密钥状态。
- `internal/codex/domain` 负责 `auth.json` 快照、身份识别与额度状态模型；`internal/codex/application` 负责账号导入、状态刷新、独立登录会话与活动账号切换；`internal/codex/infrastructure/localfile` 负责 `<active_codex_home>/alter0-accounts` 下的账号快照、备份与登录工作目录。
- `internal/interfaces/web/frontend` 中的 `ReactManagedCodexAccountsRouteBody` 负责 Codex Accounts 控制面的运行时概览、账号卡片列表、导入/登录操作侧栏与登录会话展示，并复用 `/api/control/codex/accounts*` 控制接口完成刷新与切换；当前 live `auth.json` 未匹配托管快照时，前端需回退展示 `active.live` 快照并输出未托管提示，首次加载阶段则输出同构 skeleton 保持页面结构稳定；账号区按断点切换为多列卡片、全宽卡片区 + 双侧栏和单列卡片，避免额度与切换入口被横向滚动隐藏。
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
- Models 控制面需要保持空 API Key 语义、占位值过滤、禁用态恢复和默认 Provider 收敛。
- Environment registry 按 Web & Queue、Async Tasks、Terminal、Session Memory、Persistent Memory、LLM 模块声明 key、类型、默认值、校验规则、敏感性与生效方式。
- Environment 配置更新写入 audit store，控制面按时间倒序读取变更记录。
- Codex Accounts 服务固定解析当前活动 `CODEX_HOME`，未显式设置时回退到 `$HOME/.codex`；托管账号写入 `<active_codex_home>/alter0-accounts`，活动账号切换只替换 `<active_codex_home>/auth.json`。
- 独立登录会话通过临时 `CODEX_HOME` 执行 `codex login`，完成后再把新 `auth.json` 保存为托管账号，避免直接污染当前正在服务的运行时认证。
- LLM 运行参数 `llm_temperature`、`llm_max_tokens`、`llm_react_max_iterations` 通过 Environment 配置即时或重启后参与运行时解析，仍受 Provider 与模型能力约束。
- Runtime 重启必须由 supervisor 托管，候选实例通过 readyz 后才切换。
- 共享 Web 运行时内置通用 workspace service 注册表 `.alter0/workspace-services.json`：控制面 `PUT /api/control/workspace-services/{session_id}` 注册默认 `web` 服务，`PUT /api/control/workspace-services/{session_id}/{service_id}` 注册附加服务。`frontend_dist` 会校验 git 工作区和 `internal/interfaces/web/static/dist` 构建产物，并在 Host 命中 `<session_short_hash>.alter0.cn` 或 `<service>.<session_short_hash>.alter0.cn` 时优先分发 `/`、`/chat`、`/assets/*` 与 `/legacy/*`；`http` 服务则把请求反向代理到注册的 upstream。
- Web 登录态继续由 `server.go` 的 `authMiddleware + loginHandler` 统一管理；当请求 Host 命中主域或其预览子域时，登录 cookie 会把 `Domain` 收敛到根域 `alter0.cn`，使主域工作台与短哈希预览 host 共享同一登录会话，而不是各自维护孤立 cookie。
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
