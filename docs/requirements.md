# Requirements

> Last update: 2026-04-07

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
| R-013 | 流式传输能力 | supported | 提供端到端流式响应：后端增量推送生成内容，前端实时渲染并可感知进行中/完成/失败状态；长时间无增量时 SSE 通道持续发送保活帧，避免浏览器或代理提前断流；`Chat / Agent` 消息区在高频增量期间需采用逐条 patch 与逐帧合并刷新，避免长会话反复整段重建；已进入 Agent 执行链的请求在前端断连后继续完成后端执行 |
| R-014 | 移动端真机适配增强 | supported | 优化小屏与键盘场景（安全区、输入区跟随、遮罩关闭、Chat 输入区信息收纳与手势交互），并确保 Chat/Terminal 在 iOS/Android 键盘输入与输入法候选确认时不回顶、不抖动；Chat/Agent 输入区在 `VisualViewport` 高度变化时持续贴底可见，且仅在输入框实际聚焦并达到键盘阈值时追加底部键盘偏移，避免浏览器工具栏伸缩后残留空白；移动端 `page-mode` 路由页也需消费 `VisualViewport` 高度，保证 Terminal 与其他信息页在工具栏回弹、键盘收起后不残留底部空白；Chat 发送按钮需与会话设置入口同排，运行时文案保持单语言一致，配置弹层在窄宽度下采用独立底部面板并保持内容不与发送区层叠，Agent 选项改为短摘要展示，连续勾选 Skill/Tool/MCP 时滚动位置不回顶；移动端后台轮询与视口同步需按页面可见性和输入状态降频 |
| R-015 | 移动端会话创建与信息完整性 | supported | 确保移动端可稳定新建会话，并完整展示会话必要信息（标题、入口状态、空态提示） |
| R-016 | 会话级并发控制与全局限流 | supported | 支持多会话并发处理，同时保证同一会话顺序一致，并提供系统级并发上限、排队与超时降级能力 |
| R-017 | 会话短期记忆 | supported | 在单会话内维护可控窗口的上下文记忆，提升多轮对话连续性与指代解析能力 |
| R-018 | 跨会话长期记忆 | supported | 支持跨会话沉淀用户偏好与长期事实，并按用户/租户范围检索后按相关性注入上下文 |
| R-019 | 会话内容持久化 | supported | 持久化用户/助手消息主数据与路由结果，支持重启恢复、会话分页、按时间范围检索，以及按 `session_id` 删除历史并清理对应会话工作区 |
| R-020 | 上下文压缩 | supported | 超长上下文触发分层压缩，结构化回写摘要与关键事实并保留消息引用关系，降低长会话 token 成本 |
| R-021 | Skills/MCP 标准化能力模型 | supported | 统一 Skills 与 MCP 的配置结构、启停状态、作用域与校验规则，形成可治理的标准化能力层 |
| R-022 | 用户配置 Skills 接入 Codex | supported | 将启用的用户 Skills 以标准协议注入 Codex / Agent 执行上下文，支持 description、guide、排序、文件路径、可写属性与冲突处理；所有 Agent 还会自动附带私有 file-backed Skill `.alter0/agents/<agent_id>/SKILL.md` 用于沉淀该 Agent 的可复用规则；内置 `memory` 负责统一记忆路由，travel 页面规则沉淀到 `travel-master` 私有 Skill |
| R-023 | 用户配置 MCP 接入 Codex | supported | 将用户配置的 MCP Server 安全映射到 Codex 运行配置，支持按会话/请求启用与审计追踪 |
| R-024 | 跨会话持久化记忆分级管理（参考 L1/L2/L3 Cache） | supported | 参考计算机缓存分层实现记忆分级：L1 高优先/低容量，L2 平衡层，L3 大容量归档层；按命中率与重要性动态迁移并分级限额 |
| R-025 | 天级记忆与长期记忆（Markdown 统一存储） | supported | 支持天级记忆落盘与长期记忆沉淀，并对每日记忆做压缩归档；R-017~R-024 的记忆数据统一以 Markdown 格式存储 |
| R-026 | 强制要求上下文文件（如 SOUL.md） | supported | 支持独立上下文文件存储用户强制要求，启动与会话初始化高优先级加载，并在冲突场景下覆盖普通记忆 |
| R-027 | Agent Memory 模块与页面收敛 | supported | 前端移除 `Workspace` 与 `Configuration` 页面；在 `Agent` 下新增 `Memory` 模块，可视化长期记忆、天级记忆与持久化记忆（`SOUL.md`） |
| R-028 | Memory 模块说明文档持久化与可视化 | supported | 新增记忆体系说明文档并持久化纳入仓库；前端 `Agent -> Memory` 提供文档视图入口，支持稳定查看 `USER.md`、`AGENTS.md`、`MEMORY.md`、`memory/YYYY-MM-DD.md`、`SOUL.md` 的职责说明与映射关系 |
| R-029 | 新对话空白会话唯一性约束 | supported | 前端与会话创建链路不允许生成多个“空白会话”；当已存在空白会话时，`New Chat` 必须复用并聚焦该会话，而不是继续新建 |
| R-030 | 会话与异步任务映射模型 | supported | 建立 `session_id` 与 `task_id` 的标准映射，支持长耗时请求异步化执行（快速应答 + 后台任务 + 任务日志回读）；删除会话时同步清理关联任务记录与任务产物，避免残留孤儿数据 |
| R-031 | 任务摘要跨会话记忆与按需深检索 | supported | 默认仅注入最近 3-5 条任务摘要控制上下文体积；当用户询问更早历史时自动切换深检索，从全量任务摘要库召回并按需下钻任务详情 |
| R-032 | `.alter0` 任务历史存储规范与 Memory 查阅 | supported | 统一任务运行态数据在 `.alter0` 下的目录结构、留存策略与回链规则；前端 `Agent -> Memory` 新增任务历史查阅能力（摘要默认可见、日志按需下钻） |
| R-033 | Control 任务观测台基础视图 | supported | 在 `Control` 页面新增 `Tasks` 入口，提供任务列表、详情抽屉与基础筛选能力，形成统一任务观测入口 |
| R-034 | 任务触发来源标识与溯源视图 | supported | 为任务补齐并展示 `trigger_type/channel_type/channel_id/correlation_id` 等来源字段；定时任务补充 `job_id/job_name/fired_at` |
| R-035 | 任务日志流式观测与断线续读 | supported | 提供任务日志 SSE 流式观测能力，支持游标断点续读与回补查询，满足长任务实时观测；前端日志抽屉在高频日志到达期间需按逐帧节奏合并刷新，避免每条日志事件都同步整块重绘 |
| R-036 | 任务控制动作与会话回链 | supported | 提供 `retry/cancel` 控制面板与任务-会话双向跳转，保证问题定位与重试闭环 |
| R-037 | Web 产物可访问交付层（替代本地路径暴露） | supported | Web 会话不再向用户返回本地文件路径；通过任务产物引用与下载/预览接口完成交付，保障可访问性与路径安全 |
| R-038 | 对话历史面板折叠能力 | supported | 在 Chat 页面支持“最近会话”历史区折叠/展开，减少侧栏占用并提升长对话阅读空间 |
| R-039 | 页面滚动隔离（侧栏固定） | supported | 全站页面默认固定侧边栏并禁止其随主页面滚动；仅在侧栏内容受高度限制发生溢出时允许侧栏自身内部滚动 |
| R-040 | Cron 可视化配置与触发会话归档 | supported | 支持以可视化方式配置 Cron 与任务（周期/固定时间点），实时展示 cron 表达式；Cron 触发后创建专属会话并可在会话历史中查看 |
| R-041 | 任务执行前复杂度预判与同步/异步分流 | supported | 用户消息进入执行器前先做复杂度预估；若创建异步任务则先同步返回可跳转任务详情的任务卡片（含任务 ID 与简要描述），任务完成后向用户主动同步结果 |
| R-042 | 异步任务执行模块与前端全量执行明细 | supported | 建立专门异步执行模块并限制最多 5 个任务并发执行；前端任务模块按当前界面风格展示任务执行明细，并完整回传终端运行细节（如“已运行 xxx”）；`Codex CLI` 长任务按 1 分钟心跳续租运行超时窗口，并在列表卡片与任务详情展示最近心跳和当前窗口截止时间 |
| R-043 | Environments 关键配置统一管理 | supported | 完善 Environments 模块，支持对任务并发等关键运行参数进行可视化配置、校验与持久化；展示当前在线实例最近启动时间与 commit hash；重启操作通过单一站内确认弹窗提供默认勾选的远端同步项，并在完成后自动刷新提示当前页面已连接最新实例；systemd 部署基线将服务 `HOME` 收敛到运行根目录 `/var/lib/alter0`，并要求运行账户具备稳定的 PATH、GitHub App 凭证、提交签名初始化能力，以及可供 `Codex CLI` 直接执行 Node/Playwright 测试链路的 `node`/`npm`/`npx` 运行时 |
| R-044 | Channels 迁移至 Settings 模块 | supported | 前端导航将 `Channels` 从 `Control` 分组迁移到 `Settings` 分组，保持页面能力与接口不变并兼容原有直达路由 |
| R-045 | Session/Task 关键信息披露增强 | supported | 在 Session、Task 等页面补充展示必要上下文字段（如 `channel`、`message_id`、`channel_id`、`trigger_type`），提升问题定位与执行链路可追溯性 |
| R-046 | Terminal 模块与会话式终端代理 | supported | 新增独立 `Terminal` 模块，支持在模块内以类 Chat 方式持续对话；同一终端会话内复用上下文并连续串联输入/输出，完整保留终端交互轨迹，并提供工作区头部与会话列表两侧可达的 `Close / Delete` 生命周期操作，其中删除会同步清理持久化状态与工作区文件；工作区头部采用标题 + 状态点 + 紧凑工具栏，Terminal 会话态统一为 `ready / busy / exited / interrupted`，执行态继续在 turn/step 维度展示 `running / completed / failed / interrupted`，运行态退出/中断提示以内嵌状态条贴合输入区且仅在当前空闲会话上展示，用户重新发送恢复或切入 `New` 待创建态时需立即清空旧提示，且待创建态不得继承旧会话残留的底部留白；Terminal 新会话默认先使用占位标题，自动标题在早期多轮内需按更具体的后续输入继续升级，尤其覆盖“拉取仓库 / 分析仓库”等通用开场；全站消息阅读体验统一收敛为浅色文档式主题，用户消息右对齐且宽度不超过消息区 80%，助手回复弱化厚重卡片层级，Terminal 中用户输入不再额外展示命令前缀符号或强调色，最终回复按 Chat 助手消息逻辑直接渲染并提供一键复制，且同一轮最终输出出现后自动折叠对应 `Process`，移动端 `Process` 头部与步骤行保持单行摘要阅读，标题超长时自动截断，时间与状态固定在右侧独立区域；Terminal 会话侧栏的状态徽标与 `Delete` 动作在长标题和双行元信息下保持统一高度与垂直居中；移动端长输出阅读取消消息与区块头部吸顶，统一保持自然文档流滚动，并提供右侧圆形四键组用于回到顶部、上一条、下一条与回到底部，以及浅色低对比阅读主题；Terminal 工作区在浏览器底部工具栏伸缩、软键盘收起或视口回弹后需立即回贴可见视口底边，不保留底部空白；Terminal 轮询与本地持久化需按页面可见性、输入与滚动活跃度自动降频，桌面活跃输入窗口内同样延后非必要工作区重绘，滚动中的导航计算与位置测量需合并到逐帧节奏并复用稳定锚点缓存 |
| R-047 | OpenAI Go SDK 接入 | supported | 已接入 `github.com/openai/openai-go` SDK，统一支持 OpenAI 兼容 API、自定义 `base_url`、`/responses`、`/chat/completions` 双接口模式，并可对 `OpenRouter` 注入官方头部与路由扩展字段 |
| R-048 | ReAct 模式 Agent 调用 | supported | Agent 执行链已提供 ReAct 循环、多轮观察、记忆文件检索读写与 `codex_exec` 驱动的具体执行能力；Agent 作为用户与 Codex 之间的代理层，在自身侧消化 system prompt、记忆与规则，只向 Codex 下发当前执行所需的最小上下文与具体指令；Web 会将 `action / observation` 执行细节收敛为可折叠 `Process`，最终答复保持正文阅读并提供一键复制；当迭代耗尽未显式 `complete` 时，系统需显式返回上限提示与最后一次工具观察，避免空收口；前端页面事件与连接生命周期不得中断已启动的 Agent 执行 |
| R-049 | 模型配置管理 | supported | 支持多 Provider/多模型配置、启用禁用、默认 Provider 与默认模型切换，并对禁用默认项自动收敛到可用配置；`Models` 控制面可直接维护 `OpenAI Compatible / OpenRouter` Provider 信息，`api_key` 占位值不会被持久化为真实凭据，历史缺失密钥的 Provider 会在加载时自动降为禁用态以保证控制面可恢复 |
| R-050 | Web 登录后按 Agent 隔离 Session 视图 | supported | 密码验证通过后，Web 对话页按目标 Agent 维护独立 Session 历史；具备独立前端入口的 Agent 不进入通用 Agent 页面历史；`Chat / Agent` 新会话默认先使用占位标题，自动标题在早期多轮内需按更具体的用户消息继续升级，尤其覆盖通用开场 |
| R-051 | Terminal 会话持久标识与超时恢复 | supported | 持久化存储 Codex CLI 会话标识；Terminal 历史在同一 Web 登录态下跨设备共享，不再按浏览器 client 标识隔离；会话态升级后继续兼容历史 `running / starting` 持久化值并自动归一到 `ready / busy`；运行态缺失后继续发送会自动恢复原会话并保留历史，工作区仍按会话独立隔离；同一状态周期内不重复追加相同的运行态中断提醒，且恢复发送开始后旧的退出/中断提示需立即清空 |
| R-052 | Agent Memory Files 勾选注入与文件可写记忆对齐 | supported | Agent Profile 支持勾选 `USER.md`、`SOUL.md`、Agent 私有 `AGENTS.md`、长期 `MEMORY.md` 与 Daily Memory；执行前将所选文件内容与路径注入运行时上下文，其中 `AGENTS.md` 固定绑定 `.alter0/agents/<agent_id>/AGENTS.md` 且不跨 Agent 共享，`USER.md`、`SOUL.md` 与长期/日记忆继续共享；运行时还会自动维护当前 Agent 在当前 Session 下的只读 `Agent Session Profile`，用于沉淀会话级稳定上下文；Agent 可在可写记忆文件上执行关键字检索、定向读取与受控写入，并可搭配独立 `memory` Skill 统一记忆读写规范 |
| R-053 | 内置 Agent Catalog 与主从委派 | supported | 运行时统一聚合内置 Agent 与用户管理 Agent；`Chat` 默认绑定 `main` Agent，通用 `Agent` 页面承载其余入口 Agent；所有具体执行统一通过 `codex_exec` 落到 Codex CLI，Agent 以持续助手方式复用当前 Session 的稳定上下文推进任务，并作为用户与 Codex 的代理层保留自身 prompt/规则编排，不把 Agent 侧 system prompt 直接透传给 Codex；稳定执行事实统一通过 `alter0.codex-exec/v1` 的结构化上下文字段按需注入 Codex，只传当前执行所需的必要信息，不重复拼接完整规则，也不透传无关上下文；`codex_exec` 通过 stdin 向 Codex CLI 传递最终指令，避免长上下文直接进入系统命令行参数；`Alter0/main` 负责跨 Agent 委派与结果收口，其中 `coding` Agent 负责面向用户收口编码任务，并通过 `codex_exec` 多轮驱动具体开发执行，仓库类操作默认落到当前 Session 独立 repo 完整 clone `.alter0/workspaces/sessions/<session_id>/repo`，并在需要测试页面时将结果部署到 `https://<session_short_hash>.alter0.cn` 后再收口 |
| R-054 | Product 目录、Workspace 与管理页 | supported | 新增 `Products` 平台模块，集中管理多个 Product 定义、主 Agent、详情页空间与可选 supporting agents，并为每个详情页空间提供独立 HTML 页面入口 |
| R-055 | Product Agent 单主 Agent 草稿生成 | supported | 提供平台级 `product-builder` 与 Draft Studio，支持生成新 Product 草稿、增量扩展、审核编辑、冲突提示与发布落地；新生成的 Product 默认沉淀为单主 Agent，并把可复用领域规则放入 system prompt 与 Skill |
| R-056 | Product 总 Agent 执行编排 | supported | 每个已发布 Product 绑定唯一总 Agent，并将其同步到托管 Agent Catalog；Product 总 Agent 统一采用 Agent 协助 / Codex 执行模型，由总 Agent 直接收口结果，历史 supporting agents 仅做兼容保留 |
| R-057 | Alter0 Agent 跨 Product 信息检索与任务调度 | supported | `Alter0 Agent` 作为统一入口，按用户意图检索 Product 目录与能力信息，并在执行型请求中自动切换到目标 Product 总 Agent；当前已提供 Product 公开执行入口、Product 上下文与路由元数据注入 |
| R-058 | Travel Product 首个产品域落地 | supported | 以 `travel` 作为首个内置 Product，提供主 Agent 对话式城市页创建/修改、独立 HTML 城市页、攻略详情与 revision 能力；`travel-master` 通过其私有 `SKILL.md` 统一页面规则并借助 `codex_exec` 按需更新；Workspace Chat 在 Agent 执行链不可用时自动回退本地解析，并保留路线、地铁、美食、地图图层等结构化字段 |

## 需求细化（分文件）

1. `R-001` ~ `R-010`：[requirements-detail-r001-r010.md](requirements-details/requirements-detail-r001-r010.md)
2. `R-011` ~ `R-020`：[requirements-detail-r011-r020.md](requirements-details/requirements-detail-r011-r020.md)
3. `R-021` ~ `R-030`：[requirements-detail-r021-r030.md](requirements-details/requirements-detail-r021-r030.md)
4. `R-031` ~ `R-040`：[requirements-detail-r031-r040.md](requirements-details/requirements-detail-r031-r040.md)
5. `R-041` ~ `R-050`：[requirements-detail-r041-r050.md](requirements-details/requirements-detail-r041-r050.md)
6. `R-051` ~ `R-060`：[requirements-detail-r051-r060.md](requirements-details/requirements-detail-r051-r060.md)

说明：需求细化内容按每 10 个需求拆分维护。
