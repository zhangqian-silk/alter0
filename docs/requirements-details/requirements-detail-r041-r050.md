# Requirements Details (R-041 ~ R-050)

> Last update: 2026-04-02

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
6. 对于落到 `Codex CLI` 的异步任务，执行器必须每 `1m` 产出一次运行心跳；任务模块收到心跳后更新 `last_heartbeat_at`、`updated_at`、`timeout_at`，并追加 `heartbeat` 日志，确保“仍在运行中的长任务”不会被固定墙钟超时直接终止。
7. `async_task_timeout` 的执行语义保持为“当前运行窗口”；初始窗口从任务开始计时，后续由 `Codex CLI` 心跳续租。若到达窗口上限前既没有完成也没有收到新心跳，任务按 `task_timeout` 失败。
8. 日志展示要求：按时间顺序增量渲染，保留原始输出层级与时间戳，支持断线后续读与历史回放。
9. 前端展示要求：`Tasks` 列表卡片需先展示轻量心跳摘要；`Tasks` 详情抽屉与 `Memory -> Tasks` 详情页需明确展示 `last_heartbeat_at` 与 `timeout_at`，便于区分“持续续租中的运行态任务”和“长时间无心跳、接近超时的任务”。
10. 验收：并发提交超过 5 个异步任务时，系统稳定执行 `5` 个并将其余任务排队；前端任务明细可完整查看终端执行全量日志、心跳日志、最近心跳时间与当前超时窗口；运行中 `Codex CLI` 会话即使超过默认 `90s`，只要仍持续上报心跳，就不会被误判超时。

#### 接口拆分（草案）

1. 异步任务提交
   - `POST /api/tasks`
   - 返回：`task_id`、`status`、`queue_position`、`accepted_at`
2. 异步任务详情
   - `GET /api/tasks/{task_id}`
   - 返回：`status`、`phase`、`started_at`、`finished_at`、`queue_wait_ms`、`last_heartbeat_at`、`timeout_at`、`error`
3. 异步任务日志流与回放
   - `GET /api/tasks/{task_id}/logs/stream?cursor=`
   - `GET /api/tasks/{task_id}/logs?cursor=&limit=`
   - 返回：终端原始日志片段、时间戳、序号、阶段标签

#### Traceability

- 核心对象：`async_executor`、`max_concurrency=5`、`task_queue`、`task_id`、`terminal_logs`、`last_heartbeat_at`、`timeout_at`
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
8. 运行时披露要求：`Environments` 工具栏需展示当前在线实例最近启动时间与对应 `commit hash`，用于确认上次成功重启后实际运行的版本。
9. 重启反馈要求：Web 控制台触发运行时重启时，必须先展示单一站内确认弹窗；“同步远端 master 最新改动”作为弹窗内勾选项提供，默认勾选，不得再通过连续浏览器确认框重复询问。提交后页面需在新实例探活通过后自动刷新，并以站内弹窗明确提示当前页面已连接到最新运行实例；提示不可依赖可能被浏览器压制的异步系统弹窗。
10. systemd 部署基线要求：服务启动脚本需将运行进程 `HOME` 收敛到 `ALTER0_RUNTIME_ROOT` 对应根目录，默认值为 `/var/lib/alter0`；若历史环境文件仍配置为 `/var/lib/alter0/codex-home`，启动阶段需自动归一为 `/var/lib/alter0`，确保 Codex 认证与运行态缓存落在统一运行根目录。
11. 运行账户能力基线要求：服务运行账户的默认 `PATH` 必须稳定包含 `$HOME/.local/bin`、`/usr/local/bin`、`/usr/bin` 等标准目录，确保 `Codex CLI` 子进程可见 `codex`、`node`、运行账户级 `gh` 包装器与其他必要工具，而不依赖交互式 shell 配置。
12. Git 交付链路基线要求：若服务内允许 `Codex CLI` 执行 `git commit`、`git push`、`gh pr create`、`gh pr merge` 等仓库交付动作，部署阶段必须为运行账户补齐 GitHub App token helper、`gh` 包装器、SSH signing key 与对应全局 Git 配置；初始化过程需由 root 一次性完成，后续运行态不再依赖 root 的交互环境。
13. 验收：用户可在 Environments 页面完成任务并发等关键配置修改并持久化；系统可正确读取并在运行链路中生效，页面可查看当前生效值、生效方式以及当前在线版本对应的最近启动时间与 `commit hash`；运行时重启成功后用户可收到稳定的页面内成功提示；systemd 场景下服务进程 `HOME` 默认落在 `/var/lib/alter0`，历史 `codex-home` 子目录配置不会继续生效为独立根目录，且服务内 `Codex CLI` 可直接完成带签名的 commit / PR / merge 链路。

