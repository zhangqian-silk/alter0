# Requirements

> Last update: 2026-06-24

`alter0` 的需求清单按领域模型维护。后续新增需求不再使用线性编号，也不按提交顺序堆叠；需求应落到对应领域、子域与能力项下，使用稳定领域路径表达，例如 `runtime.execution.cli-runtime`、`memory.files.injection`、`task.workspace.runtime`。

稳定默认约定：

- Web 前端所有需要可见时间的设置视图、会话列表、详情面板与任务视图统一固定为上海时间（`Asia/Shanghai`）与 24 小时制；`Chat / Terminal` 的消息正文区不显示逐条消息或 turn 时间。
- 自然语言任务默认通过 CLI Runtime 执行：存在启用且可用的 Model Provider 时优先使用 `Claude Code + provider profile`，未配置、不可用或鉴权失败时兜底使用 `Codex Direct`。
- 服务侧负责会话、工作区、Skill 仓库、Markdown 记忆文件、运行时注入与结果归档；会话内上下文压缩由 Claude Code 或 Codex CLI 自身处理，跨会话长期记忆由定时任务加载 `memory-maintenance` Skill 整理。
- 系统维护任务不提供复杂配置项：记忆维护与会话清理作为 Scheduler 内置任务每日自动运行，内置任务不可删除，但可在 Scheduler 控制面停用或重新启用；会话清理默认每日清理超过 7 天不活跃且未置顶、无 queued/running 任务关联的会话，置顶会话始终跳过自动清理。

状态说明：

- `supported`：已在主干代码可用
- `in-progress`：正在落地，接口或行为尚未稳定
- `planned`：已确认方向，待排期

## 领域索引

| 领域 | 范围 | 状态 | 细化文档 |
| --- | --- | --- | --- |
| Runtime & Orchestration | 输入通道、统一消息、意图路由、CLI Runtime 选择、调度、存储、观测 | supported | [runtime-orchestration.md](requirements-details/runtime-orchestration.md) |
| Conversation & Session Experience | Chat、Terminal 运行入口、历史、移动端、阅读与输入体验 | supported | [conversation-session-experience.md](requirements-details/conversation-session-experience.md) |
| Skill & Memory | CLI Runtime 注入、Skill 仓库、Memory Files、长期记忆、会话归档与定时整理 | supported | [skill-memory.md](requirements-details/skill-memory.md) |
| Task, Terminal & Workspace | 异步任务、任务观测、任务日志、产物交付、Terminal 会话、独立工作区 | supported | [task-terminal-workspace.md](requirements-details/task-terminal-workspace.md) |
| Control, Operations & Governance | 控制面配置、Model Provider、Claude Code provider profile、Codex Runtime、服务重启、部署基线、认证凭据、TDD 研发约束 | supported | [control-operations-governance.md](requirements-details/control-operations-governance.md) |

## Runtime & Orchestration

核心对象：`UnifiedMessage`、`OrchestrationResult`、`Intent`、`Command`、`ExecutionPort`、`RuntimeResolver`、`CLIRuntime`、`SchedulerJob`、`Channel`、`TraceContext`。

稳定需求：

- CLI、Web、Cron 等输入源统一转换为 `UnifiedMessage`，并携带 `message_id`、`session_id`、`trace_id`、`channel_type`、`trigger_type` 与业务载荷。
- 编排层负责意图识别与路由：命令进入 `CommandRegistry` / `CommandHandler`，自然语言进入 `ExecutionPort` 后由 `RuntimeResolver` 选择 CLI Runtime；Cron 触发复用同一编排链路；当消息显式指定 `alter0.execution.engine=codex` 时，斜线前缀输入视为 Codex 原生命令或提示词，直接进入 `Codex Direct`。
- 命令能力稳定提供 `/help`、`/echo`、`/time` 与 `/now`。
- 自然语言执行通过 CLI Runtime 完成：启用且可用的 Model Provider 对应 `Claude Code + provider profile`，无可用 Provider 或 Claude Code 运行失败时进入 `Codex Direct`。
- 调度领域支持 Cron Job 配置、内置不可删除 Job、可视化周期字段、触发记录、触发会话归档、runs 查询与来源回链；内置 Job 允许切换 enabled 状态。
- 存储默认采用 `.alter0` 本地文件，Control 配置与 Scheduler 状态以 JSON 为主，Memory 主存以 Markdown 为主。
- 观测能力覆盖结构化日志、Prometheus metrics、`/healthz`、`/readyz` 与 trace/session/message 维度。

## Conversation & Session Experience

核心对象：`Session`、`Message`、`LiveUserMessage`、`RuntimeTraceEvent`、`ViewportState`、`SessionHistoryBucket`。

稳定需求：

