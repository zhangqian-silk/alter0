# Technical Solution

> Last update: 2026-04-10

`alter0` 的技术方案按与需求清单一致的领域模型维护。后续新增或调整需求时，技术方案必须落到对应领域与子域，不再按时间顺序、任务编号或零散专题堆叠。

## 维护规则

- 需求归属先确定领域路径，例如 `runtime.orchestration.intent`、`agent.execution.codex-exec`、`product.travel.workspace`。
- 技术方案使用相同领域路径补充实现信息，包括包边界、核心对象、调用链路、接口契约、存储形态、错误处理、观测字段和测试策略。
- 一个能力只允许有一个主归属领域；跨领域影响通过“依赖与边界”记录，不复制成多个重复方案。
- 影响架构边界、数据结构、接口、执行链路、存储、部署或研发治理的需求变更，必须同步更新本文件。
- 用户可见行为变化继续同步 `README.md`，稳定需求口径同步 `docs/requirements.md`，实现细节同步 `docs/requirements-details/*.md`。

## 领域方案索引

| 领域 | 主包/模块 | 技术方案重点 |
| --- | --- | --- |
| Runtime & Orchestration | `internal/interfaces`、`internal/shared`、`internal/orchestration`、`internal/execution/domain`、`internal/scheduler` | 统一消息、意图路由、执行端口、调度触发、观测与健康检查 |
| Conversation & Session Experience | `internal/interfaces/web`、`internal/session`、Web static assets | Chat/Agent 会话、SSE、历史隔离、移动端视口、消息渲染 |
| Agent Capability & Memory | `internal/agent`、`internal/execution`、`internal/llm`、`internal/orchestration` | Agent Catalog、ReAct、工具执行、Skills/MCP、Memory Context、上下文压缩 |
| Task, Terminal & Workspace | `internal/task`、`internal/tasksummary`、`internal/terminal`、`.alter0/workspaces` | 异步任务、日志流、心跳、产物交付、Terminal 会话、工作区隔离 |
| Product Domain | `internal/product`、`internal/interfaces/web/product_*`、`cmd/alter0/builtin_skills.go` | Product 目录、Draft Studio、Product 总 Agent、Workspace、Travel 产品域 |
| Control, Operations & Governance | `internal/control`、`internal/llm`、`cmd/alter0`、`scripts`、`docs/deployment` | 控制面配置、模型 Provider、运行时重启、部署凭据、测试与 TDD 约束 |

## Runtime & Orchestration

### 包边界

- `internal/interfaces/*` 负责外部输入适配，只生成内部统一消息，不承载业务路由。
- `internal/shared/domain` 承载 `UnifiedMessage`、`OrchestrationResult` 等跨领域消息对象。
- `internal/orchestration/domain` 承载 `Intent`、`Command` 等编排领域模型。
- `internal/orchestration/application` 承载意图识别、命令路由、自然语言执行分发。
- `internal/execution/domain` 定义执行端口和运行时上下文契约。
- `internal/scheduler` 负责 Cron 配置、触发与回注编排。

### 调用链路

```text
CLI / Web / Cron
  -> UnifiedMessage
  -> Orchestrator
  -> CommandHandler | ExecutionPort
  -> OrchestrationResult
  -> Interface response / Session persistence / Task handoff
```

### 技术约束

- 外部输入必须先归一为 `UnifiedMessage`，再进入编排层。
- 命令路由优先于复杂度评估和模型执行。
- Cron 触发不直接调用执行器，必须复用编排链路。
- Cron runs 接口通过 Session history 按 `trigger_type=cron` 与 `job_id` 查询触发会话，不另建独立运行记录存储。
- `ExecutionPort` 是自然语言执行能力的稳定边界，具体实现可替换。
- trace、session、message、correlation 字段贯穿日志、指标、会话与任务。

### 验证策略

- 领域对象测试覆盖消息字段归一、路由结果和错误编码。
- 编排应用测试覆盖命令优先、自然语言分发、Cron 回注。
- 接口测试覆盖 CLI/Web 输入到 `UnifiedMessage` 的转换。
- Go 单测用例说明按 `docs/testing/unit-test-cases.md` 与各 Go 包路径下的 `TEST_CASES.md` 维护，并按 Runtime、Conversation、Agent、Task、Product、Control 领域路径归档。

## Conversation & Session Experience

### 包边界

- `internal/session/domain` 定义会话与消息数据结构。
- `internal/session/application` 负责会话持久化、历史查询和删除清理。
- `internal/interfaces/web` 负责 HTTP、SSE、Web 登录、页面路由和前端静态资源。
- 前端静态资源处理展示、输入、缓存、轮询和视口状态；会话恢复阶段允许把残留 `streaming` 消息归一为失败态或任务态，但不改写后端领域事实。

