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

- [Architecture Design](./docs/architecture.md)
- [Technical Solution](./docs/technical-solution.md)
- [Domain Requirements](./docs/requirements.md)

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
- 负责配置 `Channel / Skill / Agent / CronJob`。
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
internal/interfaces/web            # Web 适配器 + Control API + 前端产物分发
internal/interfaces/web/frontend   # Vite + React 前端工程（构建输出到 static/dist）
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

Web 前端分发采用双层缓存策略：`/chat` 与 `static/dist/legacy/*` 下的兼容样式资源保持 `no-cache`，确保页面与样式刷新及时；`static/dist/assets` 下带哈希的构建产物使用长期 immutable 缓存，减少重复下载。

当前 Web Shell 使用单一 React 工作台：左侧固定主导航负责路由与语言切换，主工作区按运行态或控制页渲染具体内容；`chat`、`agent-runtime` 与 `terminal` 都收敛到统一的 workspace 骨架，运行页公共结构由共享的 `RuntimeWorkspaceShell / RuntimeComposer / RuntimeWorkspaceHeader` 链路直接产出，`agent / memory / channels / skills / mcp / models / environments / codex-accounts / cron-jobs / sessions / tasks` 等页面继续由 React 直接请求控制台或会话 API 渲染。此前用于 `/chat` 的 `LegacyWebShell / ReactRuntimeFacade / alter0:legacy-shell:* bridge / snapshot store` 已移除，`legacy` 资源仅保留样式兼容层，不再参与任何前端业务运行时。
当前桌面工作台基线收敛为两层：左侧品牌导航保持全高固定栏，顶部使用纯文字品牌位与语言切换，右侧主面板统一承载运行页和控制页。`Chat / Agent Runtime / Terminal` 的产品体验统一采用「会话列 + 主时间线工作区 + 底部 Composer + 固定 workspace header」的工作台结构；PC 端控件统一采用 8-14px 的低圆角矩形节奏，状态、短 hash、上传、发送、详情、弹窗按钮和跳转控件都不使用胶囊形态，保持更利落的控制台质感。进入 `1100px` 及以下窄屏后，三条运行页继续在工作区顶部直接提供 `Menu / Sessions / New` 操作行，会话列切为独立抽屉，不再和正文上下堆叠；普通 `page-mode` 路由页也会在标题上方补一行 `Menu` 入口，保证进入 `Tasks / Sessions / Models` 等信息页后仍能直接拉起主导航。三条运行页共享同一套 runtime workspace 页面骨架、标题区、滚动区、`Status / Details` workspace header 和双层 Composer 语义：输入区独占上层，底部工具栏左侧收口为正方形低圆角的会话设置与附件按钮，会话设置使用四点网格图标，附件使用回形针图标，配置面板内部再切换 `Agent / Model / Tools / Skills` 标签，右侧只保留紧凑 icon submit；会话侧栏统一显示 `Sessions` 标题与 `New` 新建入口，按最近时间分组为 `Today / Yesterday / Earlier`，条目统一收敛为「状态 / 标题 / 摘要 / 底部短 hash / 尾侧轻量操作」的工作台列表项；空白 Chat / Agent Runtime 会话默认标题为 `New`，空态工作区使用低对比网格与细弧线背景，但不额外挂载 route hero 或营销式说明块。Terminal 仅在终端路由内部叠加自己的布局皮肤与交互状态。控制类页面继续复用近白主表面、浅灰说明层和浅蓝选中态，不再派生独立的高装饰卡片系统；头部导航入口仅在 `1100px` 及以下切换为抽屉式交互，`760px` 及以下再压缩按钮与间距，保证真手机宽度下的可触达性。
`/chat` 与 `/login` 默认以英文文案和 `html[lang="en"]` 启动；Web Shell 内可通过语言切换入口改为中文，运行时文案与 `document.documentElement.lang` 同步更新。登录页与工作台共享 `IBM Plex Sans + Sora` 字体基线、近白卡片表面与安全入口语气，不再保留独立的默认系统登录页观感。
控制类与资产类页面默认采用更高信息密度的管理视图：`Profiles` 使用短字段并排的紧凑表单栅格与显式启用开关，`Tasks` 使用左侧任务列表 + 右侧运行详情的主从布局，`Memory` 的任务历史使用表格 + 详情侧栏，`Environments` 使用运行态工具栏 + 模块卡片栅格展示配置项，并在同页提供敏感值显隐、保存、重载、重启与审计视图，`Codex Accounts` 使用精简后的运行时概览条 + 当前 Codex 管理区 + 托管账号卡片列表 + 导入/登录操作侧栏：概览条优先展示当前账号、套餐、小时/周剩余额度与托管数量，维护类信息如 auth/config 路径、CLI 命令与配置来源收纳到 `Runtime Details` 折叠区；当前 Codex 管理区的 model 与思考深度选项直接来自 Codex app-server 返回的真实能力列表。`Channels / Skills / MCP / Models / Cron Jobs` 这组共享控制台卡片页统一复用稳定的响应式卡片网格，真窄屏下状态徽标会下沉到标题区下方、字段行改为单列展开，避免标题、徽标、复制按钮与多行字段互相挤压；`Agent` 的列表卡片、管理表单与详情区统一使用近白表面、浅灰说明层与浅蓝选中态。大屏保留“列表 + 侧栏”结构，中屏切换为全宽账号区 + 双侧栏，小屏回落为单列卡片，确保额度信息、当前 model、思考深度与切换入口始终可见。

