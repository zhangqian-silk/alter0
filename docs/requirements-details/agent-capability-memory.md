# Agent Capability & Memory Requirements

> Last update: 2026-04-23

## 领域边界

Agent Capability & Memory 负责 Agent 定义、Agent Catalog、ReAct 执行、工具上下文、Skills/MCP 接入、记忆文件注入、长期记忆、天级记忆、上下文压缩与记忆可视化。

## 核心对象

| 对象 | 职责 |
| --- | --- |
| `AgentProfile` | 用户管理 Agent 的配置主体 |
| `AgentCatalog` | 聚合内置 Agent 与用户管理 Agent |
| `ReActAgentConfig` | ReAct 循环、迭代上限与工具策略 |
| `ToolExecutor` | 执行 `codex_exec`、记忆工具、委派工具与收口工具 |
| `Skill` | 可复用能力说明、规则与 file-backed Skill 上下文 |
| `MCPServer` | 外部 MCP Server 配置与注入项 |
| `MemoryFile` | 可注入、可读写或只读的记忆文件 |
| `MemoryContext` | 执行前解析出的记忆文件与召回片段 |
| `AgentSessionProfile` | 当前 Agent 在当前 Session 内的只读画像 |
| `LongTermMemory` / `DailyMemory` | 跨会话长期记忆与天级记忆 |

## Agent Catalog

### 内置 Agent

- `main`：默认对话主 Agent `Alter0`，负责通用入口、意图理解、记忆利用与专项 Agent 调度。
- `coding`：编码 Agent，负责理解开发目标、驱动 `codex_exec` 多轮完成修改、验证、预览与交付收口。
- `writing`：写作 Agent，负责文档、文案与结构化写作任务。
- `travel`：内置旅游 Agent，显示名称为 `Travel Agent`，负责城市行程类任务的专项执行、HTML 攻略产物与公开只读子域名交付。

### 用户管理 Agent

- Control 面支持创建、更新、启用、禁用和查询用户管理 Agent Profile。
- 系统内置 Agent 不允许通过控制面覆盖或删除。
- 创建 Agent 时服务端生成 `agent_id` 与 `version`；与内置 Agent 保留 ID 冲突时自动生成下一个可用 ID。

## ReAct 执行

### 执行职责

- Agent 定位为用户代理与执行驱动器，不直接承担仓库、文件、Shell 或页面产出的具体操作。
- Agent 在自身侧吸收 system prompt、Skill、Memory Files 与会话画像。
- Agent 只向 Codex 下发当前步骤所需的最小上下文与具体指令，不透传完整 Agent 侧规则簿。

### 工具面

- 稳定工具包括 `codex_exec`、`search_memory`、`read_memory`、`write_memory` 与运行时收口工具 `complete`。
- 允许委派的 Agent 可额外启用 `delegate_agent`。
- `search_memory`、`read_memory`、`write_memory` 仅面向已解析进 `memory_context` 的记忆文件。
- 其他文件、仓库与命令操作统一通过 `codex_exec` 执行。
- Agent / ReAct 使用 `openai-completions` 时，多轮工具调用必须保留 assistant `tool_calls` 与后续 `tool_call_id` 的关联，避免 Provider 因消息配对缺失拒绝请求。
- Agent / ReAct 处理来自 Web 的多模态用户输入时，用户消息需同时保留文本 part 与图片 part，并按目标 Provider 的能力序列化为视觉请求；带图请求在模型链失败时不得静默回退到 Codex 文本执行。

### `codex_exec`

- `codex_exec` 通过 stdin 传递最终指令，命令行参数仅保留 `-` 作为 prompt 占位。
- 长上下文、Memory、Skill 与运行时结构化载荷不得直接拼入系统命令行参数。
- 存在可用 Provider 且进入 Agent / ReAct 链路时，Agent 自身吸收 `runtime_context`、`skill_context`、`mcp_context`、`memory_context`，只向 Codex 下发当前步骤的纯执行指令。
- 不存在 Provider、Agent 初始化失败或请求直接进入 Terminal / 直连 Codex 时，运行时必须为当前会话生成独立 `CODEX_HOME/config.toml`、工作区 `AGENTS.md` 与 `.alter0/codex-runtime/*`，把上述上下文编译成 Codex 原生运行配置与工作区事实。
- 仓库类任务默认切到当前 Session 独立 repo 完整 clone `.alter0/workspaces/sessions/<session_id>/repo`，不使用 `git worktree`。
- `coding` Agent 需要拿到源仓库路径、独立 repo clone 路径、远端地址、当前分支、会话工作区、预览域名与 PR 交付要求；Agent / ReAct 路径由 Agent 自身维护这些事实，直连 Codex 路径则写入 Native Runtime 资产。

