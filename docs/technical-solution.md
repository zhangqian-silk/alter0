# Technical Solution

> Last update: 2026-04-13

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
| Control, Operations & Governance | `internal/control`、`internal/llm`、`cmd/alter0`、`scripts`、`docs/deployment` | 控制面配置、模型 Provider、运行时重启、部署凭据、测试与 TDD 约束 |

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
- `internal/interfaces/web/frontend` 负责 Web Shell 的 Vite + React 构建、legacy DOM shell 渲染、兼容桥启动和 `static/dist` 产物输出。
- `internal/interfaces/web/frontend/src/shared/api/client.ts` 负责统一 JSON 请求封装、错误收敛与登录失效回调，避免新前端页面继续散落原生 `fetch`。
- `internal/interfaces/web/frontend/src/shared/time/format.ts` 负责固定 `Asia/Shanghai` 的前端显示时区与标准时间格式，避免新旧页面时间口径漂移。
- `internal/interfaces/web/frontend/src/shared/viewport/mobileViewport.ts` 负责移动端断点、键盘偏移阈值与 viewport baseline 计算，避免 Chat、Terminal 与 route 页重复维护软键盘占位逻辑。
- `internal/interfaces/web/frontend/src/features/shell` 在桥接阶段把安全壳层状态与前端信息架构前移到 React：当前路由高亮、导航折叠态、导航 tooltip、语言感知文案，以及品牌状态卡、Session Pane 上下文卡、会话卡片列表、ChatWorkspace 头部动作区、工作区 hero、欢迎区文案、欢迎区 target picker、prompt deck、欢迎区/消息区显隐、消息列表 DOM、运行时 controls/note/sheet DOM、Session 历史空态提示/可访问标签、Composer 面板、路由页 hero、路由页头部标题/副标题和主工作区的 `page-mode / data-route / chatView / routeView` 显隐状态由 React 维护；主导航跳转、新建会话、欢迎区快捷提示、会话聚焦、会话删除、语言切换、导航折叠同步、会话历史折叠同步与移动端抽屉开合等壳层动作由 React 通过 bridge 事件发给 legacy runtime；legacy runtime 通过 chat workspace snapshot bridge 提供当前会话标题、副标题、欢迎区描述与 target picker，通过 session pane snapshot bridge 提供会话卡片列表、空态与加载错误，通过 message region snapshot bridge 提供欢迎区/消息区显隐、空态与消息 HTML 快照，通过 chat runtime snapshot bridge 提供运行时 controls、错误提示、移动端 runtime sheet 与滚动位置；`components/ReactManagedAgentRouteBody.tsx`、`components/ReactManagedTerminalRouteBody.tsx`、`components/ReactManagedMemoryRouteBody.tsx`、`components/ReactManagedControlRouteBody.tsx`、`components/ReactManagedSessionsRouteBody.tsx`、`components/ReactManagedTasksRouteBody.tsx` 与 `components/ReactManagedProductsRouteBody.tsx` 统一通过 `components/ReactManagedRouteBody.tsx` 接管 route body，其中 Agent 配置页复用 `shared/api/client.ts` 请求 `/api/control/agents`、`/api/control/skills` 与 `/api/control/mcps`，在 React 内维护选中 Agent、表单草稿、保存/删除与运行页跳转入口；Terminal 页通过 React-managed host 挂载 legacy terminal controller，使 route body 退出 `renderRoute(routeBody)` 加载链，但继续复用既有终端会话、轮询与交互逻辑；Memory 页复用同一客户端请求 `/api/agent/memory` 与 `/api/memory/tasks*`，在 React 内维护任务筛选、详情下钻、日志、产物与只读记忆文档卡片；控制台卡片页复用同一客户端请求 `/api/control/channels`、`/api/control/skills`、`/api/control/mcps`、`/api/control/llm/providers`、`/api/control/environments` 与 `/api/control/cron/jobs`，Sessions 页复用同一客户端请求 `/api/sessions` 并在 React 内维护筛选表单、查询参数与会话详情卡片，Tasks 页复用同一客户端请求 `/api/control/tasks*` 并在 React 内维护筛选表单、分页、详情抽屉、日志回放与 follow-up terminal 输入；React 还会在 `routeBody` 上输出 `data-react-managed-route` ownership 标记，供 legacy runtime 精确识别哪些页面主体已由 React 托管；其余 route body 继续由 legacy runtime 直接接管。
- `internal/interfaces/web/frontend/src/styles/shell.css` 维护 React 壳层的 source-owned 设计 token 与布局基线，覆盖桌面三栏 cockpit、移动端双抽屉、品牌状态卡、工作区 hero、prompt deck 与 Composer 面板等视觉层；`public/legacy/*.css` 继续承载 legacy runtime 内容区、Terminal 细节和 route body 内容皮肤。
- `LegacyWebShell` 需要镜像 `appShell` 上由 legacy runtime 直接维护的 transient classes，例如 `nav-open`、`panel-open`、`overlay-open` 与 `runtime-sheet-open`，确保 hash 切路由、语言切换等 React rerender 不会擦掉运行时已经打开的移动端壳层状态。
- `static/dist/legacy/chat.js` 在桥接阶段不再回写 React 已接管的 `newChatButton`、`sessionToggle`、`mobileNewChatButton`、`routeTitle`、`routeSubtitle`、`sessionEmpty` 与 `sessionList[aria-label]` 文案/属性，也不再维护主导航 tooltip、直接写入 chat workspace 头部/欢迎区文案、欢迎区 target picker、session pane 空态/错误文案、会话卡片列表、消息列表 DOM、运行时 controls/note/sheet DOM，或欢迎区/消息区显隐、切换 `info-mode`、`page-mode`、`data-route` 或 `chatView / routeView` 的显隐；`agent / terminal / products / memory / channels / skills / mcp / models / environments / cron-jobs / sessions / tasks` 已从 legacy route-body 直接接管链路退出，其中 terminal 改由 React-managed host 调用 legacy terminal mount API 完成挂载，其余页面主体完全由 React 请求 API 并渲染。兼容 runtime 仅允许根据 `routeBody[data-react-managed-route="true"]` 判断是否跳过页面主体挂载。主导航跳转、新建会话、欢迎区快捷提示、会话聚焦、会话删除、语言切换、导航折叠同步与会话历史折叠同步通过 `alter0:legacy-shell:*` bridge 事件接入 legacy runtime，chat workspace 标题、欢迎区文案与 target picker 通过 `alter0:legacy-shell:sync-chat-workspace` 回写到 React，会话卡片列表、空态与加载错误通过 `alter0:legacy-shell:sync-session-pane` 回写到 React，欢迎区/消息区显隐、空态与消息 HTML 快照通过 `alter0:legacy-shell:sync-message-region` 回写到 React，运行时 controls、错误提示、移动端 runtime sheet 与滚动位置通过 `alter0:legacy-shell:sync-chat-runtime` 回写到 React；runtime 与 welcome target picker 交互改为根节点委托，避免继续在每次重渲染后重复绑定 DOM 监听；壳层展示状态以 React 为唯一来源。
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
- `/chat` 固定分发 `static/dist/index.html`，静态资源统一从 `static/dist/assets` 与 `static/dist/legacy` 提供；仓库内不再保留额外的 legacy HTML/CSS/JS 嵌入源文件。
- `static/dist/index.html` 仅保留前端挂载容器、字体与 legacy 样式入口；React 在 `frontend-root` 内渲染当前 Web Shell 所需的 legacy DOM 节点，并在运行时追加 source-owned shell 样式，确保 `chat.js` 迁移期间仍可按原有 `id`、`data-*` 与布局结构接管页面。
- React 壳层中的可变状态不得直接驱动 legacy 挂载区重渲染；涉及 route body 与 runtime panel/sheet host 需保持稳定实例，通过 `memo`、bridge snapshot 与状态边界把 React rerender 限定在安全壳层。会话列表、消息列表与 runtime panel/sheet DOM 继续消费 legacy runtime 快照并由 React 负责最终 DOM 挂载；`agent / terminal / products / memory / channels / skills / mcp / models / environments / cron-jobs / sessions / tasks` 十二类 route body 已由 React 托管，其中 terminal 通过稳定 host 挂载 legacy controller，其余页面直接请求 API 并原生渲染。
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
- 桌面宽屏下 `chat-core.css` 使用主工作区可用宽度推导 `--content-width`，并让 `.message-list`、`.msg`、`.composer` 统一消费该宽度变量，避免消息列与输入区被硬编码 860px 二次锁死。
- `chat.js` 内所有前端时间展示统一走同一北京时间格式化器，固定 `timeZone=Asia/Shanghai`、`hourCycle=h23`；时间标签输出 `HH:mm`，绝对时间输出 `YYYY-MM-DD HH:mm:ss`。
- Cron 创建表单默认时区直接复用同一前端常量 `Asia/Shanghai`，不再依赖浏览器本地时区探测。
- `agent-runtime` 会话列表前端按 `sha1(session_id)[:8]` 生成短 hash，展示在会话卡片内并通过统一 `data-copy-value` 复制链路写入剪贴板，保持与 Agent Session Profile / 预览域名使用的短标识一致。
- Markdown 渲染必须避免原始 HTML 透传；长路径、代码块和 diff 只在内容块内部滚动。

