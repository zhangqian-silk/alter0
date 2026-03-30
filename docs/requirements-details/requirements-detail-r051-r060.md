# Requirements Details (R-051 ~ R-060)

> Last update: 2026-03-30

## 需求细化（草案）

### R-051 Terminal 会话持久标识与超时恢复

1. Terminal 模块必须持久化存储 Codex CLI 会话标识，并与平台内 `terminal_session_id` 建立稳定映射；页面刷新、服务重启或运行态退出后，该标识仍可用于恢复会话。
2. 若当前实现中的 `terminal_session_id` 已实际承载 Codex CLI 会话标识，则需将其作为持久恢复主键稳定保存；若两者语义不同，则需新增独立字段保存 Codex CLI 标识，并保持映射关系可查询。
3. Terminal 会话持久化内容至少包含：`terminal_session_id`、Codex CLI 会话标识、标题、工作目录、创建时间、最近活动时间、最近一次运行状态、关联日志或步骤索引。
   - 当前实现将 Terminal 会话状态持久化到 `.alter0/state/terminal/sessions/<terminal_session_id>.json`，保留会话摘要、Codex CLI 线程标识、输出日志与步骤视图索引；终端每个会话使用独立工作目录 `.alter0/workspaces/terminal/sessions/<terminal_session_id>`。
4. Terminal 模块不再设置产品级会话上限；前端、接口与配置项中不再以“最大会话数”为用户约束，不再因达到上限拒绝创建或恢复 Terminal 会话。
5. Terminal 模块不再设置产品级会话超时淘汰规则；系统不因固定超时阈值自动销毁 Terminal 会话记录、历史消息、日志索引或 Codex CLI 标识映射。
6. 若底层 Codex CLI 运行因空闲、网络、进程退出或外部超时而中断，系统仅将该 Terminal 会话标记为已退出、已中断或待恢复状态，不删除会话本身，不清空历史，不重置标识映射。
   - 用户显式执行 `Delete` 时除外：系统需直接删除 Terminal 会话主记录与 `.alter0/state/terminal/sessions/<terminal_session_id>.json`，不再保留恢复入口，也不应先把会话改写成 `exited` 等过渡状态；该 Terminal 会话对应的独立工作区需一并删除。
7. 恢复要求：用户可对已退出或超时中断的 Terminal 会话执行恢复；恢复时系统优先复用已持久化的 Codex CLI 会话标识继续接续原会话，无法直接接续时需返回明确原因，并允许在保留同一 Terminal 会话历史的前提下重新建立运行态。
8. 输入要求：当用户向已退出或超时中断的 Terminal 会话继续发送输入时，系统可自动触发恢复流程，或先给出明确恢复提示并由用户确认后恢复，但不得强制用户新建独立会话。
   - 当前实现采用“继续发送即恢复”的交互方式；前端保留原会话，直接对已退出或已中断会话重新提交输入。
   - 当 Terminal 会话仅创建未发送首条输入、但底层运行态已不存在时，前端在首次发送时先调用恢复接口重建同一 `terminal_session_id`，随后自动重试当前输入。
   - 同一 Web 登录态下，Terminal 会话列表与历史回放必须跨设备共享；手机与 PC 访问同一套会话记录，不再按浏览器 client 标识分桶。
   - 恢复与继续发送仅以服务端持久化的 `terminal_session_id` 与 CLI 线程标识为准，不依赖设备侧 client 标识一致性。
9. 展示要求：Terminal 页面需明确区分“会话历史仍在”与“当前运行态已退出”两种状态，避免把运行态超时误展示为会话丢失或不可恢复。
   - 同一 Terminal 会话在同一次运行态中断、退出或宿主侧取消期间，只允许写入一条对应状态提醒；恢复后若再次进入新的中断周期，才补充新的提醒记录。
10. 兼容性要求：已有 Terminal 会话数据在升级后需可继续读取；若历史数据缺少 Codex CLI 会话标识，系统需允许降级为不可直连恢复但可保留历史，并支持后续重新绑定新的运行态。
11. 验收：Terminal 页面创建多个会话后不再因会话数触发上限错误；某一会话运行超时或中断后，历史记录仍可见，用户可对原会话执行恢复并继续输入，且恢复后仍能回看原有日志与上下文链路。
   - 补充验收：新建后尚未发送首条输入的 Terminal 会话即使因空闲导致运行态缺失，用户首次发送仍在原会话内成功执行，不出现“terminal session not found”阻断。
   - 补充验收：同一 Web 登录态下，手机与 PC 进入 Terminal 页面时能看到同一批会话历史；任一端创建、恢复或删除会话后，另一端刷新即可看到一致结果。
   - 补充验收：用户显式删除某个 Terminal 会话后，该会话不再可恢复，相关状态文件与对应独立工作区同步清理。
   - 补充验收：执行中的 Terminal 会话在服务关闭、宿主取消或运行态中断时，历史中只保留一条对应的中断提醒，不重复刷入相同提示。

