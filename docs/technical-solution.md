# Technical Solution

> Last update: 2026-07-06

`alter0` 的技术方案按与需求清单一致的领域模型维护。后续新增或调整需求时，技术方案必须落到对应领域与子域，不再按时间顺序、任务编号或零散专题堆叠。

2026-07-03 更新：Web Shell 顶层入口收敛为 `/chat` 与 `/settings`。Chat 通过 `/api/chat/sessions` 复用 runtime session 数据模型和底层执行链路；旧双运行页的页面、接口、owner、任务说明、适配 path 和前端运行页注册已移除。`internal/chatruntime` 包名作为当前 Chat runtime session 实现边界保留。

## 维护规则

- 需求归属先确定领域路径，例如 `runtime.orchestration.intent`、`runtime.execution.codex-exec`、`task.workspace.runtime`。
- 技术方案使用相同领域路径补充实现信息，包括包边界、核心对象、调用链路、接口契约、存储形态、错误处理、观测字段和测试策略。
- 一个能力只允许有一个主归属领域；跨领域影响通过“依赖与边界”记录，不复制成多个重复方案。
- 影响架构边界、数据结构、接口、执行链路、存储、部署或研发治理的需求变更，必须同步更新本文件。
- 用户可见行为变化继续同步 `README.md`，稳定需求口径同步 `docs/requirements.md`，实现细节同步 `docs/requirements-details/*.md`。

## 领域方案索引

| 领域 | 主包/模块 | 技术方案重点 |
| --- | --- | --- |
| Runtime & Orchestration | `internal/interfaces`、`internal/shared`、`internal/orchestration`、`internal/execution/domain`、`internal/scheduler` | 统一消息、意图路由、Runtime Resolver、CLI Runtime、调度触发、观测与健康检查 |
| Conversation & Session Experience | `internal/interfaces/web`、`internal/session`、Web static assets | Chat 会话、历史隔离、移动端视口、消息渲染 |
| Skill & Memory | `internal/runtime`、`internal/execution`、`internal/orchestration`、`docs/skills`、`.alter0/memory` | CLI Runtime 上下文注入、Skill 仓库、Memory Context、会话摘要、长期记忆、记忆维护任务 |
| Task, Chat & Workspace | `internal/task`、`internal/tasksummary`、`internal/chatruntime`、`.alter0/workspaces` | 异步任务、日志流、心跳、产物交付、Chat 会话、工作区隔离 |
| Control, Operations & Governance | `internal/control`、`internal/codex`、`cmd/alter0`、`scripts`、`docs/deployment` | 控制面配置、Model Provider、Claude Code provider profile、Codex Runtime、运行时重启、部署凭据、测试与 TDD 约束 |

## Runtime & Orchestration

### 包边界

- `internal/interfaces/*` 负责外部输入适配，只生成内部统一消息，不承载业务路由。
- `internal/shared/domain` 承载 `UnifiedMessage`、`OrchestrationResult` 等跨领域消息对象。
- `internal/orchestration/domain` 承载 `Intent`、`Command` 等编排领域模型。
- `internal/orchestration/application` 承载意图识别、命令路由、Agent 执行分发。
- `internal/execution/domain` 定义执行端口和运行时上下文契约。
- `internal/scheduler` 负责 Cron 配置、触发与回注编排。

### 调用链路

```text
CLI / Web / Cron
  -> UnifiedMessage
  -> Orchestrator
  -> CommandHandler | ExecutionPort
  -> RuntimeResolver
  -> Claude Code + provider profile | Codex Direct
  -> OrchestrationResult
  -> Interface response / Session persistence / Task handoff
```

### 技术约束

- 外部输入必须先归一为 `UnifiedMessage`，再进入编排层。
- 命令路由优先于复杂度评估和模型执行；显式 `alter0.execution.engine=codex` 的消息会在编排层绕过命令路由，使 Codex CLI 内置斜线输入作为直连 Codex 内容进入 `ExecutionPort`。
- Cron 触发不直接调用执行器，必须复用编排链路。
- Cron runs 接口通过 Session history 按 `trigger_type=cron` 与 `job_id` 查询触发会话，不另建独立运行记录存储。
- `ExecutionPort` 是 Agent 执行能力的稳定边界；具体执行由 `RuntimeResolver` 选择 CLI Runtime。
- `RuntimeResolver` 按优先级选择执行器：空执行器、`auto` 或显式 `alter0.execution.engine=codex` 进入 `Codex Direct`；显式 `alter0.execution.engine=claude` 或携带 `alter0.llm.provider_id` 时解析对应 Model Provider 并进入 `Claude Code + provider profile`；Provider 不可用时进入 `Codex Direct`；Claude 执行失败不自动回退。
- Claude Code 运行前注入 `CLAUDE.md`、provider profile 环境、Skill、Memory、MCP 和工作区事实；Codex Direct 运行前注入 `AGENTS.md`、独立 `CODEX_HOME`、Skill、Memory、MCP 和工作区事实。
- 运行过程统一归一为 `RuntimeTraceEvent`：`source` 仅允许为 `provider / adapter / alter0`，分别表示底层 SDK/CLI 直接提供、工程 adapter 从稳定协议字段转换、alter0 本地确定性生成。事件 `kind / role / lifecycle / status / blocks / action` 不允许通过自然语言正文、标题、关键词或语言模式推断；Claude Code 与 Codex Direct 后续接入新事件时必须先在 adapter 层落成确定映射，再暴露给 Chat。
- trace、session、message、correlation 字段贯穿日志、指标、会话与任务。

### 验证策略

- 领域对象测试覆盖消息字段归一、路由结果和错误编码。
- 编排应用测试覆盖命令优先、自然语言分发、Cron 回注。
- 接口测试覆盖 CLI/Web 输入到 `UnifiedMessage` 的转换。
- Go 单测用例说明按 `docs/testing/unit-test-cases.md` 与各 Go 包路径下的 `TEST_CASES.md` 维护，并按 Runtime、Conversation、Skill、Task、Control 领域路径归档。

## Conversation & Session Experience

### 包边界