- `Chat` 是唯一对话入口，默认绑定内置 `main` Skill `Alter0`，并通过会话级 Skill、Memory、MCP 与模型选择承载代码、旅行、写作等专项任务。新空白 Chat 会话默认勾选全部可用公有 Skill，用户后续可按会话手动取消或调整；历史会话恢复时，已删除或禁用的 Skill 必须按当前公有 Skill 目录实时剔除，新增勾选必须无需刷新即可作用于后续消息。显式选择 `Codex` 时，Web 对话框支持 Codex CLI 内置斜线命令候选；Terminal 当前活动会话明确为 Codex shell 时，Composer 同样提供该候选。
- Web 登录态下，Chat 会话历史统一进入服务端 Session history；历史 Chat 会话在读取阶段迁移为当前 Chat 会话结构，并统一通过 `route=chat` 进入列表与详情恢复。
- `Chat / Terminal` 运行页需生成同一规则的 8 位短 hash 标识，作为会话级引用、显式 URL 恢复参数与人工排障的稳定标识符；左侧会话列表不展示短 hash，完整会话 id 与 Terminal `terminal_session_id` 只作为接口、持久化和工作区隔离标识，不直接作为列表或 URL 展示值。`Chat` 主入口在无显式 `session_id` 时默认打开最新会话，主导航切回 Chat 时不得沿用旧 query 或浏览器上次活动会话锚点。
- Web 入口稳定提供根路径到 Chat 的默认进入、`/chat`、`/terminal`、`/settings`、`/login` 与 `/logout`。受保护页面、受保护预览工作区与 API 统一走同一登录态校验。访问工作台页面触发登录时，登录回跳只保留 canonical path，不携带会话级长 query。仅静态只读 host 保留匿名访问。
- 新会话先使用统一占位标题 `New`，早期多轮内可根据更具体输入自动升级标题，避免长期保留“拉取仓库”“分析仓库”等低辨识度名称。
- 新对话空白会话保持唯一；已有空白会话时，`New` 复用并聚焦该会话。
- 同一会话内请求保持顺序一致，由当前 Chat-scoped Terminal session 的状态表达 `busy / ready / failed`。
- `Chat` 消息提交统一使用 `POST /api/terminal/sessions/{session_id}/input?scope=chat`，复用 Terminal-compatible 数据模型与状态机但与 Terminal 默认 session 列表分开存储；前端不再接入 `/api/messages`、`/api/messages/stream`、消息 SSE、SSE parser、本地 `Thinking` 过程步骤或后台 Task API。发送后由 Terminal session 状态标记 `busy`，执行完成或失败后，再通过返回的 session 或 Chat-scoped Terminal session 详情恢复写回 assistant 正文与结构化步骤。
- `Chat` 消息入口在请求进入执行链前必须先持久化本轮 `user` 消息；assistant 消息、route、错误码与结构化步骤在执行完成后追加落库，避免浏览器关闭、刷新或请求断开时已发送用户消息只存在于前端乐观状态。
- `Chat` 在同一会话内保持追加式历史：每次发送都新增一条 `user` 消息与对应 assistant 消息，发送后当前活动时间线需立即回到底部展示本轮新增消息；后续结果或会话详情恢复只允许补丁当前未完成的 assistant 条目，已收口历史不得被迟到结果回写。
- 请求断开或页面刷新后，前端优先回源当前会话详情，用服务端已持久化的消息覆盖本地占位态，而不是立即把同一请求重发一遍。该恢复链路即使在服务端集合接口已返回当前会话摘要、但尚未附带完整消息时也必须继续触发。只有恢复失败时才收敛为失败态；若没有可用正文，失败提示需明确提示刷新。本地缓存中残留的 `streaming` 消息不得长期停留在 `In Progress`。
- 若当前活动会话的服务端历史只包含最新 `user` 消息且尚无对应 assistant 或失败消息，前端必须继续按单会话详情恢复，不得把该 user-only 历史判定为稳定完成态。
- 刷新页面时，`Chat` 必须优先保住当前活动会话：若服务端会话列表暂时尚未返回该 `session_id`，前端先从浏览器侧活动会话快照恢复当前条目与最近消息，再按 `session_id` 单独回源详情；若集合接口返回的消息历史短于本地已追加历史，且本地仍有未完成助手消息或更新中的本轮消息，前端不得用较短远端历史覆盖本地时间线；在确认服务端不存在该会话前，不得直接把当前活动会话替换成新的空白 `New` 会话。
- 刷新页面或切到其他会话后，`Chat` 仍需保住最近已知会话列表：浏览器侧最近会话快照至少覆盖当前活动会话之外的最近若干条会话；当服务端集合接口暂时漏掉其中某条会话时，左侧会话列表不得立刻把该会话删除，而应继续保留本地条目并等待单会话详情或后续集合结果确认。
- `Chat / Terminal` 前端缓存需保留当前已加载会话的完整消息或 turns：同一工作台内的 24 小时运行态缓存不得裁剪已加载历史；`Chat` 浏览器侧还需维护 24 小时 `localStorage` 完整消息快照，用于刷新、重开或 `sessionStorage` 丢失时优先恢复首屏；同时维护轻量会话信息快照，用于完整消息缓存写入失败或被清理时恢复会话列表与活动会话元数据。缓存只作为本地加速与断网前置恢复，服务端 Session history 与后续单会话详情回源仍是最终事实源；每次访问、切换、刷新或前台恢复都会按最新合并结果刷新缓存时间。
- `Chat` 的会话存在性、配置与恢复状态需由 Terminal session store 承担第一责任：输入入口在请求开始、完成、失败时分别写入 `busy / ready / failed` 等稳定状态；会话置顶、删除、列表、详情与恢复均复用 Terminal session API。前端展示、计数和发送 payload 需以当前可用公有 Skill 目录过滤后的有效选择为准，避免因浏览器刷新、请求断开、前端本地状态丢失或 Skill 目录变化导致会话“消失”、失效 Skill 继续注入或直接 `load failed`。
- 运行时执行过程需以统一 `RuntimeTraceEvent` 数据模型承载，并覆盖 Terminal turn、Terminal input 结果与会话历史持久化。Terminal turn 摘要需直接输出 `runtime_trace_events` 作为 Chat / Terminal 过程展示的 canonical 数据；事件详情通过 `session_id / turn_id / event_id` 索引，并只在用户展开具体 `Thinking / Process` 步骤时懒加载完整 detail，首屏与会话详情分页不得提前返回大段 thinking 明细。事件类型、来源、角色、生命周期、状态、block 与 action 信息只能来自底层 provider、工程 adapter 或 alter0 自身确定生成的字段，不允许通过自然语言内容 heuristic 猜测。前端优先消费结构化事件，并仅按 `RuntimeTraceEvent.kind` 执行过程披露过滤，不依赖解析自然语言过程文本。
- 消息区支持 Markdown 安全渲染、一键复制最终回复、Process 折叠状态、逐条 patch 与逐帧合并刷新；Chat / Terminal 最终输出统一使用稳定的 `MessageMarkdownShell` 承载，正文先于复制工具栏渲染，不安装脚本长按选区、假选中态或编辑态兜底，且父级无关重渲染不得重写相同 markdown 的文本 DOM；React 托管的普通页面也需对正文型字段提供同一安全 Markdown 渲染能力，覆盖 Memory 文档、Task 请求/结果/日志/产物摘要、Control 描述、Cron 输入、Skill 说明、Codex 运行时说明与 Session Profile 非等宽字段。Markdown 视觉需保持正文阅读节奏：ATX/Setext 标题紧凑、段落自然、删除线和自动链接按正文渲染、嵌套列表按 Markdown 缩进保留真实层级，列表项内允许继续承载引用与代码块，普通链接显示外链箭头，代码块保留浅灰弱边界；Markdown 表格需渲染为真实表格结构并保留列对齐，只保留横向分割线、无外框卡片和表头灰底，短表格至少铺满消息宽度，普通长文本在单元格内自动换行，链接、URL 与代码保持不硬断开，只有真实不可断内容超宽时才在消息容器内滚动；ID、路径、密钥、配置值、时间戳等元数据字段继续按纯文本或等宽字段展示。
- Chat 允许通过显式预览参数 `/chat?markdown_demo=1` 临时覆盖当前时间线视图并展示一条非持久化 assistant Markdown 语法覆盖样例；样例覆盖 ATX/Setext 标题、段落换行、强调、删除线、自动链接、图片、引用、嵌套列表、任务项、列表内引用与代码块、分割线、代码块、对齐表格与 raw HTML 转义。表格样例覆盖短字符、长中文、长 URL/代码和混合内容场景；折叠示例中的 HTML 标签按代码块展示，折叠内容本身按普通 Markdown 展示。该入口只用于渲染验收，不写入会话历史，也不替代真实 Chat 会话恢复规则。
- `Chat / Terminal` 的消息阅读结构统一采用轻量 IM 式消息流：用户消息右对齐且使用浅灰低对比紧凑气泡，气泡高度需由较小纵向 padding 与独立消息行高控制；助手消息左对齐并弱化为无边框正文阅读流，Chat 正文工作区使用白底无框阅读面，不在消息区叠加明显面板边框、背景分界或卡片容器；`Process` 默认收敛为 `Thinking / 已思考` 内联轻量披露行，只显示步骤数量，不显示耗时，展开后在当前消息内展示步骤详情，步骤行需显示与过程披露过滤一致的类型标签、耗时与状态，移动端也保持同页内联展开，最终 Markdown、图片与复制动作都收敛在对应消息区域内。模型与 Skill 配置面板需提供过程披露过滤勾选，默认只勾选 `important_text`；`reasoning / plan / tools / commands / system` 等事件只有在用户显式勾选后展示。Terminal 最终输出的 Markdown 正文必须是静态可选中文本，复制动作位于助手正文下方，代码块作为独立浅灰内容块呈现；消息正文区不显示逐条时间，只有进行中、排队、失败等状态保留紧凑状态标签；长历史默认优先渲染最新上下文，用户滚到顶部或点击 `Load earlier messages / 加载更早消息` 后按批次渐进显示本地已加载消息；Chat 与 Terminal 会话详情首个请求只返回最新 turns 页，若 `turns_paging.has_more_before=true`，前端必须继续在后台按 `turn_before` 自动补齐更早服务端分页并按 turn/message id 与时间顺序合并；后台补齐更早历史不得扩展当前阅读窗口、强制回到底部或覆盖用户正在进行的滚动、输入与配置操作；发送新消息后的轻量详情不得截断当前已加载历史。后续新增运行页若呈现用户/助手消息，也必须复用同一 `runtime-message-*` 消息外壳与 `RuntimeTimeline` block model，不再自建页面私有气泡系统。
- 专项 Skill 需显式声明 deliverables contract，作为底层 Skill 执行上下文中的最终交付物约束；Web Chat 不再提供独立 Chat 的 Deliverables 或 Session Profile 面板。
- Skill 还需支持独立的 `completion_checks` 机器规则，用于把交付契约下沉为可执行的运行时产物检查。`deliverables` 负责用户可见契约与 prompt 约束，`completion_checks` 负责文件存在、公开 URL、workspace service 发布状态、Session 属性非空等确定性校验，并可在失败时声明一轮仅面向当前 Session 的 Codex 修复指令。
- `Chat / Terminal` 的 `Process` 步骤在真机窄屏下仍需保持整列阅读宽度；步骤序号、展开图标、标题与状态信息需在同一行垂直居中；长中文说明、路径、错误日志、inline code、Markdown 表格和命令明细必须在消息容器内自然换行或仅在内容块内部横向滚动，不得塌缩成逐字竖排窄列，也不得制造页面级横向滚动；展示层还需容忍零宽断行字符和“每字一行”的异常历史文本，并在渲染前修正为可读段落。
- `Chat` 的消息时间线在内容较少时仍需顶部收口：少量消息、短回复、折叠后的 `Thinking / 已思考` 披露行与状态标签继续贴近各自消息气泡，不得被满高布局拉出大段垂直空白。
- `Chat / Terminal` 进入已有内容会话时默认定位到消息时间线或 Terminal 输出区底部，优先展示最新上下文；`Chat` 在同一活动会话发送新消息后也需回到底部展示新追加的用户消息与助手占位。除这类用户主动追加外，不得在同一会话持续更新、轮询刷新或 Process 展开期间覆盖用户的历史阅读滚动。
- `Chat / Terminal` 的阅读定位条必须以悬浮 overlay 形式附着在消息区右下角，不得继续参与消息时间线的正常文档流；空白会话或少量消息时，不允许因为定位条占位把消息区额外撑高并制造伪滚动。`上一条 / 下一条` 按当前视口可见块实时计算目标，并支持连续跳转；当前块已被上一次跳转对齐到顶部偏移后，下一次 `上一条` 必须继续指向前一块。
- `Chat` Composer 支持图片附件草稿、输入框内 PC 剪贴板图片粘贴、缩略图预览与消息内图片回显；最近会话恢复仅持久化稳定图片资产引用，避免重复保留原始大图 payload；缩略位继续使用预览图，但消息回显与再次查看必须优先读取原图资源。助手 markdown 图片需在消息区直接以内联图片懒加载显示。带图消息只允许走支持视觉输入的模型链路，不进入异步 Task，也不静默降级到 Codex 文本执行；显式选择 `Codex` 时，服务端必须把已落盘图片路径作为 Codex CLI `-i` 输入传入直连执行链。
- Web 前端所有需要可见时间的管理视图、会话列表、详情面板与任务视图统一使用北京时间（`Asia/Shanghai`）与 24 小时制；`Chat / Terminal` 的消息正文区不显示逐条消息或 turn 时间。Cron 创建表单默认时区固定为 `Asia/Shanghai`。
- Web 侧边栏、历史折叠、页面滚动隔离、克制冷灰工作台阅读主题、PC 端平面化控件、移动端轻量白色导航抽屉、移动端软键盘跟随、设置底部面板、低功耗轮询与长文本宽度约束作为统一前端体验要求维护；移动端运行页顶部 `Menu / 标题 / New` 控件必须像发送按钮一样支持首触执行，不得在输入框聚焦或软键盘打开时退化为先收键盘、第二次才响应。移动端共享 runtime Composer 的键盘位移必须采用稳定底部偏移，不依赖 transform 合成层，避免输入框阴影在 iOS Safari 键盘动画中留下残影。
- Web 前端需提供受显式开关控制的点击诊断能力，用于记录事件目标、顶层命中元素、遮罩层状态、`preventDefault` 状态、当前焦点与主线程长任务；默认不启用，不影响正常交互路径。
- Terminal 长输出复制必须保持可用且不放大 DOM 体积：复制 payload 不得完整写入 `data-*` 属性，长输出轮询、草稿输入和复制操作不得触发整段 Markdown 反复解析或相同 `innerHTML` 反复写入造成明显卡顿；`Chat / Terminal` 最终输出不得依靠全局 `user-select !important` 补丁维持选择能力，应通过统一稳定的 markdown shell 保留浏览器原生长按选中与复制菜单。
- 当前运行页的 Session 列表需直接展示在左侧主导航内，采用工作台式分组：置顶会话单独进入 `Pinned / 置顶` 分组并固定在 `Today / 今天` 上方，其余会话再按最近时间分到 `Today / Yesterday / Earlier`（中文对应 `今天 / 昨天 / 更早`）；`Chat / Terminal` 统一使用 `Sessions` 栏标题与 `New` 新建入口；移动端会话列表随左侧主导航抽屉展示，两条运行页顶部都只通过 `Menu` 打开左侧导航抽屉。列表条目主体只展示标题并在可用宽度内单行截断，真实会话尾侧固定提供三点更多按钮；展开菜单承载置顶、查看详情与删除操作，查看详情聚焦对应会话并打开 `Details`，删除需经过确认弹窗后才进入删除链路。空白 `New` 草稿/占位统一视作虚拟会话，不显示三点菜单，不支持置顶、详情或删除；同一路由只保留一个空白虚拟会话，重复点击 `New` 只聚焦现有空白入口，不创建多个空会话。长标题不得撑开导航、分组、列表或列表项宽度；新增会话插入、列表刚好填满或跨过滚动阈值时，不得触发滚动锚点补偿、滚动槽宽度重算或重新分配 rail 头部高度，也不得造成 `Sessions / New` 区块在不同运行页之间跳动；处理中会话在标题旁显示 loading，其他状态不显示状态灯。
- `Chat` 的已发送会话必须以服务端 Session history 为恢复源，并在同一 Web 登录态下跨设备共享；历史 Chat 会话加载时迁移为当前 Chat 会话模型，详情恢复统一使用 `route=chat`。未发送草稿与当前浏览器局部 UI 状态可继续本地保存，但不得阻断服务端会话摘要、配置和消息历史的恢复。
- Session history 必须维护 `last_active_at` 与 `pinned`。发送消息、assistant 收口、打开会话详情、Terminal 输入/输出和任务结果写回会刷新活跃时间；运行页会话列表先把置顶会话汇入独立 `Pinned / 置顶` 分组，再让非置顶会话按最近活跃时间排序并分组。系统默认清理超过 7 天不活跃且未置顶的会话，并同步移除 Session history、运行时 registry、会话附件/工作区和关联任务引用；仍有关联 queued/running 任务的会话在任务进入终态前跳过清理。
- 本地 Session history 物理文件按会话拆分：新 Chat 会话使用自身 `session_id` 写入 `.alter0/sessions/_default/<session_id>.json` 或 `.md`；历史 `alter0-chat` 归档日文件与 `chat` 分文件布局在读取时合并到当前 Chat 会话模型；旧版 `.alter0/sessions.json` / `.alter0/sessions.md` 在读取时自动重构为新的分文件布局并移除旧聚合文件。
- `Chat / Terminal` 的会话条目不展示 ready/failed/exited/interrupted 等行内状态灯；仅处理中会话在标题旁显示 loading。workspace header 的状态按钮继续共享当前会话状态语义，状态名称仅保留给可访问性语义与悬浮提示。
- Web Shell 由 React 单一工作台直接渲染：`src/app/WorkbenchApp.tsx` 负责 `/chat`、`/terminal`、`/settings` 三个稳定顶层路由、语言切换、主导航折叠/抽屉和运行页/设置页分派。主导航只暴露 `Chat / Terminal / Settings`，Settings 内部按 `Runtime / Skills / Memory / Schedules` 分区渲染。壳层稳定暴露 `app-shell[data-workbench-route]`、Settings 页的 `data-route-family="settings"` 与各视图自己的 `data-route / data-conversation-*` 作为样式钩子。
- `/chat`、`/terminal` 与 `/login` 默认以英文启动，HTML 根节点语言标记为 `en`；Web Shell 保留显式语言切换入口，切到中文后需同步更新壳层文案与 `document.documentElement.lang`。
- 登录页需与工作台共享同一视觉基线：使用 `IBM Plex Sans + Sora` 字体组合、近白安全入口表面与安全入口语气，避免退回默认系统登录页样式。
- Web Shell 的稳定视觉基线收敛为两层：左侧固定主导航负责品牌、`Chat / Terminal / Settings` 三条稳定入口、当前运行页 Session 列表、Settings 工具入口与语言切换，右侧主面板统一承载运行页和 Settings 管理页；`Chat / Terminal` 在主面板内部统一采用「主时间线工作区 + 底部 Composer + 固定 workspace header」结构，并直接复用 workspace body、chat screen、composer、消息气泡与移动端顶部操作行语义 class。Web Shell 视觉基线需参考 Gemini 式扁平工作台：主工作区、Settings frame、管理分区、表格、详情面板和空态不再依赖外层圆角、卡片边框或厚阴影，层级通过留白、轻量分割线、低对比选中态与 Composer 胶囊建立；设计图需维护在 `docs/design/workbench-flat-redesign.html` 与对应 PNG。Chat、Terminal 与 Settings 顶部标题需共享紧凑 `workbench-title` chrome：运行页保留会话标题、状态信号与 `Details`，Settings 路由页只保留当前路由标题与同规格标题标记，并收进同规格主面板 frame；Settings 的 frame、标题、正文区和分区索引不得使用独立淡入、位移或页面出现动效；移动端顶部 `Menu / New` 等边缘操作需使用无边框图标按钮视觉并保留可访问文本标签；Settings 正文必须作为 frame 内部滚动区，不得因外层 frame 裁切而失去滚动能力。`/chat` 仅作为旧 URL alias 进入 `/chat`，不再生成独立运行页、Skill 选择器、Deliverables 或 Session Profile 面板。
- `Skill` 与其他 React 托管页面共享同一扁平 workbench surface system：列表、管理表单、托管字段块与消息块使用一致的白底主表面、浅灰辅助层和低对比选中态，不再默认使用卡片边框、厚圆角或重阴影表达层级。
- `/chat` 与登录页的对外品牌文案统一使用 `Alter0`：浏览器标题、登录标题、导航品牌位、会话栏标题与欢迎区 tag 不再暴露小写服务名。
- Terminal 路由页继续由 React 原生实现，会话栏、工作区头部、Process、输出区和 Composer 的状态与交互全部由 React 维护；旧版 Terminal 仅作为布局关系与 `terminal-*` DOM 契约参照，不恢复 legacy runtime 控制器或脚本接管。
- 主导航、控制台与资产页默认以高密度信息架构呈现：主导航主工作流只保留 `Chat / Terminal` 三个入口，并用单个 `Management` 工具入口承接所有管理能力；控制面和资产能力在 Management 页内部用分组切换和高密度正文承载，长列表优先使用表格或主从视图，不再把大量配置和任务详情平铺为低密度信息矩阵。
- 移动端 Web Shell 使用 `100dvh` 动态视口协调壳体、顶部 workbar 和输入区：浏览器工具栏切换与软键盘弹起时，`html / body / #frontend-root` 不做 fixed 页面锁和 `overflow: hidden` 根层锁，App Shell 使用 `height: 100dvh` 自动贴合可见高度，并在 viewport meta 中声明 `interactive-widget=resizes-content` 作为支持浏览器的键盘 resize 策略。真手机宽度下真实顶部移动 workbar 固定为 `position: fixed; top: 0`，并用无 transition 的合成层 `transform: translate3d(0, var(--mobile-viewport-offset-top, 0px), 0)` 对齐 iOS Safari 平移后的 visual viewport 顶边，workspace body 第一行仅保留 header footprint；真实底部 Composer 通过顶层 portal 脱离 `.runtime-workspace-body` 与 workspace grid，作为 `position: fixed; bottom: 0` 浮层贴住动态视口底边，workspace body 仅保留静态 Composer footprint。workspace header、正文滚动区、空态、阅读定位条、命令候选和配置面板不消费 `--keyboard-offset` 或 transform 改写高度，不出现底部空白、内容裁切、整页上移或输入区重复上移。
- 移动端运行页的左侧导航抽屉必须保持统一开合语义：`Chat / Terminal` 都只保留 `Menu` 作为抽屉入口。点击遮罩、切换路由、切换会话或创建新会话后不得残留旧的展开层。
- 移动端运行页左侧导航抽屉需优先保证真机稳定性：遮罩保留淡入淡出，抽屉本体仅保留一层轻量侧滑，不叠加多层位移、淡出或条目级顺序动画；抽屉面板使用近白表面、平面菜单、细分割线和自然滚动的会话区，抽屉内置顶会话单独位于 `Pinned / 置顶` 分组，其余会话再按最近时间分组，并统一采用「标题 / 尾侧三点菜单」的紧凑导航列表结构，仅处理中会话在标题旁显示 loading，避免退回松散白色块、状态灯、元信息或过度胶囊化。
- 共享运行时的短哈希预览 host 与主域工作台必须落在同一登录保护边界内：`/login` 可直接在预览 host 打开，登录态 cookie 需对 `*.alter0.cn` 生效，避免主域与预览子域重复维护独立会话。
- 共享运行时采用 `supervisor -> web child` 进程模型时，主 Web child 必须继承非空 `web_login_password`；只有 workspace service 托管出来的预览后端允许通过专用运行时标记移除自身登录层，复用共享网关登录态。
- `Chat / Terminal` 的移动端键盘弹出链路需保留浏览器原生软键盘手势：首次触摸主输入框时不得在 `pointerdown / touchstart` 捕获阶段取消默认行为，不得主动 focus、锁定 `window` page scroll、通过 `scrollTo` 干预真实焦点，也不得记录或回放页面级滚动锚点。键盘开合过渡期内，运行页只通过 `100dvh` 自然同步 App Shell 可见高度，fixed Composer 通过 `bottom: 0` 贴住动态视口底边；输入框后方的 `workspaceBody / runtime-workspace-screen` 等滚动容器不做短时锁定，移动 workbar 只消费 `VisualViewport.offsetTop` 做独立 transform 坐标对齐，workspace header 与正文 panel 不单独消费键盘变量做位移。其他组件必须由 App Shell 动态视口高度、document 正常滚动语义与静态 workspace inset 保持原位，不再通过页面级滚动锚回逻辑接管浏览器键盘动画；键盘动画不得造成页面整体分辨率/可视区域突变。
- `Chat` 的移动端发送按钮在触摸提交时，必须先让当前主输入框失焦，再提交当前草稿；键盘回收与 composer 回弹由 `100dvh` 和 fixed bottom 自然恢复，不允许发送后键盘停留不收或残留悬空底部占位。
- `Chat / Terminal` 在移动端采用 fixed 底部 Composer 时，真实 Composer 必须位于 workspace body 外的顶层 portal，消息滚动区与空态工作区只保留静态 Composer footprint，不随软键盘高度动态压缩；对话、长输出与空态说明不得被键盘链路改写高度或位置。
- `Chat / Terminal` 在移动端的 Composer 回弹到底边时，运行区保持原位；键盘收起、输入框失焦和视口回弹后，不允许遗留额外底部空白、悬空按钮或上一轮键盘高度对应的占位残影。
- `Chat / Terminal` 在移动端键盘弹起与收回期间，只允许底部 Composer 通过 fixed bottom 贴住动态视口底边；顶部操作行、紧凑 workspace header、正文滚动区、空态、命令候选与配置面板保持布局原位，不跟随键盘位移做额外动画。
- `Chat / Terminal` 在移动端软键盘弹起期间，底部 Composer 必须保持为运行页最上层交互层；消息阅读定位按钮与 Terminal 四键定位条在主输入框聚焦后必须主动隐藏，待输入框失焦、键盘收起后再恢复，不得压到输入框、附件条或键盘上方。
- `Chat / Terminal` 的主输入框在移动端必须显式关闭系统自动填充、卡片、地址与密码类输入辅助条；键盘上沿不得再额外挂出会暴露底部残留页面层的系统输入助手。
- `Chat / Terminal` 的移动端主输入框必须保持不低于 16px 的可编辑文本字号，避免 iOS Safari 聚焦输入法时触发页面自动缩放、横向裁切或分辨率突变。
- `Chat / Terminal` 的移动端发送按钮必须支持在软键盘保持打开时直接点按提交；首触发送需覆盖 `pointerdown(touch)` 与 `touchstart` 提交链路，并在同一次触摸内去重，不允许先消费成键盘收起或焦点切换，再要求第二次点击才真正发出请求。
- `Chat / Terminal` 的四键阅读定位条需统一使用同一套共享实现与圆形按钮语言，不再为不同运行页维护分叉样式或独立跳转逻辑。
- 运行页 Composer 的键盘跟随只依赖 CSS 动态视口和 `bottom: 0`，不额外叠加 `bottom` 过渡动画；键盘收起与输入区回弹阶段应保持直接、稳定的回贴节奏。
- 输入框 blur 后，运行页应沿着 `100dvh` 的实际回弹过程恢复高度，不额外保留键盘占位，避免底部输入区和正文区出现闪烁。
- 输入框聚焦且软键盘已确认弹出后，移动端运行页不得用 `VisualViewport.offsetTop`、页面级 scroll 锚点或 JS 变量驱动 App Shell、workspace header 与正文 panel 位移；高度收缩以 `100dvh` 为准，Composer 以 `bottom: 0` 贴住动态视口，避免浏览器键盘动画和脚本位移叠加造成页面整体再次上移或输入区先消失再出现。
- `Chat / Terminal` 在页面从后台恢复到前台、浏览器重新激活当前页，或系统把当前 WebView 恢复为可见状态时，除补拉会话与任务数据外，还必须立刻重算共享视口诊断变量和 Composer 静态 footprint；前台恢复后的第一帧不允许沿用后台前的旧输入区高度或旧底部占位。
- Web Shell 的抽屉式单列布局仅在主视口宽度 `1100px` 及以下触发；高于该阈值时保留左侧固定主导航与右侧主面板。进入窄屏后主导航切换为贴边抽屉，当前运行页的会话列表随主导航一起展示，由工作区头部的 `Menu` 入口打开；运行页空列表需优先展示一条 `New` 占位会话，Terminal 的占位会话在首次发送输入或添加附件时才落成真实服务端会话，点击列表占位或移动端顶部 `New` 必须关闭会话抽屉并聚焦输入框，不直接显示空态卡片或提前创建服务端会话；真实 Terminal 会话在首条输入命名前也必须使用 `New` 作为默认标题。`Terminal` 与其他 `page-mode` 页面继续保持单主面板，但 `page-mode` 路由页标题上方必须稳定提供 `Menu` 入口；`760px` 及以下再进一步压缩按钮与间距，避免窄屏下出现不可触达区域。主导航抽屉必须独立承担纵向滚动，小高度视口下不允许出现菜单或会话列表被裁切且无法滑动的状态。
- 窄屏主导航抽屉点击任一路由项后需立即关闭；页面切换完成后不得继续保留旧菜单层覆盖在目标页之上。
- 窄屏主工作区按页面类型收口为贴顶起始区：普通 `page-mode` 路由页继续采用“两行头部 + 贴顶正文起始区”节奏，第一行承载抽屉入口与主操作，第二行承载当前标题；`Chat / Terminal` 在真手机宽度下统一收敛为单层运行页 workbar，左侧保留 `Menu`，中间显示“状态信号 + 当前会话标题”的单行标题按钮，右侧固定承载 `New`，通过点击真实会话标题打开 `Details`，草稿/占位 `New` 标题不触发详情，不再把 `Details` 作为独立顶部按钮或再叠一层 header。所有页面都不得在顶部遗留额外大块留白。
- 窄屏 `Chat / Terminal` 工作区顶部固定保留统一运行页入口：三条运行页都通过 `Menu` 进入左侧主导航抽屉，`New` 直接创建当前路由对应的新会话；标题区需要稳定承载当前会话名和状态信号，并作为 `Details` 的直接触发入口，不再出现移动端无导航入口、标题缺席或只能依赖正文内按钮切换会话的状态。
- `Chat / Terminal` 工作区头部固定为共享单行 header：桌面与中宽度继续保留会话标题、状态按钮和 `Details` 入口；真手机宽度则把 `Details` 下沉到中间标题按钮。`Details` 只承载会话元信息；模型、Tools / MCP 与 Skills 调整统一通过底部 Composer 工具栏的 `Session` 按钮进入。Chat 模型区除常规 Provider / Model 外，稳定提供内置 `Codex` 直选项，选中后仅影响后续消息，并把执行链显式切到 `Codex Direct`。`/chat` 不再提供独立目标选择、Deliverables、Session Profile 或独立 Skill 配置面板。
- `Chat / Terminal` 在页面从后台恢复到前台、浏览器重新把当前页激活为可见页、bfcache 恢复或网络恢复在线时，必须走同一套共享 page-activation 补偿刷新链路；`Chat` 需立即补拉会话列表、当前活动会话详情与 pending task 状态，且页面隐藏时暂停 pending task 定时轮询；`Terminal` 则需立即刷新会话列表与当前活动会话详情，使当前页状态在恢复可见后立刻与服务端对齐。
- Chat / Terminal 单会话详情默认只返回最新 `20` 个 turns，并按约 `256KiB` 的 turns 页预算控制单次响应体；长会话通过 `turn_limit` 与 `turn_before` 按批次读取。分页响应必须带 `turns_paging`，其中包含数量边界与 `byte_limit / approx_bytes`；前端刷新、轮询和恢复时按 turn/message id 合并新片段，不得因轻量响应覆盖已加载历史，也不得让后台补页改变当前阅读窗口或滚动位置。
- `Chat` 是面向用户的唯一对话入口，工作台一级入口统一为 `/chat`、`/terminal`、`/settings`；`/chat` 仅兼容映射到 `/chat`。`Chat / Terminal` 的当前活动会话稳定反映到 URL query，统一写入 `session_id=<8位短hash>`；典型入口为 `/chat?session_id=<8位短hash>` 与 `/terminal?session_id=<8位短hash>`。
- `Chat / Terminal` 首页 Composer、会话列表项与 `Details` 面板需维持同一套浅色 runtime 表面系统：Composer 采用单一胶囊式助手输入面板，主 textarea 无内边框并与底部工具行处在同一白色 surface 内；Chat 工具行不再显示 `Session` 设置入口，Chat 与 Terminal 工具行左侧继续提供无边框会话设置、附件与必要 meta，右侧收口发送动作。桌面端按主阅读宽度居中，移动端控制输入高度、底部留白和发送按钮体量，并保持输入区具备足够横向留白；Terminal 不得为 Composer 外壳覆盖更深背景、更低底部 padding 或外层状态 note 行，失败、退出与附件错误提示需进入共享工具栏 meta。PC 端上传、发送、状态、详情、流程入口与弹窗动作保持平面化，除 Composer 胶囊外不通过额外圆角、边框或厚阴影表达层级；详情面板需保留清晰标题栏、显式关闭按钮、紧凑摘要栅格和轻量字段分隔，会话列表项和详情面板不再退回旧式轻表单或松散卡片观感；空态工作区使用低对比网格与细弧线背景，同时禁止保留可拖拽滚动，不得把头部操作行或输入区顶出可视区。
- `Chat` 的桌面端草稿输入必须保持低延迟：仅因未发送草稿变化时，不得同步重建整条消息时间线、Markdown 正文或 `Process` 结构；浏览器草稿缓存允许延迟落盘，但不得影响当前输入内容、会话切换后的草稿恢复与发送结果。
- `1100px` 及以下的移动工作台需优先保证真机滚动与抽屉切换流畅度：主工作区、Conversation/Terminal 抽屉遮罩、抽屉面板本体与运行页容器不得继续依赖大面积 `backdrop-filter`、持续背景光晕或其他会导致整页重绘的装饰层，统一保持静态浅色表面。
- `Terminal` 窄屏工作区头部不得重复输出内部会话入口；`Sessions` 入口统一由壳层头部提供并打开左侧主导航抽屉，工作区头部仅保留与当前会话直接相关的操作。
- `Chat` 空态首屏在桌面与中宽度下必须保持居中首屏节奏：欢迎区标题、描述、target/prompt 需在 header 与 Composer 之间沿欢迎区中轴竖向居中展示；真窄屏继续贴近头部下沿起排。Composer 继续沿主工作区自然贴底排布；不允许通过 `margin-top: auto`、过大的欢迎区上边距或类似弹性占位把输入区推到底部，造成首屏中上部出现大块无效空白。
- 移动端 Chat 在主输入框与会话设置底部面板之间切换时，不允许保留“键盘 + 设置面板”双重底部占位：打开设置前先释放输入焦点并清理键盘偏移，回到主输入框时先自动收起设置面板；Terminal 在真手机宽度下允许工作区头部工具栏换行，操作按钮不得被长标题挤出可见区域。
- 桌面宽屏下 Chat 消息列与底部输入区需按主工作区宽度自适应扩展，并统一收敛到居中的 `960px` 最大阅读宽度，避免正文与输入区无限拉长。
- `Chat / Terminal` 统一展示右侧四键阅读定位条，承载 `回到顶部 / 上一条 / 下一条 / 回到底部`；定位目标按当前视口中的可见消息块或 Terminal turn 动态计算。`回到底部` 只在最后一条内容的底边仍位于视口外时显示，不得因为消息区尾部仅剩空白或 padding 继续保留伪底部跳转。移动端四键定位条固定停靠在工作区右侧、输入区上沿之上，四键统一使用独立圆形触达面；当当前消息滚动容器内存在有效文本选区时，定位条必须立即隐藏，避免遮挡复制拖拽与选区手柄。

