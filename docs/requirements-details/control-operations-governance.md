# Control, Operations & Governance Requirements

> Last update: 2026-06-23

## 领域边界

Control, Operations & Governance 负责运行时配置管理、Model Provider、Claude Code provider profile、Codex Runtime、环境配置、部署基线、运行时重启、认证凭据、工具链初始化与研发流程约束。它维护系统可治理性，不直接定义业务对话行为。

## 核心对象

| 对象 | 职责 |
| --- | --- |
| `ChannelConfig` | 输入通道配置 |
| `Capability` | Skill、MCP 等能力配置的统一生命周期对象 |
| `CapabilityAudit` | Capability 创建、更新、启停和删除审计 |
| `SkillConfig` | Skill 配置、启停与文件型属性 |
| `MCPServer` | MCP Server 配置、启停与运行上下文注入来源 |
| `RuntimeProfile` | CLI Runtime 的默认执行配置、Skill 组合与上下文注入策略 |
| `ModelProvider` | Claude Code provider profile 的模型、base URL、凭据状态与健康状态 |
| `ClaudeProviderProfile` | 启动 Claude Code 时使用的 provider/profile、环境变量和模型选择 |
| `CodexAccount` | 托管的 Codex `auth.json` 快照与活动账号映射 |
| `CodexLoginSession` | 独立 `codex login` 会话状态、device-code 关键信息、日志与结果 |
| `RuntimeInstance` | 当前在线实例、启动时间与 commit hash |
| `WorkspaceServiceRegistration` | Session 短哈希域名到前端构建或 HTTP 测试服务的绑定关系 |
| `DeploymentBaseline` | systemd、Nginx、HOME、PATH、凭据与工具链要求 |
| `EngineeringPolicy` | TDD、文档同步、提交与验证规范 |

## Control API

### Channel

- 支持 Channel 创建、更新、删除与列表查询。
- Channel 配置至少包含类型、启停状态和稳定标识。
- Channels 作为兼容控制能力保留在 Settings 内部，不再作为一级入口展示。

### Skill

- 支持 Skill 创建、更新、删除与列表查询。
- 默认提供 `memory`、`preview-publish`、`frontend-design`、`doc-coauthoring`、`fullstack-developer`、`code-reviewer`、`webapp-testing`、`find-skills`、`test-driven-development`、`ui-ux-pro-max`、`code-simplifier`、`code-review`、`brainstorming` 与 `travel` 公有 Skill；`memory-maintenance` 作为系统维护专用私有 Skill 保留。
- 这些项目内置 file-backed Skill 都由源码仓库直接承载并在启动时校验文件存在；标准 skill 继续使用 `docs/skills/<skill_id>/SKILL.md`，附属脚本或参考文件与 skill 一同放在对应目录中；plugin-style 的 `code-simplifier` 与 `code-review` 保留 `.claude-plugin/plugin.json` 元数据，并分别以 `docs/skills/code-simplifier/SKILL.md`、`docs/skills/code-review/commands/code-review.md` 作为 alter0 的注入入口。CLI Runtime 会把本轮选中的可读 file-backed Skill 目录复制到当前会话工作区，Claude Code 路径写入 `.alter0/claude-runtime/skills/<skill_id>/`，Codex Direct 路径写入 `.alter0/codex-runtime/skills/<skill_id>/`，运行时上下文中的 `file_path` 指向工作区内副本。
- `preview-publish` 额外提供 `docs/skills/preview-publish/scripts/publish_preview_artifact.sh`，用于把文本、图片、代码等静态产物组装为单页预览并挂到 `<service>-<session_short_hash>.alter0.cn`。所有需要给用户浏览器查看的静态产物都必须通过该 skill 发布，不得返回服务器本地路径、工作区内部路径、`file://`、`localhost` 或 `127.0.0.1` 作为用户入口；需要完整 Web 应用或后端路由时同样使用 `preview-publish`。
- Skill 协议支持文件路径与可写属性。
- 服务不再注册内置业务编排；执行层只注入当前会话显式选择且控制面可见的 Skill。

### Capability 与 MCP

- Capability 统一接口支持按类型查询、创建或更新、删除 Skill、MCP 等能力配置。
- MCP 专用接口支持 MCP Server 创建或更新、列表查询、启用、禁用与删除。
- Capability 与 MCP 生命周期变更必须写入审计记录，审计列表支持按 capability type 查询。
- 旧 Skill/MCP 专用接口与统一 Capability 接口返回同一能力字段结构，避免前端维护两套协议。
- Capability 控制面只负责配置生命周期；实际是否注入执行链由 Runtime Profile、会话选择和运行时上下文解析决定。