- `internal/session/domain` 定义会话与消息数据结构。
- `internal/session/application` 负责会话持久化、历史查询和删除清理。
- `internal/interfaces/web` 负责 HTTP API、增量轮询、Web 登录、页面路由和前端静态资源分发。
- `internal/interfaces/web/frontend` 负责 Web Shell 的 Vite + React 构建、legacy DOM shell 渲染和 `static/dist` 产物输出。
- `scripts/build_alter0_service.sh` 是服务二进制的统一构建入口：先在 `internal/interfaces/web/frontend` 执行 `npm run build`，校验 `static/dist/index.html` 中的哈希 JS/CSS 资产引用，再执行 `go build -o <target> ./cmd/alter0`。`start_alter0_service.sh`、`relaunch_service.sh`、`make build` 与 Runtime supervisor 的候选二进制构建必须复用该入口，supervisor 通过 `ALTER0_BUILD_OUTPUT` 指定候选输出路径，避免 Go 服务重建时嵌入过期前端产物。
- `internal/interfaces/web` 在输出主 Web Shell 与 `frontend_dist` workspace preview HTML 前，会扫描 `/assets/index-*.js|css` 引用，读取对应 JS/CSS 内容并注入 `?v=<content-hash>`；已有固定 `?v=` 会被服务端内容 hash 替换。页面继续使用 `no-cache`，构建资产继续使用长期 immutable 缓存，但重启、快进或预览刷新后只要资产内容变化，浏览器会命中新 URL，避免旧 bundle 因 immutable 缓存残留。
- `internal/interfaces/web/frontend/src/shared/api/client.ts` 负责统一 JSON 请求封装、错误收敛与登录失效回调，避免新前端页面继续散落原生 `fetch`。
- `internal/interfaces/web/frontend/src/shared/session/sessionHash.ts` 负责运行页会话短标识生成、短 hash 判定与短 hash 到完整会话 id 的前端列表解析；`Chat` 的运行页 URL 参数统一使用该入口把完整会话 id 派生为 8 位短 hash，左侧会话列表不展示短 hash，完整会话 id 与 Chat `chat_session_id` 仅保留在接口、持久化、Details 与工作区路径语义中。
- `internal/interfaces/web/frontend/src/shared/time/format.ts` 负责固定 `Asia/Shanghai` 的前端显示时区与标准时间格式，避免新旧页面时间口径漂移；管理页中需要分钟精度的额度重置、运行时间等时间戳也必须复用这里的共享格式器，而不是在页面组件里手写 UTC 文案。
- `internal/interfaces/web/frontend/src/shared/time/sessionListGroups.ts` 负责把Chat 会话列表按 `Pinned / Today / Yesterday / Earlier` 分组；调用方通过 `getPinned` 把置顶会话抽到独立首组，其余会话再按最近时间分组，避免 Chat 漂移成不同的分组策略。
- `internal/interfaces/web/frontend/src/shared/viewport/mobileViewport.ts` 负责移动端断点、键盘偏移阈值、composer 贴底偏移与 viewport baseline 计算，避免 Chat 与 route 页重复维护软键盘占位逻辑。初始同步、页面刷新或 WebView 恢复过程中，若没有输入框聚焦且上一帧没有键盘证据，状态机以 layout viewport 作为 baseline，不把短暂或持续偏小的 visual viewport 报告升级为键盘态；已有 baseline 且上一帧存在键盘证据或当前输入聚焦时，仍保留键盘诊断偏移。
- `internal/interfaces/web/frontend/src/shared/debug/clickDiagnostics.ts` 负责显式开关控制的前端点击诊断。入口只在 `?debug_clicks=1`、`?debugClicks=true` 或 `localStorage["alter0.debug.clicks"]="on"` 命中时注册 capture 监听与 `PerformanceObserver`，输出 `[alter0:click]` 与 `[alter0:longtask]` 控制台记录，默认路径不注册全局事件监听。
- `internal/interfaces/web/frontend/src/styles/root.css` 维护全局 motion token、标准缓动曲线与 `prefers-reduced-motion` 降级；`src/styles/shell.css` 的最终 `Interaction polish baseline` 覆盖层负责把按钮、导航、列表项、Composer、弹层、焦点环、等宽数字和滚动隔离收敛到同一交互合同。该层只增强反馈和可达性，不改变 Settings 静态 frame、移动端 workspace grid、Composer footer 或详情浮层的 DOM 契约。
- `internal/interfaces/web/frontend/src/shared/visibility/usePageActivation.ts` 是 Chat 的共享页面激活补偿入口，监听 `focus`、`visibilitychange`、`pageshow` 与 `online`，在页面可见时刷新当前 owner 列表并触发 owner 增量轮询，在 `resync_required`、缓存不完整或 user-only 待恢复时触发必要快照回源，通过短 debounce 合并同一恢复阶段的重复事件。
- `internal/interfaces/web/frontend/index.html`、`static/dist/index.html` 与登录页模板统一以 `html[lang="en"]` 启动；`src/app/WorkbenchApp.tsx` 通过写回 `document.documentElement.lang` 统一驱动中英文壳层文案切换。`renderLoginPage` 继续直接输出服务端 HTML，但视觉与文案已对齐工作台基线：复用 `IBM Plex Sans + Sora` 字体组合、近白卡片表面与安全入口 copy。
- `ConversationRuntimeProvider.tsx` 的会话状态提交入口统一通过 `setSessionsByRoute` 与 `setActiveSessionByRoute` 先写 `conversationRuntimeCache`、route 长期会话快照和轻量信息快照，再调用 React state 更新；`upsertRuntimeSession` 也必须走 `mergeRuntimeSessions`，不得以迟到 summary 直接替换本地 busy 会话。`sendPrompt` 在 input 请求未完成前追加本地 queued user 消息并置 busy；`normalizeRuntimeSession / mergeRuntimeSessions / mergePagedMessages` 继续按消息 id 合并服务端 turns，服务端返回对应 turn 后压缩同文本 queued user 消息并补入 assistant 结果，避免短 turns 页或旧列表摘要覆盖本地已追加时间线。消息排序使用稳定比较器：同一 turn 固定 user 在 assistant/Thinking 前，本地 queued user 在首个 busy assistant patch 前，不只按时间戳排序；会话列表排序使用 `activityAt / lastOutputAt / updatedAt / 最新消息时间 / createdAt`，置顶仍优先于活跃时间；缓存恢复也走同一排序，修正旧快照中的倒序消息。Provider 关闭 `useRuntimeSessionController` 的自动 progressive history，避免稳定会话在后台 `turn_before` 补页后直接刷新 active session。
- `ConversationWorkspace.tsx` 的长历史可见窗口使用 `timelineWindow + pendingHistoryScrollRestoreRef` 维护：按钮点击和滚动触顶都调用同一 `loadEarlierMessages`，先记录当前可见消息节点的 `data-message-id` 与相对容器 top 偏移，再记录 `scrollHeight / scrollTop` 作为兜底；本地已有隐藏消息时只扩展窗口，服务端仍有更早 turns 但本地无隐藏消息时调用 `runtime.loadEarlierHistory()` 显式请求 `turn_before` 分页。历史分页合入后优先按同一消息节点锚点恢复阅读坐标，并在下一帧二次校正，只有锚点不可测时才回退到高度差恢复。当前批次恢复完成前，同一会话的后续触顶 scroll 事件直接合并，避免连续事件把窗口一次性扩到全量、把阅读区强制带回顶部，或因 Markdown/图片高度回流导致可见内容向上突变。
- Chat 的 `ScrollJumpStrip` 只使用 `.runtime-message-user[data-message-id]` 作为跳转测量目标，保证 `上一条 / 下一条` 都按用户消息定位；assistant 消息、Thinking / Process 披露区和过程步骤不参与 Chat 阅读定位目标计算。Chat 仍按自身 turn/item selector 维持等宽输出定位。
- `ConversationRuntimeProvider.tsx` 接受 `chat` conversation route：读取本地活动会话、最近快照、服务端集合、单会话详情和用户手动 focus 时都会使用当前 route owner 的会话模型；历史 `chat` snapshot 与短 hash query 在加载时迁移为 Chat 会话，Chat 则继续恢复 Chat owner 会话。`session_id` query 只作为显式恢复输入和用户手动 focus 的输出；当 `/chat` 没有显式 `session_id` 时，Provider 以当前 owner 的服务端集合与本地最近快照合并后的第一条会话作为当前会话，不让其他 route 的活动会话覆盖最新入口。会话列表在前端按 `pinned -> created_at` 归一排序；对尚未进入 Session history 的空白本地会话，pin 请求失败时仍更新本地会话快照，保证当前工作区即时反馈不被 history 缺口吞掉。
- `internal/session/application.Service` 在 Session summary 中输出 `last_active_at` 与 `pinned`。`SetSessionPinned` 持久化置顶 metadata，`TouchSession` 写入活跃时间 metadata，`CleanupInactiveSessions` 按固定阈值扫描会话并返回删除、置顶跳过和扫描统计；没有显式活跃时间的历史会话使用最后消息时间参与排序和清理判断。
- `internal/interfaces/web/maintenance.go` 负责系统维护任务：服务启动时创建每日记忆维护与会话清理调度循环；`RunMemoryMaintenance` 通过编排入口发送系统消息并注入 `memory-maintenance` Skill；`RunSessionCleanup` 调用 Session application 清理超过 7 天不活跃且未置顶的会话，并同步删除任务关联与 Session workspace。
- `ConversationRuntimeProvider.tsx` 的恢复判定识别本地或远端存在的未完成 assistant，以及当前活动会话最后一条消息仍是 user 的窗口期；恢复流程在要求稳定 assistant 时必须等到详情接口返回非占位 assistant 或失败态后才 upsert。Provider 将恢复轮询条件与输入锁条件拆分：`shouldPollRuntimeBackedSession` 继续覆盖可恢复占位和 user-only 状态，`shouldBlockRuntimeInput` 只在会话或消息状态仍为 `busy / running / queued / in_progress` 时禁用 Composer，失败、中断或退出终态允许同一会话继续提交下一条输入。
- `internal/orchestration/application/SessionPersistenceService` 将非 Chat 入口的会话落库拆为请求开始与结果收口两段：`Handle` 进入下游执行前先追加本轮 `user` 记录，执行完成后追加 assistant 记录及 route、错误码和结构化过程。Chat 当前运行页以 Chat turn store 为恢复来源。
- `internal/interfaces/web/server_chat.go` 对 Chat owner 的输入使用服务端请求生命周期控制 runtime turn；浏览器刷新或前端主动断开请求只结束当前 HTTP 回传，不取消已进入 runtime 的执行，前端恢复链路依赖当前 owner 的 session 详情补拉最终结果。
- `internal/chatruntime/application.Service` 的 `Create` 与 `Recover` 只在全局 `sessions` map 读写期间持有服务锁；新建或恢复的 `runtimeSession` 放入 map 后立即释放锁，再执行 `persistSession`、session update hook 与 bounded detail 构建。update hook 可通过 `Get / ListTurns` 回读同一会话以发布 `session.created / session.updated`，但不得与创建/恢复请求形成锁重入。
- `internal/chatruntime/application.Service.List` 在读取内存 map 前调用 `syncMissingPersistedSessions`，扫描当前 state directory 中内存缺失的 JSON session，并按恢复逻辑补入 map；该同步不覆盖已有内存 session，避免轮询列表把正在运行的 turn 回退到旧文件快照。列表同步与创建/恢复一样在持久化事件发布前释放全局锁，迁移旧记录时只对补入的缺失会话写回。
- `ConversationWorkspace.tsx` 的移动端发送手势链路继续复用共享 `RuntimeComposer` submit capture，但在 `chat` 路由下会先检查当前聚焦的主 textarea，并在直达 `sendPrompt` 前主动 `blur()`；发送动作与软键盘回弹依赖浏览器动态视口恢复和 workspace grid footer 回贴完成，不再由键盘偏移变量驱动主布局。
- `ConversationWorkspace.tsx` 额外负责把 Conversation 会话态归一为共享 `statusTone`：当前 assistant 消息为 `streaming / queued / running / in_progress` 时输出 `busy`，显式错误、失败、取消或 `message.error` 输出 `failed`，其余稳定态输出 `ready`；同一派生结果驱动会话列表项和 workspace header。会话列表只消费 `busy` 并渲染 loading，其他状态不渲染行内状态灯；header 可见层只保留信号本身，状态名称仅通过无障碍名称与悬浮提示暴露，避免头部长期固定显示 `Ready`。
- `ConversationRuntimeProvider.tsx` 维护模块级 `conversationRuntimeCache`，专门覆盖同一 SPA 工作台内从 Chat 切到其他路由再返回的个人单设备恢复场景。缓存按 route 保存会话列表、活动会话 id 和每个会话完整已加载消息或 turns，写入时复制消息、附件和 process events，读取时按 24 小时 TTL 校验；命中时作为 Provider 初始状态，随后当前 owner 的集合接口继续回源并沿现有合并规则覆盖。当前活动会话若已经带有完整稳定消息缓存，仍只作为首屏加速来源；首次进入、刷新、前台恢复和 page-activation 会强制回源当前 active 会话的最新 bounded detail，普通非强制详情恢复才允许用 `revision <= detailRevision` 的稳定缓存短路。Provider 同步写入当前 route 的 24 小时 `localStorage` 完整消息快照，用于刷新、重开或 `sessionStorage` 丢失后的首屏恢复；同时写入当前 route 只含会话元数据的轻量快照，用于完整消息快照超出浏览器配额或被清理后的会话列表兜底。持久快照读取时先按 route 校验 TTL，再按 session 合并完整快照与轻量快照：完整快照保留消息、过程事件、附件与分页状态，轻量快照只补齐完整快照缺失的会话摘要，不得把已加载消息降级为空。本地快照不作为跨设备事实来源，删除、置顶、输入返回、列表刷新和详情刷新仍通过 React state 统一回写当前 route 缓存并等待服务端回源确认。

### 实时更新通道

`Chat` 会话更新采用 owner 级增量轮询，替代旧长连接与固定高频详情轮询。HTTP 快照仍是最终恢复入口，updates 只承载低成本增量：

```text
POST /api/chat/sessions/{session_id}/input
  -> Chat session store 持久化 user turn + busy summary
  -> SessionUpdateLog append session.updated / turn.started
  -> POST /api/chat/sessions/updates
     { since_event_id, limit, byte_limit, sessions[].turns[].event_seq_ranges }
  -> CLI Runtime 执行并持续 append turn.event.*
  -> finishTurn append turn.completed | turn.failed | turn.interrupted
  -> 客户端按 event_id / revision patch 本地 session
```

