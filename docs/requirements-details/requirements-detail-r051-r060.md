# Requirements Details (R-051 ~ R-060)

> Last update: 2026-03-28

## 需求细化（草案）

### R-051 Terminal 会话持久标识与超时恢复

1. Terminal 模块必须持久化存储 Codex CLI 会话标识，并与平台内 `terminal_session_id` 建立稳定映射；页面刷新、服务重启或运行态退出后，该标识仍可用于恢复会话。
2. 若当前实现中的 `terminal_session_id` 已实际承载 Codex CLI 会话标识，则需将其作为持久恢复主键稳定保存；若两者语义不同，则需新增独立字段保存 Codex CLI 标识，并保持映射关系可查询。
3. Terminal 会话持久化内容至少包含：`terminal_session_id`、Codex CLI 会话标识、标题、工作目录、创建时间、最近活动时间、最近一次运行状态、关联日志或步骤索引。
   - 当前实现将 Terminal 会话状态持久化到 `.alter0/state/terminal/sessions/<terminal_session_id>.json`，保留会话摘要、Codex CLI 线程标识、输出日志与步骤视图索引。
4. Terminal 模块不再设置产品级会话上限；前端、接口与配置项中不再以“最大会话数”为用户约束，不再因达到上限拒绝创建或恢复 Terminal 会话。
5. Terminal 模块不再设置产品级会话超时淘汰规则；系统不因固定超时阈值自动销毁 Terminal 会话记录、历史消息、日志索引或 Codex CLI 标识映射。
6. 若底层 Codex CLI 运行因空闲、网络、进程退出或外部超时而中断，系统仅将该 Terminal 会话标记为已退出、已中断或待恢复状态，不删除会话本身，不清空历史，不重置标识映射。
   - 用户显式执行 `Delete` 时除外：系统需删除 Terminal 会话主记录、`.alter0/state/terminal/sessions/<terminal_session_id>.json` 与 `.alter0/workspaces/terminal/sessions/<terminal_session_id>`，不再保留恢复入口。
7. 恢复要求：用户可对已退出或超时中断的 Terminal 会话执行恢复；恢复时系统优先复用已持久化的 Codex CLI 会话标识继续接续原会话，无法直接接续时需返回明确原因，并允许在保留同一 Terminal 会话历史的前提下重新建立运行态。
8. 输入要求：当用户向已退出或超时中断的 Terminal 会话继续发送输入时，系统可自动触发恢复流程，或先给出明确恢复提示并由用户确认后恢复，但不得强制用户新建独立会话。
   - 当前实现采用“继续发送即恢复”的交互方式；前端保留原会话，直接对已退出或已中断会话重新提交输入。
   - 当 Terminal 会话仅创建未发送首条输入、但底层运行态已不存在时，前端在首次发送时先调用恢复接口重建同一 `terminal_session_id`，随后自动重试当前输入。
   - 浏览器端需将 Terminal client 标识持久化到本地可复用存储；若仅临时 `sessionStorage` 丢失但仍保留原会话记录，刷新后仍应优先沿用原 client 标识访问会话。
   - 当服务端仍保留该 Terminal 会话且恢复请求携带同一 `terminal_session_id` 与既有 CLI 线程标识时，恢复接口需允许在同一 Web 登录态下重新绑定新的前端 client 标识，不返回误导性的“terminal session not found”。
9. 展示要求：Terminal 页面需明确区分“会话历史仍在”与“当前运行态已退出”两种状态，避免把运行态超时误展示为会话丢失或不可恢复。
10. 兼容性要求：已有 Terminal 会话数据在升级后需可继续读取；若历史数据缺少 Codex CLI 会话标识，系统需允许降级为不可直连恢复但可保留历史，并支持后续重新绑定新的运行态。
11. 验收：Terminal 页面创建多个会话后不再因会话数触发上限错误；某一会话运行超时或中断后，历史记录仍可见，用户可对原会话执行恢复并继续输入，且恢复后仍能回看原有日志与上下文链路。
   - 补充验收：新建后尚未发送首条输入的 Terminal 会话即使因空闲导致运行态缺失，用户首次发送仍在原会话内成功执行，不出现“terminal session not found”阻断。
   - 补充验收：页面已缓存 Terminal 会话历史时，即使浏览器临时 `sessionStorage` 被清空或前端 client 标识重新生成，刷新后也不会把原会话误标为中断；若服务端仍保留原会话，后续恢复或继续发送仍绑定到同一 Terminal 会话历史。
   - 补充验收：用户显式删除某个 Terminal 会话后，该会话不再可恢复，相关状态文件与工作区目录同步清理。

