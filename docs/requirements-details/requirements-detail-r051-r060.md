# Requirements Details (R-051 ~ R-060)

> Last update: 2026-04-03

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
   - 会话态统一使用 `ready / busy / exited / interrupted`：`ready` 表示当前会话可继续输入，`busy` 表示当前轮正在执行，`exited / interrupted` 表示当前运行态结束但原会话仍可恢复。
   - turn / step 维度继续使用 `running / completed / failed / interrupted` 表示执行过程，不再让 session.status 兼任“是否正在执行当前轮”的含义。
   - 同一 Terminal 会话在同一次运行态中断、退出或宿主侧取消期间，只允许写入一条对应状态提醒；恢复后若再次进入新的中断周期，才补充新的提醒记录。
   - 对已退出或已中断会话发起恢复发送时，输入区上的旧运行态提示需立即清空；恢复请求飞行期间不得继续展示上一状态周期的 `Exited / Interrupted / Failed` 文案。
10. 兼容性要求：已有 Terminal 会话数据在升级后需可继续读取；若历史数据缺少 Codex CLI 会话标识，系统需允许降级为不可直连恢复但可保留历史，并支持后续重新绑定新的运行态。
   - 历史持久化中的 `running / starting` 会话值需分别兼容归一到 `ready / busy`，不要求用户手工迁移状态文件。
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
  - 2026-04-01：Terminal session.status 重构为 `ready / busy / exited / interrupted`；历史 `running / starting` 持久化值在加载时自动归一，空闲会话跨重启不再被误标为 `interrupted`。
  - 2026-04-02：Terminal 对已退出或已中断会话重新发送时，输入区上的旧运行态提示会在请求开始后立即清空，不再与当前恢复发送态并存。

### R-052

1. `Agent Profile` 配置页必须新增 `Memory Files` 勾选区，与 `Skills`、`MCP` 采用同级勾选交互。
2. 当前支持的选择项至少包括：`USER.md`、`SOUL.md`、`AGENTS.md`、长期 `MEMORY.md / memory.md`、`Daily Memory (Today)`、`Daily Memory (Yesterday)`；其中 `AGENTS.md` 固定表示当前 `agent_id` 的私有规则文件，不是共享文件。
3. Agent 请求命中对应 Profile 后，服务端必须将勾选结果写入统一运行时元数据，并在执行前解析为结构化 `memory_context`。
4. 所有 Agent Session 还必须自动维护一个只读 `Agent Session Profile` 文件，路径固定为 `.alter0/agents/<agent_id>/sessions/<session_id>.md`，用于沉淀该 Agent 在当前 Session 内的稳定上下文；该文件不需要在 Profile 页面额外勾选，但必须默认注入 `memory_context`。
5. `memory_context` 最小字段包括：`protocol`、`files[].id`、`files[].selection`、`files[].title`、`files[].path`、`files[].exists`、`files[].writable`、`files[].updated_at`、`files[].content`。
6. 文件内容注入需要保留可写文件路径；当目标文件不存在时，仍需返回预期路径与 `exists=false`，允许 Agent 后续直接创建并写入。
   - `agents_md` 的目标路径固定为 `.alter0/agents/<agent_id>/AGENTS.md`，不再回退到仓库根目录共享 `AGENTS.md`。
   - `USER.md`、`SOUL.md`、长期记忆与日记忆继续作为共享文件解析。
   - `agent_session_profile` 的目标路径固定为 `.alter0/agents/<agent_id>/sessions/<session_id>.md`；运行时需在注入前自动刷新该文件内容，并标记为 `writable=false`。
7. Agent 执行链路必须同时支持两类消费方式：
   - ReAct Agent：在 system prompt 中显式暴露已选记忆文件的路径、存在状态与内容。
   - Codex 执行链：在 `alter0.codex-exec/v1` 载荷中新增 `memory_context` 字段，保证 fallback 与 `codex_exec` 一致可见。
