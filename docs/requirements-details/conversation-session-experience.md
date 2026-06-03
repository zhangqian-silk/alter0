# Conversation & Session Experience Requirements

> Last update: 2026-06-03

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
- `/chat`、`/agent-runtime` 与 `/terminal` 提供三条主工作流入口；主导航底部固定提供 `Management` 工具入口并进入 `/management`。Control、Memory、Tasks、Sessions、Models、Environments、Codex Accounts 等管理能力统一收敛到同一个 Management 页面内分组展示，不再保留独立工作台 path，并继续复用同一 Web Shell、登录态、主导航与页面骨架。
- Web Shell 的前端构建源位于 `internal/interfaces/web/frontend`，`/chat` 固定分发 `static/dist/index.html`；该入口仅保留前端挂载容器与静态资源引用，由 React 渲染稳定的 shell DOM，并通过兼容样式层保持旧 DOM 契约。
- `/chat` 与 `/login` 默认以英文文案和 `html[lang="en"]` 启动；Web Shell 导航中的语言切换入口负责在英文与中文之间切换，并同步更新根节点语言标记。
- 登录页在启用密码保护时继续作为统一入口，但视觉需与 Web Shell 保持一致：使用 `IBM Plex Sans + Sora` 字体、近白工作台卡片和安全入口说明文案，不保留独立的默认系统表单风格。
- Web Shell 由 React 单一工作台直接渲染：`src/app/WorkbenchApp.tsx` 负责 `/chat`、`/agent-runtime`、`/terminal`、`/management` 四个稳定顶层路由、语言切换、主导航折叠/抽屉和运行页/管理页分派；主导航的主工作流只暴露 `Chat / Agent / Terminal`，并用单个 `Management` 工具入口进入合并管理页；`chat` 与 `agent-runtime` 通过 `ConversationRuntimeProvider + ConversationWorkspace` 渲染 terminal-style workspace；`terminal` 通过 `/terminal` 独立 path 进入并继续由共享 runtime host 承载；`agent / memory / channels / skills / mcp / models / environments / cron-jobs / sessions / tasks / codex-accounts` 等能力由 `/management` 页面内的本地分区切换接管。根壳层稳定暴露 `app-shell[data-workbench-route]` 与 `data-route-family="management"`，各运行页与 route body 继续输出 `data-route / data-conversation-*` 作为样式与测试锚点；兼容层仅保留样式，不再通过 legacy 脚本接管 `/chat` 业务运行时。
- `channels / skills / mcp / models / cron-jobs` 共享控制台卡片页统一复用同一组响应式卡片网格；窄屏下标题区允许徽标下沉、字段行改为单列堆叠、底部标签区保持纵向拉伸，避免复制按钮、状态徽标与多行字段发生重叠或横向溢出。
- Web Shell 的稳定界面基线收敛为两层：左侧固定主导航负责品牌、三条主工作流入口、当前运行页会话列表、Management 工具入口与语言切换，右侧主面板统一承载运行页和 Management 管理页；`Chat / Agent Runtime / Terminal` 在主面板内部统一采用「时间线工作区 + 底部 Composer + 固定 workspace header」结构，并直接复用工作区容器、工作区头部、聊天滚动区、Composer 与移动端顶部操作行复合 class 语义，标题、状态按钮和 `Details` 按钮也保持同一组共享 header 元素。到真手机宽度时，运行页顶层再收敛成单层 workbar：左侧 `Menu`，中间“状态信号 + 当前会话标题”标题按钮，右侧固定保留 `New`；三条运行页都不再显示独立 `Sessions` 顶部按钮，会话列表随左侧导航抽屉展示。`Details` 不再独占手机顶栏，而是由中间标题按钮直接触发。`Terminal` 继续保持原有 `terminal-*` DOM class 契约与布局关系，状态与交互全部由 React 直接维护，但不再通过专属 header kind 或独立 details toggle 派生不同头部皮肤。为避免信息重复，当前壳层遵循单层信息架构：主导航不展示额外品牌口号或实现状态；Conversation workspace 自身承担运行态配置，不再叠加 bridge 期的 welcome/runtime sheet 双轨壳层；`Control / Sessions / Tasks / Memory / Codex Accounts` 等 React 托管 route 页进入同一 Management 页面，页内共享近白表面、浅灰辅助层与浅蓝选中态，并用紧凑分组切换区分能力。
- Web Shell 的稳定界面基线收敛为两层：左侧固定主导航负责品牌、三条主工作流入口、当前运行页会话列表、Management 工具入口与语言切换，右侧主面板统一承载运行页和 Management 管理页；`Chat / Agent Runtime / Terminal` 在主面板内部统一采用「时间线工作区 + 底部 Composer + 固定 workspace header」结构，并直接复用工作区容器、工作区头部、聊天滚动区、Composer 与移动端顶部操作行复合 class 语义，标题、状态按钮和 `Details` 按钮也保持同一组共享 header 元素。到真手机宽度时，运行页顶层再收敛成单层 workbar：左侧 `Menu`，中间“状态信号 + 当前会话标题”标题按钮，右侧固定保留 `New`；三条运行页都不再显示独立 `Sessions` 顶部按钮，会话列表随左侧导航抽屉展示。`Details` 不再独占手机顶栏，而是由中间标题按钮直接触发。`Terminal` 继续保持原有 `terminal-*` DOM class 契约与布局关系，状态与交互全部由 React 直接维护，但不再通过专属 header kind 或独立 details toggle 派生不同头部皮肤。为避免信息重复，当前壳层遵循单层信息架构：主导航不展示额外品牌口号或实现状态；Conversation workspace 自身承担运行态配置，不再叠加 bridge 期的 welcome/runtime sheet 双轨壳层；`Control / Sessions / Tasks / Memory / Codex Accounts` 等 React 托管 route 页进入同一 Management 页面，页内共享近白表面、浅灰辅助层与浅蓝选中态，并用紧凑分组切换区分能力。Management 桌面布局需使用左侧管理索引和右侧内容区，分区入口提供图标、短标识与活动态；真手机宽度下索引切换为双列入口栅格，保证 Profiles、Memory、Skills、MCP、Sessions、Tasks、Cron Jobs、Channels、Models、Environments 与 Codex Accounts 都可直接扫读。各分区内部的卡片、表格、筛选条、详情面板、空态和错误态需共享一致的 12px 以内低圆角、低阴影、浅色状态面板和紧凑字段行。
- `Agent` 与其他 React 托管页面共享同一表面体系：列表卡片、管理表单、托管字段块与消息块使用一致的近白主表面、浅灰辅助层与浅蓝选中态。
- `/chat` 页面标题、登录页标题、导航品牌位、会话栏标题与欢迎区 tag 统一展示 `Alter0`，不再混用 `alter0` 小写品牌词。
- Web Shell 的抽屉式单列工作台仅在主视口宽度 `1100px` 及以下触发；高于该阈值时保留左侧固定主导航与右侧主面板，避免只对聊天内容列做最大阅读宽度限制而让整体壳层失衡。
- 进入窄屏工作台后，主导航切换为贴左侧视口边缘的全高抽屉；当前运行页的会话列表随主导航一起进入同一个左侧抽屉，不再在工作区内部生成独立浮层或上下堆叠列；真手机宽度下抽屉使用近白全高面板、平面菜单和细分割线，会话区按自然高度滚动，不再出现重卡片式列表容器；`info-mode` 页面继续只保留主导航抽屉，避免壳层断点先切换而页面主体仍保留桌面列布局。
- 窄屏主导航抽屉中的菜单区必须作为独立滚动容器保留纵向滚动能力；一级菜单固定为 `Chat / Agent / Terminal`，语言切换入口保持可触达，不得出现底部菜单项被裁切且无法继续下滑的状态。
- 窄屏主导航抽屉点击任一路由项后需立即收起；切页操作不会保留旧的菜单遮罩或抽屉层，用户进入目标页后直接看到新的正文区域。
- 窄屏主工作区按页面类型收口为贴顶起始区：Management 管理页继续采用两行头部，第一行承载 `Menu`，第二行承载统一标题，并在正文起始处提供页内分组切换；中等窄屏允许分组入口横向滚动，真手机宽度必须回落为双列换行入口，避免进入 `Tasks / Sessions / Models` 等管理能力后失去主导航入口或管理分区入口；`Chat / Agent Runtime / Terminal` 在真手机宽度下统一采用单层运行页 workbar，不再叠加第二层 workspace header。运行页标题区必须在一行内承载当前会话标题与状态信号，并通过点击标题打开 `Details`；不得在操作行下重复输出模型、工具或目标摘要，也不得把详情入口再拆成额外按钮层。所有页面正文都需贴近头部下沿起始，不得在顶部留下额外大块空白。
- `Chat` 空态欢迎区采用紧凑首屏节奏：桌面与中宽度下，欢迎 tag、标题、描述、target picker 与快捷提示需在 header 与 Composer 之间沿欢迎区中轴竖向居中展示；真窄屏继续贴近头部下沿起排。Composer 直接按自然文档流沿主工作区底边贴底排布；桌面与窄屏都不再通过自动顶距把 Composer 推到底边，避免欢迎区与输入区之间出现大块空白。
- 桌面端主导航采用紧凑间距节奏，主工作流只保留三条，并在底部固定 `Management` 工具入口；控制类与资产类路由在 Management 页内部优先使用高密度主从或表格视图，Management 分组导航自身作为左侧索引常驻，避免在宽屏上保留大块无效留白或把分区入口堆成顶部标签云。
- `static/dist/assets/*` 使用构建产物哈希文件名并返回长期 immutable 缓存；`/chat` 与 `static/dist/legacy/*` 下的兼容样式资源保持 `no-cache`，确保页面与样式能及时刷新到最新版本。
- `/login` 提供统一登录入口；`/logout` 清理当前登录态并回到登录流程。
- Web Shell、短哈希预览 host 与受保护 API 统一使用同一登录态校验；静态只读预览 host 保留匿名访问。

