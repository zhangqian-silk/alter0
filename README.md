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
internal/product/domain            # Product 领域模型（产品定义与主 Agent 上下文）
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
- `Chat / Agent` 新建会话先使用默认占位标题；自动标题在早期多轮内会按更具体的用户消息继续升级，尤其覆盖“拉取仓库 / 看仓库 / 分析仓库”一类通用开场，直到主题稳定。
- 运行时配置收敛在输入框底部单行操作栏：`Model`、`Tools / MCP`、`Skills` 与发送按钮同排；移动端默认折叠为单一“会话设置”入口，并与发送按钮共用同一行，优先保留输入与发送主动作。
- 移动端 `Chat / Agent` 的“会话设置”展开后采用独立固定底部面板，带独立遮罩、关闭入口和内部滚动区；配置内容不会再与底部发送条发生层叠覆盖。
- `Agent` 选项卡片在会话设置中使用短摘要展示：优先显示 Agent description，并限制在简短可扫读的卡片文案内；完整 system prompt 不直接出现在选择面板里。
- 会话设置中连续勾选 `Skill / Tool / MCP` 时，当前滚动位置需保持稳定，不能在每次勾选后跳回顶部。
- `Chat` 会话设置面板中的标题、说明与右侧标签在窄宽度下需保持可读：主标题按可用宽度截断，说明文案允许换行，避免发生重叠或互相覆盖。
- 移动端 `Chat / Agent` 输入区在软键盘弹起、收起与可视视口高度变化期间，会基于 `VisualViewport` 同步有效视口高度；输入区持续贴住可见底部，聚焦输入框不会被键盘遮挡；仅在输入框实际聚焦且软键盘占位达到阈值时才追加键盘底部偏移，浏览器工具栏伸缩或键盘收起后不保留额外底部留白。
- 移动端 `page-mode` 路由页会同步消费 `VisualViewport` 高度；`Terminal` 与其他信息页在浏览器底部工具栏伸缩、软键盘收起或可视视口回弹后，页面底边需立即回贴可见视口，不保留额外底部空白。
- 移动端 `Chat / Agent` 的后台任务轮询会按页面可见性自动降频；页面隐藏时停止高频扫描，恢复前台后再立即补一次刷新，降低持续耗电与发热。
- `Provider / Model`、`Tools / MCP`、`Skills` 可在会话过程中继续调整，并作用于后续发送的消息。
- `Alter0` 默认使用 ReAct 执行链，可直接完成通用任务，并在需要时调度专项 Agent。
- 选中的 Agent 工具会在模型调用时作为 function tools 注入；当前稳定工具集收敛为 `codex_exec`、`search_memory`、`read_memory`、`write_memory`，以及仅对可委派 Agent 暴露的 `delegate_agent`。
- `Models` 控制面支持同时维护 `OpenAI Compatible` 与 `OpenRouter` Provider；`OpenRouter` 可直接配置 `Site URL`、`App Name`、回退模型和 Provider 路由偏好，系统会分别注入官方请求头与请求体扩展字段。
- `OpenAI Compatible` / `OpenRouter` Provider 均支持按 `api_type` 选择上游接口：`openai-responses` 走 `/responses`，`openai-completions` 走 `/chat/completions`；配置自定义 `base_url` 时，需要目标服务兼容所选接口。`OpenRouter` 默认使用 `https://openrouter.ai/api/v1` 与 `openai-completions`。
- `Models` 控制面保存 Provider 时，`api_key` 输入框留空表示保持现有密钥；若前端中间态传入占位值 `-`，服务端会按空值处理，不会把 `-` 持久化为真实凭据。
- 历史 `model_config.json` 若残留缺失 `api_key` 的 Provider，加载阶段会自动收敛为禁用态并保留在 `Models` 控制面中，页面不会因旧配置直接返回 500；补齐密钥后可重新启用。
- 默认 Provider 只会落在已启用配置上；若默认 Provider 被禁用、删除或历史配置已失效，系统会自动切换到下一可用 Provider，无可用项时清空默认值。
- 复杂度评估阶段会优先复用当前消息选中的 `Provider / Model`；未显式选择时，回退到默认 Provider 与默认模型。
- 默认走实时执行。
- 流式对话会先直接启动回复；复杂度评估与回复并行进行。
- `Chat / Agent` 消息区在流式增量、Agent `Process` 展开收起与任务状态回填期间采用逐条 patch，并把高频刷新合并到浏览器逐帧节奏，避免长会话中反复整段重建消息列表。
- 当请求复杂度较高且仍在执行中时，系统会中途转为后台 `Task` 执行，并先返回一条任务说明消息，包含任务目标、执行计划与任务入口。
- 若当前消息已进入 Agent 执行链，前端页面切换、标签页隐藏、SSE 断开或浏览器主动取消请求都不会中断后端执行；连接只负责回传，最终结果仍会落到会话历史。
- 聊天气泡支持常用 Markdown 渲染，包括标题、列表、引用、链接、行内代码与代码块；原始 HTML 不直接透传。
- Chat 消息会标注实际回复来源，用于区分当前内容来自模型执行链还是 `Codex CLI` 执行链。
- Chat / Agent 助手最终回复提供一键复制入口；若同条消息包含 `Process`，复制内容仅包含最终正文，不包含折叠的执行细节。

