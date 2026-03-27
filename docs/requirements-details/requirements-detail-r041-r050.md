# Requirements Details (R-041 ~ R-050)

> Last update: 2026-03-27

## 需求细化（草案）

### R-041 任务执行前复杂度预判与同步/异步分流

1. 用户消息进入编排链路后、转发执行器实际执行前，必须先进行一次任务复杂度预判，输出至少包含：`estimated_duration_seconds`、`complexity_level`、`execution_mode`（`streaming` 或 `async`）。
2. 分流规则：当预估执行时长 `<= 30s` 时，任务走同步流式执行路径，执行结果以流式事件实时推送到对话会话。
3. 分流规则：当预估执行时长 `> 30s` 时，任务走异步任务路径，并在受理后立即向当前会话同步返回任务卡片，再由后台继续执行。
4. 异步任务卡片要求：卡片内容至少包含“当前任务较复杂，已创建后台任务执行，请稍后”、`task_id`、任务简要描述（`task_summary`），并提供可点击的任务详情跳转入口（`task_detail_url` 或等价路由信息）。
5. 流式执行路径要求：会话中需持续展示进行中状态与增量结果；执行完成后写入完整结果与终态。
6. 异步执行路径要求：创建任务后必须建立 `session_id` 与 `task_id` 映射，用户可在任务列表、任务详情页与会话历史中查看进度与最终结果。
7. 主动同步要求：异步任务进入终态（`success/failed/canceled`）时，系统必须向原会话主动推送结果通知，至少包含 `task_id`、终态、完成时间、结果摘要或失败原因。
8. 失败兜底：当复杂度预判不可用或超时时，默认走异步路径，并仍需返回任务卡片，避免阻塞会话主链路。
9. 验收：长任务请求可先收到可跳转任务卡片，任务完成后用户无需手动刷新即可在会话中收到主动同步结果；短任务仍保持实时流式返回。

#### 接口拆分（草案）

1. 执行入口响应扩展
   - `POST /api/messages`
   - 返回字段补充：`execution_mode`、`task_id`（异步时必填）、`estimated_duration_seconds`、`task_card`（异步时必填）
   - `task_card` 字段：`notice`、`task_id`、`task_summary`、`task_detail_url`
2. 任务复杂度预判（内部能力）
   - 编排层调用执行器预判接口（内部），输出用于同步/异步分流决策
3. 异步任务终态主动同步
   - 基于现有会话推送通道向原会话发送 `task_completed`/`task_failed`/`task_canceled` 事件
   - 事件字段：`task_id`、`status`、`finished_at`、`result_summary`、`error_summary`
4. 异步任务查询
   - 复用现有任务查询接口返回分流决策信息，便于前端展示“已转异步”状态

#### Traceability

- 核心对象：`estimated_duration_seconds`、`complexity_level`、`execution_mode`、`task_card`、`task_id`、`task_detail_url`、`session_id`
- 依赖需求：`R-013`、`R-030`、`R-033`、`R-036`
- 验证口径：分流命中准确性、任务卡片可达性、终态主动同步到达率、流式首包时延、预判失败兜底可用性

### R-042 异步任务执行模块与前端全量执行明细

1. 异步任务必须由独立执行模块统一调度与执行，不与同步流式链路混用执行线程与队列，确保任务治理边界清晰。
2. 异步执行模块并发上限固定为 `5`，当运行中任务达到上限时，新任务进入等待队列并展示排队状态。
3. 异步任务状态至少包含：`queued`、`running`、`success`、`failed`、`canceled`，并提供状态变更时间与阶段信息。
4. 前端 `Tasks` 模块需提供任务执行明细视图，交互风格与当前对话界面一致，支持实时查看任务阶段、日志流与终态结果。
5. 异步任务日志必须完整回传执行终端输出细节，不做语义裁剪；包括步骤状态与运行事件（如“已运行 xxx”）。
6. 日志展示要求：按时间顺序增量渲染，保留原始输出层级与时间戳，支持断线后续读与历史回放。
7. 验收：并发提交超过 5 个异步任务时，系统稳定执行 `5` 个并将其余任务排队；前端任务明细可完整查看终端执行全量日志与最终结果。