8. 所选记忆文件默认视为可维护对象，Agent 通过专用 `search_memory`、`read_memory`、`write_memory` 工具维护这些文件；其中 `search_memory` 用于在已挂载的历史记忆文件中按关键字检索相关上下文，具体实现、验证与其他文件系统操作仍统一走 `codex_exec`。
   - `AGENTS.md` 的读写范围仅限当前 Agent 对应的 `.alter0/agents/<agent_id>/AGENTS.md`，不得跨 Agent 读写其他 Agent 的 `AGENTS.md`。
   - `Agent Session Profile` 属于运行时自动维护的只读上下文，Agent 可检索与读取，但不能通过 `write_memory` 直接覆盖。
9. 长期记忆与日记忆路径需兼容 `alter0` 当前目录约定，同时允许对齐 OpenClaw 常见文件名：
   - 长期记忆优先识别 `MEMORY.md`、`memory.md`、`.alter0/memory/long-term/MEMORY.md`
   - 日记忆优先识别 `memory/YYYY-MM-DD.md` 与 `.alter0/memory/YYYY-MM-DD.md`
10. 为控制 prompt 体积，单文件与总注入体积必须设置截断上限；截断后保留显式标记，避免模型误以为内容完整。
11. `Agent Session Profile` 至少需要记录 `agent_id`、`agent_name`、`session_id`、更新时间、通道信息与 Session 工作区；`coding` Agent 还需补充当前仓库路径、远端仓库地址、当前分支、PR 基线分支（可探测时）、Session 工作区与预览地址。
12. 验收：
   - Web `Agent Profiles` 页面可稳定保存和回显 `memory_files`
   - `POST /api/agent/messages` 会将勾选结果注入为 `alter0.memory.include`
   - 执行器可生成 `alter0.memory-context/v1`
   - ReAct 与 Codex 两条链路都能看到相同记忆文件集
   - 文件不存在时 Agent 仍能拿到目标路径，并可通过 `write_memory` 或 `codex_exec` 创建目标文件
   - Agent 可先通过 `search_memory` 在已注入的记忆文件中按关键字定位相关历史，再决定是否调用 `read_memory` / `write_memory`
   - 选择 `agents_md` 时，注入路径固定为 `.alter0/agents/<agent_id>/AGENTS.md`，且不会读取其他 Agent 的 `AGENTS.md`
   - `USER.md`、`SOUL.md`、长期记忆与日记忆在不同 Agent 间保持共享可见
   - 所有 Agent Session 都会自动拿到 `.alter0/agents/<agent_id>/sessions/<session_id>.md`
   - `coding` Agent 的会话画像会自动沉淀仓库、分支、工作区与预览地址等上下文
13. 默认提供独立 `memory` Skill，可与 `memory_files` 同时启用；该 Skill 负责向 Agent / Codex 说明记忆模块、文件职责、读写边界与读写时机，具体文件内容仍以 `memory_context` 为准。
14. Skill 协议需支持可选文件型属性，至少包括：`skills[].file_path`、`skills[].writable`；当 Skill 绑定可维护规则文件时，Codex CLI 可结合该文件上下文执行读取或更新，Agent 侧不依赖通用原生工具做路径猜测。
15. 当前实现中，`AGENTS.md` 的共享仓库根文件不再作为 `agents_md` 注入源；仓库根 `AGENTS.md` 仅保留仓库维护用途，不参与跨 Agent 共享记忆。

#### Traceability

- 实现文件：`internal/control/domain/agent.go`、`internal/interfaces/web/server.go`、`internal/interfaces/web/static/assets/chat.js`
- 执行注入：`internal/execution/domain/memory_context.go`、`internal/execution/application/memory_context_resolver.go`、`internal/execution/application/agent_session_profile.go`、`internal/execution/application/service.go`
- 执行消费：`internal/execution/infrastructure/hybrid_nl_processor.go`、`internal/execution/infrastructure/codex_cli_processor.go`
- 测试覆盖：`internal/control/domain/agent_test.go`、`internal/execution/application/service_test.go`、`internal/execution/infrastructure/codex_cli_processor_test.go`、`internal/interfaces/web/server_message_test.go`

### R-053

