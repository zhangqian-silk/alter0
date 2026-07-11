# Conversation & Session Experience Requirements

> Last update: 2026-07-08

## 领域边界

Conversation & Session Experience 负责用户在 Web/Chat/Settings 页面中的会话、消息、过程展示、移动端适配、输入稳定性和阅读体验。它消费 Runtime、Skill、Task 的执行结果，但不定义底层执行器行为。

## 核心对象

| 对象 | 职责 |
| --- | --- |
| `Session` | 会话身份、标题、历史归属与生命周期 |
| `Message` | 用户与助手消息主数据 |
| `LiveUserMessage` | 执行中补充输入与当前轮可见用户意图 |
| `RuntimeTraceEvent` | Chat 共享的结构化运行过程事件 |
| `SessionHistoryBucket` | 按 Skill 或入口隔离的历史集合 |
| `ViewportState` | 移动端可视视口、键盘、滚动与输入状态 |

## 会话入口

### Web Shell

- 根路径 `/` 默认进入 Chat 工作台。
- `/chat` 是唯一主工作流入口。主导航底部固定提供 `Settings` 工具入口并进入 `/settings`。Runtime、Skills、Memory 与 Schedules 统一收敛到同一个 Settings 页面内分组展示，不再保留独立工作台 path，并继续复用同一 Web Shell、登录态、主导航与页面骨架。
- Web Shell 的前端构建源位于 `internal/interfaces/web/frontend`，`/chat` 固定分发 `static/dist/index.html`；该入口仅保留前端挂载容器与静态资源引用，由 React 渲染稳定的 shell DOM，并通过兼容样式层保持旧 DOM 契约。
- `/chat` 与 `/login` 默认以英文文案和 `html[lang="en"]` 启动；Web Shell 导航中的语言切换入口负责在英文与中文之间切换，并同步更新根节点语言标记。
- 登录页在启用密码保护时继续作为统一入口，但视觉需与 Web Shell 保持一致：使用 `IBM Plex Sans + Sora` 字体、近白工作台卡片和安全入口说明文案，不保留独立的默认系统表单风格。
- Web Shell 由 React 单一工作台直接渲染：`src/app/WorkbenchApp.tsx` 负责 `/chat`、`/settings` 两个稳定顶层路由、语言切换、主导航抽屉和运行页/设置页分派；桌面宽屏保留固定主导航，不再提供折叠侧栏阶段；主导航的主工作流只暴露 `Chat`，并用单个 `Settings` 工具入口进入设置页；`RuntimeRouteHost` 只挂载 `chat` path 到 `ConversationRuntimeProvider + ConversationWorkspace`。Runtime、Skills、Memory 与 Schedules 由 `/settings` 页面内的本地分区切换接管。根壳层稳定暴露 `app-shell[data-workbench-route]` 与 `data-route-family="settings"`，各运行页与 route body 继续输出 `data-route / data-conversation-*` 作为样式与测试锚点。
- `channels / skills / mcp / models / cron-jobs` 共享控制台页统一复用同一组响应式内容网格；窄屏下标题区允许徽标下沉、字段行改为单列堆叠、底部标签区保持纵向拉伸，避免复制按钮、状态徽标与多行字段发生重叠或横向溢出。
- Settings 桌面布局需使用左侧设置索引和右侧内容区，分区入口提供图标、短标识与活动态；真手机宽度下索引切换为双列入口栅格，保证 Runtime、Skills、Memory 与 Schedules 都可直接扫读。各分区内部的表格、筛选条、详情面板、空态和错误态需共享扁平白底、轻量分割线、浅灰辅助层和紧凑字段行，不再默认使用外层卡片边框、厚圆角或重阴影表达层级。
- `Skill` 与其他 React 托管页面共享同一扁平表面体系：列表、管理表单、托管字段块与消息块使用一致的白底主表面、浅灰辅助层与低对比选中态，不再默认使用卡片边框、厚圆角或重阴影表达层级。
- `/chat` 页面标题、登录页标题、导航品牌位、会话栏标题与欢迎区 tag 统一展示 `Alter0`，不再混用 `alter0` 小写品牌词。
- Web Shell 的抽屉式单列工作台仅在主视口宽度 `1280px` 及以下触发；高于该阈值时保留左侧固定主导航与右侧主面板，避免只对聊天内容列做最大阅读宽度限制而让整体壳层失衡。
- 进入窄屏工作台后，主导航切换为贴左侧视口边缘的全高抽屉；当前运行页的会话列表随主导航一起进入同一个左侧抽屉，不再在工作区内部生成独立浮层或上下堆叠列；真手机宽度下抽屉使用近白全高面板、平面菜单和细分割线，会话区按自然高度滚动，不再出现重卡片式列表容器；`info-mode` 页面继续只保留主导航抽屉，避免壳层断点先切换而页面主体仍保留桌面列布局。
- 窄屏主导航抽屉中的菜单区必须作为独立滚动容器保留纵向滚动能力；一级菜单固定为 `Chat`，语言切换入口保持可触达，不得出现底部菜单项被裁切且无法继续下滑的状态。
- 窄屏主导航抽屉点击任一路由项后需立即收起；切页操作不会保留旧的菜单遮罩或抽屉层，用户进入目标页后直接看到新的正文区域。抽屉关闭后遮罩层不得继续挂载为可命中元素或拦截触控，顶部 `Menu / New`、输入区发送、附件和会话设置等按钮必须立即恢复可点击。
- 窄屏主工作区按页面类型收口为贴顶起始区：Management 管理页继续采用两行头部，第一行承载 `Menu`，第二行承载统一标题，并在正文起始处提供页内分组切换；中等窄屏允许分组入口横向滚动，真手机宽度必须回落为双列换行入口，避免进入 `Tasks / Sessions / Models` 等管理能力后失去主导航入口或管理分区入口；`Chat` 在真手机宽度下统一采用单层运行页 workbar，不再叠加第二层 workspace header。运行页标题区必须在一行内承载当前会话标题与状态信号，并通过点击标题打开 `Details`；不得在操作行下重复输出模型、工具或目标摘要，也不得把详情入口再拆成额外按钮层。所有页面正文都需贴近头部下沿起始，不得在顶部留下额外大块空白；Chat 加载态的 `Thinking` 披露、用户消息和进行中占位必须沿时间线顶部自然排列，不得被满高滚动容器或后加载键盘隔离样式居中、反向拉开或推到中部。
- `Chat` 空态欢迎区采用紧凑首屏节奏：桌面与中宽度下，欢迎 tag、标题、描述、target picker 与快捷提示需在 header 与 Composer 之间沿欢迎区中轴竖向居中展示；真窄屏继续贴近头部下沿起排。Composer 直接按自然文档流沿主工作区底边贴底排布；桌面与窄屏都不再通过自动顶距把 Composer 推到底边，避免欢迎区与输入区之间出现大块空白。
- 桌面端主导航采用紧凑间距节奏，主工作流只保留三条，并在底部固定 `Management` 工具入口；控制类与资产类路由在 Management 页内部优先使用高密度主从或表格视图，Management 分组导航自身作为左侧索引常驻，避免在宽屏上保留大块无效留白或把分区入口堆成顶部标签云。
- `static/dist/assets/*` 使用构建产物哈希文件名并返回长期 immutable 缓存；`/chat` 与 `static/dist/legacy/*` 下的兼容样式资源保持 `no-cache`，确保页面与样式能及时刷新到最新版本。
- `/login` 提供统一登录入口；`/logout` 清理当前登录态并回到登录流程。
- Web Shell、短哈希预览 host 与受保护 API 统一使用同一登录态校验；静态只读预览 host 保留匿名访问。

### Chat

- `Chat` 面向通用对话入口，默认直接通过 Claude Code CLI 或 Codex CLI 执行。
- `Chat` 不再绑定内置 `main` Skill，也不再默认调度内置专项 Skill。
- `Provider / Model` 与 `Skills` 可在 Chat 会话过程中调整，并作用于后续发送的消息；Tools / MCP 继续由服务端运行配置注入，不再提供独立 Composer 控制面板。`Chat` 的 `Provider / Model` 选择器额外暴露内置 `Codex` 项，允许用户不经过常规 LLM Provider 直接切到 `Codex CLI` 执行链。Web `Chat` 不再提供独立空态、私有 Skill 面板或会话级目标切换；旧 Chat 会话加载时仅迁移为 Chat 会话并保留目标 Skill 名称作为历史元数据。

### Settings 页面

- `Skill` 页面仅承载用户管理 Skill 的配置与历史兼容能力，不再由服务启动时注入内置业务编排。
- Skill 选项卡片在配置面板中展示短摘要，完整 system prompt 不直接暴露在选择面板。
- `Chat` 与历史运行时会话共用消息阅读和输入体验规范；新 Chat 请求默认按当前会话的模型、工具、MCP 与 Skill 选择直接进入 CLI 执行链。

### Session 历史

- Web 登录后，Chat 已发送会话通过服务端 Chat session store 在同一 Web 登录态下跨设备共享；每个 Chat 会话使用后端生成的短 canonical id，格式为 `c_` 加 16 位小写字母数字。旧 `alter0-chat`、`chat-*` 长 id 和浏览器旧短 hash 只视为历史残留，不再迁移为当前会话。
- Chat 消息接口接受请求后，服务端先把本轮 `user` 消息写入 Session history，再进入同步执行；assistant 回复在执行完成、失败或任务收口后追加写入。同一轮请求的浏览器关闭、刷新、请求断开或前端取消不会让用户已发送内容只留在浏览器缓存中。
- Session history 维护会话级 `last_active_at` 与 `pinned`。`last_active_at` 在用户发送消息、assistant 完成或失败、结果收口、打开会话详情、Chat 输入/详情读取和任务结果写回时刷新；没有显式活跃时间的历史会话回退使用最后消息时间。
- Chat 会话列表把置顶会话汇入独立 `Pinned / 置顶` 分组并固定在 `Today / 今天` 上方；非置顶会话继续按最近活跃时间排序并进入时间分组。Settings 的 Sessions 页面展示最后活跃时间并提供置顶/取消置顶操作。置顶状态持久化在 Chat session store 中，不改变消息内容；尚未产生消息、只存在于当前浏览器的空白 `Chat` 会话，也必须在前端快照可用范围内保留置顶反馈。
- 系统维护任务默认每日清理超过 7 天不活跃的未置顶会话。清理会删除该会话的 Session history、运行时 registry、关联任务引用和 `<runtime_root>/workspaces/sessions/<session_id>` 下的附件/工作区数据；置顶会话始终跳过自动清理，仍有关联 queued/running 任务的会话在任务进入终态前跳过清理。
- 会话清理不提供复杂配置项。`Settings > Schedules` 的内置会话清理任务只提供当前状态、上次/下次运行、手动触发、失败重试，以及删除数量、置顶跳过数量、任务保护数量和扫描数量。清理后续资源删除失败时，本次维护状态必须记录为 `failed` 并暴露失败原因。
- 具备独立前端入口的 Skill 不进入通用 Settings 页面历史。
- `Sessions` 系统页面可展示跨来源会话数据，但不作为 Chat 分栏依据。
- 未发送文本草稿、附件草稿、GitHub 仓库选择草稿与当前浏览器中的临时空白会话允许继续本地保存；切换 Chat / Settings、切换会话或点击 New 时不得弹出丢弃草稿确认，原会话草稿按 route 与 session 继续缓存，返回后恢复。这些局部态不要求跨设备同步，但不能覆盖服务端已存在的会话摘要、配置与消息历史。