- 服务端 `SessionUpdateLog` 作为 runtime session store 的派生事件日志，和 session JSON 使用同一 owner/session 事实源。每条事件包含 `event_id`、`owner_id`、`session_id`、可选 `turn_id`、`event_type`、`revision`、`created_at` 和最小 payload。`revision` 按 session 单调递增，用于客户端幂等合并；`event_id` 在当前服务实例内单调递增，并保留最近窗口以支持 cursor 续接。
- 查询路径为 `/api/chat/sessions/updates`，使用 Chat owner；客户端必须以 `POST` 携带 `since_event_id`、`limit`、`byte_limit` 与本地 ack manifest。manifest 只覆盖当前 owner 下 `busy / recoverable` 的本地会话，结构为 `sessions[].id -> turns[].id -> event_seq_ranges / event_ids`。`event_seq_ranges` 使用闭区间压缩连续 step seq，例如 `[[1, 57]]` 表示前端已持有该 turn 内第 1 到 57 个 runtime step；少量没有稳定 seq 或非连续缺口的 step 使用 `event_ids`。响应为 JSON envelope：`owner_id / cursor / resync_required / has_more / events[]`。无变化时返回空 `events` 与最新 cursor，不返回会话列表或完整历史。
- 事件 payload 严格控制体积：`session.updated` 由 `registerChatSessionUpdateHook` 通过 `buildChatSessionEventDetail` 构造，相当于单会话详情的 `turn_limit=1` 最新页，携带最新 turn、`turns_paging`、标题、状态、置顶、更新时间、错误摘要和 revision，使前端在轮询命中时即可合并 busy 状态、过程事件与最终输出；返回前再根据 ack manifest 裁剪 `turns[].runtime_trace_events`，只保留前端缺失的 Thinking / tool / command step，并在发生裁剪时写入 `runtime_trace_events_partial=true`。前端把该字段映射为 message 级 partial 标记，按 `raw.ref / id / turn_id:seq` 合并 `processEvents`；同 turn 的详情快照若未携带 runtime steps，只补正文和状态，不清空本地已有 `processEvents`。`turn.event.appended / updated` 只包含 `RuntimeTraceEvent` 摘要、preview、status 和 `raw.has_detail`；`turn.completed` 可包含 final output 摘要或本 turn 最新 assistant 消息片段，但超出预算时只传 `turn_id / revision / has_detail=true`，由前端补拉详情。附件原图、完整历史页、大段 thinking 和完整 event detail 不进入 updates。
- 当 `since_event_id` 早于服务端保留窗口、服务重启后无法证明事件连续、或 session revision 不匹配时，updates 返回 `resync_required:true`。客户端按当前 owner 的本地会话列表筛出 `busy / recoverable` runtime-backed 会话并逐个补拉详情，再用后续 updates 继续增量更新。新进程恢复持久化 session 时，孤儿 `running` turn 会被收敛为 `interrupted`；正在服务进程内存中但已经没有 live worker 的残留 `busy / running` 会话，也会在列表、详情、turns、entries 和继续输入前通过 `reconcileOrphanedRuntimeSession` 懒校准为 `interrupted` 并写回 store。补拉详情必须覆盖本地旧 `busy` 快照。
- `POST /input` 成功进入服务端执行链后，HTTP 请求断开只影响本次响应写回，不取消 runtime goroutine。input handler 在启动 runtime 前必须先 append 首个 `session.updated / turn.started` 事件，确保刷新或轮询稍后启动时可以从快照和事件日志看到本轮已被接受。
- 前端在 `ConversationRuntimeProvider` 内维护 owner 级 update cursor：按 route 保存 `updateCursorByRoute`，页面可见且存在 `busy / recoverable` 会话时按短间隔请求 updates。请求前从本地缓存构造 ack manifest：每个可恢复 session 只上报最近若干已加载 turn，turn 内 process events 按 `seq` 压缩为连续区间，非 seq 事件用有限 `event_ids` 兜底。收到事件后走同一 `mergeRuntimeSessions / mergePagedMessages` reducer，只 patch 命中的 session，不重建完整 session 列表或已稳定消息；`session.updated` 携带 bounded turns 页时直接合并当前 turn 的 user、Thinking/process 和 assistant 结果。partial step patch 必须以追加/覆盖单个 step 的方式进入本地缓存，保留已展开/折叠状态、已加载 step detail 和本地已有 step 顺序；事件级合并以 `raw.ref / event id / turn+seq` 为稳定 key，若本地事件已经通过 detail 接口写入 `blocks` 且 `raw.has_detail=false`，后续摘要 patch 只更新状态、标题、生命周期等轻量字段，不覆盖 detail blocks，也不把该事件重新标记为待加载。同一 owner 下非当前活动会话只更新摘要和必要最近 turn 状态，不主动拉取完整历史。updates 返回空 `events`、只返回未命中本轮可恢复 session 的旧 backlog，或命中 session 但 revision/activity/messages/process steps 均未推进时，前端累加连续无进展次数，并在第 10、20、50 次以及之后每 50 次对仍处于 `busy / recoverable` 的 session 触发一次 bounded detail 兜底；命中会话出现新的 busy revision、activity、消息或过程步骤时重置退避计数，详情成功但会话仍需继续轮询时保留退避计数，只有命中会话收敛到不再需要轮询时才重置。
- `mergeRuntimeSessions` 需要把本地 `busy / recoverable` 视为高优先级瞬时事实：当远端列表摘要缺少完整 turns、revision 未推进或仅返回旧 `ready` 状态时，不覆盖本地 `busy`；当远端详情明确带有稳定 assistant、失败、中断或删除结果时，再收口业务状态。该规则同时作用于 workspace header、移动端标题按钮和左侧会话列表，避免状态在列表刷新后回跳。
- 前端事件合并按 `event_id` 去重、按 `revision` 保序、按 `session_id + turn_id + runtime_event_id` 幂等写入。`turn.event.appended` 可直接追加结构化过程摘要；`turn.event.updated` 只能 patch 已存在事件或触发单 turn/detail 补拉；`turn.completed` 如果 payload 不含完整 final output，只更新状态并补拉当前会话详情，不把 preview 当最终正文。
- 前端轮询状态独立于业务状态：请求失败、退避或页面隐藏只用于诊断或轻量状态提示，不会把 assistant 消息标记为 failed。只有快照或事件明确返回 `turn.failed`、`turn.interrupted`、`session.deleted` 或 404，才改变会话业务状态。
- 前端刷新恢复顺序固定为：读取本地 route 快照渲染首屏；拉取 owner 会话列表；对当前 active server session 补拉最新 bounded detail 校准正文、状态与 `runtime_trace_events`；历史分页、手动刷新或 updates 返回 `resync_required` 时继续按对应会话补拉详情；随后按当前 owner cursor 请求 updates。该顺序保证刷新过程中即使网络短暂失败，也不会把本地 `Thinking...` 或 user-only turn 直接收敛为失败，也不会把常规增量轮询退化为全量详情轮询。
- 轮询频率按可恢复状态驱动：页面可见且存在运行中会话时，短时窗口内按约 `1s` 只拉 owner updates，超过短时窗口后退避到约 `5s`；updates 未对本轮可恢复 session 产生相关进展时不立即回源详情，而是按连续无进展第 10、20、50 次以及之后每 50 次触发最新 bounded detail 兜底；新的 busy revision、activity、消息或过程步骤会重置退避计数，仍未收敛且也没有新进展时继续保持退避计数；稳定完成且 `has_more_before=false` 的完整缓存会话不参与详情补偿。
- 观测字段统一带上 `owner_id`、`session_id`、`turn_id`、`event_id`、`revision`、`update_cursor`、`resync_required`、`poll_interval_ms` 与 `disconnect_reason`。事件窗口过期和补快照都以 info/warn 结构化日志记录，业务失败仍只由 turn/session 终态记录。
- `internal/interfaces/web/frontend/src/features/shell/components/MessageMarkdown.ts` 负责消息正文 markdown 安全渲染前的输入归一化：除常规 `CRLF -> LF` 外，还需剔除零宽断行字符，并把“每字一行”的病态段落折回单段文本，避免 provider 或 adapter 写入异常换行后在真机上继续显示为逐字竖排。
- `internal/interfaces/web/frontend/src/features/shell/components/RuntimeTimeline.tsx` 在可折叠 Chat step 场景下必须稳定输出三列步骤头结构：`.chat-step-toggle-icon`、标题节点和 meta 节点按固定顺序进入 `grid-template-columns: 12px minmax(0, 1fr) auto`。不能只依赖 CSS 假定第一列存在，否则标题会落进 12px 图标列并在移动端退化成单字符窄列。
- `ConversationWorkspace.tsx` 必须把 Conversation 时间线装配结果按可见消息数组稳定 memoize，长历史初始只传入最新 32 条消息，顶部 `topContent` 渲染 `Load earlier messages / 加载更早消息` 控件，点击或滚到顶部后每次再扩展 32 条并按滚动高度差恢复阅读位置；本地隐藏历史已经展开且服务端仍有更早 turns 时，workspace 才调用 `ConversationRuntimeProvider.loadEarlierHistory()` 显式请求 `turn_before` 下一页。Workspace 记录上一帧完整消息 id 与可见消息 id；当历史分页只是在当前会话前方 prepend 旧消息时，当前渲染窗口继续使用上一帧可见消息，不因为全量缓存变长而刷新短会话正文区。只有用户触发加载更早消息时，`visibleCount` 才扩展到下一批历史。若新消息追加前当前可见窗口已经覆盖全部已加载消息，追加后 `visibleCount` 扩展到新的消息总数，避免发送后的本轮消息把原先可见历史挤出窗口。Workspace 还需基于 `.runtime-workspace-screen` 的 `scrollHeight - clientHeight` 测量结果维护 `data-runtime-scrollable`，通过 `ResizeObserver`、`MutationObserver` 和窗口 resize 同步短内容与长内容状态。`ChatMessageRegion.tsx` / `buildChatTimelineItems` 进一步以 `session + language + callback + message id + message signature` 缓存单条 `RuntimeTimelineItem`，并用消息对象引用缓存签名计算，使结果 patch 只重建当前变化的 assistant 消息，不重复生成稳定历史消息的 Markdown HTML 与 Process 树；`RuntimeTimeline.tsx` 继续通过 `memo` 只在 `items / topContent / emptyState / overlay` 真正变化时重渲染；`RuntimeWorkspacePage.tsx` 也要把 session pane、workspace header 与 workspace content 这些重节点 memo 成稳定 ReactElement。仅有 Composer 草稿变化时，不重复解析 Markdown、不重建 `Process` 树，也不重跑整条消息时间线。
- `internal/interfaces/web/frontend/public/legacy/chat-chat.css` 及其镜像产物 `internal/interfaces/web/static/dist/legacy/chat-chat.css` 继续承接 `chat-*` 旧 DOM 契约皮肤；其中 `.chat-log-text` 必须保持 `word-break: normal` 与 `overflow-wrap: break-word`，避免移动端 prompt bubble 在 shrink-to-fit 场景下被 `overflow-wrap: anywhere` 压成逐字断行。
- `internal/interfaces/web/frontend/src/styles/root.css` 与 `shell.css` 共同承担运行页横向边界：`html / body / #frontend-root` 使用 `overflow-x:hidden` + `overflow-x:clip` 与 `overscroll-behavior-x:none` 禁止页面级横向滚动偏移，`runtime-workspace-* / runtime-timeline / chat-* / message-markdown-*` 主容器统一声明 `min-width: 0`、`max-width: 100%` 与 `box-sizing: border-box`；长路径、错误日志、inline code、markdown pre/code 与 `.chat-md-table-wrap` 只在内容块内换行或内部滚动，不能把移动端顶部操作行、消息卡片或 Composer 撑出视口。
- `internal/interfaces/web/frontend/public/legacy/chat-chat.css` 中的 `.chat-step-title` 与 `.chat-step-richtext` 需要显式占满可用列宽，并声明 `min-width: 0`、`overflow-wrap: break-word` 与首尾段落 margin 修正，保证说明类步骤正文在窄屏下维持整列阅读，而不会被步骤容器缩成窄列。
- `internal/interfaces/web/frontend/public/legacy/chat-core.css` 继续承担 Conversation runtime 内容区与 `Process` 阅读层皮肤，其中 `runtime-process-step`、`runtime-process-step-head` 与 `runtime-process-step-body` 必须显式声明 `min-width: 0`、正文 `width: 100%` 与 Markdown 子节点的整列换行约束，保证真机窄屏下长中文步骤说明不会塌缩成逐字竖排窄列。
- `internal/interfaces/web/frontend/src/styles/shell.css` 中的 `.runtime-timeline` 必须以 `min-height: 100% + align-content: start + grid-auto-rows: max-content` 维持顶部收口；移动端 Chat 再通过最终覆盖层把 `runtime-workspace-screen / runtime-timeline / runtime-thinking-shell` 固定为顶部流式布局，这样在少量消息、短回复、加载中的 `Thinking` 披露行或折叠 `Process` 场景下，消息块与状态标签仍按内容高度自然堆叠，不会被满高 grid 轨道居中或拉伸。
- `RuntimeWorkspaceScreen.tsx` 负责把运行页消息滚动面与 overlay 控件分层：时间线继续放在 `.runtime-workspace-screen` 内独立滚动，`ScrollJumpStrip` 与 `chat-jump-cluster` 则挂到外层 `.runtime-workspace-panel` 作为悬浮层，避免按钮组继续参与消息流高度计算，把空白会话或短消息场景错误撑成可滚动区域。`shell.css` 的最终 Chat scroll guard 只在 `data-runtime-scrollable="true"` 时保留 `overflow-y:auto / -webkit-overflow-scrolling:touch`，短内容状态切到 `overflow-y:hidden / overscroll-behavior:none`；最终 Chat composer compact guard 继续在级联末端约束 footer 安全内边距、外层 form 圆角、内部 textarea 直角、工具行与按钮尺寸，避免输入面边框贴屏或底部留白撑高。最终 mobile horizontal overflow guard 额外约束 Conversation root、workspace、screen、timeline 与 composer 容器的横向裁切，避免 iOS Safari 偶发保留横向 scroll offset。
- `internal/interfaces/web/frontend/src/app/WorkbenchContext.tsx` 与 `WorkbenchApp.tsx` 统一维护移动端运行页面板状态：Chat 会话列表直接挂在主导航抽屉内，`Chat` 移动 workbar 都只暴露 `Menu` 抽屉入口；普通 `page-mode` 路由页新增的 `Menu` 入口也复用同一套状态切换与关闭路径，切路由、点遮罩或切会话时都通过同一条关闭链路收口，不再由各页面各自维护独立开关。
- 移动端键盘布局不再使用 `useRuntimeComposerViewportSync`、body-level Composer portal、fixed bottom 或 `runtime-composer-spacer`。`RuntimeWorkspacePage.tsx` 始终把共享 Composer 作为 workspace footer 传给 `RuntimeWorkspaceShell`；`runtimeKeyboardIsolation.css` 在 `shell.css` 之后加载，把 App Shell 改为 `height: var(--mobile-viewport-height, 100dvh)` 的动态视口容器，并在 `760px` 及以下把 workspace body 收敛为 `mobile header / minmax(0, 1fr) panel / Composer footer` 三行。grid 中间行用 `minmax(0, 1fr)` 自动扣除 header 与真实 Composer；正文 panel 只占据第二行，不再手写 `VisualViewport height - header - Composer` 的高度公式。运行页后方层不再消费键盘高度、fixed bottom 或 transform 改写高度。
- `RuntimeComposer.tsx` 为Chat共享主 `textarea` 的移动端输入契约：默认写入 `autocomplete="off" / autocorrect="off" / autocapitalize="off" / spellcheck=false / enterKeyHint="send"`，把主输入明确声明为普通命令文本输入，避免 iOS 在软键盘上沿追加钥匙串、卡片或地址类系统输入助手并露出底部残留页面层。
- `shell.css` 在 `@media (max-width: 760px)` 下对 `.composer textarea` 与 `.runtime-composer-input` 显式声明 `font-size: 16px`，把 iOS Safari 聚焦输入框时的自动页面缩放风险收敛在样式契约内，避免重新打开浏览器后首次唤起输入法造成横向裁切或分辨率突变。
- `RuntimeWorkspaceShell.tsx` 对移动端工作区头部的 `Menu / 标题 / Session / New` 按钮统一安装 `pointerdown(touch) capture` 与 `touchstart capture` 首触处理，并按按钮维度维护一次性手势锁；这些动作在输入框聚焦、软键盘打开或浏览器可能吞掉合成 `click` 的场景下仍由首个触摸直接执行，后续同一触摸链路产生的 `click` 不再重复触发。
- `RouteBodyPrimitives.tsx` 的 `CopyValueButton` 只为短值保留 `data-copy-value` 调试属性，长输出复制值不进入 DOM 属性；Chat 最终输出的复制按钮通过组件闭包持有 payload，点击时写入剪贴板，避免长日志在 DOM 中重复存储。
- `public/legacy/chat-chat.css` 中的移动端 `.chat-composer-shell` 不再声明 `transition: bottom ...`；主运行页 Composer 位置由 CSS 动态视口和 workspace grid 驱动，避免键盘回弹动画和 CSS 补间叠加造成卡顿。
- `shared/viewport/mobileViewport.ts` 仍保留移动端视口状态推导，用于诊断、面板高度上限与 `--keyboard-offset` 输出；Chat 主运行页布局不依赖键盘 offset 驱动 workspace header、正文 panel 或 Composer 位移，键盘后的可见高度由 `--mobile-viewport-height` 承接。
- Web 壳层的品牌展示统一由前端源与服务端模板共同维护：`frontend/index.html` / `static/dist/index.html` 负责 `Alter0 Chat` 页签标题，`renderLoginPage` 负责 `Alter0 Login` 与 `Alter0 Console Login`，`legacyShellCopy.ts`、`PrimaryNav.tsx` 与 `ConversationWorkspace.tsx` 负责导航品牌位、会话列表标题和运行区 copy；这些展示文案调整不影响 `alter0.*` 事件名、存储 key、cookie 或元数据字段等运行契约。
- `internal/interfaces/web/frontend/public/legacy/chat-routes.css` 继续承载 `Skill` 与部分 legacy route primitives 的类名皮肤，但视觉已对齐 shell 基线：`.runtime-route-card / .runtime-builder-form / .runtime-builder-managed-item` 统一使用白底主表面、浅灰辅助层、必要分割线和低对比选中态，避免 legacy 类名页面继续漂移到独立视觉体系。
- `static/dist/legacy/*` 当前仅承载兼容样式资源，不再包含 `/chat` 启动所需脚本。`/chat` 页面只加载 `static/dist/index.html` 中的 React bundle；兼容层通过 `app-shell[data-workbench-route]`、`data-route` 与 `data-conversation-*` 等稳定钩子让样式与页面结构继续协同工作，而不再让 legacy 脚本回写业务状态。
- 前端静态资源处理展示、输入、缓存、轮询和视口状态；会话恢复阶段允许把残留 `streaming` 消息归一为失败态或任务态，但不改写后端领域事实。

