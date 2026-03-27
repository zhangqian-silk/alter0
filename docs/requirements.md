# Requirements

> Last update: 2026-03-27

状态说明：

1. `supported`：已在主干代码可用
2. `planned`：已确认方向，待排期

## 需求列表

| ID | 需求 | 状态 | 说明 |
| --- | --- | --- | --- |
| R-001 | CLI 消息接入 | supported | 启动后自动开启 CLI 交互输入，统一进入编排层 |
| R-002 | Web 消息接入 | supported | `POST /api/messages` |
| R-003 | 统一消息模型 | supported | `UnifiedMessage` 统一 channel/trigger/trace |
| R-004 | 命令路由执行 | supported | `/help` `/echo` `/time` |
| R-005 | 自然语言路由执行 | supported | `ExecutionPort` 执行 |
| R-006 | 基础可观测性 | supported | metrics + health + structured logs |
| R-007 | Channel 配置管理 | supported | `GET/PUT/DELETE /api/control/channels/*` |
| R-008 | Skill 配置管理 | supported | `GET/PUT/DELETE /api/control/skills/*` |
| R-009 | 定时任务管理 | supported | `GET/PUT/DELETE /api/control/cron/jobs/*` |
| R-010 | 定时任务触发执行 | supported | Scheduler 触发消息并复用编排链路 |
| R-011 | 本地文件存储 | supported | Control/Scheduler 支持 `json`/`markdown` 本地存储 |
| R-012 | Web 侧边栏交互优化 | supported | 点击侧边栏项后进入“信息展示模式”：主区域仅展示对应信息，不再弹出或保留对话框；侧边栏在展开态提供显式折叠按钮 |
| R-013 | 流式传输能力 | supported | 提供端到端流式响应：后端增量推送生成内容，前端实时渲染并可感知进行中/完成/失败状态 |
| R-014 | 移动端真机适配增强 | supported | 优化小屏与键盘场景（安全区、输入区跟随、遮罩关闭与手势交互），并确保 Chat/Terminal 在 iOS/Android 键盘输入与输入法候选确认时不回顶、不抖动 |
| R-015 | 移动端会话创建与信息完整性 | supported | 确保移动端可稳定新建会话，并完整展示会话必要信息（标题、入口状态、空态提示） |
| R-016 | 会话级并发控制与全局限流 | supported | 支持多会话并发处理，同时保证同一会话顺序一致，并提供系统级并发上限、排队与超时降级能力 |
| R-017 | 会话短期记忆 | supported | 在单会话内维护可控窗口的上下文记忆，提升多轮对话连续性与指代解析能力 |
| R-018 | 跨会话长期记忆 | supported | 支持跨会话沉淀用户偏好与长期事实，并按用户/租户范围检索后按相关性注入上下文 |
| R-019 | 会话内容持久化 | supported | 持久化用户/助手消息主数据与路由结果，支持重启恢复、会话分页与按时间范围检索 |
| R-020 | 上下文压缩 | supported | 超长上下文触发分层压缩，结构化回写摘要与关键事实并保留消息引用关系，降低长会话 token 成本 |
| R-021 | Skills/MCP 标准化能力模型 | supported | 统一 Skills 与 MCP 的配置结构、启停状态、作用域与校验规则，形成可治理的标准化能力层 |
| R-022 | 用户配置 Skills 接入 Codex | supported | 将启用的用户 Skills 以标准协议注入 Codex / Agent 执行上下文，支持 description、guide、排序与冲突处理 |
| R-023 | 用户配置 MCP 接入 Codex | supported | 将用户配置的 MCP Server 安全映射到 Codex 运行配置，支持按会话/请求启用与审计追踪 |
| R-024 | 跨会话持久化记忆分级管理（参考 L1/L2/L3 Cache） | supported | 参考计算机缓存分层实现记忆分级：L1 高优先/低容量，L2 平衡层，L3 大容量归档层；按命中率与重要性动态迁移并分级限额 |
| R-025 | 天级记忆与长期记忆（Markdown 统一存储） | supported | 支持天级记忆落盘与长期记忆沉淀，并对每日记忆做压缩归档；R-017~R-024 的记忆数据统一以 Markdown 格式存储 |
| R-026 | 强制要求上下文文件（如 SOUL.md） | supported | 支持独立上下文文件存储用户强制要求，启动与会话初始化高优先级加载，并在冲突场景下覆盖普通记忆 |
| R-027 | Agent Memory 模块与页面收敛 | supported | 前端移除 `Workspace` 与 `Configuration` 页面；在 `Agent` 下新增 `Memory` 模块，可视化长期记忆、天级记忆与持久化记忆（`SOUL.md`） |
| R-028 | Memory 模块说明文档持久化与可视化 | supported | 新增记忆体系说明文档并持久化纳入仓库；前端 `Agent -> Memory` 提供文档视图入口，支持稳定查看 `USER.md`、`AGENTS.md`、`MEMORY.md`、`memory/YYYY-MM-DD.md`、`SOUL.md` 的职责说明与映射关系 |
| R-029 | 新对话空白会话唯一性约束 | supported | 前端与会话创建链路不允许生成多个“空白会话”；当已存在空白会话时，`New Chat` 必须复用并聚焦该会话，而不是继续新建 |
| R-030 | 会话与异步任务映射模型 | supported | 建立 `session_id` 与 `task_id` 的标准映射，支持长耗时请求异步化执行（快速应答 + 后台任务 + 任务日志回读），避免对话链路阻塞与上下文膨胀 |
| R-031 | 任务摘要跨会话记忆与按需深检索 | supported | 默认仅注入最近 3-5 条任务摘要控制上下文体积；当用户询问更早历史时自动切换深检索，从全量任务摘要库召回并按需下钻任务详情 |
| R-032 | `.alter0` 任务历史存储规范与 Memory 查阅 | supported | 统一任务运行态数据在 `.alter0` 下的目录结构、留存策略与回链规则；前端 `Agent -> Memory` 新增任务历史查阅能力（摘要默认可见、日志按需下钻） |
| R-033 | Control 任务观测台基础视图 | supported | 在 `Control` 页面新增 `Tasks` 入口，提供任务列表、详情抽屉与基础筛选能力，形成统一任务观测入口 |
| R-034 | 任务触发来源标识与溯源视图 | supported | 为任务补齐并展示 `trigger_type/channel_type/channel_id/correlation_id` 等来源字段；定时任务补充 `job_id/job_name/fired_at` |
| R-035 | 任务日志流式观测与断线续读 | supported | 提供任务日志 SSE 流式观测能力，支持游标断点续读与回补查询，满足长任务实时观测 |
| R-036 | 任务控制动作与会话回链 | supported | 提供 `retry/cancel` 控制面板与任务-会话双向跳转，保证问题定位与重试闭环 |
| R-037 | Web 产物可访问交付层（替代本地路径暴露） | supported | Web 会话不再向用户返回本地文件路径；通过任务产物引用与下载/预览接口完成交付，保障可访问性与路径安全 |
| R-038 | 对话历史面板折叠能力 | supported | 在 Chat 页面支持“最近会话”历史区折叠/展开，减少侧栏占用并提升长对话阅读空间 |
| R-039 | 页面滚动隔离（侧栏固定） | supported | 全站页面默认固定侧边栏并禁止其随主页面滚动；仅在侧栏内容受高度限制发生溢出时允许侧栏自身内部滚动 |
| R-040 | Cron 可视化配置与触发会话归档 | supported | 支持以可视化方式配置 Cron 与任务（周期/固定时间点），实时展示 cron 表达式；Cron 触发后创建专属会话并可在会话历史中查看 |
| R-041 | 任务执行前复杂度预判与同步/异步分流 | supported | 用户消息进入执行器前先做复杂度预估；若创建异步任务则先同步返回可跳转任务详情的任务卡片（含任务 ID 与简要描述），任务完成后向用户主动同步结果 |
| R-042 | 异步任务执行模块与前端全量执行明细 | supported | 建立专门异步执行模块并限制最多 5 个任务并发执行；前端任务模块按当前界面风格展示任务执行明细，并完整回传终端运行细节（如“已运行 xxx”） |
| R-043 | Environments 关键配置统一管理 | supported | 完善 Environments 模块，支持对任务并发等关键运行参数进行可视化配置、校验与持久化，并将当前可配置项统一纳入同一管理入口 |
| R-044 | Channels 迁移至 Settings 模块 | supported | 前端导航将 `Channels` 从 `Control` 分组迁移到 `Settings` 分组，保持页面能力与接口不变并兼容原有直达路由 |
| R-045 | Session/Task 关键信息披露增强 | supported | 在 Session、Task 等页面补充展示必要上下文字段（如 `channel`、`message_id`、`channel_id`、`trigger_type`），提升问题定位与执行链路可追溯性 |
| R-046 | Terminal 模块与会话式终端代理 | supported | 新增独立 `Terminal` 模块，支持在模块内以类 Chat 方式持续对话；同一终端会话内复用上下文并连续串联输入/输出，完整保留终端交互轨迹，并在移动端提供会话面板收纳、元信息折叠、长输出折叠与稳定输入体验 |
| R-047 | OpenAI Go SDK 接入 | planned | 接入 `github.com/openai/openai-go` SDK，支持 OpenAI 兼容 API（含自定义 base_url），提供统一 LLM 调用层 |
| R-048 | ReAct 模式 Agent 调用 | planned | 实现 Reasoning + Acting 循环：Thought → Action → Observation → Thought，支持工具调用与多轮推理 |
| R-049 | 模型配置管理 | planned | 支持配置多个 LLM Provider（base_url、api_key、model_name），提供 Provider 级别的启用/禁用与默认切换 |
| R-050 | Web 登录后统一 Session 视图 | supported | 密码验证通过后，Web 内 `Chat`、`Agent` 等页面继续保留各自定位与入口，但共享同一套 Session 历史；不再按页面来源拆分 Session，并移除相关冗余文案与状态内容 |
| R-051 | Terminal 会话持久标识与超时恢复 | supported | 持久化存储 Codex CLI 会话标识；不再对 Terminal 设置产品级会话上限与超时淘汰；运行态缺失后继续发送会自动恢复原会话并保留历史 |
| R-052 | Agent Memory Files 勾选注入与文件可写记忆对齐 | supported | Agent Profile 支持勾选 `USER.md`、`SOUL.md`、`AGENTS.md`、长期 `MEMORY.md` 与 Daily Memory；执行前将所选文件内容与路径注入运行时上下文，并可搭配独立 `memory` Skill 统一记忆读写规范 |

## 需求细化（分文件）

1. `R-001` ~ `R-010`：[requirements-detail-r001-r010.md](requirements-details/requirements-detail-r001-r010.md)
2. `R-011` ~ `R-020`：[requirements-detail-r011-r020.md](requirements-details/requirements-detail-r011-r020.md)
3. `R-021` ~ `R-030`：[requirements-detail-r021-r030.md](requirements-details/requirements-detail-r021-r030.md)
4. `R-031` ~ `R-040`：[requirements-detail-r031-r040.md](requirements-details/requirements-detail-r031-r040.md)
5. `R-041` ~ `R-050`：[requirements-detail-r041-r050.md](requirements-details/requirements-detail-r041-r050.md)
6. `R-051` ~ `R-060`：[requirements-detail-r051-r060.md](requirements-details/requirements-detail-r051-r060.md)

说明：需求细化内容按每 10 个需求拆分维护。