## Skill & Memory

核心对象：`CLIRuntime`、`RuntimeProfile`、`Skill`、`SkillRepository`、`MCPServer`、`MemoryFile`、`MemoryContext`、`ConversationSummary`、`LongTermMemory`、`DailyMemory`、`ProjectMemory`。

稳定需求：

- Chat 是一个可执行任务的 CLI 运行时，由 Claude Code 或 Codex CLI 承担任务推理、工具调用和会话内上下文压缩。服务侧负责选择运行时、准备工作区、注入 Skill 与 Memory、归档会话结果。
- Runtime Resolver 按优先级选择执行器：已启用且健康的 Model Provider 使用 `Claude Code + provider profile`；未配置 Provider、Provider 不可用或 Claude Code 启动失败时使用 `Codex Direct`。
- Product Skill 独立维护在 `docs/skills/<skill_id>/SKILL.md`，编码、旅行、前端设计、部署预览、文档协作、测试、评审与记忆整理都以 Skill 表达执行规则和交付要求。
- 启动 CLI Runtime 前，运行时按会话工作区生成 `AGENTS.md` 或 `CLAUDE.md`，同步选中 Skill 文件、Memory 文件、MCP 配置、会话事实、工作区边界、仓库路径与交付要求。
- 代码、旅行、前端设计、部署预览、文档协作、测试、评审与记忆整理都通过 Skill 注入当前会话；执行层直接使用 Claude Code CLI 或 Codex CLI。`travel` 需把行程安排沉淀为移动端 HTML 攻略、路线化内容和按行程密度生成的 Codex 手绘地图图片资产。
- Context Files 支持根级 `AGENTS.md`、`SOUL.md`、`USER.md`、长期 `MEMORY.md`、`daily/<YYYY-MM-DD>.md`、`projects/<project>.md`、`conversations/<conversation_id>/summary.md`，并支持启动参数解析后的长期记忆文件与天级记忆目录。`AGENTS.md` 是运行规则上下文，`SOUL.md` 是强约束上下文，其余为事实型记忆。用户可见记忆文件保持 Markdown 主存，不在正文中暴露 confidence、source、status、sensitivity 等附加元数据。所有持久记忆 Markdown 均由 CLI Runtime 维护，服务侧不直接把会话轮次、压缩片段或任务摘要写入 Daily/Long-term Markdown。
- 记忆更新由三条路径触发：用户显式要求记住时由当前 CLI Runtime 写入目标记忆文件；会话结束或归档时服务生成 `ConversationSummary`；系统维护任务每日启动同一 CLI Runtime 并加载 `memory-maintenance` Skill，把会话摘要、日记忆和长期记忆合并整理。
- 会话内上下文压缩由 Claude Code 或 Codex CLI 自身处理；alter0 保存原始消息、运行日志、结果与摘要，用于恢复、审计、跨会话召回和定时记忆整理。
- `Skill -> Memory` 页面提供长期记忆、天级记忆、项目记忆、会话摘要与运行说明的只读可视化入口。