### 调用链路

```text
Web input
  -> Web handler
  -> UnifiedMessage
  -> Orchestrator / Skill execution
  -> JSON response | Task result
  -> Session history
  -> UI result patch / session detail recovery
```

### 技术约束

- Chat 默认绑定 `main` Skill，Settings 页面按目标 Skill 隔离会话历史。
- 根路径 `/`、`/chat`、`/login`、`/logout` 是稳定 Web Shell 入口；页面、受保护预览工作区与 API 共享同一登录态校验，静态只读 host 保留匿名访问。
- `/chat` 固定分发 `static/dist/index.html`，静态资源统一从 `static/dist/assets` 与 `static/dist/legacy` 提供；兼容层仅保留 legacy CSS，不再通过 legacy JS 启动 `/chat` 运行时。
- `static/dist/index.html` 仅保留前端挂载容器、字体与 legacy 样式入口；React 在 `frontend-root` 内渲染当前 Web Shell 所需的 legacy DOM 节点，并在运行时追加 source-owned shell 样式，确保既有 `id`、`data-*` 与布局结构保持稳定。
- React 壳层中的可变状态不得清空主工作区稳定实例；涉及 route body、消息区与 runtime panel/sheet host 需通过状态边界、结构化 snapshot store 与局部 DOM 更新把 React rerender 限定在安全壳层。会话列表、消息列表与 runtime panel/sheet 原生 DOM 由 React 直接消费并渲染自身状态；当前 React 托管入口集中在 `chat / settings`，Settings 内再切分 Runtime、Skills、Memory 与 Schedules。
- `ChatView` 采用“滚动内容区 + 固定底部 Composer”结构：欢迎区和消息区共享主内容栅格并各自独立滚动，Composer 独占底部行，避免空态欢迎区、消息流和输入面板相互覆盖。
- `/chat` 与 `static/dist/legacy/*` 统一返回 `Cache-Control: no-cache`，保证桥接期 HTML 与固定文件名 runtime 资源总能拿到最新版本；`static/dist/assets/*` 基于 Vite 哈希文件名返回 `Cache-Control: public, max-age=31536000, immutable`。
- 开发态可通过 `ALTER0_WEB_FRONTEND_DEV_ORIGIN` 启用 Go -> Vite dev server 反向代理：`/chat` 直接转发到前端开发服务器，`/@vite/*`、`/@react-refresh`、`/src/*`、`/node_modules/*` 等运行时资源也由同一代理提供；Vite 侧再通过 `ALTER0_WEB_BACKEND_ORIGIN` 把 `/api`、登录和健康检查路径代理回 Go。
- Chat 请求只负责回传本轮结果，前端断连不得取消已被 Web 层接受的后端执行。
- Skill 管理接口继续维护用户管理 Skill 的 `session_profile_fields`、`deliverables` 与 `completion_checks` 字段集，用于历史兼容和控制面编辑；服务启动时不再注册内置业务编排，Web 对话运行页也不再由 Chat 入口、Deliverables 面板或 Session Profile 面板驱动。
- `travel` 作为内置 Skill 提供旅游攻略规则，不再对应内置业务编排。选择 `travel` Skill 后，当前 Chat 会话仍直接由 Claude Code CLI 或 Codex CLI 执行；HTML 攻略、路线卡、图片资产和 `travel` workspace service 发布要求由 Skill 文档与 `preview-publish` 共同约束。
- `internal/execution/application.Service` 不再执行 Session Profile 抽取旁路；会话上下文只由用户消息、选中 Skill、Memory、MCP、模型配置和运行时事实共同组成。
- 读取本地缓存时先归一残留 `streaming` 消息，消息通过当前 route owner 的 session 详情恢复。
- Chat Web 前端不再使用 Web 流式网关；`ConversationRuntimeProvider.tsx` 只通过 `POST /api/chat/sessions/{session_id}/input` 获取更新后的 runtime session，并通过 Chat session 详情恢复请求断开后的最终消息。
- 请求异常收尾时，`ConversationRuntimeProvider.tsx` 会先调用 `/api/chat/sessions/{session_id}` 回补服务端已落库的最终 turn，再决定是否展示失败文案，避免把同一条 Chat 请求通过 fallback 端点重新提交一遍。只有在没有可用正文且回补失败时才渲染带刷新提示的失败文案。
- 页面初始化会对当前活动会话执行一次轻量恢复：只要本地存在历史占位、失败态助手消息，或者集合接口返回了不带完整 turns 的摘要会话，前端就会调用 `/api/chat/sessions/{session_id}`，以服务端已持久化 turn 覆盖本地失败态。
- 运行时执行器在运行期把工具动作与观察收敛为结构化 `RuntimeTraceEvent`；Chat 应用层在 turn summary 中输出 `turns[].runtime_trace_events` 后透传给前端，事件详情通过 `session_id / turn_id / event_id` 单独读取。
- 移动端输入区以 CSS 动态视口为有效高度来源，Composer 作为 workspace grid footer 参与同一高度闭合链路。
- 移动端 root 在窄屏下不再使用 fixed 定位隔离 document 滚动，也不锁定 `html / body / #frontend-root` 的 `overflow:hidden`；前端入口 viewport meta 声明 `viewport-fit=cover, interactive-widget=resizes-content`，让支持该属性的浏览器把虚拟键盘纳入内容视口 resize。App Shell 由 `runtimeKeyboardIsolation.css` 固定为 `height: var(--mobile-viewport-height, 100dvh)`，运行页和 workspace body 继承该可见视口高度链，不再使用 `calc(var(--mobile-viewport-height) + var(--keyboard-offset))` 参与键盘布局；Composer 是 workspace grid 的第三行 footer，不消费 fixed bottom 或 transform。正文滚动窗口由 workspace grid 中间行收敛到 Composer 上沿，`.runtime-workspace-panel` 不再自己计算可见高度，`.runtime-workspace-screen` 不再用额外 bottom inset 扩大可滚范围，避免浏览器工具栏或软键盘变化导致底部留白、内容裁切、页面级上移、顶部 header 位移或重复键盘补偿。
- `shared/viewport/mobileViewportSync.ts` 作为根壳层共享 controller，除常规 `resize / visualViewport.resize / visualViewport.scroll / focusin / focusout` 外，还在 `visibilitychange(visible) / window focus / pageshow` 上强制补做一次同步；浏览器从后台回前台、标签页重新激活或 iOS WebView 恢复可见时，重建视口诊断变量，主运行页键盘布局仍以 `--mobile-viewport-height` 和 workspace grid 为准。
- 移动端 Chat 在输入框与会话设置底部面板之间切换时，打开移动端设置面板前先 blur 当前输入，主输入框重新聚焦时则先关闭设置面板，再交给浏览器原生键盘和动态视口完成贴底。
- 移动端不安装 Composer viewport sync hook，不在 Composer 输入框首触时记录或回放页面级、workspace 级滚动锚点，也不调用 `window.scrollTo`。用户在消息区产生新的 `touchstart / pointerdown / touchmove / pointermove / wheel / scroll` 时，由当前滚动容器直接处理。
- 移动端不安装 `touchstart / touchmove` 滚动桥接，不阻止 `.runtime-workspace-screen` 的默认触摸滚动，也不通过脚本写入 `scrollTop` 模拟滚动；移动端键盘态消息区滚动必须继续走浏览器原生 `overflow` 滚动路径，避免破坏 iOS Safari 的惯性滚动。
- `runtimeKeyboardIsolation.css` 在移动端只在 overlay owner 要求不可交互时通过 `data-runtime-composer-interactive="false"` 禁用 workspace footer Composer 命中；常规键盘态下 Composer shell 和 form 保持原生命中，textarea、工具按钮、附件预览、发送按钮和配置面板不经过额外 pointer-events 复写。这样避免输入面空白区参与后方消息滚动补丁，也避免把真实输入 surface 变成半透明覆盖层。
- `RuntimeWorkspaceShell.tsx` 与 `RuntimeComposer.tsx` 共用 `runtimeKeyboardDismissal.ts`，在移动 workbar 动作、抽屉/面板入口、Composer 工具栏、附件与发送按钮的 `pointerdown / touchstart` 捕获阶段先释放当前 editable 焦点；主输入框本身和正文滚动区不安装该释放逻辑，保留浏览器原生输入与消息滚动手势。
- 桌面宽屏下 React 壳层使用 `shell.css` 中的 `--shell-reading-width=960px` 统一约束欢迎区、消息列与 Composer；legacy `chat-core.css` 继续基于主工作区可用宽度推导 `--content-width`，并让 `.message-list`、`.msg`、`.composer` 消费该宽度变量，避免消息列与输入区在不同渲染路径上出现双重宽度口径。
- React 壳层在 `shell.css` 中把 `1280px` 及以下统一视为抽屉式导航工作台：`primary-nav` 改为贴左侧视口边缘的全高抽屉，并在当前运行页直接承载会话列表，避免导航和会话列形成双浮层；桌面宽屏不再提供折叠侧栏阶段，`760px` 及以下继续压缩按钮和内边距，保证真手机宽度下的可触达性。
- `ConversationWorkspace` 负责运行页头部、`Details` 面板、消息区与 Composer 的排版，`ConversationRuntimeProvider` 负责 `compact` 断点感知、Chat session 输入收口和草稿恢复；其中桌面端输入性能约束由 provider 的延迟草稿落盘与 workspace 的时间线 memoization 共同保证。Go 侧源码测试与前端组件测试共同约束这组契约。
- `chat.js` 内所有前端时间展示统一走同一北京时间格式化器，固定 `timeZone=Asia/Shanghai`、`hourCycle=h23`；时间标签输出 `HH:mm`，绝对时间输出 `YYYY-MM-DD HH:mm:ss`；控制台与账户管理视图的分钟精度时间戳输出 `YYYY-MM-DD HH:mm`，同样由共享时间格式器负责。
- Cron 创建表单默认时区直接复用同一前端常量 `Asia/Shanghai`，不再依赖浏览器本地时区探测。
- `Chat` 会话列表前端继续按 `hashSessionIDShort(session_id)` 生成 8 位短 hash，并作为运行页 URL query、预览域名映射和排障记录引用；共享会话列表项不再展示短 hash，Chat 不再把完整 `chat_session_id` 填入 `shortHash` 字段。
- `src/app/routeState.ts` 负责运行页路由与会话 query 协调：路由只解析 canonical path，不再使用 hash fragment；`/chat` 统一识别 `session_id` 多会话恢复参数，写入时通过 `sessionRouteToken` 把完整会话 id 收敛为 8 位短 hash。主导航进入 `Chat` 时即使当前已经位于 `/chat`，也会删除旧 `session_id` 并派发路由同步事件；`ConversationRuntimeProvider.tsx` 监听该同步事件，在 query 缺失时切到当前会话列表第一项。`/chat` 兼容映射到 `/chat`，`ConversationRuntimeProvider.tsx` 在 Chat 初始化时读取 Chat owner session 集合并把本地旧 `chat` snapshot 合并到 Chat 列表；`/chat` 初始化时读取 Chat owner session 集合。Chat query 缺失或目标会话不存在时，运行页才回退到 Chat 独立 `sessionStorage` 快照与服务端列表默认项。
- Markdown 渲染必须避免原始 HTML 透传；长路径、代码块和 diff 只在内容块内部滚动。`MessageMarkdown.ts` 作为 Web Shell 的共享安全 Markdown 渲染器，被 `ChatMessageRegion`、Chat 步骤/最终输出、`RouteFieldRow` 的正文模式、Memory 文档、Control 描述、Skill/Codex 说明与 Session Profile 非等宽字段共同复用；解析核心使用 `markdown-it`，禁用 raw HTML，开启自动链接与软换行，并通过 renderer rules 保持 `assistant-inline-image`、`chat-md-pre`、`chat-md-inline-code`、`chat-md-table-wrap` 与 `chat-md-table` 等既有 DOM class。渲染器在进入 `markdown-it` 前继续清理零宽字符、修正“每字一行”的异常段落、剥离任务列表 marker，并把危险 Markdown URL 降级为可读文本；列表解析保留原始行缩进，把缩进更深的同类或异类列表、引用与代码块保留在父级 `<li>` 内，避免把层级关系拍平成连续顶层条目；机器标识类字段继续走纯文本或等宽展示，不进入 Markdown 解析。`shell.css` 在 `[data-runtime-view="conversation"]` 作用域下维护 Chat 的无框阅读流与 Markdown 视觉节奏：工作区正文白底无框，助手消息透明输出，标题、段落、列表、链接、引用和代码块通过共享 class 做弱边界排版；Markdown 表格保留真实 table DOM 与列对齐，但视觉上只使用横向分割线，不使用卡片外框、圆角边界或表头灰底；表格使用 `width: 100%` 与 `min-width: 100%` 铺满消息宽度，普通单元格声明 `overflow-wrap: anywhere` 处理长中文和长说明，表格内 `a/code` 继续 `white-space: nowrap` 保持链接、URL 和代码可复制性，只有真实不可断内容超宽时才触发表格块内部滚动，不改变 `MessageMarkdownShell` 的 DOM 与安全解析边界。
- `ChatMessageRegion.tsx` 统一负责 Conversation runtime 的消息正文与尾部元信息；已完成的 Chat 助手消息不渲染尾部元信息，运行中/排队/失败等瞬时状态才渲染紧凑状态标签，且不再附带逐条时间，避免在每条回复后重复输出 route/source/status/time 标签。
- `shell.css` 通过 `.runtime-workspace-head.is-sticky`、`.workspace-header-status` 和标题按钮维护运行页共享的固定 workspace header 视觉状态：标题区吸顶、状态按钮按 `ready / busy / failed / interrupted / exited` 输出统一颜色反馈，但可见层只保留信号本身，并通过 `inline-flex` 信号槽直接复用会话列表 `.runtime-session-signal` 的中心点、描边与波纹规格；会话 `Details` 入口并入当前标题按钮，不再渲染独立右侧详情按钮。`.workspace-details-layer / .workspace-details-backdrop / .workspace-details-panel` 负责把详情面板挂到顶层浮层、限制最大可视区域、提供点击外部关闭与独立滚动容器，并通过更高层级、明确背景和 `dialog` 语义保证浮层稳定可见，`.workspace-details-content / .workspace-details-summary / .workspace-details-body` 则把首屏统一为紧凑摘要栅格、窄标签字段行与压缩复制控件，Conversation 与 Chat 只在详情内容内部保留差异化组件。`ConversationRuntimeProvider.tsx` 分离 `inspectorOpen` 与 `inspectorTabOpen`：`inspectorOpen` 只控制 Composer 配置面板，`inspectorTabOpen` 控制当前 `Model / Tools / MCP / Skills` 内容区；`toggleInspector(tab)` 在当前 tab 上再次触发时只切换内容区展开状态，不影响 workspace `Details` 浮层。
- `shell.css` 额外维护共享 header 状态信号样式：`.runtime-session-signal` 及其 `ready / busy / failed / interrupted / exited` 变体负责 workspace header 的微型中心点、双层波纹脉冲和红黄绿状态令牌，并在 `prefers-reduced-motion` 下回退为静态信号；会话列表不再使用该状态灯，改用 `.runtime-session-loading` 表达处理中状态。
- `ScrollJumpStrip.tsx` 负责 Chat 的四键阅读定位条。目标计算以滚动容器内可见消息块或 Chat turn 为单位缓存测量结果，并在滚动、窗口 resize、DOM 变更、展开折叠和 watch key 变化时重算。`上一条` 在当前最上方可见块尚未对齐时先把该块对齐到顶部偏移；若该块已处在目标偏移位置，下一次点击直接指向它前一块，保证连续上跳不会卡在同一内容块。
- `ConversationWorkspace.tsx` 在 Chat 时间线滚动容器上只监听 `scroll`：普通滚动到顶且本地仍有隐藏消息时展开下一批本地消息，已完全展开且服务端仍有更早 turns 时调用 `loadEarlierHistory()` 显式请求 `turn_before` 下一页，不再调用 `refreshActiveSession()`，也不注册额外触摸历史加载监听。Workspace 通过消息 id 顺序区分“旧消息前置”和“新消息追加”，显式分页前置旧消息只更新缓存与下一批历史，不触发回底，只有当前会话尾部真实追加用户消息时才按发送语义贴到底部。
- `ConversationWorkspace.tsx` 的主输入框不在移动端 `pointerdown / touchstart` 捕获阶段取消默认行为，首次触摸输入框仍走浏览器原生软键盘手势；运行页不再对首触输入框调用 `focus({ preventScroll: true })`，也不安装 page scroll 锁或 workspace 背景滚动锁。兼容 Chat 组件保留同类输入恢复能力，但当前 `/chat` path 使用 Conversation runtime 输入链路和 Chat owner。
- `shell.css` 在共享 runtime 作用域下继续叠加 workbench 精修：会话列表项保留左侧激活竖线与尾侧三点更多按钮，菜单浮层相对条目右侧定位，导航栏内动作列固定为单按钮宽度，避免移动抽屉里标题被多枚操作按钮压缩；`Pinned / 置顶` 分组与时间分组共用同一低噪音 group header 与列表项布局，保证置顶组插入后不改变 rail 宽度或 `Sessions / New` chrome 位置；`Details` 面板使用带标题栏、显式关闭按钮、轻量遮罩和紧凑字段行的浅色 surface，空态阅读区与 Composer 使用同一套浅色 surface，最终扁平 reset 不再抹掉 Details 面板的必要边界与圆角；会话态消息区在 Conversation 视图下退回白底无框正文面，避免在主工作区内继续出现嵌套边框、背景分界或卡片式助手回复；首页 Composer 固定为外层 form 适度圆角、内部 textarea 直角的助手输入面板，textarea 去除内层边框和 resize，工具栏左侧对齐工作区工具、附件与可选 meta，右侧单独保留 icon submit；桌面端 form 使用 `width: min(100%, 860px)` 居中，移动端 form 回落为满宽，并由最终覆盖层把 `.runtime-composer-shell` 收敛为安全区内边距、把 form 压缩为紧凑高度、把 textarea 与工具行白色留白压缩到稳定尺寸。`760px` 及以下的共享 `.runtime-composer-shell` 在Chat 工作区内使用 workspace grid footer，不使用 fixed bottom、`bottom: 0`、键盘 offset 或 `transform` 创建额外合成层，避免 iOS Safari 键盘动画期间复用旧阴影层造成灰色残影或覆盖时间线尾部消息。`ConversationWorkspace.tsx` 在空态为 console panel 与 chat screen 追加 `is-empty` class，`shell.css` 以内部滚动容器承接空态滚动，并通过低对比网格与细弧线背景提升空态画布层次，避免窄屏空页把头部操作行顶离可视区。
- `shell.css` 在 `@media (max-width: 1100px)` 下对工作台性能做额外收敛：关闭 `body::before/after` 光晕层，移除 `primary-nav / chat-pane / mobile-backdrop / runtime-workspace-session-pane-backdrop / runtime-workspace-session-pane-shell / runtime-workspace-body` 的 `backdrop-filter`，把移动运行页与抽屉回落为静态浅色表面，减少真机滚动和抽屉切换时的整页合成开销。

