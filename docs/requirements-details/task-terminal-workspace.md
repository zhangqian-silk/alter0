# Task, Terminal & Workspace Requirements

> Last update: 2026-04-26

## 领域边界

Task, Terminal & Workspace 负责后台异步任务、任务观测、日志流、产物交付、会话式终端代理和执行工作区隔离。它接收 Runtime、Conversation、Agent 的执行请求，并提供可追踪、可恢复、可回放的运行态。

## 核心对象

| 对象 | 职责 |
| --- | --- |
| `Task` | 后台任务主数据、状态、来源与执行配置 |
| `TaskSummary` | 可注入 Memory 的任务摘要 |
| `TaskLog` | 可流式读取和回补的任务日志 |
| `ArtifactRef` | Web 可访问产物引用 |
| `TerminalSession` | Terminal 会话身份、标题、状态、工作区和 Codex 线程 |
| `TerminalTurn` / `TerminalStep` | Terminal 执行轮次与步骤明细 |
| `Workspace` | Chat、Agent、Task、Terminal 的默认执行目录 |
| `RuntimeHeartbeat` | 长任务存活心跳与超时续租窗口 |

## 异步任务

### 分流

- 高复杂度、长耗时或产物型请求可转为后台 Task。
- 请求被接受后先返回任务卡片，包含任务目标、任务 ID、简要计划与详情入口。
- 异步任务使用独立后台执行池，不占用主会话同步交流的串行执行槽位。

### 并发与心跳

- 异步任务模块限制后台并发数量，当前稳定上限为最多 5 个任务并发执行。
- Codex CLI 长任务执行期间每 1 分钟记录一次心跳。
- 心跳用于续租任务超时窗口，避免仍健康运行的长任务被固定短超时误杀。
- 列表卡片与详情展示最近心跳和当前窗口截止时间。

### 会话映射

- Task 必须记录 `session_id`、`source_message_id`、`channel_type`、`channel_id`、`trigger_type`、`correlation_id`。
- Cron 触发任务补充 `job_id`、`job_name`、`fired_at`。
- 删除会话时同步清理关联任务记录与任务产物，避免孤儿数据。

## 任务观测

### 任务视图

- Control 页面提供 `Tasks` 入口。
- 列表支持状态、类型、时间、会话、触发类型、通道、来源消息与结果消息等筛选。
- 桌面端默认采用左侧任务列表 + 右侧详情面板的主从布局；任务列表保持紧凑摘要，右侧详情区展示基础信息、来源字段、运行状态、心跳、日志、产物、控制动作和会话回链。

### 日志

- 任务日志支持 SSE 流式观测。
- 日志流支持游标断点续读与回补查询。
- 普通任务与 Control 任务日志均支持游标查询；Control 任务详情页可通过日志流接口持续观察后台执行。
- 高频日志到达时，前端按浏览器逐帧节奏合并刷新，避免每条日志事件都同步整块重绘。
- 日志不可用时需给出重建或不可恢复提示。

### 控制动作

- Task 支持 `retry` 与 `cancel`。
- Control 任务详情在可交互 Terminal 任务上支持追加输入，追加输入以 follow-up Task 形式进入后台执行，并保留原任务、锚点任务和 Terminal 会话元数据。
- Task 详情抽屉中的 follow-up terminal 输入支持最多 5 张图片附件、缩略图预览、移除和仅图片发送；附件需作为统一消息元数据随 follow-up Task 一起进入后台执行。
- 任务详情与会话详情支持双向跳转。
- 任务完成后向聊天区回写精简摘要；完整输出保留在任务详情、日志与产物中。

## 产物交付

- Web 会话不直接向用户返回本地文件路径。
- 文件、页面、截图、构建结果等交付物必须通过 `ArtifactRef`、下载接口或预览接口访问。
- 产物列表响应不得泄露本地 `file://` 或宿主绝对路径；下载接口以附件形式交付，预览接口只开放支持安全预览的内容类型。
- 产物引用需保留任务、会话与来源消息回链。
- 删除会话或任务时，相关产物按留存策略清理。

## 任务记忆视图

