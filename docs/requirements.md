# Requirements

> Last update: 2026-07-11

`alter0` 的需求清单按领域模型维护。后续新增需求不再使用线性编号，也不按提交顺序堆叠；需求应落到对应领域、子域与能力项下，使用稳定领域路径表达，例如 `runtime.execution.cli-runtime`、`memory.files.injection`、`task.workspace.runtime`。

稳定默认约定：

- Web 前端所有需要可见时间的设置视图、会话列表、详情面板与任务视图统一固定为上海时间（`Asia/Shanghai`）与 24 小时制；`Chat` 的消息正文区不显示逐条消息或 turn 时间。
- 自然语言任务默认通过 `Codex Direct` 执行；显式选择具体 Provider/Model 或声明 Claude 执行器时使用 `Claude Code + provider profile`，Claude 执行失败不自动回退。
- 服务侧负责会话、工作区、Skill 仓库与结果归档；所有 Codex 任务共享当前活动 `CODEX_HOME`，跨会话记忆统一由 Codex 原生 Memories 生成、精炼与召回。
- 系统维护任务只保留会话清理：Scheduler 每日清理超过 7 天不活跃且未置顶、无 queued/running 任务关联的会话，置顶会话始终跳过自动清理；旧记忆维护任务不再注册。

状态说明：

- `supported`：已在主干代码可用
- `in-progress`：正在落地，接口或行为尚未稳定
- `planned`：已确认方向，待排期

## 领域索引

| 领域 | 范围 | 状态 | 细化文档 |
| --- | --- | --- | --- |
| Runtime & Orchestration | 输入通道、统一消息、意图路由、CLI Runtime 选择、调度、存储、观测 | supported | [runtime-orchestration.md](requirements-details/runtime-orchestration.md) |
| Conversation & Session Experience | Chat 运行入口、历史、移动端、阅读与输入体验 | supported | [conversation-session-experience.md](requirements-details/conversation-session-experience.md) |
| Skill & Memory | Codex 用户级原生 Skill 生命周期、共享 `CODEX_HOME` 与全局原生 Memories 设置 | supported | [skill-memory.md](requirements-details/skill-memory.md) |
| Task, Chat & Workspace | 异步任务、任务观测、任务日志、产物交付、Chat 会话、独立工作区 | supported | [chat-runtime-workspace.md](requirements-details/chat-runtime-workspace.md) |
| Control, Operations & Governance | 控制面配置、Model Provider、Claude Code provider profile、Codex Runtime、服务重启、部署基线、认证凭据、TDD 研发约束 | supported | [control-operations-governance.md](requirements-details/control-operations-governance.md) |

## Runtime & Orchestration

核心对象：`UnifiedMessage`、`OrchestrationResult`、`Intent`、`Command`、`ExecutionPort`、`RuntimeResolver`、`CLIRuntime`、`SchedulerJob`、`Channel`、`TraceContext`。

稳定需求：

- CLI、Web、Cron 等输入源统一转换为 `UnifiedMessage`，并携带 `message_id`、`session_id`、`trace_id`、`channel_type`、`trigger_type` 与业务载荷。
- 编排层负责意图识别与路由：命令进入 `CommandRegistry` / `CommandHandler`，自然语言进入 `ExecutionPort` 后由 `RuntimeResolver` 选择 CLI Runtime；Cron 触发复用同一编排链路；当消息显式指定 `alter0.execution.engine=codex` 时，斜线前缀输入视为 Codex 原生命令或提示词，直接进入 `Codex Direct`。
- 命令能力稳定提供 `/help`、`/echo`、`/time` 与 `/now`。
- 自然语言执行通过 CLI Runtime 完成：启用且可用的 Model Provider 对应 `Claude Code + provider profile`，无可用 Provider 或 Claude Code 运行失败时进入 `Codex Direct`。
- 调度领域支持 Cron Job 配置、内置不可删除 Job、可视化周期字段、触发记录、触发会话归档、runs 查询与来源回链；内置 Job 允许切换 enabled 状态。
Codex 原生 Memories 数据由活动 `CODEX_HOME` 管理
- 观测能力覆盖结构化日志、Prometheus metrics、`/healthz`、`/readyz` 与 trace/session/message 维度。

## Conversation & Session Experience

核心对象：`Session`、`Message`、`LiveUserMessage`、`RuntimeTraceEvent`、`ViewportState`、`SessionHistoryBucket`。

稳定需求：