### 验证策略

- Web handler 测试覆盖会话创建、历史隔离、流式事件和取消语义。
- Web handler 测试覆盖 `/api/chat/sessions/updates` 与 `/api/chat/sessions/updates`：owner 隔离、`since_event_id` 续接、`limit / byte_limit` 传输预算、ack manifest 裁剪已知 runtime steps、事件窗口过期时返回 `resync_required`、HTTP input 断开不取消后端执行。
- Chat application 测试覆盖 input 接受后先持久化 `busy` session 与 user turn，再发布 `session.updated / turn.started`；Codex event、完成、失败和中断分别生成幂等 revision 事件；服务重启或运行进程丢失后残留的孤儿 `busy / running` 会话必须在列表/详情读取和继续输入前校准为 `interrupted`，并持久化中断 entry 与 `Interrupted` runtime event。
- 前端组件测试覆盖不创建浏览器长连接、重复 event id 幂等丢弃、`resync_required` 触发单会话详情补拉、updates 命中时不对每个 busy 会话固定周期拉完整详情、连续空 updates 或连续无相关进展只在第 10、20、50 次以及之后每 50 次触发 bounded detail 兜底，并覆盖 advancing busy update 不触发详情兜底、latest user-only 待水合状态继续短周期读取 owner updates。
- 传输预算测试覆盖 updates payload 不包含完整历史 turns、大段 event detail、附件原图或超大 Markdown 正文；这些内容继续通过详情或懒加载接口读取。
- Web static delivery 测试覆盖主 Web Shell 与 `frontend_dist` workspace preview 的内容 hash asset version 注入，确保长期 immutable asset 缓存不会在服务重启或预览刷新后继续命中旧 bundle。
- 前端 E2E 覆盖 Chat、移动端输入、设置面板和长会话渲染。
- 前端组件测试需覆盖 React 工作台的稳定契约，至少校验 `WorkbenchApp` 的 canonical path 路由、语言切换、移动端导航收口、左侧主导航会话列表，以及 Conversation / Chat workspace 的固定 header、消息区、Composer 和 `Details` 面板未被回归破坏；Conversation 消息区还需覆盖轻量 IM 气泡 DOM 与样式契约、长历史最新优先渲染、加载更早批次和 Chat 运行态缓存的未过期恢复、接口回源更新、过期失效。
- `legacyRouteLayoutStyles.test.ts` 需继续对 `chat-core.css` 的 `Process` 阅读契约做源码断言，至少覆盖步骤标题收缩、正文整列宽度和 `max-width: 760px` 下的移动端可读性约束。
- 图片输入链路的最小稳定测试面包括：前端文件选择与剪贴板图片读取限制、Composer 附件预览与移除、Web 消息接口对附件元数据的编码、`RuntimeResolverProcessor` 对图片 part 的构造与禁回退约束、OpenAI Responses / Chat Completions 适配层对视觉内容的序列化。
- `src/app/routeState.test.ts`、`src/app/WorkbenchApp.test.tsx`、`features/shell/legacyShellConfig.test.ts`、`features/shell/components/PrimaryNav.test.tsx`、`shellLayoutStyles.test.ts`、`legacyRouteLayoutStyles.test.ts` 与各 `ReactManaged*RouteBody.test.tsx` 共同覆盖路由解析、三入口主导航、Management 工具入口、Management 页族标记、语言切换、Conversation runtime 入口、Skill/Chat/Memory/Control/Tasks/Sessions 页面取数与窄屏布局契约；Go 侧 `internal/interfaces/web/server_*_test.go` 继续通过源码与嵌入资产断言校验 `WorkbenchApp`、`ConversationRuntimeProvider`、`ConversationWorkspace`、`ReactManagedRouteBody`、共享样式和静态资源分发策略。
- 图片输入链路的最小稳定测试面包括：前端文件选择与剪贴板图片读取限制、发送 payload 与会话恢复预览资产的分离、Composer 附件预览与移除、AI markdown 图片渲染、Web 消息接口对附件元数据的编码、`RuntimeResolverProcessor` 对图片 part 的构造与禁回退约束、OpenAI Responses / Chat Completions 适配层对视觉内容的序列化。
- 回归测试优先覆盖空白会话重复、软键盘残留空白、整段列表重建、断流恢复与残留 `In Progress` 等高频问题。