## 接口边界

- `GET /` 进入默认 Chat 工作台。
- `GET /chat` 返回 Web Shell。
- `GET /login` 与 `POST /login` 处理登录页和登录提交。
- `GET /logout` 清理当前登录态。
- `POST /api/chat/sessions/{session_id}/input` 处理 Chat owner 输入提交；`/api/chat/sessions` 不再作为 Web Shell 公开接口注册。
- `GET /api/chat/repositories?query=&cursor=` 使用服务端当前个人 `gh` 登录列出可访问的 GitHub 仓库，只返回稳定 id、`owner/name`、私有标记、默认分支和更新时间；前端不得读取 GitHub token、clone URL 或凭据。input 可选携带结构化 `repository { provider, id, full_name }`，服务端必须按稳定 id 重新解析仓库，不信任前端 clone 地址。
- `POST /api/chat/sessions/{session_id}/repository/retry` 只允许重试当前会话已失败的仓库准备任务；它复用原 turn、原用户输入与原 Skill 选择，不新增消息或 turn，也不得改变绑定仓库。
- 会话集合与详情的脱敏 DTO 需携带仓库显示名、固定相对工作区路径、默认分支、准备状态、可用 head 与脱敏错误；不得返回 token、clone URL 或服务端绝对路径。
- Web `Chat` 独立消息入口已移除；对话消息统一由 Chat owner 的 runtime session input 处理，运行页列表与详情由 `/api/chat/sessions` 接口恢复。
- 上述消息接口在 `content` 之外还接受 `attachments[]`；当前稳定支持两种图片输入：首次上传时携带 `data_url`、文件名与 MIME 类型，或在同一 Session 内复用已上传的 `id + asset_url + preview_url` 资产引用。允许仅发送图片，服务端会补齐稳定占位文本并把图片载荷并入统一消息元数据。
- `POST /api/sessions/{session_id}/attachments` 用于把会话图片提前写入当前 Session 工作区，并返回稳定 `asset_url / preview_url`。Conversation runtime 的草稿恢复、最近会话列表与已发送消息都应优先保存这组引用，不再长期持久化原始大图 `data_url`；其中 `preview_url` 只用于缩略图位，历史消息回显与预览弹层必须优先读取 `asset_url` 原图。
- Chat Composer 复用同一附件接口：图片先落到当前 Session 工作区，再以 `asset_url / preview_url` 引用参与提交；Chat 额外允许常见文本/文档文件直接走同一接口上传原文件，并在返回中仅保留稳定 `asset_url`。前端草稿、缩略预览与历史回显应优先消费这些稳定引用，而不是在这些链路里长期保留原始 `data_url`；其中缩略位继续使用 `preview_url`，再次查看时统一切回 `asset_url`。
- assistant 最终回复中的 markdown 外链图片也属于会话图片资产：服务端在返回最终结果与落库前，需要把可下载的 `http(s)` 图片拉取到当前 Session 工作区并改写成 `/api/sessions/{session_id}/attachments/{asset_id}/original` 这类本地附件 URL；下载失败时保留原链接，不影响主回复返回。
- `GET /api/chat/sessions` 返回 Chat owner 会话摘要，至少包含标题、Skills 选择、创建时间、内容更新时间、状态、置顶状态与稳定 session id。`updated_at` 只表示对话内容或 runtime 状态更新时间：用户输入、turn 创建、运行态推进、输出完成、失败/中断会推进它；pin/unpin、配置变更、标题管理、查看详情和列表刷新不得推进它。列表接口返回前需补扫当前持久化状态目录，只加载内存 map 缺失的 session，不覆盖正在运行的内存会话；服务重启、预览子服务切换或状态文件晚于进程启动出现时，列表仍需展示已持久化的历史会话和 `interrupted / failed` 等终态。集合接口是轻量 summary 契约，不返回完整 turns、附件原图、事件详情、`last_output_at / activity_at / runtime_session_id / owner_id / shell / working_dir` 等运行元数据；前端以 `updated_at` 判断会话状态新鲜度和详情是否需要回源。
- `POST /api/chat/sessions` 创建当前 owner 的真实 runtime session；服务端在返回前必须完成 session store 写入，并在全局会话锁外执行持久化、会话摘要级 `session.created` 事件发布和 HTTP 响应所需 bounded detail 构建。恢复、置顶和删除等会话级动作分别发布摘要级 `session.updated / session.deleted`；input 成功后的运行态变化由应用层 `turn.started` typed event 表达，不再额外发布摘要级 `session.updated`。事件发布链路不得反向阻塞创建或恢复请求，避免移动端 `New` 首触后长时间无响应。
- `GET /api/chat/sessions/{session_id}` 返回单个 Chat 会话详情，默认只返回最新 turns 页，并通过 `turns_paging` 提供总量、页边界、`next_before_turn_id` 与是否仍有更早内容；前端首次进入和刷新时，会在本地缓存首屏恢复后对当前 active server session 拉取一次不带 `turn_before` 的最新详情，用于校准最终正文、状态与 `runtime_trace_events`。普通页面激活只在本地仍处于 `local_running / recovering`、缓存不完整或存在可恢复占位时触发补偿；稳定 `ready` 会话不因 focus/pageshow/online 自动拉取详情。只有用户点击 `Load earlier messages / 加载更早消息` 或滚动到顶部触发历史加载时，才按 `turn_before` 显式请求更早 turns。详情至少包含 runtime `turns`、用户附件引用、轻量 `runtime_trace_events`、`updated_at` 与当前恢复到的运行态状态；详情读取是只读行为，不得推进 `updated_at`。所有时间字段均为毫秒时间戳，`finished_at` 无值时固定返回 `null`；展示层统一按 `Asia/Shanghai` 与 24 小时制格式化。前端仅在详情或增量事件携带完整 turns 时推进本地详情新鲜度，列表摘要里的空 turns 不能被视作已加载详情。
- `POST /api/chat/sessions/{session_id}/pin` 更新当前 owner 会话置顶状态，并在返回的 session payload 中显式携带 `pinned` 布尔值；取消置顶返回 `pinned:false`，前端刷新恢复必须以该服务端状态覆盖本地旧快照。
- `POST /api/chat/sessions/updates` 提供当前 owner 的增量轮询通道。客户端携带 `after_update_id`、`limit`、默认/最大 `1MiB` 的 `byte_limit` 与 `visible_event_kinds`；`after_update_id` 为空时从当前 owner 可续接窗口的最新位置读取，非空时只返回该 update 之后的变更。服务端返回 `latest_update_id`、`resync_required`、可选 `has_more` 与 `updates[]`。运行态变化必须由应用层发布 `turn.started / turn.event.appended / turn.event.updated / turn.completed / turn.failed / turn.interrupted` 语义事件；Web 层输出 direct typed update。Codex `agent_message.channel=commentary` 必须作为 `turn.event.appended` 发布单个可见过程 step；`channel=final` 或无频道内容只进入 turn 收口后的 `final_output`，不得作为过程 step 先下发再删除。单条 update 固定使用 `update_id / type / session_id / turn_id / runtime_event` 顶层结构，必要时携带当前 session 摘要和目标 turn patch；`turn.completed / failed / interrupted` update 只包含当前 turn 的最终状态、毫秒时间戳和可用 `final_output`，不携带完整 runtime step 历史。`session.created / session.updated / session.deleted` 只用于会话摘要、标题、置顶、状态、删除等用户可见会话级变化，不再承载常规 runtime step 增长。用户未勾选 commands/tools/system 等过程类型时，对应纯过程更新可不下发，但 `latest_update_id` 仍推进。单会话详情或 turn 收口事件若只补最终正文且没有携带 step，也不得清空本地已缓存过程。完整历史页、附件原图、event detail 以及 `owner_id / shell / working_dir / runtime_session_id` 等前端不消费运行元数据不进入增量 payload。
- 增量 update envelope 必须包含全局递增或可比较的 `update_id`、`type`、`session_id`、可选 `turn_id`、毫秒 `created_at` 与 payload。`update_id` 是 owner updates 续接游标，不是 UI 序号；当不可见事件被过滤、事件归属其他 owner 或响应受预算截断时，前端可见 update id 可以跳号。事件类型至少覆盖 `session.created`、`session.updated`、`turn.started`、`turn.event.appended`、`turn.event.updated`、`turn.completed`、`turn.failed`、`turn.interrupted` 与 `session.deleted`。`session.updated` payload 使用会话摘要 patch；大段 event detail、历史 turns、附件原图和超大 Markdown 正文继续按需通过详情接口读取。
- `Chat` 前端会话列表、详情刷新、输入、置顶、删除、附件、事件明细、历史分页、timeline item 构造和 model / Skill / MCP catalog 加载必须由共享 runtime session controller、runtime timeline builder 与 catalog hook 承担；`chat` 仍是 API owner、缓存和草稿命名空间分叉点，页面不得保留消息或 turn 的私有展示转换链路。
- `GET /api/sessions` 查询会话摘要列表，支持来源和时间过滤。
- `GET /api/sessions/{session_id}/messages` 查询会话消息。
- `DELETE /api/sessions/{session_id}` 删除会话，并触发关联工作区和任务清理。

## 会话生命周期

### 标题

- Chat 会话标题由底层 ChatRuntime session store 生成与维护；前端只能展示当前 session summary/detail/update 返回的标题，不得从首条 prompt、本地 queued user 消息或浏览器草稿派生并覆盖标题。
- 首次发送时，如果尚无真实 active session，前端需先创建无显式 title 的 ChatRuntime session，再向该 session 提交 input；创建请求不得携带首条 prompt 作为 `title`。
- 新会话先使用统一占位标题 `New`。
- Codex/Claude 等外部 CLI Runtime 暴露自身 thread title 时，ChatRuntime 必须优先采用该 title，并把会话标记为外部标题来源；同一底层 thread 后续再次发布 title 更新时继续覆盖当前标题。
- 外部 runtime title 更新必须持久化，并通过 `session.updated` 增量事件同步到当前 owner 的会话列表、workspace header 和移动端标题按钮。
- 已由外部 runtime title 接管的会话，不再被后续 prompt 自动标题或补充约束改写；只有新的外部 title 更新或显式会话级标题管理才能改变它。
- 早期多轮输入仍偏通用时，标题可继续等待更具体输入。
- 后续出现更具体目标后，标题需自动升级，不长期停留在“拉取仓库”“分析仓库”等低辨识度名称。
- 已形成主题型标题后，后续“而不是 / 不要 / instead of / rather than”等补充约束、修正说明或实现偏好不得覆盖当前主题标题；这类输入应作为消息内容和执行上下文保留。
- 读取历史会话时，如果旧自动标题已被后续补充约束覆盖，服务端需按最早有效主题 prompt 做一次纠偏并回写持久化记录；手动标题不参与纠偏。

