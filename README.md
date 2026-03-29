# alter0

一个面向个人部署的 Agent 运行时骨架，采用 DDD 分层，强调可组合、可观察、可演进。

作为项目负责人，我把 `alter0` 定位为一套“先跑通，再扩展”的基础设施：

1. 先把消息链路打通（CLI/Web/Cron -> Orchestration -> Execution）。
2. 再把控制面补齐（Skill/Channel/Cron 配置与治理）。
3. 最后平滑演进到多执行器、多通道、多环境部署。

## Why alter0

很多 Agent 项目在早期就耦合了大量能力，导致难以迭代。`alter0` 的原则是：

1. 最小闭环优先：先有可运行链路，再谈复杂能力。
2. 领域边界清晰：Gateway、Orchestration、Execution、Control 各司其职。
3. 全链路可观测：每条消息都有 trace/session/message 维度。
4. 演进友好：默认单机，无鉴权；后续可以平滑加存储、鉴权和多租户。

## Documentation

详细技术文档见 [docs](./docs/README.md)：

1. [Architecture Design](./docs/architecture.md)
2. [Technical Solution](./docs/technical-solution.md)
3. [Requirements](./docs/requirements.md)

## Output Convention

1. 所有临时产物统一写入 `output/` 目录。
2. 包含但不限于测试结果、截图、Smoke 测试记录、调试导出文件、临时脚本输出与本地排查产物。
3. 不在仓库根目录或业务目录散落创建临时文件、日志文件与一次性调试文件。
4. 需要保留的正式文档、示例数据与工程代码，仍按原有目录结构维护，不放入 `output/`。

## Architecture

系统由两条主线组成：

1. Data Plane（执行面）
- 负责处理消息通信与任务执行。
- 路径：`Channel Adapter -> UnifiedMessage -> Orchestrator -> Executor`。

2. Control Plane（控制面）
- 负责配置 `Channel / Skill / Agent / Product / CronJob`。
- 通过 API 管理运行时行为，不直接绕开编排层。

核心链路：

1. CLI/Web/定时任务输入统一转换为 `UnifiedMessage`。
2. `IntentClassifier` 判断是命令还是自然语言。
3. 命令交由 `CommandRegistry` 与 `CommandHandler` 执行。
4. 自然语言请求交由 `ExecutionPort` 执行。
5. 定时任务由 `SchedulerManager` 触发，并复用同一编排链路。

## Repository Layout

```text
cmd/alter0                         # 程序入口（web/cli）
internal/interfaces/cli            # CLI 适配器
internal/interfaces/web            # Web 适配器 + Control API
internal/control/domain            # Control 领域模型（Channel/Skill）
internal/control/application       # Control 应用服务（配置增删改查）
internal/product/domain            # Product 领域模型（产品定义与 Agent 矩阵）
internal/product/application       # Product 应用服务（内置 Product + 托管 CRUD）
internal/scheduler/domain          # 定时任务模型
internal/scheduler/application     # 定时任务管理器（触发到编排层）
internal/orchestration/domain      # 编排领域模型（Intent/Command）
internal/orchestration/application # 编排应用服务
internal/orchestration/infrastructure
internal/execution/domain          # 执行领域接口
internal/execution/application     # 执行应用服务
internal/execution/infrastructure  # NL 执行器实现（示例）
internal/storage/infrastructure    # 存储适配实现（本地文件等）
internal/shared/domain             # UnifiedMessage / OrchestrationResult
internal/shared/infrastructure     # ID、日志、metrics
```

## Built-in Commands

1. `/help`：查看命令列表
2. `/echo ...`：回显参数
3. `/time`（别名 `/now`）：输出 UTC 时间（RFC3339）

## Natural Language Handling

自然语言请求按用户交互形态分为 `Chat`、`Agent` 与 `Terminal` 三类：