- `Chat` 是唯一对话入口，通过全局 Codex Skills、Memories 与会话 Model 选择承载代码、旅行、写作等专项任务；不提供会话级 Skill、Memory、Tools 或 MCP 选择。显式选择 `Codex` 时，Web 对话框支持 Codex CLI 内置斜线命令候选。
- Web 登录态下，Chat 会话历史统一进入服务端 Session history；历史运行时会话在读取阶段迁移为当前 Chat 会话结构，并统一通过 `route=chat` 进入列表与详情恢复。
- `Chat` 运行页使用后端生成的短 canonical id 作为会话级引用、显式 URL 恢复参数与人工排障的稳定标识符；该 id 格式为 `c_` 加 16 位小写字母数字，并直接用于接口、持久化和工作区隔离。左侧会话列表不展示该 id。`Chat` 主入口在无显式 `session_id` 时默认打开最新 Chat 会话，主导航切回 Chat 时不得沿用旧 query 或浏览器上次活动会话锚点。
- Web 入口稳定提供根路径到 Chat 的默认进入、`/chat`、`/settings`、`/login` 与 `/logout`。受保护页面、受保护预览工作区与 API 统一走同一登录态校验。访问工作台页面触发登录时，登录回跳只保留 canonical path，不携带会话级长 query。仅静态只读 host 保留匿名访问。
- 新会话标题由底层 ChatRuntime session store 生成与维护；前端创建真实会话时不得把首条 prompt 作为 create title，也不得在本地乐观消息阶段改写标题。底层会话先使用统一占位标题 `New`；Codex/Claude 等外部 CLI Runtime 暴露自身 thread title 时，ChatRuntime 必须优先采用该 title，持久化为外部标题状态，并通过 `session.updated` 同步到前端。后续同一底层 thread 多次更新 title 时继续覆盖当前会话标题；未收到外部 thread title 时，早期多轮内可根据更具体输入自动升级标题，避免长期保留“拉取仓库”“分析仓库”等低辨识度名称。
- 新对话空白会话保持唯一；已有空白会话时，`New` 复用并聚焦该会话。
- `Chat` 左侧会话列表中，置顶会话固定进入 `Pinned / 置顶` 分组；距离 7 天不活跃清理阈值还剩不超过 2 天的未置顶会话进入 `Expiring Soon / 即将清理` 提醒分组；其余会话按内容更新时间归入 `Today / 今天`、`Yesterday / 昨天`、`Earlier / 更早`，缺少更新时间时才回退到创建时间。
- 同一会话内请求保持顺序一致，由当前 Chat owner 的 runtime session 状态表达 `busy / ready / failed`。
- `Chat` 创建真实会话与恢复已有会话时，服务端必须先写入 owner session store，再在锁外发布更新事件和构建 bounded detail；前端 `New` 入口不得因为事件发布链路回读会话详情而长期停留在无反馈状态。会话列表接口必须能补扫持久化状态目录，把服务重启、部署切换或详情懒恢复前已存在但尚未进入内存 map 的 owner 会话重新纳入列表，避免用户刷新后只看到本地缓存、看不到服务端中断态或历史会话。
- `Chat` 消息入口在请求进入执行链前必须先持久化本轮 `user` 消息；assistant 消息、route、错误码与结构化步骤在执行完成后追加落库，避免浏览器关闭、刷新或请求断开时已发送用户消息只存在于前端乐观状态。
- Codex 返回 `no rollout found for thread id` 时，该 Chat 会话必须收敛为不可继续的失败态，不得清空 thread id 后在原会话内静默重建。服务端需将仍已持久化的逐轮用户消息、附件名称与助手最终回复导出为独立 Markdown 历史文件，并在失败消息中提供包含绝对文件路径、可直接复制到新会话发送的续接指令；导出内容不得把过程日志伪装成助手最终回复。
- `Chat` Composer 必须提供独立于附件按钮的 GitHub 仓库图标。发送前的搜索与勾选只保存一个本地仓库草稿，并按 route/session 使用独立于附件的本地缓存 key，不创建真实会话、不访问仓库、不拉取代码；首次发送时仓库稳定引用与用户消息一起提交，服务端通过当前个人 `gh` 登录重新解析并在 Agent 启动前 clone 到固定相对目录 `repo/`。一个会话最多绑定一个仓库，绑定后不可替换；准备失败可在同一 turn 原地重试。绑定不自动 pull/fetch/reset、commit 或 push，也不把 token、clone URL 或服务端绝对路径暴露给前端和 Agent prompt。
- `Chat` 在同一会话内保持追加式历史：每次发送都新增一条 `user` 消息与对应 assistant 消息，发送后当前活动时间线需立即回到底部展示本轮新增消息；后续结果或会话详情恢复只允许补丁当前未完成的 assistant 条目，已收口历史不得被迟到结果回写。
- 请求断开或页面刷新后，前端优先回源当前会话详情，用服务端已持久化的消息覆盖本地占位态，而不是立即把同一请求重发一遍。该恢复链路即使在服务端集合接口已返回当前会话摘要、但尚未附带完整消息时也必须继续触发。只有恢复失败时才收敛为失败态；若没有可用正文，失败提示需明确提示刷新。本地缓存中残留的 `streaming` 消息不得长期停留在 `In Progress`。
- `Chat` 实时更新采用“HTTP 快照 + owner 级增量轮询 + bounded detail 兜底”模型。增量轮询通过 `POST /api/chat/sessions/updates` 读取当前 owner 在 `after_update_id` 之后发生的 session/turn/event 变更，响应返回 `latest_update_id`、`resync_required`、可选 `has_more` 与 `updates[]`，并通过 `limit` 与默认/最大 `1MiB` 的 `byte_limit` 控制体积；前端不建立长连接。运行态变化由 `chatruntime/application` 在输入接受、runtime step 追加/更新和 turn 收口时发布 `turn.started / turn.event.appended / turn.event.updated / turn.completed / turn.failed / turn.interrupted` 语义事件，单条 update 固定使用 `update_id / type / session_id / turn_id / runtime_event` 顶层结构，必要时携带 session 摘要与目标 turn patch。被用户过滤隐藏的 command/tool/system 等纯过程事件可不下发，但 `latest_update_id` 必须继续推进，前端可见 `update_id` 允许不连续。updates payload 只包含前端消费字段，不携带 `owner_id / shell / working_dir / runtime_session_id / last_output_at / activity_at` 等运行元数据；`session.updated` 只用于恢复、标题、置顶等会话级摘要变化，input 接受后的 busy 状态由 `turn.started` 携带，常规 runtime step 增长不得再靠全量 session 快照表达。自动轮询只覆盖 `local_running / recovering` 会话；LLM 执行期间 updates 连续空返回、只命中无关 backlog，或当前待恢复会话的 `updated_at / messages / process steps` 均未推进，属于正常无进展窗口，前端按连续无进展次数退避：updates 轮询从约 `2s` 起步，连续无进展后退到约 `3s / 5s / 8s`；bounded detail 只在第 6 次连续无进展补拉一次，之后每 8 次补拉一次；当前会话收到新的 `updated_at`、消息或过程步骤时必须重置退避计数，详情仍未收敛时继续保持退避计数。会话进入 `ready / failed / interrupted / exited / deleted` 等终态后停止自动 updates 和 detail 请求；跨设备变化通过显式手动刷新、切换会话或下一次真实运行态恢复。首次进入和刷新时，当前 active server session 即使已命中完整本地缓存，也会在首屏恢复后补拉一次不带 `turn_before` 的最新 bounded detail；稳定 `ready` 会话在普通 focus/pageshow/online 激活时不自动请求列表、详情或 updates。
- `Chat` 的会话状态交互以服务端业务状态为准：发送成功进入服务端 input handler 后，当前活动会话和左侧对应会话项必须立即进入处理中；当前 owner 下只有本机产生的 `local_running / recovering` 会话通过同一 owner 增量轮询同步更新标题、loading 与最终状态，普通服务端 `busy / running` 摘要不触发后台自动轮询。网络失败、页面刷新和 bfcache 恢复不触发失败文案；失败、中断和删除只由服务端会话详情或明确增量事件决定。
- 若当前活动会话的服务端历史只包含最新 `user` 消息且尚无对应 assistant 或失败消息，前端必须继续按单会话详情恢复，不得把该 user-only 历史判定为稳定完成态。
- 刷新页面时，`Chat` 必须优先保住当前活动会话：若服务端会话列表暂时尚未返回该 `session_id`，前端先从浏览器侧活动会话快照恢复当前条目与最近消息，再按 `session_id` 单独回源详情；若集合接口返回空列表、短列表或短于本地已追加历史的消息历史，前端不得用较短远端结果清空本地可见会话、active session 或当前时间线；在确认服务端不存在该会话前，不得直接把当前活动会话替换成新的空白 `New` 会话。
- 刷新页面或切到其他会话后，`Chat` 仍需保住最近已知会话列表：浏览器侧最近会话快照至少覆盖当前活动会话之外的最近若干条会话；当服务端集合接口暂时漏掉其中某条会话时，左侧会话列表不得立刻把该会话删除，而应继续保留本地条目并等待单会话详情或后续集合结果确认。
- `Chat` 前端缓存需保留当前已加载会话的完整消息或 turns：同一工作台内的 24 小时运行态缓存不得裁剪已加载历史；浏览器侧按 `chat` 分别维护独立 `sessionStorage / localStorage` key，覆盖 active session、文本草稿、附件草稿、模型/Provider、Tools/MCP、Skills、过程披露过滤、24 小时完整消息快照与轻量会话信息快照，用于刷新、重开或 `sessionStorage` 丢失时优先恢复当前 route 首屏。完整消息快照与轻量会话信息快照读取时按 session 合并：同一 session 优先保留完整消息、过程事件、`turns_paging`、`updated_at` 与本地详情新鲜度，轻量快照补齐缺失的会话列表条目，不允许因存在完整快照而丢弃其他已知会话摘要。缓存只作为本地加速与断网前置恢复，服务端 Session history 与后续单会话详情回源仍是最终事实源；首次访问和刷新会在当前 active server session 上补拉最新 bounded detail 校准缓存，普通前台恢复只刷新本地缓存时间，不对稳定 `ready` 会话自动回源。若本地完整历史已确认 `turns_paging.has_more_before=false`，后续最新 bounded detail 或增量事件不得把本地分页状态重新打开；即使远端最新页自身带有 `has_more_before=true`，也只能在用户显式加载更早历史且本地规范分页状态仍允许时请求 `turn_before`。
- `Chat` 的会话存在性、配置与恢复状态需由 Chat session store 承担第一责任：输入入口在请求开始、完成、失败时分别写入 `busy / ready / failed` 等稳定状态；会话置顶、删除、列表、详情与恢复均复用 Chat session API，且列表、详情和置顶响应必须显式返回 `pinned` 布尔值，包括取消置顶后的 `false`。前端展示、计数和发送 payload 需以当前可用公有 Skill 目录过滤后的有效选择为准，避免因浏览器刷新、请求断开、前端本地状态丢失或 Skill 目录变化导致会话“消失”、失效 Skill 继续注入或直接 `load failed`。
- 运行时执行过程需以统一 `RuntimeTraceEvent` 数据模型承载，并覆盖 Chat turn、Chat input 结果与会话历史持久化。当前 owner 的 turn 摘要需直接输出轻量 `runtime_trace_events` 作为过程展示的 canonical 数据；轻量结构只保留 `id / kind / status / text / detail_available / created_at / completed_at / duration_ms`，删除 `seq / session_id / turn_id / provider / source / role / lifecycle / title / summary / visibility / raw / action / blocks`，其中时间字段均为毫秒时间戳，`completed_at` 无值时为 `null`。`kind` 枚举固定为 `important_text / reasoning / plan / tools / commands / system`。事件详情通过当前 route owner 的 `session_id / turn_id / event_id` 索引读取，并只在用户展开具体 `Thinking / Process` 步骤时懒加载完整 detail，结构化 `blocks` 只在 event detail 接口返回，首屏与会话详情分页不得提前返回大段 thinking 明细。用户已加载的步骤 detail blocks 必须继续保留在前端消息缓存与完整快照中，后续轻量详情、列表摘要或增量 patch 不得把该步骤降级回空 blocks 或重新标记为待加载。Codex `agent_message` 必须按 `channel` 分流：`commentary` 属于可见过程文本并通过 updates 增量进入 `RuntimeTraceEvent`，`final` 或无频道内容只作为最终 assistant 正文收口，不得先作为过程 step 发布再在完成时删除。前端优先消费结构化事件，并仅按 `RuntimeTraceEvent.kind` 执行过程披露过滤，不依赖解析自然语言过程文本。
- 消息区支持 Markdown 安全渲染、一键复制最终回复、Process 折叠状态、逐条 patch 与逐帧合并刷新；Chat 最终输出统一使用稳定的 `MessageMarkdownShell` 承载，正文先于复制工具栏渲染，不安装脚本长按选区、假选中态或编辑态兜底，且父级无关重渲染不得重写相同 markdown 的文本 DOM；React 托管的普通页面也需对正文型字段提供同一安全 Markdown 渲染能力，覆盖 Task 请求/结果/日志/产物摘要、Control 描述、Cron 输入、Skill 说明、Codex 运行时说明与 Session Profile 非等宽字段。Markdown 视觉需保持正文阅读节奏：ATX/Setext 标题紧凑、段落自然、删除线和自动链接按正文渲染、嵌套列表按 Markdown 缩进保留真实层级，列表项内允许继续承载引用与代码块，普通链接显示外链箭头，代码块保留浅灰弱边界；Markdown 表格需渲染为真实表格结构并保留列对齐，只保留横向分割线、无外框卡片和表头灰底，短表格至少铺满消息宽度，普通长文本在单元格内自动换行，链接、URL 与代码保持不硬断开，只有真实不可断内容超宽时才在消息容器内滚动；ID、路径、密钥、配置值、时间戳等元数据字段继续按纯文本或等宽字段展示。
- Chat 允许通过显式预览参数 `/chat?markdown_demo=1` 临时覆盖当前时间线视图并展示一条非持久化 assistant Markdown 语法覆盖样例；样例覆盖 ATX/Setext 标题、段落换行、强调、删除线、自动链接、图片、引用、嵌套列表、任务项、列表内引用与代码块、分割线、代码块、对齐表格与 raw HTML 转义。表格样例覆盖短字符、长中文、长 URL/代码和混合内容场景；折叠示例中的 HTML 标签按代码块展示，折叠内容本身按普通 Markdown 展示。该入口只用于渲染验收，不写入会话历史，也不替代真实 Chat 会话恢复规则。
- `Chat` 的消息阅读结构统一采用轻量 IM 式消息流：用户消息右对齐且使用浅灰低对比紧凑气泡，气泡高度需由较小纵向 padding 与独立消息行高控制；助手消息左对齐并弱化为无边框正文阅读流，Chat 正文工作区使用白底无框阅读面，不在消息区叠加明显面板边框、背景分界或卡片容器；`Process` 默认收敛为 `Thinking / 已思考` 内联轻量披露行，只显示步骤数量，不显示耗时，展开后在当前消息内展示步骤详情，步骤行需显示与过程披露过滤一致的类型标签、耗时与状态，移动端也保持同页内联展开，最终 Markdown、图片与复制动作都收敛在对应消息区域内。同一条 assistant 消息同时存在最终正文和过程事件时，`Thinking / 已思考` 披露入口先于最终 Markdown 渲染，最终正文和复制动作位于其下方。模型与 Skill 配置面板需提供过程披露过滤勾选，默认勾选 `important_text` 与 `reasoning`，确保 commentary 与 thinking/reasoning 步骤默认可见；`plan / tools / commands / system` 等事件只有在用户显式勾选后展示。`/chat` 使用同一 Conversation runtime Markdown shell，但 route owner、API、缓存、草稿和活跃会话状态必须按 `chat` 分离；最终正文必须是静态可选中文本，复制动作位于助手正文下方，代码块作为独立浅灰内容块呈现；消息正文区不显示逐条时间，只有进行中、排队、失败等状态保留紧凑状态标签；长历史默认优先渲染最新上下文，用户滚到顶部或点击 `Load earlier messages / 加载更早消息` 后按批次渐进显示本地已加载消息，两种触发方式必须共享同一批次扩展和滚动坐标恢复，滚动触顶的连续事件在恢复完成前只允许触发一次扩展；发送新消息后的轻量详情不得截断当前已加载历史。Conversation runtime 前端会话生命周期、详情刷新、渐进补历史、输入、附件、置顶、删除、事件明细、turns/paging 合并、turn-to-message 转换、timeline item 构造与 model / Skill / MCP catalog 加载必须走同一套 runtime session controller、runtime session view model、timeline builder 和 catalog hook，并由当前 route 决定 owner API 与本地存储命名空间。
- 专项 Skill 需显式声明 deliverables contract，作为底层 Skill 执行上下文中的最终交付物约束。
- Skill 还需支持独立的 `completion_checks` 机器规则，用于把交付契约下沉为可执行的运行时产物检查。`deliverables` 负责用户可见契约与 prompt 约束，`completion_checks` 负责文件存在、公开 URL、workspace service 发布状态、Session 属性非空等确定性校验，并可在失败时声明一轮仅面向当前 Session 的 Codex 修复指令。
- `Chat` 的 `Process` 步骤在真机窄屏下仍需保持整列阅读宽度；步骤序号、展开图标、标题与状态信息需在同一行垂直居中；长中文说明、路径、错误日志、inline code、Markdown 表格和命令明细必须在消息容器内自然换行或仅在内容块内部横向滚动，不得塌缩成逐字竖排窄列，也不得制造页面级横向滚动；展示层还需容忍零宽断行字符和“每字一行”的异常历史文本，并在渲染前修正为可读段落。
- `Chat` 的消息时间线在内容较少或加载中时仍需顶部收口：少量消息、短回复、折叠后的 `Thinking / 已思考` 披露行、状态标签与用户消息继续贴近各自消息气泡并按时间线顺序起排，不得被满高布局居中或拉出大段垂直空白。消息视口需根据实际内容溢出状态开启或关闭纵向滚动；内容不足一屏时不得保留可拖拽的空滚动层或 iOS 惯性回弹，内容溢出后再恢复原生滚动。
- `Chat` 进入已有内容会话时默认定位到消息时间线或 Chat 输出区底部，优先展示最新上下文；`Chat` 在同一活动会话发送新消息后也需回到底部展示新追加的用户消息与助手占位。除这类用户主动追加外，不得在同一会话持续更新、轮询刷新或 Process 展开期间覆盖用户的历史阅读滚动。
- `Chat` 的阅读定位条必须以悬浮 overlay 形式附着在消息区右下角，不得继续参与消息时间线的正常文档流；空白会话或少量消息时，不允许因为定位条占位把消息区额外撑高并制造伪滚动。`上一条 / 下一条` 按当前视口可见块实时计算目标，并支持连续跳转；当前块已被上一次跳转对齐到顶部偏移后，下一次 `上一条` 必须继续指向前一块。Chat 的阅读定位目标以用户消息为准，上一条和下一条都不得把 assistant 的 Thinking / Process 块作为中间跳转目标；视口位于两个用户消息锚点之间时，两个按钮分别指向上方与下方最近锚点，不得因视口内暂时没有锚点而同时隐藏。
- `Chat` Composer 支持图片附件草稿、输入框内 PC 剪贴板图片粘贴、缩略图预览与消息内图片回显；最近会话恢复仅持久化稳定图片资产引用，避免重复保留原始大图 payload；缩略位继续使用预览图，但消息回显与再次查看必须优先读取原图资源。助手 markdown 图片需在消息区直接以内联图片懒加载显示。带图消息只允许走支持视觉输入的模型链路，不进入异步 Task，也不静默降级到 Codex 文本执行；显式选择 `Codex` 时，服务端必须把已落盘图片路径作为 Codex CLI `-i` 输入传入直连执行链，并在首轮调用中显式分隔可变长图片参数与文本 prompt，确保 prompt 不会被解析为额外图片路径。
- Web 前端所有需要可见时间的管理视图、会话列表、详情面板与任务视图统一使用北京时间（`Asia/Shanghai`）与 24 小时制；`Chat` 的消息正文区不显示逐条消息或 turn 时间。Cron 创建表单默认时区固定为 `Asia/Shanghai`。
- Web 侧边栏、历史折叠、页面滚动隔离、克制冷灰工作台阅读主题、PC 端平面化控件、移动端轻量白色导航抽屉、移动端软键盘跟随、设置底部面板、低功耗轮询与长文本宽度约束作为统一前端体验要求维护；移动端运行页顶部 `Menu / 标题 / New` 控件必须像发送按钮一样支持首触执行，不得在输入框聚焦或软键盘打开时退化为先收键盘、第二次才响应。Chat Composer 外层 form 保留适度圆角，内部 textarea 保持直角；边框必须落在屏幕安全内边距内，避免贴底输入区贴边裁切；移动端 textarea、输入面和底部工具行使用紧凑高度，不得用大块白色留白撑高 footer。移动端Chat runtime Composer 的键盘位移必须采用稳定底部偏移，不依赖 transform 合成层；页面刷新或 WebView 恢复且没有输入框聚焦、上一帧也没有键盘证据时，短暂或持续的 visual viewport 收缩不得触发键盘布局，避免输入框阴影在 iOS Safari 键盘动画或刷新卡顿中留下残影。
- Web Shell 的交互反馈需由统一 motion token 驱动：悬停、按压、弹层进入、列表项抬升、焦点环、等宽数字和滚动隔离使用同一套 CSS 合同；常规反馈保持 120-200ms 即时响应，弹层与表面进入使用较长 ease-out 曲线，`prefers-reduced-motion` 必须关闭或压缩动效时长，避免 Settings 静态 frame、移动端键盘布局或阅读滚动被额外动画扰动。
- Web 前端需提供受显式开关控制的点击诊断能力，用于记录事件目标、顶层命中元素、遮罩层状态、`preventDefault` 状态、当前焦点与主线程长任务；默认不启用，不影响正常交互路径。
- Chat 长输出复制必须保持可用且不放大 DOM 体积：复制 payload 不得完整写入 `data-*` 属性，长输出轮询、草稿输入和复制操作不得触发整段 Markdown 反复解析或相同 `innerHTML` 反复写入造成明显卡顿；`Chat` 最终输出不得依靠全局 `user-select !important` 补丁维持选择能力，应通过统一稳定的 markdown shell 保留浏览器原生长按选中与复制菜单。
- 当前运行页的 Session 列表需直接展示在左侧主导航内，采用工作台式分组：置顶会话单独进入 `Pinned / 置顶` 分组并固定在 `Today / 今天` 上方，其余会话再按最近时间分到 `Today / Yesterday / Earlier`（中文对应 `今天 / 昨天 / 更早`）；`Chat` 统一使用 `Sessions` 栏标题与 `New` 新建入口；从 Chat 切到 Settings 时复用当前已注册 Chat 会话栏，不因页面切换主动刷新；直接打开 `/settings` 或当前没有已注册会话栏数据时，才请求 `/api/chat/sessions` 并渲染真实会话或真实空列表，不得展示本地伪造的单条 `New` 会话。移动端会话列表随左侧主导航抽屉展示，两条运行页顶部都只通过 `Menu` 打开左侧导航抽屉。列表条目主体只展示标题并在可用宽度内单行截断，真实会话尾侧固定提供三点更多按钮；展开菜单承载置顶、查看详情与删除操作，查看详情聚焦对应会话并打开 `Details`，删除需经过确认弹窗后才进入删除链路。空白 `New` 草稿/占位统一视作虚拟会话，不显示三点菜单，不支持置顶、详情或删除；同一路由只保留一个空白虚拟会话，重复点击 `New` 只聚焦现有空白入口，不创建多个空会话。长标题不得撑开导航、分组、列表或列表项宽度；新增会话插入、列表刚好填满或跨过滚动阈值时，不得触发滚动锚点补偿、滚动槽宽度重算或重新分配 rail 头部高度，也不得造成 `Sessions / New` 区块在不同运行页之间跳动；处理中会话在标题旁显示 loading，其他状态不显示状态灯。
- `Chat` 的已发送会话必须以服务端 Chat session store 为恢复源，并在同一 Web 登录态下跨设备共享；只恢复后端短 canonical id 对应的当前 Chat 会话模型，详情恢复统一使用 `route=chat`。未发送草稿与当前浏览器局部 UI 状态可继续本地保存；切换 Chat / Settings、切换会话或点击 New 时不得弹出丢弃草稿确认，原会话草稿按 route 与 session 继续缓存，返回后恢复，且不得阻断服务端会话摘要、配置和消息历史的恢复。
- Session history 必须维护 `last_active_at` 与 `pinned`。发送消息、assistant 收口、打开会话详情、Chat 输入/输出和任务结果写回会刷新活跃时间；Chat 会话列表先把置顶会话汇入独立 `Pinned / 置顶` 分组，再让非置顶会话按最近活跃时间排序并分组。系统默认清理超过 7 天不活跃且未置顶的会话，并同步移除 Session history、运行时 registry、会话附件/工作区和关联任务引用；仍有关联 queued/running 任务的会话在任务进入终态前跳过清理。
- 本地 Session history 物理文件按会话拆分：Chat 会话使用自身短 canonical `session_id` 写入 `.alter0/state/chat/sessions/<session_id>.json`，并使用 `.alter0/workspaces/chat/sessions/<session_id>` 作为独立工作区；旧 `alter0-chat` 归档日文件与 `chat-*` 长 id 状态不再合并到当前 Chat 会话模型。
- `Chat` 的会话条目不展示 ready/failed/exited/interrupted 等行内状态灯；仅处理中会话在标题旁显示 loading。workspace header 的状态按钮继续共享当前会话状态语义，状态名称仅保留给可访问性语义与悬浮提示。
- Web Shell 由 React 单一工作台直接渲染，主导航只暴露 `Chat / Settings`；Settings 内部按 `Runtime / Skills / Schedules` 分区，原生 Memories 的全局开关与活动状态归入 Runtime。
- `/chat` 与 `/login` 默认以英文启动，HTML 根节点语言标记为 `en`；Web Shell 保留显式语言切换入口，切到中文后需同步更新壳层文案与 `document.documentElement.lang`。
- 登录页需与工作台共享同一视觉基线：使用 `IBM Plex Sans + Sora` 字体组合、近白安全入口表面与安全入口语气，避免退回默认系统登录页样式。
- Web Shell 的稳定视觉基线收敛为两层：左侧固定主导航负责品牌、`Chat / Settings` 三条稳定入口、当前运行页 Session 列表、Settings 工具入口与语言切换，右侧主面板统一承载运行页和 Settings 管理页；`Chat` 在主面板内部统一采用「主时间线工作区 + 底部 Composer + 固定 workspace header」结构，并直接复用 workspace body、chat screen、composer、消息气泡与移动端顶部操作行语义 class。Web Shell 视觉基线需参考 Gemini 式扁平工作台：主工作区、Settings frame、管理分区、表格、详情面板和空态不再依赖外层圆角、卡片边框或厚阴影，层级通过留白、轻量分割线、低对比选中态与 Composer 输入面建立；设计图需维护在 `docs/design/workbench-flat-redesign.html` 与对应 PNG。Chat 与 Settings 顶部标题需共享紧凑 `workbench-title` chrome：运行页保留会话标题、状态信号与 `Details`，Settings 路由页只保留当前路由标题与同规格标题标记，并收进同规格主面板 frame；Settings 的 frame、标题、正文区和分区索引不得使用独立淡入、位移或页面出现动效；移动端顶部 `Menu / New` 等边缘操作需使用无边框图标按钮视觉并保留可访问文本标签；Settings 正文必须作为 frame 内部滚动区，不得因外层 frame 裁切而失去滚动能力。`/chat` 仅作为旧 URL alias 进入 `/chat`，不再生成独立运行页、Skill 选择器、Deliverables 或 Session Profile 面板。
- `Skill` 与其他 React 托管页面共享同一扁平 workbench surface system：列表、管理表单、托管字段块与消息块使用一致的白底主表面、浅灰辅助层和低对比选中态，不再默认使用卡片边框、厚圆角或重阴影表达层级。
- `/chat` 与登录页的对外品牌文案统一使用 `Alter0`：浏览器标题、登录标题、导航品牌位、会话栏标题与欢迎区 tag 不再暴露小写服务名。
- 主导航、控制台与资产页默认以高密度信息架构呈现：主导航主工作流只保留 `Chat` 三个入口，并用单个 `Management` 工具入口承接所有管理能力；控制面和资产能力在 Management 页内部用分组切换和高密度正文承载，长列表优先使用表格或主从视图，不再把大量配置和任务详情平铺为低密度信息矩阵。
- 移动端 Web Shell 使用 `--mobile-viewport-height` 动态视口协调壳体、顶部 workbar 和输入区：浏览器工具栏切换与软键盘弹起时，`html / body / #frontend-root` 不做 fixed 页面锁和 `overflow: hidden` 根层锁，App Shell 使用 `height: var(--mobile-viewport-height, 100dvh)` 自动贴合可见高度，并在 viewport meta 中声明 `interactive-widget=resizes-content` 作为支持浏览器的键盘 resize 策略。真手机宽度下 workspace body 仅承载三行：顶部 workbar footprint、正文 panel、真实 Composer footer；Composer 不得通过顶层 portal 脱离 `.runtime-workspace-body`，不得作为 fixed bottom 浮层，也不得依赖 `runtime-composer-spacer`。Composer footer 的左右 padding 必须覆盖安全区并使输入面边框完整落在屏幕内，form、textarea 与工具栏需保持紧凑高度；root、workspace、timeline 与 Composer 容器不得产生页面级横向滚动偏移。正文 panel 由 workspace grid 的 `minmax(0, 1fr)` 中间行承载，不能再手写 `VisualViewport height - mobile workbar - Composer` 或把键盘高度写入 workspace grid、Composer spacer、`.runtime-workspace-screen` 的底部 padding/scroll-padding。运行页需发布 `mobile-rest / mobile-keyboard / mobile-primary-nav-drawer / mobile-session-drawer / mobile-composer-panel / mobile-details-dialog / mobile-attachment-preview` 布局状态；主导航抽屉、会话抽屉、详情弹层或附件预览拥有视口时必须释放主输入焦点，抽屉与遮罩层级高于 Composer；主导航抽屉与遮罩必须按动态可视高度裁剪，内部会话区自行滚动，不得超出屏幕底部或触发页面级拖拽；Composer 保持可见但不可交互，不通过隐藏或卸载修正层级。workspace header、正文滚动区、空态、阅读定位条、命令候选和配置面板不消费键盘高度或 transform 改写高度，不出现底部空白、内容裁切、整页上移、输入区重复上移或 Composer 覆盖抽屉。
- 移动端 Chat 的左侧导航抽屉必须保持统一开合语义：`Chat` 都只保留 `Menu` 作为抽屉入口。点击遮罩、切换路由、切换会话或创建新会话后不得残留旧的展开层。
- 移动端运行页左侧导航抽屉需优先保证真机稳定性：遮罩保留淡入淡出，抽屉本体仅保留一层轻量侧滑，不叠加多层位移、淡出或条目级顺序动画；抽屉面板使用近白表面、平面菜单、细分割线和自然滚动的会话区，抽屉高度不得超过动态可视高度；抽屉内置顶会话单独位于 `Pinned / 置顶` 分组，其余会话再按最近时间分组，并统一采用「标题 / 尾侧三点菜单」的紧凑导航列表结构，仅处理中会话在标题旁显示 loading，避免退回松散白色块、状态灯、元信息或过度胶囊化。
- 共享运行时的短哈希预览 host 与主域工作台必须落在同一登录保护边界内：`/login` 可直接在预览 host 打开，登录态 cookie 需对 `*.alter0.cn` 生效，避免主域与预览子域重复维护独立会话。
- 共享运行时采用 `supervisor -> web child` 进程模型时，主 Web child 必须继承非空 `web_login_password`；只有 workspace service 托管出来的预览后端允许通过专用运行时标记移除自身登录层，复用共享网关登录态。
- `Chat` 的移动端键盘弹出链路需保留浏览器原生软键盘手势：首次触摸主输入框时不得在 `pointerdown / touchstart` 捕获阶段取消默认行为，不得主动 focus、锁定 `window` page scroll、通过 `scrollTo` 干预真实焦点，也不得记录或回放页面级滚动锚点。键盘开合过渡期内，运行页只通过 `--mobile-viewport-height` 自然同步 App Shell 可见高度，Composer 作为 workspace grid footer 跟随容器底边；输入框后方的 `workspaceBody / runtime-workspace-screen` 等滚动容器不做短时锁定，移动 workbar 不消费 `VisualViewport.offsetTop` 做 transform，workspace header 与正文 panel 不单独消费键盘变量做位移。其他组件必须由 App Shell 动态视口高度、document 正常滚动语义与静态 workspace inset 保持原位，不再通过页面级滚动锚回逻辑接管浏览器键盘动画；键盘动画不得造成页面整体分辨率/可视区域突变。
- `Chat` 首触主输入框后的键盘动画稳定窗口内，运行页不得回放页面级滚动锚点或通过 `window.scrollTo` 修正背景位置；workspace screen 与 workspace body 的滚动位置由当前滚动容器自身维护，用户一旦在消息区产生 `touchmove / pointermove / wheel / scroll` 意图必须继续即时生效。
- `Chat` 在移动端主输入框聚焦期间，消息区必须保持浏览器原生滚动路径；前端不得在 `.runtime-workspace-screen` 上安装 `touchmove` 接管、不得阻止默认滚动，也不得用脚本写入 `scrollTop` 模拟手势滚动。键盘 overlay 造成的可见高度差只能转换为 `.runtime-workspace-panel` 的实际高度，不得叠加到 `.runtime-workspace-screen` 的底部 padding 或 scroll-padding。
- `Chat` 的移动端 Composer 作为 workspace grid footer 时不得拦截消息区滚动手势：外层 footer 和 form surface 空白不应制造独立滚动层或捕获正文拖动，只有 textarea、附件、工具、发送、附件预览和配置面板等真实控件接收事件。
- `Chat` 在移动端软键盘打开期间，只允许主输入框自身和正文消息滚动区保留键盘焦点；移动 workbar、左侧抽屉、Composer 工具栏、附件、发送、设置面板、遮罩与其他运行页控件在触发动作前必须释放当前输入焦点，避免出现“抽屉/面板已打开但键盘仍占用视口”的叠层状态。
- `Chat` 的移动端发送按钮在触摸提交时，必须先让当前主输入框失焦，再提交当前草稿；键盘回收与 composer 回弹由 `--mobile-viewport-height` 和 workspace grid 自然恢复，不允许发送后键盘停留不收或残留悬空底部占位。
- `Chat` 在移动端的 Composer 回弹到底边时，运行区保持原位；键盘收起、输入框失焦和视口回弹后，不允许遗留额外底部空白、悬空按钮或上一轮键盘高度对应的占位残影。
- `Chat` 在移动端键盘弹起与收回期间，只允许 App Shell 的 `--mobile-viewport-height` 改变可见高度；底部 Composer 必须是 workspace grid footer，随容器底边移动，不得通过 fixed bottom、transform 或 spacer 追加位移。顶部操作行、紧凑 workspace header、正文滚动区、空态、命令候选与配置面板保持布局原位，不跟随键盘位移做额外动画。
- `Chat` 在移动端软键盘弹起期间，底部 Composer 必须高于消息阅读定位按钮与 Chat 四键定位条；消息阅读定位按钮与 Chat 四键定位条在主输入框聚焦后必须主动隐藏，待输入框失焦、键盘收起后再恢复，不得压到输入框、附件条或键盘上方。左侧抽屉打开时先 blur 当前输入框并收起键盘，抽屉和遮罩层级高于 Composer，Composer 保持可见但退出交互层。
- `Chat` 的主输入框在移动端必须显式关闭系统自动填充、卡片、地址与密码类输入辅助条；键盘上沿不得再额外挂出会暴露底部残留页面层的系统输入助手。
- `Chat` 的移动端主输入框必须保持不低于 16px 的可编辑文本字号，避免 iOS Safari 聚焦输入法时触发页面自动缩放、横向裁切或分辨率突变。
- `Chat` 的移动端发送按钮必须支持在软键盘保持打开时直接点按提交；首触发送需覆盖 `pointerdown(touch)` 与 `touchstart` 提交链路，并在同一次触摸内去重，不允许先消费成键盘收起或焦点切换，再要求第二次点击才真正发出请求。
- `Chat` 的四键阅读定位条需统一使用同一套共享实现与圆形按钮语言，不再为不同运行页维护分叉样式或独立跳转逻辑。
- 运行页 Composer 的键盘跟随只依赖 CSS 动态视口和 workspace grid，不额外叠加 `bottom`、transform、scrollTop 或 spacer 过渡动画；键盘收起与输入区回弹阶段应保持直接、稳定的回贴节奏。
- 输入框 blur 后，运行页应沿着 `--mobile-viewport-height` 的实际回弹过程恢复高度，不额外保留键盘占位，避免底部输入区和正文区出现闪烁。
- 输入框聚焦且软键盘已确认弹出后，移动端运行页不得用 `VisualViewport.offsetTop`、页面级 scroll 锚点或 JS 变量驱动 App Shell、workspace header、正文 panel 或 Composer 位移；高度收缩以 `--mobile-viewport-height` 为准，Composer 作为 workspace grid footer 随容器底边移动，避免浏览器键盘动画和脚本位移叠加造成页面整体再次上移或输入区先消失再出现。
- `Chat` 在页面从后台恢复到前台、浏览器重新激活当前页，或系统把当前 WebView 恢复为可见状态时，除补拉会话与任务数据外，还必须立刻重算共享视口诊断变量；前台恢复后的第一帧不允许沿用后台前的旧可视高度或旧底部空白。
- Web Shell 的抽屉式单列布局仅在主视口宽度 `1100px` 及以下触发；高于该阈值时保留左侧固定主导航与右侧主面板。进入窄屏后主导航切换为贴边抽屉，当前运行页的会话列表随主导航一起展示，由工作区头部的 `Menu` 入口打开；运行页空列表需优先展示一条 `New` 占位会话，Chat 的占位会话在首次发送输入或添加附件时才落成真实服务端会话，点击列表占位或移动端顶部 `New` 必须关闭会话抽屉并聚焦输入框，不直接显示空态卡片或提前创建服务端会话；真实 Chat 会话在首条输入命名前也必须使用 `New` 作为默认标题。`Chat` 与其他 `page-mode` 页面继续保持单主面板，但 `page-mode` 路由页标题上方必须稳定提供 `Menu` 入口；`760px` 及以下再进一步压缩按钮与间距，避免窄屏下出现不可触达区域。主导航抽屉必须独立承担纵向滚动，小高度视口下不允许出现菜单或会话列表被裁切且无法滑动的状态。
- 窄屏主导航抽屉点击任一路由项后需立即关闭；页面切换完成后不得继续保留旧菜单层覆盖在目标页之上。
- 窄屏主工作区按页面类型收口为贴顶起始区：普通 `page-mode` 路由页继续采用“两行头部 + 贴顶正文起始区”节奏，第一行承载抽屉入口与主操作，第二行承载当前标题；`Chat` 在真手机宽度下统一收敛为单层运行页 workbar，左侧保留 `Menu`，中间显示“状态信号 + 当前会话标题”的单行标题按钮，右侧固定承载 `New`，通过点击真实会话标题打开 `Details`，草稿/占位 `New` 标题不触发详情，不再把 `Details` 作为独立顶部按钮或再叠一层 header。所有页面都不得在顶部遗留额外大块留白。
- 窄屏 `Chat` 工作区顶部固定保留统一运行页入口：Chat都通过 `Menu` 进入左侧主导航抽屉，`New` 直接创建当前路由对应的新会话；标题区需要稳定承载当前会话名和状态信号，并作为 `Details` 的直接触发入口，不再出现移动端无导航入口、标题缺席或只能依赖正文内按钮切换会话的状态。
- `Chat` 工作区头部固定为共享单行 header：桌面与中宽度继续保留会话标题、状态按钮和 `Details` 入口；真手机宽度则把 `Details` 下沉到中间标题按钮。`Details` 只承载会话元信息；模型与 Skills 调整统一通过底部 Composer 工具栏的 `Session` 按钮进入，Tools / MCP 不再提供独立 Composer 面板；GitHub 仓库选择使用独立图标，不能复用附件按钮。Chat 模型区除常规 Provider / Model 外，稳定提供内置 `Codex` 直选项，选中后仅影响后续消息，并把执行链显式切到 `Codex Direct`。`/chat` 不再提供独立目标选择、Deliverables、Session Profile 或独立 Skill 配置面板。
- `Chat` 在页面隐藏时停止轮询。回到前台的后台时长不足 5 分钟时，稳定 `ready` 会话不得请求列表或详情，本机已提交但未完成的会话只恢复 owner updates；后台时长达到 5 分钟时，必须拉取一次轻量全量列表并补拉当前活动会话最新详情。首次进入先展示浏览器缓存，再拉轻量全量列表与当前详情；用户显式刷新同样同时刷新列表与当前详情。若增量窗口过期、服务重启要求 resync、缓存不完整或 updates 长期没有产生实际会话进展，再补拉必要详情。focus/pageshow/online 的重复事件需去抖，页面隐藏期间不得启动新请求。
- Chat 单会话详情默认只返回最新 `20` 个 turns，并按约 `1MiB` 的前端 API turn DTO 页预算控制单次响应体；summary、turns 与分页边界必须来自应用层同一次原子快照。长会话通过 `turn_limit` 与 `turn_before` 按批次读取；无效 `turn_before` 必须显式返回 `before_turn_found=false` 和空页，不得静默回到最新页。分页响应必须带 `turns_paging`，其中包含数量边界与 `byte_limit / approx_bytes`；更早历史只允许添加未知 turn，不得修改已存在 turn、最新详情新鲜度或当前阅读窗口。集合接口固定只返回 `id / title / status / created_at / updated_at / pinned` 与可选的脱敏 repository binding，成功响应对服务端会话成员关系具有权威性，摘要不得携带或清空 turns。
- `Chat` 发送输入时必须先把带 `client_request_id` 的 optimistic user message 写入当前 route 状态和浏览器缓存，再发起 input 请求；服务端对应 turn 必须回传同一 `client_request_id` 完成对账。会话运行态只能通过共享 runtime controller 的单一提交入口修改；React effect、ref 和持久缓存不得把旧副本再次合并回运行态。服务端 input、updates、列表和详情响应必须先校验目标 session、请求代次、`update_id` 与内容 `updated_at`，旧响应在任何 message/turn 合并前整体拒绝。summary 只更新列表字段，title/pin 变化不得推进内容 `updated_at` 或重写内容缓存；detail/event 才能推进内容。
- 浏览器缓存按个人单设备首屏恢复设计：全部会话只持久化轻量 summary，完整内容最多保留当前会话与最近 4 个会话；附件 data URL、按需加载的 runtime event blocks 不得进入长期快照。会话切换不得拉列表；仅在内容缺失、缓存损坏、summary `updated_at` 新于内容时间、未完成会话没有活跃 updates，或用户显式刷新时拉详情。匹配 `updated_at` 的稳定已加载会话必须直接展示且不发详情请求。
- `Chat` 是唯一对话运行时，工作台一级入口统一为 `/chat`、`/settings`；`/chat` 挂载 Conversation runtime UI，并使用单一 Chat owner、API、active session 与草稿缓存命名空间。当前活动会话稳定反映到 URL query，统一写入后端生成的短 canonical id，格式为 `session_id=c_<16位小写字母数字>`；典型入口为 `/chat?session_id=c_x8k4p9m2q7vd3n6a`，恢复对应 Chat 会话模型。该 id 同时作为列表、详情接口、updates、持久化文件和工作区路径的唯一会话标识，前端不再维护临时引用到完整 id 的映射。
- `Chat` 首页 Composer、会话列表项与 `Details` 面板需维持同一套浅色 runtime 表面系统：Composer 采用单一胶囊式助手输入面板，主 textarea 无内边框并与底部工具行处在同一白色 surface 内；Chat 工具行左侧依次提供无边框 `Session`、独立 GitHub 仓库、附件与必要 meta，右侧收口发送动作，GitHub 与附件不得复用同一图标或按钮。桌面端按主阅读宽度居中，移动端控制输入高度、底部留白和发送按钮体量，并保持输入区具备足够横向留白；Chat 不得为 Composer 外壳覆盖更深背景、更低底部 padding 或外层状态 note 行，失败、退出与附件错误提示需进入共享工具栏 meta。PC 端上传、发送、状态、详情、流程入口与弹窗动作保持平面化，除 Composer 胶囊外不通过额外圆角、边框或厚阴影表达层级；详情面板需保留清晰标题栏、显式关闭按钮、紧凑摘要栅格和轻量字段分隔，会话列表项和详情面板不再退回旧式轻表单或松散卡片观感；空态工作区使用低对比网格与细弧线背景，同时禁止保留可拖拽滚动，不得把头部操作行或输入区顶出可视区。
- `Chat` 的桌面端草稿输入必须保持低延迟：仅因未发送草稿变化时，不得同步重建整条消息时间线、Markdown 正文或 `Process` 结构；浏览器草稿缓存允许延迟落盘，但不得影响当前输入内容、会话切换后的草稿恢复与发送结果。
- `1100px` 及以下的移动工作台需优先保证真机滚动与抽屉切换流畅度：主工作区、Conversation/Chat 抽屉遮罩、抽屉面板本体与运行页容器不得继续依赖大面积 `backdrop-filter`、持续背景光晕或其他会导致整页重绘的装饰层，统一保持静态浅色表面。
- `Chat` 窄屏工作区头部不得重复输出内部会话入口；`Sessions` 入口统一由壳层头部提供并打开左侧主导航抽屉，工作区头部仅保留与当前会话直接相关的操作。
- `Chat` 空态首屏在桌面与中宽度下必须保持居中首屏节奏：欢迎区标题、描述、target/prompt 需在 header 与 Composer 之间沿欢迎区中轴竖向居中展示；真窄屏继续贴近头部下沿起排。Composer 继续沿主工作区自然贴底排布；不允许通过 `margin-top: auto`、过大的欢迎区上边距或类似弹性占位把输入区推到底部，造成首屏中上部出现大块无效空白。
- 移动端 Chat 在主输入框与会话设置底部面板之间切换时，不允许保留“键盘 + 设置面板”双重底部占位：打开设置前先释放输入焦点并清理键盘偏移，回到主输入框时先自动收起设置面板；Chat 在真手机宽度下允许工作区头部工具栏换行，操作按钮不得被长标题挤出可见区域。
- 桌面宽屏下 Chat 消息列与底部输入区需按主工作区宽度自适应扩展，并统一收敛到居中的 `960px` 最大阅读宽度，避免正文与输入区无限拉长。
- `Chat` 统一展示右侧四键阅读定位条，承载 `回到顶部 / 上一条 / 下一条 / 回到底部`；定位目标按当前视口中的可见消息块或 Chat turn 动态计算。`回到底部` 只在最后一条内容的底边仍位于视口外时显示，不得因为消息区尾部仅剩空白或 padding 继续保留伪底部跳转。移动端四键定位条固定停靠在工作区右侧、输入区上沿之上，四键统一使用独立圆形触达面；当当前消息滚动容器内存在有效文本选区时，定位条必须立即隐藏，避免遮挡复制拖拽与选区手柄。