### Chat

- `Chat` 默认绑定内置 `main` Agent `Alter0`。
- `Chat` 面向通用对话入口，并允许在需要时调度专项 Agent。
- `Provider / Model`、`Tools / MCP`、`Skills` 可在会话过程中调整，并作用于后续发送的消息；其中 `Chat` 与 `Agent Runtime` 的 `Provider / Model` 选择器都需额外暴露内置 `Codex` 项，允许用户不经过常规 LLM Provider 直接切到 `Codex CLI` 执行链。Agent Runtime 的空会话欢迎区需直接渲染 Agent 选择器，并在浏览器侧缓存最近一次手动选择，供下一次新会话默认回填；`Details > Agent` 仍可继续查看或调整首条消息前的目标。移动端欢迎区中的 Agent 选择器使用单列列表结构，条目正文只保留 Agent 名称与一行简短介绍，不再使用多列网格堆叠较长说明；左侧主图标需使用能直接表达能力方向的语义化图案，而不是默认字母缩写块。Agent Runtime 的 `Skills` 面板需把当前 Agent 私有 Skill 固定展示为已启用且不可取消，未启用可选项只展示公有 Skill，不把其他 Agent 私有 Skill 暴露为可选项；`coding` 默认启用 `memory`、`deploy-test-service`、`frontend-design`、`artifact-preview`、`doc-coauthoring`、`fullstack-developer`、`code-reviewer`、`webapp-testing`、`find-skills`、`test-driven-development`、`ui-ux-pro-max`、`code-simplifier`、`code-review` 与 `brainstorming`，新建 Coding Agent 会话的 Skills 摘要需直接体现这套工程执行默认集；`travel` 默认启用 `memory`、`deploy-test-service`、`frontend-design`、`artifact-preview`、`doc-coauthoring`、`find-skills`、`ui-ux-pro-max` 与 `brainstorming`，新建 Travel Agent 会话的 Skills 摘要需直接体现这套旅游执行默认集。

### Agent 页面

- `Agent` 页面承载无独立入口的内置 Agent 与用户管理 Agent。
- Agent 选项卡片在配置面板中展示短摘要，完整 system prompt 不直接暴露在选择面板。
- `Chat` 与 `Agent` 共用消息阅读和输入体验规范，但历史按目标 Agent 隔离。

### Session 历史

