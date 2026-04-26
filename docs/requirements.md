# Requirements

> Last update: 2026-04-25

`alter0` 的需求清单按领域模型维护。后续新增需求不再使用线性编号，也不按提交顺序堆叠；需求应落到对应领域、子域与能力项下，使用稳定领域路径表达，例如 `agent.execution.react`、`memory.files.injection`、`task.workspace.runtime`。

状态说明：

- `supported`：已在主干代码可用
- `in-progress`：正在落地，接口或行为尚未稳定
- `planned`：已确认方向，待排期

## 领域索引

| 领域 | 范围 | 状态 | 细化文档 |
| --- | --- | --- | --- |
| Runtime & Orchestration | 输入通道、统一消息、意图路由、执行端口、调度、存储、观测 | supported | [runtime-orchestration.md](requirements-details/runtime-orchestration.md) |
| Conversation & Session Experience | Chat、Agent 入口会话、SSE、历史、移动端、阅读与输入体验 | supported | [conversation-session-experience.md](requirements-details/conversation-session-experience.md) |
| Agent Capability & Memory | Agent Catalog、ReAct、工具、Skills、MCP、Memory Files、长期记忆、上下文压缩 | supported | [agent-capability-memory.md](requirements-details/agent-capability-memory.md) |
| Task, Terminal & Workspace | 异步任务、任务观测、任务日志、产物交付、Terminal 会话、独立工作区 | supported | [task-terminal-workspace.md](requirements-details/task-terminal-workspace.md) |
| Control, Operations & Governance | 控制面配置、模型 Provider、Environments、部署基线、认证凭据、TDD 研发约束 | supported | [control-operations-governance.md](requirements-details/control-operations-governance.md) |

## Runtime & Orchestration

核心对象：`UnifiedMessage`、`OrchestrationResult`、`Intent`、`Command`、`ExecutionPort`、`SchedulerJob`、`Channel`、`TraceContext`。

稳定需求：

- CLI、Web、Cron 等输入源统一转换为 `UnifiedMessage`，并携带 `message_id`、`session_id`、`trace_id`、`channel_type`、`trigger_type` 与业务载荷。
- 编排层负责意图识别与路由：命令进入 `CommandRegistry` / `CommandHandler`，自然语言进入 `ExecutionPort`，Cron 触发复用同一编排链路。
- 命令能力稳定提供 `/help`、`/echo`、`/time` 与 `/now`。
- 自然语言执行通过可替换执行端口对接 LLM、Agent、Codex CLI 或后续 Workflow 执行器。
- 调度领域支持 Cron Job 配置、可视化周期字段、触发记录、触发会话归档、runs 查询与来源回链。
- 存储默认采用 `.alter0` 本地文件，Control 配置与 Scheduler 状态以 JSON 为主，Memory 主存以 Markdown 为主。
- 观测能力覆盖结构化日志、Prometheus metrics、`/healthz`、`/readyz` 与 trace/session/message 维度。

## Conversation & Session Experience

核心对象：`Session`、`Message`、`LiveUserMessage`、`StreamEvent`、`AgentEntry`、`ViewportState`、`SessionHistoryBucket`。

稳定需求：