前端开发态支持双向代理联调：为 Go 服务设置 `ALTER0_WEB_FRONTEND_DEV_ORIGIN=http://127.0.0.1:5173` 后，访问 `http://127.0.0.1:18088/chat` 会转到 Vite dev server；为 Vite 设置 `ALTER0_WEB_BACKEND_ORIGIN=http://127.0.0.1:18088` 后，`npm run dev` 会把 `/api`、登录与健康检查请求代理回 Go 服务。

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
- `Chat / Agent` 新建会话先使用统一占位标题 `New`；自动标题在早期多轮内会按更具体的用户消息继续升级，尤其覆盖“拉取仓库 / 看仓库 / 分析仓库”一类通用开场，直到主题稳定。
- `Chat / Agent Runtime` 统一使用 terminal-style workspace：左侧主导航之外，运行页内部固定为会话列、主时间线工作区、底部 Composer 与固定 workspace header；header 统一只保留当前会话标题、状态按钮与 `Details` 入口，`Details` 面板顶部先用紧凑摘要栅格承载会话元数据，再由各页面继续展开专属配置内容。消息、过程步骤与最终输出都落在同一时间线中推进，不再保留桥接期的欢迎区 / runtime sheet 双轨壳层。会话列、workspace、chat screen 与 composer 全部由共享的 `RuntimeWorkspaceShell / RuntimeComposer / RuntimeWorkspaceHeader / RuntimeTimeline` 直接渲染，运行页公共 DOM 以 `runtime-*` 为唯一主契约，三条运行页只在各自路由内注入数据和变体皮肤。
- `Chat / Agent Runtime` 的会话图片资产统一落在当前 Session 工作区：用户选图后前端通过 `POST /api/sessions/{session_id}/attachments` 把原图与预览图写入 `.alter0/workspaces/sessions/<session_id>/attachments/<asset_id>/`，随后消息请求、最近会话恢复与页面重开都只保留 `asset_url / preview_url` 引用；assistant 最终回复里的外链 markdown 图片也会在返回前下载进同一目录并改写成本地附件 URL，避免会话历史长期依赖远端外链或把原始大图 `data_url` 堆进浏览器存储。
- `Chat / Agent Runtime` 首页 Composer 收敛为双层紧凑输入面：上层为单一主输入区，下层工具栏左侧提供正方形低圆角的会话设置与附件按钮，右侧提供紧凑 icon submit；桌面与移动端都压缩外层留白、输入高度和发送按钮体量，同时保持主输入区满宽铺开、具备足够横向留白和更高的可读输入面；发送按钮直接复用 Terminal 的紧凑 icon submit 皮肤；PC 端上传、发送、状态、详情和短标识控件都采用低圆角矩形，输入区、底部工具栏、会话卡片和 `Details` 面板沿同一套浅色 terminal-runtime 皮肤出图，不再混用默认 terminal footer slab 与旧式轻表单观感；空态工作区不允许保留可拖拽滚动，把头部和输入区顶离可视区。
- `Chat / Agent Runtime` Composer 支持最多 5 张图片附件：前端按会话草稿缓存附件、提供缩略图预览与移除操作，用户消息时间线与最近会话恢复仅保留图片预览资产，不再重复持久化原始大图 payload；助手消息中的 markdown 图片会直接以内联图片方式懒加载显示。仅支持视觉输入的模型允许发送带图消息；图片请求不会切到异步 Task，也不会在模型链失败后静默降级到 Codex 文本执行。
- `Terminal` 页面 Composer 支持最多 5 个附件：图片继续提供缩略图预览与移除，常见文本/文档文件以文件条目展示；选中文件后统一先写入 `.alter0/workspaces/sessions/<session_id>/attachments/<asset_id>/`，提交时仅发送稳定附件引用。图片继续映射为 Codex CLI `-i` 输入；普通文件会同步写入当前 Terminal 工作区 `input-attachments/<turn_id>/`，并在同轮 prompt 中注入可直接读取的 workspace 相对路径，供 Codex 按需读盘。`Tasks` 详情抽屉里的 follow-up terminal 输入当前稳定支持图片附件，并继续沿统一消息元数据透传。
- 移动端工作台优先保证输入、抽屉和滚动流畅度：`Chat / Agent Runtime / Terminal` 的移动表面在窄屏下不再依赖大面积 `backdrop-filter` 或持续背景光晕动效，运行页容器、抽屉遮罩、抽屉面板本体和主工作区统一回落到静态浅色 surface，避免真机滚动和抽屉切换出现卡顿。
- 移动端运行页的 `Menu` 与 `Sessions` 抽屉统一使用同一套面板开合语义：两者始终互斥，打开其一时立即收起另一侧；点击遮罩、切换路由、切换会话或创建新会话后，不保留旧的抽屉覆盖层。
- 移动端运行页的 `Menu / Sessions` 抽屉在真机上优先保证稳定性：遮罩保留淡入淡出，抽屉本体仅保留一层轻量侧滑，不再叠加容易闪烁的多层位移、淡出或条目级顺序动画；抽屉内会话项按最近时间分组展示，统一收敛为状态文本、标题、摘要、底部短 hash 与尾侧删除动作。
- `/chat`、登录页和主工作区品牌文案对外统一展示为 `Alter0`，浏览器标题、登录标题、导航品牌位、会话栏标题与欢迎区 tag 不再混用小写服务名。
- `Chat / Agent Runtime` 的会话操作、模型选择、Agent 选择、Tools / MCP 与 Skills 都收敛到工作台内部；两条运行页的 `Details > Model` 都额外内置 `Codex` 直选项，选中后后续消息会直接走 `Codex CLI` 执行链而不是普通 LLM Provider；Agent Runtime 的 Agent 选择列表不展示 `Alter0/main` 主助手，仅保留专项内置 Agent 与用户管理 Agent；Agent Runtime 的 `Details > Skills` 会把当前 Agent 私有 Skill 作为已启用且不可取消的固定项展示，剩余可勾选项只展示公有 Skill，例如 `travel` 的旅游领域规则属于私有 Skill，`deploy-test-service` 这类基础部署能力属于公有 Skill；窄屏下主导航仍走抽屉，小高度视口中导航分组、底部设置项与语言切换入口保持独立纵向滚动并全部可触达。
- 窄屏主导航抽屉在点击路由项后会立即收起；切页后不保留覆盖在新页面上的菜单层，用户直接进入目标页内容区。
- 运行页内的 Session 列表保持工作台式紧凑结构：按最近时间分组，并与主导航 `menu` 复用同一套分组外壳、hover 与激活态语言；条目采用导航式线性排布，标题独立占一行并在可用宽度内单行截断，摘要独立占一行，短 hash 固定在条目下缘，删除动作作为尾侧轻量文本操作存在，不再把正文拆成额外 footer 卡片。
- Runtime 配置统一通过 workspace `Details` 面板切换，不再使用独立 bridge sheet；`Details` 默认先展示高密度摘要区，字段标签、复制按钮和多行内容按统一紧凑规格排列，并以顶层浮层方式覆盖在运行页上方，打开时不再推动消息区或对话框位置；浮层最大可视区域保持克制，内部 tab/按钮支持再次点击只收起当前配置内容且保留 `Details` 面板，点击浮层外区域或按 `Escape` 才关闭整个面板，移动端仍要求面板与输入区互不遮挡，切换时优先保证输入焦点、键盘占位和主动作可达。
- `Agent` 选项卡片在会话设置中使用短摘要展示：优先显示 Agent description，并限制在简短可扫读的卡片文案内；完整 system prompt 不直接出现在选择面板里。
- `Agent` 运行页的会话侧栏为每个会话展示 8 位短 hash 标识，便于在预览地址、排障记录和跨端沟通中引用同一会话。
- 会话设置中连续勾选 `Skill / Tool / MCP` 时，当前滚动位置需保持稳定，不能在每次勾选后跳回顶部。
- `Chat` 会话设置面板中的标题、说明与右侧标签在窄宽度下需保持可读：主标题按可用宽度截断，说明文案允许换行，避免发生重叠或互相覆盖。
- Web 前端所有时间展示统一使用北京时间（`Asia/Shanghai`）与 24 小时制；Chat、Agent、Terminal、Task、Cron 等页面不再跟随浏览器本地时区漂移，Cron 表单默认时区固定为 `Asia/Shanghai`。
- 移动端 `Chat / Agent` 输入区在软键盘弹起、收起与可视视口高度变化期间，会基于 `VisualViewport` 同步有效视口高度；App Shell 在键盘弹起时保持稳定基线高度，仅由输入区消费 `--keyboard-offset` 贴住可见底部，避免浏览器工具栏状态切换或输入聚焦把整个工作区一并顶起；仅在输入框实际聚焦且软键盘占位达到阈值时才追加键盘底部偏移，浏览器工具栏伸缩或键盘收起后不保留额外底部留白。
- 移动端 `Chat / Agent Runtime` 的输入框首次触摸需与 `Terminal` 使用同一条聚焦链路：通过 `preventScroll` 聚焦 textarea，并在聚焦期间监听 `window.scroll + VisualViewport resize/scroll` 把页面锚定在 `scrollY = 0`，避免首次弹出键盘时把公共操作行顶出可视区或让页面分辨率/可视区域出现跳变。
- 移动端 `page-mode` 路由页会同步消费 `VisualViewport` 高度；`Terminal` 与其他信息页在浏览器底部工具栏伸缩、软键盘收起或可视视口回弹后，页面底边需立即回贴可见视口，不保留额外底部空白。
- 移动端 `Terminal` 在输入框聚焦且软键盘抬起后，底部 Composer 会按 `VisualViewport` 推导出的 `--keyboard-offset` 直接上移到可见底边；Terminal 工作区主体保持原位，键盘弹起不会把页面整体向上推出；长历史输出继续留在 `terminal-chat-screen` 内独立滚动，不允许通过增大 footer padding 把输入区整体挤出屏幕。
- 四键阅读定位条仅保留在 `Agent` 运行态消息区与 `Terminal` 对话区，采用与主仓库 Terminal 一致的右侧箭头四键：`回到顶部 / 上一条 / 下一条 / 回到底部`。普通 `Chat` 消息区与其他 `page-mode` 路由页不再展示这组按钮；滚动超出阈值后显示顶部与底部入口，并按当前视口中的可见消息块或 Terminal turn 动态计算上下目标。
- 移动端 `Chat / Agent` 的后台任务轮询会按页面可见性自动降频；页面隐藏时停止高频扫描，恢复前台后再立即补一次刷新，降低持续耗电与发热。
- `Provider / Model`、`Tools / MCP`、`Skills` 可在会话过程中继续调整，并作用于后续发送的消息。
- `Alter0` 默认使用 ReAct 执行链，可直接完成通用任务，并在需要时调度专项 Agent。
- 选中的 Agent 工具会在模型调用时作为 function tools 注入；当前稳定工具集收敛为 `codex_exec`、`search_memory`、`read_memory`、`write_memory`，以及仅对可委派 Agent 暴露的 `delegate_agent`。
- `Models` 控制面支持同时维护 `OpenAI Compatible` 与 `OpenRouter` Provider；`OpenRouter` 可直接配置 `Site URL`、`App Name`、回退模型和 Provider 路由偏好，系统会分别注入官方请求头与请求体扩展字段。
- `OpenAI Compatible` / `OpenRouter` Provider 均支持按 `api_type` 选择上游接口：`openai-responses` 走 `/responses`，`openai-completions` 走 `/chat/completions`；配置自定义 `base_url` 时，需要目标服务兼容所选接口。`OpenRouter` 默认使用 `https://openrouter.ai/api/v1` 与 `openai-completions`。
- 当 Agent / ReAct 在 `openai-completions` 路径下进入多轮工具调用时，运行时会保留 assistant `tool_calls` 与后续 `tool_call_id` 的完整关联，保证兼容要求严格校验工具消息顺序的 Provider。
- `Models` 控制面保存 Provider 时，`api_key` 输入框留空表示保持现有密钥；若前端中间态传入占位值 `-`，服务端会按空值处理，不会把 `-` 持久化为真实凭据。
- 历史 `model_config.json` 若残留缺失 `api_key` 的 Provider，加载阶段会自动收敛为禁用态并保留在 `Models` 控制面中，页面不会因旧配置直接返回 500；补齐密钥后可重新启用。
- `Codex Accounts` 控制面位于 `Settings`，支持在同页查看紧凑化的运行时概览、当前 Codex 运行时状态、当前活动 profile、活动 model、思考深度、导入已有 `auth.json`、启动独立 `codex login` 会话、查看当前账号套餐/额度状态，并切换当前运行时生效账号；概览区采用“当前账号主身份区 + 套餐/小时额度/周额度/托管数量”紧凑指标列，其中小时/周额度以进度条展示剩余额度并附带 reset 时间，key/value 默认同列对齐，概览不再暴露活动 auth 路径，维护类信息统一通过 `Runtime Details` 折叠区展开；`Current Codex` 仅保留一套可编辑的 model / 思考深度字段，不再在选择器下重复展示当前值；托管账号区采用高密度行式列表，持续暴露账号身份、计划、额度进度条、reset 时间与切换入口，并用更平的控制台按钮/状态文本和分隔线式布局替代层层胶囊与内嵌方框；model 与思考深度的可选项来自 Codex app-server 的 `model/list`，当前生效值与来源来自 `config/read`，保存时通过 `config/batchWrite` 写回当前用户配置。当前运行中的 `auth.json` 若尚未纳入托管，会在概览与空态中明确标记为“已生效但未托管”；只要当前 live 账号成功拿到 quota，概览区仍直接展示该 live 账号的套餐与额度，加载阶段则保留完整面板骨架而不是退化成单行提示。
- 默认 Provider 只会落在已启用配置上；若默认 Provider 被禁用、删除或历史配置已失效，系统会自动切换到下一可用 Provider，无可用项时清空默认值。
- 复杂度评估阶段会优先复用当前消息选中的 `Provider / Model`；未显式选择时，回退到默认 Provider 与默认模型。若 Chat 或 Agent Runtime 当前显式选择 `Codex`，前端会改写消息 metadata 为 `alter0.execution.engine=codex`，由执行层直接进入 `Codex CLI` 链路。
- 默认走实时执行。
- 流式对话会先直接启动回复；复杂度评估与回复并行进行。
- `Chat / Agent` 消息区在流式增量、Agent `Process` 展开收起与任务状态回填期间采用逐条 patch，并把高频刷新合并到浏览器逐帧节奏，避免长会话中反复整段重建消息列表。
- Agent SSE 在工具循环期间会优先推送结构化 `process` 事件，按步骤实时更新 `Process` 面板；最终正文继续通过输出事件与 `done` 结果收口。
- 流式连接在已收到正文后若中途断开，前端保留已到达的正文并把该条消息收敛为失败态，避免整条消息被统一覆盖成空白错误文案。
- 当请求复杂度较高且仍在执行中时，系统会中途转为后台 `Task` 执行，并先返回一条任务说明消息，包含任务目标、执行计划与任务入口。
- 若当前消息已进入 Agent 执行链，前端页面切换、标签页隐藏、SSE 断开或浏览器主动取消请求都不会中断后端执行；连接只负责回传，最终结果仍会落到会话历史。
- 浏览器本地缓存里的历史消息若残留 `streaming` 状态，页面恢复时会自动收敛为失败态或任务态，不再把旧消息长期停留在 `In Progress`。
- 若 Agent 流式连接在没有可用正文时中断，前端失败文案会明确提示刷新；刷新页面后，当前活动会话会优先从服务端会话历史恢复已成功写入的最终回复，而不是继续停留在本地失败态。
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
- `travel`：负责旅游任务的专项执行、HTML 攻略产物和子域名交付
- Web `Chat` 页面默认绑定 `Alter0`；`Agent` 页面提供统一 Agent 运行入口，用于承载未占用独立前端入口的专项内置 Agent 与用户管理 Agent，主助手 `Alter0/main` 不作为 Agent Runtime 可选项出现。
- Agent 的职责收敛为“作为用户的持续助手并驱动执行”，而不是直接自己操作仓库或 Shell。稳定工具面以 `codex_exec` 为具体执行入口，`Alter0/main` 与其他允许委派的 Agent 可额外使用 `delegate_agent`，所有 Agent 可在已注入的记忆文件范围内使用 `search_memory`、`read_memory`、`write_memory` 维护长期偏好、缩写映射与稳定协作约束；`coding` 与 `travel` 还会启用 `deploy_test_service`，用于把当前 Session 工作区发布到共享运行时的短哈希网关。`search_memory` 用于按关键字跨记忆文件定位历史信息，再决定是否精读或更新具体文件。Agent 负责吸收用户意图、会话上下文、Agent 规则与记忆，并把它们转译成当前这一步的精确执行指令；这些 Agent 侧 prompt 与编排规则默认留在 Agent 自身，不直接透传给 Codex。当前稳定支持两种 Codex 执行策略：存在可用 LLM Provider 且进入 Agent / ReAct 链路时，Agent 负责吸收 Skill、MCP、Memory 与运行时上下文，只向 Codex 下发当前步骤的纯执行指令；不存在 Provider、Agent 初始化失败或请求直接进入 Terminal / 直连 Codex 链路时，运行时会为该会话生成独立的 `CODEX_HOME`、原生 `config.toml`、工作区 `AGENTS.md` 与 `.alter0/codex-runtime/*` 文件，把 MCP、Skill、Memory 与运行时信息编译成 Codex 原生运行配置与工作区事实。`coding` Agent 的仓库类执行会优先落到当前 Session 独立的 repo 完整 clone：`.alter0/workspaces/sessions/<session_id>/repo`，该目录自带自己的 `.git` 元数据，不依赖 `git worktree`；运行时会把当前仓库远端地址、源仓库路径、独立 repo clone 路径、当前分支、会话工作区、短哈希预览域名与 PR 交付要求写入 Agent 上下文或直连 Codex 的 Native Runtime 资产。测试仓库的关键配置需要与正式服务保持一致，包括 model provider、Codex 执行路径、agent 路径等，只有会话缓存、会话历史等 session 级数据允许不同。涉及测试页面或独立测试服务时，`coding` 与 `travel` 都需要通过 `/api/control/workspace-services/{session_id}/{service_id}` 或脚本 `scripts/deploy_test_service.sh` 把当前 Session 工作区注册到共享运行时的动态网关；默认 `web` 服务映射到 `https://<session_short_hash>.alter0.cn`，共享运行时会按注册的启动命令托管当前分支后端，再让该子域名直连这份后端；附加服务默认映射到 `https://<service>.<session_short_hash>.alter0.cn`。其中 `travel` 服务固定映射到 `https://<session_short_hash>.travel.alter0.cn`，并且该 host 只读、免登录。只有显式指定 `service_type=frontend_dist` 时，才会发布纯静态前端预览。`travel` 在正常对话结果之外，还要求额外产出一份 HTML 旅游攻略，并发布到当前 Session 的公开 travel 子域名。部署成功后先由用户决定是否进行手动测试，再继续后续交付闭环。
- 预览短哈希 host 与主域工作台共用同一套登录保护；访问 `https://<session_short_hash>.alter0.cn` 时可直接打开该 host 自身的 `/login` 登录页，登录 cookie 会共享到 `*.alter0.cn`，默认 `web` 全栈预览内部启动的 runtime child 不再额外启自己的第二层登录，避免主域与预览 host 各自重复登录。
- 若 Agent 在 `max_iterations` 耗尽前仍未显式 `complete`，运行时会返回带有“达到迭代上限”说明和最后一次工具观察的最终答复，避免 Web 流式消息在 `codex_exec` 观察后空收口。
- Agent 执行过程会在运行时产出结构化 `process_steps`，并通过实时 `process` SSE 事件、`done` 结果、Task 结果与会话历史一并返回；前端优先按结构化步骤渲染可折叠 `Process` 区块，仅对历史旧消息保留基于文本标记的兼容解析。
- Agent 流式回复中的 `Process` 与最终正文在收口后继续同时保留；刷新页面或从服务端会话历史恢复时，结构化步骤不会因最终正文已落库而丢失。
- Agent 请求一旦进入后端执行链，浏览器侧任何交互事件都只影响当前连接状态，不影响 Agent 本身的执行与会话持久化；断开后重新进入历史即可查看最终结果。
- 每个 Agent 可独立配置名称、system prompt、tool 白名单、公有 Skill 选择、MCP 选择与 Memory Files 选择。
- Agent 可按 Profile 勾选 `USER.md`、`SOUL.md`、`AGENTS.md`、长期 `MEMORY.md / memory.md`、当天与前一天 Daily Memory；其中 `AGENTS.md` 固定解析为当前 `agent_id` 的私有文件 `.alter0/agents/<agent_id>/AGENTS.md`，不与其他 Agent 共享；`USER.md`、`SOUL.md`、长期/日记忆继续作为共享记忆注入执行链路。
- 所有 Agent 还会自动携带一个私有、可维护的 file-backed Skill，默认文件为 `.alter0/agents/<agent_id>/SKILL.md`；它用于沉淀该 Agent 的可复用工作模式、输出结构、检查清单与稳定偏好，不需要在 Agent Profile 里手工勾选，也不会在 Agent Runtime 的 Skill 面板里被用户取消。
- `coding` Agent 的私有 `SKILL.md` 会进一步沉淀 Git 交付规则，例如 Session repo 完整 clone、认证签名提交、测试页部署成功后的快速 `gh` PR/merge 流程与交付汇报口径。
- `AGENTS.md` 与私有 `SKILL.md` 职责分离：`AGENTS.md` 负责当前 Agent 的协作边界、仓库/工作区操作规则与交付约束；`SKILL.md` 负责该 Agent 自身的可复用打法、模板和长期偏好。一次性任务细节仍应留在当前任务或记忆文件，不写入 Skill。
- 所有 Agent 会额外自动维护当前 Session 的私有画像文件 `.alter0/agents/<agent_id>/sessions/<session_id>.md`，并以只读 `Agent Session Profile` 注入执行链路；该文件用于沉淀当前 Agent 在该 Session 内的稳定工作上下文与结构化实例属性。实例属性支持通过请求 metadata 增量写入，统一落在 `## Instance Attributes` 区块，适合承载 `travel` 场景的 `city / district` 一类会话态事实；`coding` 场景还会自动补齐 `repository_path / remote_repository / branch / base_branch / preview_url / preview_subdomain` 等仓库与预览属性。每个 Agent 还可预设自己的 Session Profile 字段定义，内置 `coding`、`writing`、`travel` 已分别补齐仓库、写作和城市行程相关字段。执行前运行时会先走一条旁路 `Session Profile` 抽取链路：优先基于字段 schema 从本轮自然语言提取可写字段，再把 patch 合并回 profile；默认实现使用受限 Codex 窄调用作为 fallback，而不是复用主 Agent 的长上下文会话。
- Agent 与 Chat 共用会话短期记忆链路：执行前优先使用进程内最近轮次；当会话因重启、跨天续聊或 TTL 淘汰导致内存窗口为空时，会从持久化 session history 回填最近完成轮次，再注入当前 prompt，保证 UI 可见的同一 Session 历史也能被模型用于指代解析和执行结果回顾。
- Memory Files 解析后会基于本轮用户输入执行轻量自动召回，命中的片段以 `memory_context.recall[]` 注入 Agent prompt 与 `codex_exec` 结构化载荷；`search_memory`、`read_memory`、`write_memory` 继续用于二次检索、精读和受控写入。
- 注入内容同时携带可写文件路径；Agent 只能在这些已解析出的记忆文件上使用 `search_memory`、`read_memory`、`write_memory` 做检索与持久化维护，其余具体文件与命令操作统一交给 `codex_exec`。
- `codex_exec` 调用 `Codex CLI` 时通过 stdin 传递当前步骤的执行指令，命令行参数仅保留 `-` 作为 prompt 占位；Agent / ReAct 路径不再额外向 Codex 透传 Skill、MCP 或 Memory 结构化载荷，直连 Codex 路径则改为使用原生 `CODEX_HOME/config.toml`、工作区 `AGENTS.md` 与 `.alter0/codex-runtime/*` 承载这些运行时信息。
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
- Terminal Composer 支持最多 5 个附件。图片附件继续保留缩略图预览、草稿缓存与 `asset_url / preview_url` 提交语义；常见文本/文档文件改为文件条目展示并复用同一附件上传接口，只提交稳定 `asset_url` 引用。发送时，图片会继续走 Codex CLI `-i` 输入，普通文件则写入会话工作区 `input-attachments/<turn_id>/` 并通过同轮 prompt 告知可读取路径；纯附件输入会自动补齐稳定占位文本。
- Terminal 的 `Details` 面板支持选择控制面中启用且非私有的公有 Skill，选择结果随下一次 `/api/terminal/sessions/{id}/input` 请求以 `skill_ids` 发送；后端会把选中的 Skill 编译到当前 Terminal 工作区的 `.alter0/codex-runtime/skills.md` 和托管 `AGENTS.md` 指令块中，仅作用于后续 Terminal 输入。
- 同一 Terminal 会话在单次运行态中断或退出后，只记录一条对应状态提醒；恢复后若再次发生新的中断或退出，再按新的状态周期补充提醒。
- Terminal 输入区上缘的运行态 hint 只服务于当前空闲会话；一旦用户重新发送恢复当前会话，或从旧会话切到 `New` 待创建态，旧的 `Exited / Interrupted / Failed` 提示会立即清空，不再在发送中残留。
- Terminal 工作区头部仅保留 `Details` 等阅读辅助工具；会话删除统一从会话列表触发，`Delete` 会移除会话记录、持久化状态文件与该会话对应的独立工作区。
- Terminal 会话侧栏复用共享会话列表项，当前态、标题摘要、底部短 hash 与 `Delete` 动作在长标题、双行元信息与中英文混排场景下保持统一行距和尾侧对齐，不因标题长度变化产生错位。
- 当前正在查看旧会话时点击 `New`，前端会先切入一个干净的待创建会话态；创建请求完成前不再沿用旧会话的 `Interrupted / Exited / Failed` 提示文案，也不继承旧会话残留的底部键盘留白。
- 同一 Web 登录态下，手机与 PC 访问同一批 Terminal 会话历史；刷新或跨端切换后不再因设备标识不同而看到不同会话列表。
- Terminal 不再设置产品级会话数量上限或固定超时淘汰策略。
- 访问 Terminal 时，轮询刷新不会重建已聚焦输入框；移动端输入法每次确认词句后，若输入框仍保持聚焦，页面继续延迟重绘并保持当前位置，直到失焦后再刷新视图；桌面端在连续输入窗口内也会暂缓非必要工作区重绘，待输入停顿后自动补齐刷新。
- Terminal 轮询刷新采用会话列表与工作区局部更新；当用户正在滚动输出区时，前端保留原消息滚动容器与滚动位置，不再按周期整块重建终端视图。
- Terminal 轮询刷新按会话状态自适配：`busy` 会话继续保留实时刷新，但用户正在滚动输出区时暂停明细刷新；`ready` 会话改为低频、列表级刷新，不再在阅读过程中频繁重刷当前会话详情，避免滚动被打断。
- Terminal 滚动状态同步会合并到浏览器逐帧刷新节奏内执行；上一条 / 下一条定位所需的 turn 位置在视图结构稳定时复用缓存，避免在连续滑动中反复全量测量消息区。
- Terminal 浏览器侧会话缓存采用滚动感知的延后持久化；输出持续增长时优先让出主线程给滚动与渲染，再在滚动停顿后写入本地存储。
- Terminal 会按页面可见性、输入聚焦与滚动活跃度自动调整轮询频率；活跃阅读或输入期间降频，页面隐藏后进一步延长刷新周期，恢复前台后再回到正常刷新节奏。
- Chat 与各路由消息流统一采用克制的冷灰工作台阅读主题：用户消息保留右对齐并限制在消息区宽度的 80% 内，消息气泡、流程卡与正文区统一使用低对比边框、近白表面和有限强调色；正文排版优先于装饰层级，用户消息与其后续回复继续保持更紧凑的同轮分组间距。
- Chat 助手消息尾部默认只保留时间；仅在回复仍处于生成、排队或失败等需要即时反馈的状态下显示紧凑状态标签，不再为已完成消息追加 route/source/status 标签。
- Chat 与 Agent Runtime 工作区头部固定为共享单行 header：只保留会话标题、状态按钮和 `Details` 入口，不再在头部直接放置 `Model / Tools / MCP / Agent` 控件，也不重复展示 `Chat / Agent` 标签与目标摘要；运行时配置、会话元数据和页面专属详情统一放入 `Details` 面板，面板首屏先以紧凑摘要栅格提高信息密度，再承接页面专属交互区。Agent Runtime 在 `Details` 中额外提供 `Session Profile` 区块，直接展示当前 Agent 预设字段及当前 Session 实例值。
- Chat 桌面宽屏会按可用主工作区宽度自适应扩展消息列与底部输入区，并统一收敛到居中的 `960px` 阅读宽度上限，避免正文无限拉长。
- Chat 移动端输入区默认隐藏装饰性附注与字数计数，底部只保留输入框，以及同排的会话设置、附件与发送按钮；会话设置以独立底部面板承载 `Agent / Model / Tools / Skills` 标签，统一使用当前语言口径，避免中英混排，并在小屏与软键盘场景下维持完整可读与可滚动操作区。
- Chat、Agent Runtime 与 Terminal 在移动端都会按当前 fixed composer 的实际遮挡高度回收消息视口；输入区贴底期间，消息列表、空态说明和长输出阅读都必须停在 composer 上沿，不允许再被底部输入框盖住。
- Chat、Agent Runtime 与 Terminal 在软键盘收起、输入框失焦和 composer 回弹到底边的过程中，消息视口与跳转控件也要同步释放旧的底部占位；页面上不能残留上一轮键盘高度对应的空白带或悬空控件。
- Chat、Agent Runtime 与 Terminal 在移动端键盘弹起和收回期间，只有 composer 自身允许跟随 `VisualViewport` 做贴底动画；顶部 `Menu / Sessions / New` 操作行、紧凑 workspace header 和 Terminal 右侧四键定位条都保持原位，不跟着键盘位移一起跳动。
- Chat 与 Terminal 的移动端发送按钮在软键盘打开期间支持直接点按提交；首触发送会立即命中当前提交逻辑，不需要先收起键盘再补第二次点击。
- 运行页 composer 跟随键盘回弹时不再叠加额外的 `bottom` 补间过渡；可视视口收起后输入区会直接回贴底边，避免回弹拖滞或明显卡顿。
- 输入框失焦后，如果 `VisualViewport` 还没恢复到最终高度，页面会继续保留上一段键盘偏移并随视口回弹逐步释放，不会先闪回到底边再被下一帧重新顶起。
- Web Shell 主工作区的首屏内容保持紧凑起始区：桌面与中宽度下的 `Chat` 空态欢迎区会在 header 与 composer 之间沿主工作区中轴做竖向居中，欢迎 tag、标题、描述与 prompt 统一围绕欢迎区中线排布；真窄屏继续贴近头部下沿起排，输入区不再依赖大块弹性留白把首屏内容拉散。普通 `page-mode` 路由页 `route-head` 与 `Terminal` 工作区继续沿用“两行头部 + 贴顶正文起始区”基线，而 `Chat` 与 `Agent Runtime` 空态在中窄屏都复用 terminal-style 顶部操作行与单行紧凑 workspace header：顶部直接显示 `Menu / Sessions / New`，workspace header 单行显示当前会话标题、状态按钮与 `Details` 入口，不再展示冗余摘要文案。
- Terminal 路由直接进入工作区，不再在工作区上方额外挂载页面级说明 hero；运行区根节点直接挂在共享 `workbench-pane-shell` 下，不再额外经过 `route-view / route-body` 包裹，从 `Chat / Agent Runtime` 切到 `Terminal` 时保持相同的 runtime workspace 骨架，避免首屏布局与滚动容器发生跳变。会话栏、工作区容器、工作区头部和窄屏顶部操作行与 `Chat / Agent Runtime` 复用同一套工作台表面语义与节奏；Terminal 只在终端路由内补充会话状态、jump controls、step 详情和 composer 皮肤等变体。顶部通用栏直接提供 `Menu / Sessions / New`，工作区头部收敛为会话标题、状态按钮与 `Details` 工具栏；`Details` 首屏同样先展示紧凑摘要栅格，再承接终端会话字段；发送区使用自适应输入框与紧凑发送按钮，运行态退出或中断提示以内嵌状态条贴合输入区上缘展示；消息与 `Process` 头部保持自然文档流滚动，不再启用吸顶导航；阅读定位由右侧低圆角四键组承担，支持回到顶部、上一条、下一条与回到底部，并统一为浅色低对比滚动条与阅读主题；软键盘收起或浏览器底部工具栏回弹后，底部输入条需立即回贴可见底边，不保留额外占位空白。
- Terminal 移动端的右侧四键定位条只按静态 composer footprint 停靠，不跟随软键盘位移一起上移；键盘弹起时按钮组保持原位，键盘收起后继续稳定回到输入区上沿之上，不在底边残留半截控件。
- Terminal 发送按钮在首次点击时必须立即进入 `Sending...` 禁用态；若当前还没有 terminal session，前端先创建会话再继续提交输入，但首击期间不能保留可重复点击的静止按钮，避免用户误判为“第一次点击无效”。
- Terminal 在窄屏 `page-mode` 下继续由 `terminal-chat-screen` 独立承担消息区纵向滚动；外层 `workbench-main / chat-pane / terminal-view` 只负责提供满高约束与滚动隔离，不得吞掉消息页滚动手势。
- Terminal 移动端的 `Sessions` 抽屉与主 `Menu` 保持同一份面板状态；从工作区头部或顶部操作行打开会话列表时，会自动替换主导航抽屉，而返回 `Menu` 时也会立即收起会话列表，不出现双层覆盖或残留展开态。
- Terminal 的会话抽屉与 Chat / Agent Runtime 共享同一套列表视觉语义：条目头部展示当前态 badge，再展示标题、最近输出和会话短标识，删除入口固定留在卡片尾侧；会话运行状态保留在 workspace header 与 `Details` 摘要中，不在列表项内额外渲染独立状态徽标。
- Terminal 窄屏工作区头部不再重复渲染第二枚 `Sessions` 按钮；会话抽屉入口统一由壳层头部承接，工作区操作栏只保留状态按钮与 `Details` 等当前会话阅读动作。
- Terminal 工作区头部在真手机宽度下允许标题、状态按钮与 `Details` 工具栏自适应换行；长标题优先保留可读性，不允许把操作按钮挤出可见宽度。
- Terminal `Process` 步骤列表保持单行摘要阅读：步骤标题在可用宽度内自动截断，时间与状态固定收在右侧独立区域，不与标题文本重叠。
- Terminal 中四键导航会跟随当前视口里的可见 turn 重新计算上下目标；`Process` 折叠或展开后，上一条与下一条状态会随重排结果同步更新。
- Terminal 的整体视觉语言与 Chat 收敛：会话侧栏、工作区容器、输入区与输出块统一使用低对比边框、近白表面、有限强调色和更克制的圆角节奏；用户输入不再额外展示命令前缀符号或强调色，最终回复直接按 Chat 助手消息样式渲染；`Process` 保留低对比虚线提示区和左侧纵向时间线，内部步骤压缩为单行摘要，展开后只展示详细内容，不再重复状态标签。
- Terminal 桌面端继续沿用旧版 master-detail 布局关系作为阅读基线：左侧会话列表显示当前态、标题、最近输出与短标识，右侧工作区头部收敛为会话标题、状态按钮与 `Details` 工具栏；这套布局由 React 组件直接渲染，不再依赖旧版脚本控制。
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