### Runtime Profile

- 支持 Runtime Profile 的创建、更新、启用、禁用与查询。
- Runtime Profile 作为历史配置模型保留，当前稳定 Chat 入口不再依赖内置 Runtime Profile 或内置业务编排。
- 代码开发、旅行攻略、结构化写作等业务能力通过用户选择的 Skill 组合表达，不改变底层 CLI 执行链。
- Runtime Profile 编辑页中的短字段优先采用并排栅格布局，`Enabled` 使用显式开关控件。

### Cron 与 Codex Runtime

- Cron Job 控制面接口用于配置普通定时任务、展示系统内置维护任务、切换任务启停和查看触发记录；调度执行归属 Runtime & Orchestration。系统内置维护任务返回 `builtin=true`，不能删除，只能通过 `enabled` 停用或重新启用。
- Schedules 控制面展示系统内置维护任务的运行状态，不提供复杂配置项。内置任务包括每日记忆维护和每日会话清理；前端提供状态、上次运行、下次运行、失败信息、手动运行和失败重试。记忆维护执行器不可用时必须返回失败状态，不得记录为空运行成功。
- 会话清理固定使用超过 7 天不活跃的默认阈值，覆盖 Chat/Agent Session history 与 Terminal session store，跳过置顶会话、仍有关联 queued/running 任务的会话，以及 Terminal 中仍处于 busy/starting 的运行态会话；手动 `Clean up now` 与自动清理走同一后端服务，并返回扫描数量、删除数量、置顶跳过数量、保护跳过数量和 Terminal 专属明细统计。清理 Session history 后，关联任务、运行时 registry 或工作区删除失败时，本次维护状态必须标记为失败并返回错误信息；清理 Terminal 会话时复用 Terminal 删除服务同步移除状态文件与独立工作区，删除失败同样标记本次维护失败。
- Codex Runtime 控制面负责展示服务运行账户当前 Codex 身份、额度、profile、活动 model、思考深度与 LLM Provider 注册状态，并允许直接更新当前 Codex 配置中的 model 与思考深度。首屏加载时，运行时状态与 LLM Provider 状态必须并行读取，避免互不依赖的接口串行拖慢 Settings Runtime 分区。
- Codex Runtime 控制面支持启动 `device_auth` 登录会话；后端必须以独立登录目录运行 `codex login --device-auth`，并从登录输出中提取验证链接、完整验证链接、用户码、过期秒数、轮询秒数与原始日志。前端需在 Runtime 面板内展示这些关键信息并轮询会话状态，成功后刷新当前 Runtime 身份与额度。该能力仅辅助当前服务运行账户完成无头登录，不恢复多账号导入、切换或账号管理侧栏。
- Codex Runtime 控制面支持通过 Claude Code Provider Console 连续注册和编辑多个 OpenAI-compatible Provider；桌面端 registry 与 editor 在同一容器内左右分栏，窄屏单列展开。前端收集 Provider 名称、base URL、API key 与 models，models 使用全宽多行编辑区，支持换行或逗号分隔并按输入顺序去重，提交到 `POST /api/control/llm/providers` 或 `PUT /api/control/llm/providers/{id}`，默认使用 `openai-completions` API type，写入启用状态、多个启用模型和首个默认模型，并在成功后刷新 LLM Provider 注册状态。已注册 Provider 列表需展示名称、base URL、默认 model、模型数量、模型列表与启用/默认状态；点击编辑时将 Provider 当前 base URL 与 models 载入表单，API key 输入留空表示保留已保存密钥。每次成功后表单清空 base URL、API key 与 models，并自动准备下一个未占用的 `Claude Code N` 默认名称；用户已手动填写的非空表单不会被后台刷新覆盖。该入口复用 Model Provider 注册表与凭据遮蔽语义，不单独维护 Claude Code 私有配置源。
- Web Shell 由 `/settings` 单页承接 Runtime、Skills、Memory 与 Schedules 能力的读取、加载、空态与错误态渲染；这些能力不再作为一级侧栏入口或独立工作台 path 展示，而是在页内按 `Runtime / Skills / Memory / Schedules` 分区切换。桌面端分区切换作为左侧设置索引常驻，入口包含图标、短标识与活动态；真手机宽度下切换区使用双列按钮栅格，所有设置分区入口需直接可见且不依赖横向滚动。各分区正文需统一使用 Settings 作用域下的扁平 route surface：列表、表格、筛选表单、主从详情、空态与错误态共享白底、浅灰辅助层、必要分割线和紧凑字段行，不再默认使用外层卡片边框、厚圆角或重阴影表达层级。控制台页面中的描述、Cron 输入、Skill 说明与 Codex 运行时说明按安全 Markdown 渲染，ID、路径、密钥、配置值与时间戳保持纯文本或等宽字段展示。