#### 实施边界（草案）

1. 本需求聚焦 Terminal 会话生命周期治理，不调整普通 Chat/Agent 会话模型。
2. 本需求取消的是产品级 Terminal 会话上限与超时淘汰策略，不等同于取消底层网络超时、代理超时或外部 CLI 自身超时现象。
3. 当底层 Codex CLI 已彻底失效且无法基于原标识恢复时，系统可重建运行态，但需保持 Terminal 会话主记录、历史输出与恢复失败原因可见。
4. 本需求不要求无限保留底层进程常驻；允许运行态退出，但不允许因此丢失 Terminal 会话身份与恢复入口。
5. 本需求取消的是 Terminal 历史与工作目录的设备级隔离；不涉及不同 Web 登录态之间的历史合并。

#### 接口拆分（草案）

1. Terminal 会话查询
   - `GET /api/terminal/sessions`
   - 返回字段补充：Codex CLI 会话标识、可恢复状态、最近退出原因、最近恢复时间
2. Terminal 会话恢复
   - `POST /api/terminal/sessions/recover`
   - 入参至少包含：`terminal_session_id` 与已持久化的 Codex CLI 会话标识或其可解析映射
   - 返回：恢复后的运行状态、是否复用原会话标识、失败原因
3. Terminal 输入续写
   - `POST /api/terminal/sessions/{terminal_session_id}/input`
   - 当会话处于已退出或待恢复状态时，允许自动恢复或显式恢复后继续输入
4. Terminal 会话删除
   - `DELETE /api/terminal/sessions/{terminal_session_id}`
   - 行为：直接删除成功时返回 `204 No Content`；前端据此立即移除会话，不依赖返回一个已改写状态的会话对象
5. Terminal 配置与约束
   - 不再暴露 `task_terminal_max_sessions` 或等价 Terminal 会话上限配置为用户侧可调能力

#### Traceability

- 核心对象：`terminal_session_id`、`codex_cli_session_id`、`recoverable_state`、`runtime_status`、`terminal_logs`
- 依赖需求：`R-019`、`R-035`、`R-046`
- 验证口径：标识持久化完整性、超时后历史保留完整性、恢复成功率、恢复失败可解释性、无会话上限拦截
- 验证记录：
- 2026-03-30：Terminal 会话历史在同一 Web 登录态下改为跨设备共享；工作区仍按 `terminal_session_id` 独立隔离到 `.alter0/workspaces/terminal/sessions/<terminal_session_id>`。
  - 2026-03-30：Terminal 历史在同一 Web 登录态下改为跨设备共享，服务端不再按浏览器 client 标识隔离会话列表与详情访问。
  - 2026-03-30：Terminal 会话在单次运行态中断周期内收敛为单条中断提醒；服务关闭与执行协程收尾同时触发时不再重复写入相同状态记录。

### R-052

1. `Agent Profile` 配置页必须新增 `Memory Files` 勾选区，与 `Skills`、`MCP` 采用同级勾选交互。
2. 当前支持的选择项至少包括：`USER.md`、`SOUL.md`、`AGENTS.md`、长期 `MEMORY.md / memory.md`、`Daily Memory (Today)`、`Daily Memory (Yesterday)`。
3. Agent 请求命中对应 Profile 后，服务端必须将勾选结果写入统一运行时元数据，并在执行前解析为结构化 `memory_context`。
4. `memory_context` 最小字段包括：`protocol`、`files[].id`、`files[].selection`、`files[].title`、`files[].path`、`files[].exists`、`files[].writable`、`files[].updated_at`、`files[].content`。
5. 文件内容注入需要保留可写文件路径；当目标文件不存在时，仍需返回预期路径与 `exists=false`，允许 Agent 后续直接创建并写入。
6. Agent 执行链路必须同时支持两类消费方式：
   - ReAct Agent：在 system prompt 中显式暴露已选记忆文件的路径、存在状态与内容。
   - Codex 执行链：在 `alter0.codex-exec/v1` 载荷中新增 `memory_context` 字段，保证 fallback 与 `codex_exec` 一致可见。
