# Conversation & Session Experience Requirements

> Last update: 2026-04-25

## 领域边界

Conversation & Session Experience 负责用户在 Web/Chat/Agent 页面中的会话、消息、流式展示、移动端适配、输入稳定性和阅读体验。它消费 Runtime、Agent、Task 的执行结果，但不定义底层执行器行为。

## 核心对象

| 对象 | 职责 |
| --- | --- |
| `Session` | 会话身份、标题、历史归属与生命周期 |
| `Message` | 用户与助手消息主数据 |
| `LiveUserMessage` | 执行中补充输入与当前轮可见用户意图 |
| `StreamEvent` | SSE 增量、完成、错误与保活事件 |
| `SessionHistoryBucket` | 按 Agent 或入口隔离的历史集合 |
| `ViewportState` | 移动端可视视口、键盘、滚动与输入状态 |

## 会话入口

### Web Shell

- 根路径 `/` 默认进入 Chat 工作台。
- `/chat` 提供 Chat、Agent、Terminal、Control 与 Memory 的统一 Web Shell。
- Web Shell 的前端构建源位于 `internal/interfaces/web/frontend`，`/chat` 固定分发 `static/dist/index.html`；该入口仅保留前端挂载容器与静态资源引用，由 React 渲染稳定的 shell DOM，并通过兼容样式层保持旧 DOM 契约。
- `/chat` 与 `/login` 默认以英文文案和 `html[lang="en"]` 启动；Web Shell 导航中的语言切换入口负责在英文与中文之间切换，并同步更新根节点语言标记。
- 登录页在启用密码保护时继续作为统一入口，但视觉需与 Web Shell 保持一致：使用 `IBM Plex Sans + Sora` 字体、近白工作台卡片和安全入口说明文案，不保留独立的默认系统表单风格。
- Web Shell 由 React 单一工作台直接渲染：`src/app/WorkbenchApp.tsx` 负责 hash 路由、语言切换、主导航折叠/抽屉与运行页/控制页分派；`chat` 与 `agent-runtime` 通过 `ConversationRuntimeProvider + ConversationWorkspace` 渲染 terminal-style workspace；`agent / terminal / memory / channels / skills / mcp / models / environments / cron-jobs / sessions / tasks` 等页面继续由 `ReactManagedRouteBody` 接管。根壳层稳定暴露 `app-shell[data-workbench-route]`，各运行页与 route body 继续输出 `data-route / data-conversation-*` 作为样式与测试锚点；兼容层仅保留样式，不再通过 legacy 脚本接管 `/chat` 业务运行时。
- `channels / skills / mcp / models / cron-jobs` 共享控制台卡片页统一复用同一组响应式卡片网格；窄屏下标题区允许徽标下沉、字段行改为单列堆叠、底部标签区保持纵向拉伸，避免复制按钮、状态徽标与多行字段发生重叠或横向溢出。
- Web Shell 的稳定界面基线收敛为两层：左侧固定主导航负责品牌、路由与语言切换，右侧主面板统一承载运行页和控制页；`Chat / Agent Runtime / Terminal` 在主面板内部统一采用「会话列 + 时间线工作区 + 底部 Composer + 固定 workspace header」结构，并直接复用会话栏、工作区容器、工作区头部、聊天滚动区、Composer 与窄屏 `Menu / Sessions / New` 复合 class 语义；固定 header 只保留当前会话标题、状态按钮和 `Details` 入口，具体会话详情与页面差异化内容统一在 `Details` 面板内承载，面板首屏默认以紧凑摘要栅格承载会话元数据和高频配置。`Terminal` 继续保持原有 `terminal-*` DOM class 契约与布局关系，状态与交互全部由 React 直接维护。为避免信息重复，当前壳层遵循单层信息架构：主导航不展示额外品牌口号或实现状态；Conversation workspace 自身承担会话和运行态配置，不再叠加 bridge 期的 welcome/runtime sheet 双轨壳层；`Control / Sessions / Tasks / Memory / Terminal / Codex Accounts` 等 React 托管 route 页继续共享近白表面、浅灰辅助层与浅蓝选中态。
- `Agent` 与其他 React 托管页面共享同一表面体系：列表卡片、管理表单、托管字段块与消息块使用一致的近白主表面、浅灰辅助层与浅蓝选中态。
- `/chat` 页面标题、登录页标题、导航品牌位、会话栏标题与欢迎区 tag 统一展示 `Alter0`，不再混用 `alter0` 小写品牌词。
- Web Shell 的抽屉式单列工作台仅在主视口宽度 `1100px` 及以下触发；高于该阈值时保留左侧固定主导航与右侧主面板，避免只对聊天内容列做最大阅读宽度限制而让整体壳层失衡。
- 进入窄屏工作台后，主导航切换为贴左侧视口边缘的全高抽屉；Conversation workspace 的会话列在同一 `1100px` 阈值收敛为与正文上下堆叠，不再作为独立浮层；`info-mode` 页面继续只保留主导航抽屉，避免壳层断点先切换而页面主体仍保留桌面列布局。
- 窄屏主导航抽屉中的菜单区必须作为独立滚动容器保留纵向滚动能力；当视口高度不足以完整容纳全部导航分组时，用户仍需能滑到 `Settings` 分组与语言切换入口，不得出现底部菜单项被裁切且无法继续下滑的状态。
- 窄屏主导航抽屉点击任一路由项后需立即收起；切页操作不会保留旧的菜单遮罩或抽屉层，用户进入目标页后直接看到新的正文区域。
- 窄屏主工作区按页面类型收口为贴顶起始区：普通 `page-mode` 路由页与 `Terminal` 工作区继续采用两行头部，第一行承载 `Menu / Sessions / New` 等抽屉入口；其中普通 `page-mode` 路由页至少要稳定提供 `Menu`，避免进入 `Tasks / Sessions / Models` 等信息页后失去主导航入口；第二行承载当前标题。`Chat` 与 `Agent Runtime` 空态也保留与 Terminal 对齐的单行紧凑 workspace header，在操作行下显示当前会话标题、状态按钮与 `Details` 入口，但不再展示模型、工具或目标摘要这类重复头部信息；所有页面正文都需贴近头部下沿起始，不得在顶部留下额外大块空白。
- `Chat` 空态欢迎区采用紧凑首屏节奏：桌面与中宽度下，欢迎 tag、标题、描述、target picker 与快捷提示需在 header 与 Composer 之间沿欢迎区中轴竖向居中展示；真窄屏继续贴近头部下沿起排。Composer 直接按自然文档流沿主工作区底边贴底排布；桌面与窄屏都不再通过自动顶距把 Composer 推到底边，避免欢迎区与输入区之间出现大块空白。
- 桌面端主导航采用紧凑间距节奏，优先保证在常见笔记本高度下完整展示主要模块组；控制类与资产类路由优先使用高密度主从或表格视图，避免在宽屏上保留大块无效留白。
- `static/dist/assets/*` 使用构建产物哈希文件名并返回长期 immutable 缓存；`/chat` 与 `static/dist/legacy/*` 下的兼容样式资源保持 `no-cache`，确保页面与样式能及时刷新到最新版本。
- `/login` 在登录密码启用时提供登录入口；`/logout` 清理当前登录态并回到登录流程。
- 登录密码未启用时，Web Shell 直接进入受保护页面；登录密码启用后，受保护页面和 API 使用同一登录态校验。