## 接口边界

- `GET /api/control/channels`、`PUT /api/control/channels/{channel_id}`、`DELETE /api/control/channels/{channel_id}` 管理 Channel。
- `GET /api/control/capabilities`、`PUT /api/control/capabilities/{type}/{capability_id}`、`DELETE /api/control/capabilities/{type}/{capability_id}` 管理统一 Capability。
- `GET /api/control/capabilities/audit` 查询 Capability 生命周期审计。
- `GET /api/control/skills`、`PUT /api/control/skills/{skill_id}`、`POST /api/control/skills/{skill_id}`、`DELETE /api/control/skills/{skill_id}` 管理 Skill 兼容接口。
- `GET /api/control/mcps`、`PUT /api/control/mcps/{mcp_id}`、`POST /api/control/mcps/{mcp_id}`、`DELETE /api/control/mcps/{mcp_id}` 管理 MCP 兼容接口。
- `GET /api/control/workspace-services`、`GET /api/control/workspace-services/{session_id}`、`PUT /api/control/workspace-services/{session_id}`、`GET /api/control/workspace-services/{session_id}/{service_id}`、`PUT /api/control/workspace-services/{session_id}/{service_id}`、`DELETE /api/control/workspace-services/{session_id}/{service_id}` 管理 Session 级 workspace service 注册表。
- `GET /api/control/runtime` 读取在线实例信息。
- `POST /api/control/runtime/restart` 请求 supervisor 重启。
- `GET /api/control/codex/runtime` 查询当前服务运行账户的 Codex 身份、额度、profile、model、思考深度与可选 model 能力；`PUT /api/control/codex/runtime` 更新当前 Codex 配置中的 `model` 与 `model_reasoning_effort`。
- `POST /api/control/codex/accounts/login-sessions` 创建 Codex 登录会话；请求体支持 `auth_method=device_auth`，返回 `LoginSession`。`GET /api/control/codex/accounts/login-sessions/{session_id}` 查询登录会话状态、device-code 关键信息、日志、错误与成功后的账号快照。
- `GET /api/control/llm/providers`、`POST /api/control/llm/providers`、`GET /api/control/llm/providers/{provider_id}`、`PUT /api/control/llm/providers/{provider_id}`、`POST /api/control/llm/providers/{provider_id}`、`DELETE /api/control/llm/providers/{provider_id}` 管理 Model Provider。
- `GET /api/control/cron/jobs` 返回普通 Cron Job 与内置维护 Job；内置维护 Job 不允许 `DELETE`，允许通过 `PUT /api/control/cron/jobs/{job_id}` 的 `enabled` 字段停用或重新启用。
- `POST /api/sessions/{session_id}/pin` 更新会话置顶状态，body 使用 `{"pinned": true|false}`。
- `POST /api/terminal/sessions/{session_id}/pin` 更新 Terminal 会话置顶状态，body 使用 `{"pinned": true|false}`。

## Model Provider

### Provider 类型

- 支持 OpenAI Compatible Provider。
- 支持 OpenRouter Provider。
- Provider 支持启用、禁用、默认切换、模型列表、base URL、API type、Claude Code profile 名称和健康状态。
- Runtime 页的 Claude Code Provider 快速注册创建或更新 OpenAI Compatible Provider，支持连续维护多个 Provider，并把每个 Provider 填写的 models 同步为启用模型列表，首个 model 作为默认模型。
- 启用且健康的默认 Provider 作为用户显式选择 Provider/Model 或显式 Claude 执行器时的 Claude Code 运行来源；普通 Agent 请求默认仍进入 Codex Direct。

### API type

- `openai-responses` 走 `/responses`。
- `openai-completions` 走 `/chat/completions`。
- 自定义 `base_url` 时，目标服务必须兼容所选接口。
- OpenRouter 默认使用 `https://openrouter.ai/api/v1` 与 `openai-completions`。