### 空白会话

- 前端与后端链路不允许产生多个可见空白会话。
- 已存在空白会话时，`New` 复用并聚焦当前空白会话。
- 会话产生有效用户消息后，才进入普通历史会话生命周期。

### 持久化与恢复

- 用户与助手消息主数据、路由结果、时间戳、来源字段以及恢复运行页所需的请求 metadata 必须持久化。
- 用户消息中的图片附件需要和文本一起进入会话时间线；页面刷新、切会话和最近会话恢复时保留稳定的图片资产引用，不重复持久化原始大图 payload；时间线内图片与预览弹层再次查看时必须优先显示原图资源，避免缩略图被放大后失真。
- 未发送文本草稿在桌面端输入期间允许延迟写回浏览器缓存；当前输入值、切换前 flush、刷新后的草稿恢复和发送结果必须保持一致，不能为了持久化把每次按键都绑定到同步存储写入。
- 页面刷新、跨设备重开或服务重启后，用户可恢复最近会话与历史消息；恢复结果需保留当前 Session 的目标 Skill、Model 与 Tools / Skills / MCP 选择。
- 页面刷新时，前端需先用浏览器侧保存的当前活动会话快照恢复最近一条活跃 `Chat` 会话，避免服务端列表短暂缺席时把当前会话清空或替换为新的空白会话；随后再按 `session_id` 回源单会话详情，用服务端最新结果覆盖本地快照。
- `POST /api/chat/sessions/{session_id}/input` 在 Web 层接受请求后，后端执行与持久化不得再依赖浏览器连接持续存活；页面刷新、标签页切换、请求断开或前端取消只允许中断当前 HTTP 回传，不得直接取消本轮会话执行。
- `Chat` 的 URL query 只表达显式会话恢复：页面首次加载、刷新、手动粘贴 `/chat?session_id=<chat_session_id>` 或浏览器恢复带 query 的标签页时，Chat 先读取 `session_id` 恢复目标会话。`chat_session_id` 是后端生成的短 canonical id，格式为 `c_` 加 16 位小写字母数字，并与 `/api/chat/sessions/{session_id}`、updates payload、持久化文件和工作区路径完全一致。访问 `/chat` 或从主导航切回 `Chat` 时，工作台清理旧 `session_id`，并按服务端会话列表与本地最近快照的合并结果打开最新会话，避免上一次活动会话被 query 或 sessionStorage 固定。
- 浏览器侧会额外持久化最近会话列表的轻量快照，而不只保留当前活动会话；当用户刷新其他会话、切换设备前短暂刷新，或服务端集合接口暂时返回空列表、短列表、漏掉刚创建/最近活跃会话时，前端仍需在侧栏继续展示这些最近会话，并按 `session_id` 单独补拉详情，直到服务端明确确认不存在。
- `Chat` 在同一 SPA 工作台内切到 Settings 或其他页面再返回时，应优先使用浏览器内存级运行态缓存恢复各自 owner 的会话列表、当前活动会话和完整已加载消息；缓存 TTL 为 24 小时，不裁剪当前已加载消息。缓存按 `chat` 分桶，不允许 Chat 会话被 Chat 缓存覆盖。该缓存只服务路由切换后的首屏恢复，不替代服务端历史或刷新恢复快照；会话列表和单会话详情接口返回后必须继续按现有合并规则更新视图并刷新缓存时间，超过 TTL 的缓存不得参与首屏渲染。
- `Chat` 的完整消息快照与轻量会话信息快照读取时按 session 合并：同一 session 优先保留完整消息、过程事件、附件引用与 `turnsPaging`，轻量信息快照只补齐缺失会话摘要和列表可见字段。完整快照中已有某条会话时，不得因此丢弃轻量快照里的其他最近会话；轻量快照也不得把完整快照中的 `messagesLoaded=true`、过程详情或已加载历史降级为空列表。
- `Chat` 需复用 Chat session store 作为服务端会话事实来源，记录 `session_id -> title / skills / status / turns / pinned / updated_at` 等最小恢复视图；`updated_at` 是会话摘要与详情合并的新鲜度依据，不再维护独立版本字段。浏览器本地快照只作为次级兜底，不承担会话存在性的唯一事实来源。
- 删除会话时同步清理关联任务记录与会话工作区。
- `Chat` 会话列表统一由左侧主导航承载，使用 `Sessions` 标题与 `New` 新建入口；移动端通过同一个左侧导航抽屉展示当前Chat 会话列表。运行页互相切换时，左侧会话列表的 `Sessions` 标题与 `New` 按钮由主导航稳定持有，不随页面切换重建；当前运行页只更新数量文案、列表内容和 `New` 动作绑定，rail 数据尚未注册时使用稳定 fallback rail，已访问过的运行页切回时先复用该 route 最近一次有效 rail body，不得先清空公共 rail、回退占位 rail 再恢复。从 Chat 切到 Settings 时复用当前已注册 Chat 会话 rail，不因页面切换主动刷新；直接打开 `/settings` 或当前没有已注册会话 rail 时，才挂载 Chat session 数据源并请求 `/api/chat/sessions`，以服务端返回的真实会话或真实空列表渲染；不得把本地 fallback 的单条 `New` 占位当作 Settings 侧栏会话列表。列表项主体只展示标题，真实会话尾侧固定提供单个三点更多按钮；展开菜单承载置顶、查看详情与删除操作，查看详情会聚焦该会话并打开 `Details` 面板且不主动收起已打开的移动会话抽屉，删除操作必须二次确认后才进入会话删除流程。处理中会话在标题旁显示 loading，其他状态不显示状态灯、时间、会话标识、Skill 标签或额外摘要。运行页空列表、Chat 本地空白草稿和 Chat 本地空白草稿优先展示一条 active `New` 占位会话；`New` 草稿/占位只作为输入入口，不显示三点菜单，不支持置顶、详情或删除，同一路径内重复点击 `New` 只聚焦既有空白虚拟会话，不创建多个空会话。`/chat` 首次发送时创建真实 Chat session；短 canonical id 继续用于接口、持久化、URL 与工作区隔离，不直接作为列表展示值。
- `Chat` 的会话列表项与 workspace header 状态按钮共享同一会话状态语义：前端按当前 assistant 消息与任务态派生 `ready / busy / failed / interrupted`；其中 `streaming / queued / running / in_progress`、空 assistant 占位与挂起任务映射为 `busy`，错误、失败、取消与显式 `message.error` 映射为 `failed`，请求已被接受但恢复失败时映射为 `interrupted`，其余稳定态映射为 `ready`。列表项只在 `busy` 时于标题旁显示 loading，`ready / failed / interrupted` 不显示行内状态灯；workspace header 的状态按钮可见层只显示信号，状态名称只保留给读屏与悬浮语义。

## 消息结果与恢复

### Chat 消息语义

- `message` 是前端时间线展示单元：一条 user message 对应用户输入、附件和发送时间，一条 assistant message 对应同一 turn 的最终正文与过程披露入口。message 由服务端 turn 派生，本地 queued message 只作为发送后的临时乐观状态，服务端 turn 回来后按 message id/turn id 压缩替换。
- `turn` 是一次用户输入到一次运行时结果的服务端事实单元，包含 `prompt`、输入附件、运行状态、`final_output`、开始/结束时间和 `runtime_trace_events`。同一会话内 turn 严格追加，前端渲染时固定 user message 在前、assistant/Thinking message 在后。
- `event` 是会话、turn 或运行过程的增量事实。增量轮询里的 event 描述 session/turn 状态变化；`runtime_trace_events` 描述单个 turn 内的过程步骤，用于 `Thinking / 已思考` 披露。事件详情可通过 `/turns/{turn_id}/events/{event_id}` 懒加载，避免把大段 tool output、thinking 或日志放入轮询响应。
- Chat 发送后由 Chat session 状态进入 `busy`；执行完成或失败后，用 input 返回结果或 Chat session 详情恢复当前消息区。
- Chat 的过程展示统一消费 `RuntimeTraceEvent`。turn 摘要与 updates 中的轻量事件只保留 `id / kind / status / text / detail_available / created_at / completed_at / duration_ms`；`created_at / completed_at / duration_ms` 均为毫秒数值，未完成事件的 `completed_at` 为 `null`。轻量事件不得包含 `seq / session_id / turn_id / provider / source / role / lifecycle / title / summary / visibility / raw / action / blocks`；完整 `blocks` 只通过 `/turns/{turn_id}/events/{event_id}` 详情接口返回。`kind` 枚举固定为 `important_text / reasoning / plan / tools / commands / system`，前端只按 `RuntimeTraceEvent.kind` 过滤展示类型，不通过标题、正文、关键词或语言模式推断事件类型。
- Chat 显式选择 `Codex` 且消息包含图片附件时，服务端需把已上传并落盘的原图路径传给 Codex CLI `-i` 参数；前端提示词不需要再描述“图片已存在”才能触发图片读取。

### 执行不中断

- 已进入 运行时执行链的请求不因浏览器断连、页面切换、标签页隐藏、请求写失败或前端取消而中断后端执行。
- 当前 HTTP 请求只负责回传本次结果；最终结果仍需落到会话历史。

### 实时更新与断线恢复