## Skill & Memory

核心对象：`CLIRuntime`、`RuntimeProfile`、`Skill`、`SkillRepository`、`MCPServer`、`CodexHome`、`NativeMemoriesSettings`。

稳定需求：

- Chat 是一个可执行任务的 CLI 运行时，由 Claude Code 或 Codex CLI 承担任务推理、工具调用和会话内上下文压缩。服务侧负责选择运行时、准备工作区与归档会话结果。
- Runtime Resolver 按优先级选择执行器：已启用且健康的 Model Provider 使用 `Claude Code + provider profile`；未配置 Provider、Provider 不可用或 Claude Code 启动失败时使用 `Codex Direct`。
- Product Skill 独立维护在 `docs/skills/<skill_id>/SKILL.md`；启用且非私有的 file-backed Skill 原子同步到 Codex 用户级 Skill 目录，由 Codex 隐式匹配或通过 `$skill-name` 显式调用。
- Chat 不提供分会话 Skill 选择，不发送 `skill_ids`，也不向会话工作区或绑定仓库复制 Skill、生成 Skill manifest 或托管 `AGENTS.md`。
- 服务首次发现原生 Memories 可用且配置键缺失时，将 `features.memories`、`memories.generate_memories` 与 `memories.use_memories` 默认写为 `true`；后续新建与续接任务直接读取活动 `config.toml`，不得追加会话级或命令行强制覆盖。
- `Settings > Runtime` 提供全局 Memories 总开关、生成开关与召回开关，并展示生成文件数量和最近活动；不得解析未公开 schema、展示生成内容或暴露 `CODEX_HOME`。独立 Memory 分区与专属 API 不再保留。
- 仓库中的 `AGENTS.md` 属于项目运行规则。绑定仓库后 Codex 直接从 `repo/` 运行并按原生层级发现规则；服务不得改写仓库 `AGENTS.md`。

