# Requirements Details (R-021 ~ R-030)

> Last update: 2026-03-27

## 需求细化（草案）

### R-021 Skills/MCP 标准化能力模型

1. 定义统一能力模型：`id`、`name`、`type`（skill/mcp）、`enabled`、`scope`、`metadata`。
2. 提供一致的配置校验与版本规则，避免不同能力类型出现结构漂移。
3. 支持标准化生命周期动作（启用、禁用、更新、删除）与变更审计。
4. 验收：同一套 API/存储约束可同时管理 Skills 与 MCP 能力对象。

#### Traceability

- 实现文件：`internal/control/domain/capability.go`、`internal/control/application/service.go`、`internal/storage/infrastructure/localfile/control_store.go`、`internal/interfaces/web/server.go`
- 前端展示：`internal/interfaces/web/static/assets/chat.js`
- 验证：`go test ./...`

### R-022 用户配置 Skills 接入 Codex

1. 根据用户配置筛选启用的 Skills，并按优先级注入 Codex 与 Agent 执行上下文。
2. 定义 Skills 注入协议（名称、描述、guide、参数模板、约束）以避免自由文本拼接。
3. 支持内置 Skill 作为稳定能力底座；当前默认提供 `default-nl` 与 `memory`，其中 `memory` 负责明确记忆文件的读取决策、写入路由、冲突优先级与写入禁止项。
4. 提供冲突处理策略（同名 skill、重复能力、参数冲突）并可观测。
5. 验收：关闭某 Skill 后不再注入；开启后可在 Codex / Agent 执行结果中体现其能力影响。

#### Traceability

- 实现文件：`internal/execution/domain/skill_context.go`、`internal/execution/application/skill_context_resolver.go`、`internal/execution/application/service.go`、`internal/execution/infrastructure/codex_cli_processor.go`、`internal/orchestration/application/service.go`、`cmd/alter0/builtin_skills.go`、`cmd/alter0/main.go`
- 测试覆盖：`internal/execution/application/service_test.go`、`internal/execution/infrastructure/codex_cli_processor_test.go`、`internal/orchestration/application/service_test.go`
- 协议约束：`alter0.skill-context/v1`，注入结构固定为 `protocol`、`skills[].name/description/guide/parameter_template/constraints`、`resolved_parameters`、`conflicts`；通过 `alter0.skill_context` 元数据注入，禁止自由文本拼接。
- 验证命令：`GOSUMDB=sum.golang.org GOTOOLCHAIN=auto go test ./...`
- 验证记录：
  - 2026-03-03：仅注入 `enabled=true` 的 Skills，按 `skill.priority` 降序注入；关闭 Skill 后 `skills.injected_count` 与注入上下文立即收敛。
  - 2026-03-03：同名 Skill、重复能力、参数值冲突按优先级保留高优先级项，冲突明细写入 `skills.conflicts` 与 `skills.conflict_types`。
  - 2026-03-03：Codex 执行参数切换为结构化 JSON 载荷（`alter0.codex-exec/v1`），执行结果元数据可观测 `skills.injected_ids`、`skills.injected_count`、`skills.conflict_count`。
  - 2026-03-27：Skill 协议新增 `guide` 字段，可承载独立操作说明；默认内置 `memory` Skill，用于向 Agent / Codex 说明 alter0 记忆模块、文件职责与读写规则。
  - 2026-03-27：`memory` Skill guide 收敛为显式操作规则，按“运行时契约 / 读取逻辑 / 写入路由 / 冲突规则 / 写入约束”组织，直接约束不同类型信息应落入哪个记忆文件。

### R-023 用户配置 MCP 接入 Codex

1. 将用户配置的 MCP Server 映射到 Codex 可识别配置，支持 stdio 与 HTTP 两类传输。
2. 支持按会话或请求粒度启用 MCP，避免全局污染与越权调用。
3. 对 MCP 接入过程补齐安全控制：白名单、超时、失败隔离、访问审计。
4. 验收：用户新增 MCP 配置后，可在指定会话中被 Codex 正确调用，且日志可追踪。