- 会话更新采用“HTTP 快照 + owner 增量轮询 + bounded detail 兜底”模型。页面首次进入、刷新和切会话时先读取会话列表，并用浏览器缓存立即恢复当前会话首屏；集合接口返回空列表或短列表时只能作为 summary patch，不能清空本地可见会话、active session 或已加载详情。随后对当前 active server session 读取一次最新单会话详情快照。普通前台恢复仅处理本地 `local_running / recovering`、缓存不完整或可恢复占位；稳定 `ready` 会话不自动读取会话列表、详情或 updates。当前 route owner 的 `latest_update_id` 只在本地存在同步意图或显式恢复时续接 `/api/chat/sessions/updates`，并按 `update_id / updated_at` 把增量 patch 到本地缓存。
- 服务端在接受输入后必须立即持久化 `user` turn、`busy` 状态和首个增量事件，再启动 CLI Runtime。增量轮询晚于输入提交、页面刷新或短暂断开时，客户端通过详情快照和 `after_update_id` 续接缺口，不重新提交同一条用户消息。
- 前端发送输入时先把本轮 user 消息、附件引用、busy 状态、活动会话和最近会话列表同步写入当前 route 的内存缓存、长期会话快照和轻量信息快照，再提交可见状态渲染。浏览器刷新、路由切换或移动端 WebView 恢复时，首屏可直接使用这些快照恢复已发送消息，不等待 input 请求完成。
- 用户提交输入后，当前活动会话的消息区立即追加 user 消息并回到底部，workspace header、移动端标题按钮和左侧会话列表项同步进入 `busy`。若随后集合接口或详情接口返回较旧的 `ready` 摘要，但没有带来完整 assistant 终态、失败态或更新后的 `updated_at`，前端不得用该旧摘要覆盖本地 `local_running / recovering` 状态。
- 左侧主导航会话列表跟随当前 route owner 的增量轮询。用户停留在某个会话时，其他运行中会话的标题、置顶、loading 和终态仍通过同一 owner updates 响应更新；稳定完成且本地缓存完整的会话不因轮询保持周期详情刷新。
- 轮询状态只作为轻量诊断态存在。请求失败、退避或页面隐藏时，不在消息区追加错误消息，不弹出失败 toast，不改变会话列表排序；可在 Details 或无障碍提示中暴露“正在同步”语义，但不得遮挡输入、导航和消息阅读。
- 网络失败只影响下一次增量读取，不改变 session、turn 或 assistant 消息的业务状态。前端不得因为轮询请求失败、网络切换、浏览器刷新、bfcache 恢复或标签页隐藏而把当前会话标记为 `failed`。只有服务端详情或增量事件明确返回 `turn.failed / session.failed / interrupted`，或单会话详情确认该会话不存在，才允许进入失败或中断态。
- 前端维护每个 owner 的 `latest_update_id` 与每个 session 的 `updated_at`。收到旧 `updated_at`、重复 `update_id` 或已应用 turn/message id 时必须幂等丢弃；由于不可见事件过滤和预算截断会让可见 `update_id` 不连续，前端只用 `latest_update_id` 续接，不按连续数字判断丢包。发现 `resync_required=true`、本地 `updated_at` 低于服务端摘要或本地缓存不完整时，立即补拉对应单会话详情，并按既有 turns 分页合并规则恢复。若服务重启后无法续接旧 `after_update_id`，服务端必须返回 owner 级 `resync_required`；该响应即使不携带 `session_id`，前端也必须补拉当前 owner 下仍处于 `local_running / recovering` 的 runtime-backed 会话详情。
- 服务端会话列表、详情、turns、entries 与输入入口必须在返回业务状态或执行 busy 检查前校准孤儿运行态：当会话仍标记为 `busy` 或存在 live turn，但当前进程没有 `turnRunning / turnCancel` 对应的 live worker 时，服务端将该 turn 与 session 收敛为 `interrupted`、追加 `Interrupted` 过程事件和系统 entry，并立即写回 session store。该校准只作用于无 live worker 的会话，不影响当前进程内仍在运行的请求。
- 更早历史分页不属于实时刷新链路。稳定会话存在 `turns_paging.has_more_before=true` 时，前端不得自动后台请求 `turn_before` 并把结果实时合入当前可见时间线；首次进入、刷新和 page-activation 的单会话详情校准也只读取最新页。只有用户点击 `Load earlier messages / 加载更早消息` 或滚动到顶部触发历史加载，才允许请求对应分页并在保持阅读锚点后展开。本地已确认 `has_more_before=false` 后，最新 1 个 turn 的 bounded detail 或 owner updates 不得因服务端页自身存在更早数据而重新打开本地更早历史入口。
- 增量轮询按当前 owner 合并所有进行中或可恢复会话，避免为每个会话分别拉取详情。前端只对收到增量的会话做局部 patch；不会按固定间隔重新拉取每个 busy 会话的完整最新 turns 页。updates 请求不再上报 step `seq` 区间；服务端按 `after_update_id` 续接并只返回当前可见的轻量 runtime event。
- 页面可见且存在 `local_running / recovering` 会话时，轮询采用按无进展次数退避的 owner updates 通道：发送后或最新用户消息尚未被 assistant / 任务消息 / 失败态回填时约 `2s` 起步；连续无进展后退避到约 `3s / 5s / 8s`。会话进入 `ready / failed / interrupted / exited / deleted` 等终态后停止自动 updates、event detail、会话列表和当前详情自动拉取；跨设备产生的标题、置顶、删除或终态变化通过用户手动刷新、切换会话或下一次真实运行态恢复。
- 传输预算按语义事件控制：`session.updated` 只传会话摘要字段，不传 turns；`turn.event.appended / updated` 只传单个新增或更新的 `runtime_event` 轻量摘要与 `detail_available` 标记；`turn.completed / failed / interrupted` 只传当前 turn 收口状态和可用最终正文。完整 event detail、完整历史页、附件原图、结构化 `blocks` 和超大 Markdown 正文继续通过详情接口读取。

### 请求恢复

- `Chat` 在同一 Session 内保持追加式会话历史；每轮请求都要追加新的用户消息与新的助手消息占位，不得把后续回复继续回写到已完成的历史消息。
- `Chat` 是唯一 Web 对话运行态，页面初始化、发送消息、附件草稿、服务端回源和刷新恢复都围绕 Chat 会话模型进行；旧 `chat` 会话只在加载阶段迁移进 Chat，不再保留多业务编排、多 session 的独立运行态。
- `Chat` 的长期历史按北京时间 05:00 作为归档日边界分文件存储；05:00 之前的消息归入前一归档日，05:00 及之后的消息归入当天归档日。该分文件规则只改变运行时线程归档，不改变当前 Chat 会话的短 canonical `session_id`。
- `Chat` 直连 Codex 的 thread id 与同一归档日绑定，写入当前 Chat 工作区 `.alter0/codex-runtime/threads/<YYYY-MM-DD>.json`；归档日切换后，新文件不存在即代表新的 Codex 会话环境，运行时不得继续 resume 前一归档日的 Codex thread。
- 单条 assistant 消息只能由当前请求结果或会话详情恢复补丁；补丁目标只能是当前活跃的未完成消息，消息进入稳定结果后，迟到结果不得重新打开或覆盖最终正文。
- 运行页初始化时，服务端会话详情回填不得覆盖当前浏览器里已经新追加、但服务端详情请求发起时尚未落库的本地消息；本地新消息与当前请求占位优先级高于陈旧详情响应。
- 若运行页刷新后服务端会话集合接口暂时未包含当前活动 `session_id`，前端仍需保留本地恢复出的该条会话，并主动尝试按 `GET /api/chat/sessions/{session_id}` 补拉详情；在单会话详情也确认不存在之前，不得立刻创建新的空白会话顶替当前活动会话。
- 若运行页刷新后服务端会话集合接口暂时未包含某条最近会话，即使该会话当前并非活动会话，前端也不得立刻把它从 `Sessions` 列表移除；左侧最近会话列表以本地快照和服务端结果合并视图为准，只有在用户显式删除或后续回源明确确认不存在时才允许消失。
- Chat 的状态更新链路需直接复用 Chat session：输入接收后立即标记当前会话为 `busy`，同步完成后标记为 `ready`，请求失败时标记为 `failed`；Skills 变更由前端在下一次 input payload 中提交，并支持空数组代表显式清空选择。历史会话恢复出的 Skill 选择需在前端按当前启用的统一 Skill 目录实时过滤，已删除或禁用项不参与已选计数、勾选态或下一次 input payload；用户在历史会话中新增勾选或取消 Skill 后，无需刷新即可作用于后续发送。当 turn history 仍未对当前请求可见时，集合接口与单会话详情也必须先返回 Chat session 的最近视图，而不是直接返回缺失或 404。
- `travel` 会话若在本轮执行中先收到了正文攻略，但 HTML 页与 `travel` 子域名尚未就绪，服务端需继续在同一轮完成自动页面固化与发布收口；前端只在自动收口结束后接收最终成功态或明确阻塞态，不把“纯文本成功但无页面”的中间态当作完成结果。
- 浏览器本地缓存中残留的 `streaming`、`Thinking...` 或 `Load failed` 临时消息在页面恢复时必须先按 session id 回源详情或读取 owner 增量；只有服务端确认没有对应已接受 turn、没有可恢复运行状态且没有后续事件时，才允许转为明确失败态。
- 若请求断开且没有可用正文，前端先显示可恢复状态并尝试快照回源与增量轮询；失败文案只能在服务端确认不可恢复后出现，并需要明确说明可刷新页面恢复最新已保存回复。
- 若请求断开但服务端已接受本轮消息，前端需先拉取当前会话详情，用服务端已持久化的 `assistant` 消息覆盖本地 `Thinking...`、`Load failed` 或其他临时失败态；该恢复链路不得通过重新提交同一条用户消息来实现。
- 页面刷新后，若当前活动会话存在本地失败态流式消息，前端需优先拉取服务端会话消息，并用已持久化结果覆盖本地失败态；即使服务端集合接口已经返回了该会话的摘要项，只要未附带完整消息，也必须继续补拉单会话详情。
- 页面刷新、前后台恢复或集合接口刷新后，若服务端集合返回的当前会话消息数量短于浏览器侧已追加历史，且本地仍存在本轮用户消息、`Thinking...`、`streaming` assistant 或其他可恢复状态，前端必须继续保留本地时间线并等待单会话详情或后续集合结果确认，不得把刚发送的消息从 UI 中移除。
- 若当前活动会话已经从服务端恢复出最新 `user` 消息，但该 user 之后尚无 assistant、任务或失败消息，运行页继续按待恢复会话处理并重试详情接口；只有拿到稳定 assistant 或明确失败态后，才停止本轮恢复。
- `Chat` 的输入锁只由真实运行中状态触发：会话或消息处于 `busy / running / queued / in_progress / local_running / recovering` 时禁止重复提交；`failed / interrupted / exited` 以及无 assistant 输出的失败 turn 不得继续禁用 Composer。自动恢复轮询只覆盖本机产生的 `local_running / recovering` 同步意图；普通服务端 `busy / running` 摘要不触发后台 updates/detail，用户可通过页面激活补偿或手动刷新恢复跨设备变化，终态会话可直接发送下一条输入并复用原会话恢复运行态。
- 运行页恢复依赖会话详情快照补拉与 owner 增量轮询，不再通过后台 Task API 轮询任务状态。

### 渲染策略