- `Chat` 默认绑定内置 `main` Agent `Alter0`，作为通用对话入口；`Agent` 页面承载无独立入口的内置 Agent 与用户管理 Agent。
- Web 登录态下，Chat/Agent 按目标 Agent 隔离会话历史，具备独立前端入口的 Agent 不进入通用 Agent 页面历史。
- `Agent` 运行页的会话历史需展示可复制的 8 位短 hash 标识，作为会话级引用与人工排障的稳定标识符。
- Web 入口稳定提供根路径到 Chat 的默认进入、`/chat`、`/login` 与 `/logout`，登录密码启用后受保护页面和 API 统一走同一登录态校验。
- 新会话先使用占位标题，早期多轮内可根据更具体输入自动升级标题，避免长期保留“拉取仓库”“分析仓库”等低辨识度名称。
- 新对话空白会话保持唯一；已有空白会话时，`New Chat` 复用并聚焦该会话。
- 同一会话内同步请求保持顺序一致，系统提供全局并发上限、排队与超时降级。
- SSE 流式响应提供 `start / delta / done` 与保活帧；Agent 工具循环期间还需追加结构化 `process` 事件，把步骤状态实时推送到前端。已进入 Agent 执行链的请求不因浏览器断连、页面切换或前端取消而中断后端执行。
- 流式连接中断时，前端保留已收到的正文并把消息收敛为失败态；若没有可用正文，失败提示需明确提示刷新，并在页面恢复时优先用服务端已持久化的会话消息覆盖本地失败态。本地缓存中残留的 `streaming` 消息不得长期停留在 `In Progress`。
- Agent 执行过程需以结构化 `process_steps` 贯穿 SSE `done`、Task 结果与会话历史持久化，前端优先消费结构化步骤而不是依赖解析 `[agent] action / observation` 文本。
- 消息区支持 Markdown 安全渲染、一键复制最终回复、Process 折叠状态、逐条 patch 与逐帧合并刷新。
- `Chat / Agent Runtime` Composer 支持图片附件草稿、缩略图预览与消息内图片回显；最近会话恢复仅持久化消息图片预览资产，避免重复保留原始大图 payload；助手 markdown 图片需在消息区直接以内联图片懒加载显示。带图消息只允许走支持视觉输入的模型链路，不进入异步 Task，也不静默降级到 Codex 文本执行。
- Web 前端所有时间显示统一使用北京时间（`Asia/Shanghai`）与 24 小时制；Cron 创建表单默认时区固定为 `Asia/Shanghai`。
- Web 侧边栏、历史折叠、页面滚动隔离、克制冷灰工作台阅读主题、PC 端低圆角非胶囊控件、移动端软键盘跟随、设置底部面板、低功耗轮询与长文本宽度约束作为统一前端体验要求维护。
- 会话侧栏中的 Session 列表需采用工作台式最近时间分组：按 `Today / Yesterday / Earlier`（中文对应 `今天 / 昨天 / 更早`）收口，并与主导航 `menu` 复用同一套分组容器、hover 与激活态视觉；条目按导航式线性关系排布，标题独立一行并在可用宽度内单行截断、摘要独立换行、短 hash 固定在条目下缘、删除动作以尾侧轻量文本操作收纳，不再拆出额外 footer 或胶囊操作区。
- Web Shell 由 React 单一工作台直接渲染：`src/app/WorkbenchApp.tsx` 负责 hash 路由、语言切换、主导航折叠/抽屉与运行页/控制页分派；运行页共享同一套 slot 化 workspace scaffold，`chat` 与 `agent-runtime` 通过 `ConversationRuntimeProvider + ConversationWorkspace` 渲染 terminal-style workspace，`terminal` 在保持原有交互与 DOM 契约的前提下直接挂在共享 `workbench-pane-shell` 下复用同一骨架，不再额外包裹 `route-view / route-body`，`agent / memory / channels / skills / mcp / models / environments / cron-jobs / sessions / tasks` 等页面继续由 React 直接请求控制台或会话 API 渲染。壳层稳定暴露 `app-shell[data-workbench-route]` 与各视图自己的 `data-route / data-conversation-*` 作为样式钩子；`legacy` 资源仅保留兼容样式，不再保留 `LegacyWebShell / ReactRuntimeFacade / bridge / snapshot store`。
- `/chat` 与 `/login` 默认以英文启动，HTML 根节点语言标记为 `en`；Web Shell 保留显式语言切换入口，切到中文后需同步更新壳层文案与 `document.documentElement.lang`。
- 登录页需与工作台共享同一视觉基线：使用 `IBM Plex Sans + Sora` 字体组合、近白卡片表面与安全入口语气，避免退回默认系统登录页样式。
- Web Shell 的稳定视觉基线收敛为两层：左侧固定主导航负责品牌、路由与语言切换，右侧主面板统一承载运行页和控制页；`Chat / Agent Runtime / Terminal` 在主面板内部统一采用「会话列 + 主时间线工作区 + 底部 Composer + 固定 workspace header」结构，并直接复用会话栏、workspace body、chat screen、composer 与窄屏 `Workspace actions` 语义 class；移动端顶部只保留一个四宫格入口按钮，面板内集中承载 `Menu / Sessions / New`；固定 workspace header 只保留会话标题、状态按钮与 `Details` 入口，具体会话详情与页面差异化配置统一放入 `Details` 面板，且面板首屏默认以紧凑摘要栅格承载高频字段。Composer 统一采用“上层输入区 + 下层工具栏”两层结构：左侧收口为单一 `Session` 会话设置入口与附件按钮，配置在面板内部按 `Agent / Model / Tools / Skills` tab 切换，右侧只保留发送动作；输入区需要保持足够横向留白与可读宽度。`Terminal` 继续保持原有 `terminal-*` DOM class 契约与布局关系，三者状态与交互全部由 React 直接维护。常规工作台页面保持近白表面、低对比边框、浅灰说明层和浅蓝选中态，不再为不同页面维持分散的高装饰视觉语言。
- `Agent` 与其他 React 托管页面共享同一 restrained workbench surface system：列表卡片、管理表单、托管字段块与消息块使用一致的近白主表面、浅灰辅助层和浅蓝选中态。
- `/chat` 与登录页的对外品牌文案统一使用 `Alter0`：浏览器标题、登录标题、导航品牌位、会话栏标题与欢迎区 tag 不再暴露小写服务名。
- Terminal 路由页继续由 React 原生实现，会话栏、工作区头部、Process、输出区和 Composer 的状态与交互全部由 React 维护；旧版 Terminal 仅作为布局关系与 `terminal-*` DOM 契约参照，不恢复 legacy runtime 控制器或脚本接管。
- 主导航、控制台与资产页默认以高密度信息架构呈现：导航组与条目间距压缩，控制面长列表优先使用表格或主从视图，不再把大量配置和任务详情平铺为低密度卡片矩阵。
- 移动端 Web Shell 使用 `VisualViewport` 驱动的 `--mobile-viewport-height` 与 `--keyboard-offset` 协调壳体和输入区：浏览器工具栏切换时壳体继续贴合可视区域，软键盘弹起时主工作区保持稳定高度，仅输入区按键盘偏移移动，不出现底部空白、内容裁切或整页上移。
- 移动端运行页的 `Menu` 与 `Sessions` 抽屉必须保持统一开合语义：二者共用同一份当前面板状态，始终互斥；打开其一时立即关闭另一侧，点击遮罩、切换路由、切换会话或创建新会话后不得残留旧的展开层。
- 移动端运行页的 `Menu / Sessions` 抽屉需优先保证真机稳定性：遮罩保留淡入淡出，抽屉本体仅保留一层轻量侧滑，不叠加多层位移、淡出或条目级顺序动画；抽屉内会话项按最近时间分组，并统一采用「状态文本 / 标题 / 摘要 / 底部短标识 / 尾侧删除」结构，避免退回松散白卡片或过度胶囊化。
- 共享运行时的短哈希预览 host 与主域工作台必须落在同一登录保护边界内：`/login` 可直接在预览 host 打开，登录态 cookie 需对 `*.alter0.cn` 生效，避免主域与预览子域重复维护独立会话。
- `Chat / Agent Runtime` 的移动端键盘弹出链路需与 `Terminal` 对齐：首次触摸输入框时使用 `preventScroll` 聚焦并在聚焦阶段持续锚定 `window.scrollY = 0`，不允许首次弹出软键盘时把公共操作行顶出可视区，也不允许因 `VisualViewport` 首次收缩造成页面整体分辨率/可视区域突变。
- `Chat / Agent Runtime / Terminal` 在移动端采用固定底部 Composer 时，消息滚动区与空态工作区都必须按当前 Composer 的真实遮挡高度动态回收；对话、长输出与空态说明不得落到输入区下方，也不得依赖静态 padding 估算占位。
- `Chat / Agent Runtime / Terminal` 在移动端的 Composer 回弹到底边时，运行区必须继续跟随释放旧的遮挡高度；键盘收起、输入框失焦和视口回弹后，不允许遗留额外底部空白、悬空按钮或上一轮键盘高度对应的占位残影。
- `Chat / Terminal` 在移动端键盘弹起与收回期间，只允许底部 Composer 按 `VisualViewport` 偏移贴住可见底边；顶部 `Workspace actions` 入口、紧凑 workspace header 和 Terminal 四键定位条都保持原位，不跟随键盘位移做额外动画。
- `Chat / Terminal` 的移动端发送按钮必须支持在软键盘保持打开时直接点按提交；首触发送不允许先消费成键盘收起或焦点切换，再要求第二次点击才真正发出请求。
- 运行页 Composer 的键盘跟随只依赖 `VisualViewport` 同步后的实时定位，不额外叠加 `bottom` 过渡动画；键盘收起与输入区回弹阶段应保持直接、稳定的回贴节奏。
- 输入框 blur 后，若 `VisualViewport` 尚未恢复到最终高度，`--keyboard-offset` 不得提前清零；运行页应沿着实际视口回弹过程逐步释放键盘占位，避免底部输入区和正文区出现闪烁。
- Web Shell 的抽屉式单列布局仅在主视口宽度 `1100px` 及以下触发；高于该阈值时保留左侧固定主导航与右侧主面板。进入窄屏后主导航切换为贴边抽屉，`Chat / Agent Runtime` 会话列也在同一阈值切为独立左侧抽屉，由工作区头部的 `Sessions` 入口显式打开；`Terminal` 与其他 `page-mode` 页面继续保持单主面板，但 `page-mode` 路由页标题上方必须稳定提供 `Menu` 入口；`760px` 及以下再进一步压缩按钮与间距，避免窄屏下出现不可触达区域。主导航抽屉和 Conversation 会话抽屉都必须独立承担纵向滚动，小高度视口下不允许出现菜单或会话列表被裁切且无法滑动的状态。
- 窄屏主导航抽屉点击任一路由项后需立即关闭；页面切换完成后不得继续保留旧菜单层覆盖在目标页之上。
- 窄屏主工作区按页面类型收口为贴顶起始区：普通 `page-mode` 路由页与 `Terminal` 继续采用“两行头部 + 贴顶正文起始区”节奏，第一行承载抽屉入口与主操作，第二行承载当前标题；`Chat` 与 `Agent Runtime` 空态复用 terminal-style 顶部操作行，同时保留与 Terminal 对齐的单行紧凑 workspace header 显示当前会话标题、状态按钮与 `Details` 入口，但不再额外输出模型、工具或目标摘要文案；所有页面都不得在顶部遗留额外大块留白。
- 窄屏 `Chat / Agent Runtime` 工作区头部固定保留单一 `Workspace actions` 入口；面板内集中提供 `Menu / Sessions / New`，其中 `Menu` 与壳层主导航抽屉共用同一开关状态，`Sessions` 单独控制 Conversation 会话抽屉，`New` 直接创建当前路由对应的新会话，不再出现移动端无导航入口或只能依赖正文内按钮切换会话的状态。
- `Chat / Agent Runtime` 工作区头部固定为共享单行 header：仅保留会话标题、状态按钮和 `Details` 入口，不再在头部直接放置 `Model / Tools / MCP / Agent` 选择控件，也不重复展示 `Chat / Agent` 标签和目标摘要；模型、Agent、Tools / MCP、Skills 以及会话元数据统一在 `Details` 面板中展示和调整，面板首屏先以紧凑摘要栅格承载会话元数据，再承接页面专属配置区。两条运行页的模型区除常规 LLM Provider / Model 外，都需额外提供内置 `Codex` 直选项，选中后仅影响后续消息，并把执行链显式切到 `Codex CLI`；Agent Runtime 的 Agent 选择列表不展示 `Alter0/main` 主助手，并额外提供 `Session Profile` 视图，用于展示当前 Agent 预设字段和当前 Session 实例属性。Agent Runtime 的 `Skills` 面板需区分当前 Agent 私有 Skill 与公有 Skill：私有 Skill 固定展示为已启用且不可取消，公有 Skill 才进入可选列表；例如 `travel` 的旅游领域规则属于私有 Skill，`deploy-test-service` 这类部署基础能力属于公有 Skill。`Details` 以顶层浮层形式打开，面板内部独立滚动；再次点击当前 `Model / Tools / MCP / Skills / Session Profile` tab 只收起该 tab 的内容区并保留摘要与面板，点击浮层外区域或按 `Escape` 才关闭整个面板，且不得推动消息区或会话正文重新排版。
- `Chat / Agent Runtime` 首页 Composer、会话卡片与 `Details` 面板需维持同一套浅色 terminal-runtime 表面系统：Composer 采用“上层输入区 + 下层工具栏”的双层结构，工具栏左侧提供工作区工具、附件与 meta，右侧收口发送动作；桌面与移动端都要控制输入高度、底部留白和发送按钮体量，并保持输入区满宽铺开、具备足够横向留白；发送按钮直接复用 Terminal 的紧凑 icon submit 皮肤；PC 端上传、发送、状态、详情、流程入口、短 hash 与弹窗动作统一采用低圆角矩形，不使用胶囊按钮或胶囊标签；会话卡片和详情面板不再退回旧式轻表单或松散卡片观感；空态工作区使用低对比网格与细弧线背景，同时禁止保留可拖拽滚动，不得把头部操作行或输入区顶出可视区。
- `1100px` 及以下的移动工作台需优先保证真机滚动与抽屉切换流畅度：主工作区、Conversation/Terminal 抽屉遮罩、抽屉面板本体与运行页容器不得继续依赖大面积 `backdrop-filter`、持续背景光晕或其他会导致整页重绘的装饰层，统一保持静态浅色表面。
- `Terminal` 窄屏工作区头部不得重复输出会话抽屉入口；`Sessions` 入口统一由壳层头部提供，工作区头部仅保留与当前会话直接相关的操作。
- `Chat` 空态首屏在桌面与中宽度下必须保持居中首屏节奏：欢迎区标题、描述、target/prompt 需在 header 与 Composer 之间沿欢迎区中轴竖向居中展示；真窄屏继续贴近头部下沿起排。Composer 继续沿主工作区自然贴底排布；不允许通过 `margin-top: auto`、过大的欢迎区上边距或类似弹性占位把输入区推到底部，造成首屏中上部出现大块无效空白。
- 移动端 Chat/Agent 在主输入框与会话设置底部面板之间切换时，不允许保留“键盘 + 设置面板”双重底部占位：打开设置前先释放输入焦点并清理键盘偏移，回到主输入框时先自动收起设置面板；Terminal 在真手机宽度下允许工作区头部工具栏换行，操作按钮不得被长标题挤出可见区域。
- 桌面宽屏下 Chat 消息列与底部输入区需按主工作区宽度自适应扩展，并统一收敛到居中的 `960px` 最大阅读宽度，避免正文与输入区无限拉长。
- 四键阅读定位条仅保留在 `Agent` 运行态消息区与 `Terminal` 对话区，采用右侧箭头四键承载 `回到顶部 / 上一条 / 下一条 / 回到底部`；普通 `Chat` 消息区与其他托管 route 页不展示该控件，定位目标按当前视口中的可见消息块或 Terminal turn 动态计算。