#### Traceability

- 实现文件：`internal/execution/domain/mcp_context.go`、`internal/execution/application/mcp_context_resolver.go`、`internal/execution/application/service.go`、`internal/execution/infrastructure/codex_cli_processor.go`
- 测试覆盖：`internal/execution/application/service_test.go`、`internal/execution/infrastructure/codex_cli_processor_test.go`
- 协议约束：`alter0.mcp-context/v1`，注入结构固定为 `protocol`、`servers[].transport/command/args/url/headers/tool_whitelist/timeout_ms/failure_isolation`、`audit[]`；通过 `alter0.mcp_context` 元数据注入，执行载荷统一为 `alter0.codex-exec/v1`。
- 验证命令：`GOSUMDB=sum.golang.org GOTOOLCHAIN=auto go test ./...`
- 验证记录：
  - 2026-03-03：`mcp.transport` 支持 `stdio/http` 映射，分别输出 `command/args/env` 与 `url/headers` 到 Codex 执行载荷。
  - 2026-03-03：会话级启用通过 `alter0.mcp.session.enable` 绑定 `session_id`，请求级启用通过 `alter0.mcp.request.enable` 按次生效，跨会话不共享。
  - 2026-03-03：`mcp.tools.whitelist`、`mcp.timeout_ms`、`mcp.failure_isolation` 生效；异常配置按 Server 级隔离并写入 `mcp.audit`。
  - 2026-03-03：执行结果元数据新增 `mcp.injected_ids`、`mcp.injected_count`、`mcp.audit_count`、`mcp.audit`，日志输出 `mcp resolved` 支持访问审计追踪。

### R-024 跨会话持久化记忆分级管理（参考 CPU Cache）

1. 采用类缓存分层模型：`L1` 作为热记忆层（高价值、低延迟、最小容量），`L2` 作为平衡层（中等容量与时效），`L3` 作为归档层（大容量、低频访问）。
2. 每层独立配置约束：单条长度上限、层总容量上限、保留时长（TTL）、淘汰策略（优先 LRU，可扩展评分淘汰）。
3. 查询注入遵循缓存命中路径：优先命中 `L1`，再查 `L2/L3`；在 token 预算不足时按 `L1 > L2 > L3` 截断。
4. 支持“缓存迁移”机制：高频命中或被显式标记的重要记忆可晋升（`L3 -> L2 -> L1`）；长期未命中项可降级。
5. 支持写入策略配置（如 write-through / write-back 风格）：保证持久化一致性与性能平衡。
6. 验收：跨会话与重启后记忆可恢复；命中链路与晋升降级可观测；在相同 token 预算下高价值记忆命中率提升。

#### Traceability

- 实现文件：`internal/orchestration/application/long_term_memory.go`、`internal/orchestration/application/mandatory_context.go`、`cmd/alter0/main.go`
- 测试覆盖：`internal/orchestration/application/long_term_memory_test.go`、`internal/orchestration/application/tiered_long_term_memory_acceptance_test.go`、`internal/orchestration/application/service_test.go`
- 运行参数：
  - `-long-term-memory-path`
  - `-long-term-memory-write-policy`
  - `-long-term-memory-writeback-flush`
  - `-long-term-memory-token-budget`
- 验证命令：`GOSUMDB=sum.golang.org GOTOOLCHAIN=auto go test ./...`
- 验证记录：
  - 2026-03-03：长期记忆升级为 `L1/L2/L3` 分层模型，支持分层单条长度、总容量、TTL 与 `LRU/score` 淘汰策略。
  - 2026-03-03：查询注入固定命中链路 `L1 -> L2 -> L3`，并在 token 预算不足时按层级优先级截断，输出链路与预算元数据。
  - 2026-03-03：命中反馈支持高频晋升与长期未命中降级，显式 `memory_long_term_important=true` 可直接进入高优先层，晋升/降级计数可观测。
  - 2026-03-03：支持 `write_through/write_back` 写入策略与重启恢复，验收测试确认同等预算下高价值记忆命中率提升。