## Travel Agent

`travel` 是系统内置 Agent。

1. `travel` 的显示名称为 `Travel Agent`。
2. `travel` 负责旅游任务的专项执行与结果收口，沿用统一的 “Agent 协助编排，Codex CLI 负责具体执行” 模型。
3. 旅游领域的稳定规则继续沉淀在私有 file-backed Skill `.alter0/agents/travel/SKILL.md`，用于约束城市行程、地铁、美食与地图输出结构。
4. `travel` 作为 Agent Runtime 可选入口直接出现在 `/api/agents` 列表中，支持用户显式切换到旅游专项执行链路。
5. `travel` 在正常对话之外，还要求额外生成一份 HTML 格式的旅游攻略，并把它发布到当前 Session 的公开只读子域名 `https://<session_short_hash>.travel.alter0.cn`；该域名不需要登录。

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
4. Chat / Agent 执行链交给 `Codex CLI` 的最终指令通过 stdin 输入，避免不同操作系统在长上下文场景触发命令行长度限制；其中 Agent / ReAct 路径只传递当前步骤指令，直连 Codex 路径会为当前会话额外准备独立 `codex-home/`、原生 `config.toml`、工作区 `AGENTS.md` 与 `.alter0/codex-runtime/*`。

其中 `Chat` 再细分为两种执行方式：

