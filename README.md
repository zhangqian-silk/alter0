# alter0

一个面向个人部署的 Skill 运行时骨架，采用 DDD 分层，强调可组合、可观察、可演进。

作为项目负责人，我把 `alter0` 定位为一套“先跑通，再扩展”的基础设施：

1. 先把消息链路打通（CLI/Web/Cron -> Orchestration -> Execution）。
2. 再把控制面补齐（Skill/Channel/Cron 配置与治理）。
3. 最后平滑演进到多执行器、多通道、多环境部署。

## Project Defaults

- Web 前端的所有时间显示统一固定为上海时间（`Asia/Shanghai`）与 24 小时制。
- 该约定覆盖现有与后续新增的前端页面、组件、管理视图和运行态界面；浏览器本地时区不参与显示格式决策。

## Why alter0

很多业务编排 项目在早期就耦合了大量能力，导致难以迭代。`alter0` 的原则是：

1. 最小闭环优先：先有可运行链路，再谈复杂能力。
2. 领域边界清晰：Gateway、Orchestration、Execution、Control 各司其职。
3. 全链路可观测：每条消息都有 trace/session/message 维度。
4. 演进友好：默认单机，无鉴权；后续可以平滑加存储、鉴权和多租户。

## Documentation

详细技术文档见 [docs](./docs/README.md)：

- [Architecture Design](./docs/architecture.md)
- [Technical Solution](./docs/technical-solution.md)
- [Domain Requirements](./docs/requirements.md)

## Output Convention

1. 所有临时产物统一写入 `output/` 目录。
2. 包含但不限于测试结果、截图、Smoke 测试记录、调试导出文件、临时脚本输出与本地排查产物。
3. 不在仓库根目录或业务目录散落创建临时文件、日志文件与一次性调试文件。
4. 需要保留的正式文档、示例数据与工程代码，仍按原有目录结构维护，不放入 `output/`。

## Architecture

系统由三条主线组成：

1. Data Plane（执行面）
- 负责处理消息通信与任务执行。
- 路径：`Channel Adapter -> UnifiedMessage -> Orchestrator -> Runtime Resolver -> CLI Runtime`。

2. Control Plane（控制面）
- 负责配置 `Channel / Skill / Model Provider / Codex Account / CronJob`。
- 通过 API 管理运行时行为，不直接绕开编排层。

3. Context Plane（上下文面）
- 负责维护会话工作区、Skill 仓库、Markdown 记忆文件与系统记忆整理任务。
- 会话内上下文压缩由实际 CLI Runtime 管理；跨会话记忆由 alter0 的 Memory System 管理。

核心链路：

1. CLI/Web/定时任务输入统一转换为 `UnifiedMessage`。
2. `IntentClassifier` 判断是命令还是自然语言。
3. 命令交由 `CommandRegistry` 与 `CommandHandler` 执行。
4. Agent 请求交由 Runtime Resolver 选择 CLI Runtime。
5. 已配置可用 Model Provider 时使用 `Claude Code + provider profile`，未配置或不可用时使用 `Codex Direct`；Claude 执行失败直接返回错误，不自动改走 Codex。
6. 定时任务由 `SchedulerManager` 触发，并复用同一编排链路。

## Chat

`alter0` 不自研多业务编排 执行框架，服务侧只负责会话、工作区、Skill、记忆与 CLI 运行时调度；具体任务执行交给成熟 CLI Runtime：

1. `Claude Code + configured provider`
- 当控制面存在启用的 Model Provider 时作为首选运行时。
- Provider 维护 `base_url / api_key / model / profile`，用于手动接入 Claude Code 可用的模型网关或供应商配置。
- 每个会话使用隔离的 Claude 运行目录和 profile 环境，避免并发会话共享全局切换状态。

2. `Codex Direct`
- 当未配置 Model Provider，或 provider 不可用、未通过健康状态时作为运行时；已选择 Claude 后执行失败直接返回错误。
- 使用当前服务运行账户的 Codex 登录态、额度与 Codex 配置。
- 每个会话维护独立 `CODEX_HOME` 与 Codex thread。

启动运行时前，服务会为当前会话工作区注入：

- `AGENTS.md` 或 `CLAUDE.md`
- 选中的 `skills/<skill_id>/SKILL.md`
- `memory/USER.md`、`memory/MEMORY.md`、`memory/daily/<date>.md` 与项目记忆
- 当前会话、工作区、仓库、产物与边界说明

## Skill And Memory

Skill 作为产品能力仓库单独维护，不绑定固定业务 Skill。代码开发、旅行攻略、前端设计、部署预览、文档协作与记忆整理都通过 Skill 文件表达规则、流程与交付要求。

Memory 与运行规则使用简单 Markdown 文件作为上下文主存：

```text
AGENTS.md
SOUL.md
memory/
  USER.md
  MEMORY.md
  daily/YYYY-MM-DD.md
  projects/<project>.md
  conversations/<conversation_id>/summary.md
```

`AGENTS.md` 保存仓库/工作区运行规则，`SOUL.md` 保存强约束；二者作为规则型上下文注入，不承载逐轮事实记录。所有持久记忆 Markdown 都由 CLI Runtime 维护。用户显式要求“记住”时，Runtime 先读取已解析记忆文件，再只更新合适目标：`USER.md` 保存稳定用户偏好，Daily Memory 保存当日活跃上下文与候选，`MEMORY.md` 保存跨会话稳定事实，项目记忆保存项目级规则与阶段性上下文。服务侧不再把会话轮次、压缩片段或任务摘要直接写入 Daily/Long-term Markdown；会话内压缩由 Claude Code 或 Codex 自身处理，会话归档只生成 `ConversationSummary`。跨会话长期记忆由系统维护任务每日定时启动同一 CLI Runtime 并加载 `memory-maintenance` Skill 进行整理。记忆维护与会话清理作为 Scheduler 内置任务随服务启动注册，不能删除，可在 `Settings > Schedules` 停用或重新启用、查看状态并手动触发。

## Repository Layout

```text
cmd/alter0                         # 程序入口（web/cli）
internal/interfaces/cli            # CLI 适配器
internal/interfaces/web            # Web 适配器 + Control API + 前端产物分发
internal/interfaces/web/frontend   # Vite + React 前端工程（构建输出到 static/dist）
internal/control/domain            # Control 领域模型（Channel/Skill）
internal/control/application       # Control 应用服务（配置增删改查）
internal/scheduler/domain          # 定时任务模型
internal/scheduler/application     # 定时任务管理器（触发到编排层）
internal/orchestration/domain      # 编排领域模型（Intent/Command）
internal/orchestration/application # 编排应用服务
internal/orchestration/infrastructure
internal/execution/domain          # 执行领域接口
internal/execution/application     # 执行应用服务
internal/execution/infrastructure  # Agent 执行器实现（示例）
internal/storage/infrastructure    # 存储适配实现（本地文件等）
internal/shared/domain             # UnifiedMessage / OrchestrationResult
internal/shared/infrastructure     # ID、日志、metrics
```

Web 前端分发采用双层缓存策略：`/chat` 与 `static/dist/legacy/*` 下的兼容样式资源保持 `no-cache`，确保页面与样式刷新及时；`static/dist/assets` 下带哈希的构建产物使用长期 immutable 缓存，减少重复下载。服务端在输出 Web Shell HTML 时会按实际 JS/CSS 内容自动为 `/assets/index-*.js|css` 注入 `?v=<content-hash>`，主服务重启、快进或 session 级预览服务刷新后，只要资产内容变化，浏览器就会请求新 URL，不依赖人工 cache-bust。

服务二进制构建统一通过 `scripts/build_alter0_service.sh` 收口：脚本会先执行 `internal/interfaces/web/frontend` 下的前端构建并校验 `static/dist/index.html` 引用了新的哈希 JS/CSS 产物，再执行 `go build` 生成服务二进制。`scripts/start_alter0_service.sh`、`scripts/relaunch_service.sh`、`make build` 与 Runtime supervisor 候选二进制构建都复用该入口，避免服务重启只拉取 Go 源码而继续嵌入旧前端产物。

当前 Web Shell 使用单一 React 工作台：左侧主导航只暴露 `Chat / Terminal / Settings` 三个稳定入口，主工作区按运行态或设置页渲染。`/chat` 是主要对话运行时入口，负责承载通用对话、代码开发、旅行、写作等由 Skill 驱动的任务；`/terminal` 暂时保留为兼容入口，同样挂载 Conversation runtime 工作区，但以 `route=terminal` 继续使用 Terminal owner 的接口、会话和本地存储。历史 `/chat` 会自动映射到 `/chat`，旧 Chat 会话会作为 Chat 会话继续展示和恢复。`/settings` 承接 Runtime、Skills、Memory 与 Schedules。

当前桌面工作台基线收敛为两层：左侧品牌导航保持全高固定栏，当前运行页的 `Sessions / New` 会话列表直接展示在同一左侧导航内；会话条目尾侧固定为三点更多按钮，展开后提供置顶、详情与删除操作，查看详情会聚焦对应会话并打开详情面板，删除需经过确认弹窗后才进入删除链路。右侧主面板承载 Conversation runtime 时间线或 Settings 内容流，`/terminal` 复用同一运行面但保持 Terminal owner。工作台采用参考 Gemini 的扁平视觉基线：主工作区、Settings frame、设置分区、表格、详情面板和空态不再依赖外层圆角、卡片边框或厚阴影，视觉层级主要通过留白、轻量分割线、选中态底色和 Composer 胶囊建立；设计图保存在 `docs/design/workbench-flat-redesign.html` 与 `docs/design/workbench-flat-redesign-*.png`。Chat、Terminal 与 Settings 顶部标题使用同一套紧凑工作台标题栏节奏：运行页显示会话标题、状态信号与 `Details`，Settings 显示同规格标题标记与路由标题，并收进同规格主面板 frame，不再使用独立大标题块、标题副文案或页面出现动效；移动端 `Menu / New` 等边缘操作使用无边框图标按钮并保留可访问文本标签。Settings 正文作为 frame 内部滚动区承载紧凑分区：`Runtime` 管理服务重启、Codex Runtime、Model 与思考深度，`Skills` 管理可注入 Skill，`Memory` 查看记忆与任务摘要，`Schedules` 管理普通定时任务、内置维护任务启停、手动触发与触发记录。旧管理子页面不再作为一级路由展示。