- Web 登录后，Chat/Agent 已发送会话通过服务端 Session history 在同一 Web 登录态下跨设备共享；`Chat` 固定维护单一长期逻辑会话 `alter0-chat`，`Agent Runtime` 继续按目标 Agent 维护独立 Session 历史。
- 本地 Session history 按会话类型拆分物理文件：`Chat` 的 `alter0-chat` 按北京时间 05:00 的归档日边界写入 `.alter0/sessions/_default/alter0-chat/<YYYY-MM-DD>.json` 或 `.md`；`Agent Runtime` 按目标 Agent 与会话身份写入 `.alter0/sessions/<agent_id>/<session_id>.json` 或 `.md`。缺少 Agent 来源的非 Chat 会话归入 `_default`；服务读取旧版 `.alter0/sessions.json` 或 `.alter0/sessions.md` 聚合文件时需立即重构为新的分文件布局，并删除旧聚合文件。
- Chat/Agent Runtime 消息接口接受请求后，服务端先把本轮 `user` 消息写入 Session history，再进入同步或流式执行；assistant 回复在执行完成、失败或任务收口后追加写入。同一轮请求的浏览器关闭、刷新、SSE 断开或前端取消不会让用户已发送内容只留在浏览器缓存中。
- 具备独立前端入口的 Agent 不进入通用 Agent 页面历史。
- `Sessions` 系统页面可展示跨来源会话数据，但不作为 Chat/Agent 分栏依据。
- 未发送文本草稿、附件草稿与当前浏览器中的临时空白会话允许继续本地保存；这些局部态不要求跨设备同步，但不能覆盖服务端已存在的会话摘要、配置与消息历史。

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
- `POST /api/sessions/{session_id}/attachments` 用于把会话图片提前写入当前 Session 工作区，并返回稳定 `asset_url / preview_url`。Conversation runtime 的草稿恢复、最近会话列表与已发送消息都应优先保存这组引用，不再长期持久化原始大图 `data_url`；其中 `preview_url` 只用于缩略图位，历史消息回显与预览弹层必须优先读取 `asset_url` 原图。
- `Terminal` 页面 Composer 与 `Tasks` 详情抽屉 follow-up terminal 输入也复用同一附件接口：图片先落到当前 Session 工作区，再以 `asset_url / preview_url` 引用参与提交；Terminal 额外允许常见文本/文档文件直接走同一接口上传原文件，并在返回中仅保留稳定 `asset_url`。前端草稿、缩略预览与历史回显应优先消费这些稳定引用，而不是在这些链路里长期保留原始 `data_url`；其中缩略位继续使用 `preview_url`，再次查看时统一切回 `asset_url`。
- assistant 最终回复中的 markdown 外链图片也属于会话图片资产：服务端在返回最终结果与落库前，需要把可下载的 `http(s)` 图片拉取到当前 Session 工作区并改写成 `/api/sessions/{session_id}/attachments/{asset_id}/original` 这类本地附件 URL；下载失败时保留原链接，不影响主回复返回。
- `GET /api/agents` 返回可进入 Agent Runtime 的专项 Agent；当前内置入口包括 `coding`、`writing` 与 `travel`，不包含绑定 Chat 默认入口的 `main / Alter0`。
- `GET /api/conversation-runtime/sessions?route=chat|agent-runtime` 返回运行页会话摘要，至少包含标题、目标 Agent/Model、Tools / Skills / MCP 选择、创建时间与稳定 session id。
- `GET /api/conversation-runtime/sessions/{session_id}?route=chat|agent-runtime` 返回单个运行页会话详情，至少包含历史消息、用户附件引用、结构化 `process_steps` 与当前恢复到的运行态配置。
- `GET /api/sessions` 查询会话摘要列表，支持来源和时间过滤。
- `GET /api/sessions/{session_id}/messages` 查询会话消息。
- `DELETE /api/sessions/{session_id}` 删除会话，并触发关联工作区和任务清理。

## 会话生命周期

### 标题

- 新会话先使用统一占位标题 `New`。
- 早期多轮输入仍偏通用时，标题可继续等待更具体输入。
- 后续出现更具体目标后，标题需自动升级，不长期停留在“拉取仓库”“分析仓库”等低辨识度名称。

### 空白会话

- 前端与后端链路不允许产生多个可见空白会话。
- 已存在空白会话时，`New` 复用并聚焦当前空白会话。
- 会话产生有效用户消息后，才进入普通历史会话生命周期。

### 持久化与恢复

- 用户与助手消息主数据、路由结果、时间戳、来源字段以及恢复运行页所需的请求 metadata 必须持久化。
- 用户消息中的图片附件需要和文本一起进入会话时间线；页面刷新、切会话和最近会话恢复时保留稳定的图片资产引用，不重复持久化原始大图 payload；时间线内图片与预览弹层再次查看时必须优先显示原图资源，避免缩略图被放大后失真。
- 未发送文本草稿在桌面端输入期间允许延迟写回浏览器缓存；当前输入值、切换前 flush、刷新后的草稿恢复和发送结果必须保持一致，不能为了持久化把每次按键都绑定到同步存储写入。
- 页面刷新、跨设备重开或服务重启后，用户可恢复最近会话与历史消息；恢复结果需保留当前 Session 的目标 Agent、Model 与 Tools / Skills / MCP 选择。
- 页面刷新时，前端需先用浏览器侧保存的当前活动会话快照恢复最近一条活跃 `Chat / Agent Runtime` 会话，避免服务端列表短暂缺席时把当前会话清空或替换为新的空白会话；随后再按 `session_id` 回源单会话详情，用服务端最新结果覆盖本地快照。
- `POST /api/messages`、`POST /api/messages/stream`、`POST /api/agent/messages` 与 `POST /api/agent/messages/stream` 在 Web 层接受请求后，后端执行与持久化不得再依赖浏览器连接持续存活；页面刷新、标签页切换、SSE 断开或前端取消只允许中断当前传输，不得直接取消本轮会话执行。
- `Chat` 当前活动会话固定为 `alter0-chat`，不写入或读取 URL query 作为多会话入口；`Agent Runtime` 当前活动会话需稳定投影到 URL query：`/agent-runtime?session_id=<8位短hash>`。页面首次加载、刷新、手动粘贴当前链接或浏览器恢复标签页时，`Agent Runtime` 先读取 `session_id` 恢复目标会话；只有参数缺失或目标会话不存在时，才允许回退到浏览器快照与服务端列表默认项。
- 浏览器侧会额外持久化最近会话列表的轻量快照，而不只保留当前活动会话；当用户刷新其他会话、切换设备前短暂刷新，或服务端集合接口暂时漏掉刚创建/最近活跃会话时，前端仍需在侧栏继续展示这些最近会话，并按 `session_id` 单独补拉详情，直到服务端明确确认不存在。
- `Chat / Agent Runtime` 需维护独立的服务端会话 registry，记录 `session_id -> route / title / target / model / capabilities / status / updated_at` 等最小恢复视图；浏览器本地快照只作为次级兜底，不承担会话存在性的唯一事实来源。
- 删除会话时同步清理关联任务记录与会话工作区。
- `Chat / Agent Runtime / Terminal` 会话列表统一由左侧主导航承载，使用 `Sessions` 标题与 `New` 新建入口；移动端通过同一个左侧导航抽屉展示当前运行页会话列表。运行页互相切换时，左侧会话列表的 `Sessions` 标题与 `New` 按钮由主导航稳定持有，不随页面切换重建；当前运行页只更新数量文案、列表内容和 `New` 动作绑定，rail 数据尚未注册时使用稳定 fallback rail，已访问过的运行页切回时先复用该 route 最近一次有效 rail body，不得先清空公共 rail、回退占位 rail 再恢复。列表项主体只展示标题，处理中会话在标题旁显示 loading，其他状态不显示状态灯、时间、短 hash、Agent 标签或额外摘要。运行页空列表和 Terminal 加载态优先展示一条 active `New` 占位会话；Terminal 占位会话不立即写入服务端，首次发送输入或添加附件后才创建真实 Terminal session 并替换占位项。三条运行页继续生成同一规则的 8 位短 hash，用于运行页 URL 恢复、预览域名映射与人工排障引用；左侧会话列表不展示短 hash。完整会话 id 与 Terminal `terminal_session_id` 继续用于接口、持久化和工作区隔离，不直接作为列表或 URL 展示值。
- `Chat / Agent Runtime` 的会话列表项与 workspace header 状态按钮共享同一会话状态语义：前端按当前 assistant 消息与任务态派生 `ready / busy / failed`；其中 `streaming / queued / running / in_progress` 与挂起任务映射为 `busy`，错误、失败、取消与显式 `message.error` 映射为 `failed`，其余稳定态映射为 `ready`。列表项只在 `busy` 时于标题旁显示 loading，`ready / failed` 不显示行内状态灯；workspace header 的状态按钮可见层只显示信号，状态名称只保留给读屏与悬浮语义。