#### 实施边界（草案）

1. 本需求聚焦 Terminal 会话生命周期治理，不调整普通 Chat/Agent 会话模型。
2. 本需求取消的是产品级 Terminal 会话上限与超时淘汰策略，不等同于取消底层网络超时、代理超时或外部 CLI 自身超时现象。
3. 当底层 Codex CLI 已彻底失效且无法基于原标识恢复时，系统可重建运行态，但需保持 Terminal 会话主记录、历史输出与恢复失败原因可见。
4. 本需求不要求无限保留底层进程常驻；允许运行态退出，但不允许因此丢失 Terminal 会话身份与恢复入口。

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
4. Terminal 配置与约束
   - 不再暴露 `task_terminal_max_sessions` 或等价 Terminal 会话上限配置为用户侧可调能力

#### Traceability

- 核心对象：`terminal_session_id`、`codex_cli_session_id`、`recoverable_state`、`runtime_status`、`terminal_logs`
- 依赖需求：`R-019`、`R-035`、`R-046`
- 验证口径：标识持久化完整性、超时后历史保留完整性、恢复成功率、恢复失败可解释性、无会话上限拦截

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

#### Traceability

- 实现文件：`internal/control/domain/agent.go`、`internal/interfaces/web/server.go`、`internal/interfaces/web/static/assets/chat.js`
- 执行注入：`internal/execution/domain/memory_context.go`、`internal/execution/application/memory_context_resolver.go`、`internal/execution/application/service.go`
- 执行消费：`internal/execution/infrastructure/hybrid_nl_processor.go`、`internal/execution/infrastructure/codex_cli_processor.go`
- 测试覆盖：`internal/control/domain/agent_test.go`、`internal/execution/application/service_test.go`、`internal/execution/infrastructure/codex_cli_processor_test.go`、`internal/interfaces/web/server_message_test.go`

### R-053

1. 运行时必须提供统一 `Agent Catalog`，同时聚合系统内置 Agent 与控制面管理的 Agent Profile。
2. 内置 Agent 至少包括：
   - `main`：默认主 Agent，负责通用对话入口与子 Agent 调度
   - `coding`：专项编码 Agent
   - `writing`：专项写作 Agent
3. `Chat` 页面必须默认绑定内置 `main` Agent，不再以 Raw Model 作为默认执行目标。
4. 前端必须提供专项 Agent 入口路由；当前至少包括 `Coding` 与 `Writing`，并默认分别绑定 `coding`、`writing` Agent。
5. 控制面 `Agent Profiles` 页面仅管理用户自定义 Agent；系统内置 Agent 不允许通过控制面覆盖或删除。
6. 运行时必须提供统一入口 Agent 列表接口，供前端选择 Agent 与展示内置 Agent。
   - 当前接口：`GET /api/agents`
7. 主 Agent 调度子 Agent 必须走统一运行时工具能力，不通过页面跳转或伪造用户请求实现。
   - 当前内置委派工具：`delegate_agent`
8. `delegate_agent` 至少支持：
   - `agent_id`：目标 Agent 标识
   - `task`：下发给子 Agent 的具体任务
   - `context`：可选补充上下文
9. 被调度 Agent 需复用统一 Agent Profile 注入链路，包括 `provider/model`、工具白名单、Skills、MCP 与 Memory Files。
10. 需限制主从递归深度与自委派，避免无限递归。
11. 验收：
   - `GET /api/agents` 可返回内置入口 Agent
   - `Chat` 默认走 `main` Agent
   - `Coding`、`Writing` 路由可直接进入对应内置 Agent 会话
   - 主 Agent 可通过 `delegate_agent` 调用 `coding`、`writing` 等专项 Agent 并回收结果
   - 用户新建 Agent 时若名称与内置 Agent 保留 ID 冲突，服务端会自动生成下一个可用 ID

#### Traceability

- 实现文件：`internal/agent/application/catalog.go`、`internal/agent/application/builtin.go`、`internal/execution/infrastructure/hybrid_nl_processor.go`、`internal/interfaces/web/server.go`、`internal/interfaces/web/static/assets/chat.js`、`internal/interfaces/web/static/chat.html`、`cmd/alter0/main.go`
- 测试覆盖：`internal/execution/infrastructure/hybrid_nl_processor_test.go`、`internal/interfaces/web/server_control_test.go`、`internal/interfaces/web/server_message_test.go`

### R-054

暂无细化内容。

### R-055

暂无细化内容。

### R-056

暂无细化内容。

### R-057

暂无细化内容。

### R-058

暂无细化内容。

### R-059

暂无细化内容。

### R-060

暂无细化内容。