1. `Chat`
- 面向 Web 会话消息。
- 默认绑定内置 `Alter0`（`main`），作为通用对话入口。
- Web 登录后，Web 对话页按目标 Agent 维护独立 Session 历史；带独立前端入口的 Agent 不进入通用 `Agent` 页的会话历史。
- 运行时配置收敛在输入框底部操作栏：`Provider / Model`、`Tools / MCP`、`Skills` 都在发送区附近完成。
- `Provider / Model`、`Tools / MCP`、`Skills` 可在会话过程中继续调整，并作用于后续发送的消息。
- `Alter0` 默认使用 ReAct 执行链，可直接完成通用任务，并在需要时调度专项 Agent。
- 选中的原生工具会在模型调用时作为 function tools 注入，当前内置工具包括 `list_dir`、`read`、`write`、`edit`、`bash`。
- `OpenAI` Provider 支持按 `api_type` 选择上游接口：`openai-responses` 走 `/responses`，`openai-completions` 走 `/chat/completions`；配置自定义 `base_url` 时，需要目标服务兼容所选接口。
- 默认 Provider 只会落在已启用配置上；若默认 Provider 被禁用、删除或历史配置已失效，系统会自动切换到下一可用 Provider，无可用项时清空默认值。
- 复杂度评估阶段会优先复用当前消息选中的 `Provider / Model`；未显式选择时，回退到默认 Provider 与默认模型。
- 默认走实时执行。
- 流式对话会先直接启动回复；复杂度评估与回复并行进行。
- 当请求复杂度较高且仍在执行中时，系统会中途转为后台 `Task` 执行，并先返回一条任务说明消息，包含任务目标、执行计划与任务入口。
- 聊天气泡支持常用 Markdown 渲染，包括标题、列表、引用、链接、行内代码与代码块；原始 HTML 不直接透传。
- Chat 消息会标注实际回复来源，用于区分当前内容来自模型执行链还是 `Codex CLI` 执行链。

2. `Agent`
- 面向“先执行再汇报”的目标型任务。
- 请求进入后会创建一个 ReAct 执行环，以当前任务为目标持续推进。
- 运行时统一维护 `Agent Catalog`：同时聚合系统内置 Agent 与用户管理的 Agent Profile。
- 当前内置 Agent 包括：
  - `main`：默认对话主 Agent `Alter0`，可调度专项 Agent
  - `coding`：面向仓库分析、代码修改与验证
  - `writing`：面向文档、文案与结构化写作
  - `product-builder`：用于创建和扩展 Product 定义与子 Agent 矩阵
  - `travel-master`：负责旅游 Product 的总控编排
- Web `Chat` 页面默认绑定 `Alter0`；`Agent` 页面提供统一 Agent 运行入口，用于承载未占用独立前端入口的内置 Agent 与用户管理 Agent。
- Agent 默认可使用 `list_dir`、`read`、`write`、`edit`、`bash`、`codex_exec` 等工具，并依据执行结果决定继续推进还是完成收口；支持 `delegate_agent` 时可向其他可委派 Agent 下发子任务并回收结果。
- 每个 Agent 可独立配置名称、system prompt、tool 白名单、Skill 选择、MCP 选择与 Memory Files 选择。
- Agent 可按 Profile 勾选 `USER.md`、`SOUL.md`、`AGENTS.md`、长期 `MEMORY.md / memory.md`、当天与前一天 Daily Memory；所选文件会在执行前以结构化 `memory_context` 注入 Agent 与 Codex 执行链路。
- 注入内容同时携带可写文件路径，Agent 可继续通过 `read`、`write`、`edit` 工具直接维护这些记忆文件，适配“先读规则、再执行、需要时落盘更新”的工作流。
- Web 端将用户管理的 Agent Profile 放在 `Agent Profiles` 页面；系统内置 Agent 由服务注册，不能通过控制面覆盖或删除。
- Agent 的 `id`、`version` 等系统字段由服务端统一生成和维护，管理页不要求用户手填。
- `Agent` 页面继续保留独立执行入口与配置，并按当前选中 Agent 展示独立 Session 历史；带独立前端入口的 Agent 不进入该页历史。