1. 运行时必须提供统一 `Agent Catalog`，同时聚合系统内置 Agent 与控制面管理的 Agent Profile。
2. 内置 Agent 至少包括：
   - `main`：默认主 Agent `Alter0`，负责通用对话入口、用户意图理解、记忆利用与子 Agent 调度
   - `coding`：专项编码 Agent，负责理解用户开发目标、保持交互收口，并通过 `codex_exec` 多轮推进具体实现与验证
   - `writing`：专项写作 Agent，通过 `codex_exec` 推进写作相关落地任务
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
   - Agent system prompt 需采用持续助手口径，要求优先复用当前 Session 已确认的稳定上下文，而不是在多轮编码过程中重复索取相同信息。
   - Agent 作为用户与 Codex 之间的代理层，负责在 Agent 侧消化 system prompt、Skill、记忆和会话画像；这些 Agent 编排信息不直接透传给 Codex，Codex 只接收当前执行所需的最小上下文与具体指令。
   - `alter0.codex-exec/v1` 需统一支持结构化上下文字段，至少包括按需注入的 `runtime_context`、`product_context`、`product_discovery`、`skill_context`、`mcp_context`、`memory_context`；Codex 仅接收当前执行所需的最小必要信息，不要求 Agent 在每轮 `codex_exec` 指令里重复转述完整规则簿，也不得把与当前执行无关的上下文冗余透传。
   - 仓库类 `codex_exec` 需默认切到当前 Session 独立 repo worktree `.alter0/workspaces/sessions/<session_id>/repo`，先在该工作区拉取/检查代码，再进行实现、构建、测试与部署，不直接在主仓库工作目录开发。
   - 运行时需向 `coding` Agent 注入当前项目的远端仓库地址、源仓库路径、独立 repo worktree 路径、当前分支、会话工作区、PR 基线分支与交付要求。
   - 运行时需同步维护 `coding` Agent 在当前 Session 下的私有画像文件 `.alter0/agents/coding/sessions/<session_id>.md`，用于沉淀源仓库路径、独立 repo worktree 路径、仓库状态与交付上下文，供后续轮次直接复用。
   - 涉及测试页面时，预览域名必须使用当前会话标识派生的 8 位短 hash，格式为 `https://<session_short_hash>.alter0.cn`；需求完成前需先把页面部署或更新到该地址。
   - `coding` Agent 在完成收口前，需明确交付代码变更、验证结果、文档同步状态、预览页地址（如适用）与 PR handoff 信息。
10. 每个 Agent 还必须自动维护自己的私有 file-backed Skill，路径固定为 `.alter0/agents/<agent_id>/SKILL.md`；该 Skill 不要求出现在 Agent Profile 的 `skills` 勾选结果中，但在命中该 Agent 执行时必须随运行时 Skill 上下文一起注入。
   - 私有 Skill 用于沉淀该 Agent 的可复用工作模式、输出结构、检查清单、偏好和长期用户约束。
   - `AGENTS.md` 继续只负责该 Agent 的协作边界、仓库/工作区操作规则和交付要求；不得把一次性任务细节或 Agent 可复用打法混写进 `AGENTS.md`。
   - 若私有 Skill 文件不存在，运行时需自动创建默认规则簿，保证 Agent 首次执行时即可按用户诉求更新自己的 Skill。
11. 被调度 Agent 需复用统一 Agent Profile 注入链路，包括 `provider/model`、工具白名单、Skills、MCP 与 Memory Files；其稳定工具面默认收敛为 `codex_exec`、`search_memory`、`read_memory`、`write_memory`，以及在允许时暴露的 `delegate_agent`。
12. 需限制主从递归深度与自委派，避免无限递归。
13. 验收：
   - `GET /api/agents` 可返回内置入口 Agent
   - `Chat` 默认走 `main` Agent
   - `coding` Agent 在通用 `Agent` 入口中可直接选择，并以 Agent 主导、Codex 执行的方式推进编码任务
   - `coding` Agent 的运行时提示包含源仓库、独立 repo worktree、分支、会话预览域名、Session 画像上下文和 PR 交付规则；对应 `codex_exec` 载荷同步携带结构化 `runtime_context`
   - 任一 Agent 命中执行时，都能在 Skill 上下文里看到自己的私有 Skill `.alter0/agents/<agent_id>/SKILL.md`
   - 当前 Agent 的私有 Skill 文件不存在时，首次执行后会自动生成默认规则簿，并允许后续按用户提出的稳定偏好更新
   - 涉及测试页面时，预览地址遵循 `https://<session_short_hash>.alter0.cn`，且 `coding` Agent 收口前已完成对应页面部署或更新
   - `Agent` 页可直接进入 `coding`、`writing` 等无独立前端入口的内置 Agent 会话
   - 主 Agent 可通过 `delegate_agent` 调用 `coding`、`writing` 等专项 Agent 并回收结果
   - 用户新建 Agent 时若名称与内置 Agent 保留 ID 冲突，服务端会自动生成下一个可用 ID