## Task, Chat & Workspace

核心对象：`TaskSummary`、`TaskLog`、`ArtifactRef`、`ChatSession`、`ChatTurn`、`RuntimeTraceEvent`、`Workspace`、`CodexThreadID`。

稳定需求：

- Chat 页面 Composer 支持最多 5 个附件，稳定覆盖图片与常见文本/文档文件；普通文件写入会话工作区 `input-attachments/<turn_id>/`，绑定仓库时提示路径相对 `repo/` 计算。Chat 当前活动会话的 shell 明确为 Codex 时，输入 `/` 显示 Web 适用的 Codex CLI 斜线命令候选。Chat 输出正文、Markdown 正文与代码结果保留浏览器原生文本选择能力。Composer 与 Details 均不提供 Skill 选择入口，输入 API 不接受会话级 Skill 配置。
- Web 会话不直接暴露本地文件路径。
- 默认工作区按执行上下文隔离：Session/Task 使用 `.alter0/workspaces/sessions/<session_id>` 与其会话下的 `tasks/<task_id>`；Chat 使用 `.alter0/workspaces/chat/sessions/<chat_session_id>`，其中 `chat_session_id` 是后端生成的短 canonical id，格式为 `c_` 加 16 位小写字母数字。
- Chat 的会话图片资产需要随 Session 工作区落盘：用户上传图片的原图与预览图统一写入 `.alter0/workspaces/sessions/<session_id>/attachments/<asset_id>/`，前端持久化与消息请求默认复用 `asset_url / preview_url` 引用；其中 `preview_url` 只服务缩略位，消息回显与再次查看统一优先读取 `asset_url` 原图。assistant 最终回复里的外链 markdown 图片也应在会话返回与落库前改写到同一路径下的本地附件 URL。
- 直连 Codex 的 Chat 会话使用共享活动 `CODEX_HOME`，会话隔离由独立工作区、仓库 checkout、附件与 Codex thread id 保证。会话删除不得删除共享认证、配置、Skills 或 Memories。
- Chat 是独立会话式运行时代理，持久化 Codex CLI 线程标识、会话状态、标题、工作区、日志与 `RuntimeTraceEvent` 视图索引。
- Chat API 支持会话创建、列表、恢复、输入、删除、详情读取以及 turn/runtime event 明细读取，前端可按事件展开或检索执行细节。
- Chat 会话态统一为 `ready / busy / failed / exited / interrupted`，执行态在 turn/runtime event 维度维护 `running / completed / failed / interrupted`；请求失败时 session 与 turn 同步进入 `failed` 终态，但 failed session 仍可接受下一条输入并重新进入 busy。运行态退出、失败或中断后保留历史；Composer 只由真实 busy 状态锁定，不得因失败占位、未回答的历史 user turn，或历史 `ready` summary 与 failed turn 的同毫秒时间戳长期锁定，继续发送即可恢复。
- Chat 恢复默认优先复用已持久化 Codex CLI 线程；若续写命中远端 compact 失败，则保留原会话历史、工作区和线程标识，下一次输入继续 resume 同一 Codex CLI 线程。
- Chat 会话删除统一从左侧会话列表触发，`Delete` 会同步清理状态文件和独立工作区；工作区头部不再提供单独的 `Close` 入口。删除成功后，无论删除的是历史会话还是当前活动会话，当前会话列表所在的左侧导航抽屉都保持删除前的展开状态，便于继续清理其他会话；用户随后通过 `Menu` 或抽屉外部遮罩主动关闭时，抽屉必须正常收起。前端在后续列表刷新、轮询和 page-activation 补偿刷新中也不得把该会话重新补回，直到服务端列表稳定反映删除结果。
- Chat 历史在同一 Web 登录态下跨设备共享，不按浏览器 client 标识隔离；不设置产品级会话数量上限或固定超时淘汰。
- Chat 移动端、输入稳定性、滚动导航、Process 折叠、一键复制、长输出阅读、轮询降频与缓存写入节奏作为 Chat 子域体验要求维护。
- Chat 四键阅读定位条按当前视口中的可见 turn 集合计算目标：`上一条` 固定指向最上可见 turn；`下一条` 在单条 turn 可见时指向真实下一条、在多条 turn 同屏可见时指向最下可见 turn；最后一条 turn 单独可见时隐藏 `下一条`。
- Chat 发送按钮首次点击必须立即进入 pending 反馈；若当前还没有 active session，前端允许先创建会话再继续发送，但首击期间按钮需同步切到 `Sending...` 与禁用态，避免重复点击和“第一次点击无反应”的错觉。
- Chat 刷新节奏需按会话状态自适配：执行中的会话保留实时刷新，空闲会话停止周期轮询并依靠页面激活补偿刷新；用户正在滚动阅读输出时，不得因明细轮询而打断当前滚动。
- Chat 在同一浏览器工作台内切换到其他页面后再返回时，前端需优先使用未过期的 24 小时运行态内存缓存恢复会话列表和当前活动会话的已加载内容；`Chat` 缓存完整已加载消息，`Chat` 缓存完整已加载 turns，接口返回后继续合并更新。缓存不作为跨设备或服务端事实来源；每次访问、切换、刷新或前台恢复都会按最新合并结果刷新缓存时间。
- Chat 窄屏消息页必须保持 `workbench-main -> chat-pane -> chat-view -> chat-chat-screen` 的闭合高度链，由 `chat-chat-screen` 独立承担纵向滚动；外层容器不得因 `overflow: hidden` 或高度塌陷吃掉滚动。
- Chat 移动端在输入框聚焦且软键盘抬起后，Composer 必须作为 workspace grid footer 贴住动态视口底边；长对话或长输出期间不得通过 `position: fixed`、拉高 footer padding、改变滚动容器或破坏高度闭合链把输入区挤出屏幕。
- Chat 移动端的 `chat-chat-screen` 必须继续按当前 Composer 的真实遮挡高度动态收口；会话空态、长输出与 Process 阅读都要稳定停在输入区上沿，不允许被底部 Composer 覆盖。
- Chat 移动端的命令与 prompt 气泡需保持自然整词换行；路径、flag 和短 shell 片段优先按空格或真实长单词边界断行，不允许因窄屏收缩把命令压成逐字或逐 token 的碎行。
- Chat `Process` 的步骤头必须保持稳定三列：左侧独立展开图标列、中间标题主列、右侧耗时与状态列。标题只能在中间主列内截断，不允许因为节点缺失、DOM 顺序错误或 grid 列错位把标题挤进图标列，导致移动端只显示单个字符。
- Chat `Process` 展开后的自然语言步骤详情需使用同一套阅读修正：`reasoning / plan / message / text` 等说明类内容优先按 markdown 正文块整列换行，展示前移除零宽断行字符，并把“每字一行”的病态段落归一回可读正文；仅等宽输出、diff 与代码类块继续保留预格式化渲染。
- Chat 的 `Thinking / 已思考` 外层披露展开时只进入步骤列表态；该动作会收起同一 assistant 消息下已打开的单步详情，避免移动端把历史详情重新展开并造成视口突跳。单步详情仍由用户点开具体步骤后展示。
- Chat 的过程详情必须共用同一套最终 detail surface 渲染规则：运行时、代码、diff、tool input 与 JSON 类输出直接使用等宽内容块，说明、markdown、thinking、文本型 tool output 与 error 直接使用富文本正文块，并保留 block 标题、文件名与起始行号；Chat 中需要单独拉取 detail 的步骤，必须在 detail 返回后再展开步骤体，不允许先显示 preview 兜底再二次跳变。
- Chat 移动端的四键阅读定位条停靠在 Composer 上沿之上，不跟随软键盘位移动态上移；输入框聚焦且键盘弹起时按钮组主动隐藏，键盘收起或浏览器视口回弹后再恢复到 Composer 上沿之上，不得留下悬空残影。