### Chat

- `Chat` 默认绑定内置 `main` Agent `Alter0`。
- `Chat` 面向通用对话入口，并允许在需要时调度专项 Agent。
- `Provider / Model`、`Tools / MCP`、`Skills` 可在会话过程中调整，并作用于后续发送的消息；其中 `Chat` 与 `Agent Runtime` 的 `Provider / Model` 选择器都需额外暴露内置 `Codex` 项，允许用户不经过常规 LLM Provider 直接切到 `Codex CLI` 执行链。Agent Runtime 的 `Skills` 面板需把当前 Agent 私有 Skill 固定展示为已启用且不可取消，未启用可选项只展示公有 Skill，不把其他 Agent 私有 Skill 暴露为可选项；`coding` 默认启用 `frontend-design`，新建 Coding Agent 会话的 Skills 摘要应直接体现该前端设计规则。

### Agent 页面

- `Agent` 页面承载无独立入口的内置 Agent 与用户管理 Agent。
- Agent 选项卡片在配置面板中展示短摘要，完整 system prompt 不直接暴露在选择面板。
- `Chat` 与 `Agent` 共用消息阅读和输入体验规范，但历史按目标 Agent 隔离。

### Session 历史

- Web 登录后，Chat/Agent 按目标 Agent 维护独立 Session 历史。
- 具备独立前端入口的 Agent 不进入通用 Agent 页面历史。
- `Sessions` 系统页面可展示跨来源会话数据，但不作为 Chat/Agent 分栏依据。