3. `Terminal`
- 面向交互式终端会话。
- 仍属于自然语言处理，但使用独立上下文边界。
- 默认仅注入运行时必需上下文，不复用 Chat 会话记忆与长期记忆。
- 每个 Terminal 会话使用独立工作区目录，不再默认落在仓库根目录。
- Terminal 会持久化 Codex CLI 线程标识与会话状态；运行态退出后保留原会话历史，继续发送即可在同一会话内恢复。
- Terminal 工作区头部同时提供 `Close` 与 `Delete`：`Close` 仅退出当前运行态并保留会话历史与线程标识，`Delete` 会同时移除会话记录、持久化状态文件与对应工作区目录。
- 浏览器端会持久化 Terminal client 标识；页面刷新、重新打开路由或临时 `sessionStorage` 丢失后，仍优先复用原 Terminal 会话归属，不把仍可恢复的会话误判为丢失。
- Terminal 不再设置产品级会话数量上限或固定超时淘汰策略。
- 移动端访问 Terminal 时，轮询刷新不会重建已聚焦输入框；输入法每次确认词句后，若输入框仍保持聚焦，页面继续延迟重绘并保持当前位置，直到失焦后再刷新视图。
- Terminal 轮询刷新采用会话列表与工作区局部更新；当用户正在滚动输出区时，前端保留原消息滚动容器与滚动位置，不再按周期整块重建终端视图。
- Terminal 浏览器侧会话缓存采用滚动感知的延后持久化；输出持续增长时优先让出主线程给滚动与渲染，再在滚动停顿后写入本地存储。
- Terminal 移动端采用紧凑 `Header + 独立滚动消息区 + 底部输入条` 结构；顶部通用栏保留 `Menu`，并在同一行右侧提供 `Sessions / New`，工作区头部收敛为单行会话摘要操作栏；发送区使用自适应输入框与紧凑发送按钮，长输出支持展开/收起、用户问题吸顶与快速回到底部。
- Terminal 最终输出中的 Markdown 链接按链接文本渲染为可点击链接，不再直接暴露整段 Markdown 源码与长路径。
- Terminal 与 Chat 的长路径、超长单词、代码块和 diff 仅允许在内容块内部横向滚动，不再把外层消息卡片或聊天框撑出边框。
- 键盘弹起后输入区继续贴底可输入，页面不会因刷新回到顶部。

补充说明：

1. `Chat / Agent / Terminal` 只要落到 `Codex CLI` 执行链，都要求服务运行账户本身具备可用的 Codex / OpenAI 认证。
2. 若服务账户缺少认证，Web 端会快速返回认证失败，而不会长时间保持等待态。

## Product Model

Product 用于承载“某个业务产品 / 应用的总 Agent 与子 Agent 矩阵”，当前稳定行为如下：

1. Web 端新增 `Products` 页面，统一承载 Product 目录管理与 `Draft Studio`；同一入口内可维护 Product 定义、主 Agent、入口路由、知识源、产物类型与子 Agent 矩阵，并生成待审核草稿。
2. Product 由服务端统一维护 `id`、`version` 与 `owner_type`；内置 Product 只读展示，用户管理 Product 支持新增、编辑、删除、生成草稿矩阵并在审核后发布。
3. `Draft Studio` 当前提供 `POST /api/control/products/generate`、`GET /api/control/products/drafts`、`GET /api/control/products/drafts/{draft_id}`、`PUT /api/control/products/drafts/{draft_id}`、`POST /api/control/products/drafts/{draft_id}/publish`、`POST /api/control/products/{product_id}/matrix/generate`，发布时会同时落地 Product 与对应的托管 Agent 矩阵。
4. `Alter0` 在默认 `main` Agent 下会先做 Product 发现；命中 Product 后会补充 `matched_product_ids`、`selected_product_id`、`selection_reason`、`master_agent_id`、`product_execution_mode` 等元数据，并在执行型请求中自动切换到目标 Product 的 `master_agent_id`。
5. 当前内置 `travel` Product 默认公开可见，并绑定以下 Agent 角色：
   - `travel-master`
   - `travel-city-guide`
   - `travel-route-planner`
   - `travel-metro-guide`
   - `travel-food-recommender`
   - `travel-map-annotator`
6. `travel` Product 面向按城市聚合的旅游攻略场景，预留 `city_guide`、`itinerary`、`map_layers` 等产物类型以及 `city_profile`、`poi_catalog`、`metro_network`、`food_catalog` 等知识源。
7. 已发布且公开的 Product 提供独立执行入口：`POST /api/products/{product_id}/messages` 与 `POST /api/products/{product_id}/messages/stream`；请求会自动绑定到该 Product 的 `master_agent_id`，并注入 `alter0.product_context` 与 `alter0.product.discovery`。
8. `travel` 额外提供结构化攻略接口：`POST /api/products/travel/guides`、`GET /api/products/travel/guides/{guide_id}`、`POST /api/products/travel/guides/{guide_id}/revise`；攻略输出稳定包含景点、地铁、路线、美食、说明与地图图层字段，便于后续接地图高亮和路线渲染。