## 流式响应

### SSE 语义

- 流式接口返回 `start / delta / done` 等事件；Agent 工具循环期间还需返回结构化 `process` 事件。
- 长时间无模型增量时，SSE 通道持续发送保活帧。
- 复杂度评估可与首段回复并行进行。

### 执行不中断

- 已进入 Agent 执行链的请求不因浏览器断连、页面切换、标签页隐藏、SSE 写失败或前端取消而中断后端执行。
- 当前连接只负责回传；最终结果仍需落到会话历史。

### 断流恢复

- `Chat / Agent Runtime` 在同一 Session 内保持追加式会话历史；每轮请求都要追加新的用户消息与新的助手消息占位，不得把后续回复继续回写到已完成的历史消息。
- `Chat` 固定使用单一长期逻辑会话 `alter0-chat`，页面初始化、发送消息、附件草稿、服务端回源和刷新恢复都围绕该会话进行；前端不得再把 URL、浏览器快照或服务端历史中的旧 Chat session 作为可切换会话入口。`Agent Runtime` 继续保留多 Agent、多 session 的运行态。
- `Chat` 的长期历史按北京时间 05:00 作为归档日边界分文件存储；05:00 之前的消息归入前一归档日，05:00 及之后的消息归入当天归档日。该分文件规则只改变本地存储和迁移形态，不改变 `alter0-chat` 对外的逻辑 `session_id`。
- `Chat` 直连 Codex 的 thread id 与同一归档日绑定，写入当前 Chat 工作区 `.alter0/codex-runtime/threads/<YYYY-MM-DD>.json`；归档日切换后，新文件不存在即代表新的 Codex 会话环境，运行时不得继续 resume 前一归档日的 Codex thread。
- 单条 assistant 消息内部仍可按 `process / delta / done` 流式更新，但补丁目标只能是当前活跃的未完成消息；消息进入 `done` 或任务态后，迟到的 SSE 事件必须直接丢弃。
- 运行页初始化时，服务端会话详情回填不得覆盖当前浏览器里已经新追加、但服务端详情请求发起时尚未落库的本地消息；本地新消息与流式更新优先级高于陈旧详情响应。
- 若运行页刷新后服务端会话集合接口暂时未包含当前活动 `session_id`，前端仍需保留本地恢复出的该条会话，并主动尝试按 `GET /api/conversation-runtime/sessions/{session_id}` 补拉详情；在单会话详情也确认不存在之前，不得立刻创建新的空白会话顶替当前活动会话。
- 若运行页刷新后服务端会话集合接口暂时未包含某条最近会话，即使该会话当前并非活动会话，前端也不得立刻把它从 `Sessions` 列表移除；左侧最近会话列表以本地快照和服务端结果合并视图为准，只有在用户显式删除或后续回源明确确认不存在时才允许消失。
- 会话 registry 的状态更新链路需与 Terminal 类似由服务端驱动：消息入口在请求接收后立即标记当前会话为 `busy`，同步完成后标记为 `ready`，请求失败时标记为 `failed`；当 Session history 仍未对当前请求可见时，集合接口与单会话详情也必须先返回 registry 里的最近视图，而不是直接返回缺失或 404。
- `travel` 会话若在本轮执行中先收到了正文攻略，但 HTML 页与 `travel` 子域名尚未就绪，服务端需继续在同一轮完成自动页面固化与发布收口；前端只在自动收口结束后接收最终成功态或明确阻塞态，不把“纯文本成功但无页面”的中间态当作完成结果。
- 已收到部分正文后若连接中断，前端保留已到达正文，并把该条消息收敛为失败态与可重试态。
- 浏览器本地缓存中残留的 `streaming` 消息在页面恢复时必须立即归一：无任务标识的消息转为失败态，带任务标识的消息转为对应任务态并继续轮询。
- 若流式连接在没有可用正文时失败，前端失败文案需明确提示刷新页面以恢复最新已保存回复。
- 若流式连接在本轮消息已收到 `start` 后中断，但尚未收到 `done`，前端需先拉取当前会话详情，用服务端已持久化的 `assistant` 消息覆盖本地 `Thinking...`、浏览器读流错误或其他临时失败态；该恢复链路不得通过重新提交同一条用户消息来实现。
- 页面刷新后，若当前活动会话存在本地失败态流式消息，前端需优先拉取服务端会话消息，并用已持久化结果覆盖本地失败态；即使服务端集合接口已经返回了该会话的摘要项，只要未附带完整消息，也必须继续补拉单会话详情。
- 页面刷新、前后台恢复或集合接口刷新后，若服务端集合返回的当前会话消息数量短于浏览器侧已追加历史，且本地仍存在本轮用户消息、`Thinking...`、`streaming` assistant 或其他可恢复状态，前端必须继续保留本地时间线并等待单会话详情或后续集合结果确认，不得把刚发送的消息从 UI 中移除。
- 若当前活动会话已经从服务端恢复出最新 `user` 消息，但该 user 之后尚无 assistant、任务或失败消息，运行页继续按待恢复会话处理并重试详情接口；只有拿到稳定 assistant 或明确失败态后，才停止本轮恢复。
- 任务轮询与任务回填仅以真实存在的 `task_id` 为准，不得把空值或占位值误判为后台任务。

### 渲染策略