## Task, Terminal & Workspace

核心对象：`TaskSummary`、`TaskLog`、`ArtifactRef`、`TerminalSession`、`TerminalTurn`、`RuntimeTraceEvent`、`Workspace`、`CodexThreadID`。

稳定需求：

- Terminal 页面 Composer 支持最多 5 个附件，稳定覆盖图片与常见文本/文档文件：图片继续提供缩略图预览、纯图片发送与图片回显，并支持 PC 输入框内直接粘贴剪贴板图片；图片先写入当前 Session 工作区附件目录后仅提交 `asset_url / preview_url` 引用；缩略位使用预览图，但 turn 历史与后续预览弹层再次查看时必须优先读取原图资源。普通文件同样先落到同一附件目录并只提交稳定附件引用，执行前再写入当前 Terminal 工作区 `input-attachments/<turn_id>/` 供 Codex 按路径读取。Terminal 当前活动会话的 shell 明确为 Codex 时，输入 `/` 需显示 Web 适用的 Codex CLI 斜线命令候选并支持点击补全；候选按命令作用分组顺序展示，并使用短动作说明；权限、TUI 显示、键位、剪贴板、登录退出和本地 CLI 会话管理类命令不进入候选，普通 shell 会话不显示 Codex 候选。Terminal 输出正文、Markdown 正文与代码结果必须保留浏览器原生文本选择能力，用户可直接手动选中并复制局部输出；移动端最终输出不得安装脚本长按选区、假选中态、浮动复制层、`contenteditable`、隐藏输入框或键盘编辑态兜底；阅读定位 overlay 不得截获正文拖选或长按选中。Terminal `Details` 面板支持选择控制面中启用且非私有的公有 Skill；新 Terminal 会话首次加载时默认勾选全部可用公有 Skill，仅排除 `memory`，并在发送输入时把当前 `skill_ids` 编译进 Terminal 工作区的原生 Codex Runtime。该运行时同样必须通过托管 `AGENTS.md` 与 `runtime_context` 约束 Codex 仅操作当前 Terminal 工作区及其派生文件，不得顺带修改其他会话、服务或工作区外仓库。
- Web 会话不直接暴露本地文件路径。
- 默认工作区按执行上下文隔离：Chat 使用 `.alter0/workspaces/sessions/<session_id>`，其中 Chat 的逻辑 `session_id` 固定为 `alter0-chat`；Task 使用其会话下的 `tasks/<task_id>`，Terminal 使用 `.alter0/workspaces/terminal/sessions/<terminal_session_id>`。
- Chat 的会话图片资产需要随 Session 工作区落盘：用户上传图片的原图与预览图统一写入 `.alter0/workspaces/sessions/<session_id>/attachments/<asset_id>/`，前端持久化与消息请求默认复用 `asset_url / preview_url` 引用；其中 `preview_url` 只服务缩略位，消息回显与再次查看统一优先读取 `asset_url` 原图。assistant 最终回复里的外链 markdown 图片也应在会话返回与落库前改写到同一路径下的本地附件 URL。
- 直连 Codex 的 Chat 会话会在各自工作区下额外维护 `.alter0/codex-runtime/` 与 `.alter0/codex-runtime/codex-home/`；Chat 使用 `.alter0/codex-runtime/thread.json` 保存 Codex CLI thread id，Chat 使用 `.alter0/codex-runtime/threads/<YYYY-MM-DD>.json` 保存北京时间 05:00 归档日对应的 Codex CLI thread id；Terminal 会话会在 `.alter0/workspaces/terminal/sessions/<terminal_session_id>/codex-home/` 下维护独立 `CODEX_HOME`。
- Terminal 是独立会话式终端代理，持久化 Codex CLI 线程标识、会话状态、标题、工作区、日志与 `RuntimeTraceEvent` 视图索引。
- Terminal API 支持会话创建、列表、恢复、输入、删除、详情读取以及 turn/runtime event 明细读取，前端可按事件展开或检索执行细节。
- Terminal 会话态统一为 `ready / busy / exited / interrupted`，执行态在 turn/runtime event 维度维护 `running / completed / failed / interrupted`；运行态退出后保留历史，继续发送即可恢复。
- Terminal 与 Chat scope 会话状态持久化统一使用 `runtime_events` 与 `next_event_id`；读取旧 `steps / next_step_id` 状态文件时会同步迁移并写回新结构。
- Terminal 恢复默认优先复用已持久化 Codex CLI 线程；若续写命中远端 compact 失败，则保留原会话历史、工作区和线程标识，下一次输入继续 resume 同一 Codex CLI 线程。
- Terminal 会话删除统一从左侧会话列表触发，`Delete` 会同步清理状态文件和独立工作区；工作区头部不再提供单独的 `Close` 入口。删除成功后，无论删除的是历史会话还是当前活动会话，当前会话列表所在的左侧导航抽屉都保持删除前的展开状态，便于继续清理其他会话；用户随后通过 `Menu` 或抽屉外部遮罩主动关闭时，抽屉必须正常收起。前端在后续列表刷新、轮询和 page-activation 补偿刷新中也不得把该会话重新补回，直到服务端列表稳定反映删除结果。
- Terminal 历史在同一 Web 登录态下跨设备共享，不按浏览器 client 标识隔离；不设置产品级会话数量上限或固定超时淘汰。
- Terminal 移动端、输入稳定性、滚动导航、Process 折叠、一键复制、长输出阅读、轮询降频与缓存写入节奏作为 Terminal 子域体验要求维护。
- Terminal 四键阅读定位条按当前视口中的可见 turn 集合计算目标：`上一条` 固定指向最上可见 turn；`下一条` 在单条 turn 可见时指向真实下一条、在多条 turn 同屏可见时指向最下可见 turn；最后一条 turn 单独可见时隐藏 `下一条`。
- Terminal 发送按钮首次点击必须立即进入 pending 反馈；若当前还没有 active session，前端允许先创建会话再继续发送，但首击期间按钮需同步切到 `Sending...` 与禁用态，避免重复点击和“第一次点击无反应”的错觉。
- Terminal 刷新节奏需按会话状态自适配：执行中的会话保留实时刷新，空闲会话停止周期轮询并依靠页面激活补偿刷新；用户正在滚动阅读输出时，不得因明细轮询而打断当前滚动。
- Chat / Terminal 在同一浏览器工作台内切换到其他页面后再返回时，前端需优先使用未过期的 24 小时运行态内存缓存恢复会话列表和当前活动会话的已加载内容；`Chat` 缓存完整已加载消息，`Terminal` 缓存完整已加载 turns，接口返回后继续合并更新。缓存不作为跨设备或服务端事实来源；每次访问、切换、刷新或前台恢复都会按最新合并结果刷新缓存时间。
- Terminal 窄屏消息页必须保持 `workbench-main -> chat-pane -> terminal-view -> terminal-chat-screen` 的闭合高度链，由 `terminal-chat-screen` 独立承担纵向滚动；外层容器不得因 `overflow: hidden` 或高度塌陷吃掉滚动。
- Terminal 移动端在输入框聚焦且软键盘抬起后，Composer 必须通过 `position: fixed; bottom: 0` 直接贴住动态视口底边；长对话或长输出期间不得通过拉高 footer padding、改变滚动容器或破坏高度闭合链把输入区挤出屏幕。
- Terminal 移动端的 `terminal-chat-screen` 必须继续按当前 Composer 的真实遮挡高度动态收口；会话空态、长输出与 Process 阅读都要稳定停在输入区上沿，不允许被底部 Composer 覆盖。
- Terminal 移动端的命令与 prompt 气泡需保持自然整词换行；路径、flag 和短 shell 片段优先按空格或真实长单词边界断行，不允许因窄屏收缩把命令压成逐字或逐 token 的碎行。
- Terminal `Process` 的步骤头必须保持稳定三列：左侧独立展开图标列、中间标题主列、右侧耗时与状态列。标题只能在中间主列内截断，不允许因为节点缺失、DOM 顺序错误或 grid 列错位把标题挤进图标列，导致移动端只显示单个字符。
- Terminal `Process` 展开后的自然语言步骤详情需使用同一套阅读修正：`reasoning / plan / message / text` 等说明类内容优先按 markdown 正文块整列换行，展示前移除零宽断行字符，并把“每字一行”的病态段落归一回可读正文；仅终端输出、diff 与代码类块继续保留预格式化渲染。
- Chat 的 `Thinking / 已思考` 外层披露展开时只进入步骤列表态；该动作会收起同一 assistant 消息下已打开的单步详情，避免移动端把历史详情重新展开并造成视口突跳。单步详情仍由用户点开具体步骤后展示。
- Chat 与 Terminal 的过程详情必须共用同一套最终 detail surface 渲染规则：终端、代码、diff、tool input 与 JSON 类输出直接使用等宽内容块，说明、markdown、thinking、文本型 tool output 与 error 直接使用富文本正文块，并保留 block 标题、文件名与起始行号；Terminal 中需要单独拉取 detail 的步骤，必须在 detail 返回后再展开步骤体，不允许先显示 preview 兜底再二次跳变。
- Terminal 移动端的四键阅读定位条只按静态 Composer footprint 停靠，不跟随软键盘位移动态上移；输入框聚焦且键盘弹起时按钮组主动隐藏，键盘收起或浏览器视口回弹后再恢复到 Composer 上沿之上，不得留下悬空残影。