## 接口边界

- `GET /` 进入默认 Chat 工作台。
- `GET /chat` 返回 Web Shell。
- `GET /login` 与 `POST /login` 处理登录页和登录提交。
- `GET /logout` 清理当前登录态。
- `POST /api/messages` 处理 Chat 普通消息。
- `POST /api/messages/stream` 处理 Chat 流式消息。
- `POST /api/agent/messages` 处理指定 Agent 普通消息。
- `POST /api/agent/messages/stream` 处理指定 Agent 流式消息。
- 上述消息接口在 `content` 之外还接受 `attachments[]`；当前稳定支持两种图片输入：首次上传时携带 `data_url`、文件名与 MIME 类型，或在同一 Session 内复用已上传的 `id + asset_url + preview_url` 资产引用。允许仅发送图片，服务端会补齐稳定占位文本并把图片载荷并入统一消息元数据。
- `POST /api/sessions/{session_id}/attachments` 用于把会话图片提前写入当前 Session 工作区，并返回稳定 `asset_url / preview_url`。Conversation runtime 的草稿恢复、最近会话列表与已发送消息都应优先保存这组引用，不再长期持久化原始大图 `data_url`。
- `Terminal` 页面 Composer 与 `Tasks` 详情抽屉 follow-up terminal 输入也复用同一附件接口：图片先落到当前 Session 工作区，再以 `asset_url / preview_url` 引用参与提交；Terminal 额外允许常见文本/文档文件直接走同一接口上传原文件，并在返回中仅保留稳定 `asset_url`。前端草稿、预览与回显应优先消费这些稳定引用，而不是在这些链路里长期保留原始 `data_url`。
- assistant 最终回复中的 markdown 外链图片也属于会话图片资产：服务端在返回最终结果与落库前，需要把可下载的 `http(s)` 图片拉取到当前 Session 工作区并改写成 `/api/sessions/{session_id}/attachments/{asset_id}/original` 这类本地附件 URL；下载失败时保留原链接，不影响主回复返回。
- `GET /api/agents` 返回可进入 Agent Runtime 的专项 Agent；当前内置入口包括 `coding`、`writing` 与 `travel`，不包含绑定 Chat 默认入口的 `main / Alter0`。
- `GET /api/sessions` 查询会话摘要列表，支持来源和时间过滤。
- `GET /api/sessions/{session_id}/messages` 查询会话消息。
- `DELETE /api/sessions/{session_id}` 删除会话，并触发关联工作区和任务清理。

## 会话生命周期

### 标题

- 新会话先使用占位标题。
- 早期多轮输入仍偏通用时，标题可继续等待更具体输入。
- 后续出现更具体目标后，标题需自动升级，不长期停留在“拉取仓库”“分析仓库”等低辨识度名称。

### 空白会话

- 前端与后端链路不允许产生多个可见空白会话。
- 已存在空白会话时，`New Chat` 复用并聚焦当前空白会话。
- 会话产生有效用户消息后，才进入普通历史会话生命周期。

### 持久化与恢复

- 用户与助手消息主数据、路由结果、时间戳和来源字段必须持久化。
- 用户消息中的图片附件需要和文本一起进入会话时间线；页面刷新、切会话和最近会话恢复时保留稳定的图片预览资产，不重复持久化原始大图 payload。
- 页面刷新或服务重启后，用户可恢复最近会话与历史消息。
- 删除会话时同步清理关联任务记录与会话工作区。
- `Agent` 运行页会话列表为每个会话展示 8 位短 hash 标识；短 hash 用于前端列表辨识、预览域名映射与人工排障引用。

