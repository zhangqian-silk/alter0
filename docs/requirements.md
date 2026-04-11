# Requirements

> Last update: 2026-04-11

`alter0` 的需求清单按领域模型维护。后续新增需求不再使用线性编号，也不按提交顺序堆叠；需求应落到对应领域、子域与能力项下，使用稳定领域路径表达，例如 `agent.execution.react`、`memory.files.injection`、`product.travel.workspace`。

状态说明：

- `supported`：已在主干代码可用
- `in-progress`：正在落地，接口或行为尚未稳定
- `planned`：已确认方向，待排期

## 领域索引

| 领域 | 范围 | 状态 | 细化文档 |
| --- | --- | --- | --- |
| Runtime & Orchestration | 输入通道、统一消息、意图路由、执行端口、调度、存储、观测 | supported | [runtime-orchestration.md](requirements-details/runtime-orchestration.md) |
| Conversation & Session Experience | Chat、Agent 入口会话、SSE、历史、移动端、阅读与输入体验 | supported | [conversation-session-experience.md](requirements-details/conversation-session-experience.md) |
| Agent Capability & Memory | Agent Catalog、ReAct、工具、Skills、MCP、Memory Files、长期记忆、上下文压缩 | supported | [agent-capability-memory.md](requirements-details/agent-capability-memory.md) |
| Task, Terminal & Workspace | 异步任务、任务观测、任务日志、产物交付、Terminal 会话、独立工作区 | supported | [task-terminal-workspace.md](requirements-details/task-terminal-workspace.md) |
| Product Domain | Product 目录、Draft Studio、Product 总 Agent、跨 Product 调度、Travel 产品域 | supported | [product-domain.md](requirements-details/product-domain.md) |
| Control, Operations & Governance | 控制面配置、模型 Provider、Environments、部署基线、认证凭据、TDD 研发约束 | supported | [control-operations-governance.md](requirements-details/control-operations-governance.md) |

## Runtime & Orchestration

核心对象：`UnifiedMessage`、`OrchestrationResult`、`Intent`、`Command`、`ExecutionPort`、`SchedulerJob`、`Channel`、`TraceContext`。

稳定需求：

- CLI、Web、Cron 等输入源统一转换为 `UnifiedMessage`，并携带 `message_id`、`session_id`、`trace_id`、`channel_type`、`trigger_type` 与业务载荷。
- 编排层负责意图识别与路由：命令进入 `CommandRegistry` / `CommandHandler`，自然语言进入 `ExecutionPort`，Cron 触发复用同一编排链路。
- 命令能力稳定提供 `/help`、`/echo`、`/time` 与 `/now`。
- 自然语言执行通过可替换执行端口对接 LLM、Agent、Codex CLI 或后续 Workflow 执行器。
- 调度领域支持 Cron Job 配置、可视化周期字段、触发记录、触发会话归档、runs 查询与来源回链。
- 存储默认采用 `.alter0` 本地文件，Control 配置与 Scheduler 状态以 JSON 为主，Memory 主存以 Markdown 为主。
- 观测能力覆盖结构化日志、Prometheus metrics、`/healthz`、`/readyz` 与 trace/session/message 维度。

## Conversation & Session Experience

核心对象：`Session`、`Message`、`LiveUserMessage`、`StreamEvent`、`AgentEntry`、`ViewportState`、`SessionHistoryBucket`。

稳定需求：