### R-025 天级记忆与长期记忆（Markdown 统一存储）

1. 支持天级记忆落盘：按日期切分存储（例如 `memory/YYYY-MM-DD.md`），记录当日会话沉淀与关键事实。
2. 支持长期记忆存储：将跨天稳定事实与偏好沉淀到长期记忆文件（例如 `MEMORY.md` 或 `memory/long-term/*.md`）。
3. 支持按天压缩：每日或定时将当日长对话压缩为摘要与关键事实，再回写到天级文件与长期记忆候选区。
4. 统一存储格式：包括 R-017~R-024 在内的记忆存储均使用 Markdown（`.md`），禁止引入独立二进制记忆格式。
5. 支持分级容量约束：按 L1/L2/L3 分别配置天级条目长度上限、层容量上限与保留时长。
6. 验收：连续多天运行后可按日期检索历史；当天压缩任务可稳定执行；长期记忆可跨会话命中且 Markdown 文件可直接审阅。

#### Traceability

- 实现文件：`internal/orchestration/application/session_memory.go`、`internal/orchestration/application/session_memory_daily_markdown.go`、`internal/orchestration/application/long_term_memory.go`、`internal/orchestration/application/memory_markdown_codec.go`、`cmd/alter0/main.go`
- 测试覆盖：`internal/orchestration/application/session_memory_test.go`、`internal/orchestration/application/long_term_memory_test.go`、`internal/orchestration/application/daily_long_term_memory_markdown_acceptance_test.go`
- 运行参数：
  - `-daily-memory-dir`
  - `-long-term-memory-path`
- 验证命令：`GOSUMDB=sum.golang.org GOTOOLCHAIN=auto go test ./internal/orchestration/application`
- 验证记录：
  - 2026-03-03：会话记忆新增天级 Markdown 落盘，按 `memory/YYYY-MM-DD.md` 记录 L1/L2/L3 分层内容、摘要与关键事实。
  - 2026-03-03：上下文压缩片段在当日自动回写到天级文件，并同步生成 `memory/long-term/YYYY-MM-DD.md` 长期记忆候选。
  - 2026-03-03：长期记忆持久化由 `.json` 升级为 `MEMORY.md` Markdown 存储，重启后可恢复并继续跨会话命中。
  - 2026-03-03：天级记忆支持分层单条长度、层容量与保留时长约束，超限或过期条目按层级策略自动清理。

### R-026 强制要求上下文文件（如 SOUL.md）

1. 支持独立的“强制要求”上下文文件（例如 `SOUL.md`），用于存储用户不可违背的长期指令与偏好。
2. 启动与会话初始化时优先加载该文件，并在请求构造时高优先级注入上下文。
3. 支持热更新：文件变更后可在后续请求中生效，并记录版本/更新时间用于审计。
4. 该文件与记忆体系隔离：不参与自动摘要压缩，不被 L1/L2/L3 淘汰策略清理。
5. 支持冲突处理规则：当强制要求与普通记忆冲突时，以强制要求文件为准并产生日志告警。
6. 验收：修改 `SOUL.md` 后新会话可立即体现强制要求；跨会话重启后仍稳定生效。

#### Traceability

- 实现文件：`internal/orchestration/application/mandatory_context.go`、`internal/orchestration/application/service.go`、`cmd/alter0/main.go`
- 测试覆盖：`internal/orchestration/application/mandatory_context_test.go`、`internal/orchestration/application/service_test.go`
- 验证命令：`GOSUMDB=sum.golang.org GOTOOLCHAIN=auto go test ./...`
- 验证记录：
  - 2026-03-03：启动参数新增 `-mandatory-context-file`，默认读取 `SOUL.md`，会话初始化时优先注入 `[MANDATORY CONTEXT]` 到请求构造。
  - 2026-03-03：上下文文件支持热更新，后续请求自动加载最新内容并输出 `mandatory_context.version`、`mandatory_context.updated_at`、`mandatory_context.loaded_at` 审计元数据。
  - 2026-03-03：与会话/长期记忆冲突时按键规则裁决，冲突记忆被过滤或覆盖，结果元数据记录 `mandatory_context.conflict_count` 并输出告警日志。
  - 2026-03-03：强制要求上下文独立于普通记忆注入链路，标记 `mandatory_context.isolated=true`，不进入记忆摘要与分层淘汰路径。