- Chat 消息区使用逐条 patch 与浏览器逐帧合并刷新；Chat input 结果或中断恢复到达时立即收口最终状态。
- 运行态消息合并以 session id、turn id 和 message id 为稳定键；本地 queued user 消息在服务端返回对应 turn 后被服务端 user/assistant 消息替换或补齐，不产生重复用户消息。同一 turn 内必须固定按 user 消息在前、assistant/Thinking 消息在后的顺序渲染；当增量 patch、input 响应、本地 queued user 或详情回源的时间戳不一致时，不能只按 `created_at / at` 排序导致 Thinking 出现在对应用户消息之前。无重叠但带稳定 assistant 结果的新 turn 应追加到现有历史后再按稳定 turn/message 顺序排序，不得因为响应页较短而丢弃已加载历史或新结果。分页状态合并以本地已加载消息集为准：最新 bounded 页可以补充新 turn，但不能把已经耗尽的本地历史重新标记为可继续加载。
- Process 展开收起与运行状态回填不得导致整段消息列表重建，也不得在长输出期间持续占满主线程导致导航、发送、详情和会话切换按钮失去响应。
- 时间线装配需按单条消息缓存稳定渲染结果；当仅当前 assistant 占位或结果变化时，未变化的历史消息不得重新生成 Markdown HTML、Process step 树或 runtime timeline item。
- 仅有 Composer 草稿变化时，Conversation 时间线、Markdown 正文与 `Process` 展示不得重新解析或整段重建；性能热点需收敛在输入区本身。
- 前端点击诊断仅在 `?debug_clicks=1`、`?debugClicks=true` 或 `localStorage["alter0.debug.clicks"]="on"` 时启用。诊断记录需覆盖 `pointerdown / pointerup / touchstart / touchend / click` 的事件目标、坐标命中的顶层元素、当前 `activeElement`、壳层 class、可拦截点击的遮罩层、`defaultPrevented` 与按钮禁用状态，并额外输出主线程 long task，用于排查真机首点无效和双击才响应问题。
- `Chat` 的移动端运行页头部按钮需直接响应首个 `touchstart` 或非鼠标 `pointerdown`：`Menu`、中间标题详情入口、`New` 会话入口以及保留的移动端会话入口都不得依赖后续合成 `click` 才触发；同一触摸产生的后续 `click` 需被去重，避免一次手势执行两次。移动端边缘操作以无边框图标按钮作为可见形态，`Menu / New` 文案保留为可访问标签。
- Chat 长输出复制按钮不得把完整输出写入 `data-*` 属性或其他 DOM 元数据；复制 payload 保留在组件闭包并通过剪贴板 API / document copy 兜底写出，避免长日志在 DOM 中重复放大并拖慢轮询、选择和点击。
- Chat 时间线渲染需按 `turns / 展开态 / step 详情 / 语言` 等稳定输入缓存，Composer 草稿、滚动活跃态、配置面板开关或复制状态变化不得触发整段输出重新解析 Markdown。
- Skill 消息中的 `Process` 使用 Chat turn `runtime_trace_events` 渲染，不再保留文本解析兼容。
- 结构化 `RuntimeTraceEvent` 需要在 Chat input 结果与会话历史恢复后保持一致，刷新页面不得把已完成消息重新退化为仅正文展示。
- Process 披露过滤按 `RuntimeTraceEvent.kind` 执行，枚举固定为 `important_text / reasoning / plan / tools / commands / system`。Codex `commentary` 映射为 `important_text`，tool/MCP/skill/hook/approval 映射为 `tools`，shell command 映射为 `commands`，runtime/system/unknown/error 映射为 `system`。过滤器只隐藏或显示折叠区事件，不改变最终 assistant 正文。
- `Chat` 的消息输出结构统一收敛到轻量 IM 式消息流：用户输入右对齐并使用浅灰低对比紧凑气泡，气泡高度需由较小纵向 padding 与独立消息行高控制，助手回复左对齐并弱化为无边框正文阅读流；Chat 消息阅读区使用白底无框正文面，视觉层级由阅读宽度、留白和角色对齐承担，不在对话区叠加明显边框、背景分界或卡片容器。Skill 与 Chat 中间步骤默认按 `Thinking / 已思考` 轻量披露行展示，展开后在当前消息内进入步骤详情，移动端也保持同页内联展开；同一条助手消息同时包含最终正文和过程事件时，`Thinking / 已思考` 披露入口先于最终 Markdown 渲染，最终正文和复制动作位于其下方。Chat 助手最终答复统一使用稳定的运行页 markdown shell，正文先于复制工具栏渲染，复制动作位于正文下方，代码块独立呈现为浅灰内容块；消息正文区不显示逐条时间，仅在进行中、排队、失败等非稳定状态下保留状态标签。新增运行页若呈现用户输入与助手输出，必须复用 `RuntimeTimeline` 与 `runtime-message / runtime-message-user / runtime-message-assistant / runtime-message-bubble` 契约，避免继续产生页面私有气泡格式。
- `Chat` 在显式访问 `/chat?markdown_demo=1` 时可临时覆盖当前时间线视图并注入一条非持久化 assistant Markdown 演示消息，用于预览环境验收 ATX/Setext 标题、段落换行、强调、删除线、自动链接、图片、引用、嵌套列表、任务项、列表内引用与代码块、分割线、代码块、对齐表格与 raw HTML 转义等当前支持语法；表格样例覆盖短字符、长中文、长 URL/代码和混合内容场景；折叠示例中的 HTML 标签按代码块展示，折叠内容本身按普通 Markdown 展示；普通 `/chat` 不显示该样例，也不把该消息写入 Session history。
- 长会话默认只渲染最新一批消息；当本地隐藏消息存在，或规范分页状态确认远端仍有更早历史时，消息区展示 `Load earlier messages / 加载更早消息` 入口，并在滚到顶部时自动按批次扩展更早消息。扩展历史时需保持当前阅读位置，不得强制跳回底部；当本地和远端都已无更早历史时，顶部加载入口必须消失。
- `Process` 步骤标题与正文在桌面和移动端都必须保持整列阅读宽度；步骤序号、展开图标、标题与状态信息需在同一行垂直居中；长中文说明、路径、命令片段与 Markdown 文本优先在当前消息容器内自然换行，不得在真机窄屏下塌缩成逐字竖排窄列。
- Conversation 展示层必须在渲染结构化过程事件与最终 markdown 前移除零宽断行字符，并对“每字一行”的病态段落做可读性归一化；该修正同时适用于消息结果和历史会话恢复。
- Chat 的最终 Markdown 输出不得复用需要额外 CSS 强制补丁的旧 shell 结构；其正文 DOM 必须保持普通静态文本语义，不绑定 `touchstart / pointerdown` 选区脚本，不设置 `contenteditable / inputmode / tabindex`，不创建浮动复制层或假选中 class。复制按钮只读取组件闭包中的原始文本，不把长 payload 镜像到 DOM 属性。
- `Chat` 的消息时间线在内容较少时必须保持顶部收口；短用户消息、折叠后的 `Thinking / 已思考` 披露行、最终回复与对应状态标签继续贴近各自消息气泡排布，不得因为时间线容器满高拉伸而出现大块垂直空白。
- `Chat` 打开已有消息的会话、刷新恢复当前会话或切换到其他会话后，时间线初始视口必须落到最新消息所在底部；当前活动会话内发送新消息后，时间线必须随新增消息回到底部，使本轮用户消息与助手占位立即可见；若用户已经在同一会话内手动滚动阅读历史，后续结果 patch、Process 展开状态变化和草稿输入不得强制把视口拉回底部。
- 助手最终回复提供一键复制；若消息含 Process，复制内容只包含最终正文。
- Web `Chat` 的 Deliverables 与 Session Profile 详情面板已移除；专项交付契约继续作为底层 Skill/Skill 执行上下文的一部分维护，不再作为独立对话运行页配置。
- 前端所有绝对时间与时分标签统一按北京时间（`Asia/Shanghai`）渲染，并固定采用 24 小时制；浏览器本地时区不参与显示格式决策，控制台管理页中的额度重置、运行时间等管理时间戳也必须复用同一口径。接口返回的业务时间字段统一为毫秒时间戳，展示层负责格式化。
- Cron 表单中的默认时区固定为 `Asia/Shanghai`，不再读取浏览器 `resolvedOptions().timeZone` 作为初始值。

## 并发与分流

### 会话级顺序

- 同一会话内同步请求保持顺序一致。
- 上一条同步执行未结束时，后续用户消息排队等待，不因短等待窗口直接失败。
- ReAct 多步执行中收到同会话补充输入时，后续迭代可吸收当前最新用户消息继续推进。

### 全局限流

- 系统提供全局并发上限、排队与超时降级能力。
- 高复杂度请求可切换为后台 Task，由 Task 领域承接执行、日志和产物交付。

## 阅读体验

### Markdown 与安全

- 聊天气泡支持 ATX/Setext 标题、强调、删除线、自动 URL/email 链接、列表、列表内引用与代码块、引用、链接、行内代码、对齐表格与代码块；Markdown 排版按正文阅读节奏呈现，标题保持紧凑层级，段落和列表以自然行高与稳定缩进组织，嵌套列表按 Markdown 缩进保留真实层级，普通链接显示外链箭头，代码块和引用只保留弱边界；表格采用无外框卡片、无表头灰底的横线分隔样式，短表格至少铺满消息宽度，普通长文本在单元格内自动换行，链接、URL 与代码保持不硬断开，不把每段输出包装成厚重卡片。
- 助手消息中的 markdown 图片按消息媒体统一以内联图片显示，使用浏览器懒加载策略，并保持链接可直接打开原图。
- React 托管页面的正文型内容统一复用运行页 Markdown 渲染器：Memory 长期/天级/强制/说明文档、Task 请求与结果、任务日志和产物摘要、Control 描述、Cron 输入、Skill 说明、Codex 运行时说明以及 Chat `Session Profile` 中的非等宽字段，都按 Markdown 正文渲染。ID、路径、密钥、配置值、时间戳和分支名等元数据保持纯文本或等宽展示，避免把机器标识误解析成富文本。
- Chat final output 统一使用 `MessageMarkdownShell` 承载最终答复，解析规则、复制按钮、选择行为和 DOM 稳定性都由同一组件负责；相同 markdown 不得因父级无关重渲染反复写入 `innerHTML`，也不得依赖 Chat 视图级 `user-select !important` 兜底。
- 原始 HTML 不直接透传。
- 长路径、超长单词、代码块和 diff 只允许在内容块内部横向滚动，不撑破外层消息容器。

### Process

- Process action / observation 与 Chat 执行细节在前端收敛为可折叠 Process，并统一在当前消息或 turn 内同页展开。
- 最终答复出现后，Process 默认折叠，阅读焦点回到正文。
- 单个步骤详情由用户点开对应步骤后展示；若步骤标记 `detail_available=true`，前端需先按 `session_id / turn_id / event_id` 拉取完整 detail 并写回当前消息缓存，再在当前浏览器会话内保留展开状态。已写入当前消息缓存的 detail blocks 优先级高于后续轻量会话摘要、bounded detail 或 owner 增量 patch；后续刷新只能更新步骤 `status / text / completed_at / duration_ms` 等摘要字段，不得清空已加载 detail 或把 `detail_available` 回退为 `false`。外层 `Thinking / 已思考` 每次展开或折叠时需收起该消息下已打开的单步详情，使移动端先稳定进入步骤列表态，不把历史详情重新撑开视口。
- Chat 过程披露中的所有步骤详情都直接渲染为同一套最终 detail surface：`chat / code / diff / tool_input` 与 JSON 类 `tool_output` 使用等宽内容块，`text / markdown / thinking / tool_output(text) / error` 以及历史 `step.detail` 使用富文本正文块；结构化 block 的标题、文件名和起始行号需在详情头部保留。轻量 `RuntimeTraceEvent` 不携带 `blocks`，有详情时必须等 detail 接口返回后再展示步骤体，不得先按普通 Markdown 文本显示再切换为最终形态。步骤行的类型标签、耗时与状态需与 Chat 同源渲染，类型标签需与过程披露过滤映射同源，不通过标题或自然语言内容推断，详情块不重复渲染状态 badge。

