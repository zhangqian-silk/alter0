# Requirements Details (R-031 ~ R-040)

> Last update: 2026-03-05

## 需求细化（草案）

### R-031 任务摘要跨会话记忆与按需深检索

1. 任务完成后必须产出结构化摘要并写入跨会话记忆层，摘要至少包含：`task_id`、`task_type`、`goal`、`result`、`status`、`finished_at`、`tags`。
2. 默认注入策略：新会话与普通问答仅注入最近 `3-5` 条高价值任务摘要，不注入详细执行日志。
3. 深检索触发策略：当用户问题包含“更早/历史/之前/上周/首个任务”等历史信号时，自动切换到全量摘要库检索，不受 `3-5` 条默认窗口限制。
4. 命中回填策略：深检索命中的历史摘要临时注入当前回答上下文，并在回复中标注可继续按 `task_id` 下钻详情。
5. 详情下钻边界：仅在用户明确请求细节时读取任务日志/产物，避免详细信息默认进入对话上下文。
6. 验收：默认模式下上下文体积稳定；历史追问可召回早期任务摘要；按 `task_id` 可继续查询完整任务日志与产物。

#### 触发词词表（草案）

- 中文：`更早`、`之前`、`历史`、`上次`、`上周`、`上个月`、`最早`、`第一条`、`当时`、`那次`。
- 英文：`earlier`、`previous`、`history`、`last time`、`last week`、`first task`、`older`。
- 时间表达：显式日期（如 `2026-03-01`）、相对时间（如 `三天前`、`两周前`）。
- 指代增强：当句子同时包含 `任务/需求/PR/执行` 与历史词时，强制启用深检索。

#### 检索排序规则（草案）

1. 先按语义相关度排序（query 与 `goal/result/tags` 的匹配分）。
2. 再按业务价值加权：`success` 与关键失败（有明确原因/产物）优先于噪声记录。
3. 再按时间衰减：默认优先近 30 天；若用户给定时间范围，则以范围约束优先。
4. 最终去重：同一 `task_id` 多条摘要仅保留最高分版本。
5. 返回策略：默认返回 `top 5` 摘要；用户明确要求“全部”时可分页继续拉取。

#### 任务历史双轨记忆策略（草案）

1. 主数据轨（Task Log）：任务系统保存完整执行日志、阶段状态、错误信息与产物引用，作为审计与排障唯一真源。
2. 摘要轨（Task Summary Memory）：任务进入终态（`success/failed/canceled`）时自动生成结构化摘要，写入跨会话记忆与天级记忆。
3. 摘要写入位置：
   - 天级摘要：`memory/YYYY-MM-DD.md`
   - 长期候选：`memory/long-term/YYYY-MM-DD.md`
   - 摘要记录必须包含 `task_id`，用于回链主数据轨。
4. 默认检索路径：先检索摘要轨并返回精简结论；不直接加载任务日志细节。
5. 按需下钻路径：仅当用户明确要求“细节/日志/报错原因/完整过程”时，基于 `task_id` 回查主数据轨并补充回答。
6. 一致性保障：若摘要轨与主数据轨状态不一致，以主数据轨为准并触发摘要重建。

#### 误触发保护规则（草案）

1. 否定语义保护：出现“不要查历史/不看以前/只看当前”等否定表达时，禁止深检索。
2. 当前上下文优先：若问题明确锚定当前任务（如“这个任务现在进度”），即便包含历史词也保持默认注入模式。
3. 双信号门槛：仅当“历史词”与“任务域词（任务/需求/PR/执行）”同时出现，或用户明确指定时间范围时触发深检索。
4. 预算保护：当深检索命中结果过多时，先返回摘要分页提示，不一次性注入超量历史内容。
5. 回退机制：深检索无命中时，自动回退到默认 `3-5` 条注入并提示可换关键词重试。
6. 可观测指标：记录 `deep_retrieval_triggered`、`deep_retrieval_overridden`、`deep_retrieval_miss`，用于评估误触发率。

#### Task Breakdown (auto-managed)

1. Scope split
   - Define task summary schema for cross-session memory.
   - Define default injection and deep-retrieval trigger rules.
2. Delivery plan
   - M1: Summary writeback pipeline and default top-k injection.
   - M2: Query-intent deep retrieval and task-detail drill-down flow.
3. Status workflow
   - Keep lifecycle as planned -> ready -> supported; update this row only.
4. Traceability
   - Keep implementation and verification records in this section after delivery.

#### Traceability