2. `Agent`
- 面向“持续协助并推进执行”的目标型任务。
- 请求进入后会创建一个具备会话连续性的 ReAct 执行环，以当前任务为目标持续推进，并复用该 Agent 在当前 Session 内已经确认的稳定上下文。
- 运行时统一维护 `Agent Catalog`：同时聚合系统内置 Agent 与用户管理的 Agent Profile。
- 当前内置 Agent 包括：
  - `main`：默认对话主 Agent `Alter0`，可调度专项 Agent
  - `coding`：专项编码 Agent，负责理解开发需求、与用户保持交互，并通过 `codex_exec` 多轮推进代码修改、验证、预览页检查与结果收口
  - `writing`：面向文档、文案与结构化写作
  - `product-builder`：用于创建和扩展 Product 定义、主 Agent 方案与可复用 Prompt/Skill 沉淀
- `travel-master`：负责旅游 Product 的单主 Agent 执行与结果收口
- Web `Chat` 页面默认绑定 `Alter0`；`Agent` 页面提供统一 Agent 运行入口，用于承载未占用独立前端入口的内置 Agent 与用户管理 Agent。
- Product `master agent` 与兼容保留的 supporting agents 同样遵循统一的“Agent 负责持续协助与编排，Codex CLI 负责具体执行”模型。新生成的 Product 默认采用单主 Agent，主 Agent 默认使用 `codex_exec`、`search_memory`、`read_memory`、`write_memory`，并通过 system prompt 与 Skill 沉淀可复用规则；运行时自动补充 `complete` 收口。
- Agent 的职责收敛为“作为用户的持续助手并驱动执行”，而不是直接自己操作仓库或 Shell。稳定工具面只保留 `codex_exec` 作为具体执行入口，`Alter0/main` 与其他允许委派的 Agent 可额外使用 `delegate_agent`，所有 Agent 可在已注入的记忆文件范围内使用 `search_memory`、`read_memory`、`write_memory` 维护长期偏好、缩写映射与稳定协作约束。`search_memory` 用于按关键字跨记忆文件定位历史信息，再决定是否精读或更新具体文件。Agent 负责吸收用户意图、会话上下文、Agent 规则与记忆，并把它们转译成当前这一步的精确执行指令；这些 Agent 侧 prompt 与编排规则默认留在 Agent 自身，不直接透传给 Codex。所有 Agent 的 `codex_exec` 都统一携带结构化上下文，按需注入 `runtime_context`、`product_context`、`product_discovery`、`skill_context`、`mcp_context`、`memory_context` 等最小必要信息，让 Codex 拿到完整执行事实，同时避免 Agent 在每次调用里重复转述整段规则或传递无关信息。`coding` Agent 负责理解用户开发目标，并根据每次 `codex_exec` 的实际返回结果持续下发下一步实现或验证动作，直到任务完成或确认阻塞。`coding` Agent 的 `codex_exec` 在仓库类任务中会优先落到当前 Session 独立的 repo 完整 clone：`.alter0/workspaces/sessions/<session_id>/repo`，该目录自带自己的 `.git` 元数据，不依赖 `git worktree`；运行时会把当前仓库远端地址、源仓库路径、独立 repo clone 路径、当前分支、会话工作区、测试页预览域名与 PR 交付要求一并纳入结构化上下文。测试仓库的关键配置需要与正式服务保持一致，包括 model provider、Codex 执行路径、agent 路径等，只有会话缓存、会话历史等 session 级数据允许不同。涉及测试页面时，`coding` Agent 需要持续驱动 `codex_exec`，直到页面成功部署或更新到 `https://<session_short_hash>.alter0.cn`；预览成功后先由用户决定是否进行手动测试，再继续后续 GitHub 交付闭环。
- 若 Agent 在 `max_iterations` 耗尽前仍未显式 `complete`，运行时会返回带有“达到迭代上限”说明和最后一次工具观察的最终答复，避免 Web 流式消息在 `codex_exec` 观察后空收口。
- Agent 流式回复中的 `action / observation` 执行细节会在助手消息内收敛为可折叠 `Process` 区块；最终答复继续作为正文展示，默认在收口后优先突出最终结果，用户可随时展开回看过程。
- Agent 请求一旦进入后端执行链，浏览器侧任何交互事件都只影响当前连接状态，不影响 Agent 本身的执行与会话持久化；断开后重新进入历史即可查看最终结果。
- 每个 Agent 可独立配置名称、system prompt、tool 白名单、Skill 选择、MCP 选择与 Memory Files 选择。
- Agent 可按 Profile 勾选 `USER.md`、`SOUL.md`、`AGENTS.md`、长期 `MEMORY.md / memory.md`、当天与前一天 Daily Memory；其中 `AGENTS.md` 固定解析为当前 `agent_id` 的私有文件 `.alter0/agents/<agent_id>/AGENTS.md`，不与其他 Agent 共享；`USER.md`、`SOUL.md`、长期/日记忆继续作为共享记忆注入执行链路。
- 所有 Agent 还会自动携带一个私有、可维护的 file-backed Skill，默认文件为 `.alter0/agents/<agent_id>/SKILL.md`；它用于沉淀该 Agent 的可复用工作模式、输出结构、检查清单与稳定偏好，不需要在 Agent Profile 里手工勾选。
- `coding` Agent 的私有 `SKILL.md` 会进一步沉淀 Git 交付规则，例如 Session repo 完整 clone、认证签名提交、测试页部署成功后的快速 `gh` PR/merge 流程与交付汇报口径。
- `AGENTS.md` 与私有 `SKILL.md` 职责分离：`AGENTS.md` 负责当前 Agent 的协作边界、仓库/工作区操作规则与交付约束；`SKILL.md` 负责该 Agent 自身的可复用打法、模板和长期偏好。一次性任务细节仍应留在当前任务或记忆文件，不写入 Skill。
- 所有 Agent 会额外自动维护当前 Session 的私有画像文件 `.alter0/agents/<agent_id>/sessions/<session_id>.md`，并以只读 `Agent Session Profile` 注入执行链路；该文件用于沉淀当前 Agent 在该 Session 内的稳定工作上下文。`coding` 场景会自动写入源仓库路径、独立 repo clone 路径、远端地址、当前分支、Session 工作区与预览地址等关键属性。
- Agent 与 Chat 共用会话短期记忆链路：执行前优先使用进程内最近轮次；当会话因重启、跨天续聊或 TTL 淘汰导致内存窗口为空时，会从持久化 session history 回填最近完成轮次，再注入当前 prompt，保证 UI 可见的同一 Session 历史也能被模型用于指代解析和执行结果回顾。
- Memory Files 解析后会基于本轮用户输入执行轻量自动召回，命中的片段以 `memory_context.recall[]` 注入 Agent prompt 与 `codex_exec` 结构化载荷；`search_memory`、`read_memory`、`write_memory` 继续用于二次检索、精读和受控写入。
- 注入内容同时携带可写文件路径；Agent 只能在这些已解析出的记忆文件上使用 `search_memory`、`read_memory`、`write_memory` 做检索与持久化维护，其余具体文件与命令操作统一交给 `codex_exec`。
- `codex_exec` 调用 `Codex CLI` 时通过 stdin 传递渲染后的执行指令，命令行参数仅保留 `-` 作为 prompt 占位；长上下文、Memory、Skill 与运行时结构化载荷不会直接拼入系统命令行参数。
- Web 端将用户管理的 Agent Profile 放在 `Agent Profiles` 页面；系统内置 Agent 由服务注册，不能通过控制面覆盖或删除。
- Agent 的 `id`、`version` 等系统字段由服务端统一生成和维护，管理页不要求用户手填。
- `Agent` 页面继续保留独立执行入口与配置，并按当前选中 Agent 展示独立 Session 历史；带独立前端入口的 Agent 不进入该页历史。