- `Agent -> Memory` 的任务历史面板复用 Task 领域数据，支持按状态、类型、时间与分页查询任务摘要。
- 任务历史列表默认以高密度表格展示，任务详情通过侧栏读取摘要、来源字段、状态、时间戳、日志入口和产物入口。
- 任务日志下钻支持游标分页读取，缺失日志时返回稳定错误并保留任务详情可读。
- 任务摘要缺失或需要刷新时，支持对单个任务触发摘要重建。
- Memory 任务视图只提供历史观测和摘要维护，不承担任务执行、重试或取消控制。

## 接口边界

### Task

- `POST /api/tasks` 创建异步任务。
- `GET /api/tasks` 查询任务列表。
- `GET /api/tasks/{task_id}` 查询任务详情。
- `POST /api/tasks/{task_id}/cancel` 取消任务。
- `POST /api/tasks/{task_id}/retry` 重试任务。
- `GET /api/tasks/{task_id}/logs` 游标读取任务日志。
- `GET /api/tasks/{task_id}/artifacts` 查询任务产物。
- `GET /api/tasks/{task_id}/artifacts/{artifact_id}/download` 下载任务产物。
- `GET /api/tasks/{task_id}/artifacts/{artifact_id}/preview` 预览支持的任务产物。
- `GET /api/sessions/{session_id}/tasks` 查询会话关联任务，支持 `message_id` 与 `latest` 过滤。

### Control Task

- `GET /api/control/tasks` 查询 Control 任务列表，支持来源、状态、时间和分页过滤。
- `GET /api/control/tasks/{task_id}` 查询 Control 任务详情。
- `GET /api/control/tasks/{task_id}/logs` 游标读取 Control 任务日志。
- `GET /api/control/tasks/{task_id}/logs/stream` 流式读取 Control 任务日志。
- `POST /api/control/tasks/{task_id}/retry` 重试 Control 任务。
- `POST /api/control/tasks/{task_id}/terminal/input` 为交互式任务追加输入并创建 follow-up Task；请求体除 `input` 外还接受 `attachments[]` 图片载荷，允许仅发送图片。

### Memory Task

- `GET /api/memory/tasks` 查询任务记忆摘要列表。
- `GET /api/memory/tasks/{task_id}` 查询任务记忆详情。
- `GET /api/memory/tasks/{task_id}/logs` 下钻任务日志。
- `GET /api/memory/tasks/{task_id}/artifacts` 下钻任务产物。
- `POST /api/memory/tasks/{task_id}/rebuild-summary` 重建任务摘要。

### Terminal

- `POST /api/terminal/sessions` 创建 Terminal 会话。
- `GET /api/terminal/sessions` 查询 Terminal 会话列表。
- `POST /api/terminal/sessions/recover` 恢复已持久化 Terminal 会话。
- `GET /api/terminal/sessions/{session_id}` 查询 Terminal 会话详情与 turn 摘要。
- `POST /api/terminal/sessions/{session_id}/input` 发送 Terminal 输入；请求体除 `input` 外还接受 `attachments[]` 附件载荷和 `skill_ids[]` 公有 Skill 选择。当前稳定支持图片与常见文本/文档文件，允许仅发送附件。

### Terminal Web Shell

- Web Shell 中的 Terminal 路由页主体由 React 原生实现，运行区根节点直接挂在共享 `workbench-pane-shell` 下，不再额外经过 `route-view / route-body` 包裹，避免从 Chat/Agent Runtime 切换时出现布局与滚动容器跳变。
- Terminal 页面直接请求 `/api/terminal/sessions`、`/api/terminal/sessions/{session_id}`、`/api/terminal/sessions/{session_id}/turns/{turn_id}/steps/{step_id}` 等接口，并在 React 内维护会话恢复、轮询、输入、删除、step 展开、滚动定位与本地草稿恢复。
- Terminal 的 session pane 容器、workspace 容器与主视图外壳在 React rerender 期间必须保持稳定实例，不能因语言切换、hash 路由变化或壳层状态更新而清空正在运行的终端内容。
- Terminal 运行页需挂载在共享 runtime workspace framework 上：统一复用会话侧栏、workspace body、slot 化头部/正文/底部区域与 backdrop 结构，同时允许 Terminal 继续注入自身的状态按钮、详情面板、Process、跳转四键与 Composer 控件；详情面板首屏先复用共享紧凑摘要栅格，再承接终端会话专属字段。
- React 版 Terminal 允许复用旧版 `terminal-*` DOM class 与布局关系作为视觉基线，但会话栏、工作区头部、详情面板、Process、输出渲染和 Composer 必须继续由 React state 驱动，不恢复 legacy runtime 脚本接管。
- 移动端 Terminal Composer 在输入框聚焦且软键盘抬起后，必须按 `VisualViewport` 推导的键盘偏移直接上移到可见底边；长历史输出继续由 `terminal-chat-screen` 独立滚动，不允许通过增加 footer padding 或让 workspace 改走外层滚动把输入区挤出视口。
- `DELETE /api/terminal/sessions/{session_id}` 删除 Terminal 会话与工作区。
- `GET /api/terminal/sessions/{session_id}/turns/{turn_id}/steps/{step_id}` 查询 Terminal step 明细。