- Chat/Agent 消息区使用逐条 patch 与浏览器逐帧合并刷新；连续 SSE `delta` 文本增量需先在前端内存态累计，再按固定短窗口合并刷新可见 assistant 文本，`done / error / 中断恢复` 事件到达时立即清理待刷新任务并收口最终状态。
- 高频流式增量、Process 展开收起与任务状态回填不得导致整段消息列表重建，也不得在长输出期间持续占满主线程导致导航、发送、详情和会话切换按钮失去响应。
- 时间线装配需按单条消息缓存稳定渲染结果；当仅当前流式 assistant 消息变化时，未变化的历史消息不得重新生成 Markdown HTML、Process step 树或 runtime timeline item。
- 仅有 Composer 草稿变化时，Conversation 时间线、Markdown 正文与 `Process` 展示不得重新解析或整段重建；性能热点需收敛在输入区本身。
- 前端点击诊断仅在 `?debug_clicks=1`、`?debugClicks=true` 或 `localStorage["alter0.debug.clicks"]="on"` 时启用。诊断记录需覆盖 `pointerdown / pointerup / touchstart / touchend / click` 的事件目标、坐标命中的顶层元素、当前 `activeElement`、壳层 class、可拦截点击的遮罩层、`defaultPrevented` 与按钮禁用状态，并额外输出主线程 long task，用于排查真机首点无效和双击才响应问题。
- `Chat / Agent Runtime / Terminal` 的移动端运行页头部按钮需直接响应首个 `touchstart` 或非鼠标 `pointerdown`：`Menu`、中间标题详情入口、`New` 会话入口以及保留的移动端会话入口都不得依赖后续合成 `click` 才触发；同一触摸产生的后续 `click` 需被去重，避免一次手势执行两次。
- Terminal 长输出复制按钮不得把完整输出写入 `data-*` 属性或其他 DOM 元数据；复制 payload 保留在组件闭包并通过剪贴板 API / document copy 兜底写出，避免长日志在 DOM 中重复放大并拖慢轮询、选择和点击。
- Terminal 时间线渲染需按 `turns / 展开态 / step 详情 / 语言` 等稳定输入缓存，Composer 草稿、滚动活跃态、配置面板开关或复制状态变化不得触发整段输出重新解析 Markdown。
- Agent `process` 事件到达后，前端需在 `done` 前实时更新当前助手消息的步骤面板，而不是等待最终正文收口后一次性生成过程展示。
- Agent 消息中的 `Process` 优先使用服务端返回的结构化 `process_steps` 渲染；仅对缺失结构化步骤的历史消息保留文本解析兼容。
- 结构化 `process_steps` 需要在 SSE `done`、Task 结果回填与会话历史恢复后保持一致，刷新页面不得把已完成消息重新退化为仅正文展示。
- `Chat / Agent Runtime / Terminal` 的消息输出结构统一收敛到轻量 IM 式消息流：用户输入右对齐并使用浅灰低对比紧凑气泡，气泡高度需由较小纵向 padding 与独立消息行高控制，助手回复左对齐并弱化为无边框正文阅读流，Agent 与 Terminal 中间步骤默认按 `Thinking / 已思考` 轻量披露行展示，展开后在当前消息内进入步骤详情，移动端也保持同页内联展开；Chat / Agent Runtime / Terminal 助手最终答复统一使用稳定的运行页 markdown shell，正文先于复制工具栏渲染，复制动作位于正文下方，代码块独立呈现为浅灰内容块；消息正文区不显示逐条时间，仅在进行中、排队、失败等非稳定状态下保留状态标签。新增运行页若呈现用户输入与助手输出，必须复用 `RuntimeTimeline` 与 `runtime-message / runtime-message-user / runtime-message-assistant / runtime-message-bubble` 契约，避免继续产生页面私有气泡格式。
- 长会话默认只渲染最新一批消息；当顶部仍存在更早历史时，消息区需展示 `Load earlier messages / 加载更早消息` 入口，并在滚到顶部时自动按批次扩展更早消息。扩展历史时需保持当前阅读位置，不得强制跳回底部。
- `Process` 步骤标题与正文在桌面和移动端都必须保持整列阅读宽度；步骤序号、展开图标、标题与状态信息需在同一行垂直居中；长中文说明、路径、命令片段与 Markdown 文本优先在当前消息容器内自然换行，不得在真机窄屏下塌缩成逐字竖排窄列。
- Conversation 展示层必须在渲染 `process_steps.title/detail` 与最终 markdown 前移除零宽断行字符，并对“每字一行”的病态段落做可读性归一化；该修正同时适用于流式消息、Task 回填和历史会话恢复。
- Agent Runtime 与 Terminal 的最终 Markdown 输出不得复用需要额外 CSS 强制补丁的旧 shell 结构；其正文 DOM 必须保持普通静态文本语义，不绑定 `touchstart / pointerdown` 选区脚本，不设置 `contenteditable / inputmode / tabindex`，不创建浮动复制层或假选中 class。复制按钮只读取组件闭包中的原始文本，不把长 payload 镜像到 DOM 属性。
- `Chat / Agent Runtime` 的消息时间线在内容较少时必须保持顶部收口；短用户消息、折叠后的 `Thinking / 已思考` 披露行、最终回复与对应状态标签继续贴近各自消息气泡排布，不得因为时间线容器满高拉伸而出现大块垂直空白。
- `Chat / Agent Runtime` 打开已有消息的会话、刷新恢复当前会话或切换到其他会话后，时间线初始视口必须落到最新消息所在底部；当前活动会话内发送新消息后，时间线必须随新增消息回到底部，使本轮用户消息与助手占位立即可见；若用户已经在同一会话内手动滚动阅读历史，后续流式 patch、Process 展开状态变化和草稿输入不得强制把视口拉回底部。
- 助手最终回复提供一键复制；若消息含 Process，复制内容只包含最终正文。
- `Agent Runtime` 需要把专项 Agent 的 deliverables contract 作为一等运行信息展示在 `Details` 中，而不是仅保留在系统 prompt 或隐式约定里；若交付物声明绑定了 Session Profile 实例属性键，前端需直接回显当前 URL/路径值。
- 前端所有绝对时间与时分标签统一按北京时间（`Asia/Shanghai`）渲染，并固定采用 24 小时制；浏览器本地时区不参与显示格式决策，控制台管理页中的额度重置、运行时间等管理时间戳也必须复用同一口径。
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
- React 托管页面的正文型内容统一复用运行页 Markdown 渲染器：Memory 长期/天级/强制/说明文档、Task 请求与结果、任务日志和产物摘要、Control 描述、Cron 输入、Agent 说明、Codex 运行时说明以及 Agent Runtime `Session Profile` 中的非等宽字段，都按 Markdown 正文渲染。ID、路径、密钥、配置值、时间戳和分支名等元数据保持纯文本或等宽展示，避免把机器标识误解析成富文本。
- Chat / Agent Runtime / Terminal final output 统一使用 `MessageMarkdownShell` 承载最终答复，解析规则、复制按钮、选择行为和 DOM 稳定性都由同一组件负责；相同 markdown 不得因父级无关重渲染反复写入 `innerHTML`，也不得依赖 Terminal 视图级 `user-select !important` 兜底。
- 原始 HTML 不直接透传。
- 长路径、超长单词、代码块和 diff 只允许在内容块内部横向滚动，不撑破外层消息容器。