1. `Sync`
- `POST /api/messages`：普通 JSON 一次性返回结果。
- `POST /api/messages/stream`：通过 SSE 流式返回 `start / delta / done` 事件；复杂度评估与回复并行进行，若请求在评估完成时仍需长耗时执行，会在同一条流中切换为异步任务并返回任务受理结果。
- `Chat / Agent` 的 SSE 流在长时间无增量输出时会持续发送保活帧，避免 Codex CLI 静默执行阶段被浏览器或代理提前断开连接。
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

`Codex Accounts` 默认把托管账号、切换备份与登录会话目录写到当前活动 `CODEX_HOME` 下的 `alter0-accounts/`；未显式设置 `CODEX_HOME` 时，对应目录即 `$HOME/.codex/alter0-accounts/`。切换账号只替换当前活动 `auth.json`；当前 Codex 管理区通过 Codex app-server 读取真实运行时能力与配置来源，并通过用户配置写接口更新当前活动配置中的 `model` 与 `model_reasoning_effort`。

若服务需要自行提交签名 commit、创建 PR 或执行合并，还需在 root 下额外执行一次：

```bash
sudo ./scripts/setup_alter0_runtime_auth.sh
```

该脚本会把 `alter0` 运行账户的 GitHub App token helper、`gh` 命令包装器、SSH signing key 与全局 Git 配置初始化到 `/var/lib/alter0`，用于服务内 `Codex CLI` 的提交 / PR / merge 链路。