- 实现文件：`internal/task/domain/task.go`、`internal/task/application/service.go`、`internal/tasksummary/application/store.go`、`internal/orchestration/application/service.go`、`internal/orchestration/application/task_summary_prompt.go`、`internal/shared/application/telemetry.go`、`internal/shared/infrastructure/observability/telemetry.go`、`cmd/alter0/main.go`
- 测试覆盖：`internal/tasksummary/application/store_test.go`、`internal/orchestration/application/service_test.go`、`internal/task/application/service_test.go`
- 验证命令：
  - `GOSUMDB=sum.golang.org GOTOOLCHAIN=auto go test ./internal/tasksummary/application ./internal/task/application ./internal/orchestration/application ./internal/scheduler/application ./cmd/alter0`
  - `GOSUMDB=sum.golang.org GOTOOLCHAIN=auto go test ./...`
- 验证记录：
  - 2026-03-04：任务进入终态后生成结构化摘要（`task_id/task_type/goal/result/status/finished_at/tags`），并写入跨会话任务摘要记忆层。
  - 2026-03-04：默认注入窗口固定最近 5 条任务摘要，不注入日志与产物详情，控制上下文体积。
  - 2026-03-04：历史追问命中后触发深检索并支持按需下钻；仅在明确“细节/日志/报错原因/完整过程”语义时注入日志与产物摘要。
  - 2026-03-04：误触发保护生效（否定语义、当前任务优先、双信号门槛、预算截断与 miss 回退），并输出 `deep_retrieval_triggered`、`deep_retrieval_overridden`、`deep_retrieval_miss` 指标。
- 核心对象：`task_summary`、`task_id`、`session_id`、`history_query_intent`
- 关联需求：`R-018`、`R-025`、`R-030`
- 验证口径：默认注入窗口、深检索命中率、误触发率、详情下钻可用性、摘要轨与主数据轨一致性

### R-032 `.alter0` 任务历史存储规范与 Memory 查阅

1. 任务历史运行态数据统一落盘到 `.alter0/`，默认不纳入仓库提交，避免业务代码与运行数据混杂。
2. 目录规范：
   - `.alter0/tasks/index.json`：任务索引（`task_id`、`session_id`、`source_message_id`、`status`、`created_at`、`finished_at`）
   - `.alter0/tasks/{task_id}/meta.json`：任务主信息与状态机快照
   - `.alter0/tasks/{task_id}/logs.jsonl`：任务阶段日志（append-only）
   - `.alter0/tasks/{task_id}/artifacts.json`：产物引用（PR/文件/报告）
   - `.alter0/memory/YYYY-MM-DD.md`：任务摘要天级记忆
   - `.alter0/memory/long-term/YYYY-MM-DD.md`：长期候选摘要
3. 回链规范：摘要记录必须包含 `task_id`，可从 Memory 视图一跳回查任务详情；任务详情页必须反向展示其摘要引用。
4. 留存策略：
   - `logs.jsonl` 按任务保留（默认 90 天，可配置）
   - `index.json` 与 `meta.json` 保留任务生命周期全量索引
   - 天级摘要长期可读，超期归档不删除 `task_id` 映射
5. 前端 `Agent -> Memory` 查阅规范：
   - 默认展示任务摘要列表（按时间倒序）
   - 支持筛选 `status/task_type/time_range`
   - 支持按 `task_id` 打开详情抽屉，按需加载 `logs/artifacts`
   - 默认不展开原始日志，避免信息过载
6. 空态与异常态：
   - 无任务历史时显示“暂无任务历史”
   - 日志缺失或文件损坏时显示可重建提示，不影响摘要列表展示
7. 验收：`Memory` 页面可查阅 `.alter0` 中任务摘要并下钻详情；同一 `task_id` 可在摘要与日志间双向跳转，且运行态数据不进入 Git 变更。

#### 后端接口拆解（MVP）

1. 任务摘要列表接口
   - `GET /api/memory/tasks?status=&task_type=&start_at=&end_at=&page=&page_size=`
   - 返回：`items[]`（摘要字段）+ `pagination`（分页游标/总数）
2. 任务详情接口
   - `GET /api/memory/tasks/{task_id}`
   - 返回：`meta`（状态机快照）+ `summary_refs`（摘要回链）
3. 任务日志接口（按需加载）
   - `GET /api/memory/tasks/{task_id}/logs?cursor=&limit=`
   - 返回：增量日志片段，支持游标续读
4. 任务产物接口（按需加载）
   - `GET /api/memory/tasks/{task_id}/artifacts`
   - 返回：产物列表（类型、链接、摘要、生成时间）