#### 当前配置盘点（可纳管候选）

1. Web 与并发队列
   - `web_addr`（当前启动参数：`-web-addr`，默认 `127.0.0.1:18088`）
   - `worker_pool_size`（`-worker-pool-size`，默认 `4`）
   - `max_queue_size`（`-max-queue-size`，默认 `128`）
   - `queue_timeout`（`-queue-timeout`，默认 `5s`）
2. 异步任务
   - `async_task_workers`（`-async-task-workers`，默认 `2`）
   - `async_task_timeout`（`-async-task-timeout`，默认 `90s`，作为单个运行窗口时长；`Codex CLI` 长任务收到心跳后按窗口续租）
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
5. 部署运行根目录
   - `ALTER0_RUNTIME_ROOT`（启动脚本默认 `/var/lib/alter0`）
   - `HOME`（systemd 部署建议固定为 `/var/lib/alter0`；历史 `/var/lib/alter0/codex-home` 会在启动阶段归一到运行根目录）
   - `PATH`（运行时需稳定包含 `$HOME/.local/bin`、`$HOME/.local/share/pnpm`、`/usr/local/bin`、`/usr/bin`、`/bin`）
   - `scripts/setup_alter0_runtime_auth.sh`（root 一次性初始化运行账户的 GitHub App token helper、`gh` 包装器、SSH signing key 与 Git 全局配置）

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
4. 运行时实例元信息
   - `GET /api/control/runtime`
   - 返回：`started_at`、`commit_hash`

#### Traceability

- 核心对象：`environments_module`、`config_registry`、`async_task_workers`、`worker_pool_size`、`queue_timeout`、`started_at`、`commit_hash`
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
   - 会话态使用 `ready / busy / exited / interrupted` 表示当前会话是否可继续交互；turn / step 维度继续使用 `running / completed / failed / interrupted` 表示执行进展。
6. 可恢复性要求：页面刷新后可恢复终端会话列表和最近上下文，保证继续交互时链路不丢失；同一 Web 登录态下，手机与 PC 进入 Terminal 时需看到一致的服务端会话集合，不因浏览器设备标识不同而分叉。
7. 异常处理要求：终端运行态退出、日志回读失败或发送失败时，模块需展示可感知错误信息，并保留当前会话的继续恢复入口。
8. 移动端输入要求：当 `Terminal` 会话处于轮询刷新中，若用户正在移动端输入框内持续输入，前端不得因状态刷新销毁并重建输入框；焦点、草稿与当前滚动位置需保持稳定。输入法候选确认会结束当前组合态，但只要输入框仍聚焦，仍不得立即触发视图重绘。
9. 移动端布局要求：`Terminal` 页面在移动端必须采用紧凑顶部会话信息区、中部独立滚动消息区、底部独立输入条的结构，不得继续使用会话列表、消息流与输入框共处同一文档流的桌面缩放式布局。
   - 页面级 `Terminal` 标题、副标题等低频说明信息在移动端不应持续占据聊天区上方高度；当前会话标题与状态需收敛到工作区头部。
   - 顶部通用栏不再显示 `New Chat`；Terminal 自身的会话切换与新建入口收敛到顶部通用栏中 `Menu` 右侧的 `Sessions / New`。