若希望服务内 `Codex CLI` 可直接执行 `internal/interfaces/web/frontend` 下的 `npm run build` / `npm run test`，以及 `internal/interfaces/web` 下的 `npm run test:e2e`、`npx playwright install chromium` 等 Node/Playwright 测试链路，还需在 root 下额外执行一次：

```bash
sudo ./scripts/setup_alter0_runtime_node.sh
```

该脚本会把带 `npm`/`npx`/`corepack` 的 Node 运行时安装到 `/var/lib/alter0/.local`，并默认在 `internal/interfaces/web` 与 `internal/interfaces/web/frontend` 目录预装 `npm ci`，随后安装 Playwright Chromium 浏览器，使服务运行账户在非交互式环境中也能同时执行前端构建、单测与 E2E 测试。

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

对应 Nginx 配置与运行权限方案见：`docs/deployment/nginx.md`。若需要会话级预览或独立测试服务，请把 `alter0.cn` 与 `*.alter0.cn` 一并反向代理到同一共享运行时，再用 `scripts/deploy_test_service.sh <session_id> [service_name] ...` 注册当前会话服务。默认 `web` 会构建前端，并把当前分支后端启动命令注册给共享运行时托管，再把短哈希子域名注册为 `http` 反代；如只需要静态 UI 预览，可显式传 `--service-type frontend_dist`。

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
- 若 Codex CLI 在线程续写阶段返回远端 compact 失败，Terminal 会保留原会话历史与工作区，但清空失效线程标识；下一次输入会在同一 Terminal 会话下自动启动新的 Codex 线程继续执行
- 对已退出或已中断的 Terminal 会话重新发送输入后，输入区上的旧运行态提示会立即让位给当前发送态，不继续显示“会话已退出”之类的过期提示
- `DELETE /api/terminal/sessions/{id}` 用于直接删除 Terminal 会话，并同步清理 `.alter0/state/terminal/sessions/{id}.json`，接口返回 `204 No Content`
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