#### 接口拆分（草案）

1. 异步任务提交
   - `POST /api/tasks`
   - 返回：`task_id`、`status`、`queue_position`、`accepted_at`
2. 异步任务详情
   - `GET /api/tasks/{task_id}`
   - 返回：`status`、`phase`、`started_at`、`finished_at`、`queue_wait_ms`、`error`
3. 异步任务日志流与回放
   - `GET /api/tasks/{task_id}/logs/stream?cursor=`
   - `GET /api/tasks/{task_id}/logs?cursor=&limit=`
   - 返回：终端原始日志片段、时间戳、序号、阶段标签

#### Traceability

- 核心对象：`async_executor`、`max_concurrency=5`、`task_queue`、`task_id`、`terminal_logs`
- 依赖需求：`R-030`、`R-033`、`R-035`、`R-041`
- 验证口径：并发上限生效、排队可见性、日志完整性、前端明细可读性

### R-043 Environments 关键配置统一管理

1. `Environments` 页面从占位状态升级为可配置模块，提供关键运行配置的查看、编辑、校验、保存与生效状态展示。
2. 首批必须纳管任务执行相关参数，至少包含：异步任务并发执行数量（`async_task_workers`）、异步任务超时（`async_task_timeout`）、异步任务重试次数（`async_task_max_retries`）、异步分流阈值（`async_long_content_threshold`）。
3. 首批纳管配置同时覆盖全局并发与队列参数（`worker_pool_size`、`max_queue_size`、`queue_timeout`），支持边界校验（最小值、最大值、必填）。
4. 对当前已存在且可参数化的运行配置建立统一配置字典（名称、类型、默认值、当前值、是否敏感、是否支持热更新），并在页面中按模块分组展示。
5. 配置持久化要求：用户保存后写入统一配置存储（环境配置文件或等价持久化介质），重启后必须保持；未配置项回退到系统默认值。
6. 生效策略要求：明确标注“即时生效”与“重启生效”配置；对需重启项在保存后给出明确提示，避免配置生效预期不一致。
7. 安全要求：敏感配置（密钥、令牌类）默认脱敏显示并支持受控查看；配置变更记录审计字段（操作者、时间、变更项）。
8. 验收：用户可在 Environments 页面完成任务并发等关键配置修改并持久化；系统可正确读取并在运行链路中生效，页面可查看当前生效值与生效方式。

#### 当前配置盘点（可纳管候选）

1. Web 与并发队列
   - `web_addr`（当前启动参数：`-web-addr`，默认 `127.0.0.1:18088`）
   - `worker_pool_size`（`-worker-pool-size`，默认 `4`）
   - `max_queue_size`（`-max-queue-size`，默认 `128`）
   - `queue_timeout`（`-queue-timeout`，默认 `5s`）
2. 异步任务
   - `async_task_workers`（`-async-task-workers`，默认 `2`）
   - `async_task_timeout`（`-async-task-timeout`，默认 `90s`）
   - `async_task_max_retries`（`-async-task-max-retries`，默认 `1`）
   - `async_long_content_threshold`（`-async-long-content-threshold`，默认 `240`）
3. 会话记忆与上下文压缩
   - `session_memory_turns`（`-session-memory-turns`，默认 `6`）
   - `session_memory_ttl`（`-session-memory-ttl`，默认 `20m`）
   - `context_compression_threshold`（`-context-compression-threshold`，默认 `1200`）
   - `context_compression_summary_tokens`（`-context-compression-summary-tokens`，默认 `220`）
   - `context_compression_retain_turns`（`-context-compression-retain-turns`，默认 `4`）
4. 持久化记忆与强制上下文
   - `daily_memory_dir`（`-daily-memory-dir`，默认 `.alter0/memory`）
   - `long_term_memory_path`（`-long-term-memory-path`，默认 `.alter0/memory/long-term/MEMORY.md`）
   - `long_term_memory_write_policy`（`-long-term-memory-write-policy`，默认 `write_through`）
   - `long_term_memory_writeback_flush`（`-long-term-memory-writeback-flush`，默认 `2s`）
   - `long_term_memory_token_budget`（`-long-term-memory-token-budget`，默认 `220`）
   - `mandatory_context_file`（`-mandatory-context-file`，默认 `SOUL.md`）