## Skill & Memory

### 包边界

- `internal/runtime/application` 保留用户管理 Skill catalog 与历史兼容查询；内置业务编排 catalog 为空。
- `internal/control/domain` 与 `internal/control/application` 负责 Runtime Profile、Skill、MCP、Model Provider 与 Codex Runtime 的控制面配置。
- `internal/execution/application` 负责 Runtime Context 解析，包括 Skill、MCP、Memory、工作区事实和交付要求。
- `internal/execution/infrastructure` 负责 Claude Code 启动、Codex Direct 启动、运行日志解析、线程/会话状态持久化和错误收口。
- `internal/orchestration/application` 负责会话摘要、长期记忆、天级记忆、项目记忆和任务摘要召回。
- `docs/skills` 承载 file-backed Skill 仓库。

### 调用链路

```text
Natural language message
  -> Runtime Profile / Skill selection
  -> MemoryContext resolution
  -> Workspace runtime injection
  -> Claude Code + provider profile | Codex Direct
  -> Final response + Process/logs
  -> Session archive + memory maintenance inputs
```

### 技术约束

- 任务推理、工具调用和会话内压缩由 Claude Code 或 Codex CLI 自身完成；服务侧围绕运行时选择、上下文注入、日志解析、结果归档和记忆维护组织执行链路。
- `internal/execution/application` 在启动前为当前会话生成 Runtime Context：选中 Skill、Memory 摘要、MCP、工作区路径、仓库路径、附件路径、产物路径、可写边界和交付要求。
- Claude Code 路径在会话工作区生成 `.alter0/claude-runtime/`、`CLAUDE.md`、Skill 副本、Memory 注入摘要和 provider profile 环境。
- Codex Direct 路径在会话工作区生成 `.alter0/codex-runtime/`、`AGENTS.md`、独立 `codex-home/`、Skill 副本、Memory 注入摘要和 thread id。
- Web 上传的会话附件经 `internal/interfaces/web/server.go` 与 `session_attachment_store.go` 规范化后统一写入 `alter0.user_input.attachments`；图片附件额外保留兼容性的 `alter0.user_input.image_attachments`。`/api/sessions/{session_id}/attachments` 现在支持“原文件 + 可选预览”模型：图片仍落原图与预览图，普通文件只落原文件并让 `preview_url` 回退到 `asset_url`。Chat input 随后只携带 `id + asset_url + preview_url` 引用，服务端再解析出工作区内的原图路径写入元数据；前端渲染层再按场景分流，缩略位读取 `preview_url`，回显与预览弹层读取 `asset_url`。assistant 最终回复中的 markdown 外链图片则由 `internal/orchestration/application/session_output_image_assets.go` 在 SessionPersistenceService 中做结果后处理：仅对可下载的 `http(s)` 图片做抓取，写入同一 Session 附件目录，并把最终输出和结构化步骤里的图片地址改写为 `/api/sessions/{session_id}/attachments/{asset_id}/original`。Chat 输入与 Control Task follow-up 输入都会复用同一附件目录与交付 URL。图片附件进入 Claude Code 或 Codex CLI 时由对应 runtime processor 解析元数据；显式 Codex Direct 由 `internal/execution/infrastructure/codex_cli_processor.go` 从同一图片 metadata 读取 `workspace_path` 并生成 Codex CLI `-i <path>` 参数；Chat 侧普通文件则不进入多模态图片 part，而是在执行前写入 Chat 工作区并通过 prompt 注入稳定路径，交给 Codex 读盘。带图请求不会在模型链失败后静默回退到 Codex CLI，避免把视觉请求错误降级为纯文本执行。
- Memory Files 注入需要携带路径、存在状态、可写性、内容摘要、召回片段和截断标记。
- Memory 上下文路径由服务配置内置解析，并统一供 `internal/execution/application.MemoryContextOptions`、Web Memory 聚合服务、会话摘要读取链路与系统维护链路使用；启动命令不再暴露记忆路径覆盖参数。
- Markdown 上下文主存结构固定为根级 `AGENTS.md`、`SOUL.md`、`memory/USER.md`、`memory/MEMORY.md`、`memory/daily/<YYYY-MM-DD>.md`、`memory/projects/<project>.md` 与 `memory/conversations/<conversation_id>/summary.md`。`AGENTS.md` 由 execution memory resolver 以 `root_instructions` selection 注入，定位为运行规则上下文；`SOUL.md` 由内置 mandatory context 路径解析为强约束上下文；其余文件承载事实型记忆。
- 持久记忆 Markdown 由 CLI Runtime 维护；服务侧会话记忆只保留在运行态和 `ConversationSummary` 中，用于恢复、召回和维护任务输入，不把每轮会话、压缩片段或任务摘要直接写入天级记忆或长期候选 Markdown。
- 用户可见 Markdown 不写入 confidence、source、status、sensitivity 等机器元数据；检索索引可作为派生文件重建。
- 用户显式记忆写入由当前 CLI runtime 完成；会话归档由服务生成 `ConversationSummary`；任务摘要保留在 Task 领域对象与存储中，不再由 `RuntimeMarkdownStore` 直接追加到 Daily/Long-term Markdown；长期整理由系统维护任务启动 CLI runtime 并加载 `memory-maintenance` Skill 完成。维护任务入口 prompt 固定要求读取当日/昨日天级记忆、对照长期记忆、只提升稳定事实/偏好/决策/流程/约束、禁止复制原始 transcript、日志、密钥和一次性任务细节，合并重复项并报告变更文件与跳过候选。该维护任务状态由 Web `maintenanceService` 记录，并通过 Settings 的 Schedules 内置任务展示和触发。
- Skill Memory Web 聚合接口只读返回 `AGENTS.md` root instructions、`SOUL.md` 强约束、长期记忆、天级记忆、项目记忆、会话摘要与说明文档；任务摘要刷新走 Task summary 子域接口。