### Codex Accounts

```bash
# 列出托管账号与当前活动账号
curl http://127.0.0.1:18088/api/control/codex/accounts

# 导入 auth.json
curl -X POST http://127.0.0.1:18088/api/control/codex/accounts \
  -H "Content-Type: application/json" \
  -d '{"name":"work","auth_file_content":"{\"auth_mode\":\"apikey\",\"OPENAI_API_KEY\":\"sk-***\"}"}'

# 启动独立登录会话
curl -X POST http://127.0.0.1:18088/api/control/codex/accounts/login-sessions \
  -H "Content-Type: application/json" \
  -d '{"name":"fresh-account"}'

# 查询登录会话状态
curl http://127.0.0.1:18088/api/control/codex/accounts/login-sessions/login-123

# 切换当前运行时账号
curl -X POST http://127.0.0.1:18088/api/control/codex/accounts/work/switch

# 查看当前 Codex 运行时
curl http://127.0.0.1:18088/api/control/codex/runtime

# 更新当前 Codex model 与思考深度
curl -X PUT http://127.0.0.1:18088/api/control/codex/runtime \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-5.4","reasoning_effort":"high"}'
```

说明：

1. 托管账号默认存储在当前活动 `CODEX_HOME` 下的 `alter0-accounts/` 目录；未设置 `CODEX_HOME` 时，即 `$HOME/.codex/alter0-accounts/`。
2. 独立登录会话会使用隔离的临时 `CODEX_HOME` 执行 `codex login`，成功后再把生成的 `auth.json` 保存成托管账号，不会直接覆盖当前运行中的活动账号。
3. 切换账号只替换当前活动 `auth.json`，并在需要时先生成备份文件；运行时设置更新通过 Codex app-server 写回当前用户配置中的 `model` 与 `model_reasoning_effort`，不会覆盖其他 Codex 配置项。

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