## Workspace Model

默认运行策略保持 `danger-full-access`，但不同入口会落到各自独立的工作区目录作为默认执行目录：

1. `Chat / Agent`
- 会话级工作区：`.alter0/workspaces/sessions/<session_id>`

2. `Async Task`
- 任务级工作区：`.alter0/workspaces/sessions/<session_id>/tasks/<task_id>`

3. `Terminal`
- 终端会话级工作区：`.alter0/workspaces/terminal/sessions/<terminal_session_id>`
- 终端会话状态：`.alter0/state/terminal/sessions/<terminal_session_id>.json`

说明：

1. 独立工作区用于隔离默认执行目录与运行时产物，不等同于文件系统权限收缩。
2. 当前默认仍为 `danger-full-access`，因此是否可访问其他绝对路径，仍取决于宿主机环境与运行账户权限。
3. 原生工具的相对路径默认按仓库根目录解析；需要操作会话工作区时，可在工具参数中显式指定 `base=workspace`。

其中 `Chat` 再细分为两种执行方式：

1. `Sync`
- `POST /api/messages`：普通 JSON 一次性返回结果。
- `POST /api/messages/stream`：通过 SSE 流式返回 `start / delta / done` 事件；复杂度评估与回复并行进行，若请求在评估完成时仍需长耗时执行，会在同一条流中切换为异步任务并返回任务受理结果。
- 同一会话内的同步请求保持串行；当上一条同步执行尚未结束时，后续用户消息会继续等待并按序执行，不再因为默认队列等待时间直接返回 5 秒超时。
- 对于启用 ReAct 多步执行的同步请求，执行中若同会话收到新的用户补充，后续迭代会自动吸收当前最新一条用户消息继续推进。

2. `Async Task`
- 适用于高复杂度、长耗时或产物型请求。
- 请求被接受后先返回任务受理结果，后续可通过任务视图或任务 API 跟踪状态、日志与产物。
- 异步任务使用独立后台执行池，不占用主会话同步交流的串行执行槽位。
- 异步任务完成后，回写到聊天区的是一轮精简后的结果摘要；完整终端输出、原始报错、代码片段与文件内容仅保留在任务详情、日志与产物中。

`Agent` 使用独立入口：

1. `POST /api/agent/messages`
- 同步返回 Agent 最终执行结果。

2. `POST /api/agent/messages/stream`
- 通过 SSE 返回 Agent 执行过程中的动作、观察与最终结果。

## Observability

1. 结构化日志（JSON）
2. `/metrics`：Prometheus 文本格式指标
3. `/healthz`：活性检查
4. `/readyz`：就绪检查
5. 关键字段：`trace_id`、`session_id`、`message_id`、`route`

## Quick Start

### Prerequisite

```bash
go version
```

建议 Go `1.25+`。

### Run Runtime

```bash
make
# or
make run
# custom port
make run WEB_ADDR=127.0.0.1:<your-port>
# or
go run ./cmd/alter0
# or
go run ./cmd/alter0 -web-addr 127.0.0.1:<your-port>
```

运行时默认行为：

1. 同时启动 Web 与 CLI 两个输入通道。
2. Web 地址默认 `127.0.0.1:18088`，可通过 `-web-addr` 参数覆盖。
3. 如果使用自定义端口，后续示例中的 URL 也需同步替换端口。
4. 默认以 `supervisor -> child runtime` 两层进程启动：父进程负责托管运行中的子进程，处理 Web 控制台发起的重启、构建、探活与切换。
5. 存储后端默认本地文件（目录 `.alter0`）。
6. 存储格式按业务场景选择：Control 配置使用 `json`，Scheduler 状态使用 `json`。

### Runtime Restart

`Environments` 页面中的“重启服务”会走运行时托管链路，而不是由当前业务进程直接自拉起：