### OpenRouter 扩展

- OpenRouter 可配置 `Site URL`、`App Name`、回退模型与 Provider 路由偏好。
- 系统分别注入官方请求头与请求体扩展字段。

### 凭据与默认项

- 保存 Provider 时，`api_key` 留空表示保持现有密钥。
- 前端中间态占位值 `-` 按空值处理，不持久化为真实凭据。
- 历史配置缺少 `api_key` 时，加载阶段自动收敛为禁用态并保留在 Models 控制面。
- 默认 Provider 只能落在已启用配置上；默认项被禁用、删除或失效时自动切换到下一可用 Provider，无可用项时清空默认值。

### 会话级选择

- Chat 发送区支持 route 级 `Provider / Model`、Tools/MCP 与 Skills 选择，并把选择持久化到浏览器本地配置。
- 当前消息默认选择 `Codex` 并进入 `Codex Direct`。
- 当前消息选择具体 Provider / Model 时，Runtime Resolver 使用对应 Claude Code provider profile。
- 未显式选择 Provider 时不回退到系统默认 Provider，直接进入 `Codex Direct`；显式 Provider 不可用时也回到 `Codex Direct`。

## Runtime Service Controls

- 旧运行参数配置页和对应接口不再提供；运行时路径、记忆文件、队列、终端 shell 等参数由源码内置默认值或启动配置控制，不在 Settings 中持久化为用户配置。
- Runtime 面板提供服务重启入口。更新远端 master 默认勾选；用户保持同步选项并确认重启后，前端先提交 `sync_remote_master=true` 请求。仅当后端检测到 Git 已跟踪本地改动并返回确认要求时，前端才进入二次确认。只有二次确认后才传入 `confirm_discard_tracked_changes=true` 并允许后端丢弃已跟踪改动；未跟踪文件保留。
- Runtime 面板展示当前在线实例最近启动时间和 commit hash，用于确认重启切换后的版本。
- 工具栏展示当前在线实例对应 commit hash。
- 页面重连到新实例后以站内成功弹窗提示。

## Workspace Service Gateway

### 域名与注册

- 共享运行时统一接收 `alter0.cn` 与 `*.alter0.cn` 的反向代理流量。
- Session 短哈希域名固定使用 `sha1(session_id)` 的前 8 位短哈希。
- 默认 `web` 服务域名格式为 `https://<session_short_hash>.alter0.cn`。
- 附加服务域名格式为 `https://<service>-<session_short_hash>.alter0.cn`。
- `travel` 服务域名固定为 `https://travel-<session_short_hash>.alter0.cn`，且该 host 只读、免登录。
- 控制面通过 `PUT /api/control/workspace-services/{session_id}` 或 `PUT /api/control/workspace-services/{session_id}/{service_id}` 注册当前会话工作区的前端构建目录或 HTTP upstream。
- 删除或重绑服务时，通过 `DELETE /api/control/workspace-services/...` 或再次 `PUT` 完成更新。
- 预览 host 继续沿用共享工作台登录保护：默认 `web` 短哈希 host 需要能直接打开 `/login`，但登录态按当前 host 独立维护，不与主域共享根域登录 cookie。

## Codex Runtime

### 运行目录

- 当前活动 `CODEX_HOME` 优先读取环境变量；未显式设置时，默认使用 `$HOME/.codex`。
- 当前活动账号以 `<active_codex_home>/auth.json` 为准；前端 Runtime 页面不提供多账号导入或切换入口。
- 当前 Codex 运行时管理通过 Codex app-server 读取与更新用户配置；稳定支持 `model` 与 `model_reasoning_effort` 两项运行时能力，实际可选值必须来自 Codex 返回的能力列表。

### 身份、额度与配置

- 控制面默认展示单一 Runtime 面板：上方展示当前 Codex 身份快照、邮箱、计划、认证模式与 profile，下方展示可编辑 model / 思考深度和 hourly / weekly 额度。
- 额度展示必须来自当前 `auth.json` 的 quota 刷新结果；quota 成功返回时即可展示具体剩余额度与 reset 时间，前端不再依赖旧账号列表接口。
- 页面不展示 Account ID / User ID、保存名称、多账号管理动作、导入/切换操作侧栏、CLI 命令、auth/config 路径、独立就绪侧栏、诊断面板或由 auth/config 文件存在性推导的 Ready/Status 文案；device-code 登录只作为当前运行账户的认证辅助动作展示。
- 当前 Codex 管理接口需返回活动 `auth.json`、当前 `auth.json` 身份快照、实时刷新后的 quota 信息、`config.toml`、当前 profile、活动 model、思考深度、配置来源与可选 model 列表，供前端直接展示身份、额度和真实可选项。
- 当前 Codex 管理区需允许直接切换活动 model 与思考深度，选择变更后立即写回当前用户配置；前端只允许提交当前所选 model 实际支持的思考深度。

