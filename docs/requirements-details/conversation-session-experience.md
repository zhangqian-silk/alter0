# Conversation & Session Experience Requirements

> Last update: 2026-04-13

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
- `/chat` 提供 Chat、Agent、Terminal、Product、Control 与 Memory 的统一 Web Shell。
- Web Shell 的前端构建源位于 `internal/interfaces/web/frontend`，`/chat` 固定分发 `static/dist/index.html`；该入口仅保留前端挂载容器与静态资源引用，当前由 React 渲染 legacy shell DOM，再由兼容桥承接旧版运行时脚本与样式。
- React 壳层当前直接维护主导航当前路由高亮、导航折叠态、导航 tooltip、品牌状态卡、Session Pane 上下文卡、会话卡片列表、ChatWorkspace 头部动作区、工作区 hero、欢迎区文案、欢迎区 target picker、prompt deck、欢迎区/消息区显隐、消息列表 DOM、运行时 controls/note/sheet DOM、Session 历史空态提示/可访问标签、Composer 面板、路由页 hero、路由页头部标题/副标题、菜单、会话入口与路由感知文案；导航抽屉、会话抽屉、导航折叠同步、会话历史折叠同步、主导航跳转、新建会话入口、欢迎区快捷提示、会话聚焦、会话删除与语言切换统一由 React 发出结构化 bridge 事件，再由 legacy runtime 执行确认、路由、快捷发送和会话业务；legacy runtime 通过 chat workspace snapshot bridge 回写当前会话标题、副标题、欢迎区描述与结构化 target picker 数据，通过 session pane snapshot bridge 回写会话卡片列表、空态与加载错误，通过 message region snapshot bridge 回写欢迎区/消息区显隐与结构化消息快照，通过 chat runtime snapshot bridge 回写运行时 controls、错误提示、移动端 runtime sheet 与滚动位置；`agent / terminal / products / memory / channels / skills / mcp / models / environments / cron-jobs / sessions / tasks` 十二类路由页由 React 接管，其中 terminal 通过 React-managed host 挂载 legacy terminal controller，其余页面直接请求控制台或会话 API 并渲染，且 React 必须在 `routeBody` 上同步当前页 `data-react-managed-route` 标记、在 `appShell` 上同步稳定的 `data-react-managed-routes` 路由清单，供 legacy runtime 精确跳过这些页面主体 DOM 的接管；其余 route body 保持静态 DOM 契约，不在桥接阶段随 React 状态重复渲染。
- Web Shell 的稳定界面基线为桌面三栏 cockpit 与移动端双抽屉工作台：主导航展示品牌状态卡与模块组，会话栏承载当前路由上下文、新建入口与历史列表，主工作区承载工作区 hero、欢迎区/消息区和独立输入面板；页面类路由使用独立 route hero 承接说明，不再沿用旧版平铺式工具页头。
- `static/dist/assets/*` 使用构建产物哈希文件名并返回长期 immutable 缓存；`/chat` 与 `static/dist/legacy/*` 保持 `no-cache`，确保桥接阶段页面与兼容 runtime 能及时刷新到最新版本。
- `/login` 在登录密码启用时提供登录入口；`/logout` 清理当前登录态并回到登录流程。
- 登录密码未启用时，Web Shell 直接进入受保护页面；登录密码启用后，受保护页面和 API 使用同一登录态校验。

### Chat

- `Chat` 默认绑定内置 `main` Agent `Alter0`。
- `Chat` 面向通用对话入口，并允许在需要时调度专项 Agent 或 Product 总 Agent。
- `Provider / Model`、`Tools / MCP`、`Skills` 可在会话过程中调整，并作用于后续发送的消息。

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
- `GET /api/agents` 返回可进入的运行时 Agent。
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
- 页面刷新或服务重启后，用户可恢复最近会话与历史消息。
- 删除会话时同步清理关联任务记录与会话工作区。
- `Agent` 运行页会话列表为每个会话展示 8 位短 hash 标识，并支持一键复制；短 hash 用于前端列表辨识、预览域名映射与人工排障引用。

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
- 原始 HTML 不直接透传。
- 长路径、超长单词、代码块和 diff 只允许在内容块内部横向滚动，不撑破外层消息容器。

### Process

- Agent action / observation 与 Terminal 执行细节在前端收敛为可折叠 Process。
- 最终答复出现后，Process 默认折叠，阅读焦点回到正文。
- 用户手动展开或折叠后，在当前浏览器会话内保留状态。

### 布局

- 全站默认固定侧边栏；仅侧栏自身内容溢出时允许侧栏内部滚动。
- Chat 历史区支持折叠与展开，减少长对话阅读空间占用。
- Session Pane 的历史折叠状态在同一浏览器会话内持久化恢复；在 `agent-runtime` 路由下，新会话入口文案切换为 Agent 会话语义，并随语言切换同步更新。
- Session 历史区的空态提示与列表可访问标签需按当前路由与语言即时切换文案；这些文案更新不得清空或重建 runtime 已注入的会话卡片节点。
- ChatWorkspace 头部的菜单按钮、会话抽屉按钮和移动端新会话入口需按当前路由与语言即时切换文案；这些壳层文案更新不得覆盖 legacy runtime 已写入的会话标题、副标题或消息内容。
- 路由页头部的标题与副标题需按当前路由与语言即时切换文案；这些页头更新不得覆盖 route body 内已由 legacy runtime 注入的页面主体内容。
- 已由 React 接管的 route body 需在 DOM 上暴露稳定 ownership 契约：`routeBody[data-react-managed-route]` 表达当前页 ownership，`appShell[data-react-managed-routes]` 表达稳定的受管路由集合；legacy runtime 只能依据这份由 React 输出的契约跳过页面主体挂载，不得继续维护额外的独立路由白名单。
- 欢迎区、prompt deck 与 Composer 面板在同一主工作区内采用独立滚动与固定底部输入区；欢迎区内容超出可视高度时，输入区仍需稳定贴底，不得与欢迎区、消息区发生叠层覆盖。
- 用户消息右对齐，宽度不超过消息区 80%，助手回复弱化厚重卡片层级。
- 桌面宽屏下 Chat 消息列与 Composer 按主工作区宽度自适应放宽，并保持统一居中；正文区保留最大阅读宽度，避免大屏下仍锁死为窄列。
- Web Shell 主导航需根据 URL hash 即时同步当前路由高亮；导航折叠与语言切换更新不得导致 legacy runtime 已注入的会话卡片、消息节点或 route 内容被清空重建。
- React 壳层发出的主导航跳转、新建会话、欢迎区快捷提示、语言切换、导航折叠同步与会话历史折叠同步事件，必须由 legacy runtime 在同一页面内完成确认、路由更新、快捷发送或会话创建，且不能要求用户重复点击或依赖不存在的全局函数。

## 移动端体验

### 输入与键盘

- Chat/Agent 输入区基于 `VisualViewport` 同步有效视口高度。
- 移动端 App Shell 高度同步 `VisualViewport` 驱动的 `--mobile-viewport-height`，避免浏览器工具栏状态切换造成底部留白或内容裁切。
- 输入区在软键盘弹起、收起、浏览器工具栏伸缩时持续贴住可见底部。
- 仅在输入框实际聚焦且软键盘占位达到阈值时追加键盘底部偏移。
- 键盘收起或视口回弹后不保留额外底部空白。

### 会话设置

- 移动端发送按钮与会话设置入口同排。
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