3. `Terminal`
- 面向交互式终端会话。
- 仍属于自然语言处理，但使用独立上下文边界。
- 默认仅注入运行时必需上下文，不复用 Chat 会话记忆与长期记忆。
- Terminal 会话历史在同一 Web 登录态下对手机与 PC 共享，但每个 Terminal 会话仍使用独立工作区 `.alter0/workspaces/terminal/sessions/<terminal_session_id>`。
- Terminal 会持久化 Codex CLI 线程标识与会话状态；会话态固定为 `ready / busy / exited / interrupted`，其中 `ready` 表示当前会话可继续交互、`busy` 表示当前轮正在执行；执行细节继续由 turn/step 维度的 `running / completed / failed / interrupted` 表示。运行态退出后保留原会话历史，继续发送即可在同一会话内恢复。
- Terminal 新会话先使用占位标题；首条输入后会按输入内容自动命名。自动标题在早期多轮内会按更具体的后续输入继续升级，尤其覆盖“拉取仓库 / 分析仓库”等通用开场，避免列表里长期堆积低辨识度会话。
- 同一 Terminal 会话在单次运行态中断或退出后，只记录一条对应状态提醒；恢复后若再次发生新的中断或退出，再按新的状态周期补充提醒。
- Terminal 输入区上缘的运行态 hint 只服务于当前空闲会话；一旦用户重新发送恢复当前会话，或从旧会话切到 `New` 待创建态，旧的 `Exited / Interrupted / Failed` 提示会立即清空，不再在发送中残留。
- Terminal 工作区头部同时提供 `Close` 与 `Delete`；会话列表中的历史会话也支持直接删除：`Close` 仅退出当前运行态并保留会话历史与线程标识，`Delete` 会移除会话记录、持久化状态文件与该会话对应的独立工作区。
- Terminal 会话侧栏中的状态徽标与 `Delete` 按钮在长标题、双行元信息与中英文混排场景下保持统一胶囊高度与垂直居中，不因标题长度变化产生错位。
- 当前正在查看旧会话时点击 `New`，前端会先切入一个干净的待创建会话态；创建请求完成前不再沿用旧会话的 `Interrupted / Exited / Failed` 提示文案，也不继承旧会话残留的底部键盘留白。
- 同一 Web 登录态下，手机与 PC 访问同一批 Terminal 会话历史；刷新或跨端切换后不再因设备标识不同而看到不同会话列表。
- Terminal 不再设置产品级会话数量上限或固定超时淘汰策略。
- 访问 Terminal 时，轮询刷新不会重建已聚焦输入框；移动端输入法每次确认词句后，若输入框仍保持聚焦，页面继续延迟重绘并保持当前位置，直到失焦后再刷新视图；桌面端在连续输入窗口内也会暂缓非必要工作区重绘，待输入停顿后自动补齐刷新。
- Terminal 轮询刷新采用会话列表与工作区局部更新；当用户正在滚动输出区时，前端保留原消息滚动容器与滚动位置，不再按周期整块重建终端视图。
- Terminal 滚动状态同步会合并到浏览器逐帧刷新节奏内执行；上一条 / 下一条定位所需的 turn 位置在视图结构稳定时复用缓存，避免在连续滑动中反复全量测量消息区。
- Terminal 浏览器侧会话缓存采用滚动感知的延后持久化；输出持续增长时优先让出主线程给滚动与渲染，再在滚动停顿后写入本地存储。
- Terminal 会按页面可见性、输入聚焦与滚动活跃度自动调整轮询频率；活跃阅读或输入期间降频，页面隐藏后进一步延长刷新周期，恢复前台后再回到正常刷新节奏。
- Chat 与各路由消息流统一采用浅色文档式阅读主题：用户消息保留右对齐并限制在消息区宽度的 80% 内，但整体改为更轻的蓝灰气泡、助手回复弱化卡片边界与阴影，正文排版优先于装饰层级；用户消息与其后续回复继续保持更紧凑的同轮分组间距。
- Chat 移动端输入区默认隐藏装饰性附注与字数计数，底部只保留输入框，以及同排的发送按钮与可展开会话设置入口；会话设置以独立底部面板承载模型、工具与技能，统一使用当前语言口径，避免中英混排，并在小屏与软键盘场景下维持完整可读与可滚动操作区。
- Terminal 移动端采用紧凑 `Header + 独立滚动消息区 + 底部输入条` 结构；顶部通用栏保留 `Menu`，并在同一行右侧提供 `Sessions / New`，工作区头部收敛为会话标题、状态点与紧凑操作工具栏；发送区使用自适应输入框与紧凑发送按钮，运行态退出或中断提示以内嵌状态条贴合输入区上缘展示；消息与 `Process` 头部保持自然文档流滚动，不再启用吸顶导航；阅读定位由右侧圆形四键组承担，支持回到顶部、上一条、下一条与回到底部，并统一为浅色低对比滚动条与阅读主题；软键盘收起或浏览器底部工具栏回弹后，底部输入条需立即回贴可见底边，不保留额外占位空白。
- Terminal `Process` 步骤列表保持单行摘要阅读：步骤标题在可用宽度内自动截断，时间与状态固定收在右侧独立区域，不与标题文本重叠。
- Terminal 中四键导航会跟随当前视口里的可见 turn 重新计算上下目标；`Process` 折叠或展开后，上一条与下一条状态会随重排结果同步更新。
- Terminal 的整体视觉语言与 Chat 收敛：会话侧栏、工作区容器、输入区与输出块统一使用低对比边框、柔和玻璃感背景和更大的圆角节奏；用户输入不再额外展示命令前缀符号或强调色，最终回复直接按 Chat 助手消息样式渲染；`Process` 保留低对比虚线提示区和左侧纵向时间线，内部步骤压缩为单行摘要，展开后只展示详细内容，不再重复状态标签。
- 同一轮 Terminal 最终输出出现后，前端会自动折叠对应 `Process` 面板，把阅读焦点收敛到输出正文；用户手动再次展开后保留该选择。
- Terminal 最终输出正文提供一键复制入口，复制内容仅包含最终回复，不包含 `Process` 步骤细节。
- 移动端下 `Process` 头部与步骤行默认保持单行信息结构，`Process` 标签、步骤摘要、耗时与状态在可用宽度内同排阅读；超长摘要按单行截断，不再把每条消息挤成上下两行。
- Terminal 最终输出中的 Markdown 链接按链接文本渲染为可点击链接，不再直接暴露整段 Markdown 源码与长路径。
- Terminal 与 Chat 的长路径、超长单词、代码块和 diff 仅允许在内容块内部横向滚动，不再把外层消息卡片或聊天框撑出边框。
- 键盘弹起后输入区继续贴底可输入，页面不会因刷新回到顶部；浏览器底部工具栏伸缩、软键盘收起或视口回弹后，Terminal 工作区底边与输入条会立即回贴可见视口，不保留残余底部空白。