10. 移动端信息收纳要求：会话列表默认收纳为可展开的会话面板；当前会话的 `terminal_session_id`、CLI 路径、工作目录等元信息默认折叠，用户按需展开查看并可复制。
   - 工作区头部需收敛为单行摘要工具栏：左侧展示当前会话标题与状态点，右侧提供紧凑 `Details / Close / Delete` 操作；状态不再以占位过大的独立按钮呈现，避免在小屏上拆成多行松散布局。
   - 会话列表中的每个会话需提供独立删除入口，用户无需先切入该会话即可直接清理历史会话。
   - `Close` 仅退出当前 Codex 运行态并保留原会话历史；`Delete` 需直接删除会话本身并同步清理持久化状态文件与该会话对应的独立工作区，前端不依赖“先改成已退出状态”再移除。
   - `Process` 步骤行保持单行摘要布局：步骤标题需可截断，右侧时间与状态需固定在独立区域，不得互相覆盖。
11. 输入区要求：发送区需采用自适应高度输入框与紧凑发送按钮，优先把纵向空间让给消息区；输入内容增多时，输入框只在设定上限内增长，超过上限后输入框内部滚动，不得把发送入口挤出可视区。
   - 运行态 `Interrupted / Exited / Failed` 等提示需以内嵌状态条贴合输入区上缘展示，不再以独立悬浮文字条打断消息流与输入流。
   - 当用户从旧会话直接点击 `New` 创建新 Terminal 会话时，工作区需立即切换到干净的待创建会话态；创建请求完成前不得继续展示旧会话的运行态提示、错误文案或退出信息。
   - 全站聊天页与 Terminal 中的用户消息需统一右对齐展示，单条用户气泡最大宽度不得超过消息区宽度的 80%；同一轮的用户消息与其后续回复之间的垂直间距需小于用户消息与上一轮回复之间的间距，避免视觉归属错位。
   - Terminal 中的用户输入气泡不得再额外叠加命令前缀符号、前缀圆点或强调色文字，视觉上直接收敛为普通用户消息样式。
   - Chat 与 Terminal 的视觉风格需统一收敛到浅色文档式阅读主题：整体以留白、低对比边框、柔和蓝灰层级和轻量玻璃感输入区为主，避免高饱和大色块与厚重卡片阴影破坏连续阅读。
12. 浏览增强要求：长回复滚动阅读时，用户消息、`Process` 与终端回复正文均保持自然文档流滚动，不再启用吸顶提示、吸顶桥接动画或相关的滚动测量状态机；阅读定位通过右侧圆形四键组承担，至少包含回到顶部、上一条、下一条与回到底部，避免长内容场景下的遮挡、回流与滚动卡顿。
   - 长输出滚动过程中不得动态插入吸顶副本、吸顶占位层或半透明覆盖条；相关滚动同步逻辑不得额外维护吸顶高度、区块桥接间距或吸顶触发区间计算。
   - 四键导航的上一条与下一条需基于当前视口中的可见 turn 实时计算；当 `Process` 折叠/展开导致布局重排时，按钮显隐与跳转目标需同步刷新，不得保留折叠前状态。
   - `Process` 折叠等交互在普通滚动态下需始终可点，不得因阅读辅助控件遮挡而失效。
13. 移动端功耗要求：`Terminal` 的会话轮询需按页面可见性、输入聚焦、滚动活跃度和会话列表更新频率自动降频；活动会话详情与会话列表需拆分不同刷新周期，服务端状态无变化时前端不得重复执行整块 `paint` 与高频本地持久化。
   - 轮询刷新不得在输出持续增长时整块替换当前终端工作区；至少要保留消息滚动容器节点与当前位置，避免用户滚动阅读过程中出现周期性卡顿。
   - 浏览器侧 Terminal 会话缓存写入需避开活跃滚动窗口；当用户持续滚动消息区时，允许延后本地持久化，待滚动停顿后再写入。