`Chat / Terminal` 的消息区采用轻量 IM 式消息流：用户消息右对齐并使用低对比紧凑气泡，助手消息左对齐为无边框正文阅读流，执行过程默认收敛为 `Thinking / 已思考` 内联披露入口，只显示事件数量，不显示耗时。展开外层 `Thinking` 时先展示事件列表，单个事件详情只在用户点开对应事件后出现；事件行会同步披露与过程过滤一致的类型标签、耗时和状态，例如 `Important text / Plan / Reasoning / Tools / Commands / System` 与 `Ready / Failed`；再次折叠或展开外层 `Thinking` 会收起该消息下已打开的事件详情，避免移动端历史详情重新撑开视口。Chat 现在复用 Terminal session 数据模型、状态机与提交链路，但通过 `/api/chat/sessions` 命名路由落到独立 Chat owner，与 Terminal 默认列表隔离；`/terminal` 作为兼容路径保留，由 `RuntimeRouteHost` 挂载同一套 Conversation runtime UI，但 `route="terminal"` 会继续使用 Terminal owner 的 `/api/terminal/sessions` 列表、详情、输入、置顶、删除、附件和事件明细接口。执行完成、失败或恢复时，由当前 owner 的 session `turns` 重建用户消息、assistant 正文与结构化过程。过程内容以 `turns[].runtime_trace_events` 作为前端唯一展示数据结构；事件详情通过当前 route namespace 读取，例如 Chat 使用 `/api/chat/sessions/{session_id}/turns/{turn_id}/events/{event_id}`，Terminal 使用 `/api/terminal/sessions/{session_id}/turns/{turn_id}/events/{event_id}`。旧状态文件中的 `steps / next_step_id` 会在读取时迁移并写回为 `runtime_events / next_event_id`，迁移后前端和接口继续只消费 `RuntimeTraceEvent`。事件类型只来自底层 provider、工程 adapter 或 alter0 自身确定生成的字段，不根据自然语言内容猜测。Chat 与 Terminal 兼容入口的过程事件行共用同一套 meta 渲染，事件详情共用同一套 detail block 渲染规则：说明、markdown、thinking、文本型 tool output 与错误日志进入富文本正文块，terminal、代码、diff、tool input 与 JSON 输出进入等宽内容块，并保留标题、文件名与起始行号，详情块不重复 Ready / Failed 等状态。模型与 Skill 配置面板提供过程披露勾选项，默认只显示 `important_text`，`reasoning / plan / tools / commands / system` 需要用户显式开启。最终回复统一使用稳定的运行页 markdown shell，复制动作位于正文下方，代码块独立呈现为浅灰内容块，逐条消息时间不在正文区显示。直连 Codex 的 `agent_message` 只把 `final` 或旧版无频道内容进入 assistant 正文；`commentary` 归入重要过程文本，其他非最终频道不进入最终回复或会话正文。长会话默认优先展示最新消息，顶部提供 `Load earlier messages / 加载更早消息`，滚到顶部只按批次展开本地已加载的更早历史；服务端更早历史由会话详情分页在后台自动补齐，本地窗口已无隐藏消息时，顶部滚动不再触发额外回源。右侧阅读定位条支持连续 `上一条 / 下一条` 跳转。
移动端 `Chat / Terminal` 输入区采用动态视口 + workspace grid 方案：`html / body / #frontend-root` 不再做 fixed 锁定或 `overflow: hidden` 页面锁，App Shell 使用 `height: var(--mobile-viewport-height, 100dvh)` 承接软键盘后的可见高度，并在 viewport meta 中声明 `viewport-fit=cover, interactive-widget=resizes-content`。真手机宽度下，workspace body 只有三行：顶部 workbar 占位、正文 panel、真实 Composer footer；Composer 不再 portal 到 `document.body`，不再作为 fixed bottom 浮层，也不再需要 `runtime-composer-spacer`。键盘弹起时由 `--mobile-viewport-height` 收缩整个 App Shell，grid 中间行自然变短，Composer 作为第三行随容器底边移动；正文 panel 不手写 `VisualViewport height - header - Composer` 公式，也不把键盘高度写入 padding、scroll-padding 或 spacer。运行页通过 `mobile-rest / mobile-keyboard / mobile-primary-nav-drawer / mobile-session-drawer / mobile-composer-panel / mobile-details-dialog / mobile-attachment-preview` 状态统一发布 overlay owner；主导航抽屉、会话抽屉、详情弹层或附件预览拥有视口时会释放主输入焦点，抽屉与遮罩覆盖在 Composer 之上，Composer 保持可见但不可交互。输入框首触后的键盘动画窗口不回放页面级滚动锚点；Chat 在输入聚焦且正文原本贴近底部时，仅保持 `.runtime-workspace-screen` 的底部阅读距离，避免键盘收缩后尾部内容被留在 Composer 下方。消息区触摸拖动、滚轮或滚动始终走 `.runtime-workspace-screen` 的浏览器原生 overflow 滚动路径；前端不安装 touch-scroll bridge，不阻止默认滚动，也不通过脚本写入 `scrollTop` 模拟手势滚动。正文滚动只发生在内部 `.runtime-workspace-screen` 容器内，空态、阅读定位条、命令候选与配置面板不按键盘高度重排，避免页面整体上滑、输入区先消失再回贴或 CSS bottom 与浏览器键盘动画互相叠加。
`/chat`、`/terminal`、`/settings` 与 `/login` 默认以英文文案和 `html[lang="en"]` 启动；Web Shell 内可通过语言切换入口改为中文。登录页只携带当前 canonical path 作为稳定回跳入口，不携带 query。`Chat / Terminal` 使用统一 `session_id=<8位短hash>` 表达显式会话恢复，不把完整会话 id 暴露在 URL 与页面提示中；从主导航进入 `Chat` 会清理旧 `session_id` 并默认打开当前最新 Chat 会话，进入 `/terminal` 时恢复当前最新 Terminal owner 会话。
控制类与资产类页面默认采用更高信息密度的管理视图：`Memory` 只展示长期记忆、天级记忆、AGENTS、SOUL 与说明文档这些当前运行上下文文件；`Codex Runtime` 使用单一顶部信息区展示当前服务运行账户的 Codex 身份、邮箱、计划、认证模式、profile、hourly / weekly 额度与 LLM Provider 注册状态；model / 思考深度直接做成面板内的一行 key-value 选择项并实时保存。Runtime 面板可启动 Codex device-code 登录，后端以独立 `CODEX_HOME` 运行 `codex login --device-auth`，页面展示验证链接、用户码、过期时间、轮询间隔与登录输出，并在成功后刷新当前运行时身份。Runtime 面板同时提供 Claude Code Provider Console：左侧 registry 展示已注册 Provider 的名称、base URL、默认 model、模型数量、模型列表与启用/默认状态，右侧 editor 连续注册或编辑 Provider。输入 Provider 名称、base URL、API key 与 models 后写入 OpenAI-compatible Provider 注册表；models 使用全宽多行编辑区，支持换行或逗号分隔，首个 model 作为默认模型，并作为 Claude Code 执行链可用来源。点击编辑可把 Provider 载入同一表单并通过 `PUT /api/control/llm/providers/{id}` 更新；编辑时 API key 留空表示保留已保存密钥，填写新值才替换。每次注册或更新成功后页面刷新 Provider 数量、清空 base URL / API key / models，并自动准备下一个未占用的 `Claude Code N` 默认名称，便于在同页连续注册多个 Provider。页面不再展示多账号导入、登录、切换、Account ID、User ID、CLI 路径、auth/config 路径、诊断侧栏或由 auth/config 文件存在性推导的 Ready/Status 文案。`Skills / Cron Jobs` 这组共享控制台页统一复用稳定的响应式内容网格，真窄屏下状态徽标会下沉到标题区下方、字段行改为单列展开，避免标题、徽标、复制按钮与多行字段互相挤压；`Skill` 的列表、管理表单与详情区统一使用扁平白底、浅灰说明层与低对比选中态。
所有 React 托管页面的正文型内容统一支持安全 Markdown 渲染：消息最终回复、Process 说明、Terminal 输出、Memory 文档、Task 请求/结果/日志/产物摘要、Control 描述、Cron 输入、Skill 说明、Codex 运行时说明与 Session Profile 的非等宽字段都会复用同一安全解析器。Chat / Terminal 最终输出统一通过稳定的 `MessageMarkdownShell` 承载，markdown HTML 与 `dangerouslySetInnerHTML` 对象按内容缓存，父级无关重渲染不得替换已渲染文本节点，从而保持浏览器原生文本选择与复制菜单。渲染器支持 ATX/Setext 标题、段落换行、强调、删除线、自动 URL/email 链接、列表、列表内引用与代码块、引用、图片、对齐表格、行内代码与代码块，并过滤 `javascript:` 等不安全链接；Markdown 视觉需保持正文阅读节奏，标题紧凑、段落自然，嵌套列表按 Markdown 缩进保留真实层级，普通链接显示外链箭头，代码块保留浅灰弱边界；Markdown 表格在消息容器内以真实表格结构呈现，只保留横向分割线、无外框卡片和表头灰底；短表格至少铺满消息宽度，普通长文本在单元格内自动换行，链接、URL 与代码保持不硬断开，只有真实不可断内容超宽时才在表格块内部横向滚动，不制造页面级横向滚动；ID、路径、密钥、配置值、时间戳和其他元数据字段继续按纯文本或等宽字段展示。
Chat 支持显式演示入口 `/chat?markdown_demo=1`，用于在预览环境中直接展示一条非持久化 assistant Markdown 语法覆盖样例；该参数会临时覆盖当前时间线视图但不写入会话历史。样例涵盖当前渲染器支持的 ATX/Setext 标题、段落换行、强调、删除线、自动链接、图片、引用、嵌套列表、任务项、列表内引用与代码块、分割线、代码块、对齐表格与 raw HTML 转义；表格样例覆盖短字符、长中文、长 URL/代码和混合内容场景；折叠示例中的 HTML 标签按代码块展示，折叠内容本身按普通 Markdown 展示。

前端开发态支持双向代理联调：为 Go 服务设置 `ALTER0_WEB_FRONTEND_DEV_ORIGIN=http://127.0.0.1:5173` 后，访问 `http://127.0.0.1:18088/chat` 会转到 Vite dev server；为 Vite 设置 `ALTER0_WEB_BACKEND_ORIGIN=http://127.0.0.1:18088` 后，`npm run dev` 会把 `/api`、登录与健康检查请求代理回 Go 服务。

前端交互诊断可通过 URL 参数 `?debug_clicks=1` 临时启用，也可在浏览器控制台设置 `localStorage.setItem("alter0.debug.clicks", "on")` 后刷新启用。开启后控制台会输出 `[alter0:click]` 事件链记录与 `[alter0:longtask]` 主线程长任务记录，用于定位移动端首点无效、遮罩吃点击、按钮被禁用或软键盘焦点链路问题；未开启时不注册全局事件监听。

Terminal 长输出复制通过剪贴板 API 或浏览器复制兜底完成，复制内容不会作为完整 DOM 属性写入页面，避免长日志、长命令输出在轮询和点击时额外放大页面体积。Terminal 输出正文、Markdown 正文与代码结果保持浏览器原生文本选择能力；移动端最终输出不安装脚本长按选区、假选中态、浮动复制层或编辑态兜底，长按复制由浏览器原生文本选择菜单承载。阅读定位按钮不会截获正文拖选或长按选中，用户可直接手动选中并复制输出片段。

## Built-in Commands

1. `/help`：查看命令列表
2. `/echo ...`：回显参数
3. `/time`（别名 `/now`）：输出 UTC 时间（RFC3339）

## Natural Language Handling

Agent 请求按用户交互形态以 `Chat` 为唯一前端对话运行时；`Terminal` 保留为兼容入口与后端终端代理能力：