### 分发规则

- 当请求 Host 命中已注册短哈希域名时，共享运行时按服务类型分发：
  - `frontend_dist` 直接分发该工作区的 `/`、`/chat`、`/assets/*` 与 `/legacy/*`。
  - `http` 把全部请求反向代理到注册的本地或远端 upstream。
- 根短哈希 `web` 服务允许直接注册为 `http`，使 `https://<session_short_hash>.alter0.cn` 整体反向代理到当前会话后端实例，包括 `/`、`/api/*`、登录页和终端相关接口。
- `frontend_dist` 仅覆盖静态前端构建；选择该模式时，`/api/*`、登录态、健康检查和共享后端能力仍由主运行时提供。
- workspace service 注册表需持久化到 `.alter0/workspace-services.json`，以便运行时重启后继续保持域名绑定。

## Runtime Restart

### 重启流程

- Web 控制台发起重启时，由 supervisor 托管子进程切换。
- 重启确认使用单一站内弹窗。
- “同步远端 master 最新改动”作为弹窗内勾选项展示，默认勾选。

### 同步远端

- `sync_remote_master=false` 时，基于当前仓库状态构建候选二进制并切换。
- `sync_remote_master=true` 时，先校验当前分支为 `master`；无 Git 已跟踪本地改动时直接拉取、快进、通过统一构建入口重建前端产物和候选二进制并切换；若存在 Git 已跟踪本地改动，后端必须返回稳定确认错误，前端据此进入二次确认。未确认时拒绝重启同步且不清理本地内容。
- Git、构建或快进失败直接返回到 Web 控制台，便于定位权限与凭据问题。

### 切换与回滚

- 候选版本只有在 `/readyz` 探活通过后才成为当前运行版本。
- 候选启动失败时自动恢复上一运行版本。
- 重启完成后页面自动刷新到新实例。

## 部署基线

### Web 暴露

- 公网部署建议通过 Nginx 反向代理。
- 若需要会话短哈希预览域名，Nginx / CDN / DNS 需将 `*.alter0.cn` 与主域一并转发到同一共享运行时实例。
- 服务进程建议绑定 localhost。
- 应用内登录页通过 `ALTER0_WEB_LOGIN_PASSWORD` 或启动参数启用。

### 运行目录

- systemd 部署基线将服务 `HOME` 收敛到 `/var/lib/alter0`。
- 历史 `HOME=/var/lib/alter0/codex-home` 启动时归一到 `/var/lib/alter0`。
- `.alter0` 运行态、Codex 认证与服务账户工具链共享同一运行根目录。

### 交付凭据

- 服务内需要执行 `git commit`、`git push`、`gh pr create`、`gh pr merge` 时，运行账户必须具备 GitHub App token helper、`gh` 包装器、SSH signing key 与全局 Git 配置。
- 初始化脚本 `scripts/setup_alter0_runtime_auth.sh` 负责配置运行账户级凭据与签名。
- 提交操作不得通过关闭签名绕过凭据问题。

### Node / Playwright 工具链