### 布局

- 全站默认固定侧边栏；仅侧栏自身内容溢出时允许侧栏内部滚动。
- Chat 历史区支持折叠与展开，减少长对话阅读空间占用。
- Conversation workspace 的新会话入口在 `chat` 路由下切换为 Chat 会话语义，并随语言切换同步更新。
- Session 历史区的空态提示与列表可访问标签需按当前路由与语言即时切换文案；这些文案更新不得清空或重建 runtime 已注入的会话卡片节点。
- 左侧主导航中的会话列表需先渲染独立 `Pinned / 置顶` 分组，再把非置顶会话按最近时间分组为 `Today / Yesterday / Earlier`（中文对应 `今天 / 昨天 / 更早`），并与主导航 `menu` 复用同一套分组容器、hover、激活态视觉和桌面会话列宽；分组内条目保持主导航式紧凑信息结构，采用低噪音列表项关系：主体只保留标题并在可用宽度内单行截断，长标题不得撑开导航会话区、分组容器、列表容器或列表项自身宽度；新增会话插入、列表刚好填满或跨过滚动阈值时，不得触发浏览器滚动锚点补偿、滚动槽宽度重算、头部高度重算或列表区重新分配，并且不得让 `Sessions / New` 区块在不同运行页之间发生位置跳变，真实会话尾侧只保留 30px 级三点更多按钮，展开菜单承载置顶、查看详情与删除操作；删除需二次确认。草稿/占位 `New` 不渲染更多按钮。不再额外挂出独立 footer、胶囊操作面、完整会话 id、时间、Skill 标签或摘要字符串。
- Session 历史区的会话条目不展示 ready、failed、exited 或 interrupted 状态灯；只有处理中条目显示 loading，并为读屏输出当前忙碌状态文案。
- Conversation workspace 头部的标题、状态按钮、会话详情入口和新会话入口需按当前路由与语言即时切换文案；状态按钮同时反映当前活动会话派生状态，但可见层只显示信号，不再展示固定 `Ready` 或其他状态文案；该信号固定排在当前会话标题左侧，会话详情入口并入当前标题按钮，不再额外渲染独立右侧详情按钮；这些壳层文案更新不得覆盖当前会话标题或消息内容。
- `Chat` 的会话列表、工作区外壳、聊天滚动区和输入区需输出 `runtime-*` 主契约并保留必要的 `chat-* + conversation-*` 兼容 class，确保 `/chat` 共用同一工作台表面与细节皮肤，同时保留 `data-conversation-*` 钩子供样式和测试使用。
- `Chat` 首页 Composer 采用单一胶囊式助手输入面板：主 textarea 透明无内边框，工具栏与输入区处在同一白色 surface 内；工具栏显示 `Session`、独立 GitHub 仓库入口、附件与发送动作。仓库入口使用 GitHub 图标，附件入口使用回形针图标，二者不得复用按钮或交互；文字 label 仅保留给可访问语义。桌面端输入面板按主阅读宽度居中，移动端压缩输入高度、外层留白与提交按钮体量，同时维持足够横向留白，避免输入区压窄；PC 端上传、发送、状态、详情、流程入口与弹窗动作保持平面化，除 Composer 胶囊外不使用额外胶囊按钮、卡片边框或厚圆角表达层级；会话列表项与 `Details` 面板保持同一浅色 runtime 质感。空态工作区需使用低对比网格与细弧线背景，并锁定为不可滚动表面，不允许通过空白区域拖拽把头部和输入区顶出可视区。
- `Chat` 在页面重新变为前台可见或浏览器重新把当前页激活时，必须复用运行页共享的 page-activation 补偿刷新链路：刷新会话列表、按 owner `latest_update_id` 读取增量，并在 `resync_required` 或缓存不完整时补拉当前活动会话详情。页面隐藏时暂停高频轮询；恢复前台后立即做一次增量检查，避免后台标签页持续发起会话详情请求。
- `Chat` 在 bfcache 恢复或网络恢复在线时复用 page-activation 入口，但只有本地仍处于 `local_running / recovering`、缓存不完整或存在可恢复占位时才发起补偿请求；Chat owner 的 session 详情默认按最新 `20` 个 turns 与约 `1MiB` 前端 API turn DTO 页预算分页返回，前端需用 `turns_paging.has_more_before` 识别分段结果。页面恢复、手动刷新、轮询或输入返回的轻量详情不得丢失本地已加载的更早消息，也不得在恢复阶段自动请求 `turn_before`、扩展当前可见窗口、强制滚动到底部或重建 Composer 输入状态与配置面板。
- `Chat` 时间线到顶交互先展开本地已加载的隐藏消息批次；本地窗口已完全展开且服务端仍有更早历史时，才由 `ConversationRuntimeProvider.loadEarlierHistory()` 按 `turn_before` 显式请求下一页。分页结果按消息 id 与时间顺序合并进时间线，并在保持阅读锚点后展开下一批。
- `Chat` 发送新消息后，服务端输入响应、后续详情刷新或分页片段只允许按 turn/message id 与时间顺序合并进现有时间线；即使响应只包含新 turn 或最新轻量页，也不得替换掉用户当前已加载的旧历史。若追加前当前渲染窗口已经覆盖全部已加载消息，追加后可见窗口需同步扩容，避免旧消息被最新一轮挤出视图。
- `Chat` 的浏览器缓存分为短期运行态、完整消息快照与轻量会话信息快照：24 小时运行态缓存按 route 保留当前已加载会话的完整消息或 turns；24 小时 `localStorage` 完整快照使用 `chat` 独立 key 保存当前 route 会话、完整消息、分页边界、`updated_at` 与本地详情新鲜度，用于刷新、重开或 `sessionStorage` 丢失时首屏恢复；轻量会话信息快照只保存标题、状态、置顶、模型与能力选择、`updated_at` 等元数据，用于完整消息缓存写入失败或被清理时恢复会话列表。active session、文本草稿、附件草稿与过程披露过滤同样按 route 使用独立 key。旧 `active snapshot / recent snapshot` sessionStorage key 只在启动时清理，不读取、不迁移。缓存不得阻断首次进入和刷新时的服务端会话列表与当前 active 会话最新 bounded detail 回源；当服务端返回更新历史时继续按现有分页合并规则覆盖或补齐本地快照，并刷新缓存时间。page-activation 对稳定 `ready` 会话复用本地缓存，不强制读取 summary 或详情。
- `Chat` Composer 支持最多 5 张图片附件；附件可通过附件按钮选择，也可在 PC 输入框内直接粘贴剪贴板图片。粘贴图片时仅拦截图片文件并进入附件草稿，普通文本粘贴继续保持 textarea 原生行为。附件在输入区以缩略图展示，可单张预览和移除，并按会话草稿持久化。缩略条继续使用预览图，但单张预览弹层必须优先显示原图。当前选中的模型若未声明视觉能力，带图发送必须直接阻止并提示切换模型。
- GitHub 仓库选择与附件草稿分别持久化。用户在真正发送消息前搜索并勾选一个仓库时，只形成可移除的本地草稿 chip，不创建会话、不访问仓库也不拉取代码。首次消息发送时仓库引用与用户语义一起提交并绑定到该会话；一个会话最多绑定一个仓库，绑定后 chip 不可移除或替换。服务端在 Agent 启动前把仓库准备到会话工作区固定相对目录 `repo/`，只做首次 clone，不自动 pull/fetch/reset，也不自动 commit/push；失败状态允许原地重试。
- 移动端 `Chat` 的左侧主导航抽屉与主工作区在 `1280px` 及以下需回落为静态表面，不保留模糊玻璃层或持续背景动效；性能优先级高于装饰层，确保真机滚动、抽屉开关和输入框聚焦不出现明显卡顿。
- 根工作台仅在窄屏时使用主导航抽屉；Chat 会话列表由主导航统一承载，避免出现导航抽屉和会话浮层叠加。
- 路由页头部的标题与副标题需按当前路由与语言即时切换文案；这些页头更新不得覆盖 route body 内已渲染的页面主体内容。Settings 路由页需复用 Chat 的主面板 frame 与紧凑工作台标题栏视觉节奏，标题栏只输出同规格标题标记和单行标题，不再使用与Chat 割裂的大号页面标题块、裸露页面标题区或标题副文案；Settings 移动端抽屉入口与运行页 `Menu` 使用一致的无边框图标按钮视觉，并保留可访问文本标签，窄屏下标题直接并入同一行 `Menu + Settings` 顶栏，不再叠加第二行标题；Settings 正文需作为 frame 内部滚动区，长内容不得被外层 frame 裁切。
- 已由 React 接管的工作台需在 DOM 上暴露稳定路由钩子：根壳层输出 `app-shell[data-workbench-route]`，运行页和控制页继续输出 `data-route / data-conversation-*` 标记；兼容层只能依据这些由 React 输出的钩子退让，不得继续维护独立白名单。
- 欢迎区与 Composer 面板在同一主工作区内采用主仓库式上下结构：欢迎区直接输出 `Alter0 workspace` tag、面向 repo / task / runtime 的默认标题与说明、target picker 与快捷提示，Composer 独立贴底；欢迎区内容超出可视高度时，输入区仍需稳定贴底，不得与欢迎区、消息区发生叠层覆盖。
- 用户消息右对齐并使用浅灰低对比紧凑气泡，`Chat` 统一采用克制的冷灰工作台阅读主题；助手回复弱化厚重卡片层级，默认呈现为无边框正文阅读流，Chat 正文工作区不显示明显外框或分隔背景；复制操作贴在正文下方，思考过程只保留一行内联可点披露入口，只展示步骤数量，不展示耗时，Process 详情和代码块只保留必要边界与有限强调色；Markdown 表格在消息正文内以真实表格结构呈现，采用横向分割线而不是卡片外框或表头色块，短表格不强制固定最小像素宽度，普通长文本在单元格内自动换行，窄屏下只有不可断内容超宽时才在表格块内部横向滚动。
- 交互手感在 `Chat / Settings` 间共享同一套 motion 与反馈基线：按钮、导航项、Composer 工具、列表项、快捷提示、确认弹窗和详情面板统一使用 ease-out expo 曲线；按压反馈应在 120-150ms 内完成轻量缩放，悬停反馈在 160-260ms 内完成边框、背景、阴影与微位移变化；弹层进入使用淡入、轻缩放与短位移，退出或系统减少动效时需快速收敛。数字、会话 id、时间和指标字段必须使用等宽数字；键盘可达控件必须使用 `focus-visible` 焦点环；内部滚动容器需声明滚动穿透隔离，横向或候选列表滚动条只在必要时可见。
- `Chat` 助手消息尾部默认不显示时间；仅当回复仍在生成、排队或失败时展示紧凑状态标签，不再为已完成消息重复展示 route/source/status 元信息。
- 桌面宽屏下 Chat 消息列与 Composer 按主工作区宽度自适应放宽，并保持统一居中；正文区统一保留 `960px` 最大阅读宽度，但外层工作台也必须同步收缩导航与间距，避免在中等桌面宽度下出现阅读区限宽而整体布局仍然拥挤、遮挡或越界。
- Web Shell 主导航需根据 URL hash 即时同步当前路由高亮；语言切换与抽屉开关更新不得导致会话卡片、消息节点或 route 内容被清空重建。
- React 壳层发出的主导航跳转、新建会话、欢迎区快捷提示、语言切换、导航抽屉开关与会话历史折叠同步事件，必须由当前前端运行时在同一页面内完成确认、路由更新、快捷发送或会话创建，且不能要求用户重复点击或依赖额外脚本注入的全局函数。
- `Chat` 提供统一的右侧箭头四键阅读定位条 `回到顶部 / 上一条 / 下一条 / 回到底部`：滚动超过阈值后显示顶部与底部入口，上一条与下一条按钮按当前可见消息块或 Chat turn 实时重算目标；内容折叠、展开或重排后，按钮显隐与目标需同步更新。连续点击 `上一条` 时，若当前最上方可见块已经被上一轮跳转对齐到顶部偏移，下一轮必须继续跳到它前一块，不得反复指向同一块。Chat 中 `上一条 / 下一条` 都以用户消息为跳转目标，assistant 的 Thinking / Process 块不作为中间目标。`回到底部` 只在最后一条内容的底边仍位于视口外时显示；若最后只剩空白或底部 padding，不得继续显示伪底部跳转。移动端四键定位条固定停靠在工作区右侧、输入区上沿之上，四个按钮统一为独立圆形触达面，不得退回正文流内或压住底部输入区；当前消息滚动容器一旦存在有效文本选区，四键需立刻隐藏并释放命中区，待选区清空后再恢复。Chat 输出正文、Chat 最终 Markdown 正文和代码结果必须保持可选中文本语义，正文区域允许浏览器原生拖选、长按选中与复制；移动端最终输出不得安装脚本长按选区、假选中态、浮动复制层、编辑态兜底或视图级强制选择补丁，避免覆盖浏览器原生复制菜单。
- 上述阅读定位条必须作为消息区 overlay 渲染，不参与 `.runtime-timeline` 或 `chat-chat-screen` 的正文高度计算；空白会话、少量消息和短 turn 场景下，消息区不得因为按钮组自身占位出现额外滚动条或被拉出超出可视区的空白高度。