补充说明：

1. `Chat / Agent / Terminal` 只要落到 `Codex CLI` 执行链，都要求服务运行账户本身具备可用的 Codex / OpenAI 认证。
2. 若服务账户缺少认证，Web 端会快速返回认证失败，而不会长时间保持等待态。
3. 若服务需要在仓库内执行 `git commit`、`git push`、`gh pr create`、`gh pr merge` 等交付动作，部署时还需为运行账户补齐 GitHub App 凭证、`gh` 包装器与 SSH 提交签名配置；仓库内提供 `scripts/setup_alter0_runtime_auth.sh` 作为一次性初始化脚本。

## Product Model

Product 用于承载“某个业务产品 / 应用的主 Agent 与可复用领域上下文”，当前稳定行为如下：

1. Web 端新增 `Products` 页面，统一承载 `Workspace` 与 `Studio` 双视图；同一入口内既可维护 Product 定义、主 Agent、入口路由、知识源、产物类型与可选 supporting agents，也可查看 Product 概览、主 Agent 对话面板与具体详情页空间。
2. Product 由服务端统一维护 `id`、`version` 与 `owner_type`；内置 Product 只读展示，用户管理 Product 支持新增、编辑、删除、生成草稿并在审核后发布。
3. `Draft Studio` 当前提供 `POST /api/control/products/generate`、`GET /api/control/products/drafts`、`GET /api/control/products/drafts/{draft_id}`、`PUT /api/control/products/drafts/{draft_id}`、`POST /api/control/products/drafts/{draft_id}/publish`、`POST /api/control/products/{product_id}/matrix/generate`。新草稿默认只生成单主 Agent，并把领域规则沉淀到主 Agent 的 system prompt 与 Skill；`matrix/generate` 当前作为兼容保留的增量扩展入口，不再默认补出新的 worker matrix。
4. `Alter0` 在默认 `main` Agent 下会先做 Product 发现；命中 Product 后会补充 `matched_product_ids`、`selected_product_id`、`selection_reason`、`master_agent_id`、`product_execution_mode` 等元数据，并在执行型请求中自动切换到目标 Product 的 `master_agent_id`。
5. 当前内置 `travel` Product 默认公开可见，并绑定单一 `travel-master` 作为唯一执行入口。
6. `travel` Product 面向按城市聚合的旅游攻略场景，预留 `city_guide`、`itinerary`、`map_layers` 等产物类型以及 `city_profile`、`poi_catalog`、`metro_network`、`food_catalog` 等知识源；城市页规则、章节顺序与稳定呈现约定统一沉淀在主 Agent prompt 与 `travel-master` 私有 `SKILL.md` 中。
7. 已发布且公开的 Product 提供独立执行入口：`POST /api/products/{product_id}/messages` 与 `POST /api/products/{product_id}/messages/stream`；请求会自动绑定到该 Product 的 `master_agent_id`，并注入 `alter0.product_context` 与 `alter0.product.discovery`。
8. 已发布且公开的 Product 同时提供 Workspace 详情入口：`GET /api/products/{product_id}/workspace`，用于返回 Product 基础信息、主 Agent 摘要与详情页空间列表；支持通过 `GET /api/products/{product_id}/workspace/spaces/{space_id}` 查看具体页面内容，并为每个空间返回独立 HTML 页面地址。
9. 每个 Product 详情页空间都可映射为独立 HTML 页面；当前 `travel` 城市页默认使用 `/products/travel/spaces/{space_id}.html` 访问，例如武汉、成都、北京等城市页都可单独打开。
10. `travel` Product 在 Workspace 中提供“主 Agent 对话 -> 城市页同步”链路：`POST /api/products/travel/workspace/chat` 会优先由 `travel-master` 解析用户意图，并将结果落到具体城市页空间；当 Agent 执行链暂不可用时，服务端会自动回退到本地规则解析，继续创建或修改如武汉、成都、北京等页面。
11. `travel-master` 使用专属私有 file-backed Skill，默认路径为 `.alter0/agents/travel-master/SKILL.md`；该文件预置城市页生成规则与 HTML 页面呈现约定，运行时会作为规则上下文交给 `codex_exec`，并仅在用户提出稳定可复用偏好时更新，不会把一次性行程约束写入 Skill。
12. `travel` 额外保留结构化攻略接口：`POST /api/products/travel/guides`、`GET /api/products/travel/guides/{guide_id}`、`POST /api/products/travel/guides/{guide_id}/revise`；攻略输出稳定包含景点、地铁、路线、美食、说明与地图图层字段，便于后续接地图高亮和路线渲染。