## 流式响应

### SSE 语义

- 流式接口返回 `start / delta / done` 等事件；Agent 工具循环期间还需返回结构化 `process` 事件。
- 长时间无模型增量时，SSE 通道持续发送保活帧。
- 复杂度评估可与首段回复并行进行。

### 执行不中断

- 已进入 Agent 执行链的请求不因浏览器断连、页面切换、标签页隐藏、SSE 写失败或前端取消而中断后端执行。
- 当前连接只负责回传；最终结果仍需落到会话历史。

### 断流恢复

- 已收到部分正文后若连接中断，前端保留已到达正文，并把该条消息收敛为失败态与可重试态。
- 浏览器本地缓存中残留的 `streaming` 消息在页面恢复时必须立即归一：无任务标识的消息转为失败态，带任务标识的消息转为对应任务态并继续轮询。
- 若流式连接在没有可用正文时失败，前端失败文案需明确提示刷新页面以恢复最新已保存回复。
- 页面刷新后，若当前活动会话存在本地失败态流式消息，前端需优先拉取服务端会话消息，并用已持久化结果覆盖本地失败态。
- 任务轮询与任务回填仅以真实存在的 `task_id` 为准，不得把空值或占位值误判为后台任务。

### 渲染策略

- Chat/Agent 消息区使用逐条 patch 与浏览器逐帧合并刷新。
- 高频流式增量、Process 展开收起与任务状态回填不得导致整段消息列表重建。
- Agent `process` 事件到达后，前端需在 `done` 前实时更新当前助手消息的步骤面板，而不是等待最终正文收口后一次性生成过程展示。
- Agent 消息中的 `Process` 优先使用服务端返回的结构化 `process_steps` 渲染；仅对缺失结构化步骤的历史消息保留文本解析兼容。
- 结构化 `process_steps` 需要在 SSE `done`、Task 结果回填与会话历史恢复后保持一致，刷新页面不得把已完成消息重新退化为仅正文展示。
- 助手最终回复提供一键复制；若消息含 Process，复制内容只包含最终正文。
- 前端所有绝对时间与时分标签统一按北京时间（`Asia/Shanghai`）渲染，并固定采用 24 小时制；浏览器本地时区不参与显示格式决策。
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

- 聊天气泡支持标题、列表、引用、链接、行内代码与代码块。
- 助手消息中的 markdown 图片按消息媒体统一以内联图片显示，使用浏览器懒加载策略，并保持链接可直接打开原图。
- 原始 HTML 不直接透传。
- 长路径、超长单词、代码块和 diff 只允许在内容块内部横向滚动，不撑破外层消息容器。

### Process

- Agent action / observation 与 Terminal 执行细节在前端收敛为可折叠 Process。
- 最终答复出现后，Process 默认折叠，阅读焦点回到正文。
- 用户手动展开或折叠后，在当前浏览器会话内保留状态。

### 布局