- `Chat` 默认绑定内置 `main` Agent `Alter0`，作为通用对话入口；`Agent` 页面承载无独立入口的内置 Agent 与用户管理 Agent。
- Web 登录态下，Chat/Agent 按目标 Agent 隔离会话历史，具备独立前端入口的 Agent 不进入通用 Agent 页面历史。
- `Agent` 运行页的会话历史需展示可复制的 8 位短 hash 标识，作为会话级引用与人工排障的稳定标识符。
- Web 入口稳定提供根路径到 Chat 的默认进入、`/chat`、`/login` 与 `/logout`，登录密码启用后受保护页面和 API 统一走同一登录态校验。
- 新会话先使用占位标题，早期多轮内可根据更具体输入自动升级标题，避免长期保留“拉取仓库”“分析仓库”等低辨识度名称。
- 新对话空白会话保持唯一；已有空白会话时，`New Chat` 复用并聚焦该会话。
- 同一会话内同步请求保持顺序一致，系统提供全局并发上限、排队与超时降级。
- SSE 流式响应提供 `start / delta / done` 与保活帧；Agent 工具循环期间还需追加结构化 `process` 事件，把步骤状态实时推送到前端。已进入 Agent 执行链的请求不因浏览器断连、页面切换或前端取消而中断后端执行。
- 流式连接中断时，前端保留已收到的正文并把消息收敛为失败态；若没有可用正文，失败提示需明确提示刷新，并在页面恢复时优先用服务端已持久化的会话消息覆盖本地失败态。本地缓存中残留的 `streaming` 消息不得长期停留在 `In Progress`。
- Agent 执行过程需以结构化 `process_steps` 贯穿 SSE `done`、Task 结果与会话历史持久化，前端优先消费结构化步骤而不是依赖解析 `[agent] action / observation` 文本。
- 消息区支持 Markdown 安全渲染、一键复制最终回复、Process 折叠状态、逐条 patch 与逐帧合并刷新。
- Web 前端所有时间显示统一使用北京时间（`Asia/Shanghai`）与 24 小时制；Cron 创建表单默认时区固定为 `Asia/Shanghai`。
- Web 侧边栏、历史折叠、页面滚动隔离、浅色文档式阅读主题、移动端软键盘跟随、设置底部面板、低功耗轮询与长文本宽度约束作为统一前端体验要求维护。
- 移动端 Web Shell 使用 `VisualViewport` 驱动的 `--mobile-viewport-height` 作为壳体高度来源，浏览器工具栏与键盘状态切换时不出现底部空白或内容裁切。
- 移动端 Chat/Agent 在主输入框与会话设置底部面板之间切换时，不允许保留“键盘 + 设置面板”双重底部占位：打开设置前先释放输入焦点并清理键盘偏移，回到主输入框时先自动收起设置面板；Terminal 在真手机宽度下允许工作区头部工具栏换行，操作按钮不得被长标题挤出可见区域。
- 桌面宽屏下 Chat 消息列与底部输入区需按主工作区宽度自适应扩展，不再长期锁定为 860px 窄列，同时保留可读性上限。

## Agent Capability & Memory

核心对象：`AgentProfile`、`AgentCatalog`、`ReActAgentConfig`、`ToolExecutor`、`Skill`、`MCPServer`、`MemoryFile`、`MemoryContext`、`AgentSessionProfile`、`LongTermMemory`、`DailyMemory`、`SOUL.md`。

稳定需求：

- 运行时统一聚合内置 Agent 与用户管理 Agent；内置 Agent 包括 `main`、`coding`、`writing`、`product-builder` 与 Product 专属总 Agent。
- Agent 采用 ReAct 执行链，负责理解用户目标、吸收 system prompt / Skill / Memory / Product Context，并把具体执行交给 `codex_exec`。
- 稳定工具面包含 `codex_exec`、`search_memory`、`read_memory`、`write_memory` 与运行时收口工具 `complete`；允许委派的 Agent 可额外使用 `delegate_agent`。
- `codex_exec` 通过 stdin 传递最终指令，并携带 `runtime_context`、`product_context`、`product_discovery`、`skill_context`、`mcp_context`、`memory_context` 等结构化上下文字段。
- Agent / ReAct 走 `openai-completions` 多轮工具调用时，assistant `tool_calls` 与后续 `tool` 结果的 `tool_call_id` 必须保持同轮关联，不能在 Provider 适配层丢失。
- Agent Profile 支持名称、system prompt、max iterations、Provider/Model、工具白名单、Skills、MCP 与 Memory Files。
- 每个 Agent 自动拥有私有 file-backed Skill `.alter0/agents/<agent_id>/SKILL.md`，用于沉淀可复用工作模式、输出结构、检查清单与稳定偏好。
- Memory Files 支持 `USER.md`、`SOUL.md`、当前 Agent 私有 `AGENTS.md`、长期 `MEMORY.md / memory.md`、当天与前一天 Daily Memory，并在注入时携带路径、存在状态、可写性、内容与自动召回片段。
- 会话短期记忆、跨会话长期记忆、上下文压缩、天级记忆、强制上下文文件与任务摘要记忆统一构成 Memory 领域能力。
- `Agent -> Memory` 页面提供长期记忆、天级记忆、强制要求、任务历史与说明文档的只读可视化入口，并支持任务摘要重建。