### R-027 Agent Memory 模块与页面收敛

1. 页面收敛：前端导航与路由中移除 `Workspace`、`Configuration` 两个页面入口，避免与 Agent 视图并行造成信息割裂。
2. Agent 模块扩展：在 `Agent` 下新增 `Memory` 页面入口，作为统一记忆可视化界面。
3. `Memory` 页面展示三类数据：
   - 长期记忆：展示跨会话长期记忆（含分层信息与更新时间）。
   - 天级记忆：按日期展示日记忆文件（如 `memory/YYYY-MM-DD.md`）及摘要。
   - 持久化记忆：展示 `SOUL.md` 当前生效内容与最近更新时间。
4. 交互要求：支持在同一页面完成三类记忆切换查看；移动端与桌面端均可达且不依赖隐藏入口。
5. 边界约束：`Memory` 页面默认只读展示，不在本需求内引入在线编辑能力。
6. 验收：侧边栏不再出现 `Workspace`、`Configuration`；`Agent -> Memory` 可稳定展示三类记忆数据，空态与加载失败态均有明确提示。

#### Task Breakdown (auto-managed)

1. Scope split
   - Remove `Workspace`/`Configuration` route and menu entries.
   - Introduce `Agent -> Memory` information architecture and rendering boundaries.
2. Delivery plan
   - M1: Navigation and routing convergence.
   - M2: Memory visualization for long-term, daily and `SOUL.md`.
3. Status workflow
   - Keep lifecycle as planned -> ready -> supported; update this row only.
4. Traceability
   - Keep implementation and verification records in this section after delivery.

### R-028 Memory 模块说明文档持久化与可视化

1. 新增记忆体系说明文档 `docs/memory/persistent-memory-module-spec.md`，作为项目内长期保留的产品与工程基线文档。
2. 文档明确五类核心文件职责与边界：`USER.md`（用户画像）、`AGENTS.md`（代理执行规范）、`MEMORY.md`（长期记忆）、`memory/YYYY-MM-DD.md`（天级记忆）、`SOUL.md`（强制上下文）。
3. 文档定义“持久化可见”原则：文档本身纳入仓库版本管理；运行时记忆数据继续落盘到 `.alter0/memory/*`，不并入业务代码提交。
4. `Agent -> Memory` 页面补充“说明文档”视图入口，支持稳定展示该文档全文，并可关联到上述文件类型的说明锚点。
5. 页面展示边界：本需求仅覆盖只读展示、结构化分段、空态与加载失败态，不引入在线编辑。
6. 验收：在同一页面可切换查看长期记忆、天级记忆、持久化记忆与说明文档四类视图，且说明文档内容与仓库文件保持一致。

#### Task Breakdown (auto-managed)

1. Scope split
   - Persist and version a dedicated memory module specification document.
   - Extend Memory module IA with a read-only `Documentation` view.
2. Delivery plan
   - M1: Finalize document schema, file responsibilities, and persistence boundaries.
   - M2: Align frontend Memory view contract with document visibility requirements.
3. Status workflow
   - Keep lifecycle as planned -> ready -> supported; update this row only.
4. Traceability
   - Keep implementation and verification records in this section after delivery.

#### Traceability