### Process

- Agent action / observation 与 Terminal 执行细节在前端收敛为可折叠 Process，并统一在当前消息或 turn 内同页展开。
- 最终答复出现后，Process 默认折叠，阅读焦点回到正文。
- 用户手动展开或折叠后，在当前浏览器会话内保留状态。

### 布局

- 全站默认固定侧边栏；仅侧栏自身内容溢出时允许侧栏内部滚动。
- Chat 历史区支持折叠与展开，减少长对话阅读空间占用。
- Conversation workspace 的新会话入口在 `agent-runtime` 路由下切换为 Agent 会话语义，并随语言切换同步更新。
- Session 历史区的空态提示与列表可访问标签需按当前路由与语言即时切换文案；这些文案更新不得清空或重建 runtime 已注入的会话卡片节点。
- 左侧主导航中的会话列表需按最近时间分组为 `Today / Yesterday / Earlier`（中文对应 `今天 / 昨天 / 更早`），并与主导航 `menu` 复用同一套分组容器、hover、激活态视觉和桌面会话列宽；分组内条目保持主导航式紧凑信息结构，采用低噪音列表卡片关系：主体只保留标题并在可用宽度内单行截断，尾侧收纳 30px 方形更多操作按钮，不再额外挂出独立 footer、胶囊操作面、完整会话 id、时间、短 hash、Agent 标签或摘要字符串。
- Session 历史区的会话条目不展示 ready、failed、exited 或 interrupted 状态灯；只有处理中条目显示 loading，并为读屏输出当前忙碌状态文案。
- Conversation workspace 头部的标题、状态按钮、`Details` 标签页和新会话入口需按当前路由与语言即时切换文案；状态按钮同时反映当前活动会话派生状态，但可见层只显示信号，不再展示固定 `Ready` 或其他状态文案；该信号固定排在当前会话标题左侧，右侧工具区只保留 `Details` 入口；这些壳层文案更新不得覆盖当前会话标题或消息内容。
- `Chat / Agent Runtime` 的会话列表、工作区外壳、聊天滚动区和输入区需输出 `runtime-*` 主契约并保留必要的 `terminal-* + conversation-*` 兼容 class，确保两条运行页与 `Terminal` 共用同一工作台表面与细节皮肤，同时保留 `data-conversation-*` 钩子供样式和测试使用。
- `Chat / Agent Runtime` 首页 Composer 采用单一圆角助手输入面板：主 textarea 透明无内边框，工具栏与输入区处在同一白色 surface 内；`Chat` 工具栏不再显示 `Session` 会话设置按钮，只保留附件与发送等直接对话动作，`Agent Runtime` 工具栏左侧继续提供正方形低圆角的会话设置、附件与必要 meta，右侧收口深色发送按钮。会话设置入口使用四点网格图标，附件入口使用回形针图标，文字 label 仅保留给可访问语义；桌面端输入面板按主阅读宽度居中，移动端压缩输入高度、外层留白与提交按钮体量，同时维持足够横向留白，避免输入区压窄；PC 端上传、发送、状态、详情、流程入口与弹窗动作统一采用 8-14px 低圆角矩形，不使用胶囊按钮或胶囊标签；会话卡片与 `Details` 面板保持同一浅色 runtime 质感，不再出现首页输入区厚重、旁侧组件过轻或材质不一致的情况；Agent Runtime 会话设置面板打开后，点击面板外任意区域都会立即关闭，点击主输入框时也需先收起面板再继续输入；Agent Runtime 移动端在输入框已聚焦时，首触 `Session` 入口就必须直接展开面板。空态工作区需使用低对比网格与细弧线背景，并锁定为不可滚动表面，不允许通过空白区域拖拽把头部和输入区顶出可视区。
- `Chat / Agent Runtime` 在页面重新变为前台可见或浏览器重新把当前页激活时，必须复用运行页共享的 page-activation 补偿刷新链路：会话列表、当前活动会话详情与 pending task 状态都要立即回源；若当前页为 `Agent Runtime`，还需同时刷新当前 Agent 的 `Session Profile`，避免 `Details` 中的实例属性停留在后台前的旧值。
- `Chat / Agent Runtime` Composer 支持最多 5 张图片附件；附件可通过附件按钮选择，也可在 PC 输入框内直接粘贴剪贴板图片。粘贴图片时仅拦截图片文件并进入附件草稿，普通文本粘贴继续保持 textarea 原生行为。附件在输入区以缩略图展示，可单张预览和移除，并按会话草稿持久化。缩略条继续使用预览图，但单张预览弹层必须优先显示原图。当前选中的模型若未声明视觉能力，带图发送必须直接阻止并提示切换模型。
- 移动端 `Chat / Agent Runtime` 的左侧主导航抽屉与主工作区在 `1100px` 及以下需回落为静态表面，不保留模糊玻璃层或持续背景动效；性能优先级高于装饰层，确保真机滚动、抽屉开关和输入框聚焦不出现明显卡顿。
- 根工作台仅在窄屏时使用主导航抽屉；运行页会话列表由主导航统一承载，避免出现导航抽屉和会话浮层叠加。
- 路由页头部的标题与副标题需按当前路由与语言即时切换文案；这些页头更新不得覆盖 route body 内已渲染的页面主体内容。
- 已由 React 接管的工作台需在 DOM 上暴露稳定路由钩子：根壳层输出 `app-shell[data-workbench-route]`，运行页和控制页继续输出 `data-route / data-conversation-*` 标记；兼容层只能依据这些由 React 输出的钩子退让，不得继续维护独立白名单。
- 欢迎区与 Composer 面板在同一主工作区内采用主仓库式上下结构：欢迎区直接输出 `Alter0 workspace` tag、面向 repo / task / runtime 的默认标题与说明、target picker 与快捷提示，Composer 独立贴底；欢迎区内容超出可视高度时，输入区仍需稳定贴底，不得与欢迎区、消息区发生叠层覆盖。
- 用户消息右对齐并使用浅灰低对比紧凑气泡，`Chat / Agent Runtime / Terminal` 统一采用克制的冷灰工作台阅读主题；助手回复弱化厚重卡片层级，默认呈现为无边框正文阅读流，复制操作贴在正文下方，思考过程只保留一行内联可点披露入口，只展示步骤数量，不展示耗时，Process 详情和代码块只保留必要边界与有限强调色。
- `Chat / Agent Runtime / Terminal` 助手消息尾部默认不显示时间；仅当回复仍在生成、排队或失败时展示紧凑状态标签，不再为已完成消息重复展示 route/source/status 元信息。
- Chat、Agent Runtime 与 Terminal 工作区头部在进入会话态或桌面空态时收敛为共享单行标题区：只显示当前会话标题、状态按钮与 `Details` 入口，不再额外叠加 `Chat / Agent / Terminal` 标签以及模型、工具或目标摘要，Terminal 也不得为标题或 `Details` 按钮派生单独元素。模型、Agent、Deliverables、Tools / MCP、Skills 与会话元数据统一放入 `Details` 面板，具体内容由各运行页自行实现，面板首屏先使用紧凑摘要栅格展示高频字段，再承接页内 tab 与配置区。Chat / Agent Runtime 的 model tab 除已启用 Provider 模型外，都需稳定展示一个可直接点击的 `Codex` chip；选中该项后，后续消息请求不再携带普通 `alter0.llm.provider_id / alter0.llm.model` 组合，而是显式写入 `alter0.execution.engine=codex`。Agent Runtime 的 Agent tab 不展示 `Alter0/main` 主助手，只展示专项内置 Agent 与用户管理 Agent；Deliverables tab 直接展示当前专项 Agent 的最终交付物契约，例如 `travel` 的 HTML 攻略、`coding` 的代码变更与验证结果；Skills tab 区分当前 Agent 私有 Skill 与公有 Skill，私有项固定启用且禁用取消交互，公有项按会话选择进入启用或可选列表。`Details > Session Profile` 在首次打开会话和每轮消息收口后都必须回源最新 profile，使 `city / district / days / hotel_area` 这类实例属性自动回显到当前会话详情。`Details` 需以顶层浮层方式覆盖在工作区上方，内部独立滚动，浮层尺寸保持克制，并始终具备明确可见的 dialog 层级；页内 tab/按钮支持再次点击只收起当前 tab 内容区并保留摘要与 `Details` 面板，点击浮层外区域或按 `Escape` 才关闭整个面板，打开时不得推动消息列表、输入区或对话正文重新布局。
- 桌面宽屏下 Chat 消息列与 Composer 按主工作区宽度自适应放宽，并保持统一居中；正文区统一保留 `960px` 最大阅读宽度，但外层工作台也必须同步收缩导航与间距，避免在中等桌面宽度下出现阅读区限宽而整体布局仍然拥挤、遮挡或越界。
- Web Shell 主导航需根据 URL hash 即时同步当前路由高亮；导航折叠与语言切换更新不得导致会话卡片、消息节点或 route 内容被清空重建。
- React 壳层发出的主导航跳转、新建会话、欢迎区快捷提示、语言切换、导航折叠同步与会话历史折叠同步事件，必须由当前前端运行时在同一页面内完成确认、路由更新、快捷发送或会话创建，且不能要求用户重复点击或依赖额外脚本注入的全局函数。
- `Chat / Agent Runtime / Terminal` 提供统一的右侧箭头四键阅读定位条 `回到顶部 / 上一条 / 下一条 / 回到底部`：滚动超过阈值后显示顶部与底部入口，上一条与下一条按钮按当前可见消息块或 Terminal turn 实时重算目标；内容折叠、展开或重排后，按钮显隐与目标需同步更新。`回到底部` 只在最后一条内容的底边仍位于视口外时显示；若最后只剩空白或底部 padding，不得继续显示伪底部跳转。移动端四键定位条固定停靠在工作区右侧、输入区上沿之上，四个按钮统一为独立圆形触达面，不得退回正文流内或压住底部输入区；当前消息滚动容器一旦存在有效文本选区，四键需立刻隐藏并释放命中区，待选区清空后再恢复。Terminal 输出正文、Agent Runtime 最终 Markdown 正文和代码结果必须保持可选中文本语义，正文区域允许浏览器原生拖选、长按选中与复制；移动端最终输出不得安装脚本长按选区、假选中态、浮动复制层、编辑态兜底或视图级强制选择补丁，避免覆盖浏览器原生复制菜单。
- 上述阅读定位条必须作为消息区 overlay 渲染，不参与 `.runtime-timeline` 或 `terminal-chat-screen` 的正文高度计算；空白会话、少量消息和短 turn 场景下，消息区不得因为按钮组自身占位出现额外滚动条或被拉出超出可视区的空白高度。