## Task, Terminal & Workspace

核心对象：`Task`、`TaskSummary`、`TaskLog`、`ArtifactRef`、`TerminalSession`、`TerminalTurn`、`TerminalStep`、`Workspace`、`CodexThreadID`、`RuntimeHeartbeat`。

稳定需求：

- 高复杂度、长耗时或产物型请求可切换为异步 Task，先返回任务卡片，再通过任务视图、日志流与会话回写完成闭环。
- Task 需建立 `session_id`、`source_message_id`、`channel_type`、`trigger_type`、`correlation_id`、Cron 触发信息与产物引用的标准映射。
- Task 观测台支持列表、详情抽屉、来源筛选、日志 SSE、游标续读、日志回补、retry/cancel、交互式续写、任务-会话双向跳转与完成结果回写。
- Task 记忆视图支持任务摘要、任务详情、日志下钻、产物引用与摘要重建，用于把历史任务纳入长期上下文召回。
- Codex CLI 长任务按心跳续租运行窗口；列表与详情展示 `Last Heartbeat` 和 `Timeout Window`。
- Web 会话不直接暴露本地文件路径，产物通过引用、下载或预览接口交付。
- 默认工作区按执行上下文隔离：Chat/Agent 使用 `.alter0/workspaces/sessions/<session_id>`，Task 使用其会话下的 `tasks/<task_id>`，Terminal 使用 `.alter0/workspaces/terminal/sessions/<terminal_session_id>`。
- Terminal 是独立会话式终端代理，持久化 Codex CLI 线程标识、会话状态、标题、工作区、日志与步骤视图索引。
- Terminal API 支持会话创建、列表、恢复、输入、关闭、删除、详情读取以及 turn/step 明细读取，前端可按步骤展开或检索执行细节。
- Terminal 会话态统一为 `ready / busy / exited / interrupted`，执行态在 turn/step 维度维护 `running / completed / failed / interrupted`；运行态退出后保留历史，继续发送即可恢复。
- Terminal 支持 `Close` 与 `Delete`：关闭仅退出运行态，删除同步清理状态文件和独立工作区。
- Terminal 历史在同一 Web 登录态下跨设备共享，不按浏览器 client 标识隔离；不设置产品级会话数量上限或固定超时淘汰。
- Terminal 移动端、输入稳定性、滚动导航、Process 折叠、一键复制、长输出阅读、轮询降频与缓存写入节奏作为 Terminal 子域体验要求维护。

## Product Domain

核心对象：`Product`、`ProductDraft`、`ProductAgentDraft`、`ProductMasterAgent`、`ProductWorkspace`、`ProductSpace`、`TravelGuide`、`ProductDiscovery`。

稳定需求：