1. `Chat`
- 面向 Web 会话消息。
- 默认绑定内置 `Alter0`（`main`），作为通用对话入口。
- Web 登录后，`Chat` 的已发送会话历史落到服务端 Session history，并在同一 Web 登录态下跨设备共享。旧 `Chat` 会话只作为历史兼容数据存在，加载时会迁移到当前 Chat 会话结构并通过 `route=chat` 读取；旧聚合文件在读取时自动重构为当前分文件布局。未发送文本草稿、附件草稿与当前页局部选择继续按浏览器隔离。
- Session history 维护 `last_active_at` 与 `pinned`：发送消息、执行完成、打开会话详情、后端 Terminal 输入与任务写回都会刷新活跃时间。系统每日自动清理超过 7 天不活跃的未置顶会话；手动置顶的会话不会被自动清理，仍有关联 queued/running 任务的会话会被保护到任务进入终态后再参与清理。会话置顶与删除在 Conversation runtime 会话列表内按当前 owner 完成；内置会话清理任务在 `Settings > Schedules` 查看、停用、启用或手动触发。
- `Chat` 新会话先使用统一占位标题 `New`，早期多轮内可按更具体输入自动升级标题。旧 `/chat?session_id=<短hash>` 入口继续恢复对应历史会话。
- `Chat` 使用 runtime workspace：会话列表由左侧主导航直接展示，置顶会话单独进入 `Pinned / 置顶` 分组并位于 `Today / 今天` 上方，其余会话再按最近时间分组；主工作区固定为主消息工作区、底部 Composer 与固定 workspace header；真实会话条目右侧固定提供三点更多按钮，展开后提供置顶、详情与删除操作，删除需二次确认。空白 `New` 草稿会话尚未进入 Session history 时只作为输入入口，不显示三点菜单，也不支持置顶、详情或删除；同一路由内只允许存在一个空白虚拟会话，重复点击 `New` 会回到现有空白草稿，首次发送后才升级为真实会话。header 统一只保留当前会话标题、状态按钮与 `Details` 入口，草稿会话的 `Details` 禁用。消息、过程步骤与最终输出都在同一轻量 IM 式消息流中推进。
- `Chat` 直连 Codex 的输出按频道收口：`final` 或旧版无频道 `agent_message` 才写入 assistant 正文与会话历史；`commentary` 进入结构化 `RuntimeTraceEvent` 的重要文本披露，`reasoning / plan / tool / command` 等类型必须由底层事件或 adapter 显式给出，不从正文猜测；非最终频道不得拼接进最终回复，避免工作进度日志污染用户可读答案。
- `Chat` 的 `Process` 阅读区在桌面与移动端都保持整列可读宽度：步骤标题与正文共享统一的缩放/换行约束，长中文句子、路径和命令说明优先在当前消息容器内换行，不允许在真机窄屏下塌缩成逐字竖排窄列；若历史或运行时过程文本混入零宽断行字符，或被异常写成“每字一行”的病态段落，前端展示层需在渲染前做归一化修正。外层 `Thinking / 已思考` 展开只打开步骤列表，不复用上一次遗留的单步详情展开态；步骤行与 Terminal 共用类型、耗时和状态 meta；所有步骤详情首帧都按最终 detail surface 渲染，终端、代码、diff 与 JSON 类输出直接进入等宽内容块，说明、markdown、thinking、tool output 与 error 直接进入富文本正文块，不先退化为普通 Markdown 文本再切换形态。
- `Chat` 的消息时间线在内容较少时仍需保持顶部收口：少量消息、短回复、折叠后的 `Thinking / 已思考` 披露行与状态标签继续贴近各自消息气泡排布，不得因为满高时间线轨道被拉伸而出现大块垂直留白。
- `Chat / Terminal` 打开已有内容的会话或切换到其他会话时，消息时间线或 Terminal 输出区默认定位到最新内容所在的底部；同一会话内发送新消息后，当前活动时间线会跟随本轮新增消息回到底部，确保用户立即看到刚发出的用户消息和助手占位回复。后续结果 patch、轮询刷新或 Process 展开不强制抢回滚动位置，保留用户阅读历史时的手动滚动状态。
- `Chat / Terminal` 的阅读定位条必须以悬浮 overlay 形式停靠在消息区右下角，不参与时间线正文排版；空白会话或少量消息场景下，消息区本身不得因为定位条占位而被额外撑高并出现无意义滚动。
- `Chat` 的会话图片资产统一落在当前 Session 工作区：用户选图后前端通过 `POST /api/sessions/{session_id}/attachments` 把原图与预览图写入 `.alter0/workspaces/sessions/<session_id>/attachments/<asset_id>/`，随后消息请求、最近会话恢复与页面重开都只保留 `asset_url / preview_url` 引用；其中 `preview_url` 仅用于输入区、列表等缩略位，消息时间线回显与预览弹层必须优先读取 `asset_url` 原图，避免再次查看时被 240px 级预览图放大。assistant 最终回复里的外链 markdown 图片也会在返回前下载进同一目录并改写成本地附件 URL，避免会话历史长期依赖远端外链或把原始大图 `data_url` 堆进浏览器存储。
- `Chat / Terminal` 首页 Composer 收敛为单一胶囊式助手输入面板：主 textarea 透明无内边框，工具栏与输入区处在同一白色 surface 内；`Chat` 工具栏只保留附件和发送等直接对话动作，不再显示 `Session` 会话设置按钮，Chat 与 Terminal 的左侧配置、附件入口使用无边框图标按钮，右侧提供紧凑 icon submit。桌面端按主阅读宽度居中，移动端压缩外层留白并保留键盘安全区，同时保持输入区足够横向留白和稳定可读高度；PC 端上传、发送、状态、详情和短标识控件保持平面化，除 Composer 胶囊外不依赖额外圆角、边框或厚阴影表达层级，输入区、底部工具栏、会话列表项和 `Details` 面板沿同一套浅色 runtime 皮肤出图，不再混用默认 terminal footer slab、Terminal 专属 note 行与旧式轻表单观感；Chat 的会话设置面板打开后，点击面板外任意区域都会立即关闭，点击主输入框时也先收起面板再继续输入；移动端在输入框已聚焦时，首触 `Session` 入口就必须直接打开面板，不允许出现先收键盘、再点第二次才能展开的状态；空态工作区不允许保留可拖拽滚动，把头部和输入区顶离可视区。
- `Chat / Terminal` 移动端主输入框固定使用不低于 16px 的输入字号，避免 iOS Safari 在重新打开浏览器后聚焦输入法时触发页面自动缩放、横向裁切或分辨率突变。
- `Chat / Terminal` 移动端输入区由 `--mobile-viewport-height` 接管软键盘后的可见高度；主输入框首触不取消默认行为，不主动 focus，不锁定或回放 page scroll。正文 panel 由 workspace grid 的中间行自然收敛到输入区上沿，正文滚动区不通过短时滚动锁接管浏览器原生键盘动画；Chat 只在用户本来贴近底部时对 viewport resize 保持底部距离。真手机宽度下共享 runtime Composer 是 workspace body 的真实 footer 行，不使用 body-level portal、fixed bottom 或 transform 合成层承载键盘位移，避免 iOS Safari 在输入框阴影层回收时留下灰色残影。
- `Chat` 的桌面端输入链路优先保证低延迟：草稿写入先更新当前输入态，再延迟落盘到浏览器草稿缓存；消息时间线、Markdown 输出与 Process 结构在仅有草稿变化时不得整段重建，避免长会话下输入时出现明显卡顿。
- `Chat` Composer 支持最多 5 张图片附件：用户可通过附件按钮选择图片，也可在 PC 输入框内直接使用 `Ctrl+V` 粘贴剪贴板图片；前端按会话草稿缓存附件、提供缩略图预览与移除操作，用户消息时间线与最近会话恢复仅保留稳定图片资产引用，不再重复持久化原始大图 payload；缩略条继续消费预览图，但再次查看、时间线回显与放大预览统一回到原图资源。助手消息中的 markdown 图片会直接以内联图片方式懒加载显示。仅支持视觉输入的模型允许发送带图消息；图片请求不会切到异步 Task，也不会在模型链失败后静默降级到 Codex 文本执行；用户显式选择 `Codex` 时，已落盘的图片原图路径会通过 Codex CLI `-i` 参数进入 `Codex Direct`，无需在提示词里额外说明图片已经存在。
- `Chat` 的左侧会话列表与消息时间线现在以 Chat owner 的 runtime session store 为准：运行页通过 `/api/chat/sessions` 和 `/api/chat/sessions/{session_id}` 恢复会话摘要、Skill 选择、附件引用、turn 历史与结构化 step；用户在会话配置面板调整 Skills 后，下一次 `POST /api/chat/sessions/{session_id}/input` 会携带过滤后的 `skill_ids`。历史会话恢复出的 Skill 选择会按当前启用且非私有的公有 Skill 目录实时收敛，已删除或禁用的 Skill 不再显示为已选，也不会进入下一次输入 payload；新增勾选的 Skill 无需刷新即可作用于下一次发送。页面初始化中的详情回填若晚于本地新发消息，或集合接口暂时只返回较短历史，不得覆盖当前未完成或更新中的本地时间线。
- `Chat` 刷新恢复采用“双层快照 + 服务端回源”策略：浏览器本地除当前活动会话外，还会保留最近会话列表的轻量快照；当服务端集合接口短暂漏掉某个刚创建或最近活跃的会话时，前端仍保留该会话在左侧列表中的可见性，并继续按 `session_id` 单独回源详情，避免刷新其他会话后新会话从列表里瞬时消失。即使集合接口已经返回当前会话的摘要项，只要本地仍残留 `Thinking...`、`Load failed`、历史 `streaming` 占位或当前会话最后一条仍是 user，运行页也会继续补拉单会话详情，并用服务端已持久化的 assistant 结果覆盖本地快照。
- `Chat / Terminal` 单会话详情默认只返回最新 `20` 个 `turns`，并按约 `256KiB` 的 turns 页预算控制单次响应体；`turns_paging` 暴露数量边界与 `byte_limit / approx_bytes`。长会话可用 `turn_limit` 与 `turn_before` 分批读取更早历史。前端访问或切换会话时先加载最新详情，再依据 `turns_paging.has_more_before` 在后台渐进补齐更早页；分页详情、后台恢复、刷新、轮询或输入返回的轻量片段都会按 turn/message id 与时间顺序合并，不会截断已加载历史。后台补齐更早历史不得扩展当前阅读窗口、强制回到底部、重建 Composer 输入态，或覆盖用户正在进行的滚动、输入与配置操作；发送后若响应只包含新 turn，已加载的旧历史仍保留在当前时间线。
- `Chat` 已接受请求后若服务端历史暂时只有最新 `user` 消息，前端继续把当前活动会话视为待恢复状态并重试单会话详情，直到 assistant 回复、任务消息或失败态落库；该中间态不会被当成完整对话停止等待。
- `Chat` 的 Web 会话执行已与浏览器请求生命周期解耦：页面刷新、请求断开或标签页短暂切走不会取消服务端已接受的会话执行；刷新后的恢复继续优先按当前 `session_id` 回源服务端详情与状态 registry，避免本轮已发出的消息因为前端断连而整轮丢失。
- `Chat` 主入口不把浏览器上次活动会话当作固定锚点：访问 `/chat` 或从主导航切回 `Chat` 时会清理旧 `session_id`，并按服务端会话列表与本地最近快照的合并结果打开最新会话。用户在会话列表中显式点选某个 Chat 会话时，URL 使用 `/chat?session_id=<8位短hash>` 精确恢复该会话；`/terminal?session_id=<8位短hash>` 恢复 Terminal owner 下的对应会话。历史 `/chat?session_id=<8位短hash>` 继续按 Chat 会话恢复对应历史会话。Settings 统一进入 `/settings`，页内切换 Runtime、Skills、Memory 与 Schedules 时不改写工作台 path。
- `Chat / Terminal` 在页面从后台回到前台、浏览器重新把当前页激活为可见工作页、bfcache 恢复或网络恢复在线时，共享 Conversation runtime page-activation 刷新链路：当前 route 只补拉对应 owner 的会话列表、当前活动会话详情和 pending 状态，避免后台期间的最新输出、标题或状态停留在旧视图。
- `Chat / Terminal` 在同一浏览器工作台内维护 24 小时 Conversation runtime 内存缓存：切到 Settings、Chat、Terminal 或其他页面后再返回时，当前 route 的未过期会话列表会先用于首屏恢复；运行态缓存保留当前已加载会话的完整消息或 turns，不裁剪历史。`Chat` 与 `Terminal` 分别使用独立 `sessionStorage / localStorage` key 保存 active session、草稿、附件草稿、过程披露过滤、完整消息快照和轻量会话信息快照；旧合并快照仅作为迁移读取来源，不再作为 Terminal 写入目标。刷新、关闭后重开或 `sessionStorage` 丢失时，前端先用当前 route 快照恢复首屏，再等待当前 owner 的会话列表与单会话详情接口回源合并。每次访问、切换、刷新或前台恢复都会刷新缓存时间；过期快照不会参与首屏渲染；服务端 Session history 仍是最终事实源。
- `Chat` 不再维护独立 Conversation Runtime registry；会话存在性、状态、置顶、turn 历史和恢复都沿用 Terminal session store。前端发送前会再次按当前 Skill 目录计算有效选择，确保历史会话里的删除项不会随旧状态重新注入。即使浏览器刷新或请求中断，服务端仍保留该会话的存在性、最近输出与恢复状态，不再把会话可见性完全交给客户端判断。
- 独立 Terminal 后端能力继续支持最多 5 个附件：图片继续提供缩略图预览与移除，常见文本/文档文件以文件条目展示；用户可通过附件按钮选择文件，也可在 PC 输入框内直接使用 `Ctrl+V` 粘贴剪贴板图片，普通文本粘贴继续保持原生输入行为。附件统一先写入 `.alter0/workspaces/sessions/<session_id>/attachments/<asset_id>/`，提交时仅发送稳定附件引用。图片继续映射为 Codex CLI `-i` 输入；普通文件会同步写入当前 Terminal 工作区 `input-attachments/<turn_id>/`，并在同轮 prompt 中注入可直接读取的 workspace 相对路径，供 Codex 按需直接读盘。Terminal turn 历史里的图片再次查看时统一优先使用原图资源，缩略位仍保留预览图。Terminal Codex CLI 远端 compact 失败时仅把当前 turn 标记失败，保留已持久化线程标识、会话历史和工作区；下一次输入继续 resume 同一运行线程。`ReactManagedTerminalRouteBody` 保留为兼容组件与测试对象，不作为当前 `/terminal` Shell 运行入口；`/terminal` 的会话列表、header、Composer、Process 与阅读定位由 Conversation runtime 以 Terminal owner 输出。
- `Terminal` 移动端的命令与 prompt 气泡保持自然整词换行：路径、flag 和短 shell 片段优先按空格或真实长单词边界断行，不允许因为窄屏收缩把 `/usr/bin/bash -lc 'ls -la'` 这类输入压成逐字或逐 token 的碎行。
- `Terminal` 输出正文区与 prompt 气泡旁不显示 turn 时间；会话列表和 `Details` 仍保留会话级更新时间，方便定位历史。
- 移动端工作台优先保证输入、抽屉和滚动流畅度：`Chat / Terminal` 的移动表面在窄屏下不再依赖大面积 `backdrop-filter` 或持续背景光晕动效，运行页容器、抽屉遮罩、抽屉面板本体和主工作区统一回落到静态浅色 surface，避免真机滚动和抽屉切换出现卡顿。
- 移动端运行页的左侧导航抽屉统一使用同一套面板开合语义：`Chat / Terminal` 都只通过 `Menu` 打开抽屉；点击遮罩、切换路由、切换会话或创建新会话后，不保留旧的抽屉覆盖层。
- 移动端运行页的左侧导航抽屉在真机上优先保证稳定性：遮罩保留淡入淡出，抽屉本体仅保留一层轻量侧滑，不再叠加容易闪烁的多层位移、淡出或条目级顺序动画；抽屉内置顶会话单独位于 `Pinned / 置顶` 分组，其余会话再按最近时间分组展示，统一收敛为标题与尾侧三点菜单，只有处理中会话显示 loading。
- `/chat`、登录页和主工作区品牌文案对外统一展示为 `Alter0`，浏览器标题、登录标题、导航品牌位、会话栏标题与欢迎区 tag 不再混用小写服务名。
- `Chat` 的会话操作、模型选择、Tools / MCP、Skills 与过程披露过滤都收敛到工作台内部；运行页输入框工具栏的 `Session` 面板内置 `Codex` 直选项，选中后后续消息会直接走 `Codex Direct`；未选 `Codex` 时按 Provider 健康状态直接走 Claude Code CLI 或 Codex CLI，不再经过内置业务编排层。`Session` 面板展示本轮可注入的公有 Skill，并允许按结构化事件类型控制过程披露内容；`Details` 只保留当前会话元信息。窄屏下主导航仍走抽屉，小高度视口中导航分组、底部设置项与语言切换入口保持独立纵向滚动并全部可触达。
- 窄屏主导航抽屉在点击路由项后会立即收起；切页后不保留覆盖在新页面上的菜单层，用户直接进入目标页内容区。
- 左侧主导航内的 Session 列表保持工作台式紧凑结构：置顶会话单独维护在 `Pinned / 置顶` 分组并固定在 `Today / 今天` 上方，其余会话按最近时间分组，并与主导航 `menu` 复用同一套分组外壳、hover、激活态语言和桌面会话列宽；条目采用独立卡片，主体只保留标题且在可用宽度内单行截断，长标题只能在条目内部省略，新增会话插入或列表刚好跨过滚动阈值时也不改变 `Sessions / New` 区块的宽度或纵向位置，尾侧保留单个三点更多按钮，展开后承载置顶、详情与删除操作，不再展示摘要、短 hash、Skill 标签、完整会话 id、状态灯或额外 footer 区块。
- Runtime 配置统一通过 workspace `Details` 面板切换，不再使用独立 bridge sheet；`Details` 默认先展示高密度摘要区，面板顶部保留标题栏与显式关闭按钮，字段标签、复制按钮和多行内容按统一紧凑规格排列，并以顶层浮层方式覆盖在运行页上方，打开时不再推动消息区或对话框位置；浮层最大可视区域保持克制，内部 tab/按钮支持再次点击只收起当前配置内容且保留 `Details` 面板，点击浮层外区域、关闭按钮或按 `Escape` 才关闭整个面板，移动端仍要求面板与输入区互不遮挡，切换时优先保证输入焦点、键盘占位和主动作可达。
- `Skill` 选项卡片在会话设置中使用短摘要展示：优先显示 Skill description，并限制在简短可扫读的卡片文案内；完整 system prompt 不直接出现在选择面板里。
- `Chat / Terminal` 继续使用同一规则生成 8 位短 hash 作为运行页 URL query、预览地址映射和排障记录引用；左侧会话列表不再展示短 hash，完整会话 id 与 `terminal_session_id` 只作为接口、持久化和工作区隔离标识。
- `Chat` 的消息请求被服务端接受后会立即把本轮 `user` 消息写入 Session history，assistant 结果在执行完成后追加；浏览器刷新、关闭或请求断开不会让已发送的用户消息只停留在前端缓存里。
- 会话设置中连续勾选 `Skill / Tool / MCP` 时，当前滚动位置需保持稳定，不能在每次勾选后跳回顶部。
- `Chat` 会话设置面板中的标题、说明与右侧标签在窄宽度下需保持可读：主标题按可用宽度截断，说明文案允许换行，避免发生重叠或互相覆盖。
- Web 前端所有时间展示统一使用北京时间（`Asia/Shanghai`）与 24 小时制；Chat、Terminal、Task、Cron 以及 Settings/Control 管理页都不再跟随浏览器本地时区漂移，Cron 表单默认时区固定为 `Asia/Shanghai`。
- 移动端 `Chat` 输入区在软键盘弹起、收起与可视视口高度变化期间，使用 `--mobile-viewport-height` 自动收缩 App Shell 高度。底部输入区作为 workspace grid footer 跟随容器底边，消息 panel 由 workspace grid 的中间行占据剩余可见高度，消息滚动区只保留固定阅读留白，不再把键盘高度叠加进底部 padding 或 scroll-padding。浏览器工具栏状态切换、输入聚焦或键盘动画不应把整个工作区重复顶起；前端不再通过滚动锚点回放或短时滚动锁接管浏览器键盘动画，浏览器工具栏伸缩或键盘收起后不保留额外底部留白。
- 移动端 `Chat / Terminal` 在页面从后台回到前台、浏览器重新激活当前标签页，或系统恢复当前 WebView 可见性时，会立刻重算共享视口诊断变量；前台恢复后的第一帧不会继续沿用后台前的旧可视高度或旧底部空白。
- 移动端 `Chat / Terminal` 的主输入框首次触摸保留浏览器原生软键盘手势，不在 `pointerdown / touchstart` 捕获阶段取消默认行为，也不主动 focus、锁定 `window` page scroll 或通过 `scrollTo` 干预真实焦点。前端不记录或回放页面级滚动锚点，程序化回焦继续用于 slash command、创建新会话后回到 Composer 等非直接输入框触摸场景。键盘开合过渡期内，App Shell 高度由 `--mobile-viewport-height` 自然变化；移动 workbar 是 workspace grid 第一行，不使用 fixed 或 `VisualViewport.offsetTop` transform，Composer 是第三行 footer，workspace header 与正文 panel 不参与整页位移，避免背景工作区参与浏览器键盘布局动画。
- 移动端 `Chat` 点击发送按钮时，会先让当前主输入框失焦，再沿原有提交流程发送当前草稿；软键盘回收阶段由 `--mobile-viewport-height` 和 workspace grid 自然回贴，避免发送后键盘停留不收或 composer 悬空。
- 移动端 `page-mode` 路由页使用动态视口高度作为可见高度；`Terminal` 与其他信息页在浏览器底部工具栏伸缩、软键盘收起或可视视口回弹后，页面底边需立即回贴动态视口，不保留额外底部空白。
- 移动端 `Terminal` 在输入框聚焦且软键盘抬起后，底部 Composer 作为 workspace grid footer 随 App Shell 的动态视口底边移动；Terminal 工作区主体、workspace header、输出区布局高度和配置浮层保持原位，键盘弹起不会把页面整体向上推出，也不会压缩长历史输出区；长历史输出继续留在 `terminal-chat-screen` 内独立滚动，不允许通过增大 footer padding 把输入区整体挤出屏幕。
- `Chat / Terminal` 统一使用同一套四键阅读定位条组件，承载 `回到顶部 / 上一条 / 下一条 / 回到底部` 四个动作，并按当前视口中的可见消息块或 Terminal turn 动态计算上下目标。Terminal 的 `上一条` 固定指向当前视口中最靠上的可见 turn，`下一条` 在单条 turn 可见时指向它后面的真实下一条、在多条 turn 同屏可见时指向最靠下的可见 turn；但只要最后一条 turn 已经进入当前视口，无论底部剩余内容是否还存在，都隐藏 `下一条`，剩余阅读交给 `回到底部`。`回到底部` 本身只在最后一条内容的底边仍位于视口外时显示；如果最后只剩容器 padding 或空白余量，不再保留伪底部跳转。这组按钮继续使用原有箭头字形，但按钮本体不参与正文文本选中或长按选中；当消息区存在有效文本选区时，四键会自动隐藏，释放完整复制操作面。
- `Chat / Terminal` 的四键阅读定位条统一使用独立圆形按钮外观与触摸反馈；移动端固定停靠在工作区右侧、输入区上沿之上，避免落回正文流内或压住输入区。
- `Chat` 与 `Terminal` 的会话设置入口统一位于底部输入框工具栏的 `Session` 按钮。新空白 Chat 会话默认勾选全部可用公有 Skill；用户可在该面板中调整 `Provider / Model`、`Tools / MCP`、`Skills`，变更会立即保存到当前会话并作用于后续发送的消息；取消全部勾选会按空选择保存，不会被旧会话配置自动补回。
- Agent 请求默认由 Claude Code CLI 或 Codex CLI 直接执行；领域规则通过会话选择的 Skill 注入。
- 选中的 Skill、MCP、Memory 摘要与工作区事实会在启动 CLI runtime 前注入当前会话工作区。
- `Models` 控制面支持同时维护 `OpenAI Compatible` 与 `OpenRouter` Provider；`OpenRouter` 可直接配置 `Site URL`、`App Name`、回退模型和 Provider 路由偏好，系统会分别注入官方请求头与请求体扩展字段。
- `OpenAI Compatible` / `OpenRouter` Provider 均支持按 `api_type` 选择上游接口：`openai-responses` 走 `/responses`，`openai-completions` 走 `/chat/completions`；配置自定义 `base_url` 时，需要目标服务兼容所选接口。`OpenRouter` 默认使用 `https://openrouter.ai/api/v1` 与 `openai-completions`。
- 启用且健康的 Provider 会生成 Claude Code provider profile；显式选择 `Codex` 或 Provider 不可用时进入 Codex Direct；Claude 执行失败不自动回退。
- `Models` 控制面保存 Provider 时，`api_key` 输入框留空表示保持现有密钥；若前端中间态传入占位值 `-`，服务端会按空值处理，不会把 `-` 持久化为真实凭据。
- 历史 `model_config.json` 若残留缺失 `api_key` 的 Provider，加载阶段会自动收敛为禁用态并保留在 `Models` 控制面中，页面不会因旧配置直接返回 500；补齐密钥后可重新启用。
- `Codex Runtime` 控制面位于 `Settings`，只管理当前服务运行账户的 Codex Direct 配置。页面在单一顶部面板中展示当前 `auth.json` 解析出的账号名、邮箱、计划、认证模式、profile、hourly / weekly 额度与 LLM Provider 注册状态；model 与思考深度的可选项来自 Codex app-server 的 `model/list`，当前生效值来自 `config/read`，选择变更后立即通过 `config/batchWrite` 写回当前用户配置。前端首屏并行读取 Codex Runtime 状态与 LLM Provider 状态，避免互不依赖的接口串行拖慢设置页加载。前端不提供多账号导入、登录、保存或切换入口，不展示 Account ID / User ID、保存名称、CLI 命令、auth/config 路径、诊断侧栏或由 auth/config 文件存在性推导的 Ready/Status 文案。
- 默认 Provider 只会落在已启用配置上；若默认 Provider 被禁用、删除或历史配置已失效，系统会自动切换到下一可用 Provider，无可用项时清空默认值。
- 复杂度评估阶段会优先复用当前消息选中的 `Provider / Model`；未显式选择时，回退到默认 Provider 与默认模型。若 Chat 当前显式选择 `Codex`，前端会改写消息 metadata 为 `alter0.execution.engine=codex`，由执行层进入 `Codex CLI` 链路；已注册的 alter0 内置命令仍优先由命令注册表执行，未注册的 `/goal` 等斜线前缀输入会原样交给 Agent 链路并按用户选择进入 Codex 或 Claude。Web 对话框在直连 Codex 且输入以 `/` 开头时会展示 Web 适用的 Codex CLI 斜线命令候选，覆盖 `/apps`、`/plugins`、`/compact`、`/diff`、`/mcp`、`/model`、`/goal`、`/status` 等命令；候选按命令作用分组顺序展示，并使用短动作说明。权限、TUI 显示、键位、剪贴板、登录退出和本地 CLI 会话管理类命令不进入 Web 候选。Terminal 在当前会话明确为 `codex` shell 时也提供同一候选补全，点击候选会补全当前命令前缀。
- 默认走实时执行。
- `Chat` 消息提交统一调用 `POST /api/chat/sessions/{session_id}/input`，前端不再接入 `/api/messages`、`/api/messages/stream`、SSE parser 或本地流式 `Thinking` 步骤；发送后当前会话按 Terminal session 状态进入 busy，并由返回或恢复到的 `turns` 重建消息区。
- `Chat` 消息区在 assistant 结果与 `Process` 展开收起期间采用逐条 patch；时间线渲染按单条消息缓存稳定 Markdown 与 Process 装配结果，避免长输出时反复重建历史消息、Markdown 与消息列表，确保导航、发送、详情和会话切换按钮保持可响应。
- `Chat` 在同一会话内继续按 `user -> assistant` 追加历史；每轮结果只允许更新当前这条尚未收口的 assistant 占位，已收口历史不得被迟到的会话详情刷新改写。
- 执行过程通过 Terminal session `turns[].runtime_trace_events` 收口；Chat 与 Terminal 前端直接消费同一组 `RuntimeTraceEvent`，并按用户选择的披露类型渲染。
- `Chat / Terminal` 会话详情首个请求只取最近 turn 页；若响应带有 `turns_paging.has_more_before`，前端会继续按 `turn_before` 在后台补齐更早历史并按 turn/message id 与时间合并。
- 请求断开或刷新后，前端优先回源当前会话详情，用服务端已持久化的最终消息覆盖本地占位态，只在恢复失败时才收敛为失败态，避免同一条 Chat 请求被浏览器重复提交。
- 若当前消息已进入 运行时执行链，前端页面切换、标签页隐藏、请求断开或浏览器主动取消请求都不会中断后端执行；最终结果仍会落到会话历史。
- 浏览器本地缓存里的历史消息若残留 `streaming` 状态，页面恢复时会自动收敛为失败态或任务态，不再把旧消息长期停留在 `In Progress`。
- 若 Chat 请求断开且本地没有可用 assistant 正文，运行页会先回源当前会话详情，用服务端已持久化的最终回复或失败状态覆盖本地 `Thinking...` / `Load failed`。只有在服务端也没有可恢复状态时，才收敛为明确提示刷新的失败文案。
- 聊天气泡支持常用 Markdown 渲染，包括标题、列表、引用、链接、表格、行内代码与代码块；原始 HTML 不直接透传，宽表格在消息内部横向滚动。助手回复采用无框正文阅读流，Markdown 按紧凑标题、自然段落、弱边界代码块和横线分隔表格呈现，避免消息区出现明显嵌套面板或厚重分割线。
- Chat 消息会标注实际回复来源，用于区分当前内容来自模型执行链还是 `Codex CLI` 执行链。
- Chat 助手最终回复提供一键复制入口；若同条消息包含 `Process`，复制内容仅包含最终正文，不包含折叠的执行细节。