### 验证策略

- Execution 应用测试覆盖 Runtime Resolver、Skill/MCP/Memory Context 注入、工作区文件生成和 Provider 选择错误收口。
- Infrastructure 测试覆盖 Claude Code 启动参数、Codex Direct 运行目录、thread 状态持久化、日志解析和错误收口。
- Memory 测试覆盖 Markdown 编解码、会话摘要、长期召回、系统维护整理输入和任务摘要深检索。

## Task, Chat & Workspace

### 包边界

- `internal/task/domain` 定义任务状态、来源字段、摘要与执行元数据。
- `internal/task/application` 负责异步执行池、复杂度预判、任务生命周期和心跳续租。
- `internal/tasksummary/application` 负责任务摘要存储与重建；记忆 Markdown 写入由 Agent 维护，不由任务摘要模块直接落盘。
- `internal/chatruntime/domain` 定义 Chat 会话态、turn 和 step。
- `internal/chatruntime/application` 负责 Chat 会话持久化、恢复、输入续写和工作区分配。

### 调用链路

```text
High-complexity message
  -> Task acceptance
  -> Async executor
  -> Workspace
  -> Codex CLI / Skill execution
  -> Task logs + heartbeat + artifacts
  -> Session result summary
```

```text
Chat input
  -> Chat session store
  -> Codex CLI resume/start
  -> Turn/runtime event append
  -> Chat view model
  -> HTTP snapshot + owner update polling
```

### 技术约束

- Task 需要保存来源字段，支持从任务回会话、从会话查任务，并支持按触发类型、通道、来源消息与结果消息过滤。
- 长任务通过心跳续租运行窗口，任务日志观测与后台心跳分离。
- Control 任务交互式续写通过追加输入创建 follow-up Task，不直接改写原任务执行记录。
- Task 不再提供独立 Settings 页面；任务记录、摘要和来源回链只作为 Chat / Memory / Chat 运行链路的内部数据与只读上下文使用。
- Web 不直接暴露本地文件路径，产物通过引用、下载或预览接口交付。
- Task 产物列表响应需要过滤本地 URI；下载和预览由任务接口按 artifact id 读取并输出安全响应头。
- Memory 任务视图读取 Task 与 task summary 数据，支持任务摘要重建；重建结果保留在 Task 存储与视图数据中，不直接写入记忆 Markdown，也不直接执行 retry/cancel。
- 工作区按 Chat、Task、Chat 分层隔离，删除会话或 Chat 时同步清理对应目录。
- 直连 Codex 的 Chat 会话在自身工作区下维护 `.alter0/codex-runtime/` 与 `.alter0/codex-runtime/codex-home/`；Chat 的 Codex thread id 写入 `.alter0/codex-runtime/thread.json`，Chat 的 Codex thread id 写入 `.alter0/codex-runtime/threads/<YYYY-MM-DD>.json`；Chat 会话在 `.alter0/workspaces/chat/sessions/<chat_session_id>/codex-home/` 下维护独立 `CODEX_HOME`。
- Chat 会话态与 turn/runtime event 执行态分离，历史 `running / starting` 需要兼容归一。
- Chat 会话详情聚合 turn 摘要；event 明细按 `session_id / turn_id / event_id` 单独读取，避免会话列表一次性加载大块执行日志。
- Chat 应用层在 `ListTurns` / 单会话详情中直接输出 `RuntimeTraceEvent`，并随 turn summary 输出 `runtime_trace_events`；Chat 前端装配 turn timeline 时直接消费该结构，把事件类型与来源写入稳定 DOM metadata。详情按展开时通过 `raw.ref` 或事件 id 对应的 `event_id` 懒加载。
- Chat session store 的恢复和读路径共同维护运行态事实：持久化状态加载时会把旧 `running` turn 收敛为 `interrupted`；服务进程存活但 CLI worker 丢失时，`List / Get / ListTurns / ListEntries / InputWithAttachments` 在读取或 busy 检查前调用同一中断收口逻辑，追加系统 entry 与 `Interrupted` runtime event 后写回 session JSON。`turnRunning=true` 或仍有 `turnCancel` 的当前进程内运行请求不会被该懒校准处理。
- `internal/chatruntime/application/service.go` 在 `InputWithAttachments` 中把 Chat 附件规范化为 turn 附件、在工作区 `input-attachments/` 下写入本轮输入文件，并按类型拆分消费：图片继续通过 `codex exec -i <file>` 或 `codex resume -i <file>` 进入 Codex 视觉输入，普通文件则在同轮 prompt 中附上 `input-attachments/<turn_id>/<filename>` 形式的 workspace 相对路径，要求 Codex 按需直接读盘；turn 摘要与持久化快照同步保留附件元数据，供 Chat 输入草稿与图片历史回显复用。Chat 输入请求携带的 `SkillContext` 会在每轮执行前渲染到工作区 `.alter0/codex-runtime/skills.md`，并通过托管 `AGENTS.md` 指令要求 Codex 只应用本轮选择的 Skill；空选择也会写入“未选择”标记，避免旧轮次 Skill 指令残留。
- Chat 应用层在 Codex CLI 返回远端 compact 失败时，仅把当前 turn 标记为 failed，保留当前会话的运行线程指针、`chat_session_id`、原工作区与日志，并让后续输入继续 resume 同一 Codex CLI thread。
- `MessageMarkdownSyntaxFixture.ts` 的构建预览样例覆盖 ATX/Setext 标题、删除线、自动链接、嵌套列表、任务项、列表内引用与代码块、对齐表格和 raw HTML 转义；其中表格样例覆盖短字符、长中文、长 URL/代码和混合内容场景，用于同时验证短表格铺满、普通长文本单元格内换行与不可断内容内部滚动；折叠示例把 `<details>` / `<summary>` 标签放入 `html` fenced code block，折叠内容按普通 Markdown 继续进入同一渲染器；`/chat?markdown_demo=1` 只注入该非持久化样例，不写入会话历史。
- `internal/interfaces/web/server_chat.go` 的单会话详情默认返回最新 `20` 个 turns，`turn_limit` 上限为 `160`，并按约 `256KiB` 的 turns 页预算从最新 turn 向前装载；`turn_before=<turn_id>` 以目标 turn 为右开边界返回更早一页，并在 `turns_paging` 中输出 `total / byte_limit / approx_bytes / has_more_before / has_more_after / oldest_turn_id / newest_turn_id / next_before_turn_id`。集合接口继续只返回会话摘要，避免左侧会话列表拉取大体积 turns。
- Chat 跨设备共享同一 Web 登录态下的服务端会话历史，不再按 browser client 分桶。
- 共享 `shell.css` 在真手机宽度下负责运行页工作区头部和移动 workbar 的收缩规则：Chat 使用同一标题、状态信号和 `Details` 触发元素，长标题按共享规则截断或换行，避免横向溢出。
- `WorkbenchApp` 继续安装共享 `mobileViewportSync` controller 输出动态视口诊断变量，但 Chat 主运行页布局只依赖 `--mobile-viewport-height` 驱动后方高度闭合；移动端 `root.css` 在 `1100px` 及以下解除 `html / body / #frontend-root` 的 fixed 页面锁和根层 `overflow:hidden`，改用动态视口最小高度。`runtimeKeyboardIsolation.css` 在 `shell.css` 之后把 App Shell 改为动态视口容器，直接用 `height: var(--mobile-viewport-height, 100dvh)` 控制高度。真手机宽度下，workspace body 只保留 header footprint、正文 panel 和真实 Composer footer；正文 panel 是唯一内部滚动区且只占据 grid 中间行，不再手写高度公式。移动端主导航由 `WorkbenchApp` 的 portal 承载，`shell.css` 的最终层把 portal、遮罩和 `primary-nav` 同步裁剪到 `--mobile-viewport-height - --mobile-viewport-offset-top`，portal 内元素使用绝对定位贴合该容器，导航内部 session rail/list 负责滚动。运行页后方层不让 document 或普通流布局承接 iOS 聚焦滚动，也不再通过 transform 处理 offsetTop；命令候选、配置面板和阅读定位控件保持静态位置，左侧抽屉打开时先释放输入焦点，Composer 保持可见但不可交互，并由更高层级的遮罩和抽屉自然覆盖。

### 验证策略

- Task 应用测试覆盖复杂度分流、并发上限、心跳续租、来源字段和删除清理。
- Web 测试覆盖任务列表、详情、日志流、retry/cancel、产物下载/预览、Memory 任务摘要重建和会话回链。
- E2E 测试覆盖 Chat 移动端输入、滚动、Process 折叠和跨设备历史口径。

## Control, Operations & Governance

### 包边界