- 全站默认固定侧边栏；仅侧栏自身内容溢出时允许侧栏内部滚动。
- Chat 历史区支持折叠与展开，减少长对话阅读空间占用。
- Conversation workspace 的新会话入口在 `agent-runtime` 路由下切换为 Agent 会话语义，并随语言切换同步更新。
- Session 历史区的空态提示与列表可访问标签需按当前路由与语言即时切换文案；这些文案更新不得清空或重建 runtime 已注入的会话卡片节点。
- Session 历史区的会话列表需按最近时间分组为 `Today / Yesterday / Earlier`（中文对应 `今天 / 昨天 / 更早`），并与主导航 `menu` 复用同一套分组容器、hover 与激活态视觉；分组内条目保持主仓库式紧凑信息结构，按导航式线性关系排布：标题独立占一行并在可用宽度内单行截断，摘要信息单独换行，短 hash 固定在条目下缘，删除动作收纳在尾侧轻量文本操作中，不再额外挂出独立 footer 或胶囊操作面。
- Conversation workspace 头部的标题、状态按钮、`Details` 标签页和新会话入口需按当前路由与语言即时切换文案；这些壳层文案更新不得覆盖当前会话标题或消息内容。
- `Chat / Agent Runtime` 的会话列、工作区外壳、聊天滚动区和输入区需输出 `terminal-* + conversation-*` 复合 class，确保两条运行页与 `Terminal` 共用同一工作台表面与细节皮肤，同时保留 `data-conversation-*` 钩子供样式和测试使用。
- `Chat / Agent Runtime` 首页 Composer 采用单一紧凑主输入面：输入框、计数 meta 与发送按钮共处一张底部 surface，桌面与移动端都需压缩输入高度、外层留白与提交按钮体量，发送按钮直接复用 Terminal 的紧凑 icon submit 皮肤；PC 端上传、发送、状态、详情、短 hash 与弹窗动作统一采用 8-14px 低圆角矩形，不使用胶囊按钮或胶囊标签；会话卡片与 `Details` 面板保持同一浅色 terminal-runtime 质感，不再出现首页输入区厚重、旁侧组件过轻或材质不一致的情况。空态工作区需锁定为不可滚动表面，不允许通过空白区域拖拽把头部和输入区顶出可视区。
- `Chat / Agent Runtime` Composer 支持最多 5 张图片附件；附件在输入区以缩略图展示，可单张预览和移除，并按会话草稿持久化。当前选中的模型若未声明视觉能力，带图发送必须直接阻止并提示切换模型。
- 移动端 `Chat / Agent Runtime` 的会话抽屉遮罩、pane shell 与主工作区在 `1100px` 及以下需回落为静态表面，不保留模糊玻璃层或持续背景动效；性能优先级高于装饰层，确保真机滚动、抽屉开关和输入框聚焦不出现明显卡顿。
- 根工作台仅在窄屏时使用主导航抽屉；运行页内部的会话列不作为独立抽屉复用，避免出现导航抽屉和会话浮层叠加。
- 路由页头部的标题与副标题需按当前路由与语言即时切换文案；这些页头更新不得覆盖 route body 内已渲染的页面主体内容。
- 已由 React 接管的工作台需在 DOM 上暴露稳定路由钩子：根壳层输出 `app-shell[data-workbench-route]`，运行页和控制页继续输出 `data-route / data-conversation-*` 标记；兼容层只能依据这些由 React 输出的钩子退让，不得继续维护独立白名单。
- 欢迎区与 Composer 面板在同一主工作区内采用主仓库式上下结构：欢迎区直接输出 `Alter0 workspace` tag、面向 repo / task / runtime 的默认标题与说明、target picker 与快捷提示，Composer 独立贴底；欢迎区内容超出可视高度时，输入区仍需稳定贴底，不得与欢迎区、消息区发生叠层覆盖。
- 用户消息右对齐，宽度不超过消息区 80%，Chat / Agent / 路由消息区统一采用克制的冷灰工作台阅读主题；助手回复弱化厚重卡片层级，消息气泡、Process 区和阅读容器统一使用低对比边框、近白表面与有限强调色。
- Chat 助手消息尾部默认只显示时间；仅当回复仍在生成、排队或失败时展示紧凑状态标签，不再为已完成消息重复展示 route/source/status 元信息。
- Chat 与 Agent Runtime 工作区头部在进入会话态或桌面空态时收敛为共享单行标题区：只显示当前会话标题、状态按钮和 `Details` 入口，不再额外叠加 `Chat / Agent` 标签以及模型、工具或目标摘要；模型、Agent、Tools / MCP、Skills 与会话元数据统一放入 `Details` 面板，具体内容由各运行页自行实现，面板首屏先使用紧凑摘要栅格展示高频字段，再承接页内 tab 与配置区。两条运行页的 model tab 除已启用 Provider 模型外，都需稳定展示一个可直接点击的 `Codex` chip；选中该项后，后续消息请求不再携带普通 `alter0.llm.provider_id / alter0.llm.model` 组合，而是显式写入 `alter0.execution.engine=codex`。Agent Runtime 的 Agent tab 不展示 `Alter0/main` 主助手，只展示专项内置 Agent 与用户管理 Agent；Skills tab 区分当前 Agent 私有 Skill 与公有 Skill，私有项固定启用且禁用取消交互，公有项按会话选择进入启用或可选列表。`Details` 需以顶层浮层方式覆盖在工作区上方，内部独立滚动，浮层尺寸保持克制；页内 tab/按钮支持再次点击只收起当前 tab 内容区并保留摘要与 `Details` 面板，点击浮层外区域或按 `Escape` 才关闭整个面板，打开时不得推动消息列表、输入区或对话正文重新布局。
- 桌面宽屏下 Chat 消息列与 Composer 按主工作区宽度自适应放宽，并保持统一居中；正文区统一保留 `960px` 最大阅读宽度，但外层工作台也必须同步收缩导航与间距，避免在中等桌面宽度下出现阅读区限宽而整体布局仍然拥挤、遮挡或越界。
- Web Shell 主导航需根据 URL hash 即时同步当前路由高亮；导航折叠与语言切换更新不得导致会话卡片、消息节点或 route 内容被清空重建。
- React 壳层发出的主导航跳转、新建会话、欢迎区快捷提示、语言切换、导航折叠同步与会话历史折叠同步事件，必须由当前前端运行时在同一页面内完成确认、路由更新、快捷发送或会话创建，且不能要求用户重复点击或依赖额外脚本注入的全局函数。
- `Agent` 运行态消息区与 `Terminal` 对话区提供与主仓库 Terminal 一致的右侧箭头四键阅读定位条 `回到顶部 / 上一条 / 下一条 / 回到底部`：滚动超过阈值后显示顶部与底部入口，上一条与下一条按钮按当前可见消息块或 Terminal turn 实时重算目标；内容折叠、展开或重排后，按钮显隐与目标需同步更新。普通 `Chat` 消息区与其他托管 route 页不再展示该控件。