#### 接口拆分（草案）

1. 环境配置字典与当前值
   - `GET /api/control/environments`
   - 返回：分组配置定义 + 当前值 + 默认值 + 生效策略 + 校验规则
2. 环境配置更新
   - `PUT /api/control/environments`
   - 入参：配置键值变更集；返回：校验结果、是否需要重启、已应用项
3. 环境配置审计（可选）
   - `GET /api/control/environments/audits`
   - 返回：变更时间、变更项、操作者、生效状态

#### Traceability

- 核心对象：`environments_module`、`config_registry`、`async_task_workers`、`worker_pool_size`、`queue_timeout`
- 依赖需求：`R-042`
- 验证口径：配置可编辑性、参数校验正确性、持久化一致性、运行时生效一致性

### R-044 Channels 迁移至 Settings 模块

1. 前端侧边栏信息架构调整：`Channels` 导航入口从 `Control` 分组迁移到 `Settings` 分组，分组标题与排序保持一致性。
2. 迁移后 `Channels` 页面功能、接口调用与数据模型保持不变，不改变现有通道管理能力与权限边界。
3. 路由兼容要求：保留原有 `#/channels` 与等价页面路径可访问，避免因导航分组调整导致历史链接失效。
4. 高亮与默认展开规则同步调整：进入 `Channels` 页面时应在 `Settings` 分组下正确高亮，不再归属 `Control` 分组视觉上下文。
5. 多端一致性要求：桌面端与移动端导航结构同步生效，不引入侧边栏折叠、滚动或遮罩交互回归。
6. 文案与国际化要求：`Channels` 的导航文案与页面标题保持一致，迁移后中英文资源不出现错位或缺失。
7. 验收：用户可在 `Settings -> Channels` 进入通道管理页；历史直达路由仍可访问；`Control` 分组不再展示 `Channels` 入口。

#### 实施边界（草案）

1. 仅调整前端导航归属与展示结构，不新增后端领域模型或控制接口。
2. 不修改 `Channel` 的存储结构、校验规则与启停逻辑。
3. 不改变现有 `GET/PUT/DELETE /api/control/channels/*` 接口契约。

#### Traceability

- 核心对象：`sidebar_navigation`、`settings_group`、`channels_route`
- 依赖需求：`R-007`、`R-012`、`R-039`
- 验证口径：导航归属正确性、路由兼容性、移动端一致性、无功能回归

### R-045 Session/Task 关键信息披露增强

1. 在前端 `Sessions`、`Tasks`、任务详情等页面补充关键链路字段展示，确保用户可快速定位一次执行从“消息 -> 会话 -> 任务”的关联关系。
2. Session 维度至少展示：`session_id`、`channel_type`、`channel_id`、最近 `message_id`（或首条/末条消息 ID）、`created_at`、`updated_at`。
3. Task 维度至少展示：`task_id`、`session_id`、`source_message_id`、`channel_type`、`channel_id`、`trigger_type`、`correlation_id`、`status`、`created_at`、`finished_at`。
4. 当任务来源于定时调度时，额外展示：`job_id`、`job_name`、`fired_at`，并保持与会话来源信息一致。
5. 字段展示分层：列表卡片默认展示高频字段；详情抽屉/详情页展示完整字段，避免首页信息过载。
6. 检索与筛选能力同步增强：支持按 `channel_type`、`channel_id`、`message_id/source_message_id`、`trigger_type` 快速过滤与定位记录。
7. 安全与可用性边界：缺失字段显示 `-`，不因历史数据不完整导致页面报错；敏感信息（如密钥类）不在该模块披露。
8. 验收：用户在 Session/Task 页面可直接看到 `channel` 与 `message_id` 等关键字段，并可据此完成跨页面定位与问题排查。

#### 字段披露优先级（草案）

1. P0（列表必显）
   - Sessions：`session_id`、`channel_type`、`last_message_id`、`updated_at`
   - Tasks：`task_id`、`status`、`channel_type`、`source_message_id`、`trigger_type`