## 工作区

### 默认目录

- Chat / Agent：`.alter0/workspaces/sessions/<session_id>`。
- Agent 仓库类执行：`.alter0/workspaces/sessions/<session_id>/repo`。
- Async Task：`.alter0/workspaces/sessions/<session_id>/tasks/<task_id>`。
- Terminal：`.alter0/workspaces/terminal/sessions/<terminal_session_id>`。

### 权限边界

- 工作区目录决定默认执行目录和运行时产物落点。
- 工作区隔离不等同于文件系统权限沙箱；可访问范围仍取决于宿主权限和运行账户。
- 删除会话或 Terminal 会话时，对应工作区同步清理。

## Terminal

### 会话模型

- Terminal 是独立模块，支持在模块内以类 Chat 方式持续对话。
- 每个 Terminal 会话持久化 Codex CLI 线程标识、标题、工作区、创建时间、最近活动时间、会话状态、输出日志与步骤索引。
- Terminal 历史在同一 Web 登录态下跨设备共享，不按浏览器 client 标识分桶。
- Terminal 不设置产品级会话数量上限或固定超时淘汰策略。
- 会话详情响应包含已持久化 turn 摘要与用户图片附件摘要；单个 turn/step 明细可按会话、turn 与 step 标识读取。

### 状态模型

- 会话态为 `ready / busy / exited / interrupted`。
- turn / step 维度继续使用 `running / completed / failed / interrupted`。
- 历史 `running / starting` 会话值加载时兼容归一到 `ready / busy`。
- 运行态退出或中断只改变当前运行状态，不删除会话身份、历史和线程标识。

### 恢复

- 对已退出或中断的 Terminal 会话继续发送输入时，系统优先复用已持久化 Codex CLI 线程继续执行。
- 若 Codex CLI 在线程续写阶段返回远端 compact 失败，系统保留原 Terminal 会话历史与独立工作区，清空失效线程标识，并在下一次输入时自动启动新的 Codex CLI 线程。
- 无法直连恢复时，保留原会话历史，并允许在同一 Terminal 会话下重新建立运行态。
- 新建但未发送首条输入的会话若底层运行态缺失，首次发送需自动恢复同一 `terminal_session_id` 并重试当前输入。
- 同一状态周期内只写入一条退出或中断提醒；恢复发送后旧提示立即清空。

### 生命周期动作

- `Create` 创建会话并返回服务端会话身份、初始状态与最近输出时间。
- `Recover` 接收已持久化会话身份与 Codex CLI 线程标识，用于在页面或运行态恢复时重建服务端会话视图。
- `Input` 向指定 Terminal 会话追加用户输入，执行完成前会话进入 `busy`；输入支持文本、附件与公有 Skill 选择混合载荷。图片附件继续以视觉输入语义交给 Codex CLI，普通文件在执行前落到当前 Terminal 工作区 `input-attachments/<turn_id>/` 并通过 prompt 注入稳定读取路径；纯附件输入由服务端补齐稳定占位文本；`skill_ids[]` 仅接收控制面中启用且非 `agent-private/private` 的 Skill，并编译为当前 Terminal 工作区 `.alter0/codex-runtime/skills.md`。
- `Close` 退出当前运行态，保留会话记录、历史、线程标识和工作区。
- `Delete` 删除 Terminal 会话主记录、状态文件和独立工作区。
- 删除成功后接口返回 `204 No Content`，前端立即移除会话，不依赖改写后的状态对象。

### 交互与渲染