7. 所选记忆文件默认视为可维护对象，Agent 可继续通过 `read`、`write`、`edit` 工具直接更新这些文件，无需额外专用写入接口。
8. 长期记忆与日记忆路径需兼容 `alter0` 当前目录约定，同时允许对齐 OpenClaw 常见文件名：
   - 长期记忆优先识别 `MEMORY.md`、`memory.md`、`.alter0/memory/long-term/MEMORY.md`
   - 日记忆优先识别 `memory/YYYY-MM-DD.md` 与 `.alter0/memory/YYYY-MM-DD.md`
9. 为控制 prompt 体积，单文件与总注入体积必须设置截断上限；截断后保留显式标记，避免模型误以为内容完整。
10. 验收：
   - Web `Agent Profiles` 页面可稳定保存和回显 `memory_files`
   - `POST /api/agent/messages` 会将勾选结果注入为 `alter0.memory.include`
   - 执行器可生成 `alter0.memory-context/v1`
   - ReAct 与 Codex 两条链路都能看到相同记忆文件集
   - 文件不存在时 Agent 仍能拿到目标路径并通过原生工具创建
11. 默认提供独立 `memory` Skill，可与 `memory_files` 同时启用；该 Skill 负责向 Agent / Codex 说明记忆模块、文件职责、读写边界与读写时机，具体文件内容仍以 `memory_context` 为准。
12. Skill 协议需支持可选文件型属性，至少包括：`skills[].file_path`、`skills[].writable`；当 Skill 绑定可维护规则文件时，Agent 可结合专用工具读取或更新该文件，而不必退回通用文件路径猜测。

#### Traceability

- 实现文件：`internal/control/domain/agent.go`、`internal/interfaces/web/server.go`、`internal/interfaces/web/static/assets/chat.js`
- 执行注入：`internal/execution/domain/memory_context.go`、`internal/execution/application/memory_context_resolver.go`、`internal/execution/application/service.go`
- 执行消费：`internal/execution/infrastructure/hybrid_nl_processor.go`、`internal/execution/infrastructure/codex_cli_processor.go`
- 测试覆盖：`internal/control/domain/agent_test.go`、`internal/execution/application/service_test.go`、`internal/execution/infrastructure/codex_cli_processor_test.go`、`internal/interfaces/web/server_message_test.go`

### R-053

1. 运行时必须提供统一 `Agent Catalog`，同时聚合系统内置 Agent 与控制面管理的 Agent Profile。
2. 内置 Agent 至少包括：
   - `main`：默认主 Agent `Alter0`，负责通用对话入口与子 Agent 调度
   - `coding`：专项编码 Agent，负责理解用户开发目标、保持交互收口，并通过 `codex_exec` 多轮推进具体实现与验证
   - `writing`：专项写作 Agent
3. `Chat` 页面必须默认绑定内置 `main` Agent（展示名 `Alter0`），进入 `Chat` 路由或新建 `Chat` 会话时需自动纠偏到该 Agent，不再以 Raw Model 作为默认执行目标。
4. 前端必须保留 `Chat` 作为内置 `main` Agent 的独立入口；其余入口 Agent 统一通过通用 `Agent` 运行页承载。具备独立前端入口的 Agent 不进入通用 `Agent` 页的候选列表与会话历史。
5. 控制面 `Agent Profiles` 页面仅管理用户自定义 Agent；系统内置 Agent 不允许通过控制面覆盖或删除。
6. 运行时必须提供统一入口 Agent 列表接口，供前端选择 Agent 与展示内置 Agent。
   - 当前接口：`GET /api/agents`
7. 主 Agent 调度子 Agent 必须走统一运行时工具能力，不通过页面跳转或伪造用户请求实现。
   - 当前内置委派工具：`delegate_agent`
8. `delegate_agent` 至少支持：
   - `agent_id`：目标 Agent 标识
   - `task`：下发给子 Agent 的具体任务
   - `context`：可选补充上下文