1. `sync_remote_master=false`：基于当前仓库状态构建候选二进制，并由 `supervisor` 完成子进程切换。
2. `sync_remote_master=true`：先校验当前分支为 `master`、已跟踪工作区干净，再执行 `git fetch --prune origin master` 与 `git merge --ff-only FETCH_HEAD`，随后构建候选二进制并切换。
3. 候选版本只有在 `/readyz` 探活通过后才会成为当前运行版本；若启动失败，会自动恢复上一运行版本。
4. Git 或构建失败会直接返回到 Web 控制台，便于定位权限、凭据、快进合并失败等问题。
5. `Environments` 工具栏会展示当前在线实例的最近启动时间与对应 `commit hash`，用于确认上次成功重启切换到的运行版本。

### Public Deployment Baseline

公网部署建议使用 Nginx 反向代理，并开启应用内登录页：

```bash
export ALTER0_WEB_LOGIN_PASSWORD='请替换为强密码'

go run ./cmd/alter0 \
  -web-addr 127.0.0.1:18088 \
  -web-bind-localhost-only=true \
  -web-login-password "$ALTER0_WEB_LOGIN_PASSWORD"
```

对应 Nginx 配置与运行权限方案见：`docs/deployment/nginx.md`。

浏览器访问：

```text
http://127.0.0.1:18088/chat
```

发送消息：

```bash
curl -X POST http://127.0.0.1:18088/api/messages \
  -H "Content-Type: application/json" \
  -d '{"session_id":"s1","channel_id":"web-default","content":"/help"}'
```

### Run in CLI Mode

```bash
go run ./cmd/alter0
```

输入 `/quit` 或 `/exit` 退出。

### Terminal Shell

- 默认终端会话在 Windows 下使用 `powershell.exe`，并在启动时自动切换到 UTF-8 输出
- Linux / macOS 默认优先使用公共路径 `/usr/local/bin/codex`；若该路径不存在，则回退为 `codex`
- 如需统一指定 Codex CLI 路径，可通过环境变量 `ALTER0_CODEX_COMMAND` 或启动参数 `-codex-command` 设置
- 如需固定 shell，可通过启动参数 `-task-terminal-shell` 或运行时环境键 `task_terminal_shell` 指定
- Windows 下显式指定 `cmd.exe` 时会补充 UTF-8 代码页初始化；如需稳定中文输出，优先使用 `powershell.exe`
- Terminal 会话退出后不会清空历史或线程标识；重新在原会话发送输入时，系统会优先复用已持久化的 Codex CLI 线程继续执行
- `POST /api/terminal/sessions/{id}/close` 用于退出当前 Terminal 运行态但保留原会话历史；`DELETE /api/terminal/sessions/{id}` 用于删除 Terminal 会话，并同步清理 `.alter0/state/terminal/sessions/{id}.json` 与 `.alter0/workspaces/terminal/sessions/{id}`
- 若 Terminal 会话在首条输入前已失去底层运行态，首次发送会自动恢复同一会话并继续执行，不要求用户新建会话
- 若浏览器端 Terminal client 标识发生漂移，但同一 Web 登录态下仍携带原 `terminal_session_id` 与 CLI 线程标识，恢复流程会自动重新绑定原会话，不要求用户手动迁移历史

## Control API

### Channel

```bash
# 列表
curl http://127.0.0.1:18088/api/control/channels

# 创建/更新
curl -X PUT http://127.0.0.1:18088/api/control/channels/web-default \
  -H "Content-Type: application/json" \
  -d '{"type":"web","enabled":true}'
```

### Skill

```bash
# 列表
curl http://127.0.0.1:18088/api/control/skills

# 创建/更新
curl -X PUT http://127.0.0.1:18088/api/control/skills/summary \
  -H "Content-Type: application/json" \
  -d '{"name":"summary","enabled":true}'
```

说明：

1. 服务启动后默认提供 `default-nl` 与 `memory` 两个内置 Skill。
2. `memory` Skill 用于向 Agent / Codex 明确记忆文件的读取决策、写入路由、冲突优先级与禁止写入项，建议与 `memory_files` 一起启用。

### Agent