- 服务内需要执行 Node/Playwright 测试时，运行账户必须具备 `node`、`npm`、`npx` 与 Playwright Chromium。
- 初始化脚本 `scripts/setup_alter0_runtime_node.sh` 默认将工具链安装到 `/var/lib/alter0/.local`。
- 初始化脚本默认在 `internal/interfaces/web` 与 `internal/interfaces/web/frontend` 预装 `npm ci`，确保 E2E、前端构建与前端单测共用同一运行账户工具链。
- 服务启动时补齐 `/var/lib/alter0/.local/bin` 到 PATH，使 Codex CLI、Web 子进程和手工切换到账户后的执行环境一致。
- 服务启动、服务重启、Runtime supervisor 候选二进制构建与维护者手工构建二进制统一使用 `scripts/build_alter0_service.sh`：该入口先重建 `internal/interfaces/web/static/dist`，校验入口 HTML 引用了哈希 JS/CSS 资产，再构建 Go 服务二进制，确保 `go:embed` 使用的前端产物与当前分支源码一致。
- Web Shell 入口 HTML 与 `frontend_dist` workspace preview HTML 在服务端输出前必须按实际 `/assets/index-*.js|css` 内容注入 `?v=<content-hash>`；`static/dist/assets` 可继续使用长期 immutable 缓存，但服务重启、代码快进或预览刷新后，资产内容变化必须产生新的浏览器 URL，避免旧 bundle 因客户端缓存继续生效。
- Session 级测试服务的标准部署入口为 `scripts/deploy_test_service.sh`，它负责构建或注册工作区服务，并调用共享运行时的 workspace service 注册接口。默认 `scripts/deploy_test_service.sh <session_id>` 会先构建前端，再把当前分支 Web 后端的启动命令、工作目录、端口与健康检查路径注册给共享运行时托管，并把默认 `web` 短哈希域名绑定成 `http` 反代；如需纯静态 UI 预览，显式传 `--service-type frontend_dist`。

## 安全与认证

- Web 登录密码启用后，未认证访问不能进入受保护页面和 API。
- `/login` 使用统一登录态写入；`/logout` 清理登录态并返回登录流程。任一工作台页面触发登录时只把当前 canonical path 作为回跳目标，不携带 query，避免在登录 URL、隐藏表单字段与页面提示中暴露 `session_id` 等会话级参数。
- 共享运行时的主 Web child 必须继承同一份 `web_login_password`；只有 workspace service 托管的预览后端可以通过专用运行时环境标记移除自身登录层并复用共享网关登录态。
- 服务账户缺少 Codex/OpenAI 认证时，Web 端快速返回认证失败，不长时间等待。
- Codex Runtime 控制面不提供导出接口，也不暴露本地文件浏览能力。
- 本地路径不直接作为 Web 产物交付方式暴露给用户。

## 研发治理

### TDD

- 功能新增、缺陷修复、行为调整或重构默认遵循 TDD。
- 缺陷修复优先补充回归测试。
- 功能新增覆盖核心领域规则、应用服务行为或外部接口契约中受影响的最小可验证面。
- 领域规则测试放在 domain，用例编排测试放在 application，Web/CLI 入口契约放在 interfaces，存储、LLM、Codex CLI、外部进程适配契约放在 infrastructure。
- Go 单元测试用例说明统一维护在 `docs/testing/unit-test-cases.md`，各 Go 包路径同步维护本地 `TEST_CASES.md`，按领域路径记录覆盖范围、边界与执行口径。
- 不得通过跳过测试、删除断言、降低断言强度、扩大 mock 宽容度或吞掉错误来掩盖回归。
- 纯文档、注释、格式化、依赖元数据或无法自动化验证的变更可不新增测试，但交付说明必须明确免测原因与替代验证方式。

### 文档同步

- 用户可见行为、交互方式、入口路由、执行模式、返回结构或默认策略变化时，同步更新 README。
- 需求变更同步更新 `docs/requirements.md` 与对应领域细化文档。
- 涉及架构、接口、数据结构、执行链路、存储、部署或测试策略时，同步更新 `docs/technical-solution.md` 的同名领域方案。
- 后续不再新增线性编号需求项或按编号拆分的需求细化文件。
- 领域边界或主链路变化时，同步更新 `docs/architecture.md`。

### 输出与临时产物

- 临时产物统一写入 `output/`。
- 测试结果、截图、调试导出、Smoke 记录、临时脚本输出与本地排查产物不得散落到根目录或业务目录。

## 依赖与边界

- Control 面负责配置治理，不绕过 Runtime 编排链路直接执行任务。
- Model Provider 属于控制与适配配置，具体执行错误收口归属 Runtime。
- 部署基线描述运行账户与宿主环境要求，不替代产品内鉴权和权限模型。

## 验收口径

- Channel、Capability、Skill、MCP、Skill、Cron、Models 与 Runtime 控制面接口可用。
- Capability 与 MCP 生命周期审计可查询。
- 禁用默认 Provider 后自动收敛到可用配置。
- Runtime 重启成功后页面连接到新实例，失败时自动回滚。
- systemd 部署下服务账户可以访问 Codex、gh、node、npm、npx 与签名凭据。
- 文档类变更说明免测原因；代码类变更按 TDD 补测试并运行匹配测试集。
