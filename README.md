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
- 负责配置 `Channel / Skill / CronJob`。
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
- 仅面向 `Raw Model` 对话。
- Web 登录后，`Chat` 与 `Agent` 页面共享同一套 Session 历史，不再按页面来源拆分会话列表与本地缓存。
- 运行时配置收敛在输入框底部操作栏：`Provider / Model`、`Tools / MCP`、`Skills` 都在发送区附近完成。
- `Provider / Model`、`Tools / MCP`、`Skills` 可在会话过程中继续调整，并作用于后续发送的消息。
- 选中的原生工具会在模型调用时作为 function tools 注入，当前内置工具包括 `list_dir`、`read`、`write`、`edit`、`bash`。
- `OpenAI` Provider 支持按 `api_type` 选择上游接口：`openai-responses` 走 `/responses`，`openai-completions` 走 `/chat/completions`；配置自定义 `base_url` 时，需要目标服务兼容所选接口。
- 复杂度评估阶段会优先复用当前消息选中的 `Provider / Model`；未显式选择时，回退到默认 Provider 与默认模型。
- 默认走实时执行。
- 流式对话会先直接启动回复；复杂度评估与回复并行进行。
- 当请求复杂度较高且仍在执行中时，系统会中途转为后台 `Task` 执行，并先返回一条任务说明消息，包含任务目标、执行计划与任务入口。
- 聊天气泡支持常用 Markdown 渲染，包括标题、列表、引用、链接、行内代码与代码块；原始 HTML 不直接透传。
- Chat 消息会标注实际回复来源，用于区分当前内容来自模型执行链还是 `Codex CLI` 执行链。

2. `Agent`
- 面向“先执行再汇报”的目标型任务。
- 请求进入后会创建一个 ReAct 执行环，以当前任务为目标持续推进。
- Agent 默认可使用 `list_dir`、`read`、`write`、`edit`、`bash`、`codex_exec` 等工具，并依据执行结果决定继续推进还是完成收口。
- 每个 Agent 可独立配置名称、system prompt、tool 白名单、Skill 选择与 MCP 选择。
- Web 端将 Agent 的“配置/管理”放在 `Agent Profiles` 页面，将 Agent 的“交互/执行”统一收敛到 `Agent` 页面。
- Agent 的 `id`、`version` 等系统字段由服务端统一生成和维护，管理页不要求用户手填。
- `Agent` 页面继续保留独立执行入口与配置，但其 Session 历史与 `Chat` 页面统一展示、统一恢复。

3. `Terminal`
- 面向交互式终端会话。
- 仍属于自然语言处理，但使用独立上下文边界。
- 默认仅注入运行时必需上下文，不复用 Chat 会话记忆与长期记忆。
- 每个 Terminal 会话使用独立工作区目录，不再默认落在仓库根目录。
- Terminal 会持久化 Codex CLI 线程标识与会话状态；运行态退出后保留原会话历史，继续发送即可在同一会话内恢复。
- Terminal 不再设置产品级会话数量上限或固定超时淘汰策略。
- 移动端访问 Terminal 时，轮询刷新不会重建已聚焦输入框；输入法每次确认词句后，若输入框仍保持聚焦，页面继续延迟重绘并保持当前位置，直到失焦后再刷新视图。
- 键盘弹起后输入区继续贴底可输入，页面不会因刷新回到顶部。

补充说明：

1. `Chat / Agent / Terminal` 只要落到 `Codex CLI` 执行链，都要求服务运行账户本身具备可用的 Codex / OpenAI 认证。
2. 若服务账户缺少认证，Web 端会快速返回认证失败，而不会长时间保持等待态。

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
- 若 Terminal 会话在首条输入前已失去底层运行态，首次发送会自动恢复同一会话并继续执行，不要求用户新建会话

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

### Agent

```bash
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
    "skills":["summary"],
    "mcps":["github"]
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
    "skills":["summary"],
    "mcps":["github"]
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
2. 创建 Agent 时不需要手填 `id` 或 `version`；服务端会自动生成 Agent ID，并在每次更新时维护版本。
3. 当前内置原生工具为 `list_dir`、`read`、`write`、`edit`、`bash`；Agent 额外支持 `codex_exec`，系统会自动补充收口工具 `complete`。
4. `Chat` 会按当前运行时勾选结果注入原生工具；`Agent` 在未显式限制工具集时默认注入核心工具。
5. Agent 的 Skill 与 MCP 选择会在执行前注入运行时上下文，执行过程仍复用统一编排链路。
6. Web `Agent Profiles` 页面用于管理 Agent Profile；`Agent` 页面作为 Agent 交互入口；`Chat` 页面仅保留 Raw Model 对话。

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