```bash
# 列出运行时可用入口 Agent（内置 + 用户管理）
curl http://127.0.0.1:18088/api/agents

# 列表
curl http://127.0.0.1:18088/api/control/agents

# 创建
curl -X POST http://127.0.0.1:18088/api/control/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name":"Researcher",
    "enabled":true,
    "system_prompt":"先执行，再汇报；不要只给建议。",
    "max_iterations":6,
    "tools":["list_dir","read","write","edit","bash","codex_exec"],
    "skills":["memory","summary"],
    "mcps":["github"],
    "memory_files":["user_md","soul_md","agents_md","memory_long_term","memory_daily_today","memory_daily_yesterday"]
  }'

# 更新
curl -X PUT http://127.0.0.1:18088/api/control/agents/researcher \
  -H "Content-Type: application/json" \
  -d '{
    "name":"Researcher",
    "enabled":true,
    "system_prompt":"先执行，再汇报；不要只给建议。",
    "max_iterations":8,
    "tools":["list_dir","read","write","edit","bash","codex_exec"],
    "skills":["memory","summary"],
    "mcps":["github"],
    "memory_files":["user_md","soul_md","agents_md","memory_long_term","memory_daily_today"]
  }'

# 使用指定 Agent 执行任务
curl -X POST http://127.0.0.1:18088/api/agent/messages \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id":"researcher",
    "session_id":"agent-session-1",
    "channel_id":"web-default",
    "content":"检查当前仓库并直接完成需要的修改"
  }'
```

说明：

1. Agent Profile 由控制面统一管理，运行时通过 `agent_id` 选择。
2. 系统默认注册内置 `main`、`coding`、`writing` 三个入口 Agent；它们通过统一 `Agent Catalog` 暴露给运行时与前端。
3. 创建 Agent 时不需要手填 `id` 或 `version`；服务端会自动生成 Agent ID，并在每次更新时维护版本。若生成 ID 与内置 Agent 冲突，会自动跳过保留 ID。
4. 当前内置原生工具为 `list_dir`、`read`、`write`、`edit`、`bash`；Agent 额外支持 `codex_exec`，系统会自动补充收口工具 `complete`。支持委派的 Agent 还可启用 `delegate_agent`。
5. `Chat` 默认绑定 `main` Agent；`Agent` 页面作为其余入口 Agent 的统一运行页，并按目标 Agent 隔离维护独立会话历史；具备独立前端入口的 Agent 不进入该页历史。
6. Agent 的 Skill、MCP 与 Memory Files 选择会在执行前注入运行时上下文，执行过程仍复用统一编排链路。
7. 内置 `memory` Skill 会明确记忆文件的读写逻辑：按任务类型决定先读哪些文件、按信息类型决定写入哪个文件、遇到冲突时按 `SOUL.md > AGENTS.md > 长期记忆 > 日记忆` 收敛；实际文件快照仍由 `memory_files` 注入提供。
8. `memory_files` 当前支持：`user_md`、`soul_md`、`agents_md`、`memory_long_term`、`memory_daily_today`、`memory_daily_yesterday`。
9. Memory Files 注入会携带文件内容、绝对路径、是否存在、最近更新时间；文件不存在时仍会暴露预期路径，便于 Agent 直接创建并写入。
10. Web `Agent Profiles` 页面用于管理用户自定义 Agent Profile；内置 Agent 由服务托管；`Chat` 页面绑定 `Alter0`；`Agent` 页面作为其余入口 Agent 的通用交互入口。

### Product