#### Traceability

- 实现文件：`internal/agent/application/catalog.go`、`internal/agent/application/builtin.go`、`internal/execution/infrastructure/hybrid_nl_processor.go`、`internal/execution/infrastructure/codex_cli_processor.go`、`internal/interfaces/web/server.go`、`internal/interfaces/web/static/assets/chat.js`、`internal/interfaces/web/static/chat.html`、`cmd/alter0/main.go`
- 测试覆盖：`internal/execution/infrastructure/hybrid_nl_processor_test.go`、`internal/execution/infrastructure/codex_cli_processor_test.go`、`internal/interfaces/web/server_control_test.go`、`internal/interfaces/web/server_message_test.go`

### R-054 Product 目录、Workspace 与管理页

1. 运行时必须新增一级 `Products` 模块，作为平台级 Product 管理入口；`Product` 是业务产品域的一等对象，不使用 `App` 作为同级领域模型命名。
2. `Products` 页面需同时提供 `Workspace` 与 `Studio` 两类视图，至少覆盖：基础信息、启停状态、入口标识、总 Agent 绑定、详情页空间列表、supporting agents 摘要、产物类型摘要。
3. Product 最小主数据结构至少包含：`product_id`、`name`、`slug`、`summary`、`status`、`visibility`、`owner_type`、`master_agent_id`、`entry_route`、`tags`、`version`、`created_at`、`updated_at`。
4. Product 必须区分系统内置与用户创建两类来源：
   - 系统内置 Product 由服务注册，不允许在控制面直接删除；
   - 用户创建 Product 允许编辑、停用、归档，并保留历史引用关系。
5. 页面交互至少包含：列表浏览、创建、编辑、启停、查看 Agent 摘要、查看 Product 详情、进入 Product Workspace、查看主 Agent 对话入口与详情页空间；不要求首版支持拖拽编排。
6. Product 详情页至少展示：
   - 基础信息：名称、描述、标签、状态、版本；
   - Agent 结构：总 Agent、supporting agents 数量、角色摘要；
   - 能力摘要：可调用工具、Skills、MCP、资料挂载；
   - 产物摘要：该 Product 的主要输出物类型；
   - Workspace 摘要：主 Agent 对话入口、详情页空间列表、当前空间详情与独立 HTML 页面入口。
7. `Products` 页面与 Product 详情页必须支持空态、禁用态和草稿态，避免在 Product 尚未发布时误进入生产执行入口。
8. Product 必须保留稳定 `product_id`，供 `Alter0 Agent`、任务系统、产物系统和会话视图统一引用。
9. 首版存储允许继续采用本地文件，但必须与既有 Control 数据解耦，避免 Product 目录、Agent Profile 与普通 Session 数据混存不可辨。
10. 验收：
   - Web 端可见独立 `Products` 模块；
   - 用户可创建和维护 Product 基础信息；
   - 每个 Product 可查看绑定总 Agent 与 Agent 摘要；
   - 系统内置 Product 与用户 Product 可被明确区分。

#### 接口拆分（当前实现）

1. Product 列表
   - `GET /api/control/products`
   - 返回：`items[]`，包含基础字段、状态、总 Agent 绑定与 Agent 摘要
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
   - 返回：基础信息 + Agent 摘要 + 能力摘要 + 产物摘要
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

### R-055 Product Agent 单主 Agent 草稿生成