2. `Skill`
- 面向“持续协助并推进执行”的目标型任务。
- Chat 使用同一个 CLI Runtime 执行任务，由 Claude Code 或 Codex CLI 承担任务推理、工具调用和会话内上下文压缩。
- Runtime Resolver 按选择结果进入执行器：显式 `Codex` 使用 `Codex Direct`；存在启用且可用的 Model Provider 时使用 `Claude Code + provider profile`；未配置或不可用时使用 `Codex Direct`；Claude 执行失败直接返回错误。
- 代码开发、旅行攻略、结构化写作等业务场景由 Skill 组合和交付契约表达，不再对应单独执行框架。代码开发复用全栈开发、测试、评审、重构、预览发布等现有 Skill；旅行攻略、前端设计、部署预览、文档协作、测试、评审与记忆整理都由 `docs/skills/<skill_id>/SKILL.md` 表达规则。
- 启动前，服务会在当前 Session 工作区注入 `CLAUDE.md` 或 `AGENTS.md`、选中 Skill、Memory 摘要、MCP 配置、仓库/附件/产物路径和可写边界。Claude Code 使用 `.alter0/claude-runtime/`，Codex Direct 使用 `.alter0/codex-runtime/` 与独立 `CODEX_HOME`。
- 代码开发任务默认在当前 Session 工作区维护独立 repo clone，并在同一会话内复用仓库、分支、预览服务和交付状态。旅行任务通过 `travel` Skill 产出移动端优先的 HTML 攻略，按行程密度生成 Codex 行程地图图片，并通过当前 Session 的只读 `travel` workspace service 暴露。
- 会话内上下文压缩由 Claude Code 或 Codex CLI 自身处理；alter0 持久化消息、日志、结果和归档摘要，用于恢复、审计和跨会话记忆整理。
- 预览短哈希 host 与主域工作台共用同一套登录保护；访问 `https://<session_short_hash>.alter0.cn` 时可直接打开该 host 自身的 `/login` 登录页，登录 cookie 会共享到 `*.alter0.cn`。主运行时的 `supervisor -> web child` 继续继承同一套 `web_login_password`，默认 `web` 全栈预览内部托管的 workspace service 子进程才会去掉第二层登录，避免主域与预览 host 各自重复登录。
- 运行时执行过程会在运行时产出结构化 `RuntimeTraceEvent`，并通过 Terminal session `turns[].runtime_trace_events`、Task 结果与历史恢复一并返回；前端按可控事件类型渲染可折叠 `Process` 区块，不再维护 Chat 与 Terminal 两套过程数据结构。
- Chat 回复中的 `Process` 与最终正文在收口后继续同时保留；刷新页面或从服务端会话历史恢复时，结构化步骤不会因最终正文已落库而丢失。
- `Chat / Terminal` 在浏览器刷新前会把当前 route 的活动会话写入独立 `sessionStorage` key，并把完整已加载消息或 turns 写入当前 route 的 24 小时 `localStorage` 缓存；同时额外写入不含消息正文的会话信息缓存，避免完整消息缓存超限时会话列表也丢失。刷新后若服务端会话列表暂时还没返回该会话，前端先用本地快照保住当前会话与消息时间线，再按 `session_id` 补拉当前 owner 的单会话详情，避免活跃会话短暂消失或被新的空白 `New` 会话顶替。
- Chat 请求一旦进入后端执行链，浏览器侧任何交互事件都不影响 Skill 本身的执行与会话持久化；断开后重新进入历史即可查看最终结果。
- Runtime Profile 保留为历史配置模型；Chat 当前稳定入口不再依赖内置 Runtime Profile 或内置业务编排 预选能力。
- Web `Chat` 独立入口与独立消息接口已移除；历史 Chat 会话仅在读取阶段迁移到 Chat 会话模型，并继续保留原目标 Skill 名称作为历史元数据。