### 调用链路

```text
Web input
  -> Web handler
  -> UnifiedMessage
  -> Orchestrator / Agent execution
  -> StreamEvent | JSON response
  -> Session history
  -> Incremental UI patch
```

### 技术约束

- Chat 默认绑定 `main` Agent，Agent 页面按目标 Agent 隔离会话历史。
- 根路径 `/`、`/chat`、`/login`、`/logout` 是稳定 Web Shell 入口；登录密码启用后页面和 API 共享同一登录态校验。
- SSE 连接只负责回传，前端断连不得取消已进入 Agent 执行链的后端任务。
- 会话标题升级、空白会话唯一性、历史折叠和页面滚动状态属于 Conversation 子域。
- `chat.js` 读取本地缓存时先归一残留 `streaming` 消息；无 `task_id` 的消息转失败态，带真实 `task_id` 的消息恢复到任务轮询链路。
- Web 流式网关把 `shareddomain.StreamEvent` 映射为 SSE：输出事件写成 `delta`，结构化步骤事件写成 `process`，最终 `done` 继续携带完整 `process_steps` 用于收口与持久化恢复。
- 流式连接异常收尾时优先保留已收到正文；只有在没有可用正文时才渲染带刷新提示的失败文案。
- 页面初始化会对当前活动会话执行一次轻量恢复：仅当本地存在流式失败态助手消息时，前端调用 `/api/sessions/{session_id}/messages`，以服务端已持久化消息覆盖本地失败态。
- Agent 执行器在运行期把工具动作与观察收敛为结构化 `process_steps`，经 `ExecutionResult -> OrchestrationResult -> session.RouteResult` 透传到 SSE `done`、Task 结果和 `/api/sessions/{session_id}/messages`。
- Agent 执行器在工具循环期间直接发出 `process` 事件，前端按 `process_step.id` 原地更新步骤状态；历史旧消息仍保留基于文本标记的回退解析。
- `chat.js` 渲染 Agent 消息时优先读取 `process_steps`；仅在缺失结构化步骤时才回退到历史 `[agent] action / observation` 文本解析，以兼容旧会话。
- 移动端输入区以 `VisualViewport` 为有效视口来源，并按聚焦、键盘和页面可见性降频刷新。
- 移动端 App Shell 高度使用 `--mobile-viewport-height`，由 `chat.js` 同步 `VisualViewport` 计算值，避免浏览器工具栏变化导致底部留白或内容裁切。
- 移动端 Chat/Agent 在输入框与会话设置底部面板之间切换时，`chat.js` 需先归一底部交互层：打开移动端设置面板前先 blur 当前输入并重新同步 `--keyboard-offset`，主输入框重新聚焦时则先关闭设置面板，再执行键盘贴底和输入框对齐逻辑。
- `chat.js` 内所有前端时间展示统一走同一北京时间格式化器，固定 `timeZone=Asia/Shanghai`、`hourCycle=h23`；时间标签输出 `HH:mm`，绝对时间输出 `YYYY-MM-DD HH:mm:ss`。
- Cron 创建表单默认时区直接复用同一前端常量 `Asia/Shanghai`，不再依赖浏览器本地时区探测。
- `agent-runtime` 会话列表前端按 `sha1(session_id)[:8]` 生成短 hash，展示在会话卡片内并通过统一 `data-copy-value` 复制链路写入剪贴板，保持与 Agent Session Profile / 预览域名使用的短标识一致。
- Markdown 渲染必须避免原始 HTML 透传；长路径、代码块和 diff 只在内容块内部滚动。

### 验证策略

- Web handler 测试覆盖会话创建、历史隔离、流式事件和取消语义。
- 前端 E2E 覆盖 Chat、Agent、移动端输入、设置面板和长会话渲染。
- 回归测试优先覆盖空白会话重复、软键盘残留空白、整段列表重建、断流恢复与残留 `In Progress` 等高频问题。

## Agent Capability & Memory

### 包边界

- `internal/agent/application` 负责内置 Agent 与用户 Agent Catalog 聚合。
- `internal/control/domain` 与 `internal/control/application` 负责 Agent Profile、Skill、MCP 的控制面配置。
- `internal/execution/application` 负责运行时上下文解析，包括 Skill、MCP、Memory 和 Agent Session Profile。
- `internal/execution/infrastructure` 负责 ReAct、Codex CLI、工具执行与模型适配实现。
- `internal/orchestration/application` 负责会话记忆、长期记忆、压缩和任务摘要召回。

### 调用链路