2. P1（详情必显）
   - Sessions：`channel_id`、`created_at`、`message_count`
   - Tasks：`channel_id`、`correlation_id`、`finished_at`、`error_summary`
3. P2（条件展示）
   - Cron 来源：`job_id`、`job_name`、`fired_at`

#### 接口拆分（草案）

1. Session 列表/详情扩展字段
   - `GET /api/sessions`
   - `GET /api/sessions/{session_id}/messages`
   - 补充返回：`channel_type`、`channel_id`、`message_id` 关联字段
2. Task 列表/详情扩展字段
   - `GET /api/tasks`
   - `GET /api/tasks/{task_id}`
   - 补充返回：`source_message_id`、`channel_type`、`channel_id`、`trigger_type`、`correlation_id`
3. 跨页面回链（扩展）
   - 从 `session_id + message_id` 跳转任务详情
   - 从 `task_id` 回跳对应会话消息定位

#### Traceability

- 核心对象：`session_id`、`task_id`、`channel_type`、`channel_id`、`message_id`、`source_message_id`、`correlation_id`
- 依赖需求：`R-019`、`R-030`、`R-034`
- 验证口径：字段完整性、筛选可用性、跨页面回链可达性、历史数据兼容性

### R-046
1. 前端新增独立 `Terminal` 模块入口，用户可在该模块内直接发起和持续进行终端会话，不依赖 `Tasks` 抽屉作为入口。
2. 终端会话模型要求：
   - 每个会话拥有稳定 `terminal_session_id`；
   - 会话内保留 `anchor_task_id` 与当前 `active_task_id`；
   - 新输入默认复用当前会话上下文，保持同一 Shell 连续执行语义。
3. 交互要求：模块内提供类 Chat 的输入输出区域，用户输入、任务受理结果、终端日志输出按时间顺序连续展示。
4. 串联要求：同一终端会话中的多次输入必须串联到同一会话轨迹，避免每次输入被视为独立无关联任务。
5. 状态要求：模块需可见当前会话关键字段（`terminal_session_id`、`anchor_task_id`、`active_task_id`、`status`），并实时刷新执行状态。
6. 可恢复性要求：页面刷新后可恢复终端会话列表和最近上下文，保证继续交互时链路不丢失；浏览器临时 `sessionStorage` 丢失时，前端仍需优先复用已持久化的 Terminal client 标识，避免把仍可恢复的会话误判为中断或丢失。
7. 异常处理要求：终端运行态退出、日志回读失败或发送失败时，模块需展示可感知错误信息，并保留当前会话的继续恢复入口。
8. 移动端输入要求：当 `Terminal` 会话处于轮询刷新中，若用户正在移动端输入框内持续输入，前端不得因状态刷新销毁并重建输入框；焦点、草稿与当前滚动位置需保持稳定。输入法候选确认会结束当前组合态，但只要输入框仍聚焦，仍不得立即触发视图重绘。
9. 移动端布局要求：`Terminal` 页面在移动端必须采用顶部会话信息区、中部独立滚动消息区、底部独立输入条的结构，不得继续使用会话列表、消息流与输入框共处同一文档流的桌面缩放式布局。
10. 移动端信息收纳要求：会话列表默认收纳为可展开的会话面板；当前会话的 `terminal_session_id`、CLI 路径、工作目录等元信息默认折叠，用户按需展开查看并可复制。
11. 输出渲染要求：长 `Final Output` 默认允许折叠/展开；Markdown 链接需按链接文本渲染为可点击链接，不直接向用户展示整段 Markdown 源码或冗长路径。
12. 验收：用户在 `Terminal` 模块创建会话后可连续发送多轮命令与追问；界面持续展示同一会话链路的输入/输出，且会话上下文保持连续；移动端输入过程中页面不会因轮询刷新或输入法候选确认跳回顶部，长输出与会话元信息可按需展开。

#### 接口拆分（草案）

1. 终端首轮任务创建
   - `POST /api/tasks`
   - 请求元数据包含 `alter0.task.terminal_session_id` 与 `alter0.task.terminal_interactive=true`