## Agent Capability & Memory

核心对象：`AgentProfile`、`AgentCatalog`、`ReActAgentConfig`、`ToolExecutor`、`Skill`、`MCPServer`、`MemoryFile`、`MemoryContext`、`AgentSessionProfile`、`LongTermMemory`、`DailyMemory`、`SOUL.md`。

稳定需求：

- 运行时统一聚合内置 Agent 与用户管理 Agent；内置 Agent 包括 `main`、`coding`、`writing` 与 `travel`。
- Agent 采用 ReAct 执行链，负责理解用户目标、吸收 system prompt / Skill / Memory 与运行时上下文，并把具体执行交给 `codex_exec`。
- 稳定工具面包含 `codex_exec`、`search_memory`、`read_memory`、`write_memory` 与运行时收口工具 `complete`；允许委派的 Agent 可额外使用 `delegate_agent`。
- `travel` 的显示名称为 `Travel Agent`；该 Agent 除正常对话答案外，还必须额外生成一份 HTML 旅游攻略，并发布到当前 Session 的公开只读子域名 `https://<session_short_hash>.travel.alter0.cn`。
- `codex_exec` 通过 stdin 传递最终指令；存在可用 Provider 且进入 Agent / ReAct 链路时，仅向 Codex 下发当前步骤指令；不存在 Provider、Agent 初始化失败或请求直接进入 Terminal / 直连 Codex 时，运行时会为当前会话生成原生 `CODEX_HOME/config.toml`、工作区 `AGENTS.md` 与 `.alter0/codex-runtime/*`，把 `runtime_context`、`skill_context`、`mcp_context`、`memory_context` 编译成 Codex 原生运行配置与工作区事实。
- Agent / ReAct 走 `openai-completions` 多轮工具调用时，assistant `tool_calls` 与后续 `tool` 结果的 `tool_call_id` 必须保持同轮关联，不能在 Provider 适配层丢失。
- Agent Profile 支持名称、system prompt、max iterations、Provider/Model、工具白名单、公有 Skills、MCP 与 Memory Files。
- 每个 Agent 自动拥有私有 file-backed Skill `.alter0/agents/<agent_id>/SKILL.md`，用于沉淀可复用工作模式、输出结构、检查清单与稳定偏好；私有 Skill 始终随当前 Agent 注入执行上下文，不受前端取消或 `alter0.skills.exclude` 排除影响。
- Memory Files 支持 `USER.md`、`SOUL.md`、当前 Agent 私有 `AGENTS.md`、长期 `MEMORY.md / memory.md`、当天与前一天 Daily Memory，并在注入时携带路径、存在状态、可写性、内容与自动召回片段。
- `Agent Session Profile` 固定落在 `.alter0/agents/<agent_id>/sessions/<session_id>.md`，由运行时自动维护并注入执行链路；该文件除会话画像外，还负责沉淀当前 Agent 当前 Session 的结构化实例属性。实例属性支持通过请求 metadata 增量更新；`coding` 默认自动维护仓库、分支和预览子域名等属性，`travel` 等专项 Agent 可维护 `city` 等领域属性。每个 Agent 还需支持独立的 Session Profile 预设字段定义，运行时与前端以同一字段集展示当前实例值。执行前还需有一条独立的旁路抽取链路，根据 Agent schema 从本轮自然语言更新可写字段，默认可退化为受限 Codex 窄调用。
- 会话短期记忆、跨会话长期记忆、上下文压缩、天级记忆、强制上下文文件与任务摘要记忆统一构成 Memory 领域能力。
- `Agent -> Memory` 页面提供长期记忆、天级记忆、强制要求、任务历史与说明文档的只读可视化入口，并支持任务摘要重建。