## Workspace Model

默认运行策略保持 `danger-full-access`，当前默认执行目录策略统一为“各执行会话独立工作区”：

1. `Chat / Agent`
- 默认执行目录：`.alter0/workspaces/sessions/<session_id>`
- `Chat` 与 `Agent` 会话历史可继续按各自会话维度回放；删除会话时会同步清理对应会话工作区

2. `Async Task`
- 默认执行目录：`.alter0/workspaces/sessions/<session_id>/tasks/<task_id>`
- 任务工作区随所属会话工作区隔离，删除所属会话时一并清理

3. `Terminal`
- 终端会话工作区：`.alter0/workspaces/terminal/sessions/<terminal_session_id>`
- 终端会话状态：`.alter0/state/terminal/sessions/<terminal_session_id>.json`
- 同一 Web 登录态下，手机与 PC 访问同一批 Terminal 会话历史，不再按浏览器设备标识分桶

说明：

1. 工作区目录仅决定默认执行目录与运行时产物落点，不等同于文件系统权限收缩。
2. 当前默认仍为 `danger-full-access`，因此是否可访问其他绝对路径，仍取决于宿主机环境与运行账户权限。
3. 具体执行统一交给 `Codex CLI`，默认执行目录会落到当前 Chat / Agent / Task / Terminal 会话各自的独立工作区。
4. Chat / Agent / Product 执行链交给 `Codex CLI` 的最终指令通过 stdin 输入，避免不同操作系统在长上下文场景触发命令行长度限制。