1. 平台必须提供一个内置 `Product Agent`，用于根据产品目标和约束生成新 Product 草案，而不是要求用户手工逐项编排完整 Agent 拓扑。
2. `Product Agent` 的输入至少包括：`name`、`goal`、`target_users`、`core_capabilities`、`constraints`、`expected_artifacts`、`integration_requirements`。
3. `Product Agent` 的输出至少包括：
   - Product 基础定义草案；
   - `master agent` 草案；
   - 可复用 Prompt/Skill 沉淀建议；
   - 能力边界与职责拆分；
   - 默认资料结构与产物类型定义。
4. 新生成草案默认采用单主 Agent；领域规则、章节顺序、稳定命名与呈现约束优先沉淀到 `master agent.system_prompt` 或可复用 Skill，而不是默认拆成多 Agent。
5. `Product Agent` 生成结果默认进入草稿态，不得直接自动发布到用户可执行入口；发布前需允许人工审核、编辑主 Agent 配置并保留兼容字段。
6. 草案生成至少支持两种模式：
   - `bootstrap`：从零创建新 Product 与主 Agent 草案；
   - `expand`：在已有 Product 下补充能力边界、Prompt 或 Skill 上下文。
7. 若已有 Product 与新输入名称或职责高度冲突，系统需给出复用或合并建议，而不是无约束重复创建多个语义相同 Product。
8. `Product Agent` 生成的 Agent 必须兼容现有 Agent Profile 基础能力，包括工具白名单、Skills、MCP、Memory Files 与模型配置引用。
9. `Product Agent` 生成的 `master agent` 必须遵循统一执行职责模型：Agent 负责持续协助、吸收上下文与结果收口，具体文件、仓库、Shell 或页面产出统一交给 `codex_exec` 执行。
10. 生成默认值要求：
   - `master agent` 默认工具集包含 `codex_exec`、`search_memory`、`read_memory`、`write_memory`
   - 运行时自动补充 `complete`
   - 默认挂载 `memory` Skill 与标准 Agent Memory Files
   - 领域型 Product 可通过主 Agent 私有 Skill 预置领域规则，例如 `travel-master` 的私有 `SKILL.md`
11. 生成过程需保留草案版本和审核记录，便于回滚、对比和多轮修订。
12. 验收：
   - 平台可通过 `Product Agent` 生成新 Product 草案；
   - 草案中可查看总 Agent 与其 Prompt/Skill 沉淀；
   - 草案中的总 Agent 默认体现 Agent 协助 / Codex 执行模型；
   - 审核通过后可发布为正式 Product；
   - 已有 Product 可按 `expand` 模式补充主 Agent 上下文。

#### 草案定义（当前实现）

1. `master agent` 最小字段
   - `agent_id`
   - `name`
   - `description`
   - `system_prompt`
   - `tools`
   - `skills`
   - `memory_files`
   - `enabled`
2. Product 级字段
   - `product_id`
   - `version`
   - `master_agent_id`
   - `artifact_types[]`
   - `knowledge_sources[]`
   - `worker_agents[]`：兼容保留字段，新草稿默认留空
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
   - 返回：Product 草案、总 Agent 草案与可选 supporting agents 字段
2. 已有 Product 增量扩展
   - `POST /api/control/products/{product_id}/matrix/generate`
   - 入参：新增能力目标与边界约束
   - 说明：当前作为兼容保留入口，默认只扩展主 Agent 上下文，不自动新增 worker matrix
3. 草案查询
   - `GET /api/control/products/drafts`
   - `GET /api/control/products/drafts/{draft_id}`
4. 草案审核与发布
   - `POST /api/control/products/drafts/{draft_id}/publish`
   - `PUT /api/control/products/drafts/{draft_id}`
   - 前端 `Draft Studio` 当前以结构化摘要 + JSON review editor 的方式支持审核、编辑主 Agent、兼容查看 supporting agents 与发布

#### Traceability

- 领域对象：`ProductDraft`、`ProductAgentDraft`、`draft_id`
- 当前实现文件：`internal/product/domain/draft.go`、`internal/product/application/draft_service.go`、`internal/storage/infrastructure/localfile/product_draft_store.go`、`internal/interfaces/web/product_features.go`、`internal/interfaces/web/static/assets/chat.js`
- 依赖需求：`R-052`、`R-053`、`R-054`
- 验证口径：草案生成完整性、审核发布闭环、单主 Agent 规则沉淀完整性、重复 Product 冲突处理