## Task, Terminal & Workspace

核心对象：`Task`、`TaskSummary`、`TaskLog`、`ArtifactRef`、`TerminalSession`、`TerminalTurn`、`TerminalStep`、`Workspace`、`CodexThreadID`、`RuntimeHeartbeat`。

稳定需求：

- 高复杂度、长耗时或产物型请求可切换为异步 Task，先返回任务卡片，再通过任务视图、日志流与会话回写完成闭环。
- Task 需建立 `session_id`、`source_message_id`、`channel_type`、`trigger_type`、`correlation_id`、Cron 触发信息与产物引用的标准映射。
- Task 观测台支持列表、详情抽屉、来源筛选、日志 SSE、游标续读、日志回补、retry/cancel、交互式续写、任务-会话双向跳转与完成结果回写。
- Task 观测台桌面端优先采用左侧任务列表 + 右侧详情面板的主从布局，详情区承载元数据、日志、产物、控制动作与 follow-up terminal 输入。
- Terminal 页面 Composer 支持最多 5 个附件，稳定覆盖图片与常见文本/文档文件：图片继续提供缩略图预览、纯图片发送与图片回显，并先写入当前 Session 工作区附件目录后仅提交 `asset_url / preview_url` 引用；普通文件同样先落到同一附件目录并只提交稳定附件引用，执行前再写入当前 Terminal 工作区 `input-attachments/<turn_id>/` 供 Codex 按路径读取。Terminal `Details` 面板支持选择控制面中启用且非私有的公有 Skill，并在发送输入时把 `skill_ids` 编译进当前 Terminal 工作区的原生 Codex Runtime。Task 详情抽屉中的 follow-up terminal 输入当前稳定支持图片附件，并继续透传到统一消息元数据。
- Task 记忆视图支持任务摘要、任务详情、日志下钻、产物引用与摘要重建，用于把历史任务纳入长期上下文召回；任务历史默认以表格承载摘要元数据，再通过详情侧栏查看长文本与日志/产物入口。
- Codex CLI 长任务按心跳续租运行窗口；列表与详情展示 `Last Heartbeat` 和 `Timeout Window`。
- Web 会话不直接暴露本地文件路径，产物通过引用、下载或预览接口交付。
- 默认工作区按执行上下文隔离：Chat/Agent 使用 `.alter0/workspaces/sessions/<session_id>`，Task 使用其会话下的 `tasks/<task_id>`，Terminal 使用 `.alter0/workspaces/terminal/sessions/<terminal_session_id>`。
- Chat / Agent Runtime 的会话图片资产需要随 Session 工作区落盘：用户上传图片的原图与预览图统一写入 `.alter0/workspaces/sessions/<session_id>/attachments/<asset_id>/`，前端持久化与消息请求默认复用 `asset_url / preview_url` 引用；assistant 最终回复里的外链 markdown 图片也应在会话返回与落库前改写到同一路径下的本地附件 URL。
- 直连 Codex 的 Chat / Agent 会话会在各自工作区下额外维护 `.alter0/codex-runtime/` 与 `.alter0/codex-runtime/codex-home/`；Terminal 会话会在 `.alter0/workspaces/terminal/sessions/<terminal_session_id>/codex-home/` 下维护独立 `CODEX_HOME`。
- Terminal 是独立会话式终端代理，持久化 Codex CLI 线程标识、会话状态、标题、工作区、日志与步骤视图索引。
- Terminal API 支持会话创建、列表、恢复、输入、删除、详情读取以及 turn/step 明细读取，前端可按步骤展开或检索执行细节。
- Terminal 会话态统一为 `ready / busy / exited / interrupted`，执行态在 turn/step 维度维护 `running / completed / failed / interrupted`；运行态退出后保留历史，继续发送即可恢复。
- Terminal 恢复默认优先复用已持久化 Codex CLI 线程；若续写命中远端 compact 失败，则保留原会话历史与工作区，并在下一次输入时自动改用同会话下的新线程继续执行。
- Terminal 会话删除统一从会话列表触发，`Delete` 会同步清理状态文件和独立工作区；工作区头部不再提供单独的 `Close` 入口。
- Terminal 历史在同一 Web 登录态下跨设备共享，不按浏览器 client 标识隔离；不设置产品级会话数量上限或固定超时淘汰。
- Terminal 移动端、输入稳定性、滚动导航、Process 折叠、一键复制、长输出阅读、轮询降频与缓存写入节奏作为 Terminal 子域体验要求维护。
- Terminal 发送按钮首次点击必须立即进入 pending 反馈；若当前还没有 active session，前端允许先创建会话再继续发送，但首击期间按钮需同步切到 `Sending...` 与禁用态，避免重复点击和“第一次点击无反应”的错觉。
- Terminal 刷新节奏需按会话状态自适配：执行中的会话保留实时刷新，空闲会话收敛为低频轻量刷新；用户正在滚动阅读输出时，不得因明细轮询而打断当前滚动。
- Terminal 窄屏消息页必须保持 `workbench-main -> chat-pane -> terminal-view -> terminal-chat-screen` 的闭合高度链，由 `terminal-chat-screen` 独立承担纵向滚动；外层容器不得因 `overflow: hidden` 或高度塌陷吃掉滚动。
- Terminal 移动端在输入框聚焦且软键盘抬起后，Composer 必须按 `VisualViewport` 同步的键盘偏移直接贴住可见底边；长对话或长输出期间不得通过拉高 footer padding、改变滚动容器或破坏高度闭合链把输入区挤出屏幕。
- Terminal 移动端的 `terminal-chat-screen` 必须继续按当前 Composer 的真实遮挡高度动态收口；会话空态、长输出与 Process 阅读都要稳定停在输入区上沿，不允许被 fixed Composer 覆盖。
- Terminal 移动端的四键阅读定位条只按静态 Composer footprint 停靠，不跟随软键盘位移动态上移；键盘弹起时按钮组保持原位，键盘收起或浏览器视口回弹后继续稳定停在 Composer 上沿之上，不得留下悬空残影。