3. `Terminal`
- 面向交互式终端会话。
- 仍属于自然语言处理，但使用独立上下文边界。
- 默认仅注入运行时必需上下文，不复用 Chat 会话记忆与长期记忆。
- Terminal 会话历史在同一 Web 登录态下对手机与 PC 共享，但每个 Terminal 会话仍使用独立工作区 `.alter0/workspaces/terminal/sessions/<terminal_session_id>`。
- Terminal 会持久化 Codex CLI 线程标识与会话状态；会话态固定为 `ready / busy / exited / interrupted`，其中 `ready` 表示当前会话可继续交互、`busy` 表示当前轮正在执行；执行细节继续由 turn/step 维度的 `running / completed / failed / interrupted` 表示。运行态退出后保留原会话历史，继续发送即可在同一会话内恢复。
- Terminal 新会话先使用与 Chat 一致的 `New` 作为占位和真实会话默认标题；首条输入后会按输入内容自动命名。自动标题在早期多轮内会按更具体的后续输入继续升级，尤其覆盖“拉取仓库 / 分析仓库”等通用开场，避免列表里长期堆积低辨识度会话。
- Terminal Composer 支持最多 5 个附件。图片附件继续保留缩略图预览、草稿缓存与 `asset_url / preview_url` 提交语义，并支持在 PC 输入框内直接粘贴剪贴板图片；常见文本/文档文件改为文件条目展示并复用同一附件上传接口，只提交稳定 `asset_url` 引用。发送时，图片会继续走 Codex CLI `-i` 输入，普通文件则写入会话工作区 `input-attachments/<turn_id>/` 并通过同轮 prompt 告知可读取路径；纯附件输入会自动补齐稳定占位文本。
- Terminal 当前活动会话的 shell 明确指向 Codex 时，输入 `/` 会在 Composer 下方弹出 Web 适用的 Codex CLI 斜线命令候选，覆盖 Apps、插件、hooks、上下文压缩、diff、记忆、skills、AGENTS.md、MCP、模型、计划、目标、后台终端、review 与 status 等命令；权限、TUI 显示、键位、剪贴板、登录退出和本地 CLI 会话管理类命令不进入候选；非 Codex shell 不显示该候选。
- Terminal 的 `Details` 面板支持选择控制面中启用且非私有的公有 Skill，选择结果随下一次 `/api/terminal/sessions/{id}/input` 请求以 `skill_ids` 发送；当前服务内置公有 Skill 除 `memory` 外，还包含 `preview-publish`、`frontend-design`、`doc-coauthoring`、`fullstack-developer`、`code-reviewer`、`webapp-testing`、`find-skills`、`test-driven-development`、`ui-ux-pro-max`、`code-simplifier`、`code-review`、`brainstorming` 与 `travel`。新 Terminal 会话首次加载 Skill 列表时，默认勾选这批公有 Skill 中除 `memory` 外的全部可用项，用户后续仍可按会话手动调整。后端会把选中的 Skill 编译到当前 Terminal 工作区的 `.alter0/codex-runtime/skills.md` 和托管 `AGENTS.md` 指令块中，仅作用于后续 Terminal 输入；托管运行时说明会同时要求 Codex 仅在当前 Terminal 工作区及其派生文件内执行，不得改动其他会话、服务或工作区外仓库，除非当前输入明确把这些目标列为本轮范围。
- 同一 Terminal 会话在单次运行态中断或退出后，只记录一条对应状态提醒；恢复后若再次发生新的中断或退出，再按新的状态周期补充提醒。
- Terminal 输入区上缘的运行态 hint 只服务于当前空闲会话；一旦用户重新发送恢复当前会话，或从旧会话切到 `New` 待创建态，旧的 `Exited / Interrupted / Failed` 提示会立即清空，不再在发送中残留。
- Terminal 工作区头部仅保留 `Details` 等阅读辅助工具；会话删除统一从左侧会话列表触发，`Delete` 会移除会话记录、持久化状态文件与该会话对应的独立工作区。删除成功后，无论删除的是历史会话还是当前活动会话，当前会话列表面板都保持现状不自动收起，便于连续清理多条会话；之后用户点 `Menu` 或点击抽屉外部遮罩时，列表仍会正常关闭。前端同时会在后续运行态轮询和 page-activation 补拉中继续屏蔽该会话，避免服务端短暂返回旧列表时已删除项又回弹到左侧列表。
- Terminal 左侧会话列表复用共享会话列表项，标题与尾侧详情、删除按钮在长标题与中英文混排场景下保持统一行距和尾侧对齐；`busy` 会话标题旁显示 loading，其他状态不显示行内状态。
- 当前正在查看旧会话时点击 `New`，前端会先切入一个干净的待创建会话态；创建请求完成前不再沿用旧会话的 `Interrupted / Exited / Failed` 提示文案，也不继承旧会话残留的底部键盘留白。
- 同一 Web 登录态下，手机与 PC 访问同一批 Terminal 会话历史；刷新或跨端切换后不再因设备标识不同而看到不同会话列表。
- Terminal 不再设置产品级会话数量上限或固定超时淘汰策略。
- 访问 Terminal 时，轮询刷新不会重建已聚焦输入框；移动端输入法每次确认词句后，若输入框仍保持聚焦，页面继续延迟重绘并保持当前位置，直到失焦后再刷新视图；桌面端在连续输入窗口内也会暂缓非必要工作区重绘，待输入停顿后自动补齐刷新。
- Terminal 轮询刷新采用会话列表与工作区局部更新；当用户正在滚动输出区时，前端保留原消息滚动容器与滚动位置，不再按周期整块重建终端视图。
- Terminal 刷新按会话状态自适配：`busy` 会话继续保留实时刷新，但用户正在滚动输出区时暂停明细刷新；`ready` 会话不再维持周期轮询，改由页面重新可见或重新获得焦点时补拉列表与当前会话详情，避免空闲页面持续耗电。
- Terminal 滚动状态同步会合并到浏览器逐帧刷新节奏内执行；上一条 / 下一条定位所需的 turn 位置在视图结构稳定时复用缓存，仅在 turn 列表、折叠态或布局尺寸变化后重测，避免在连续滑动中反复全量测量消息区。
- Terminal 浏览器侧会话缓存采用滚动感知的延后持久化；输出持续增长时优先让出主线程给滚动与渲染，再在滚动停顿后写入本地存储。
- Terminal 会按页面可见性、输入聚焦与滚动活跃度自动调整刷新节奏；`busy` 会话在活跃阅读或输入期间降频，页面隐藏后进一步延长刷新周期；`ready` 会话停止周期刷新，恢复前台后通过共享 page-activation 链路立即补拉当前列表与活动会话。
- Chat 与 Terminal 的消息流统一采用克制的冷灰工作台阅读主题：用户消息保留右对齐并使用浅灰低对比紧凑气泡，助手消息左对齐并弱化为无边框阅读块；Chat 正文工作区白底无框，靠阅读宽度、留白和角色对齐建立层级，不在消息区叠加明显边框、背景分界或卡片容器。思考过程只显示一行 `Thinking / 已思考` 披露入口，详情展开后再承载步骤内容；正文排版优先于装饰层级，用户消息与其后续回复继续保持更紧凑的同轮分组间距，长历史按最新优先和顶部渐进加载处理。后续新增运行页应复用同一 `runtime-message-*` 消息格式，不再单独定义气泡样式。
- Chat 与 Terminal 的消息正文区不显示逐条时间；仅在回复仍处于生成、排队或失败等需要即时反馈的状态下显示紧凑状态标签，不再为已完成消息追加 route/source/status 标签。
- Chat 工作区头部固定为共享单行 header：只保留会话标题、状态按钮和 `Details` 入口，不再在头部直接放置 `Model / Tools / MCP / Skill` 控件，也不重复展示运行页标签与目标摘要；运行时配置统一通过底部输入框工具栏的 `Session` 按钮进入，`Details` 面板只展示会话元信息。独立 Chat 的 `Deliverables`、`Session Profile` 与独立 Skill 面板已移除，Skill 配置统一进入 `Session` 面板。
- Chat 桌面宽屏会按可用主工作区宽度自适应扩展消息列与底部输入区，并统一收敛到居中的 `960px` 阅读宽度上限，避免正文无限拉长。
- Chat 移动端输入区默认隐藏装饰性附注与字数计数，底部保留输入框、`Session`、附件与发送按钮；运行态配置统一通过输入框工具栏的 `Session` 按钮进入，`Details` 面板只展示会话元信息。
- Chat 与 Terminal 在移动端都会由 workspace grid 自动把消息 panel 收敛到 Composer 上沿；输入区贴底期间，消息列表、空态说明和长输出阅读都不得被底部输入框盖住。软键盘作为 overlay 覆盖 layout viewport 时，消息滚动区不扩大底部 padding、scroll-padding、workspace footer 或 spacer。
- Chat 与 Terminal 在软键盘收起、输入框失焦和 composer 回弹到底边的过程中，消息视口与跳转控件也要同步释放旧的底部占位；页面上不能残留上一轮键盘高度对应的空白带或悬空控件。
- Chat 与 Terminal 在移动端软键盘弹起期间，底部 Composer 必须始终位于运行页跳转控件之上；右侧四键定位条和消息阅读定位按钮在主输入框聚焦后需主动隐藏，待输入框失焦、键盘收起后再恢复，不能压到输入框、附件条或键盘上方。
- Chat 与 Terminal 在移动端打开主导航抽屉时会先释放当前输入焦点并收起软键盘；抽屉与遮罩通过顶层 overlay 层覆盖运行页，层级高于 Composer。底部 Composer 保持可见但进入不可交互态，不再通过 `visibility: hidden` 或卸载 DOM 修正层级。
- Chat 与 Terminal 的主输入框在移动端按普通命令文本输入处理：运行页必须关闭系统自动填充、卡片、地址与密码类辅助条，避免 iOS 在键盘上沿额外挂出输入助手并露出底部残留页面层。
- Chat 与 Terminal 在移动端键盘弹起和收回期间，App Shell 使用 `--mobile-viewport-height` 收缩到可见高度，Composer 作为 workspace grid footer 自然贴住动态视口底部；顶部操作行、紧凑 workspace header 和正文 panel 不跟着键盘位移一起跳动；阅读定位条在输入框聚焦期间隐藏，失焦后再回到原有停靠位。
- Chat 与 Terminal 的移动端发送按钮在软键盘打开期间支持直接点按提交；发送动作需同时覆盖 `pointerdown(touch)` 与 `touchstart` 首触链路，并在同一次触摸内去重，避免先收起键盘、丢失焦点或要求第二次点击后才真正提交。
- 运行页 composer 跟随键盘回弹时不再叠加额外的 `bottom` 补间过渡；可视视口收起后输入区会直接回贴底边，避免回弹拖滞或明显卡顿。
- 输入框失焦后，页面依赖 `--mobile-viewport-height` 的实际回弹恢复高度，不额外保留键盘偏移，不会先闪回到底边再被下一帧重新顶起。
- Web Shell 主工作区的首屏内容保持紧凑起始区：桌面与中宽度下的 `Chat` 空态欢迎区会在 header 与 composer 之间沿主工作区中轴做竖向居中，欢迎 tag、标题、描述与 prompt 统一围绕欢迎区中线排布；真窄屏继续贴近头部下沿起排，输入区不再依赖大块弹性留白把首屏内容拉散。普通 `page-mode` 路由页 `route-head` 与 `Terminal` 工作区继续沿用“两行头部 + 贴顶正文起始区”基线，而 `Chat` 与 `Chat` 空态在中窄屏都复用 terminal-style 顶部操作行与单行紧凑 workspace header：顶部统一显示 `Menu / New`，workspace header 单行显示当前会话标题、状态按钮与 `Details` 入口，不再展示冗余摘要文案。
- Terminal 路由直接进入工作区，不再在工作区上方额外挂载页面级说明 hero；运行区根节点直接挂在共享 `workbench-pane-shell` 下，不再额外经过 `route-view / route-body` 包裹，从 `Chat` 切到 `Terminal` 时保持相同的 runtime workspace 骨架，避免首屏布局与滚动容器发生跳变。会话栏、工作区容器、工作区头部和窄屏顶部操作行与 `Chat` 复用同一套工作台表面语义与节奏；Terminal 只在终端路由内补充会话状态、jump controls、step 详情和 composer 皮肤等变体。顶部通用栏直接提供 `Menu / New`，工作区头部收敛为会话标题、状态按钮与 `Details` 工具栏；`Details` 首屏同样先展示紧凑摘要栅格，再承接终端会话字段；发送区使用自适应输入框与紧凑发送按钮，运行态退出或中断提示以内嵌状态条贴合输入区上缘展示；消息与 `Process` 头部保持自然文档流滚动，不再启用吸顶导航；阅读定位由右侧平面四键组承担，支持回到顶部、上一条、下一条与回到底部，并统一为浅色低对比滚动条与阅读主题；软键盘收起或浏览器底部工具栏回弹后，底部输入条需立即回贴可见底边，不保留额外占位空白。
- Terminal 移动端的右侧四键定位条只按静态 composer footprint 停靠，不跟随软键盘位移一起上移；键盘弹起时按钮组保持原位，键盘收起后继续稳定回到输入区上沿之上，不在底边残留半截控件。
- Terminal 发送按钮在首次点击时必须立即进入 `Sending...` 禁用态；若当前还没有 terminal session，前端先创建会话再继续提交输入，但首击期间不能保留可重复点击的静止按钮，避免用户误判为“第一次点击无效”。
- Terminal 在窄屏 `page-mode` 下继续由 `terminal-chat-screen` 独立承担消息区纵向滚动；外层 `workbench-main / chat-pane / terminal-view` 只负责提供满高约束与滚动隔离，不得吞掉消息页滚动手势。
- Terminal 移动端通过主 `Menu` 打开左侧主导航抽屉；顶部操作行不再提供单独 `Sessions` 按钮。
- Terminal 的左侧会话列表与 Chat 共享同一套列表视觉语义：条目主体只展示标题，详情与删除入口收在列表项尾侧，处理中条目在标题旁显示 loading；会话运行状态保留在 workspace header 与 `Details` 摘要中，不在列表项内额外渲染状态灯、元信息或短标识。
- Terminal 窄屏工作区头部不再重复渲染第二枚内部 `Sessions` 按钮；会话列表入口统一由主导航 `Menu` 承接，工作区操作栏只保留状态按钮与 `Details` 等当前会话阅读动作。
- Terminal 工作区头部在真手机宽度下允许标题、状态按钮与 `Details` 工具栏自适应换行；长标题优先保留可读性，不允许把操作按钮挤出可见宽度。
- Terminal `Process` 步骤列表保持单行摘要阅读：步骤标题在可用宽度内自动截断，时间与状态固定收在右侧独立区域，不与标题文本重叠。
- Terminal `Process` 的步骤头固定遵守“展开图标 / 标题 / 耗时与状态”三列结构；展开图标必须占用独立窄列，标题始终落在中间主阅读列，不允许因为节点缺失或列错位把中文标题挤进图标列，导致移动端只剩一个字符可见。
- Chat 与 Terminal `Process` 展开后的自然语言步骤详情必须按正文块渲染：`reasoning / plan / message / text` 等说明类内容沿用运行页 markdown 归一化与整列换行约束，异常零宽断行字符和“每字一行”病态段落需在展示前修正；终端输出、diff 和代码类块继续保留预格式化等宽显示。带详情接口的步骤只在用户展开具体步骤时按 `session_id / turn_id / event_id` 懒加载完整 detail，避免首屏提前返回大段 thinking 内容。
- Terminal 中四键导航会跟随当前视口里的可见 turn 重新计算上下目标：`上一条` 取最上可见 turn，`下一条` 在单条可见时取真实下一条、在多条可见时取最下可见 turn；`Process` 折叠或展开后，上一条与下一条状态会随重排结果同步更新。
- Terminal 的整体视觉语言与 Chat 收敛：左侧会话列表、工作区容器、输入区与输出块统一使用扁平白底、必要分割线、有限强调色和 Composer 胶囊；用户输入使用共享右侧紧凑气泡，不再额外展示命令前缀符号或强调色，最终回复直接按共享助手消息阅读流渲染；`Process` 保留低对比提示区和左侧纵向时间线，内部步骤压缩为单行摘要，展开后只展示详细内容，不再重复状态标签。
- Terminal 桌面端会话列表直接显示在左侧主导航内，主工作区头部收敛为会话标题、状态按钮与 `Details` 工具栏；这套布局由 React 组件直接渲染，不再依赖旧版脚本控制。
- 同一轮 Terminal 最终输出出现后，前端会自动折叠对应 `Process` 面板，把阅读焦点收敛到输出正文；用户手动再次展开后保留该选择。
- Terminal 最终输出正文提供一键复制入口，复制内容仅包含最终回复，不包含 `Process` 步骤细节。
- 移动端下 `Process` 头部与步骤行默认保持单行信息结构，`Process` 标签、步骤摘要、耗时与状态在可用宽度内同排阅读；超长摘要按单行截断，不再把每条消息挤成上下两行。
- Terminal 最终输出中的 Markdown 链接按链接文本渲染为可点击链接，不再直接暴露整段 Markdown 源码与长路径。
- Terminal 与 Chat 的长路径、超长单词、代码块和 diff 仅允许在内容块内部换行或横向滚动，不再把外层消息卡片、聊天框、顶部操作行或底部输入区撑出移动端视口。
- 键盘弹起后输入区继续贴底可输入，页面不会因刷新回到顶部；浏览器底部工具栏伸缩、软键盘收起或视口回弹后，Terminal 工作区底边与输入条会立即回贴可见视口，不保留残余底部空白。