5. 任务摘要重建接口（运维/修复）
   - `POST /api/memory/tasks/{task_id}/rebuild-summary`
   - 用于处理摘要轨与主数据轨不一致场景

#### 前端字段清单（MVP）

1. 摘要列表项字段
   - `task_id`、`task_type`、`goal`、`result`、`status`、`finished_at`、`tags`
2. 详情抽屉基础字段
   - `task_id`、`session_id`、`source_message_id`、`status`、`progress`、`created_at`、`finished_at`、`retry_count`
3. 日志视图字段
   - `seq`、`stage`、`level`、`message`、`created_at`
4. 产物视图字段
   - `artifact_type`、`uri`、`summary`、`created_at`
5. 交互状态字段
   - `isLoading`、`isLogLoading`、`hasMoreLogs`、`errorCode`、`activeTaskId`

#### Task Breakdown (auto-managed)

1. Scope split
   - Define `.alter0` runtime task storage layout and retention rules.
   - Define Memory module read model for summary-first and drill-down.
2. Delivery plan
   - M1: Runtime persistence schema and task-summary linking.
   - M2: Frontend Memory task-history browse and detail loading behavior.
3. Status workflow
   - Keep lifecycle as planned -> ready -> supported; update this row only.
4. Traceability
   - Keep implementation and verification records in this section after delivery.

#### Traceability

- 实现文件：`internal/storage/infrastructure/localfile/task_store.go`、`internal/tasksummary/application/runtime_markdown_store.go`、`internal/tasksummary/application/recorder_group.go`、`internal/interfaces/web/agent_memory.go`、`internal/interfaces/web/server.go`、`internal/interfaces/web/static/assets/chat.js`、`internal/interfaces/web/static/assets/chat.css`、`cmd/alter0/main.go`
- 测试覆盖：`internal/storage/infrastructure/localfile/task_store_test.go`、`internal/tasksummary/application/runtime_markdown_store_test.go`、`internal/interfaces/web/server_memory_task_test.go`
- 验证命令：
  - `GOTOOLCHAIN=auto GOSUMDB=sum.golang.org go test ./internal/storage/infrastructure/localfile ./internal/tasksummary/application ./internal/interfaces/web ./cmd/alter0`
  - `GOTOOLCHAIN=auto GOSUMDB=sum.golang.org go test ./...`
- 验证记录：
  - 2026-03-04：任务运行态落盘切换为 `.alter0/tasks/index.json + {task_id}/meta.json + logs.jsonl + artifacts.json`，并保持旧版 `tasks.json/tasks.md` 兼容迁移。
  - 2026-03-04：任务摘要在终态写入 `.alter0/memory/YYYY-MM-DD.md` 与 `.alter0/memory/long-term/YYYY-MM-DD.md`，摘要条目包含 `task_id` 与任务主数据回链路径。
  - 2026-03-04：新增 `GET /api/memory/tasks*` 与 `POST /api/memory/tasks/{task_id}/rebuild-summary`，支持摘要列表筛选、详情回链、日志/产物按需加载与摘要重建。
  - 2026-03-04：`Agent -> Memory` 默认展示任务摘要列表，支持 `status/task_type/time_range` 筛选，详情面板按需加载日志与产物，日志缺失时展示重建提示且不阻断摘要查看。
- 核心对象：`.alter0/tasks/*`、`.alter0/memory/*`、`task_id`
- 关联需求：`R-028`、`R-030`、`R-031`
- 验证口径：目录规范、摘要回链、前端可查阅、Git 隔离

### R-033 Control 任务观测台基础视图

1. 在 `Control` 一级导航新增 `Tasks` 页面，作为任务运行观测统一入口。
2. 提供任务列表与详情抽屉两层信息架构，避免在多个页面分散查找。
3. 列表支持基础筛选：`session_id`、`status`、`time_range`，默认按 `updated_at` 倒序。
4. 详情展示基础字段：`task_id`、`session_id`、`status`、`progress`、`retry_count`、`created_at`、`finished_at`、`error`。
5. 验收：运维人员可在 `Control -> Tasks` 单页完成任务检索与状态定位。

#### Traceability

- 核心对象：`task_id`、`session_id`、`status`、`progress`
- 依赖需求：`R-030`
- 验证口径：入口可用性、列表筛选正确性、详情字段完整性

### R-034 任务触发来源标识与溯源视图