- 说明文档：`docs/memory/persistent-memory-module-spec.md`
- 关联需求：`R-017`、`R-018`、`R-025`、`R-026`、`R-027`
- 可视化目标：`Agent -> Memory`
- 实现文件：`internal/interfaces/web/agent_memory.go`、`internal/interfaces/web/server.go`、`internal/interfaces/web/static/assets/chat.js`
- 测试覆盖：`internal/interfaces/web/server_memory_test.go`
- 验证记录：
  - 2026-03-04：`Agent -> Memory` 已支持说明文档可视化与只读展示，`R-028` 状态回填为 `supported`。

### R-029 新对话空白会话唯一性约束

1. 定义“空白会话”：会话内没有任何用户消息与助手消息（`messages.length == 0`）。
2. `New Chat` 行为约束：当存在任意空白会话时，不得创建新会话；必须复用最近创建的空白会话并切换为活动会话。
3. 仅当当前不存在空白会话时，`New Chat` 才允许创建新的会话记录。
4. 会话删除或首条消息写入后，空白会话状态实时更新，后续 `New Chat` 按最新状态判定。
5. 页面刷新后规则保持一致：本地恢复会话列表后仍遵守“最多一个空白会话”的约束。
6. 验收：连续点击 `New Chat` 5 次，最多只保留 1 个空白会话；发送首条消息后再次点击 `New Chat` 才会生成下一空白会话。

#### Task Breakdown (auto-managed)

1. Scope split
   - Define blank-session detection and uniqueness contract.
   - Apply uniqueness rule to all New Chat entry points.
2. Delivery plan
   - M1: Frontend creation guard and focus reuse behavior.
   - M2: Recovery-path consistency after local persistence reload.
3. Status workflow
   - Keep lifecycle as planned -> ready -> supported; update this row only.
4. Traceability
   - Keep implementation and verification records in this section after delivery.

#### Traceability

- 目标模块：`internal/interfaces/web/static/assets/chat.js`
- 关联需求：`R-015`、`R-019`
- 实现文件：`internal/interfaces/web/static/assets/chat.js`
- 测试覆盖：`internal/interfaces/web/server_mobile_session_test.go`
- 验证口径：`New Chat` 连续触发、刷新恢复、首条消息后再创建
- 验证记录：
  - 2026-03-04：空白会话唯一性规则已生效，`R-029` 状态回填为 `supported`。

### R-030 会话与异步任务映射模型

1. 建立统一映射关系：一个 `session` 可关联多个 `task`（`1:N`），每个任务必须持有 `session_id` 与 `source_message_id`。
2. 新增异步判定门槛：预计耗时超过阈值或属于产物型/多步骤流程时，消息路由转为后台任务执行。
3. 对话层快速应答：命中异步后立即返回 `task_id` 与初始状态（`queued/running`），不阻塞会话响应。
4. 任务状态机标准化：`queued -> running -> success/failed/canceled`，并支持进度、重试、超时、取消。
5. 任务日志与产物持久化：后台执行写入阶段日志、错误原因、产物引用；对话模块可按 `task_id` 回读并生成用户可读摘要。
6. 结果回流规则：任务完成后将摘要结果回写原 `session`，保留 `task_id` 与消息关联，支持“查任务进度/查任务结果”指令。
7. 验收：同一会话连续发起多个长任务时，会话可持续交互；每个任务可独立查询状态、日志和最终结果，且不会导致会话上下文异常膨胀。

#### Task Breakdown (auto-managed)

1. Scope split
   - Define session-task data model and id mapping contract.
   - Define async routing trigger and response contract.
2. Delivery plan
   - M1: Task lifecycle APIs and persistence model.
   - M2: Chat readback flow and completion writeback behavior.
3. Status workflow
   - Keep lifecycle as planned -> ready -> supported; update this row only.
4. Traceability
   - Keep implementation and verification records in this section after delivery.

#### 接口拆分（草案）

1. 任务创建与快速应答
   - `POST /api/tasks`
   - 入参：`session_id`、`source_message_id`、`task_type`、`input`、`idempotency_key`、`async_hint`
   - 出参：`task_id`、`status`、`accepted_at`、`estimated_wait_ms`
   - 约束：同一 `idempotency_key` 重复提交时返回已存在任务，不重复创建。