```text
Agent message
  -> Agent Catalog / Profile resolution
  -> Runtime context resolution
  -> ReAct loop
  -> ToolExecutor
  -> codex_exec | memory tools | delegate_agent | complete
  -> Final response + Process
```

### 技术约束

- Agent 负责理解与驱动，具体文件、仓库、Shell、页面产出统一通过 `codex_exec`。
- `codex_exec` 使用 stdin 传递执行指令，结构化上下文按需注入，不通过命令行拼接长上下文。
- Memory Files 注入需要携带路径、存在状态、可写性、内容快照和自动召回片段。
- `internal/llm/infrastructure` 的 `openai-completions` 适配层需要把 assistant 历史消息中的 `tool_calls` 一并序列化，并与后续 `tool` 消息的 `tool_call_id` 保持稳定配对；否则 Provider 会把该轮请求判定为非法工具消息序列。
- 私有 `AGENTS.md`、私有 Skill、Agent Session Profile 分别承担协作边界、可复用打法、会话画像职责，不混写一次性任务细节。
- `search_memory`、`read_memory`、`write_memory` 只操作已解析进 `memory_context` 的记忆文件。
- Agent Memory Web 聚合接口只读返回长期记忆、天级记忆、强制上下文与说明文档；任务摘要刷新走 Task summary 子域接口。

### 验证策略

- Agent Catalog 测试覆盖内置 Agent 与用户 Agent 聚合、保留 ID 冲突。
- Execution 应用测试覆盖 Skill/MCP/Memory Context 注入。
- Infrastructure 测试覆盖 `codex_exec` stdin、ReAct 迭代上限、Process 输出和工具错误收口。
- Memory 测试覆盖短期回填、长期召回、Markdown 编解码、压缩和任务摘要深检索。

## Task, Terminal & Workspace

### 包边界

- `internal/task/domain` 定义任务状态、来源字段、摘要与执行元数据。
- `internal/task/application` 负责异步执行池、复杂度预判、任务生命周期和心跳续租。
- `internal/tasksummary/application` 负责任务摘要存储和运行态 Markdown 记录。
- `internal/terminal/domain` 定义 Terminal 会话态、turn 和 step。
- `internal/terminal/application` 负责 Terminal 会话持久化、恢复、输入续写和工作区分配。

### 调用链路

```text
High-complexity message
  -> Task acceptance
  -> Async executor
  -> Workspace
  -> Codex CLI / Agent execution
  -> Task logs + heartbeat + artifacts
  -> Session result summary
```

```text
Terminal input
  -> Terminal session store
  -> Codex CLI resume/start
  -> Turn/step append
  -> Terminal view model
  -> Web polling / stream response
```

### 技术约束

- Task 需要保存来源字段，支持从任务回会话、从会话查任务，并支持按触发类型、通道、来源消息与结果消息过滤。
- 长任务通过心跳续租运行窗口，浏览器 SSE 保活与后台心跳分离。
- Control 任务交互式续写通过追加输入创建 follow-up Task，不直接改写原任务执行记录。
- Web 不直接暴露本地文件路径，产物通过引用、下载或预览接口交付。
- Task 产物列表响应需要过滤本地 URI；下载和预览由任务接口按 artifact id 读取并输出安全响应头。
- Memory 任务视图读取 Task 与 task summary 数据，支持任务摘要重建，但不直接执行 retry/cancel。
- 工作区按 Chat/Agent、Task、Terminal 分层隔离，删除会话或 Terminal 时同步清理对应目录。
- Terminal 会话态与 turn/step 执行态分离，历史 `running / starting` 需要兼容归一。
- Terminal 会话详情聚合 turn 摘要；step 明细按 `session_id / turn_id / step_id` 单独读取，避免会话列表一次性加载大块执行日志。
- Terminal 跨设备共享同一 Web 登录态下的服务端会话历史，不再按 browser client 分桶。
- `chat-terminal.css` 在真手机宽度下允许 Terminal 工作区头部切换为多行排布：标题最多两行，状态与操作工具栏按可用宽度换行，避免横向溢出。

### 验证策略

- Task 应用测试覆盖复杂度分流、并发上限、心跳续租、来源字段和删除清理。
- Web 测试覆盖任务列表、详情、日志流、retry/cancel、产物下载/预览、Memory 任务摘要重建和会话回链。
- Terminal 应用测试覆盖创建、恢复、续写、关闭、删除、详情读取、step 明细、状态归一和工作区分配。
- E2E 测试覆盖 Terminal 移动端输入、滚动、Process 折叠和跨设备历史口径。

## Product Domain

### 包边界