2. 终端会话续写
   - `POST /api/control/tasks/{task_id}/terminal/input`
   - 入参：`input`、`reuse_task=true`、`anchor_task_id`
3. 终端日志回读
   - `GET /api/control/tasks/{task_id}/logs?cursor=&limit=`
4. 终端状态刷新
   - `GET /api/control/tasks/{task_id}`

#### Traceability

- 核心对象：`terminal_session_id`、`anchor_task_id`、`active_task_id`、`terminal_interactive`、`terminal_logs`
- 依赖需求：`R-035`、`R-042`
- 验证口径：会话串联连续性、输入输出可读性、刷新恢复能力、退出后恢复可感知性、移动端输入稳定性

### R-047

暂无细化内容。

### R-048

暂无细化内容。

### R-049

暂无细化内容。

### R-050
Web 登录后统一 Session 视图

1. 当 `web-login-password` 生效且用户已完成密码验证后，当前 Web 登录态下的 `Chat`、`Agent` 等对话页面必须共享同一套 Session 历史数据，不再按页面来源拆分历史列表或本地 Session 缓存。
2. `Chat`、`Agent` 等页面仍需保留各自独立的页面定位、执行目标、运行模式、欢迎态与入口结构；本需求仅取消按页面来源区分 Session 的规则，不取消页面能力区分。
3. 统一范围至少覆盖：左侧最近 Session 列表、Session 详情回放、页面刷新后的 Session 恢复，以及不同 Web 对话页面之间的 Session 延续。
4. 同一条 Web Session 可在统一历史中查看、进入与继续；用户可从 `Chat` 创建 Session，再在 `Agent` 页面进入同一 Session 继续使用，反之亦然。
5. 前端需移除仅用于区分 Web Session 来源的无效内容，包括但不限于：
   - “独立会话历史”类文案；
   - 按页面来源拆分 Session 的本地缓存键、会话桶或冗余状态字段；
   - 仅因 Session 来源拆分而存在的重复会话创建逻辑；
   - 将页面类型错误表达为 Session 类型的提示文案。
6. `Sessions` 页面默认展示统一后的 Web Session 视图，不再将 `Chat` / `Agent` 作为 Session 分类口径；若保留 `trigger_type`、`channel_type` 等系统字段，仅用于调试、排障与系统链路披露，不作为 Web Session 分栏依据。
7. 登录边界要求：该共享规则在密码验证通过后的 Web 登录态内生效；未登录用户不可访问共享历史，登出或登录态失效后仍需重新验证。
8. 兼容性要求：历史已存在的 Web Session 在升级后需可继续读取并纳入统一列表；旧的按页面拆分缓存若存在，应在迁移后自动合并或失效，不得造成历史丢失或重复展示。
9. 验收：用户登录后在 `Chat` 页面创建的 Session，可在 `Agent` 页面直接看到并继续；在任一 Web 对话页新增或回复消息后，其他 Web 对话页刷新或切换后可看到同一条 Session 历史；界面中不再出现“独立会话历史”或等价的 Session 来源拆分提示，但 `Chat`、`Agent` 页面入口与能力边界保持独立。

#### 实施边界（草案）

1. 本需求聚焦 Web 端会话历史呈现与会话入口一致性，不改变后端任务、调度、审计等领域对象中的 `trigger_type`、`channel_type`、`channel_id` 存储模型。
2. 本需求不取消 Cron、CLI 等非 Web 来源的系统溯源字段；仅取消 Web 页面内部按入口来源拆分会话历史的交互与产品口径。
3. 本需求不要求合并不同用户或不同登录态之间的历史；共享范围限定在同一 Web 登录态下的可见会话集合。
4. 本需求不要求合并页面能力本身；`Chat`、`Agent` 等页面仍可保留独立表单、独立配置项与独立交互流程。

#### Traceability

- 核心对象：`web_login_password`、`web_session`、`session_history`、`chat_route`、`agent_route`
- 依赖需求：`R-002`、`R-012`、`R-019`、`R-045`
- 验证口径：统一历史可见性、跨页面续聊一致性、旧缓存迁移完整性、冗余来源文案清理完整性