### 收口与错误

- Agent 在 `max_iterations` 耗尽但未显式 `complete` 时，不得返回空最终正文。
- 运行时必须返回迭代上限说明与最后一次工具观察。
- 前端 Process 与最终正文必须在流式 `done` 时保持一致。

## Skills 与 MCP

### Skill 配置

- Skill 配置包含 `description`、`guide`、排序、启停状态、作用域、文件路径与可写属性。
- 默认提供 `memory` Skill，说明记忆文件职责、读取决策、写入路由、冲突优先级与禁止写入项。
- `coding` 内置 Agent 默认启用 `memory`、`deploy-test-service` 与 `frontend-design`，使仓库类任务稳定带上记忆、预览发布和前端设计规则；运行时仍可在 `Details > Skills` 中对公有 Skill 做会话级调整。
- 每个 Agent 自动附带私有 file-backed Skill `.alter0/agents/<agent_id>/SKILL.md`。
- 私有 Skill 用于沉淀可复用工作模式、输出结构、检查清单、偏好和长期协作约束。
- 一次性任务细节不得写入私有 Skill。

### MCP 配置

- MCP Server 配置必须安全映射到 Codex 运行配置。
- MCP 支持按会话或请求启用。
- MCP 注入需保留审计追踪，不应绕过 Agent Profile 与运行时上下文解析。
- 直连 Codex 路径必须把启用的 MCP Server 编译为当前会话 `CODEX_HOME/config.toml` 中的原生 `mcp_servers.*` 配置，而不是仅在 prompt 中声明工具名。

## Memory Files

### 可选文件

- `USER.md`：共享用户信息与长期偏好。
- `SOUL.md`：高优先级强制要求，冲突时覆盖普通记忆。
- `AGENTS.md`：当前 Agent 私有规则文件，路径固定为 `.alter0/agents/<agent_id>/AGENTS.md`。
- `MEMORY.md / memory.md`：共享长期记忆。
- `memory/YYYY-MM-DD.md`：天级记忆。
- `Agent Session Profile`：当前 Agent 当前 Session 的只读画像与实例属性文件，路径固定为 `.alter0/agents/<agent_id>/sessions/<session_id>.md`。

### 注入协议

- 执行前解析勾选结果为 `memory_context`。
- 文件不存在时仍返回预期路径、`exists=false` 与可写性。
- 注入内容携带文件 ID、selection、标题、路径、存在状态、可写性、更新时间、内容快照与截断标记。
- 单文件与总注入体积必须设置上限，截断后保留显式标记。
- 本轮输入命中记忆文件时，轻量自动召回片段写入 `memory_context.recall[]`。
- `Agent Session Profile` 的自动块需要稳定输出 `Session Identity / Session Scope / Instance Attributes / Notes`；其中 `Instance Attributes` 用于承载当前 Agent 在当前 Session 内的结构化事实。
- 实例属性支持通过请求 metadata 增量写入；运行时需要把 JSON 映射和前缀式单键更新合并到 `Instance Attributes`，并与历史已存在属性做保留式合并。
- 每个 Agent 需要支持独立的 Session Profile 预设字段定义；字段定义至少包含 `key / label / readonly`，并作为当前 Agent Session 实例属性的稳定展示骨架。
- 运行时在注入 `Agent Session Profile` 前需要先执行旁路字段抽取：输入限定为当前 Agent schema、已有 profile 属性和本轮最新用户消息，只允许产出 schema 内的可写字段 patch，不复用主 Agent 的长上下文推理链路。

### 读写边界