9. `coding` Agent 必须作为用户可直接选择的入口 Agent，对编码类请求承担主交互职责；具体开发工作优先通过 `codex_exec` 执行，并根据每轮执行结果继续追加下一步实现或验证动作，直至完成或明确阻塞。
   - 运行时需向 `coding` Agent 注入当前项目的远端仓库地址、本地仓库路径、当前分支、会话工作区、PR 基线分支与交付要求。
   - 涉及测试页面时，预览域名必须使用当前会话标识派生的 8 位短 hash，格式为 `https://<session_short_hash>.alter0.cn`。
   - `coding` Agent 在完成收口前，需明确交付代码变更、验证结果、文档同步状态、预览页地址（如适用）与 PR handoff 信息。
10. 被调度 Agent 需复用统一 Agent Profile 注入链路，包括 `provider/model`、工具白名单、Skills、MCP 与 Memory Files。
11. 需限制主从递归深度与自委派，避免无限递归。
12. 验收：
   - `GET /api/agents` 可返回内置入口 Agent
   - `Chat` 默认走 `main` Agent
   - `coding` Agent 在通用 `Agent` 入口中可直接选择，并以 Agent 主导、Codex 执行的方式推进编码任务
   - `coding` Agent 的运行时提示包含当前仓库、分支、会话预览域名和 PR 交付规则
   - 涉及测试页面时，预览地址遵循 `https://<session_short_hash>.alter0.cn`
   - `Agent` 页可直接进入 `coding`、`writing` 等无独立前端入口的内置 Agent 会话
   - 主 Agent 可通过 `delegate_agent` 调用 `coding`、`writing` 等专项 Agent 并回收结果
   - 用户新建 Agent 时若名称与内置 Agent 保留 ID 冲突，服务端会自动生成下一个可用 ID

#### Traceability

- 实现文件：`internal/agent/application/catalog.go`、`internal/agent/application/builtin.go`、`internal/execution/infrastructure/hybrid_nl_processor.go`、`internal/execution/infrastructure/codex_cli_processor.go`、`internal/interfaces/web/server.go`、`internal/interfaces/web/static/assets/chat.js`、`internal/interfaces/web/static/chat.html`、`cmd/alter0/main.go`
- 测试覆盖：`internal/execution/infrastructure/hybrid_nl_processor_test.go`、`internal/execution/infrastructure/codex_cli_processor_test.go`、`internal/interfaces/web/server_control_test.go`、`internal/interfaces/web/server_message_test.go`

### R-054 Product 目录、Workspace 与管理页

1. 运行时必须新增一级 `Products` 模块，作为平台级 Product 管理入口；`Product` 是业务产品域的一等对象，不使用 `App` 作为同级领域模型命名。
2. `Products` 页面需同时提供 `Workspace` 与 `Studio` 两类视图，至少覆盖：基础信息、启停状态、入口标识、总 Agent 绑定、详情页空间列表、子 Agent 矩阵摘要、产物类型摘要。
3. Product 最小主数据结构至少包含：`product_id`、`name`、`slug`、`summary`、`status`、`visibility`、`owner_type`、`master_agent_id`、`entry_route`、`tags`、`version`、`created_at`、`updated_at`。
4. Product 必须区分系统内置与用户创建两类来源：
   - 系统内置 Product 由服务注册，不允许在控制面直接删除；
   - 用户创建 Product 允许编辑、停用、归档，并保留历史引用关系。
5. 页面交互至少包含：列表浏览、创建、编辑、启停、查看矩阵摘要、查看 Product 详情、进入 Product Workspace、查看主 Agent 对话入口与详情页空间；不要求首版支持拖拽编排。
6. Product 详情页至少展示：
   - 基础信息：名称、描述、标签、状态、版本；
   - Agent 结构：总 Agent、子 Agent 数量、矩阵角色摘要；
   - 能力摘要：可调用工具、Skills、MCP、资料挂载；
   - 产物摘要：该 Product 的主要输出物类型；
   - Workspace 摘要：主 Agent 对话入口、详情页空间列表、当前空间详情与独立 HTML 页面入口。
7. `Products` 页面与 Product 详情页必须支持空态、禁用态和草稿态，避免在 Product 尚未发布时误进入生产执行入口。
8. Product 必须保留稳定 `product_id`，供 `Alter0 Agent`、任务系统、产物系统和会话视图统一引用。
9. 首版存储允许继续采用本地文件，但必须与既有 Control 数据解耦，避免 Product 目录、Agent Profile 与普通 Session 数据混存不可辨。
10. 验收：
   - Web 端可见独立 `Products` 模块；
   - 用户可创建和维护 Product 基础信息；
   - 每个 Product 可查看绑定总 Agent 与矩阵摘要；
   - 系统内置 Product 与用户 Product 可被明确区分。