- `internal/control` 管理 Channel、Capability、Skill、MCP、Runtime Profile、Model Provider 与运行时状态。
- `internal/control` 中的 Model Provider 配置负责生成 Claude Code provider profile 所需的 base URL、API Key、model、profile 名称与健康状态。
- `internal/codex/domain` 负责 `auth.json` 快照、身份识别与额度状态模型；`internal/codex/application` 负责读取当前活动 `auth.json`、刷新 quota，通过 Codex app-server 的 `model/list`、`config/read`、`config/batchWrite` 读取真实运行时能力并更新 `model` / `model_reasoning_effort`，以及编排独立 Codex 登录会话。`LoginSessionStartRequest.AuthMethod=device_auth` 会把命令参数收敛为 `codex login --device-auth`；登录进程使用 Store 分配的独立登录目录作为 `CODEX_HOME`，stdout/stderr 通过流式 capture 写回会话日志，并从输出中宽松解析 `verification_uri / verification_uri_complete / user_code / expires_in / interval`。
- `internal/interfaces/web/maintenance.go` 仅作为系统维护任务执行器挂入 Scheduler；前端通过 Schedules 内置任务查看和触发维护，不再保留独立维护 route 或直接维护接口。会话置顶通过 `POST /api/sessions/{session_id}/pin` 更新。
- `cmd/alter0` 管理启动、supervisor、重启、内置配置和运行时 metadata。
- `scripts` 承载运行账户凭据、Node/Playwright 工具链和部署初始化脚本；Node 初始化同时覆盖 `internal/interfaces/web` 与 `internal/interfaces/web/frontend`。
- `docs/deployment` 承载 Nginx 与部署权限说明。

### 调用链路

```text
Control UI / API
  -> Config / Capability service
  -> Codex account service
  -> Local storage
  -> Runtime resolver
  -> Execution / Scheduler
```

```text
Runtime restart
  -> Web confirmation
  -> supervisor
  -> fetch origin/master restart candidates
  -> optional git fast-forward or target commit reset, or structured discard-confirmation error
  -> build candidate binary
  -> readyz probe
  -> switch or rollback
```

### 技术约束

- Control 面只能管理运行时配置，不绕过编排层直接执行业务请求。
- Schedules 控制面只管理系统内置维护任务的状态、启停和手动触发；每日自动运行时间、7 天不活跃阈值、置顶保护和 queued/running 任务保护规则固定在服务端，不作为用户配置暴露。维护任务不可执行或后续资源清理失败时写入 `failed` 状态，不使用空运行成功作为兜底。
- `cmd/alter0/builtin_skills.go` 负责注册内置 Skill，并在启动阶段校验所有 file-backed 内置 Skill 文件存在；当前公有内置集合包含 `memory`、`preview-publish`、`frontend-design`、`doc-coauthoring`、`fullstack-developer`、`code-reviewer`、`webapp-testing`、`find-skills`、`test-driven-development`、`ui-ux-pro-max`、`code-simplifier`、`code-review`、`brainstorming` 与 `travel`，`memory-maintenance` 作为系统维护专用私有 Skill 注册。标准 skill 使用源码目录下的 `docs/skills/<skill_id>/SKILL.md`；plugin-style 的 `code-simplifier` 与 `code-review` 继续保留目录内 `.claude-plugin/plugin.json` 元数据，并分别以 `docs/skills/code-simplifier/SKILL.md`、`docs/skills/code-review/commands/code-review.md` 作为 alter0 的 file-backed 注入入口。`preview-publish` 的内置 guide 与 file-backed skill 共同要求静态用户可见产物和完整测试服务先发布到 `https://<service>-<short_hash>.alter0.cn`，避免把服务端路径、本地 URL 或工作区内部路径暴露为用户可访问链接。执行层不再合成或自动写入私有 Skill，Chat 直接按会话选择的 Skill 注入 CLI 运行时。
- Runtime Context 物化器负责在 CLI runtime 准备阶段复制 file-backed Skill：按当前服务进程工作目录向上查找可读的 skill 文件，定位 `docs/skills/<skill_id>/` 根目录，把整个目录复制到当前会话工作区。Claude Code 路径写入 `.alter0/claude-runtime/skills/<skill_id>/`，Codex Direct 路径写入 `.alter0/codex-runtime/skills/<skill_id>/`，并将注入上下文中的 `file_path` 改写为工作区内副本。
- Codex Direct 的托管 `AGENTS.md` 由 `internal/codex/infrastructure/runtimeconfig` 生成，并固定包含工作区隔离、禁止把 `/srv/...`、`.alter0/workspaces/...`、`file://`、`localhost`、`127.0.0.1` 作为用户链接返回，以及静态产物、完整服务与后端路由统一走 `preview-publish` 的交付约束。
- 服务启动时不再注册任何内置业务编排；对应业务能力通过用户选择的 Skill 组合表达。
- Models 控制面需要保持空 API Key 语义、占位值过滤、禁用态恢复和默认 Provider 收敛；Runtime 页的快速注册入口只复用创建接口，不改变更新接口的密钥保留语义。
- 旧运行参数 registry、配置更新接口和 audit 视图已移除；用户可见运行时变更集中在 Codex Runtime 配置与服务重启控制。会话内上下文压缩由 CLI runtime 自身处理，不在服务侧暴露配置项。
- Codex Runtime 服务固定解析当前活动 `CODEX_HOME`，未显式设置时回退到 `$HOME/.codex`；当前服务运行账户的 `<active_codex_home>/auth.json` 与 `config.toml` 作为运行时认证和配置来源。
- Codex 运行时状态接口同时返回活动 `auth.json`、`config.toml`、当前 profile、活动 model、思考深度、配置来源、`model/list` 返回的可选 model 能力集，以及当前 `auth.json` 解析出的身份快照与实时刷新后的 quota 信息；更新运行时设置时，后端先通过 `config/read` 解析当前生效 key path，再调用 `config/batchWrite` 更新 `model` 与 `model_reasoning_effort`，并触发 `reloadUserConfig` 让当前运行时立即生效。登录会话接口复用 Codex account service，不把 device-code 状态写入长期账号存储；只有 `codex login` 成功产出登录目录下的 `auth.json` 后，才通过既有 `AddFromRaw` 保存账号快照并允许覆盖 `runtime-device` 记录。
- CLI Runtime 运行参数通过源码内置默认值、启动参数和运行账户配置解析，仍受 Provider、Claude Code 与 Codex CLI 实际能力约束。
- Runtime 重启必须由 supervisor 托管，候选实例通过 readyz 后才切换；Runtime 页打开重启弹窗时默认选择 `sync_remote_master=true`。候选接口由 `cmd/alter0` 执行 `git fetch --prune origin master`，基于运行中二进制的 `vcs.revision` 或仓库 `HEAD` 解析当前 commit，再按 `current..origin/master` 返回全部后续提交，并通过 `git log -n 11 <current>` 追加当前 commit 及其向前 10 个历史提交。同步请求未传入 `target_commit` 时执行既有快进；传入 `target_commit` 时先解析短 hash，确认目标提交可从 `origin/master` 到达，再对本地 `master` 执行 `git reset --hard <target>` 并通过统一构建入口重建前端产物和候选二进制。当存在 Git 已跟踪改动时，`cmd/alter0` 返回 `RuntimeRestartError`，错误码固定为 `runtime_restart_discard_confirmation_required`，Web API 与 supervisor client 必须透传同一 JSON `code`，前端只在收到该错误码后进入二次确认。未确认时不得清理或回滚本地工作区内容。
- 共享 Web 运行时内置通用 workspace service 注册表 `.alter0/workspace-services.json`：控制面 `PUT /api/control/workspace-services/{session_id}` 注册默认 `web` 服务，`PUT /api/control/workspace-services/{session_id}/{service_id}` 注册附加服务。`frontend_dist` 默认校验 git 工作区和 `internal/interfaces/web/static/dist` 构建产物，并在 Host 命中 `<session_short_hash>.alter0.cn` 或 `<service>-<session_short_hash>.alter0.cn` 时优先分发 `/`、`/chat`、`/assets/*` 与 `/legacy/*`；`travel` 服务是唯一前端静态例外，固定命中 `https://travel-<session_short_hash>.alter0.cn`，当注册路径根目录已存在 `index.html` 时直接把该目录作为静态攻略根目录公开分发，并继续对该 host 只返回静态 HTML/资源、直接阻断 `/api/*` 与其他工作台路由。`http` 服务既可反向代理到外部 upstream，也可由共享运行时按注册的 `start_command + workdir + port + health_path` 托管本地子进程。默认 `scripts/deploy_test_service.sh <session_id>` 会为 `web` 合成一条当前分支后端启动命令并注册给共享运行时，先构建前端产物，再让 `https://<session_short_hash>.alter0.cn` 整体代理到这份托管后端，从而让前端与 `/api/*` 保持同一版本；当 `service_id=travel` 且未显式传入 `--repo-path` 时，脚本默认回退到当前 Session 工作区根目录，直接发布已生成的静态攻略页。由于线上证书只覆盖 `alter0.cn` 与 `*.alter0.cn`，附加服务必须保持单级子域名格式，不能再生成 `<service>.<short_hash>.alter0.cn` 或 `<short_hash>.travel.alter0.cn` 这类二级嵌套 host。共享运行时自己的 `supervisor -> web child` 继续继承 `web_login_password` 作为主登录边界，托管 workspace service 子进程启动前会剥离 `ALTER0_WEB_LOGIN_PASSWORD` 并注入 `ALTER0_WEB_REUSE_GATEWAY_AUTH=1`，使预览后端只复用共享网关登录态，不再叠第二层鉴权。
- Web 登录态继续由 `server.go` 的 `authMiddleware + loginHandler` 统一管理；当请求 Host 命中主域或其预览子域时，登录 cookie 会把 `Domain` 收敛到根域 `alter0.cn`，使主域工作台与短哈希预览 host 共享同一登录会话，而不是各自维护孤立 cookie。交互页登录回跳通过 `loginNextForRequest` 归一化：`/` 与 `/chat` 只回跳到 `/chat`，`/chat` 只回跳到 `/chat`，其他 HTML 导航仍保留安全校验后的相对 Request URI；实际运行页的会话 query 由前端收敛为 8 位短 hash，避免会话级长 id 进入登录页和稳定页面 URL。
- systemd 基线统一 `HOME=/var/lib/alter0`，确保 Codex、gh、git signing、Node/Playwright 工具链使用同一运行账户上下文。
- 提交签名问题不得通过关闭签名绕过。
- 技术文档、需求文档和 README 更新按领域同步，避免需求与方案分离。

### 验证策略

- Control 测试覆盖 Channel、Capability、Skill、MCP、Runtime Profile、Codex Runtime、Schedules 内置任务、Capability 审计和服务重启请求。
- Web 接口测试覆盖会话置顶、7 天不活跃清理、置顶跳过、queued/running 任务保护、workspace 删除、维护执行器不可用、资源清理失败、Runtime 重启候选列表、目标 commit 透传和确认错误码；前端组件测试覆盖 Schedules 内置任务、Runtime 重启默认远端同步、master 候选 commit 展示与选择、按需二次确认、Codex device-code 登录和 Runtime 页 Claude Code Provider Console 的多 Provider 连续注册、查看与编辑。
- Provider 测试覆盖创建、更新、缺失密钥恢复、默认项收敛、Claude Code profile 生成和 OpenRouter 字段。
- Runtime supervisor 测试覆盖候选版本通过统一前端感知构建入口构建、master 候选列表包含全部后续提交与当前历史窗口、指定 master commit 重置、readyz 切换、失败回滚和 metadata 展示。
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