## 移动端体验

### 输入与键盘

- Chat/Agent 输入区基于 `VisualViewport` 同步有效视口高度。
- 移动端 App Shell 高度同步 `VisualViewport` 驱动的 `--mobile-viewport-height`，避免浏览器工具栏状态切换造成底部留白或内容裁切。
- `Chat / Agent Runtime / Terminal` 的移动端会话列表共用左侧主导航抽屉：运行页顶部只保留 `Menu` 抽屉入口，并在抽屉中直接展示主工作流入口与当前运行页会话列表；点击遮罩、切换路由、切换会话或新建会话后，不保留旧的抽屉展开态。
- `Chat / Agent Runtime` 的移动端左侧抽屉在真机上优先保证稳定性：遮罩保留淡入淡出，抽屉本体仅保留一层轻量侧滑，不再叠加多层位移、条目级顺序动画或生硬的整板平推过渡。
- 输入区在软键盘弹起、收起、浏览器工具栏伸缩时持续贴住可见底部。
- 仅在输入框实际聚焦且软键盘占位达到阈值时追加键盘底部偏移。
- 键盘收起或视口回弹后不保留额外底部空白。
- `Chat / Agent Runtime / Terminal` 在页面恢复前台可见、浏览器重新激活当前标签页或系统恢复当前 WebView 时，必须立刻重算共享 `--mobile-viewport-height` 与 `--keyboard-offset`；第一帧不得沿用后台前遗留的旧视口高度、旧键盘偏移或旧底部遮挡量。
- `Chat / Agent Runtime` 首次触摸输入框时需采用与 `Terminal` 相同的 `preventScroll` 聚焦策略，并在聚焦期间监听 `window.scroll + VisualViewport resize/scroll` 把页面锚定在 `scrollY = 0`；首次弹出软键盘时公共操作行不得消失，也不得出现整页尺寸跳变。
- `Chat / Agent Runtime` 在移动端触摸发送按钮时，必须先 blur 当前主输入框，再继续原有发送链路；键盘收起期间 composer 继续按 `VisualViewport` 的真实回弹过程逐步释放 `--keyboard-offset`，不能在发送后继续维持聚焦态或把输入区悬停在空白带上。
- `Chat / Agent Runtime` 的 fixed composer 在移动端必须把实际遮挡高度同步回工作区滚动面；`.conversation-chat-screen` 与空态欢迎区都需停在 composer 上沿，不能依赖静态底部 padding 估算占位，也不能出现底部输入框覆盖最后一段消息或说明文案。
- `Chat / Agent Runtime` 在键盘收起和 composer 回弹到底边时，工作区滚动面也必须同步清理旧的遮挡高度；最后一屏消息、空态说明和阅读定位控件都不能在底边留下额外空白或残留占位。
- `Chat / Agent Runtime` 在移动端软键盘弹起期间，fixed composer 必须继续占据运行页最高交互层级；阅读定位按钮在主输入框聚焦后需主动隐藏，待输入框失焦、键盘收起后再恢复，不得压到输入框、附件条或键盘上方。
- `Chat / Agent Runtime` 的主输入框在移动端必须按普通命令文本输入处理：关闭系统自动填充、卡片、地址与密码类输入辅助条，避免键盘上沿再挂出额外输入助手并露出底部残留页面层。
- `Chat / Agent Runtime / Terminal` 的移动端主输入框需显式保持 16px 及以上可编辑文本字号；重新打开浏览器后首次聚焦输入法时，页面不得因 iOS Safari 自动输入框缩放而出现横向裁切、整体放大或分辨率突变。
- `Chat / Agent Runtime` 在移动端键盘弹起和收回期间，仅允许 fixed composer 自身跟随 `VisualViewport` 做贴底位移；顶部操作行与紧凑 workspace header 保持原位，不跟随键盘做额外动画或跳变；阅读定位按钮在输入框聚焦期间隐藏。
- `Chat / Agent Runtime` 的移动端发送按钮支持在键盘保持打开时直接点按提交；首触发送需覆盖 `pointerdown(touch)` 与 `touchstart` 提交链路，并在同一次触摸内去重，立即进入当前 `sendPrompt` 链路，不需要先收键盘或补第二次点击。
- `Chat / Agent Runtime` 的 fixed composer 不额外叠加 `bottom` 过渡动画；键盘回弹与输入区回贴底边时只消费 `VisualViewport` 的实时位置，避免补间动画与视口收缩/回弹叠加造成拖滞。
- `Chat / Agent Runtime` 在输入框失焦后，若 `VisualViewport` 仍处于收缩态，必须继续保留当前键盘偏移并随视口恢复逐步释放；不允许先把 composer 闪回到底边，再被后续 viewport resize 顶回去。
- `Chat / Agent Runtime / Terminal` 在输入框保持聚焦且软键盘占位已建立后，需容忍浏览器键盘动画中的短暂完整高度回报；该抖动事件不得立刻清空键盘偏移或让 composer 闪回底边。
- `Chat / Agent Runtime / Terminal` 的四键阅读定位条统一使用同一套圆形按钮样式和触摸反馈，避免不同运行页在跳转控件上分叉出独立实现。
- `760px` 及以下的真手机宽度下，主导航抽屉、会话列表区、头部按钮高度与间距继续压缩，避免头部按钮挤占可用阅读高度。
- 小高度窄屏下，主导航抽屉仍需保留稳定的触摸滚动链：菜单内容滚动不把整个页面带离当前上下文，抽屉底部固定区域与菜单滚动区域边界清晰。
- `1100px` 及以下时，`Chat / Agent Runtime / Terminal` 顶部固定提供 `Menu / New` 操作行；`Menu` 打开同一个左侧主导航抽屉，`New` 触发当前路由的新会话创建动作，操作入口不得因空态、已有消息或 `Details` 状态而消失。
- `760px` 及以下时，欢迎区 tag、标题与描述的顶部节奏需继续压缩；普通 `page-mode` 路由页内容区与 `Terminal` 工作区也需沿用同一贴顶节奏，避免不同页面在窄屏下出现明显不一致的顶部留白。