#### 接口拆分（当前实现）

1. Product 列表
   - `GET /api/control/products`
   - 返回：`items[]`，包含基础字段、状态、总 Agent 绑定与矩阵摘要
2. Product 创建
   - `POST /api/control/products`
   - 入参：Product 基础信息与初始配置
3. Product 更新
   - `PUT /api/control/products/{product_id}`
   - 入参：Product 基础信息、状态、入口信息、展示配置
4. Product 删除
   - `DELETE /api/control/products/{product_id}`
   - 仅允许删除托管 Product；内置 Product 维持只读
5. Product 详情
   - `GET /api/control/products/{product_id}`
   - 返回：基础信息 + Agent 矩阵摘要 + 能力摘要 + 产物摘要
6. Product Workspace
   - `GET /api/products/{product_id}/workspace`
   - `GET /api/products/{product_id}/workspace/spaces/{space_id}`
   - 返回：Product 概览、主 Agent 摘要、详情页空间列表与具体空间详情
7. Product 公共详情页空间 HTML
   - `GET /products/{product_id}/spaces/{space_id}.html`
   - 返回：具体详情页空间的独立 HTML 页面

#### Traceability

- 领域对象：`Product`、`ProductCatalog`、`product_id`、`master_agent_id`
- 当前实现文件：`internal/product/domain/product.go`、`internal/product/application/service.go`、`internal/storage/infrastructure/localfile/product_store.go`、`internal/interfaces/web/server.go`、`internal/interfaces/web/product_workspace.go`、`internal/interfaces/web/static/assets/chat.js`
- 依赖需求：`R-021`、`R-049`、`R-053`
- 验证口径：Product 管理可用性、内置/用户 Product 区分、状态切换一致性、页面入口稳定性

### R-055 Product Agent 与子产品矩阵生成

1. 平台必须提供一个内置 `Product Agent`，用于根据产品目标和约束生成新 Product 草案，而不是要求用户手工逐项编排完整矩阵。
2. `Product Agent` 的输入至少包括：`name`、`goal`、`target_users`、`core_capabilities`、`constraints`、`expected_artifacts`、`integration_requirements`。
3. `Product Agent` 的输出至少包括：
   - Product 基础定义草案；
   - `master agent` 草案；
   - 子 Agent 矩阵草案；
   - 能力边界与职责拆分；
   - 默认资料结构与产物类型定义。
4. `Product Agent` 生成的矩阵草案必须显式列出每个子 Agent 的角色、输入输出边界、可用工具、依赖关系与可委派对象，避免“泛用大 Agent”吞并所有职责。
5. `Product Agent` 生成结果默认进入草稿态，不得直接自动发布到用户可执行入口；发布前需允许人工审核、编辑和禁用部分子 Agent。
6. 矩阵生成至少支持两种模式：
   - `bootstrap`：从零创建新 Product 与完整矩阵；
   - `expand`：在已有 Product 下新增子域或补充子 Agent。
7. 若已有 Product 与新输入名称或职责高度冲突，系统需给出复用或合并建议，而不是无约束重复创建多个语义相同 Product。
8. `Product Agent` 生成的 Agent 必须兼容现有 Agent Profile 基础能力，包括工具白名单、Skills、MCP、Memory Files 与模型配置引用。
9. 生成过程需保留草案版本和审核记录，便于回滚、对比和多轮修订。
10. 验收：
   - 平台可通过 `Product Agent` 生成新 Product 草案；
   - 草案中可查看总 Agent 与子 Agent 矩阵；
   - 审核通过后可发布为正式 Product；
   - 已有 Product 可按 `expand` 模式追加子矩阵。

#### 矩阵定义（草案）

1. 子 Agent 最小字段
   - `agent_id`
   - `role`
   - `responsibility`
   - `input_contract`
   - `output_contract`
   - `allowed_tools`
   - `allowed_delegate_targets`
   - `priority`
   - `enabled`
2. 矩阵级字段
   - `product_id`
   - `version`
   - `master_agent_id`
   - `worker_agents[]`
   - `artifact_types[]`
   - `knowledge_sources[]`
3. 草稿级字段
   - `draft_id`
   - `mode`
   - `review_status`
   - `generated_by`
   - `generated_at`

#### 接口拆分（当前实现）