1. 每个任务需展示触发来源字段：`trigger_type`（`user/cron/system`）、`channel_type`、`channel_id`、`correlation_id`。
2. 定时任务需额外展示调度来源字段：`job_id`、`job_name`、`fired_at`。
3. 列表支持 `trigger_type`、`channel_type` 过滤，便于区分人工触发与自动任务。
4. 验收：同一任务列表中可直接区分用户会话触发、定时任务触发及系统触发来源。

#### Traceability

- 核心对象：`trigger_type`、`channel_type`、`channel_id`、`correlation_id`、`job_id`
- 依赖需求：`R-030`、`R-032`
- 验证口径：来源字段覆盖率、过滤准确率、cron 溯源完整性

### R-035 任务日志流式观测与断线续读

1. 提供任务日志 SSE 流式接口，支持实时接收增量日志。
2. 提供日志回补接口，支持断线后基于 `cursor` 续读。
3. 前端首屏默认展示最近日志片段，滚动按需加载历史日志。
4. 验收：任务运行中页面可持续更新日志；网络短断后可恢复并不丢日志序列。

#### 接口拆分（草案）

1. 日志流接口
   - `GET /api/control/tasks/{task_id}/logs/stream?cursor=`
2. 日志回补接口
   - `GET /api/control/tasks/{task_id}/logs?cursor=&limit=`

#### Traceability

- 核心对象：`task_id`、`seq`、`cursor`、`logs`
- 依赖需求：`R-030`
- 验证口径：流式稳定性、续读正确性、日志顺序一致性

### R-036 任务控制动作与会话回链

1. 详情页提供 `retry`、`cancel` 控制能力，并显式展示可操作状态约束。
2. 约束规则：仅 `failed/canceled` 可 `retry`，仅 `queued/running` 可 `cancel`。
3. 提供任务到会话消息、会话消息到任务详情的双向跳转。
4. 验收：从失败任务可直接重试并观察新状态；从会话记录可定位对应任务执行轨迹。

#### 接口拆分（草案）

1. 控制动作接口
   - `POST /api/control/tasks/{task_id}/retry`
   - `POST /api/control/tasks/{task_id}/cancel`
2. 回链查询接口
   - `GET /api/control/tasks/{task_id}`（包含会话回链字段）
   - `GET /api/sessions/{session_id}/tasks?latest=true`

#### Traceability

- 核心对象：`task_id`、`session_id`、`retry`、`cancel`、`message_link`
- 依赖需求：`R-030`、`R-031`
- 验证口径：动作约束正确性、重试成功率、回链可达性

### R-037 Web 产物可访问交付层（替代本地路径暴露）

1. Web 对话回复中禁止使用本地绝对路径/相对路径作为交付地址；当生成文件成功时，回复需提供可访问的产物引用（`task_id`、`artifact_id`、`download_url`）。
2. 在现有产物列表接口基础上新增文件交付接口：
   - `GET /api/tasks/{task_id}/artifacts/{artifact_id}/download`
   - `GET /api/tasks/{task_id}/artifacts/{artifact_id}/preview`
3. 任务产物需落盘为稳定快照，避免直接暴露运行工作目录：
   - `.alter0/tasks/{task_id}/files/{artifact_id}`
   - `artifacts.json` 记录 `artifact_id`、`name`、`content_type`、`size`、`download_url`、`preview_url`。
4. 前端交互要求：
   - 聊天消息或任务详情中展示“下载/预览”入口；
   - 无可预览类型时仅展示下载；
   - 下载失败或产物缺失时给出明确错误提示，不显示本地路径。
5. 安全与边界：
   - 下载/预览接口仅允许访问任务快照目录，禁止路径穿越与软链接逃逸；
   - 限制单文件大小、单任务产物数量与接口读取超时，超限返回结构化错误码。
6. 验收：Web 用户在无本地文件系统权限场景下，仍可通过界面完成产物下载/预览；对话文本与接口响应均不泄露本机路径。

#### 接口拆分（草案）

1. 任务产物列表（扩展）
   - `GET /api/tasks/{task_id}/artifacts`
   - 返回字段补齐：`artifact_id`、`name`、`content_type`、`size`、`summary`、`download_url`、`preview_url`
2. 任务产物下载
   - `GET /api/tasks/{task_id}/artifacts/{artifact_id}/download`
   - 返回：二进制流 + `Content-Disposition`
3. 任务产物预览
   - `GET /api/tasks/{task_id}/artifacts/{artifact_id}/preview`
   - 返回：文本或浏览器可直接渲染内容；不支持类型返回 `artifact_preview_not_supported`

#### Traceability

- 核心对象：`task_id`、`artifact_id`、`download_url`、`preview_url`、`.alter0/tasks/{task_id}/files/*`
- 依赖需求：`R-030`、`R-032`
- 验证口径：路径不暴露、下载可达、预览可达、安全约束生效