## 移动端体验

### 输入与键盘

- Chat/Agent 输入区基于 `VisualViewport` 同步有效视口高度。
- 移动端 App Shell 高度同步 `VisualViewport` 驱动的 `--mobile-viewport-height`，避免浏览器工具栏状态切换造成底部留白或内容裁切。
- `Chat / Agent Runtime` 的 `Menu` 与 `Sessions` 抽屉共用同一份移动端面板状态：两者保持互斥，打开会话列表时立即收起主导航，返回 `Menu` 时立即关闭会话列表；点击遮罩、切换路由、切换会话或新建会话后，不保留旧的抽屉展开态。
- `Chat / Agent Runtime` 的 `Menu / Sessions` 抽屉在真机上优先保证稳定性：遮罩保留淡入淡出，抽屉本体仅保留一层轻量侧滑，不再叠加多层位移、条目级顺序动画或生硬的整板平推过渡。
- 输入区在软键盘弹起、收起、浏览器工具栏伸缩时持续贴住可见底部。
- 仅在输入框实际聚焦且软键盘占位达到阈值时追加键盘底部偏移。
- 键盘收起或视口回弹后不保留额外底部空白。
- `Chat / Agent Runtime` 首次触摸输入框时需采用与 `Terminal` 相同的 `preventScroll` 聚焦策略，并在聚焦期间监听 `window.scroll + VisualViewport resize/scroll` 把页面锚定在 `scrollY = 0`；首次弹出软键盘时公共操作行不得消失，也不得出现整页尺寸跳变。
- `Chat / Agent Runtime` 的 fixed composer 在移动端必须把实际遮挡高度同步回工作区滚动面；`.conversation-chat-screen` 与空态欢迎区都需停在 composer 上沿，不能依赖静态底部 padding 估算占位，也不能出现底部输入框覆盖最后一段消息或说明文案。
- `Chat / Agent Runtime` 在键盘收起和 composer 回弹到底边时，工作区滚动面也必须同步清理旧的遮挡高度；最后一屏消息、空态说明和阅读定位控件都不能在底边留下额外空白或残留占位。
- `Chat / Agent Runtime` 在移动端键盘弹起和收回期间，仅允许 fixed composer 自身跟随 `VisualViewport` 做贴底位移；`Menu / Sessions / New` 操作行、紧凑 workspace header 与其他公共控件保持原位，不跟随键盘做额外动画或跳变。
- `Chat / Agent Runtime` 的移动端发送按钮支持在键盘保持打开时直接点按提交；首触发送立即进入当前 `sendPrompt` 链路，不需要先收键盘或补第二次点击。
- `Chat / Agent Runtime` 的 fixed composer 不额外叠加 `bottom` 过渡动画；键盘回弹与输入区回贴底边时只消费 `VisualViewport` 的实时位置，避免补间动画与视口收缩/回弹叠加造成拖滞。
- `Chat / Agent Runtime` 在输入框失焦后，若 `VisualViewport` 仍处于收缩态，必须继续保留当前键盘偏移并随视口恢复逐步释放；不允许先把 composer 闪回到底边，再被后续 viewport resize 顶回去。
- `760px` 及以下的真手机宽度下，主导航抽屉、会话抽屉、头部按钮高度与间距继续压缩，避免头部按钮挤占可用阅读高度。
- 小高度窄屏下，主导航抽屉仍需保留稳定的触摸滚动链：菜单内容滚动不把整个页面带离当前上下文，抽屉底部固定区域与菜单滚动区域边界清晰。
- `1100px` 及以下时，`Chat / Agent Runtime` 头部固定提供 `Menu / Sessions / New` 操作行；`Menu` 打开主导航抽屉，`Sessions` 打开会话抽屉，`New` 直接创建当前路由的新会话，操作入口不得因空态、已有消息或 `Details` 状态而消失。
- `760px` 及以下时，欢迎区 tag、标题与描述的顶部节奏需继续压缩；普通 `page-mode` 路由页内容区与 `Terminal` 工作区也需沿用同一贴顶节奏，避免不同页面在窄屏下出现明显不一致的顶部留白。