1. Product Agent 草案生成
   - `POST /api/control/products/generate`
   - 入参：Product 目标、约束与模式
   - 返回：Product 草案、总 Agent 草案、子 Agent 矩阵草案
2. 已有 Product 扩展矩阵
   - `POST /api/control/products/{product_id}/matrix/generate`
   - 入参：新增能力目标与边界约束
3. 草案查询
   - `GET /api/control/products/drafts`
   - `GET /api/control/products/drafts/{draft_id}`
4. 草案审核与发布
   - `POST /api/control/products/drafts/{draft_id}/publish`
   - `PUT /api/control/products/drafts/{draft_id}`
   - 前端 `Draft Studio` 当前以结构化摘要 + JSON review editor 的方式支持审核、编辑、禁用子 Agent 与发布

#### Traceability

- 领域对象：`ProductDraft`、`ProductMatrix`、`ProductAgent`、`draft_id`
- 当前实现文件：`internal/product/domain/draft.go`、`internal/product/application/draft_service.go`、`internal/storage/infrastructure/localfile/product_draft_store.go`、`internal/interfaces/web/product_features.go`、`internal/interfaces/web/static/assets/chat.js`
- 依赖需求：`R-052`、`R-053`、`R-054`
- 验证口径：草案生成完整性、审核发布闭环、矩阵职责清晰度、重复 Product 冲突处理

### R-056 Product 总 Agent 与子 Agent 矩阵编排

1. 每个已发布 Product 必须绑定且仅绑定一个 `master agent`，作为该 Product 的统一需求入口与任务编排中心。
2. `master agent` 负责：
   - 识别当前请求是否属于本 Product；
   - 解析用户目标、约束和上下文；
   - 决定是否拆分子任务；
   - 调度子 Agent；
   - 汇总最终结果与产物。
3. Product 下的子 Agent 以矩阵方式组织，每个子 Agent 必须具备明确职责边界，不允许多个 Agent 对同一职责做无约束重叠。
4. 子 Agent 可按 Product 自身特性定义，但至少要支持：
   - 查询型 Agent；
   - 规划型 Agent；
   - 生成型 Agent；
   - 交付型 Agent。
5. 矩阵编排必须支持顺序执行与并行执行两种模式；并行执行仅允许在输入输出边界明确、互不覆盖时启用。
6. `master agent` 与子 Agent 间的委派必须保留可追踪结构，至少记录：`product_id`、`master_agent_id`、`worker_agent_id`、`task_id`、`delegation_reason`、`delegation_order`。
7. 编排层必须限制递归深度、自委派和循环依赖，避免 `master agent -> worker -> master` 或子 Agent 互相回环导致无限递归。
8. Product 执行结果必须同时支持：
   - 用户可读结果；
   - 结构化产物；
   - 子任务链路摘要。
9. 当某个子 Agent 不可用、被禁用或执行失败时，`master agent` 需提供降级策略：重试、改派、跳过或返回部分结果，并明确告知缺失能力范围。
10. 验收：
   - 每个 Product 仅有一个总 Agent 入口；
   - 总 Agent 可按矩阵定义拆分并委派子任务；
   - 最终结果可回溯到各子 Agent；
   - 递归与循环委派被显式限制。

#### 编排规则（当前实现）

1. 路由优先级
   - Product 命中后先进入 `master agent`
   - 只有 `master agent` 可以决定是否调用子 Agent
2. 委派边界
   - 子 Agent 默认不允许直接越权调用其他 Product 的子 Agent
   - 跨 Product 调用统一回到 `Alter0 Agent` 或上层协调器裁决
3. 输出收口
   - 所有子 Agent 输出先回到 `master agent`
   - 用户侧只消费 `master agent` 汇总结果或结构化产物引用
4. 发布行为
   - 草稿发布时先将 `master agent` 与 `worker matrix` 物化为托管 Agent，再写入托管 Product 定义
   - 已发布 Product 仅暴露唯一 `master_agent_id` 作为外部执行入口

#### Traceability

- 领域对象：`ProductMasterAgent`、`ProductWorkerAgent`、`delegation_graph`
- 当前实现文件：`internal/interfaces/web/product_features.go`、`internal/interfaces/web/server.go`、`internal/agent/application/catalog.go`、`internal/execution/infrastructure/hybrid_nl_processor.go`
- 依赖需求：`R-048`、`R-053`、`R-054`、`R-055`
- 验证口径：总 Agent 唯一性、矩阵拆分稳定性、委派链路可追踪性、降级策略有效性