### R-038 对话历史面板折叠能力

1. 在 `Chat` 页面会话侧栏中，为“最近会话”历史区提供显式折叠/展开控制（按钮或图标按钮）。
2. 折叠后仅隐藏历史列表内容，不影响 `New Chat`、当前会话标题与输入发送链路。
3. 展开后恢复原有历史列表交互能力（会话切换、删除、滚动浏览）。
4. 移动端与桌面端行为保持一致：均可折叠/展开，且不与现有侧栏开合逻辑冲突。
5. 折叠状态需在当前浏览器会话内保持一致，页面刷新后可恢复上次状态。
6. 验收：连续执行“折叠 -> 展开 -> 切换会话 -> 再折叠”流程，界面无错位，历史列表状态与交互一致。

#### Traceability

- 核心对象：`session-history-panel`、`session-history-toggle`、`collapsed_state`
- 依赖需求：`R-012`、`R-015`
- 验证口径：折叠可用性、展开恢复完整性、移动端一致性、状态保持

### R-039 页面滚动隔离（侧栏固定）

1. 在所有包含侧边栏的桌面端页面中，侧边栏（一级导航与二级面板）默认保持固定，不参与页面主滚动。
2. 主页面滚动仅作用于当前页面的主内容容器（如对话消息区、配置列表区、详情内容区），侧边栏不得随页面整体下移。
3. 侧边栏默认不出现滚动条；仅当视口高度受限或侧边栏内容超出可视区域时，允许侧边栏自身内部滚动。
4. 侧边栏内部滚动与主内容滚动必须相互隔离，避免滚轮/触控事件穿透导致双区域联动滚动。
5. `Chat` 页面继续保持“消息区滚动、输入区与头部可见”的现有行为，其它页面采用同样的侧栏固定与主内容滚动原则。
6. 移动端维持当前交互模型：抽屉开合、遮罩关闭与手势逻辑不受该需求破坏。
7. 验收：在 `Chat`、`Control`、`Agent` 等页面连续执行“主内容长列表滚动 -> 页面切换 -> 返回滚动”流程，侧边栏位置稳定；仅当侧栏内容溢出时侧栏内部可滚动。

#### Traceability

- 核心对象：`primary-nav`、`side-pane`、`content-area`、`chat-view`
- 依赖需求：`R-012`、`R-014`、`R-038`
- 验证口径：全站侧栏固定性、侧栏溢出滚动边界、滚动事件隔离、跨路由无回归

### R-040 Cron 可视化配置与触发会话归档

1. 在 `Control -> Cron Jobs` 提供可视化配置表单，支持以下调度模式：
   - 每隔 N 天/小时/分钟执行
   - 每天固定时间执行（如 `09:30`）
   - 每周固定星期与时间执行（如每周一 `10:00`）
2. 可视化配置与 cron 表达式双向联动：用户修改可视化选项时实时展示表达式；用户直接编辑表达式时可回填可识别字段。
3. Cron 配置需包含任务参数配置区：任务名称、提示词/输入内容、是否启用、时区、重试策略（最小集）。
4. 每个 Cron Job 触发执行时，系统必须创建“专属新会话”（不可复用空白会话），并在会话元数据记录来源字段：`trigger_type=cron`、`job_id`、`fired_at`。
5. 用户可在会话历史中查看 Cron 触发产生的会话，并可按 `job_id` 或来源类型筛选，支持从 Cron 任务详情跳转到对应会话。
6. 验收：创建 1 个每日定时任务并触发执行后，可见新会话自动生成，且在会话历史中可追溯到该 Cron Job 与触发时间。

#### 接口拆分（草案）

1. Cron 配置管理（扩展）
   - `GET /api/control/cron/jobs`
   - `PUT /api/control/cron/jobs/{job_id}`
   - 字段扩展：`schedule_mode`、`timezone`、`cron_expression`、`task_config`
2. Cron 触发记录与会话回链
   - `GET /api/control/cron/jobs/{job_id}/runs`
   - 返回字段：`run_id`、`job_id`、`fired_at`、`session_id`、`status`
3. 会话筛选（扩展）
   - `GET /api/sessions?trigger_type=cron&job_id=...`

#### Traceability

- 核心对象：`cron_expression`、`schedule_mode`、`job_id`、`fired_at`、`session_id`
- 依赖需求：`R-009`、`R-010`、`R-034`
- 验证口径：可视化配置可用性、表达式联动正确性、触发会话可追溯性