2. 任务查询与列表
   - `GET /api/tasks/{task_id}`
   - `GET /api/tasks?session_id=&status=&page=&page_size=`
   - 能力：返回状态机节点、进度百分比、开始/结束时间、错误摘要。
3. 任务日志与产物查询
   - `GET /api/tasks/{task_id}/logs?cursor=&limit=`
   - `GET /api/tasks/{task_id}/artifacts`
   - 能力：按游标增量读取日志，产物返回可审计引用（文件路径、PR 链接、报告链接）。
4. 任务控制动作
   - `POST /api/tasks/{task_id}/cancel`
   - `POST /api/tasks/{task_id}/retry`
   - 约束：仅 `queued/running` 可取消，仅 `failed/canceled` 可重试。
5. 会话回读接口
   - `GET /api/sessions/{session_id}/tasks?latest=true`
   - 对话请求命中异步判定时，原会话响应必须附带 `task_id` 与 `task_status`，前端据此跳转任务详情或轮询。
6. 会话删除联动
   - `DELETE /api/sessions/{session_id}`
   - 删除会话时必须同步清理该 `session_id` 下所有任务记录、任务产物快照与 `.alter0/workspaces/sessions/<session_id>` 下的任务工作区，不保留孤儿目录或孤儿索引。

#### 数据模型拆分（草案）

- `Task`
  - 关键字段：`task_id`、`session_id`、`source_message_id`、`task_type`、`status`、`progress`、`retry_count`、`timeout_at`
- `TaskLog`
  - 关键字段：`task_id`、`seq`、`stage`、`level`、`message`、`created_at`
- `TaskArtifact`
  - 关键字段：`task_id`、`artifact_type`、`uri`、`summary`、`created_at`
- `TaskMessageLink`
  - 关键字段：`task_id`、`session_id`、`request_message_id`、`result_message_id`

#### Traceability

- 核心对象：`session_id`、`task_id`、`source_message_id`
- 关联需求：`R-016`、`R-019`、`R-029`
- 验证口径：异步判定、快速应答、任务查询、结果回流
- 实现文件：`internal/task/domain/task.go`、`internal/task/application/service.go`、`internal/interfaces/web/server.go`
- 测试覆盖：`internal/task/application/service_test.go`、`internal/interfaces/web/server_task_test.go`
- 新增接口：
  - `POST /api/tasks`
  - `GET /api/tasks?session_id=&status=&page=&page_size=`
  - `DELETE /api/sessions/{session_id}`
  - `GET /api/tasks/{task_id}/logs?cursor=&limit=`
  - `GET /api/tasks/{task_id}/artifacts`
  - `POST /api/tasks/{task_id}/retry`
- 验证命令：
  - `GOTOOLCHAIN=auto GOSUMDB=sum.golang.org go test ./internal/task/... ./internal/interfaces/web/...`
  - `GOTOOLCHAIN=auto GOSUMDB=sum.golang.org go test ./...`
- 验证记录：
  - 2026-03-04：任务主模型补齐 `source_message_id/task_type/timeout_at/message_link`，同一 `session_id` 下保持 `1:N` 任务映射并支持 `idempotency_key` 去重。
  - 2026-03-04：异步任务状态机覆盖 `queued -> running -> success/failed/canceled`，补齐重试、超时、取消转移与阶段日志序号化持久化。
  - 2026-03-04：会话链路支持快速应答返回 `task_id/task_status`，任务完成后摘要回写原会话并保留 `task_id -> request_message_id/result_message_id` 回链。
  - 2026-03-04：任务回读接口支持按会话筛选、按 `task_id` 查询详情、游标日志读取、产物读取与失败任务重试。
  - 2026-03-30：新增会话级联删除，`DELETE /api/sessions/{session_id}` 会同步移除任务索引、任务目录与会话工作区。