13. 输出渲染要求：Terminal 最终回复需直接按 Chat 助手消息逻辑渲染，不再额外包裹 `Final Output` 标题条、强调边框或 `Show more / Show less` 折叠按钮；Markdown 链接需按链接文本渲染为可点击链接，不直接向用户展示整段 Markdown 源码或冗长路径。
   - Terminal 最终回复正文需提供一键复制入口；复制内容仅包含最终回复，不包含 `Process` 步骤细节或中间日志。
   - `Process` 需继续采用浅色平铺阅读面板与统一阅读节奏，不再通过“盒子套盒子”的高饱和大块背景制造额外视觉层级；头部需以低对比虚线提示区承载过程信息，左侧纵向引导线从 `Process` 头部中点向下延伸。
   - `Process` 中的步骤摘要需压缩为单行结构，仅展示“截断后的命令 / 时间 / 状态”三部分；移动端也不得把时间或状态下沉到第二行。展开后正文只展示细节内容，不再重复状态标签。
   - 同一 turn 出现最终输出后，前端需自动折叠对应 `Process` 面板，把阅读焦点收敛到最终回复；用户手动再次展开后保留该折叠状态选择。
   - 移动端下 `Process` 头部需默认保持单行排布，`Process` 标签、步骤摘要与耗时在可用宽度内同排展示；步骤行中的摘要、耗时与状态也需保持同排。超长摘要按单行截断，不因默认换行把单条消息挤成上下两行。
   - 用户输入气泡、Terminal prompt、会话侧栏、工作区容器与跳转按钮需共用一致的圆角、阴影和边框节奏，形成统一的轻阅读界面语言，而不是 Chat 与 Terminal 各自独立的视觉体系。
   - 长路径、超长单词、代码块、diff 与编号输出不得把终端消息卡片、最终输出卡片或聊天区外层容器撑出边框；超宽内容仅允许在自身内容块内横向滚动。
14. 验收：用户在 `Terminal` 模块创建会话后可连续发送多轮命令与追问；界面持续展示同一会话链路的输入/输出，且会话上下文保持连续；移动端输入过程中页面不会因轮询刷新或输入法候选确认跳回顶部，长输出与会话元信息可按需展开，并支持普通流式滚动阅读、`Process` 折叠与一键回到顶部/底部。
   - 补充验收：同一轮终端结果生成最终输出后，对应 `Process` 会自动收起；若用户手动重新展开，后续刷新仍保留展开状态。
   - 补充验收：移动端浏览 Terminal 时，`Process` 头部与步骤行默认保持单行阅读，步骤摘要过长时单行截断，不出现标题、耗时或状态被拆成两行的默认排版。
   - 补充验收：用户执行 `Delete` 后，该 Terminal 会话立即从列表移除，`.alter0/state/terminal/sessions/<terminal_session_id>.json` 与 `.alter0/workspaces/terminal/sessions/<terminal_session_id>` 同步清理。
   - 补充验收：用户在查看 `Interrupted / Exited / Failed` 的旧会话时点击 `New`，输入区 hint 会立即清空；待新会话创建完成后，仅展示新会话自身状态。

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

OpenAI Go SDK 接入

1. 统一 LLM 调用层基于 `github.com/openai/openai-go` SDK 实现，屏蔽上游 `/responses` 与 `/chat/completions` 差异，对外暴露统一 `Chat` / `ChatStream` 接口。
2. Provider 级配置必须支持自定义 `base_url` 与 `api_key`，以兼容 OpenAI 官方与 OpenAI 兼容服务。
3. Provider 必须显式声明 `api_type`：
   - `openai-responses`：使用 `/responses`
   - `openai-completions`：使用 `/chat/completions`
4. 统一调用层需支持：
   - 普通文本问答
   - 流式输出
   - function tools / tool calls
   - 上下游使用量字段回传
5. 若模型调用失败，错误需保留上游错误语义，供复杂度评估、Agent 执行与 Web API 直接透传或降级处理。

#### Traceability

- 实现文件：`internal/llm/domain/llm.go`、`internal/llm/infrastructure/openai_client.go`、`internal/llm/application/model_config_service.go`
- 测试覆盖：`internal/llm/infrastructure/openai_client_test.go`
- 核心对象：`api_type`、`base_url`、`api_key`、`ChatRequest`、`ChatResponse`

### R-048

ReAct 模式 Agent 调用

1. `Agent` 请求进入执行链后，默认使用 ReAct 循环推进：模型在 `Thought / Action / Observation` 的迭代中持续决策，直到明确收口。
2. ReAct Agent 必须支持统一运行时工具调用；Agent 自身定位为“用户代理 + 执行驱动器”，所有具体仓库、文件、命令与实现动作默认通过 `codex_exec` 交给 Codex CLI 执行。
3. Agent Profile 可独立配置：
   - `provider_id`
   - `model`
   - `system_prompt`
   - `max_iterations`
   - `tools`
   - `skills`
   - `mcps`
   - `memory_files`