其中 `Chat` 再细分为两种执行方式：

1. `Sync`
- `POST /api/messages`：普通 JSON 一次性返回结果。
- `POST /api/messages/stream`：通过 SSE 流式返回 `start / delta / done` 事件；复杂度评估与回复并行进行，若请求在评估完成时仍需长耗时执行，会在同一条流中切换为异步任务并返回任务受理结果。
- `Chat / Agent / Product` 的 SSE 流在长时间无增量输出时会持续发送保活帧，避免 Codex CLI 静默执行阶段被浏览器或代理提前断开连接。
- 对已进入 Agent 执行链的消息，SSE 断开只会终止当前前端回传，不会取消后端执行。
- 同一会话内的同步请求保持串行；当上一条同步执行尚未结束时，后续用户消息会继续等待并按序执行，不再因为默认队列等待时间直接返回 5 秒超时。
- 对于启用 ReAct 多步执行的同步请求，执行中若同会话收到新的用户补充，后续迭代会自动吸收当前最新一条用户消息继续推进。

2. `Async Task`
- 适用于高复杂度、长耗时或产物型请求。
- 请求被接受后先返回任务受理结果，后续可通过任务视图或任务 API 跟踪状态、日志与产物。
- 异步任务使用独立后台执行池，不占用主会话同步交流的串行执行槽位。
- 落到 `Codex CLI` 的异步任务会维持运行态心跳：执行期每 1 分钟记录一次存活心跳，并把任务超时窗口按心跳续租，避免长时间运行但仍健康的会话被固定 90 秒窗口误杀。
- 任务运行心跳仅用于后台执行存活与超时续租；浏览器侧流式连接保活由消息 SSE 通道单独负责。
- `Tasks` 列表卡片会先展示轻量心跳摘要，`Tasks` 详情与 `Memory -> Tasks` 详情会同步展示 `Last Heartbeat / Timeout Window`，用于区分“仍在健康运行”与“长时间无心跳、即将超时”的后台任务。
- `Tasks` 详情抽屉中的日志流会把高频日志重绘合并到浏览器逐帧节奏内执行；连续输出时不再为每条日志事件同步整块重绘日志容器。
- 异步任务完成后，回写到聊天区的是一轮精简后的结果摘要；完整终端输出、原始报错、代码片段与文件内容仅保留在任务详情、日志与产物中。

`Agent` 使用独立入口：

1. `POST /api/agent/messages`
- 同步返回 Agent 最终执行结果。

2. `POST /api/agent/messages/stream`
- 通过 SSE 返回 Agent 执行过程中的动作、观察与最终结果。
- Web 前端会把这些动作与观察解析成可折叠 `Process`，避免长执行细节直接铺满正文区域。
- 若前端在执行途中断开连接，后端会继续完成 Agent 执行；中断的仅是当前 SSE 回传，不是 Agent 任务本身。

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

1. 点击“重启服务”后会打开单一站内确认弹窗；“同步远端 master 最新改动”作为弹窗内勾选项展示，默认勾选。
2. `sync_remote_master=false`：基于当前仓库状态构建候选二进制，并由 `supervisor` 完成子进程切换。
3. `sync_remote_master=true`：先校验当前分支为 `master`、已跟踪工作区干净，再执行 `git fetch --prune origin master` 与 `git merge --ff-only FETCH_HEAD`，随后构建候选二进制并切换。
4. 候选版本只有在 `/readyz` 探活通过后才会成为当前运行版本；若启动失败，会自动恢复上一运行版本。
5. Git 或构建失败会直接返回到 Web 控制台，便于定位权限、凭据、快进合并失败等问题。
6. `Environments` 工具栏会展示当前在线实例的最近启动时间与对应 `commit hash`，用于确认上次成功重启切换到的运行版本。
7. 重启完成后页面会自动刷新到新实例，并以站内成功弹窗提示用户当前页面已连接到最新运行实例。

### Public Deployment Baseline

公网部署建议使用 Nginx 反向代理，并开启应用内登录页：

```bash
export ALTER0_WEB_LOGIN_PASSWORD='请替换为强密码'
export HOME=/var/lib/alter0

go run ./cmd/alter0 \
  -web-addr 127.0.0.1:18088 \
  -web-bind-localhost-only=true \
  -web-login-password "$ALTER0_WEB_LOGIN_PASSWORD"
```

若通过 `systemd` 运行，建议在服务环境中显式设置 `HOME=/var/lib/alter0`；启动脚本也会把历史 `HOME=/var/lib/alter0/codex-home` 归一到 `/var/lib/alter0`，确保 Codex 认证与运行态数据落在统一运行根目录。

