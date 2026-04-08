# Task, Terminal & Workspace Requirements

> Last update: 2026-04-08

## 领域边界

Task, Terminal & Workspace 负责后台异步任务、任务观测、日志流、产物交付、会话式终端代理和执行工作区隔离。它接收 Runtime、Conversation、Agent、Product 的执行请求，并提供可追踪、可恢复、可回放的运行态。

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
- 列表支持状态、类型、时间等基础筛选。
- 详情抽屉展示基础信息、来源字段、运行状态、心跳、日志、产物和会话回链。

### 日志

- 任务日志支持 SSE 流式观测。
- 日志流支持游标断点续读与回补查询。
- 高频日志到达时，前端按浏览器逐帧节奏合并刷新，避免每条日志事件都同步整块重绘。
- 日志不可用时需给出重建或不可恢复提示。

### 控制动作

- Task 支持 `retry` 与 `cancel`。
- 任务详情与会话详情支持双向跳转。
- 任务完成后向聊天区回写精简摘要；完整输出保留在任务详情、日志与产物中。

## 产物交付

- Web 会话不直接向用户返回本地文件路径。
- 文件、页面、截图、构建结果等交付物必须通过 `ArtifactRef`、下载接口或预览接口访问。
- 产物引用需保留任务、会话与来源消息回链。
- 删除会话或任务时，相关产物按留存策略清理。

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

### 状态模型

- 会话态为 `ready / busy / exited / interrupted`。
- turn / step 维度继续使用 `running / completed / failed / interrupted`。
- 历史 `running / starting` 会话值加载时兼容归一到 `ready / busy`。
- 运行态退出或中断只改变当前运行状态，不删除会话身份、历史和线程标识。

### 恢复

- 对已退出或中断的 Terminal 会话继续发送输入时，系统优先复用已持久化 Codex CLI 线程继续执行。
- 无法直连恢复时，保留原会话历史，并允许在同一 Terminal 会话下重新建立运行态。
- 新建但未发送首条输入的会话若底层运行态缺失，首次发送需自动恢复同一 `terminal_session_id` 并重试当前输入。
- 同一状态周期内只写入一条退出或中断提醒；恢复发送后旧提示立即清空。

### 生命周期动作

- `Close` 退出当前运行态，保留会话记录、历史、线程标识和工作区。
- `Delete` 删除 Terminal 会话主记录、状态文件和独立工作区。
- 删除成功后接口返回 `204 No Content`，前端立即移除会话，不依赖改写后的状态对象。

### 交互与渲染

- Terminal 新会话先使用占位标题，早期多轮内可按更具体输入升级标题。
- 输入区聚焦期间，轮询刷新不得销毁输入框、焦点、草稿和滚动位置。
- 移动端输入法候选确认后，只要输入框仍聚焦，页面不得立即重绘回顶。
- Terminal 最终输出按 Chat 助手消息样式渲染，并提供一键复制。
- 同一轮最终输出出现后自动折叠对应 Process。
- Markdown 链接按链接文本渲染，不直接暴露冗长 Markdown 源码或长路径。

### 移动端

- Terminal 移动端采用紧凑 Header、独立滚动消息区与底部输入条。
- 顶部通用栏保留 `Menu`，并在右侧提供 `Sessions / New`。
- 工作区头部收敛为会话标题、状态点和紧凑工具栏。
- 长输出阅读取消消息与区块头部吸顶，保持自然文档流滚动。
- 右侧圆形四键组支持回到顶部、上一条、下一条与回到底部。
- 浏览器底部工具栏伸缩、软键盘收起或视口回弹后，底部输入条立即回贴可见底边。

### 性能

- Terminal 会话列表与工作区详情按不同周期刷新。
- 页面隐藏、输入聚焦、滚动活跃期间自动降频。
- 滚动中的导航计算与位置测量合并到逐帧节奏，并复用稳定锚点缓存。
- 浏览器侧会话缓存写入避开活跃滚动窗口，在滚动停顿后持久化。

## 依赖与边界

- Conversation 负责普通 Chat/Agent 消息体验，Terminal 负责独立终端会话。
- Agent 通过 `codex_exec` 进入工作区，Task 承接后台化执行。
- Product 详情页空间属于 Product 领域，产物引用和工作区交付由本领域提供底座。

## 验收口径

- 复杂请求可转为 Task，任务卡片、日志、心跳、产物和会话回链完整。
- Web 不暴露本地文件路径。
- 删除会话清理对应任务和工作区。
- Terminal 会话跨设备可见，退出后继续发送可恢复。
- Terminal `Delete` 同步清理状态文件与独立工作区。
- 移动端 Terminal 输入、滚动和软键盘场景无回顶、无残留底部空白。