## 移动端体验

### 输入与键盘

- Chat 输入区基于 `--mobile-viewport-height` 动态视口高度适配软键盘。
- 移动端 root 不做 fixed 页面锁，也不对 `html / body / #frontend-root` 使用 `overflow: hidden` 根层锁；App Shell 使用 `height: var(--mobile-viewport-height, 100dvh)` 承接键盘后的可见高度，同时 root、workspace、timeline 与 Composer 容器必须裁切横向溢出并禁止页面级横向 scroll offset。键盘占位不再用于拉伸 App Shell、workspace header 或正文 panel，也不通过 transform 反向移动后方运行层，避免浏览器工具栏状态切换、输入聚焦或键盘动画造成底部留白、内容裁切或整页位移。
- `Chat` 的移动端会话列表共用左侧主导航抽屉：运行页顶部只保留 `Menu` 抽屉入口，并在抽屉中直接展示主工作流入口与当前Chat 会话列表；点击遮罩、切换路由、切换会话或新建会话后，不保留旧的抽屉展开态。
- `Chat` 的移动端左侧抽屉在真机上优先保证稳定性：遮罩保留淡入淡出，抽屉本体仅保留一层轻量侧滑，不再叠加多层位移、条目级顺序动画或生硬的整板平推过渡；抽屉和遮罩按动态可视高度裁剪，内部会话 rail/list 自行滚动，不得超出屏幕底部或带动页面级纵向拖拽。
- 输入区在软键盘弹起、收起、浏览器工具栏伸缩时持续贴住动态视口底部；Composer 作为 workspace grid footer 随 `--mobile-viewport-height` 变化，不使用 fixed bottom、transform 或 spacer 驱动主布局，正文滚动区、空态、命令候选和配置面板不消费键盘高度。
- 运行页后置样式不得重新对共享 `.runtime-composer-shell` 设置 `bottom: 0`、键盘 offset 或 fixed 定位；移动端 Composer 的唯一布局锚点是 `.runtime-workspace-body` 的真实 grid footer。该约束同时覆盖 Chat 和后续复用 Conversation runtime 的运行页，避免输入区覆盖最新用户消息或助手正文。
- 移动端 Composer footer 必须以安全区内边距包住输入面，form 边框不得贴住或越过屏幕边缘；外层 form 保留适度圆角，内部 textarea 保持直角，输入面高度、textarea 高度、工具行间距和工具按钮尺寸需维持紧凑稳定，不能用空白区撑开底部 footer。
- 仅在输入框实际聚焦或 visual viewport 明确报告键盘收缩时发布键盘诊断偏移；主布局不消费该偏移追加底部位移。
- 键盘收起或视口回弹后不保留额外底部空白。
- `Chat` 在页面恢复前台可见、浏览器重新激活当前标签页或系统恢复当前 WebView 时，必须立刻重算共享视口诊断变量；第一帧不得沿用后台前遗留的旧可视高度或旧底部空白。
- `Chat` 在刷新、页面初次装载或 WebView 重新激活时，若没有输入框实际聚焦且上一帧没有键盘证据，短暂或持续小于 layout viewport 的 `VisualViewport.height` 不得建立键盘态；App Shell 使用 layout viewport 作为可见高度，避免 Composer 停在键盘弹起位置。
- `Chat` 首次触摸主输入框时需保留浏览器原生软键盘手势，不在 `pointerdown / touchstart` 捕获阶段取消默认行为，不主动 focus，不锁定 `window` page scroll，也不通过 `scrollTo` 干预真实焦点或回放页面级滚动锚点。键盘开合过渡期内，`--mobile-viewport-height` 驱动 App Shell 可见高度，Composer 作为 workspace grid footer 跟随容器底边；输入框后方的 `workspaceBody / runtime-workspace-screen` 等滚动容器不短时锁定，移动 workbar 不消费 `VisualViewport.offsetTop` 做 transform，workspace header 与正文 panel 不单独消费 `VisualViewport` 变量。正文滚动区、空态、命令候选、配置面板和公共操作行由 App Shell 动态视口高度与静态 workspace inset 保持原位，不再做页面级滚动锚回，避免背景滚动与 iOS Safari 原生键盘动画互相竞争。首次弹出软键盘时公共操作行不得消失，也不得出现整页尺寸跳变。
- 主输入框首触后的键盘动画稳定窗口不得回放页面级滚动锚点，也不得通过 `window.scrollTo` 修正背景位置；Chat 在输入框聚焦且消息区原本贴近底部时可保持 `.runtime-workspace-screen` 的底部阅读距离。消息区一旦产生 `touchmove / pointermove / wheel / scroll` 意图，当前滚动容器必须继续即时响应。
- 主输入框保持聚焦时，消息区单指纵向拖动必须继续走浏览器原生滚动分派；前端不得通过 touch-scroll bridge、`touchmove.preventDefault()` 或脚本写入 `.runtime-workspace-screen.scrollTop` 来模拟滚动，避免破坏 iOS Safari 的惯性滚动与触摸响应。
- 消息区只有在实际内容高度超过视口高度时才开启纵向滚动与 iOS 惯性滚动；短消息、短回复、折叠 `Thinking / 已思考`、少量状态标签或加载中内容不足一屏时，`.runtime-workspace-screen` 必须关闭竖向触摸滚动和 overscroll 回弹，避免空滚动手势引发页面抖动。
- 软键盘打开时，真手机宽度下消息区实际高度必须由 App Shell 的 `--mobile-viewport-height` 和 workspace grid 共同收敛到当前可见阅读窗口，避免内部滚动容器仍以键盘未弹起时的过大高度参与触摸滚动。软键盘以 overlay 方式覆盖 layout viewport 时，也只能更新 App Shell 可见高度；`.runtime-workspace-panel` 只作为 grid 中间行，`.runtime-workspace-screen` 只保留固定阅读留白，不得通过额外底部 inset、手写 panel 高度、workspace grid 行高或 `runtime-composer-spacer` 扩展可滚范围，也不得改变顶部 workbar、workspace header 或空态的布局高度。
- Composer 的外层 footer、渐变背景、form surface 空白和外层留白不得制造独立滚动层或吞掉正文滚动手势；只有输入区内部真实控件接收事件，确保视觉上露出的后方消息区仍能直接拖动滚动。
- `Chat` 在移动端触摸发送按钮时，必须先 blur 当前主输入框，再继续原有发送链路；键盘收起期间 composer 继续随 `--mobile-viewport-height` 的真实回弹贴底，不能在发送后继续维持聚焦态或把输入区悬停在空白带上。
- `Chat` 的移动端顶部 workbar 必须作为 fixed 顶层固定在 `top: var(--mobile-viewport-offset-top, 0px)`，不通过 CSS transform 跟随浏览器栏滚动；workspace body 只保留对应高度的 header footprint。底部 Composer 是 `.runtime-workspace-body` 的真实 grid footer，随动态视口底边移动，不通过顶层 portal、fixed bottom 或 spacer 贴底；`.conversation-chat-screen` 与空态欢迎区在软键盘弹起期间只在内部滚动，不能因键盘高度变化带动顶部 header 或正文出现额外位移动画。
- `Chat` 在键盘收起和 composer 回弹到底边时，工作区滚动面保持原位；最后一屏消息、空态说明和阅读定位控件都不能在底边留下额外空白或残留占位。
- `Chat` 在移动端软键盘弹起期间，底部 Composer 必须高于阅读定位按钮；阅读定位按钮在主输入框聚焦后需主动隐藏，待输入框失焦、键盘收起后再恢复，不得压到输入框、附件条或键盘上方。左侧主导航抽屉打开时，当前主输入框必须 blur，workspace footer Composer 保持可见但设置为不可交互，抽屉与遮罩保持最高可触达层级。
- `Chat` 的移动端 overlay owner 由统一状态机发布，状态包括 `mobile-primary-nav-drawer`、`mobile-session-drawer`、`mobile-composer-panel`、`mobile-details-dialog` 与 `mobile-attachment-preview`；Composer 是否可交互只由该状态机推导，不由具体 route 或单个按钮额外传递局部开关。抽屉态不得通过 `visibility: hidden`、`opacity: 0` 或卸载 Composer 修正层级。
- 软键盘打开期间，除主输入框自身和正文消息滚动区的滚动手势外，运行页其他交互入口都必须先释放当前输入焦点再执行动作；该规则覆盖移动 workbar、主导航抽屉、Composer 工具栏、附件、发送、会话设置、遮罩和详情/配置入口。
- `Chat` 的主输入框在移动端必须按普通命令文本输入处理：关闭系统自动填充、卡片、地址与密码类输入辅助条，避免键盘上沿再挂出额外输入助手并露出底部残留页面层。
- `Chat` 的移动端主输入框需显式保持 16px 及以上可编辑文本字号；重新打开浏览器后首次聚焦输入法时，页面不得因 iOS Safari 自动输入框缩放而出现横向裁切、整体放大或分辨率突变。
- `Chat` 在移动端键盘弹起和收回期间，仅允许 App Shell 使用 `--mobile-viewport-height` 收缩到可见高度；顶部 workbar 是 workspace grid 第一行，底部 Composer 是 workspace grid footer，不得通过 fixed bottom 或 `VisualViewport.offsetTop` transform 追加位移；紧凑 workspace header、正文滚动区、空态、命令候选与配置面板保持布局原位，不跟随键盘做额外动画或跳变；阅读定位按钮在输入框聚焦期间隐藏。
- `Chat` 的移动端发送按钮支持在键盘保持打开时直接点按提交；首触发送需覆盖 `pointerdown(touch)` 与 `touchstart` 提交链路，并在同一次触摸内去重，立即进入当前 `sendPrompt` 链路，不需要先收键盘或补第二次点击。
- `Chat` 的 Composer 不额外叠加 `bottom` 过渡动画；键盘回弹与输入区回贴底边时只消费动态视口底边，避免补间动画与视口收缩/回弹叠加造成拖滞。
- `Chat` 在输入框失焦后，必须随 `--mobile-viewport-height` 恢复逐步释放高度；不允许先把 composer 闪回到底边，再被后续视口变化顶回去。
- `Chat` 在输入框保持聚焦且软键盘占位已建立后，不再用 `VisualViewport.height + offsetTop`、JS 键盘占位或页面级 scroll 锚点作为恢复判定；root、App Shell、workspace header、正文 panel 与 Composer 不得被脚本变量驱动位移。若浏览器已通过 `interactive-widget=resizes-content` 把布局视口缩到键盘后的高度，运行页只消费 `--mobile-viewport-height`，避免输入区被浏览器键盘动画和 CSS bottom 双重上移。
- `Chat` 的Chat runtime Composer 在真手机宽度下必须作为 `.runtime-workspace-body` 的真实 footer 行渲染，不得通过 `runtime-composer-portal-host` 渲染到 `document.body`，不得作为 fixed bottom 浮层，也不得依赖 `runtime-composer-spacer` 伪造 footprint；输入框阴影和白色 surface 在键盘弹起、收起或浏览器工具栏回弹期间不得留下灰色残影、旧层缓存或底部悬空阴影块。
- `Chat` 的四键阅读定位条统一使用同一套圆形按钮样式和触摸反馈，避免不同运行页在跳转控件上分叉出独立实现。
- `760px` 及以下的真手机宽度下，主导航抽屉、会话列表区、头部按钮高度与间距继续压缩，避免头部按钮挤占可用阅读高度。
- 小高度窄屏下，主导航抽屉仍需保留稳定的触摸滚动链：菜单内容滚动不把整个页面带离当前上下文，抽屉底部固定区域与菜单滚动区域边界清晰。
- `1280px` 及以下时，`Chat` 顶部固定提供 `Menu / New` 操作行；`Menu` 打开同一个左侧主导航抽屉，`New` 触发当前路由的新会话创建动作，操作入口不得因空态、已有消息或 `Details` 状态而消失。
- `760px` 及以下时，欢迎区 tag、标题与描述的顶部节奏需继续压缩；普通 `page-mode` 路由页内容区与 `Chat` 工作区也需沿用同一贴顶节奏，避免不同页面在窄屏下出现明显不一致的顶部留白。