补充说明：

1. `Chat / Terminal` 只要落到 `Codex CLI` 执行链，都要求服务运行账户本身具备可用的 Codex / OpenAI 认证。
2. 若服务账户缺少认证，Web 端会快速返回认证失败，而不会长时间保持等待态。
3. 若服务需要在仓库内执行 `git commit`、`git push`、`gh pr create`、`gh pr merge` 等交付动作，部署时还需为运行账户补齐 GitHub App 凭证、`gh` 包装器与 SSH 提交签名配置；仓库内提供 `scripts/setup_alter0_runtime_auth.sh` 作为一次性初始化脚本。

## Travel Skill

`travel` 是系统内置 Skill，不再对应内置业务编排。

1. 旅游领域的稳定规则沉淀在统一 Skill 仓库中的 `docs/skills/travel/SKILL.md`，用于约束城市行程、地铁、美食、路线卡、地图图片与 HTML 攻略输出结构。
2. 用户在 Chat 或 Terminal 中选择 `travel` Skill 后，当前会话仍直接由 Claude Code CLI 或 Codex CLI 执行；Skill 只提供领域规则，不创建额外 Skill 编排层。
3. 旅游任务在正常对话之外，还要求额外生成一份 HTML 格式的旅游攻略，并把它发布到当前 Session 的公开只读子域名 `https://travel-<session_short_hash>.alter0.cn`。只有当前请求已经在当前 Session 工作区根目录生成或更新了对应的 `index.html` 时，才允许把这份静态攻略作为 `travel` 服务发布；缺图、断链或未发布图片均视为攻略未完成。