4. 当用户在会话级显式选择 `Provider / Model` 时，执行链优先使用当前会话选择；未显式指定时回退到 Agent Profile，再回退到系统默认 Provider。
5. 执行中需保留观察日志与输出增量，支持同步响应、流式响应与异步任务场景复用。
6. ReAct 工具收口必须支持显式 `complete`，避免模型在没有结束信号时无限循环。
7. Agent 运行时的稳定工具面至少包括：`codex_exec`、`search_memory`、`read_memory`、`write_memory`、`complete`；仅允许委派的 Agent 可额外挂载 `delegate_agent`。
8. `search_memory` / `read_memory` / `write_memory` 仅面向已解析进 `memory_context` 的记忆文件。`search_memory` 负责按关键字在多份记忆文件中定位历史偏好、缩写指代和长期约束，`read_memory` 用于精读单个目标文件，`write_memory` 用于在必要时维护这些记忆文件本身；除此之外不再向 Agent 暴露通用原生文件/命令工具。
9. Web 端对 Agent 流式返回的 `action / observation` 细节必须收敛为可折叠 `Process` 区块，避免多轮执行日志直接淹没最终答复；最终答复继续按普通助手正文渲染。
10. 当同一条 Agent 回复已出现最终答复时，`Process` 默认折叠；用户手动展开后需在当前浏览器会话内保留该折叠状态。
11. Agent 最终答复正文需提供一键复制入口；若同条消息同时包含 `Process`，复制内容仅包含最终答复，不包含折叠的 `action / observation` 细节。
12. 若 Agent 在 `max_iterations` 耗尽前仍未调用 `complete`，运行时不得返回空最终正文；系统必须显式返回“达到迭代上限”的说明，并附带最后一次工具观察结果，保证 Web 流式消息与最终 `done` 收口一致可见。

#### Traceability

- 实现文件：`internal/llm/domain/react.go`、`internal/execution/infrastructure/hybrid_nl_processor.go`、`internal/control/domain/agent.go`、`internal/interfaces/web/static/assets/chat.js`、`internal/interfaces/web/static/assets/chat-core.css`、`internal/interfaces/web/static/assets/chat-terminal.css`
- 测试覆盖：`internal/llm/domain/react_test.go`、`internal/execution/infrastructure/hybrid_nl_processor_test.go`
- 核心对象：`ReActAgentConfig`、`ToolExecutor`、`max_iterations`、`codex_exec`
- 验证记录：
  - 2026-03-31：Web `Chat / Agent` 已将 Agent 流式 `action / observation` 解析为消息内可折叠 `Process`，最终答复与执行细节分区展示。
  - 2026-03-31：Agent 最终答复与 Terminal 最终输出均新增一键复制入口；复制内容仅包含最终正文，不包含 `Process` 细节。
  - 2026-03-31：Agent 在 ReAct 迭代耗尽但仍停留在工具调用阶段时，运行时改为显式返回迭代上限提示与最后一次工具观察，前端 `done` 事件在空 `result.output` 场景下保留已流出的正文。

### R-049

模型配置管理

1. 控制面必须提供 LLM Provider 配置管理入口，支持新增、编辑、删除、启用、禁用与设置默认 Provider。
2. 单个 Provider 至少包含：
   - `id`
   - `name`
   - `api_type`
   - `base_url`
   - `api_key`
   - `default_model`
   - `models[]`
   - `is_enabled`
3. 单个模型项至少包含：
   - `id`
   - `name`
   - `is_enabled`
   - `supports_tools`
   - `supports_vision`
   - `supports_streaming`
   - `max_tokens`
4. 默认策略约束：
   - 默认 Provider 只能指向已启用 Provider
   - 默认模型只能指向当前 Provider 下已启用模型
   - 若当前默认 Provider 被禁用、删除或历史配置已失效，系统自动切换到下一个可用 Provider；若无可用 Provider，则清空默认值