## Control, Operations & Governance

核心对象：`ChannelConfig`、`SkillConfig`、`RuntimeProfile`、`ModelProvider`、`ClaudeProviderProfile`、`CodexAccount`、`CodexLoginSession`、`RuntimeInstance`、`DeploymentBaseline`、`EngineeringPolicy`。

稳定需求：

- Control API 管理 Channel、Capability、Skill、MCP、Runtime Profile、Cron Job、Model Provider 与 Codex Runtime 配置，并保留 Capability 生命周期审计。
- 服务启动后默认提供 `memory`、`preview-publish`、`frontend-design`、`doc-coauthoring`、`fullstack-developer`、`code-reviewer`、`webapp-testing`、`find-skills`、`test-driven-development`、`ui-ux-pro-max`、`code-simplifier`、`code-review`、`brainstorming` 与 `travel` 公有内置 Skill；`memory-maintenance` 作为系统维护专用私有 Skill 保留，不进入 Chat / Terminal 常规选择列表。这些 skill 均由源码仓库直接承载在 `docs/skills/` 下，其中标准 skill 继续使用 `docs/skills/<skill_id>/SKILL.md` 作为 file-backed 入口，plugin-style 的 `code-simplifier` 与 `code-review` 则分别以 `docs/skills/code-simplifier/SKILL.md` 和 `docs/skills/code-review/commands/code-review.md` 作为 alter0 注入入口。Codex 运行前必须把本轮选中的可读 file-backed Skill 目录物化到当前会话工作区 `.alter0/codex-runtime/skills/<skill_id>/`，并将运行时 `file_path` 改写为物化后的工作区内路径。`preview-publish` 是静态用户可见产物与完整测试服务的统一发布通道；HTML、Markdown 预览、截图、图片集合、文本报告、JSON 示例和代码样例必须先发布到 `https://<service>-<short_hash>.alter0.cn`，不得把 `/srv/...`、`.alter0/workspaces/...`、`file://`、`localhost` 或 `127.0.0.1` 作为用户验收链接。服务不再注册内置业务编排；Chat 默认直接通过 Claude Code CLI 或 Codex CLI 执行，所有可复用规则都通过统一 Skill 目录和当前会话 Skill 选择进入运行时。
- 共享 Web 运行时需要支持通用 workspace service 注册：`GET /api/control/workspace-services` 查询注册表，`PUT /api/control/workspace-services/{session_id}` 绑定默认 `web` 服务，`PUT /api/control/workspace-services/{session_id}/{service_id}` 绑定附加服务，`DELETE` 接口用于清理绑定；当请求 Host 命中 `<session_short_hash>.alter0.cn` 或 `<service>-<session_short_hash>.alter0.cn` 时，共享运行时需按注册类型分发前端构建或反向代理到目标 HTTP 服务。`travel` 服务是唯一例外，固定命中 `https://travel-<session_short_hash>.alter0.cn`，且该 host 只读、免登录，只允许返回静态 HTML/资源。标准 `web` 部署默认应把当前会话后端启动命令注册给共享运行时托管，再以 `http` 方式绑定短哈希子域名，确保前端与 `/api/*` 同时来自当前分支；`frontend_dist` 仅作为静态预览模式保留。
- Channels 入口归属 Settings 模块，旧直达路由保持兼容。
- Models 控制面支持 Claude Code provider profile 配置，包含 base URL、API Key 保留语义、model、profile、Provider 路由偏好、默认项自动收敛与历史缺密钥配置恢复；启用且健康的 Provider 作为 Claude Code 首选运行来源。
- Codex Runtime 作为 `Codex Direct` 的账号与模型管理来源，在无可用 Model Provider 或 Claude Code 运行失败时承接自然语言任务兜底执行。
- Runtime 设置页支持在线实例启动时间与 commit hash 展示、运行时重启、默认启用的远端 master 快进同步、仅在后端检测到 Git 已跟踪本地改动后才触发的二次确认、确认后丢弃已跟踪改动的重启前同步策略、通过统一前端感知构建入口生成候选二进制、readyz 探活与失败回滚。旧运行参数配置页、环境变量可视化配置、队列、终端 shell、记忆路径参数均不再对用户暴露。
- Settings 页面提供 Codex Runtime 面板，使用单一顶部面板承载当前服务运行账户的 Codex 身份快照、邮箱、计划、认证模式、hourly / weekly 额度、profile、LLM Provider 注册状态，以及基于 Codex app-server 真实能力返回值的活动 model / 思考深度切换。首屏加载时 Codex Runtime 状态与 LLM Provider 状态需并行读取。页面支持为当前运行账户启动 Codex device-code 登录，并展示验证链接、用户码、过期时间、轮询间隔与登录输出；登录成功后刷新 Runtime 身份与额度。页面同时支持通过 Claude Code Provider Console 连续注册与编辑多个 OpenAI-compatible Provider；桌面端 registry 与 editor 在同一容器内左右分栏，窄屏单列展开。字段包含 Provider 名称、base URL、API key 与 models；models 使用全宽多行编辑区，支持换行或逗号分隔，提交后写入 Model Provider 注册表，首个 model 作为默认模型，并刷新 Provider 状态。已注册 Provider 需展示名称、base URL、默认 model、模型数量、模型列表与启用/默认状态；编辑时 API key 留空表示保留已保存密钥。每次注册或更新成功后表单清空 base URL / API key / models，并自动准备下一个未占用的 `Claude Code N` 默认名称。页面不展示 Account ID / User ID、保存名称、多账号导入/切换入口、CLI 命令、auth/config 路径、诊断侧栏或由 auth/config 文件存在性推导的 Ready/Status 文案。额度必须来自当前 `auth.json` 的实时 quota 刷新结果，model / 思考深度选择变更后仅实时写回当前用户配置中的 `model` 与 `model_reasoning_effort`。
- 公网部署基线要求服务绑定 localhost、启用 Web 登录密码、统一 `HOME=/var/lib/alter0`，并通过 Nginx 做反向代理。
- 服务内 GitHub 交付要求运行账户具备 GitHub App token helper、`gh` 包装器、SSH 提交签名、稳定 PATH 与 Codex CLI 可用认证。
- Node/Playwright 测试链路通过运行账户级工具链初始化，保证 Codex CLI 可执行 `internal/interfaces/web/frontend` 的构建与单测，以及 `internal/interfaces/web` 的 Playwright E2E。
- 研发流程遵循 TDD：功能新增、缺陷修复、行为调整与重构默认先以测试表达目标行为，再完成实现与重构；纯文档、注释、格式化、依赖元数据或无法自动化验证的变更需说明免测原因与替代验证。

## 维护规则

- 新需求必须先选择领域，再选择子域；无法归类时优先补充领域模型，而不是新增线性编号。
- 一个需求只允许有一个主归属领域；跨领域影响通过 `依赖与边界` 说明，不复制成多个重复需求。
- 用户可见行为、交互方式、入口路由、执行模式、返回结构或默认策略发生变化时，同步更新 `README.md`。
- 需求细节、接口、状态、验收和边界放入对应 `requirements-details/*.md` 文件；技术方案、包边界、调用链路、存储、观测和测试策略放入 `technical-solution.md` 的同名领域下；`requirements.md` 只维护稳定总览与领域索引。