1. 服务启动后默认提供 `default-nl`、`memory`、`deploy-test-service` 与 `frontend-design` 四个内置 Skill。
2. `memory` Skill 用于向 Agent / Codex 明确记忆文件的读取决策、写入路由、冲突优先级与禁止写入项，建议与 `memory_files` 一起启用。
3. `deploy-test-service` 与 `frontend-design` 都是项目内置的 file-backed Skill，默认文件分别位于 `.alter0/skills/deploy-test-service/SKILL.md` 与 `docs/skills/frontend-design/SKILL.md`；前者约束会话级测试服务发布，后者约束前端页面、组件与应用实现的视觉方向、字体/配色/动效/构图质量。
4. `coding` 内置 Agent 默认启用 `memory`、`deploy-test-service` 与 `frontend-design`，进入 Coding Agent Runtime 后即可继承仓库记忆、预览发布与前端设计规则。
5. 每个 Agent 在运行时都会自动附带自己的私有 file-backed Skill，默认路径为 `.alter0/agents/<agent_id>/SKILL.md`；该 Skill 不出现在控制面内置列表里，但会稳定注入当前 Agent 的执行上下文，供 Agent 根据用户提出的长期偏好更新自己的可复用规则。Agent Runtime 的 Skill 面板会展示当前 Agent 私有 Skill，但该项固定启用且不可取消；可选区只列出 `memory`、`deploy-test-service`、`frontend-design` 等公有 Skill。
6. `travel` 的私有 `SKILL.md` 会预置 travel 城市页、行程、地铁、美食与地图输出规则，作为 travel agent 独占的可复用规则簿；稳定偏好写入该文件，一次性行程细节仍只保留在目标城市页数据中。

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
2. 系统默认注册内置 `main`、`coding`、`writing`、`travel` Agent；其中 `main / Alter0` 绑定 Chat 默认入口，不进入 Agent Runtime 可选列表，其余专项入口通过统一 `Agent Catalog` 暴露给运行时与前端。
3. 创建 Agent 时不需要手填 `id` 或 `version`；服务端会自动生成 Agent ID，并在每次更新时维护版本。若生成 ID 与内置 Agent 冲突，会自动跳过保留 ID。
4. Agent 运行时固定采用“Codex CLI 负责具体执行、Agent 负责理解和驱动”的模式：稳定工具面包括 `codex_exec`、`search_memory`、`read_memory`、`write_memory`，系统会自动补充收口工具 `complete`；允许委派的 Agent 可额外启用 `delegate_agent`；`coding` Agent 额外启用 `deploy_test_service`。`search_memory` 负责在已解析的记忆文件内按关键字检索历史偏好、缩写和上下文，再配合 `read_memory` / `write_memory` 做精读和更新。存在可用 Provider 且进入 Agent / ReAct 链路时，Agent 作为用户与 Codex 之间的代理层，在自身侧消化 system prompt、Skill、记忆与会话上下文，并只把当前执行所需的最小指令交给 `codex_exec`；不存在 Provider、Agent 初始化失败或 Terminal 直连 Codex 时，运行时会在当前会话工作区内生成独立 `codex-home/`、原生 `config.toml`、工作区 `AGENTS.md` 与 `.alter0/codex-runtime/*`，把 MCP、Skill、Memory 与运行时信息直接编译成 Codex 原生运行配置。`coding` Agent 会优先把实质性开发与验证步骤交给 `codex_exec`，并按每轮执行结果继续推进后续步骤；仓库类执行默认切到当前 Session 独立的 repo 完整 clone `.alter0/workspaces/sessions/<session_id>/repo`，该 clone 自带自己的 `.git` 元数据并允许直接分支与提交，运行时同步维护源仓库路径、独立 repo clone 路径、活动分支、会话工作区、短哈希预览域名与 PR 交付规则。需要测试页面或附加测试服务时，完成实现后必须先通过 `/api/control/workspace-services/{session_id}/{service_id}` 或 `scripts/deploy_test_service.sh` 把当前工作区注册到对应短哈希域名再结束本轮交付；默认 `web` 路径会把当前分支后端启动命令交给共享运行时托管，再把子域名直接路由到这份后端。
5. `Chat` 默认绑定 `main` Agent；`Agent` 页面作为其余入口 Agent 的统一运行页，并按目标 Agent 隔离维护独立会话历史；`Alter0/main` 不在 Agent 页面中作为可选 Agent 展示，具备独立前端入口的 Agent 不进入该页历史。
6. Agent 的公有 Skill、MCP 与 Memory Files 选择会在执行前先解析为运行时上下文；当前 Agent 的私有 Skill 始终自动注入，不受前端取消或 `alter0.skills.exclude` 排除。Agent / ReAct 路径由 Agent 自身吸收这些规则并仅向 Codex 下发当前步骤指令；直连 Codex 路径则把解析结果编译到当前会话的原生 Codex Runtime 中，Memory Files 的自动召回片段会写入 `.alter0/codex-runtime/memory/recall.md` 等工作区文件。同一 Session 的短期记忆会在内存窗口不足时从持久化 session history 回填最近完成轮次。
7. 内置 `memory` Skill 会明确记忆文件的读写逻辑：按任务类型决定先读哪些文件、按信息类型决定写入哪个文件、遇到冲突时按 `SOUL.md > AGENTS.md > 长期记忆 > 日记忆` 收敛；其中 `AGENTS.md` 固定为当前 Agent 的非共享规则文件 `.alter0/agents/<agent_id>/AGENTS.md`，`USER.md`、`SOUL.md` 与长期/日记忆继续共享；实际文件快照仍由 `memory_files` 注入提供。
8. 每个 Agent 还会自动拿到自己的私有 Skill 文件 `.alter0/agents/<agent_id>/SKILL.md`；运行时会把它作为可写 Skill 上下文注入，Agent 需要在用户提出稳定、可复用、会影响后续该 Agent 行为的偏好时按需更新它，而不是把一次性任务细节写进去。
9. `memory_files` 当前支持：`user_md`、`soul_md`、`agents_md`、`memory_long_term`、`memory_daily_today`、`memory_daily_yesterday`。
10. Memory Files 注入会携带文件内容、绝对路径、是否存在、最近更新时间；文件不存在时仍会暴露预期路径，便于 Agent 直接创建并写入。
11. Web `Agent Profiles` 页面用于管理用户自定义 Agent Profile；内置 Agent 由服务托管；`Chat` 页面绑定 `Alter0`；`Agent` 页面作为其余专项入口 Agent 的通用交互入口。

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

后续开发默认遵循 TDD：功能新增、缺陷修复、行为调整或重构需先新增或更新表达目标行为的测试，再完成实现与重构。纯文档、注释、格式化、依赖元数据或无法自动化验证的变更可免新增测试，但交付说明需明确免测原因与替代验证方式。

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

欢迎提 Issue / PR。代码类变更需遵循 TDD，并在提交前执行与改动范围匹配的测试；涉及共享链路、跨模块契约或用户可见行为时，建议执行：

```bash
go test ./...
```

## License

MIT, see [LICENSE](./LICENSE).