5. Web `Chat` 发送区必须支持会话级 `Provider / Model` 选择；复杂度评估、同步执行与 Agent/ReAct 执行需复用该选择结果。
6. Agent Profile 必须支持独立绑定 `provider_id` 与 `model`，用于为不同 Agent 预设执行模型。
7. 历史配置兼容要求：旧配置中若默认 Provider 指向禁用项或缺失项，加载时需自动收敛到可用默认值，不得因脏数据阻断系统启动。

#### Traceability

- 实现文件：`internal/llm/domain/model_config.go`、`internal/llm/application/model_config_service.go`、`internal/interfaces/web/server.go`、`internal/interfaces/web/static/assets/chat.js`
- 测试覆盖：`internal/llm/domain/model_config_test.go`、`internal/llm/infrastructure/model_config_storage_test.go`、`internal/interfaces/web/server_llm_test.go`
- 核心对象：`providers`、`default_provider_id`、`default_model`、`alter0.llm.provider_id`、`alter0.llm.model`

### R-050
Web 登录后按 Agent 隔离 Session 视图

1. 当 `web-login-password` 生效且用户已完成密码验证后，当前 Web 登录态下的 Web 对话页面必须按目标 Agent 维护独立 Session 历史，不再将不同 Agent 的会话混放在同一历史列表中。
2. `Chat` 页面绑定具备独立前端入口的 Agent；`Agent` 页面承载未占用独立前端入口的内置 Agent 与用户管理 Agent。两类入口都需保持各自独立的欢迎态、运行模式与会话列表。
3. 隔离范围至少覆盖：左侧最近 Session 列表、Session 详情回放、页面刷新后的 Session 恢复，以及同一 Agent 下的续聊恢复。
4. 同一 Agent 的历史可在其对应入口内继续进入与续聊；不同 Agent 之间的历史默认隔离，不在其他 Agent 的会话列表中展示。
5. 前端需移除错误的统一会话口径与冗余状态，包括但不限于：
   - 将不同 Agent 会话混放到同一列表的本地缓存结构；
   - 仅按页面来源而非 Agent 目标划分会话的旧状态字段；
   - 会导致独立入口 Agent 历史出现在通用 `Agent` 页面中的筛选缺失。
6. `Sessions` 页面继续展示系统级 Session 数据；若保留 `trigger_type`、`channel_type` 等系统字段，仅用于调试、排障与系统链路披露，不作为前端 Agent 会话分栏依据。
7. 登录边界要求：该隔离规则在密码验证通过后的 Web 登录态内生效；未登录用户不可访问会话历史，登出或登录态失效后仍需重新验证。
8. 兼容性要求：历史已存在的 Web Session 在升级后需按其目标 Agent 自动归入对应历史桶；旧缓存缺少 Agent 历史分桶信息时，前端需在读取时自动补齐，不得造成历史丢失或重复展示。
9. 验收：`Chat` 页面仅展示 `Alter0` 会话历史；`Agent` 页面仅展示当前选中 Agent 的会话历史，且不会出现 `Alter0` 这类具备独立前端入口的 Agent 历史；页面刷新后仍能回到对应 Agent 的独立历史。

#### 实施边界（草案）

1. 本需求聚焦 Web 端会话历史呈现与会话入口一致性，不改变后端任务、调度、审计等领域对象中的 `trigger_type`、`channel_type`、`channel_id` 存储模型。
2. 本需求不取消 Cron、CLI 等非 Web 来源的系统溯源字段；仅调整 Web 页面内部的 Agent 会话可见性与本地恢复逻辑。
3. 本需求不要求合并不同用户或不同登录态之间的历史；隔离范围限定在同一 Web 登录态下的可见会话集合。
4. 本需求不要求新增前端专属 Agent 页面；通用 `Agent` 页面可承载无独立入口的 Agent。

#### Traceability

- 核心对象：`web_login_password`、`web_session`、`session_history`、`chat_route`、`agent_route`
- 依赖需求：`R-002`、`R-012`、`R-019`、`R-045`
- 验证口径：按 Agent 的历史隔离可见性、独立入口 Agent 的历史排除、旧缓存迁移完整性、本地恢复一致性