### R-057 Alter0 Agent 跨 Product 信息检索与任务调度

1. `Alter0 Agent` 作为平台统一入口，必须具备跨 Product 的意图识别、信息检索、任务分发与结果收口能力。
2. 当用户问题涉及 Product 信息查询时，`Alter0 Agent` 需先读取 Product 目录与该 Product 的公开信息，再决定是否进入执行态。
3. 当用户问题涉及 Product 任务执行时，`Alter0 Agent` 必须调用目标 Product 的 `master agent`，而不是直接越过 Product 边界调用其子 Agent。
4. `Alter0 Agent` 至少支持三类动作：
   - `discover_product`：识别用户意图并匹配 Product；
   - `read_product_context`：读取 Product 基础信息、矩阵摘要、能力边界；
   - `run_product_master`：调用目标 Product 的总 Agent 执行任务。
5. 当用户请求跨多个 Product 时，`Alter0 Agent` 可按顺序或并行调度多个 Product `master agent`，但必须在最终结果中明确区分各 Product 的输出来源。
6. 若命中的 Product 被禁用、未发布或无可用总 Agent，`Alter0 Agent` 需返回明确原因，并建议可用 Product 或退回通用处理路径。
7. `Alter0 Agent` 对 Product 的调度必须保留结构化元数据，至少包括：`matched_product_ids`、`selected_product_id`、`selection_reason`、`master_agent_id`、`product_execution_mode`。
8. Product 相关信息查询结果需可在普通 Chat 会话内返回，不强制用户先切换到 Product 专属入口。
9. 对于敏感或仅控制面可见的 Product 配置，`Alter0 Agent` 只能读取允许公开给运行态的摘要，不直接暴露内部草稿、凭据或审核记录。
10. 验收：
   - 用户可在默认 `Chat` 中询问某个 Product 的能力与入口；
   - 用户可直接在 `Alter0 Agent` 下发 Product 任务；
   - 系统可调用对应 Product 的总 Agent；
   - 多 Product 结果可被统一收口并区分来源。

#### 接口与运行时能力（当前实现）

1. Product 公开目录
   - `GET /api/products`
   - 返回：已发布且允许公开的 Product 列表与摘要
2. Product 公开详情
   - `GET /api/products/{product_id}`
   - 返回：基础信息、总 Agent 摘要、能力摘要、入口信息
3. Product 执行入口
   - `POST /api/products/{product_id}/messages`
   - 由目标 Product 的 `master agent` 负责处理
   - `POST /api/products/{product_id}/messages/stream`
   - 请求进入执行前会自动绑定 Product 的 `master_agent_id`，并注入 `alter0.product-context/v1`
4. Alter0 内部调度
   - 默认 `main` Agent 会先做 Product 发现，并补充 `alter0.product.discovery`
   - 对执行型请求自动切换到目标 Product `master agent`

#### Traceability

- 领域对象：`matched_product_ids`、`selected_product_id`、`run_product_master`
- 当前实现文件：`internal/interfaces/web/product_features.go`、`internal/interfaces/web/server.go`、`internal/execution/domain/product_context.go`、`internal/execution/infrastructure/hybrid_nl_processor.go`
- 依赖需求：`R-053`、`R-054`、`R-056`
- 验证口径：Product 命中准确性、公开信息读取边界、总 Agent 调度正确性、跨 Product 收口一致性

### R-058 Travel Product 首个产品域落地

1. 平台首个内置 Product 定义为 `travel`，作为 Product 平台与多 Agent 矩阵能力的首个验证场景。
2. `travel` 必须绑定独立 `travel-master`，并预留以下子 Agent 角色：
   - `travel-city-guide`
   - `travel-route-planner`
   - `travel-metro-guide`
   - `travel-food-recommender`
   - `travel-map-annotator`
3. `travel-master` 负责理解旅游场景用户需求，至少支持：城市、天数、出行风格、预算、同行人群、必去点、规避条件。
4. `travel` 的首批产物是城市旅游攻略；攻略结果必须同时支持：
   - 用户可读文案；
   - 结构化攻略数据；
   - 后续迭代修改所需的稳定字段。