### 验证策略

- Web handler 测试覆盖会话创建、历史隔离、流式事件和取消语义。
- 前端 E2E 覆盖 Chat、Agent、移动端输入、设置面板和长会话渲染。
- 前端组件测试需覆盖 React 渲染出的 legacy shell 契约，至少校验导航、会话列表、消息区、Composer 和 runtime sheet host 等关键节点未破坏 `chat.js` 的接管前提。
- 壳层组件测试继续覆盖 hash 路由高亮、导航折叠、导航 tooltip、Session Pane 与 ChatWorkspace 头部动作区的语言/路由文案切换，并校验品牌状态卡、Session 上下文卡、Chat hero、prompt deck、Composer 面板与 route hero 等信息架构锚点，同时验证路由跳转、新建会话、欢迎区快捷提示、会话聚焦、会话删除、语言切换、导航折叠同步、会话历史折叠同步，以及 chat workspace/session pane/message region/chat runtime bridge 能稳定收发；React 直接承接 `agent / terminal / products / memory / channels / skills / mcp / models / environments / cron-jobs / sessions / tasks` 十二类页面的取数、加载、空态、错误态、筛选状态与双语卡片渲染，其中 terminal 额外验证 React-managed host 能稳定挂载 legacy controller，其余未托管 route body / runtime host 不因 React 状态更新被清空。
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
- `codex_exec` 使用 stdin 传递执行指令，结构化上下文按需注入，不通过命令行拼接长上下文。
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
- Terminal 会话态与 turn/step 执行态分离，历史 `running / starting` 需要兼容归一。
- Terminal 会话详情聚合 turn 摘要；step 明细按 `session_id / turn_id / step_id` 单独读取，避免会话列表一次性加载大块执行日志。
- Terminal 应用层在 Codex CLI 返回远端 compact 失败时，会把当前会话的运行线程指针复位为初始 `terminal_session_id`，保留原工作区与日志，并让后续输入自动走新线程而不是继续 resume 已失效 thread。
- Terminal 跨设备共享同一 Web 登录态下的服务端会话历史，不再按 browser client 分桶。
- `chat-terminal.css` 在真手机宽度下允许 Terminal 工作区头部切换为多行排布：标题最多两行，状态与操作工具栏按可用宽度换行，避免横向溢出。

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
- `cmd/alter0` 管理启动、supervisor、重启、内置配置和运行时 metadata。
- `scripts` 承载运行账户凭据、Node/Playwright 工具链和部署初始化脚本；Node 初始化同时覆盖 `internal/interfaces/web` 与 `internal/interfaces/web/frontend`。
- `docs/deployment` 承载 Nginx 与部署权限说明。

### 调用链路

```text
Control UI / API
  -> Config / Capability service
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
- LLM 运行参数 `llm_temperature`、`llm_max_tokens`、`llm_react_max_iterations` 通过 Environment 配置即时或重启后参与运行时解析，仍受 Provider 与模型能力约束。
- Runtime 重启必须由 supervisor 托管，候选实例通过 readyz 后才切换。
- systemd 基线统一 `HOME=/var/lib/alter0`，确保 Codex、gh、git signing、Node/Playwright 工具链使用同一运行账户上下文。
- 提交签名问题不得通过关闭签名绕过。
- 技术文档、需求文档和 README 更新按领域同步，避免需求与方案分离。

### 验证策略

- Control 测试覆盖 Channel、Capability、Skill、MCP、Agent、Environment 配置持久化、Capability 审计和 Environment audit。
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