- Product 是业务产品域的一等对象，用于承载产品定义、主 Agent、入口路由、知识源、产物类型、详情页空间与可选 supporting agents。
- `Products` 页面提供 `Workspace` 与 `Studio` 视图；内置 Product 只读展示，用户管理 Product 支持新增、编辑、删除、停用、生成草稿、审核与发布。
- Draft Studio 通过 `product-builder` 生成或扩展 Product 草稿；新草稿默认采用单主 Agent，把可复用领域规则沉淀到主 Agent system prompt 与 Skill。
- 每个已发布 Product 绑定唯一 `master_agent_id`，Product 总 Agent 统一采用 Agent 协助 / Codex 执行模型，历史 supporting agents 仅做兼容保留。
- 默认 `main` Agent 可做 Product 发现，并在执行型请求中自动切换到目标 Product 总 Agent，同时注入 Product Context 与路由元数据。
- 已发布且公开的 Product 提供公共目录、详情、Workspace、Space 详情、独立 HTML 页面与 Product 消息执行入口。
- `travel` 是首个内置 Product，绑定唯一 `travel-master`，支持城市页创建/修改、结构化攻略、独立 HTML 城市页、Workspace Chat 与 Agent 执行失败时的本地解析回退。
- `travel-master` 使用私有 Skill `.alter0/agents/travel-master/SKILL.md` 沉淀城市页、行程、地铁、美食与地图输出规则；稳定偏好写入 Skill，一次性行程约束写入目标城市页数据。

## Control, Operations & Governance

核心对象：`ChannelConfig`、`SkillConfig`、`AgentProfile`、`ModelProvider`、`EnvironmentConfig`、`RuntimeInstance`、`DeploymentBaseline`、`EngineeringPolicy`。

稳定需求：

- Control API 管理 Channel、Capability、Skill、MCP、Agent Profile、Product、Cron Job、Model Provider 与 Environment 配置，并保留 Capability 生命周期审计。
- Channels 入口归属 Settings 模块，旧直达路由保持兼容。
- Models 控制面支持 OpenAI Compatible 与 OpenRouter Provider，支持 `/responses` 与 `/chat/completions`，支持 base URL、API Key 保留语义、Provider 路由偏好、默认项自动收敛与历史缺密钥配置恢复。
- `openai-completions` 适配层必须正确序列化 assistant `tool_calls` 与 tool output，兼容严格校验工具消息配对关系的上游 Provider。
- Environments 页面支持 Web/Queue、Async Tasks、Terminal、Session Memory、Persistent Memory 与 LLM 运行参数可视化配置、配置审计、在线实例启动时间与 commit hash 展示、运行时重启、远端 master 快进同步、候选二进制构建、readyz 探活与失败回滚。
- 公网部署基线要求服务绑定 localhost、启用 Web 登录密码、统一 `HOME=/var/lib/alter0`，并通过 Nginx 做反向代理。
- 服务内 GitHub 交付要求运行账户具备 GitHub App token helper、`gh` 包装器、SSH 提交签名、稳定 PATH 与 Codex CLI 可用认证。
- Node/Playwright 测试链路通过运行账户级工具链初始化，保证 Codex CLI 可执行 `internal/interfaces/web/frontend` 的构建与单测，以及 `internal/interfaces/web` 的 Playwright E2E。
- 研发流程遵循 TDD：功能新增、缺陷修复、行为调整与重构默认先以测试表达目标行为，再完成实现与重构；纯文档、注释、格式化、依赖元数据或无法自动化验证的变更需说明免测原因与替代验证。

## 维护规则

- 新需求必须先选择领域，再选择子域；无法归类时优先补充领域模型，而不是新增线性编号。
- 一个需求只允许有一个主归属领域；跨领域影响通过 `依赖与边界` 说明，不复制成多个重复需求。
- 用户可见行为、交互方式、入口路由、执行模式、返回结构或默认策略发生变化时，同步更新 `README.md`。
- 需求细节、接口、状态、验收和边界放入对应 `requirements-details/*.md` 文件；技术方案、包边界、调用链路、存储、观测和测试策略放入 `technical-solution.md` 的同名领域下；`requirements.md` 只维护稳定总览与领域索引。