### R-056 Product 总 Agent 执行编排

1. 每个已发布 Product 必须绑定且仅绑定一个 `master agent`，作为该 Product 的统一需求入口与任务编排中心。
2. `master agent` 负责：
   - 识别当前请求是否属于本 Product；
   - 解析用户目标、约束和上下文；
   - 直接收口领域结果与结构化产物；
   - 决定是否复用兼容保留的 supporting agents；
   - 汇总最终结果与产物。
3. 新发布 Product 默认不依赖多 Agent 编排；领域规则、结构化字段与可复用约束优先沉淀到 `master agent` 的 system prompt 与 Skill。
4. Product 仍可兼容保留 supporting agents，但必须具备明确职责边界，不允许多个 Agent 对同一职责做无约束重叠。
5. 若存在 supporting agents，`master agent` 与其委派必须保留可追踪结构，至少记录：`product_id`、`master_agent_id`、`worker_agent_id`、`task_id`、`delegation_reason`、`delegation_order`。
6. 编排层必须限制递归深度、自委派和循环依赖，避免 `master agent -> worker -> master` 或 supporting agents 互相回环导致无限递归。
8. Product 执行结果必须同时支持：
   - 用户可读结果；
   - 结构化产物；
   - 必要时的子任务链路摘要。
9. 当某个 supporting agent 不可用、被禁用或执行失败时，`master agent` 需优先回退为单 Agent 直接执行，并明确告知缺失能力范围。
10. Product 总 Agent 与兼容 supporting agents 必须沿用统一职责模型：保持助手角色，负责理解用户目标、利用记忆与上下文推进执行，具体执行统一通过 `codex_exec` 完成；仅在允许且确有必要时通过 `delegate_agent` 调度其他 Agent。
11. 对通过 Draft Studio 发布的 Product，发布链路必须对 `master agent` 的工具、Skill 与 Memory Files 做归一化补齐，确保旧草稿或手工编辑后的草稿仍能回到当前稳定执行模型；兼容存在的 supporting agents 继续做同样归一化。
12. 验收：
   - 每个 Product 仅有一个总 Agent 入口；
   - 总 Agent 默认可直接完成领域执行；
   - 总 Agent 与兼容 supporting agents 具备 `codex_exec` 驱动的统一执行职责；
   - 最终结果可回溯到总 Agent，必要时可追溯到 supporting agents；
   - 递归与循环委派被显式限制。

#### 编排规则（当前实现）

1. 路由优先级
   - Product 命中后先进入 `master agent`
   - 只有 `master agent` 可以决定是否调用 supporting agents
2. 委派边界
   - supporting agents 默认不允许直接越权调用其他 Product 的 supporting agents
   - 跨 Product 调用统一回到 `Alter0 Agent` 或上层协调器裁决
3. 输出收口
   - 所有 supporting agents 输出先回到 `master agent`
   - 用户侧只消费 `master agent` 汇总结果或结构化产物引用
4. 发布行为
   - 草稿发布时至少将 `master agent` 物化为托管 Agent，再写入托管 Product 定义
   - 发布时自动补齐稳定工具面、领域 Skill、`memory` Skill 与标准 Memory Files，避免旧草稿继续保留过时执行模型
   - 已发布 Product 仅暴露唯一 `master_agent_id` 作为外部执行入口

#### Traceability

- 领域对象：`ProductMasterAgent`、`delegation_graph`
- 当前实现文件：`internal/interfaces/web/product_features.go`、`internal/interfaces/web/server.go`、`internal/agent/application/catalog.go`、`internal/execution/infrastructure/hybrid_nl_processor.go`
- 依赖需求：`R-048`、`R-053`、`R-054`、`R-055`
- 验证口径：总 Agent 唯一性、单 Agent 执行稳定性、委派链路可追踪性、降级策略有效性

### R-057 Alter0 Agent 跨 Product 信息检索与任务调度