- Terminal 新会话先使用占位标题，早期多轮内可按更具体输入升级标题。
- 输入区聚焦期间，轮询刷新不得销毁输入框、焦点、草稿和滚动位置。
- Terminal 页面输入条支持通用附件体验：输入条采用“上层输入区 + 下层工具栏”的双层结构，工具栏左侧提供正方形低圆角的会话设置与回形针附件入口，右侧收口发送动作；输入区需保持足够横向留白，避免长命令或多段追问输入时出现压窄观感；选择图片后立即显示缩略图并支持点击预览，选择普通文件后显示文件条目与移除动作；发送后，图片会继续在历史区回显该轮缩略图，普通文件至少保留发送侧条目与执行侧 workspace 路径语义。
- Terminal `Details` 面板在摘要字段后提供公有 Skill 选择区；勾选项作用于后续输入，不展示或允许取消 Agent 私有 Skill。
- 移动端输入法候选确认后，只要输入框仍聚焦，页面不得立即重绘回顶。
- Terminal 最终输出按 Chat 助手消息样式渲染，并继续使用浅蓝轻科幻主题下的冷色阅读容器与一键复制入口。
- 同一轮最终输出出现后自动折叠对应 Process。
- Markdown 链接按链接文本渲染，不直接暴露冗长 Markdown 源码或长路径。
- Terminal 发送按钮在首次点击时必须立即切到 pending 反馈；若当前尚未存在 active session，前端先创建 Terminal 会话再发送输入，但按钮和可访问名称需在会话创建阶段就进入 `Sending...` 禁用态，避免用户误判首击无效并重复提交。
- Terminal 会话栏、工作区、输入区、跳转控件与 Process 区统一采用浅蓝渐变背景、低对比玻璃感面板和冷色高亮，确保与 Chat / Agent 的整体视觉语言一致。
- Terminal 桌面端维持旧版 master-detail 布局关系：左侧会话列表复用共享列表项与共享运行页会话列宽，承载当前态、标题、最近输出、由会话 id 派生的 8 位短标识与删除入口，列表底部不展示完整 `terminal_session_id`；右侧工作区头部收敛为标题、状态按钮与 `Details` 工具栏，运行状态不在列表项内额外渲染独立徽标；Terminal route body 顶部不再额外挂载页面级说明 hero。
- Terminal React 版继续保留 `terminal-*` DOM class 与数据钩子，但会话栏、工作区容器、工作区头部和窄屏顶部 `Menu / Sessions / New` 操作行需与 Chat / Agent Runtime 复用同一套工作台表面语义，避免同属运行页却出现独立壳层节奏。
- `1100px` 及以下的窄屏 Terminal 页面中，会话抽屉入口统一由壳层头部 `Sessions` 按钮承接；Terminal 工作区头部不再重复渲染第二枚 `Sessions` 按钮，避免顶部操作区出现重复入口。
- Terminal 路由页沿用单层信息架构：页面级说明由工作区头部独占，不再在工作区上方叠加第二层 route hero 或重复简介。

### 移动端