- `internal/product/domain` 定义 Product、Draft、TravelGuide 等领域对象。
- `internal/product/application` 负责内置 Product、托管 Product、草稿生成、发布和 Travel 领域服务。
- `internal/interfaces/web/product_*` 负责 Product 控制面、公开入口、Workspace 和独立 HTML 页面。
- `cmd/alter0/builtin_skills.go` 提供内置 Product 相关 Skill 初始化。

### 调用链路

```text
Product request
  -> Product discovery / public product lookup
  -> master_agent_id binding
  -> Product Context injection
  -> Agent execution
  -> Product artifact / workspace space
  -> Public result / HTML page
```

### 技术约束

- Product 是一等领域对象，用户管理 Product 与内置 Product 的写权限必须区分。
- 新草稿默认采用单主 Agent，supporting agents 只作为兼容字段保留。
- 发布链路必须把主 Agent 归一到当前稳定 Agent 执行模型，并补齐工具、Skill 与 Memory Files。
- Product 总 Agent 是已发布 Product 的唯一默认执行入口。
- `main` Agent 跨 Product 调度只读取运行态允许公开的摘要，不暴露草稿、凭据或审核记录。
- `travel` 领域规则沉淀在 `travel-master` system prompt 与私有 Skill，不反向污染通用 Agent 规则。

### 验证策略

- Product domain/application 测试覆盖 Product CRUD、内置/用户来源、版本、草稿发布和冲突提示。
- Web 测试覆盖 Product 公共目录、控制面详情、Workspace、Space HTML 和 Product 消息入口。
- Travel 测试覆盖攻略创建、修订、Workspace Chat、Agent fallback 与结构化字段完整性。

## Control, Operations & Governance

### 包边界

- `internal/control` 管理 Channel、Capability、Skill、MCP、Agent Profile 和 Environment 配置。
- `internal/llm` 管理 Model Provider、上游 API type、OpenRouter 扩展与密钥状态。
- `cmd/alter0` 管理启动、supervisor、重启、内置配置和运行时 metadata。
- `scripts` 承载运行账户凭据、Node/Playwright 工具链和部署初始化脚本。
- `docs/deployment` 承载 Nginx 与部署权限说明。

### 调用链路

```text
Control UI / API
  -> Config / Capability service
  -> Local storage
  -> Runtime resolver
  -> Execution / Agent / Scheduler
```

```text
Environment restart
  -> Web confirmation
  -> supervisor
  -> optional git fast-forward
  -> build candidate binary
  -> readyz probe
  -> switch or rollback
```

### 技术约束

- Control 面只能管理运行时配置，不绕过编排层直接执行业务请求。
- Skill 与 MCP 专用接口需要复用统一 Capability 数据结构；Capability 审计记录生命周期动作，供控制面按类型查询。
- Models 控制面需要保持空 API Key 语义、占位值过滤、禁用态恢复和默认 Provider 收敛。
- Environment registry 按 Web & Queue、Async Tasks、Terminal、Session Memory、Persistent Memory、LLM 模块声明 key、类型、默认值、校验规则、敏感性与生效方式。
- Environment 配置更新写入 audit store，控制面按时间倒序读取变更记录。
- LLM 运行参数 `llm_temperature`、`llm_max_tokens`、`llm_react_max_iterations` 通过 Environment 配置即时或重启后参与运行时解析，仍受 Provider 与模型能力约束。
- Runtime 重启必须由 supervisor 托管，候选实例通过 readyz 后才切换。
- systemd 基线统一 `HOME=/var/lib/alter0`，确保 Codex、gh、git signing、Node/Playwright 工具链使用同一运行账户上下文。
- 提交签名问题不得通过关闭签名绕过。
- 技术文档、需求文档和 README 更新按领域同步，避免需求与方案分离。

### 验证策略

- Control 测试覆盖 Channel、Capability、Skill、MCP、Agent、Environment 配置持久化、Capability 审计和 Environment audit。
- LLM 测试覆盖 Provider 创建、更新、缺失密钥恢复、默认项收敛和 OpenRouter 字段。
- Runtime supervisor 测试覆盖候选版本构建、readyz 切换、失败回滚和 metadata 展示。
- 文档治理变更至少运行 Markdown 引用与空白检查；代码变更按 TDD 运行对应包或全量测试。
- Go 单测新增或调整时，同步维护 `docs/testing/unit-test-cases.md` 与对应 Go 包路径下 `TEST_CASES.md` 的覆盖范围和边界说明。

## 变更模板

后续需求或技术方案变更使用以下字段维护：

```markdown
## <Domain>

### <Subdomain / Capability>

- 需求路径：
- 主归属领域：
- 涉及包：
- 核心对象：
- 调用链路：
- 接口契约：
- 存储与迁移：
- 错误与降级：
- 观测字段：
- 测试策略：
- 依赖与边界：
```