1. `Alter0 Agent` 作为平台统一入口，必须具备跨 Product 的意图识别、信息检索、任务分发与结果收口能力。
2. 当用户问题涉及 Product 信息查询时，`Alter0 Agent` 需先读取 Product 目录与该 Product 的公开信息，再决定是否进入执行态。
3. 当用户问题涉及 Product 任务执行时，`Alter0 Agent` 必须调用目标 Product 的 `master agent`，而不是直接越过 Product 边界调用其 supporting agents。
4. `Alter0 Agent` 至少支持三类动作：
   - `discover_product`：识别用户意图并匹配 Product；
   - `read_product_context`：读取 Product 基础信息、Agent 摘要、能力边界；
   - `run_product_master`：调用目标 Product 的总 Agent 执行任务。
   - 上述动作属于运行时编排语义；真正的具体实现与执行仍由 `codex_exec` 或被委派的目标 Agent 完成。
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

1. 平台首个内置 Product 定义为 `travel`，作为 Product 平台单主 Agent + Skill 沉淀模式的首个验证场景。
2. `travel` 必须绑定独立 `travel-master`，并作为唯一默认执行入口承接旅游领域需求。
3. `travel-master` 负责理解旅游场景用户需求，至少支持：城市、天数、出行风格、预算、同行人群、必去点、规避条件。
4. `travel` 的首批产物是城市旅游攻略；攻略结果必须同时支持：
   - 用户可读文案；
   - 结构化攻略数据；
   - 后续迭代修改所需的稳定字段。
5. 结构化攻略最小字段至少包含：`city`、`days`、`travel_style`、`must_visit`、`avoid`、`pois`、`metro_lines`、`daily_routes`、`foods`、`notes`、`map_layers`。
6. 用户后续补充要求时，`travel-master` 需在已有攻略基础上做 revision，而不是每次丢弃上下文重建整份攻略。
7. `travel` 首版允许先以结构化攻略和文本攻略为主；地图高亮、路线绘制和 POI 点位需在产物结构中预留标准字段，便于后续接地图能力。
8. `travel-master` 的主职责必须覆盖城市概览、景点组织、行程排序、地铁与公共交通建议、餐饮推荐、地图图层表达与结果收口，不依赖默认拆分出的子 Agent。
9. `travel` 必须支持在 Product 详情和后续 Product Workspace 中查看其总 Agent、主要产物类型以及相关规则沉淀。
10. `travel` Workspace 必须支持和 `travel-master` 对话，并将创建/修改请求同步到具体城市页空间；当用户选择武汉、成都、北京等城市页后，后续修改默认作用于当前城市页。
11. `travel-master` 必须使用专属私有 file-backed Skill，默认文件路径为 `.alter0/agents/travel-master/SKILL.md`，作为城市页生成规则、章节组织与 HTML 呈现约定的统一规则簿。
12. `travel-master` 在命中执行时必须自动注入该私有 Skill；其具体城市页生成、规则读取与规则更新统一通过 `codex_exec` 结合 Skill 上下文完成，不再依赖独立 Skill 读写工具。
13. 当用户提出稳定、可复用、应影响后续多个城市页的偏好时，`travel-master` 需按需通过 `codex_exec` 更新自己的私有 Skill 规则簿；若只是某个城市或某次出行的一次性要求，则只更新目标城市页，不写入 Skill。
14. `POST /api/products/travel/workspace/chat` 需优先走 `travel-master` 的结构化解析；若 Agent 执行链、模型响应或 CLI fallback 不可用，服务端需自动切换到本地规则解析，继续完成城市页创建或修订，不向用户直接暴露底层执行失败。
15. `travel` 的每个城市页空间都必须提供独立 HTML 页面，默认使用 `/products/travel/spaces/{space_id}.html` 访问，页面内容与 Workspace 当前城市页详情保持同步。
16. 验收：
   - 平台内可见 `travel` Product；
   - `Alter0 Agent` 可识别并路由到 `travel-master`；
   - 用户可生成指定城市的旅游攻略；
   - 用户可基于补充条件修改已有攻略；
   - `travel-master` 可结合自己的私有 Skill 上下文，并在稳定偏好变更时通过 `codex_exec` 更新规则簿；
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