- Terminal 移动端采用紧凑 Header、独立滚动消息区与底部输入条。
- Terminal 在 `1100px` 及以下窄屏下由 `ReactManagedTerminalRouteBody` 直接输出顶部 `Menu / Sessions / New` 操作行，分别打开主导航抽屉、会话抽屉与终端会话创建动作。
- Terminal 的 `Menu / Sessions` 抽屉在真机上优先保证稳定性；遮罩保留淡入淡出，抽屉本体仅保留一层轻量侧滑，不再叠加多层位移、条目级顺序动画或整块白板式平推动画。
- Terminal 窄屏消息阅读继续由 `terminal-chat-screen` 作为唯一纵向滚动容器；`workbench-main / chat-pane / terminal-view` 只负责提供满高和滚动隔离，不允许因高度未闭合或外层 `overflow: hidden` 造成消息页无法滑动。
- 工作区头部收敛为会话标题、状态按钮和紧凑工具栏；`Details` 面板默认先展示高密度摘要字段，并以顶层浮层方式覆盖在工作区上方，面板内部独立滚动，点击浮层外区域或按 `Escape` 可关闭，打开时不得推动输出区或会话正文重新排版；真手机宽度下允许标题与工具栏换行，`Details` 不得被长标题挤出屏幕。
- 长输出阅读取消消息与区块头部吸顶，保持自然文档流滚动。
- 右侧低圆角四键组支持回到顶部、上一条、下一条与回到底部。
- 浏览器底部工具栏伸缩、软键盘收起或视口回弹后，底部输入条立即回贴可见底边。
- 输入框聚焦时仅允许 Composer 自身按键盘偏移上移，不能把工作区整体撑高到可视视口之外；Terminal 主工作区的顶部位置与主体高度在键盘弹起期间保持稳定，长对话下输入框仍需保持可见、可聚焦、可提交。
- Terminal 移动端需把 fixed Composer 的真实遮挡高度同步回 `terminal-chat-screen`；无论是空态、长输出还是 Process 展开态，最后一屏内容都必须停在输入区上沿，不允许再被底部输入条覆盖。
- Terminal 移动端的 `Menu` 与 `Sessions` 抽屉共用同一份当前面板状态：从顶部操作行或工作区工具栏打开会话列表时，主导航抽屉必须立即收起；重新打开 `Menu` 时，会话列表也必须立即关闭，避免双层覆盖和残留展开态。
- Terminal 会话抽屉内的条目统一采用工作台列表项语义：头部展示当前态文本，正文展示标题、最近输出和 8 位短标识，删除入口固定在尾侧；列表容器保持独立滚动并输出稳定 `role="list"` 语义，视觉层级保持克制，不使用多余胶囊装饰；PC 端状态、详情、发送、上传、短标识与跳转控件统一使用低圆角矩形节奏。
- Terminal 在移动端键盘弹起和收回期间，除 Composer 外的公共控件都保持原位；工作区头部、状态区与右侧四键定位条不跟随键盘位移做额外动画。
- Terminal 的移动端发送按钮支持在软键盘保持打开时直接点按提交；首触发送立即进入 `submitInput`，不允许先触发键盘收起或焦点切换，再要求第二次点击。
- Terminal 的 fixed Composer 不再额外叠加 `bottom` 过渡动画；键盘收起与输入区回弹阶段直接按 `VisualViewport` 实时位置回贴底边，避免明显卡顿。
- Terminal 在输入框失焦后，若 `VisualViewport` 仍未恢复到最终高度，必须继续保留当前键盘偏移并随视口回弹逐步释放；不能先闪回到底边再被后续 resize 顶起。
- Terminal 移动端的 `terminal-jump-cluster` 只按静态 Composer footprint 停靠，不跟随键盘位移一起上移：四键按钮在键盘弹起时保持原位，键盘取消与视口回弹后再稳定回到 Composer 上沿之上，不允许落到 fixed 输入条下方或留下半截可见残影。
- Terminal 在输入框失焦、键盘收起和 Composer 回弹到底边的过渡阶段，`terminal-chat-screen` 与 `terminal-jump-cluster` 也必须同步释放旧的遮挡高度；页面底部不得残留上一轮键盘高度对应的空白带。

### 性能

- Terminal 会话列表与工作区详情按不同周期刷新。
- Terminal 轮询策略需按当前会话状态调整：`busy` 会话可继续刷新会话详情与列表，`ready` 会话仅保留低频列表刷新；用户正在滚动输出区时暂停当前会话明细刷新，避免阅读过程中出现卡顿或滚动中断。
- 页面隐藏、输入聚焦、滚动活跃期间自动降频。
- 滚动中的导航计算与位置测量合并到逐帧节奏，并复用稳定锚点缓存。
- 浏览器侧会话缓存写入避开活跃滚动窗口，在滚动停顿后持久化。

## 依赖与边界

- Conversation 负责普通 Chat/Agent 消息体验，Terminal 负责独立终端会话。
- Agent 通过 `codex_exec` 进入工作区，Task 承接后台化执行。

## 验收口径

- 复杂请求可转为 Task，任务卡片、日志、心跳、产物和会话回链完整。
- Web 不暴露本地文件路径。
- 删除会话清理对应任务和工作区。
- Terminal 会话跨设备可见，退出后继续发送可恢复。
- Terminal `Delete` 同步清理状态文件与独立工作区。
- 移动端 Terminal 输入、滚动和软键盘场景无回顶、无残留底部空白。