5. 结构化攻略最小字段至少包含：`city`、`days`、`travel_style`、`must_visit`、`avoid`、`pois`、`metro_lines`、`daily_routes`、`foods`、`notes`、`map_layers`。
6. 用户后续补充要求时，`travel-master` 需在已有攻略基础上做 revision，而不是每次丢弃上下文重建整份攻略。
7. `travel` 首版允许先以结构化攻略和文本攻略为主；地图高亮、路线绘制和 POI 点位需在产物结构中预留标准字段，便于后续接地图能力。
8. `travel` 的子 Agent 职责边界要求：
   - `travel-city-guide`：聚合城市概览与景点分层；
   - `travel-route-planner`：负责行程排序与每日路线；
   - `travel-metro-guide`：负责地铁与公共交通建议；
   - `travel-food-recommender`：负责餐饮与用餐分布；
   - `travel-map-annotator`：负责地图图层和点线路径表达。
9. `travel` 必须支持在 Product 详情和后续 Product Workspace 中查看其总 Agent、子 Agent 矩阵和主要产物类型。
10. `travel` Workspace 必须支持和 `travel-master` 对话，并将创建/修改请求同步到具体城市页空间；当用户选择武汉、成都、北京等城市页后，后续修改默认作用于当前城市页。
11. `travel` 必须内置独立 `travel-page` Skill，作为城市页生成规则、章节组织与 HTML 呈现约定的统一规则簿；默认文件路径为 `.alter0/skills/travel-page.md`。
12. `travel-master` 必须默认挂载 `travel-page` Skill，并具备 `read_skill`、`write_skill` 工具，用于读取或维护该规则簿。
13. 当用户提出稳定、可复用、应影响后续多个城市页的偏好时，`travel-master` 需按需更新 `travel-page` Skill；若只是某个城市或某次出行的一次性要求，则只更新目标城市页，不写入 Skill。
14. `POST /api/products/travel/workspace/chat` 需优先走 `travel-master` 的结构化解析；若 Agent 执行链、模型响应或 CLI fallback 不可用，服务端需自动切换到本地规则解析，继续完成城市页创建或修订，不向用户直接暴露底层执行失败。
15. `travel` 的每个城市页空间都必须提供独立 HTML 页面，默认使用 `/products/travel/spaces/{space_id}.html` 访问，页面内容与 Workspace 当前城市页详情保持同步。
16. 验收：
   - 平台内可见 `travel` Product；
   - `Alter0 Agent` 可识别并路由到 `travel-master`；
   - 用户可生成指定城市的旅游攻略；
   - 用户可基于补充条件修改已有攻略；
   - `travel-master` 可读取 `travel-page` Skill，并在稳定偏好变更时更新规则簿；
   - 当 `travel-master` 执行失败时，Workspace Chat 仍可通过本地解析继续创建或修订城市页；
   - 每个城市页都可通过独立 HTML 路由直接访问；
   - 结果中保留可供地图与路线后续增强的结构化字段；
   - `Products -> travel -> Workspace` 中可直接通过主 Agent 对话创建或修改城市页。

#### 接口拆分（当前实现）

1. `travel` Product 信息
   - `GET /api/products/travel`
   - `GET /api/products/travel/workspace`
   - `GET /api/products/travel/workspace/spaces/{space_id}`
   - `POST /api/products/travel/workspace/chat`
   - `POST /api/products/travel/messages`
   - `POST /api/products/travel/messages/stream`
2. 城市攻略生成
   - `POST /api/products/travel/guides`
   - 入参：城市与用户约束
   - 返回：攻略文案 + 结构化攻略对象
3. 攻略修订
   - `POST /api/products/travel/guides/{guide_id}/revise`
   - 入参：追加要求、保留条件、替换条件
4. 攻略详情
   - `GET /api/products/travel/guides/{guide_id}`

#### Traceability

- 领域对象：`travel-master`、`guide_id`、`daily_routes`、`map_layers`
- 当前实现文件：`internal/product/domain/travel_guide.go`、`internal/product/application/travel_service.go`、`internal/product/application/builtin.go`、`internal/interfaces/web/product_workspace.go`、`internal/interfaces/web/product_workspace_page.go`、`internal/interfaces/web/static/assets/chat.js`、`internal/interfaces/web/server_message_test.go`、`cmd/alter0/builtin_skills.go`、`internal/execution/infrastructure/hybrid_nl_processor.go`
- 依赖需求：`R-054`、`R-055`、`R-056`、`R-057`
- 验证口径：`travel` Product 可见性、攻略生成完整性、revision 连续性、结构化结果可扩展性

### R-059

暂无细化内容。

### R-060

暂无细化内容。