## Control, Operations & Governance

核心对象：`ChannelConfig`、`SkillConfig`、`AgentProfile`、`ModelProvider`、`EnvironmentConfig`、`CodexAccount`、`CodexLoginSession`、`RuntimeInstance`、`DeploymentBaseline`、`EngineeringPolicy`。

稳定需求：

- Control API 管理 Channel、Capability、Skill、MCP、Agent Profile、Cron Job、Model Provider、Environment 与 Codex 多账号配置，并保留 Capability 生命周期审计。
- 服务启动后默认提供 `default-nl`、`memory`、`deploy-test-service` 与 `frontend-design` 四个内置 Skill；其中 `deploy-test-service` 固定落在 `.alter0/skills/deploy-test-service/SKILL.md`，`frontend-design` 固定落在 `docs/skills/frontend-design/SKILL.md`。`coding` 内置 Agent 默认启用 `memory`、`deploy-test-service` 与 `frontend-design`，用于覆盖仓库记忆、预览发布和前端页面/组件实现质量。
- 共享 Web 运行时需要支持通用 workspace service 注册：`GET /api/control/workspace-services` 查询注册表，`PUT /api/control/workspace-services/{session_id}` 绑定默认 `web` 服务，`PUT /api/control/workspace-services/{session_id}/{service_id}` 绑定附加服务，`DELETE` 接口用于清理绑定；当请求 Host 命中 `<session_short_hash>.alter0.cn` 或 `<service>.<session_short_hash>.alter0.cn` 时，共享运行时需按注册类型分发前端构建或反向代理到目标 HTTP 服务。`travel` 服务是唯一例外，固定命中 `https://<session_short_hash>.travel.alter0.cn`，且该 host 只读、免登录，只允许返回静态 HTML/资源。标准 `web` 部署默认应把当前会话后端启动命令注册给共享运行时托管，再以 `http` 方式绑定短哈希子域名，确保前端与 `/api/*` 同时来自当前分支；`frontend_dist` 仅作为静态预览模式保留。
- Channels 入口归属 Settings 模块，旧直达路由保持兼容。
- Models 控制面支持 OpenAI Compatible 与 OpenRouter Provider，支持 `/responses` 与 `/chat/completions`，支持 base URL、API Key 保留语义、Provider 路由偏好、默认项自动收敛与历史缺密钥配置恢复。
- `openai-completions` 适配层必须正确序列化 assistant `tool_calls` 与 tool output，兼容严格校验工具消息配对关系的上游 Provider。
- Environments 页面支持 Web/Queue、Async Tasks、Terminal、Session Memory、Persistent Memory 与 LLM 运行参数可视化配置、敏感值显隐、配置审计、在线实例启动时间与 commit hash 展示、运行时重启、远端 master 快进同步、候选二进制构建、readyz 探活与失败回滚。
- Settings 页面提供 Codex Accounts 面板，使用高密度运行时概览、当前 Codex 管理区、托管账号列表与操作侧栏承载 `auth.json` 导入、独立登录会话、托管账号状态查看、当前运行时账号切换，以及基于 Codex app-server 真实能力返回值的活动 model / 思考深度切换；概览区采用当前账号主身份区配合套餐、小时/周剩余额度和托管数量的紧凑指标列，其中额度必须以进度条展示并附带 reset 时间，key/value 默认按同列对齐，概览本身不展示活动 auth 路径，维护类信息通过 `Runtime Details` 折叠区展开；当前 Codex 管理区仅保留一套可编辑的 model / 思考深度字段，不重复展示当前值摘要；托管账号区采用高密度行式列表，在不同断点下持续暴露账号身份、套餐、额度进度条、reset 时间、当前 model、思考深度与切换入口，并使用更平的控制台控件与分隔线布局代替嵌套胶囊和内嵌方框；若当前运行中的 `auth.json` 尚未导入托管仓库，页面仍需展示该活动账号并提示其处于未托管状态，且在 quota 已可用时继续展示该 live 账号的套餐与额度，加载阶段需保留整页骨架布局。
- 公网部署基线要求服务绑定 localhost、启用 Web 登录密码、统一 `HOME=/var/lib/alter0`，并通过 Nginx 做反向代理。
- 服务内 GitHub 交付要求运行账户具备 GitHub App token helper、`gh` 包装器、SSH 提交签名、稳定 PATH 与 Codex CLI 可用认证。
- Node/Playwright 测试链路通过运行账户级工具链初始化，保证 Codex CLI 可执行 `internal/interfaces/web/frontend` 的构建与单测，以及 `internal/interfaces/web` 的 Playwright E2E。
- 研发流程遵循 TDD：功能新增、缺陷修复、行为调整与重构默认先以测试表达目标行为，再完成实现与重构；纯文档、注释、格式化、依赖元数据或无法自动化验证的变更需说明免测原因与替代验证。

## 维护规则

- 新需求必须先选择领域，再选择子域；无法归类时优先补充领域模型，而不是新增线性编号。
- 一个需求只允许有一个主归属领域；跨领域影响通过 `依赖与边界` 说明，不复制成多个重复需求。
- 用户可见行为、交互方式、入口路由、执行模式、返回结构或默认策略发生变化时，同步更新 `README.md`。
- 需求细节、接口、状态、验收和边界放入对应 `requirements-details/*.md` 文件；技术方案、包边界、调用链路、存储、观测和测试策略放入 `technical-solution.md` 的同名领域下；`requirements.md` 只维护稳定总览与领域索引。