### 会话设置

- 移动端发送按钮与会话设置入口在 Agent Runtime 同排；Chat 不再显示会话设置入口。
- 移动端会话列表不再与正文上下堆叠，也不再使用运行页内部独立抽屉；三条运行页都通过 `Menu` 打开左侧主导航抽屉。抽屉内的会话区左侧将会话标题与会话总数收敛为上下两行，右侧保留 `New` 入口并复用运行页紧凑按钮规格；列表项沿用标题-only 卡片与尾侧更多按钮结构，处理中会话显示 loading，并支持遮罩点击收起。
- 左侧主导航内的会话条目统一采用工作台列表项语义：列表先按最近时间分组，再在条目内展示标题与尾侧删除动作；列表容器需保留独立滚动能力并输出稳定 `role="list"` 语义，视觉层级保持克制，不使用多余胶囊装饰。
- 会话设置展开后采用独立固定底部面板，带遮罩、关闭入口与内部滚动区。
- 会话设置面板的关闭路径在桌面与移动端保持一致：关闭按钮、遮罩、面板外点击和主输入框点击都必须走同一条收口逻辑，不保留“点输入框但面板仍悬停”的状态。
- Agent Runtime 移动端在主输入框与会话设置底部面板之间切换时，前端必须保证底部只保留一种主交互层：打开设置前先释放输入焦点并清理键盘占位，且首触 `Session` 入口就应直接展开面板；回到主输入框时先自动收起设置面板，再处理软键盘跟随与输入框贴底。Chat 不再提供该 `Session` 入口。
- 连续勾选 Skill、Tool、MCP 时，当前滚动位置保持稳定，不回到顶部。
- 设置面板标题、说明与标签在窄屏下保持可读，不重叠。

### 低功耗刷新

- 页面隐藏时停止高频扫描，恢复前台后补一次刷新。
- 输入聚焦、滚动活跃或移动端软键盘场景下降低非必要轮询与重绘。

## 依赖与边界

- Runtime 提供消息路由与流式事件。
- Agent 提供 Process 结构与最终答复。
- Task 提供任务卡片、状态回填和日志入口。
- Terminal 的会话式终端交互由 Task, Terminal & Workspace 领域维护；用户 prompt、最终输出与本领域运行页共享同一 `runtime-message-*` 消息格式。

## 验收口径

- Chat 默认进入 `main` Agent，Agent 页面不混入独立入口 Agent 历史。
- 新建空白会话不重复。
- SSE 断连后后端 Agent 执行仍完成并写入历史。
- 页面恢复后，本地缓存中的旧消息不得长期显示 `In Progress`。
- 移动端软键盘弹起与收起后输入区贴底，无回顶和残留空白。
- 长会话流式增量不触发整段消息列表重建。
- `Chat / Agent Runtime / Terminal` 的箭头四键阅读定位条可在滚动后稳定出现，并能把阅读位置跳到当前视口相邻的上一条或下一条内容。真机窄屏下，四键需保持固定右侧停靠、位于输入区上沿之上，且每个按钮保持圆形。