- `AGENTS.md` 只能读写当前 Agent 对应文件，不跨 Agent 共享。
- `Agent Session Profile` 由运行时自动维护，Agent 可检索与读取，不可通过 `write_memory` 覆盖。
- `coding` Agent 需要在实例属性里自动维护 `repository_path`、`source_repository_path`、`remote_repository`、`branch`、`base_branch`、`preview_url` 与 `preview_subdomain` 等稳定交付事实。
- `travel` 与其他专项 Agent 的实例属性可按领域预设字段，例如 `city / district / days / hotel_area`，并允许在当前 Session 内继续维护对应键值。
- Agent Runtime `Details` 需要提供 `Session Profile` 只读视图，按当前 Agent 预设字段顺序展示实例属性值。
- 旁路抽取失败不能阻塞主执行链路；默认可退化为一次受限 Codex 窄调用，只返回 JSON patch，由运行时统一校验和落盘。
- 稳定、可复用、影响后续行为的偏好可写入长期记忆、私有 `AGENTS.md` 或私有 Skill。
- 一次性任务约束只进入当前会话、任务或目标产物。

## 记忆体系

### 短期记忆

- Chat 与 Agent 在单 Session 内维护可控窗口上下文。
- 内存窗口不足时，从持久化 session history 回填最近完成轮次。
- 回填内容用于指代解析、结果回顾与执行连续性。

### 长期记忆

- 跨会话长期记忆按用户或租户范围沉淀事实与偏好。
- 召回结果按相关性、优先级与 token 预算注入。
- 重要记忆可按 L1/L2/L3 分级管理，并按命中率与重要性迁移。

### 天级记忆

- Daily Memory 使用 Markdown 存储。
- 每日记忆可压缩归档，并按时间和相关性参与召回。

### 上下文压缩

- 超长上下文触发分层压缩。
- 压缩结果保留摘要、关键事实与原消息引用关系。
- 压缩结果可回写为长期或任务摘要记忆。

### 任务记忆

- 任务摘要默认只注入最近少量条目，控制上下文体积。
- 用户询问更早历史时自动进入深检索，从全量任务摘要库召回并按需下钻任务详情。
- 任务记忆需要保留 `session_id`、`message_id`、`task_id` 回链。
- 任务摘要可按单个任务重建；重建只刷新 Memory 可召回摘要，不改变原任务执行状态、日志和产物。

## Memory 页面

- 前端 `Agent -> Memory` 提供长期记忆、天级记忆、强制要求、任务历史与说明文档入口。
- Web Shell 桥接阶段中，`memory` 路由页主体由 React 直接请求 `/api/agent/memory` 与 `/api/memory/tasks*` 并渲染标签页、筛选表单、任务详情、日志、产物与只读记忆文档卡片；legacy runtime 不再托管该页主体 DOM。
- `GET /api/agent/memory` 返回长期记忆、天级记忆、强制上下文和说明文档的只读聚合视图。
- 说明文档支持稳定查看 `USER.md`、`AGENTS.md`、`MEMORY.md`、`memory/YYYY-MM-DD.md`、`SOUL.md` 的职责说明与映射关系。
- Memory 聚合接口默认只读展示，不提供在线编辑；非 GET 请求返回方法不允许。
- 任务历史摘要默认可见，桌面端以表格承载 `task_id / task_type / status / start / end / tags` 元数据，再通过右侧详情面板按需下钻日志、产物与摘要引用。
- 任务摘要缺失或过期时，页面可触发单任务摘要重建并刷新当前任务视图。

## 依赖与边界

- Conversation 使用 Agent 的最终回复和 Process 数据，不定义 ReAct 工具协议。
- Task 使用 Agent 产出的异步执行和任务摘要，不拥有 Memory 文件写入规则。
- Control 面管理 Agent Profile、Skill、MCP、Provider，不直接参与执行决策。

## 验收口径

- `GET /api/agents` 返回内置与用户管理入口 Agent。
- Chat 默认走 `main`，Agent 页面可选择 `coding`、`writing` 等入口 Agent。
- ReAct 执行可见 Process、最终答复与迭代上限错误收口。
- Memory Files 注入包含路径、内容、可写性和召回片段。
- Agent 私有 `AGENTS.md`、私有 Skill 与 Session Profile 路径隔离正确。
- `codex_exec` 使用 stdin，不因长上下文触发命令行长度限制；Agent / ReAct 路径与直连 Codex 路径都需要保持该约束。