### 会话设置

- 移动端发送按钮与会话设置入口同排。
- 移动端会话列表不再与正文上下堆叠，而是采用独立左侧抽屉；抽屉内保留会话总数、`New` 和关闭入口，并支持遮罩点击收起。
- 会话抽屉内的会话条目统一采用工作台列表项语义：列表先按最近时间分组，再在条目内展示当前态文本、标题、摘要、底部短 hash 与尾侧删除动作；列表容器需保留独立滚动能力并输出稳定 `role="list"` 语义，视觉层级保持克制，不使用多余胶囊装饰。
- 会话设置展开后采用独立固定底部面板，带遮罩、关闭入口与内部滚动区。
- 移动端在主输入框与会话设置底部面板之间切换时，前端必须保证底部只保留一种主交互层：打开设置前先释放输入焦点并清理键盘占位，回到主输入框时先自动收起设置面板，再处理软键盘跟随与输入框贴底。
- 连续勾选 Skill、Tool、MCP 时，当前滚动位置保持稳定，不回到顶部。
- 设置面板标题、说明与标签在窄屏下保持可读，不重叠。

### 低功耗刷新

- 页面隐藏时停止高频扫描，恢复前台后补一次刷新。
- 输入聚焦、滚动活跃或移动端软键盘场景下降低非必要轮询与重绘。

## 依赖与边界

- Runtime 提供消息路由与流式事件。
- Agent 提供 Process 结构与最终答复。
- Task 提供任务卡片、状态回填和日志入口。
- Terminal 的会话式终端交互由 Task, Terminal & Workspace 领域维护，视觉基线与本领域保持一致。

## 验收口径

- Chat 默认进入 `main` Agent，Agent 页面不混入独立入口 Agent 历史。
- 新建空白会话不重复。
- SSE 断连后后端 Agent 执行仍完成并写入历史。
- 页面恢复后，本地缓存中的旧消息不得长期显示 `In Progress`。
- 移动端软键盘弹起与收起后输入区贴底，无回顶和残留空白。
- 长会话流式增量不触发整段消息列表重建。
- `Agent` 运行态消息区与 `Terminal` 对话区的箭头四键阅读定位条可在滚动后稳定出现，并能把阅读位置跳到当前视口相邻的上一条或下一条内容；普通 `Chat` 消息区与其他托管 route 页不出现这组按钮。