## Workspace Model

默认运行策略保持 `danger-full-access`，当前默认执行目录策略统一为“各执行会话独立工作区”：

1. `Chat`
- 默认执行目录：`.alter0/workspaces/sessions/<session_id>`
- `Chat` 与 `Skill` 会话历史可继续按各自会话维度回放；删除会话时会同步清理对应会话工作区

2. `Terminal`
- 终端会话工作区：`.alter0/workspaces/terminal/sessions/<terminal_session_id>`
- 终端会话状态：`.alter0/state/terminal/sessions/<terminal_session_id>.json`
- 同一 Web 登录态下，手机与 PC 访问同一批 Terminal 会话历史，不再按浏览器设备标识分桶

说明：

1. 工作区目录仅决定默认执行目录与运行时产物落点，不等同于文件系统权限收缩。
2. 当前默认仍为 `danger-full-access`，因此是否可访问其他绝对路径，仍取决于宿主机环境与运行账户权限。
3. 具体执行统一交给 CLI Runtime，默认执行目录会落到当前 Chat / Task / Terminal 会话各自的独立工作区。
4. Chat / 运行时执行链按选择结果使用 `Claude Code + provider profile` 或 `Codex Direct`；两条路径都会在当前会话工作区准备运行时 home、Skill、Memory、MCP 和边界文件。Chat 按 Session 持久化 CLI runtime 会话状态用于续聊；Chat 按北京时间 05:00 归档日持久化运行时状态，新归档日自动开启新的会话线程。

其中 `Chat` 再细分为两种执行方式：

1. `Sync`
- `POST /api/chat/sessions/{session_id}/input`：Chat owner 输入提交入口；`POST /api/terminal/sessions/{session_id}/input`：Terminal owner 输入提交入口；两者返回同一 runtime session 视图。
- Chat 前端不再调用 `/api/messages` 或 `/api/messages/stream`，不再依赖 SSE 增量、保活帧或浏览器读流状态驱动 Chat UI。
- 对已进入运行时执行链的输入，请求断开只会终止当前 HTTP 回传，不会取消后端执行；最终结果仍通过 Terminal session 详情恢复。
- 同一会话内的同步请求保持串行；当上一条同步执行尚未结束时，后续用户消息会继续等待并按序执行，不再因为默认队列等待时间直接返回 5 秒超时。
- 对于仍在执行中的同步请求，同会话后续用户补充会按顺序排队，并在下一轮 CLI runtime 输入中继续推进。

Web `Chat` 独立消息链路已移除。对话消息统一使用 Chat owner 的 runtime session 输入接口；历史 Chat 会话在读取阶段兼容迁移为当前 Chat 消息结构，运行页列表与详情由 `/api/chat/sessions` 系列接口恢复，并与 Terminal 默认 session 列表分开存储。

## Observability

1. 结构化日志（JSON）
2. `/metrics`：Prometheus 文本格式指标
3. `/healthz`：活性检查
4. `/readyz`：就绪检查
5. 关键字段：`trace_id`、`session_id`、`message_id`、`route`

## Quick Start

### Prerequisite

```bash
go version
```

建议 Go `1.25+`。

### Run Runtime

构建生产服务二进制：

```bash
make build
# or
scripts/build_alter0_service.sh
```

该构建入口会先生成前端 `static/dist`，再构建 Go 服务。若只做本地开发态联调，可继续使用 `make run` 或 `go run`，并按需配合 Vite dev server。

```bash
make
# or
make run
# custom port
make run WEB_ADDR=127.0.0.1:<your-port>
# or
go run ./cmd/alter0
# or
go run ./cmd/alter0 -web-addr 127.0.0.1:<your-port>
```

运行时默认行为：

1. 同时启动 Web 与 CLI 两个输入通道。
2. Web 地址默认 `127.0.0.1:18088`，可通过 `-web-addr` 参数覆盖。
3. 如果使用自定义端口，后续示例中的 URL 也需同步替换端口。
4. 默认以 `supervisor -> child runtime` 两层进程启动：父进程负责托管运行中的子进程，处理 Web 控制台发起的重启、构建、探活与切换。
5. 存储后端默认本地文件（目录 `.alter0`）。
6. 存储格式按业务场景选择：Control 配置使用 `json`，Scheduler 状态使用 `json`。

### Runtime Restart

`Settings > Runtime` 中的“重启服务”会走运行时托管链路，而不是由当前业务进程直接自拉起：

1. 点击“重启服务”后会打开站内确认弹窗；“同步远端 master 最新改动”作为弹窗内勾选项展示，默认勾选。
2. `sync_remote_master=false`：基于当前仓库状态调用统一构建入口，先重建前端 `static/dist`，再构建候选二进制，并由 `supervisor` 完成子进程切换。
3. `sync_remote_master=true`：先校验当前分支为 `master`；无 Git 已跟踪本地改动时直接执行同步重启；若后端检测到 Git 已跟踪本地改动，会以结构化错误要求前端进入二次确认。只有用户二次确认并传入 `confirm_discard_tracked_changes=true` 时才会丢弃这些改动，否则不会清理本地工作区内容。确认后执行 `git fetch --prune origin master` 与 `git merge --ff-only FETCH_HEAD`，随后通过统一构建入口先重建前端 `static/dist`、再构建候选二进制并切换。
4. 候选版本只有在 `/readyz` 探活通过后才会成为当前运行版本；若启动失败，会自动恢复上一运行版本。
5. Git 或构建失败会直接返回到 Web 控制台，便于定位权限、凭据、快进合并失败等问题。
6. Runtime 面板会展示当前在线实例的最近启动时间与对应 `commit hash`，用于确认上次成功重启切换到的运行版本。
7. 重启完成后页面会自动刷新到新实例，并以站内成功弹窗提示用户当前页面已连接到最新运行实例。

### Public Deployment Baseline

公网部署建议使用 Nginx 反向代理，并开启应用内登录页：

```bash
export ALTER0_WEB_LOGIN_PASSWORD='请替换为强密码'
export HOME=/var/lib/alter0

go run ./cmd/alter0 \
  -web-addr 127.0.0.1:18088 \
  -web-bind-localhost-only=true \
  -web-login-password "$ALTER0_WEB_LOGIN_PASSWORD"
```

若通过 `systemd` 运行，建议在服务环境中显式设置 `HOME=/var/lib/alter0`；启动脚本也会把历史 `HOME=/var/lib/alter0/codex-home` 归一到 `/var/lib/alter0`，确保 Codex 认证与运行态数据落在统一运行根目录。

`Codex Runtime` 固定读取当前活动 `CODEX_HOME` 下的 `auth.json` 与 `config.toml`；未显式设置 `CODEX_HOME` 时，对应目录即 `$HOME/.codex/`。Runtime 页面通过 Codex app-server 读取真实运行时能力与配置来源，并通过用户配置写接口更新当前活动配置中的 `model` 与 `model_reasoning_effort`。

若服务需要自行提交签名 commit、创建 PR 或执行合并，还需在 root 下额外执行一次：

```bash
sudo ./scripts/setup_alter0_runtime_auth.sh
```

该脚本会把 `alter0` 运行账户的 GitHub App token helper、`gh` 命令包装器、SSH signing key 与全局 Git 配置初始化到 `/var/lib/alter0`，用于服务内 `Codex CLI` 的提交 / PR / merge 链路。

若希望服务内 `Codex CLI` 可直接执行 `internal/interfaces/web/frontend` 下的 `npm run build` / `npm run test`，以及 `internal/interfaces/web` 下的 `npm run test:e2e`、`npx playwright install chromium` 等 Node/Playwright 测试链路，还需在 root 下额外执行一次：

```bash
sudo ./scripts/setup_alter0_runtime_node.sh
```

该脚本会把带 `npm`/`npx`/`corepack` 的 Node 运行时安装到 `/var/lib/alter0/.local`，并默认在 `internal/interfaces/web` 与 `internal/interfaces/web/frontend` 目录预装 `npm ci`，随后安装 Playwright Chromium 浏览器，使服务运行账户在非交互式环境中也能同时执行前端构建、单测与 E2E 测试。正式服务启动与重启使用 `scripts/build_alter0_service.sh`，因此运行账户需要能在 `PATH` 中找到这套 Node 工具链。

之所以默认落在 `/var/lib/alter0/.local`，是因为这里属于 `alter0` 服务运行账户自己的运行时目录：既不会污染系统全局 `/usr/local/bin`，也不依赖宿主机预装 `npm`。脚本会把实际安装目录中的 `node`、`npm`、`npx`、`corepack` 软链接到 `/var/lib/alter0/.local/bin`，再由服务启动时补齐该目录到 `PATH`，这样 `Codex CLI`、Web 子进程和手工切到 `alter0` 账户执行时看到的都是同一套稳定工具链。

新服务启用时，建议直接按下面顺序执行：

```bash
# 1. 准备运行环境
sudo install -d -m 750 /etc/alter0
sudo sh -c "printf 'ALTER0_WEB_LOGIN_PASSWORD=请替换为强密码\nALTER0_RUN_AS=alter0\nALTER0_RUNTIME_ROOT=/var/lib/alter0\nHOME=/var/lib/alter0\n' > /etc/alter0/alter0.env"
sudo chmod 600 /etc/alter0/alter0.env

# 2. 确保公共路径可见 codex / node / gh
which /usr/local/bin/codex
which /usr/local/bin/node || which node
which /usr/bin/gh

# 3. 准备 GitHub App token 生成器
sudo test -x /usr/local/bin/github-app-token
sudo test -f /etc/github-app/config.json

# 4. 初始化 alter0 运行账户的 git / gh / ssh signing
sudo ./scripts/setup_alter0_runtime_auth.sh

# 5. 初始化 alter0 运行账户的 node / npm / playwright
sudo ./scripts/setup_alter0_runtime_node.sh

# 6. 重启服务
sudo systemctl restart alter0.service
```

若不是默认部署路径，可在执行初始化脚本前覆写这些变量：

```bash
sudo ALTER0_RUN_AS=myservice \
  ALTER0_RUNTIME_ROOT=/data/myservice \
  ALTER0_HOME=/data/myservice \
  ALTER0_REPO_DIR=/srv/myservice/app \
  ALTER0_GIT_USER_NAME='my-bot[bot]' \
  ALTER0_GIT_USER_EMAIL='123456+my-bot[bot]@users.noreply.github.com' \
  ./scripts/setup_alter0_runtime_auth.sh
```

初始化完成后，建议至少做一次快速验证：

```bash
sudo -u alter0 env HOME=/var/lib/alter0 PATH=/var/lib/alter0/.local/bin:/usr/local/bin:/usr/bin:/bin gh auth status
sudo -u alter0 env HOME=/var/lib/alter0 PATH=/var/lib/alter0/.local/bin:/usr/local/bin:/usr/bin:/bin bash -lc 'printf "protocol=https\nhost=github.com\n\n" | git credential fill'
curl --noproxy '*' http://127.0.0.1:18088/readyz
```

验证通过后，服务内由 `Codex CLI` 发起的 `git commit`、`git push`、`gh pr create`、`gh pr merge` 会复用这套运行账户级凭证与签名配置。