```bash
# 公共可见 Product 列表
curl http://127.0.0.1:18088/api/products

# 查看单个公共 Product
curl http://127.0.0.1:18088/api/products/travel

# 通过 Product 总 Agent 执行任务
curl -X POST http://127.0.0.1:18088/api/products/travel/messages \
  -H "Content-Type: application/json" \
  -d '{
    "session_id":"travel-session-1",
    "channel_id":"web-default",
    "content":"生成一份上海三日游攻略，侧重地铁通勤和本地美食"
  }'

# 直接生成 travel 攻略
curl -X POST http://127.0.0.1:18088/api/products/travel/guides \
  -H "Content-Type: application/json" \
  -d '{
    "city":"Shanghai",
    "days":3,
    "travel_style":"metro-first",
    "budget":"mid-range",
    "must_visit":["The Bund","Yu Garden"],
    "additional_requirements":["more local food","slow mornings"]
  }'

# 修订已有攻略
curl -X POST http://127.0.0.1:18088/api/products/travel/guides/shanghai-guide/revise \
  -H "Content-Type: application/json" \
  -d '{
    "days":4,
    "keep_conditions":["keep The Bund"],
    "replace_conditions":["less museum time"],
    "additional_requirements":["more neighborhood walks"]
  }'

# 控制面 Product 列表（内置 + 用户管理）
curl http://127.0.0.1:18088/api/control/products

# 创建 Product
curl -X POST http://127.0.0.1:18088/api/control/products \
  -H "Content-Type: application/json" \
  -d '{
    "name":"Travel Premium",
    "slug":"travel-premium",
    "summary":"面向高密度城市旅游攻略与地图标注的产品。",
    "status":"active",
    "visibility":"public",
    "master_agent_id":"travel-premium-master",
    "entry_route":"products",
    "tags":["travel","city-guide"],
    "artifact_types":["city_guide","itinerary","map_layers"],
    "knowledge_sources":["poi_catalog","metro_network","food_catalog"],
    "worker_agents":[
      {"agent_id":"travel-premium-city-guide","role":"city-guide","responsibility":"聚合城市景点与攻略骨架","enabled":true},
      {"agent_id":"travel-premium-route-planner","role":"route-planner","responsibility":"生成按天路线与顺序","enabled":true}
    ]
  }'

# 更新 Product
curl -X PUT http://127.0.0.1:18088/api/control/products/travel-premium \
  -H "Content-Type: application/json" \
  -d '{
    "name":"Travel Premium",
    "slug":"travel-premium",
    "summary":"补充地铁、美食和地图图层输出。",
    "status":"active",
    "visibility":"public",
    "master_agent_id":"travel-premium-master",
    "entry_route":"products",
    "worker_agents":[
      {"agent_id":"travel-premium-city-guide","role":"city-guide","responsibility":"聚合城市景点与攻略骨架","enabled":true},
      {"agent_id":"travel-premium-route-planner","role":"route-planner","responsibility":"生成按天路线与顺序","enabled":true},
      {"agent_id":"travel-premium-map-annotator","role":"map-annotator","responsibility":"输出点位与路线高亮图层","enabled":true}
    ]
  }'
```

说明：

1. `Products` 页面与 `/api/control/products` 返回内置 Product 与用户管理 Product 的统一视图。
2. `GET /api/products` 与 `GET /api/products/{product_id}` 仅暴露 `active + public` 的 Product。
3. 内置 Product 由服务注册，不能通过控制面覆盖或删除。
4. Product 更新时由服务端自动递增 `version`；新建 Product 默认从 `v1.0.0` 开始。

### Cron Jobs

```bash
# 列表
curl http://127.0.0.1:18088/api/control/cron/jobs

# 创建/更新（可视化字段 + cron_expression）
curl -X PUT http://127.0.0.1:18088/api/control/cron/jobs/job1 \
  -H "Content-Type: application/json" \
  -d '{
    "name":"daily-summary",
    "enabled":true,
    "timezone":"Asia/Shanghai",
    "schedule_mode":"daily",
    "cron_expression":"30 9 * * *",
    "task_config":{
      "input":"summarize yesterday tasks",
      "retry_limit":1
    }
  }'

# 查看指定 cron job 的触发记录与会话回链
curl http://127.0.0.1:18088/api/control/cron/jobs/job1/runs

# 按 cron 来源筛选会话历史
curl "http://127.0.0.1:18088/api/sessions?trigger_type=cron&job_id=job1"
```

## Testing

```bash
go test ./...
```

## Roadmap

1. Skill 配置与执行链路打通（按 skill 选择执行器/参数）。
2. Control 存储（SQLite/PostgreSQL）与热更新。
3. Channel 扩展（IM/HTTP 回调）与统一回投能力。
4. 任务调度增强（Cron 表达式、重试、幂等、死信）。
5. 鉴权与多租户。

## Contributing

欢迎提 Issue / PR。建议在提交前执行：

```bash
go test ./...
```

## License

MIT, see [LICENSE](./LICENSE).