## Control, Operations & Governance

核心对象：`ChannelConfig`、`SkillConfig`、`RuntimeProfile`、`ModelProvider`、`ClaudeProviderProfile`、`CodexAccount`、`CodexLoginSession`、`RuntimeInstance`、`DeploymentBaseline`、`EngineeringPolicy`。

稳定需求：

- Control API 管理 Channel、Capability、Skill、MCP、Runtime Profile、Cron Job、Model Provider 与 Codex Runtime 配置，并保留 Capability 生命周期审计。
- 服务启动后提供 `preview-publish`、`frontend-design`、`doc-coauthoring`、`fullstack-developer`、`code-reviewer`、`webapp-testing`、`find-skills`、`test-driven-development`、`ui-ux-pro-max`、`code-simplifier`、`code-review`、`brainstorming` 与 `travel` 等公有内置 Skill；旧 `memory` 与 `memory-maintenance` Skill 从持久化控制状态清理。所有内置 Skill 统一使用标准 `SKILL.md` 入口，并同步到 Codex 用户级原生 Skill 目录。生命周期同步只管理带 alter0 标记的目标，不覆盖或删除用户自行安装的 Skill；无效源或目标冲突会拒绝对应生命周期变更。
- 共享 Web 运行时需要支持通用 workspace service 注册：`GET /api/control/workspace-services` 查询注册表，`PUT /api/control/workspace-services/{session_id}` 绑定默认 `web` 服务，`PUT /api/control/workspace-services/{session_id}/{service_id}` 绑定附加服务，`DELETE` 接口用于清理绑定；当请求 Host 命中 `<session_short_hash>.alter0.cn` 或 `<service>-<session_short_hash>.alter0.cn` 时，共享运行时需按注册类型分发前端构建或反向代理到目标 HTTP 服务。`travel` 服务是唯一例外，固定命中 `https://travel-<session_short_hash>.alter0.cn`，且该 host 只读、免登录，只允许返回静态 HTML/资源。标准 `web` 部署默认应把当前会话后端启动命令注册给共享运行时托管，再以 `http` 方式绑定短哈希子域名，确保前端与 `/api/*` 同时来自当前分支；`frontend_dist` 仅作为静态预览模式保留。
- Channels 入口归属 Settings 模块，旧直达路由保持兼容。
- Models 控制面支持 Claude Code provider profile 配置，包含 base URL、API Key 保留语义、model、profile、Provider 路由偏好、默认项自动收敛与历史缺密钥配置恢复；启用且健康的 Provider 作为 Claude Code 首选运行来源。
- Codex Runtime 作为 `Codex Direct` 的账号与模型管理来源，在无可用 Model Provider 或 Claude Code 运行失败时承接自然语言任务兜底执行。
- Runtime 设置页支持在线实例启动时间与 commit hash 展示、运行时重启、默认启用的远端 master 同步、打开重启弹窗时拉取当前运行 commit 之后全部 master 候选 commit，并追加当前运行 commit 向前 10 个历史 commit，候选列表按提交时间从新到旧展示、按短 hash 选择目标 commit 重启、仅在后端检测到 Git 已跟踪本地改动后才触发的二次确认、确认后丢弃已跟踪改动的重启前同步策略、通过统一前端感知构建入口生成候选二进制、readyz 探活与失败回滚。旧运行参数配置页、环境变量可视化配置、队列、运行时 shell、记忆路径参数均不再对用户暴露。
- Settings 页面提供 Codex Runtime 面板，使用单一顶部面板承载当前服务运行账户的 Codex 身份快照、邮箱、计划、认证模式、hourly / weekly 额度、profile、LLM Provider 注册状态，以及基于 Codex app-server 真实能力返回值的活动 model / 思考深度切换。首屏加载时 Codex Runtime 状态与 LLM Provider 状态需并行读取。页面支持为当前运行账户启动 Codex device-code 登录，并展示验证链接、用户码、过期时间、轮询间隔与登录输出；登录成功后刷新 Runtime 身份与额度。页面同时支持通过 Claude Code Provider Console 连续注册与编辑多个 OpenAI-compatible Provider；桌面端 registry 与 editor 在同一容器内左右分栏，窄屏单列展开。字段包含 Provider 名称、base URL、API key 与 models；models 使用全宽多行编辑区，支持换行或逗号分隔，提交后写入 Model Provider 注册表，首个 model 作为默认模型，并刷新 Provider 状态。已注册 Provider 需展示名称、base URL、默认 model、模型数量、模型列表与启用/默认状态；编辑时 API key 留空表示保留已保存密钥。每次注册或更新成功后表单清空 base URL / API key / models，并自动准备下一个未占用的 `Claude Code N` 默认名称。页面不展示 Account ID / User ID、保存名称、多账号导入/切换入口、CLI 命令、auth/config 路径、诊断侧栏或由 auth/config 文件存在性推导的 Ready/Status 文案。额度必须来自当前 `auth.json` 的实时 quota 刷新结果，model / 思考深度选择变更后仅实时写回当前用户配置中的 `model` 与 `model_reasoning_effort`。
- 公网部署基线要求服务绑定 localhost、启用 Web 登录密码、显式配置 `ALTER0_RUNTIME_ROOT`，并通过 Nginx 做反向代理。
- 服务内 GitHub 交付要求运行账户具备 GitHub App token helper、`gh` 包装器、SSH 提交签名、稳定 PATH 与 Codex CLI 可用认证。
- Codex CLI 默认从运行账户托管目录、NVM 稳定入口与版本目录、显式候选、公共路径和 `PATH` 中自动选择已安装的最高语义版本；历史具体 Node 版本路径不得阻止服务切换到更新 CLI，只有显式 pinned 模式允许锁版。
- Node/Playwright 测试链路通过运行账户级工具链初始化，保证 Codex CLI 可执行 `internal/interfaces/web/frontend` 的构建与单测，以及 `internal/interfaces/web` 的 Playwright E2E。
- 研发流程遵循 TDD：功能新增、缺陷修复、行为调整与重构默认先以测试表达目标行为，再完成实现与重构；纯文档、注释、格式化、依赖元数据或无法自动化验证的变更需说明免测原因与替代验证。

## 维护规则

- 新需求必须先选择领域，再选择子域；无法归类时优先补充领域模型，而不是新增线性编号。
- 一个需求只允许有一个主归属领域；跨领域影响通过 `依赖与边界` 说明，不复制成多个重复需求。
- 用户可见行为、交互方式、入口路由、执行模式、返回结构或默认策略发生变化时，同步更新 `README.md`。
- 需求细节、接口、状态、验收和边界放入对应 `requirements-details/*.md` 文件；技术方案、包边界、调用链路、存储、观测和测试策略放入 `technical-solution.md` 的同名领域下；`requirements.md` 只维护稳定总览与领域索引。