对应 Nginx 配置与运行权限方案见：`docs/deployment/nginx.md`。若需要会话级预览或独立测试服务，请把 `alter0.cn` 与 `*.alter0.cn` 一并反向代理到同一共享运行时，再用 `scripts/deploy_test_service.sh <session_id> [service_name] ...` 注册当前会话服务。默认 `web` 会构建前端，并把当前分支后端启动命令注册给共享运行时托管，再把短哈希子域名注册为 `http` 反代；如只需要静态 UI 预览，可显式传 `--service-type frontend_dist`。

浏览器访问：

```text
http://127.0.0.1:18088/chat
```

发送消息：

```bash
session_id="$(curl -sS -X POST http://127.0.0.1:18088/api/terminal/sessions \
  -H "Content-Type: application/json" \
  -d '{"title":"CLI check"}' | jq -r '.session.id')"

curl -X POST "http://127.0.0.1:18088/api/terminal/sessions/${session_id}/input" \
  -H "Content-Type: application/json" \
  -d '{"input":"/help","attachments":[],"skill_ids":[]}'
```

### Run in CLI Mode

```bash
go run ./cmd/alter0
```

输入 `/quit` 或 `/exit` 退出。

### Terminal Shell

- 默认终端会话在 Windows 下使用 `powershell.exe`，并在启动时自动切换到 UTF-8 输出
- Linux / macOS 默认优先使用公共路径 `/usr/local/bin/codex`；若该路径不存在，则回退为 `codex`
- 如需统一指定 Codex CLI 路径，可通过环境变量 `ALTER0_CODEX_COMMAND` 或启动参数 `-codex-command` 设置
- 运行时会自动补齐 `$HOME/.local/bin`、`$HOME/.local/share/pnpm`、`/usr/local/bin`、`/usr/bin` 等标准 PATH，确保服务内 `Codex CLI` 可见 `codex`、`node`、`npm`、`npx` 与运行账户自带的 `gh` 包装器
- Windows 下显式指定 `cmd.exe` 时会补充 UTF-8 代码页初始化；如需稳定中文输出，优先使用 `powershell.exe`
- Terminal 会话退出后不会清空历史或线程标识；重新在原会话发送输入时，系统会优先复用已持久化的 Codex CLI 线程继续执行
- 若 Codex CLI 在线程续写阶段返回远端 compact 失败，Terminal 会保留原会话历史与工作区，但清空失效线程标识；下一次输入会在同一 Terminal 会话下自动启动新的 Codex 线程继续执行
- 对已退出或已中断的 Terminal 会话重新发送输入后，输入区上的旧运行态提示会立即让位给当前发送态，不继续显示“会话已退出”之类的过期提示
- `DELETE /api/terminal/sessions/{id}` 用于直接删除 Terminal 会话，并同步清理 `.alter0/state/terminal/sessions/{id}.json`，接口返回 `204 No Content`
- 若 Terminal 会话在首条输入前已失去底层运行态，首次发送会自动恢复同一会话并继续执行，不要求用户新建会话
- 同一 Web 登录态下，Terminal 会话历史默认跨设备共享；手机与 PC 均通过同一组服务端持久化记录恢复会话，不要求用户迁移历史

## Control API

### Channel

```bash
# 列表
curl http://127.0.0.1:18088/api/control/channels

# 创建/更新
curl -X PUT http://127.0.0.1:18088/api/control/channels/web-default \
  -H "Content-Type: application/json" \
  -d '{"type":"web","enabled":true}'
```

### Codex Runtime

```bash
# 查看当前 Codex 运行时
curl http://127.0.0.1:18088/api/control/codex/runtime

# 更新当前 Codex model 与思考深度
curl -X PUT http://127.0.0.1:18088/api/control/codex/runtime \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-5.4","reasoning_effort":"high"}'
```

说明：

1. Runtime 页面只读取服务运行账户当前活动 `CODEX_HOME` 的 `auth.json` 与 `config.toml`。
2. 额度信息来自当前 `auth.json` 的 quota 刷新结果；页面不再使用旧账号列表接口作为额度来源。
3. 运行时设置更新通过 Codex app-server 写回当前用户配置中的 `model` 与 `model_reasoning_effort`，不会覆盖其他 Codex 配置项。

### Skill

```bash
# 列表
curl http://127.0.0.1:18088/api/control/skills

# 创建/更新
curl -X PUT http://127.0.0.1:18088/api/control/skills/summary \
  -H "Content-Type: application/json" \
  -d '{"name":"summary","enabled":true}'
```

说明：

1. 服务启动后默认提供 `memory`、`preview-publish`、`frontend-design`、`doc-coauthoring`、`fullstack-developer`、`code-reviewer`、`webapp-testing`、`find-skills`、`test-driven-development`、`ui-ux-pro-max`、`code-simplifier`、`code-review`、`brainstorming` 与 `travel` 公有内置 Skill；`memory-maintenance` 作为系统维护专用私有 Skill 保留，不进入 Chat / Terminal 常规选择列表。
2. `memory` Skill 用于向 Skill / Codex 明确记忆文件的读取决策、写入路由、冲突优先级与禁止写入项，建议与 `memory_files` 一起启用。
3. 项目内置 Skill 全部由源码仓库直接承载。标准 skill 使用 `docs/skills/<skill_id>/SKILL.md` 作为 file-backed 主入口；其中 `preview-publish` 的静态产物发布脚本位于 `docs/skills/preview-publish/scripts/publish_preview_artifact.sh`。`code-simplifier` 与 `code-review` 两个 plugin-style 条目则分别以 `docs/skills/code-simplifier/SKILL.md` 和 `docs/skills/code-review/commands/code-review.md` 作为 alter0 的 file-backed 注入入口，并保留各自 `.claude-plugin/plugin.json` 元数据。Codex 启动前会把本轮选中的可读 file-backed Skill 目录复制到当前工作区 `.alter0/codex-runtime/skills/<skill_id>/`，并将运行时 `file_path` 重写为该工作区内路径，保证 Terminal 与 Chat 不依赖源码仓库相对路径。
4. `preview-publish` 是静态用户可见产物与完整测试服务的统一发布通道。Skill / Terminal 不得把 `/srv/...`、`.alter0/workspaces/...`、`file://`、`localhost` 或 `127.0.0.1` 作为用户可打开链接返回；HTML、Markdown 预览、截图、图片集合、文本报告、JSON 示例和代码样例等静态产物必须先发布到 `https://<service>-<short_hash>.alter0.cn` 后再作为交付入口。需要完整 Web 应用、后端路由或 API 联动时，也使用 `preview-publish` 发布会话级服务地址。
5. 服务不再随启动注册任何内置业务编排；Chat 默认直接通过 Claude Code CLI 或 Codex CLI 执行。
6. 所有可复用规则统一进入控制面可见的 `docs/skills/<skill_id>/SKILL.md` 或 plugin-style skill 入口，并由当前会话的 Skill 选择显式注入运行时。
7. `travel` Skill 会预置城市页、行程、地铁、美食、路线卡与 Codex 行程地图图片输出规则；稳定偏好写入 `docs/skills/travel/SKILL.md`，一次性行程细节仍只保留在目标城市页数据中。

### Skill

```bash
# 更新 Skill
curl -X PUT http://127.0.0.1:18088/api/control/skills/research \
  -H "Content-Type: application/json" \
  -d '{
    "name":"Research",
    "enabled":true,
    "metadata":{
      "skill.description":"面向资料整理和结论交付的可选能力。",
      "skill.file_path":"docs/skills/research/SKILL.md"
    }
  }'

# 通过 Chat 执行任务
session_id="$(curl -sS -X POST http://127.0.0.1:18088/api/terminal/sessions \
  -H "Content-Type: application/json" \
  -d '{"title":"检查当前仓库"}' | jq -r '.session.id')"

curl -X POST "http://127.0.0.1:18088/api/terminal/sessions/${session_id}/input" \
  -H "Content-Type: application/json" \
  -d '{"input":"检查当前仓库并直接完成需要的修改","attachments":[],"skill_ids":[]}'
```

说明：

1. Runtime Profile 作为历史配置模型保留；当前稳定 Chat 入口通过会话选择 Skill、MCP、Memory 与模型/Codex 进入同一底层执行链。
2. Web 对话统一使用 `Chat`；代码、写作、旅行等任务不再依赖内置业务编排。
3. Runtime Resolver 使用已启用且健康的 `Claude Code + provider profile`；无可用 Provider 时使用 `Codex Direct`；Claude 执行失败直接返回错误。
4. 启动前会在当前 Session 工作区注入 `CLAUDE.md` 或 `AGENTS.md`、Skill 副本、Memory 摘要、MCP 配置、仓库/附件/产物路径和工作区边界；Codex Direct 的托管 `AGENTS.md` 同时约束用户可见产物必须先发布到会话预览或服务域名，不得把服务器本地路径作为用户验收入口。
5. Skill 文件由 `docs/skills/<skill_id>/SKILL.md` 承载，业务能力通过 Skill 复用；用户可在会话级调整公有 Skill 选择。
6. Context Files 当前使用根级 `AGENTS.md`、`SOUL.md` 与 `memory/USER.md`、`memory/MEMORY.md`、`memory/daily/<YYYY-MM-DD>.md`、`memory/projects/<project>.md`、`memory/conversations/<conversation_id>/summary.md`，并支持启动参数解析后的长期记忆文件和天级记忆目录。`AGENTS.md` 是运行规则上下文，`SOUL.md` 是强约束上下文，其余为事实型记忆。所有记忆 Markdown 写入都必须通过当前 CLI Runtime 完成；服务侧只解析路径、注入上下文、提供只读聚合与维护入口，不直接把会话轮次、任务摘要或压缩片段落到记忆 Markdown。
7. 用户显式要求记住时，当前 CLI runtime 可写入对应 Markdown 记忆；会话归档生成 summary；系统维护任务每日加载 `memory-maintenance` Skill 做长期整理。维护 prompt 要求读取当日/昨日 Daily Memory、对照长期记忆、只提升稳定事实/偏好/决策/流程/约束，禁止复制原始 transcript、日志、密钥和一次性任务细节。系统维护任务作为 Scheduler 内置 Cron Job 注册，不可删除，可停用或重新启用。
8. `Settings > Skills` 用于管理可选能力；`Chat` 是通用执行入口。

### Cron Jobs

```bash
# 列表
curl http://127.0.0.1:18088/api/control/cron/jobs

# 内置任务返回 builtin=true；例如 system-memory-maintenance 与 system-session-cleanup。
# 内置任务不能 DELETE，可通过 enabled 停用或重新启用。
curl -X PUT http://127.0.0.1:18088/api/control/cron/jobs/system-memory-maintenance \
  -H "Content-Type: application/json" \
  -d '{"enabled":false}'

# 创建/更新（可视化字段 + cron_expression）
curl -X PUT http://127.0.0.1:18088/api/control/cron/jobs/job1 \
  -H "Content-Type: application/json" \
  -d '{
    "name":"daily-summary",
    "enabled":true,
    "timezone":"Asia/Shanghai",
    "schedule_mode":"daily",
    "cron_expression":"30 9 * * *",
    "task_config":{
      "input":"summarize yesterday tasks",
      "retry_limit":1
    }
  }'

# 查看指定 cron job 的触发记录与会话回链
curl http://127.0.0.1:18088/api/control/cron/jobs/job1/runs

# 按 cron 来源筛选会话历史
curl "http://127.0.0.1:18088/api/sessions?trigger_type=cron&job_id=job1"
```

## Testing

后续开发默认遵循 TDD：功能新增、缺陷修复、行为调整或重构需先新增或更新表达目标行为的测试，再完成实现与重构。纯文档、注释、格式化、依赖元数据或无法自动化验证的变更可免新增测试，但交付说明需明确免测原因与替代验证方式。

```bash
go test ./...
```

## Roadmap

1. Skill 配置与执行链路打通（按 skill 选择执行器/参数）。
2. Control 存储（SQLite/PostgreSQL）与热更新。
3. Channel 扩展（IM/HTTP 回调）与统一回投能力。
4. 任务调度增强（Cron 表达式、重试、幂等、死信）。
5. 鉴权与多租户。

## Contributing

欢迎提 Issue / PR。代码类变更需遵循 TDD，并在提交前执行与改动范围匹配的测试；涉及共享链路、跨模块契约或用户可见行为时，建议执行：

```bash
go test ./...
```

## License

MIT, see [LICENSE](./LICENSE).
