# Runtime & Orchestration Requirements

> Last update: 2026-06-14

## 领域边界

Runtime & Orchestration 负责把所有触发源归一成稳定执行链路，并为上层 Chat、Chat、Task、Chat 提供统一消息、路由、CLI Runtime 选择、调度与观测底座。

## 核心对象

| 对象 | 职责 |
| --- | --- |
| `UnifiedMessage` | 承载消息身份、通道、触发类型、追踪字段与业务载荷 |
| `OrchestrationResult` | 承载路由、输出、错误编码与执行摘要 |
| `Intent` | 表达命令或自然语言意图 |
| `Command` / `CommandHandler` | 承载内置命令与命令处理逻辑 |
| `ExecutionPort` | 隔离 Agent 执行契约，并把请求交给 Runtime Resolver |
| `RuntimeResolver` | 根据 Provider、显式执行引擎、健康状态和入口上下文选择 CLI Runtime |
| `CLIRuntime` | 表示 Claude Code 或 Codex Direct 的一次执行 |
| `RuntimeTraceEvent` | 表示 provider、adapter 或 alter0 确定产生的运行过程事件 |
| `Channel` | 表示 CLI、Web、Scheduler 等输入通道 |
| `SchedulerJob` | 表示可配置定时任务及其触发计划 |
| `TraceContext` | 贯穿 trace、session、message、correlation 维度 |

## 输入通道

### CLI

- 启动后默认开启 CLI 输入能力。
- CLI 输入必须转换为 `UnifiedMessage`，并进入同一编排层。
- `/quit` 与 `/exit` 退出 CLI 交互，不影响 Web 服务需求口径。

### Web

- Web 消息入口必须转换为 `UnifiedMessage`，并携带 Web channel 信息。
- Web Chat JSON 消息与后台 Task 触发复用同一编排语义。
- Web 消息入口除文本内容外，还需接收图片附件数组，并把附件以稳定元数据键写入 `UnifiedMessage.Metadata`，供执行链路继续解析。
- Web 登录、会话历史、移动端体验等入口侧规则由 Conversation & Session Experience 领域维护。

### Scheduler

- Scheduler 到点触发后生成 `trigger_type=cron`、`channel_type=scheduler` 的 `UnifiedMessage`。
- Cron 触发必须复用编排层，不允许绕过 Orchestration 直接调用执行器。
- Scheduler 支持服务内置 Job。内置 Job 随服务启动注册、出现在 Cron Job 列表中、不能删除，但可以通过 `enabled=false` 取消后续自动触发；重新启用后继续按原内置计划运行。
- Cron 触发记录需要保留 `job_id`、`job_name`、`fired_at` 与会话回链。

## 编排路由

### 意图识别

- 编排层先判断输入是命令还是自然语言。
- 命令请求必须优先于复杂度评估执行，避免 `/help` 等命令进入模型或任务分流。
- 当 `UnifiedMessage.Metadata` 显式声明 `alter0.execution.engine=codex` 时，斜线前缀输入不进入 alter0 `CommandRegistry`；该内容作为直连 Codex 会话输入原样交给 `ExecutionPort`，用于支持 Codex CLI 内置斜线命令。
- Web 直连 Codex 会话的斜线命令候选属于前端输入辅助，不改变编排层路由语义；候选补全后的文本仍按原始用户输入进入统一消息链路。该辅助覆盖 Chat 的直连 Codex 模型选择，以及 Chat 中 shell 明确为 Codex 的活动会话；候选集合按 Web 适用的 Codex CLI 斜线命令维护，并按命令作用分组顺序与短动作说明展示。权限、TUI 显示、键位、剪贴板、登录退出和本地 CLI 会话管理类命令不进入候选。
- Agent 请求进入 `ExecutionPort`，再由 `RuntimeResolver` 选择默认 `Codex Direct` 或显式 `Claude Code + provider profile`。

### 命令执行

- 稳定内置命令包括 `/help`、`/echo`、`/time` 与 `/now`。
- 命令输出必须进入统一 `OrchestrationResult`。
- 新增命令应在 Orchestration 领域注册，并补充命令行为测试。

### Agent 执行