若服务需要自行提交签名 commit、创建 PR 或执行合并，还需在 root 下额外执行一次：

```bash
sudo ./scripts/setup_alter0_runtime_auth.sh
```

该脚本会把 `alter0` 运行账户的 GitHub App token helper、`gh` 命令包装器、SSH signing key 与全局 Git 配置初始化到 `/var/lib/alter0`，用于服务内 `Codex CLI` 的提交 / PR / merge 链路。

若希望服务内 `Codex CLI` 可直接执行 `npm install`、`npm run test:e2e`、`npx playwright install chromium` 等 Node/Playwright 测试链路，还需在 root 下额外执行一次：

```bash
sudo ./scripts/setup_alter0_runtime_node.sh
```

该脚本会把带 `npm`/`npx`/`corepack` 的 Node 运行时安装到 `/var/lib/alter0/.local`，并默认在 `internal/interfaces/web` 目录预装 `npm ci` 与 Playwright Chromium 浏览器，使服务运行账户在非交互式环境中也能完整执行前端 E2E 测试。

之所以默认落在 `/var/lib/alter0/.local`，是因为这里属于 `alter0` 服务运行账户自己的运行时目录：既不会污染系统全局 `/usr/local/bin`，也不依赖宿主机预装 `npm`。脚本会把实际安装目录中的 `node`、`npm`、`npx`、`corepack` 软链接到 `/var/lib/alter0/.local/bin`，再由服务启动时补齐该目录到 `PATH`，这样 `Codex CLI`、Web 子进程和手工切到 `alter0` 账户执行时看到的都是同一套稳定工具链。

新服务启用时，建议直接按下面顺序执行：

```bash
# 1. 准备运行环境
sudo install -d -m 750 /etc/alter0
sudo sh -c "printf 'ALTER0_WEB_LOGIN_PASSWORD=请替换为强密码\nALTER0_RUN_AS=alter0\nALTER0_RUNTIME_ROOT=/var/lib/alter0\nHOME=/var/lib/alter0\n' > /etc/alter0/alter0.env"
sudo chmod 600 /etc/alter0/alter0.env

# 2. 确保公共路径可见 codex / node / gh
which /usr/local/bin/codex
which /usr/local/bin/node || which node
which /usr/bin/gh

# 3. 准备 GitHub App token 生成器
sudo test -x /usr/local/bin/github-app-token
sudo test -f /etc/github-app/config.json

# 4. 初始化 alter0 运行账户的 git / gh / ssh signing
sudo ./scripts/setup_alter0_runtime_auth.sh

# 5. 初始化 alter0 运行账户的 node / npm / playwright
sudo ./scripts/setup_alter0_runtime_node.sh

# 6. 重启服务
sudo systemctl restart alter0.service
```

若不是默认部署路径，可在执行初始化脚本前覆写这些变量：

```bash
sudo ALTER0_RUN_AS=myservice \
  ALTER0_RUNTIME_ROOT=/data/myservice \
  ALTER0_HOME=/data/myservice \
  ALTER0_REPO_DIR=/srv/myservice/app \
  ALTER0_GIT_USER_NAME='my-bot[bot]' \
  ALTER0_GIT_USER_EMAIL='123456+my-bot[bot]@users.noreply.github.com' \
  ./scripts/setup_alter0_runtime_auth.sh
```

初始化完成后，建议至少做一次快速验证：

```bash
sudo -u alter0 env HOME=/var/lib/alter0 PATH=/var/lib/alter0/.local/bin:/usr/local/bin:/usr/bin:/bin gh auth status
sudo -u alter0 env HOME=/var/lib/alter0 PATH=/var/lib/alter0/.local/bin:/usr/local/bin:/usr/bin:/bin bash -lc 'printf "protocol=https\nhost=github.com\n\n" | git credential fill'
curl --noproxy '*' http://127.0.0.1:18088/readyz
```