### 会话设置

- Chat 的会话设置入口统一位于底部 Composer 工具栏的 `Session` 按钮；同一工具栏另设独立 GitHub 仓库按钮与附件按钮。发送按钮只负责提交当前文本、附件与可选仓库草稿。
- 移动端会话列表不再与正文上下堆叠，也不再使用Chat 内部独立抽屉；Chat都通过 `Menu` 打开左侧主导航抽屉。抽屉内的会话区左侧将会话标题与会话总数收敛为上下两行，右侧保留 `New` 入口并复用运行页紧凑按钮规格；列表项沿用标题-only 卡片与尾侧三点更多菜单结构，处理中会话显示 loading，并支持遮罩点击收起。
- 左侧主导航内的会话条目统一采用工作台列表项语义：置顶会话进入独立分组；距离 7 天不活跃清理阈值还剩不超过 2 天的未置顶会话进入 `Expiring Soon / 即将清理` 提醒分组；其余列表先按内容更新时间分组，缺少更新时间时才回退到创建时间，再在条目内展示标题与尾侧三点更多菜单；菜单内承载置顶、详情、删除动作，删除需确认弹窗；列表容器需保留独立滚动能力并输出稳定 `role="list"` 语义，视觉层级保持克制，不使用多余胶囊装饰。
- 会话设置展开后采用独立固定底部面板，带遮罩、关闭入口与内部滚动区。
- 会话设置面板的关闭路径在桌面与移动端保持一致：关闭按钮、遮罩、面板外点击和主输入框点击都必须走同一条收口逻辑，不保留“点输入框但面板仍悬停”的状态。
- 连续勾选 Skill、Tool、MCP 时，当前滚动位置保持稳定，不回到顶部。
- 设置面板标题、说明与标签在窄屏下保持可读，不重叠。

### 低功耗刷新

- 页面隐藏时停止高频扫描，恢复前台后补一次刷新。
- `Chat` 优先通过 owner 级增量轮询接收进行中会话变更；轮询响应受 `limit / byte_limit` 控制，`byte_limit` 默认/最大为 `1MiB`，默认只合并会话摘要、单 step patch 和 turn 收口 patch，不维持固定周期完整详情轮询。只有 `local_running / recovering` 会话参与自动 updates；忙碌会话的周期恢复先请求 `/api/chat/sessions/updates`，并随请求提交 `after_update_id` 与当前过程披露过滤对应的 `visible_event_kinds`；被过滤隐藏的纯过程变化不驱动前端合并，但 `latest_update_id` 仍推进。当 updates 连续返回空事件、返回的事件连续未命中本地仍判定为 `local_running / recovering` 的会话，或当前待恢复会话的 `updated_at / messages / process steps` 均未推进时，前端把这些结果视为 LLM 长耗时期间的正常无进展窗口，updates 轮询从约 `2s` 起步并退避到约 `3s / 5s / 8s`；只在连续无进展达到第 6 次以及之后每 8 次时按对应 `session_id` 补拉一次 bounded detail，用服务端详情校准最终状态、assistant 正文和失败/中断事实。当前会话收到新的 `updated_at`、消息或过程步骤时重置退避计数；详情仍未收敛时不重置退避计数。该兜底只读取最新详情页，不自动请求更早 `turn_before` 历史。
- 页面隐藏、移动端软键盘输入、滚动活跃或系统低功耗场景下，非必要轮询与重绘必须暂停或降频。
- 增量窗口过期、服务重启后返回 `resync_required`、显式手动刷新、详情打开、历史分页、本地缓存不完整或本地 `local_running / recovering` page-activation 补偿时，可直接补拉当前活动会话与仍处于 `local_running / recovering` 的会话详情；常规 updates 未对本地 `local_running / recovering` 会话产生相关进展时，仅按第 6 次以及之后每 8 次的连续无进展退避阈值补拉；新的 `updated_at`、消息或过程步骤属于有效进展，会重置连续无进展计数。详情补偿仅限最新 bounded 页，不替代用户显式加载更早历史。终态会话停止自动 updates 和 detail 请求，跨设备变化通过手动刷新、切换会话或下一次真实运行态恢复。

## 依赖与边界

- Runtime 提供消息路由与结构化执行结果。
- Skill 提供 Process 结构与最终答复。
- Chat 的会话式运行时交互由 Task, Chat & Workspace 领域维护；用户 prompt、最终输出与本领域运行页共享同一 `runtime-message-*` 消息格式。

## 验收口径

- Chat 默认进入 `main` Skill，Settings 页面不混入独立入口 Chat 历史。
- 新建空白会话不重复。
- 请求断开后后端运行时执行仍完成并写入历史。
- 页面恢复、网络重连或后端服务重启后，本地缓存中的旧消息不得长期显示 `In Progress`；若服务端已把孤儿运行态收敛为 `interrupted`，前端必须通过详情补拉或 updates resync 同步该状态。
- 移动端软键盘弹起与收起后输入区贴底，无回顶和残留空白。
- 长会话结果回填不触发整段消息列表重建。
- Chat 时间线顶部滚动先展开本地已加载历史；本地历史已完全展开且 `turns_paging.has_more_before=true` 时，才显式请求下一页服务端更早历史。
- Chat 移动端加载态下，`Thinking / 已思考` 披露行、用户消息与状态标签按正常时间线顶部起排，不得居中、错行或与用户气泡拉出异常间距。
- Chat 不后台自动补齐更早历史；显式历史加载只更新完整消息缓存和下一批可见历史，并保持当前阅读锚点，不刷新稳定消息批次。
- Chat 滚动触顶自动加载与点击“加载更早消息”使用同一批次扩展逻辑和滚动坐标恢复；连续触顶 scroll 事件在当前批次恢复完成前只合并为一次加载，不得把阅读区强制带回顶部。
- Chat 发送新消息后仍保留已加载历史；轻量输入响应不得把旧消息从时间线中替换掉。
- Chat 刷新或重开后可从 24 小时本地快照恢复完整已加载消息、过程 step 与已加载 event detail blocks，并在服务端详情返回后继续合并；长期快照写入使用稳定的 `runtime_trace_events` 字段。旧 sessionStorage 快照与非 canonical session id 不参与恢复。
- Chat 已有完整稳定本地缓存时可先用缓存恢复首屏；切回会话或刷新仍需按 active session 拉取最新 bounded detail 校准正文、状态与过程事件，普通前台恢复仅在本地 `local_running / recovering`、缓存不完整或存在可恢复占位时补偿，且不得自动请求更早 `turn_before` 历史页。
- `Chat` 的箭头四键阅读定位条可在滚动后稳定出现，并能把阅读位置跳到当前视口相邻的上一条或下一条内容。真机窄屏下，四键需保持固定右侧停靠、位于输入区上沿之上，且每个按钮保持圆形。