- 执行端口只表达执行契约，不直接绑定具体模型。
- Runtime Resolver 按优先级执行：显式 `alter0.execution.engine=codex`、空执行器或 `auto` 均进入 `Codex Direct`；显式 `alter0.execution.engine=claude` 或携带 `alter0.llm.provider_id` 时解析对应 Provider 并进入 `Claude Code + provider profile`；Provider 不可用时回到 `Codex Direct`。Claude Code 执行失败时返回原始错误，不自动切换到 Codex。
- Claude Code 运行时启动前需要注入 `CLAUDE.md`、provider profile、Skill、Memory、MCP 和工作区事实。
- Codex Direct 运行时启动前需要注入 `AGENTS.md`、独立 `CODEX_HOME/config.toml`、Skill、Memory、MCP 和工作区事实。
- 运行时执行结果统一回写 `OrchestrationResult`、Session history、Task 或 Cron run。
- 执行错误需保留可诊断错误编码，供 Web、Task 与 Skill 收口使用。
- 运行时过程事件统一归一为 `RuntimeTraceEvent`。事件必须保留 `source`、provider 引用、角色、类型、生命周期、状态、结构化 blocks 与可选 action；`source=provider` 表示底层 SDK/CLI 明确提供，`source=adapter` 表示工程 adapter 从稳定协议字段转换，`source=alter0` 表示 alter0 本地确定性生成。系统不得根据自然语言正文、标题、关键词或语言模式推断事件类型。

## 调度能力

### Cron Job 配置

- Control 面提供普通 Cron Job 创建、更新、删除与列表查询；内置 Cron Job 只允许切换启停状态。
- Cron Job 支持可视化周期字段、timezone、cron expression 与任务配置。
- 可视化字段与 cron expression 必须保持一致；无法表达时需返回明确错误。
- Cron Job 接口通过 `GET /api/control/cron/jobs`、`PUT /api/control/cron/jobs/{job_id}`、`DELETE /api/control/cron/jobs/{job_id}` 管理配置；内置 Job 的 `DELETE` 必须返回冲突，`PUT` 仅接受 `enabled` 切换。

### 触发归档

- 每次 Cron 触发必须创建或关联可回查会话。
- 会话列表支持按 Cron 来源筛选。
- Cron Job 详情支持查看触发记录与会话回链。
- `GET /api/control/cron/jobs/{job_id}/runs` 按 `job_id`、分页参数和 `trigger_type=cron` 读取会话摘要，返回 `session_id`、触发时间与会话回链信息。

## 存储

### 本地存储

- 默认存储后端为 `ALTER0_RUNTIME_ROOT` 派生的本地文件；业务 JSON 存储位于 `<runtime_root>/storage/`，执行工作区、Chat state、日志和运行输出从同一 runtime root 派生。
- `ALTER0_STORAGE_DIR` 仅作为旧部署兼容入口保留；当它与 `ALTER0_RUNTIME_ROOT` 同时存在时，必须等于 `<runtime_root>/storage`。
- Control 配置与 Scheduler 状态优先使用 JSON。
- Memory 主存使用 Markdown，派生索引可按 Memory 领域策略重建。

### 可替换性

- 存储实现位于 infrastructure 层，不应反向污染 domain 与 application。
- 后续替换 SQLite、PostgreSQL 或配置中心时，不应改动核心业务服务代码契约。

## 观测与健康

### 日志与指标

- 结构化日志需包含 `trace_id`、`session_id`、`message_id`、`route` 等关键字段。
- `/metrics` 输出 Prometheus 文本格式指标。
- 关键指标覆盖输入消息数、路由请求数、命令请求数、错误数与路由耗时。

### 健康检查

- `/healthz` 表示进程活性。
- `/readyz` 表示当前实例已可对外承接请求。
- `/metrics` 提供 Prometheus 格式指标。
- Runtime 重启与候选二进制切换必须以 readyz 通过作为稳定切换条件。

## 依赖与边界

- Conversation 领域消费 Runtime 的消息结果与结构化过程，不拥有编排路由规则。
- Skill & Memory 领域提供 Skill 与 Memory 注入上下文，不直接改写通道模型。
- Task 领域可承接复杂度分流后的后台执行，但触发源仍来自统一消息。
- 业务入口通过统一 CLI Runtime 接入执行链，不绕过 Orchestration。

## 验收口径

- CLI、Web、Cron 三类输入都能进入统一编排链路。
- 命令请求不进入模型执行。
- Agent 请求默认进入 Codex Direct；显式 Provider/Claude 请求进入 Claude Code。
- Cron 触发可在会话与任务视图中追踪来源。
- 结构化日志、metrics、healthz、readyz 均可用。
- 存储实现替换不要求改动 domain 对象。