验证通过后，服务内由 `Codex CLI` 发起的 `git commit`、`git push`、`gh pr create`、`gh pr merge` 会复用这套运行账户级凭证与签名配置。

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
- 运行时会自动补齐 `$HOME/.local/bin`、`$HOME/.local/share/pnpm`、`/usr/local/bin`、`/usr/bin` 等标准 PATH，确保服务内 `Codex CLI` 可见 `codex`、`node`、`npm`、`npx` 与运行账户自带的 `gh` 包装器
- 如需固定 shell，可通过启动参数 `-task-terminal-shell` 或运行时环境键 `task_terminal_shell` 指定
- Windows 下显式指定 `cmd.exe` 时会补充 UTF-8 代码页初始化；如需稳定中文输出，优先使用 `powershell.exe`
- Terminal 会话退出后不会清空历史或线程标识；重新在原会话发送输入时，系统会优先复用已持久化的 Codex CLI 线程继续执行
- 对已退出或已中断的 Terminal 会话重新发送输入后，输入区上的旧运行态提示会立即让位给当前发送态，不继续显示“会话已退出”之类的过期提示
- `POST /api/terminal/sessions/{id}/close` 用于退出当前 Terminal 运行态但保留原会话历史；`DELETE /api/terminal/sessions/{id}` 用于直接删除 Terminal 会话，并同步清理 `.alter0/state/terminal/sessions/{id}.json`，接口返回 `204 No Content`
- 若 Terminal 会话在首条输入前已失去底层运行态，首次发送会自动恢复同一会话并继续执行，不要求用户新建会话
- 同一 Web 登录态下，Terminal 会话历史默认跨设备共享；手机与 PC 均通过同一组服务端持久化记录恢复会话，不要求用户迁移历史

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
3. 每个 Agent 在运行时都会自动附带自己的私有 file-backed Skill，默认路径为 `.alter0/agents/<agent_id>/SKILL.md`；该 Skill 不出现在控制面内置列表里，但会稳定注入当前 Agent 的执行上下文，供 Agent 根据用户提出的长期偏好更新自己的可复用规则。
4. `travel-master` 的私有 `SKILL.md` 会预置 travel 城市页、行程、地铁、美食与地图输出规则，作为 travel agent 独占的可复用规则簿；稳定偏好写入该文件，一次性行程细节仍只保留在目标城市页数据中。

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
    "tools":["codex_exec","search_memory","read_memory","write_memory"],
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
    "tools":["codex_exec","search_memory","read_memory","write_memory"],
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
4. Agent 运行时固定采用“Codex CLI 负责具体执行、Agent 负责理解和驱动”的模式：稳定工具面包括 `codex_exec`、`search_memory`、`read_memory`、`write_memory`，系统会自动补充收口工具 `complete`；允许委派的 Agent 可额外启用 `delegate_agent`。`search_memory` 负责在已解析的记忆文件内按关键字检索历史偏好、缩写和上下文，再配合 `read_memory` / `write_memory` 做精读和更新。Agent 作为用户与 Codex 之间的代理层，负责在自身侧消化 system prompt、Skill、记忆与会话上下文，并只把当前执行所需的最小上下文和具体指令交给 `codex_exec`；稳定执行事实统一通过 `alter0.codex-exec/v1` 的结构化上下文字段按需传给 Codex，不要求 Agent 在每轮指令中重复搬运全部规则，也不允许把与当前执行无关的上下文一并透传。`coding` Agent 会优先把实质性开发与验证步骤交给 `codex_exec`，并按每轮执行结果继续推进后续步骤；仓库类执行默认切到当前 Session 独立的 repo 完整 clone `.alter0/workspaces/sessions/<session_id>/repo`，该 clone 自带自己的 `.git` 元数据并允许直接分支与提交，运行时同步注入源仓库路径、独立 repo clone 路径、活动分支、会话工作区、预览域名与 PR 交付规则。需要测试页面时，完成实现后必须先部署或更新 `https://<session_short_hash>.alter0.cn` 再结束本轮交付。
5. `Chat` 默认绑定 `main` Agent；`Agent` 页面作为其余入口 Agent 的统一运行页，并按目标 Agent 隔离维护独立会话历史；具备独立前端入口的 Agent 不进入该页历史。
6. Agent 的 Skill、MCP 与 Memory Files 选择会在执行前注入运行时上下文，Memory Files 会基于本轮输入自动召回少量相关片段并写入 `memory_context.recall[]`，执行过程仍复用统一编排链路；同一 Session 的短期记忆会在内存窗口不足时从持久化 session history 回填最近完成轮次。
7. 内置 `memory` Skill 会明确记忆文件的读写逻辑：按任务类型决定先读哪些文件、按信息类型决定写入哪个文件、遇到冲突时按 `SOUL.md > AGENTS.md > 长期记忆 > 日记忆` 收敛；其中 `AGENTS.md` 固定为当前 Agent 的非共享规则文件 `.alter0/agents/<agent_id>/AGENTS.md`，`USER.md`、`SOUL.md` 与长期/日记忆继续共享；实际文件快照仍由 `memory_files` 注入提供。
8. 每个 Agent 还会自动拿到自己的私有 Skill 文件 `.alter0/agents/<agent_id>/SKILL.md`；运行时会把它作为可写 Skill 上下文注入，Agent 需要在用户提出稳定、可复用、会影响后续该 Agent 行为的偏好时按需更新它，而不是把一次性任务细节写进去。
9. `memory_files` 当前支持：`user_md`、`soul_md`、`agents_md`、`memory_long_term`、`memory_daily_today`、`memory_daily_yesterday`。
10. Memory Files 注入会携带文件内容、绝对路径、是否存在、最近更新时间；文件不存在时仍会暴露预期路径，便于 Agent 直接创建并写入。
11. Web `Agent Profiles` 页面用于管理用户自定义 Agent Profile；内置 Agent 由服务托管；`Chat` 页面绑定 `Alter0`；`Agent` 页面作为其余入口 Agent 的通用交互入口。

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
5. Draft Studio 生成或发布 Product 时，会自动把 `master agent` 归一到当前稳定执行模型；即使草稿中残留旧版工具配置，发布后也会补齐 `codex_exec`、记忆工具、默认记忆文件与领域 Skill。历史 `worker agent` 配置仍可兼容读取，但新生成草稿默认不会再新增多 Agent 矩阵。

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
