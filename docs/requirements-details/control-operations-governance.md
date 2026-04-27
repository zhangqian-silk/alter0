# Control, Operations & Governance Requirements

> Last update: 2026-04-24

## 领域边界

Control, Operations & Governance 负责运行时配置管理、模型 Provider、环境配置、部署基线、运行时重启、认证凭据、工具链初始化与研发流程约束。它维护系统可治理性，不直接定义业务对话行为。

## 核心对象

| 对象 | 职责 |
| --- | --- |
| `ChannelConfig` | 输入通道配置 |
| `Capability` | Skill、MCP、Agent 等能力配置的统一生命周期对象 |
| `CapabilityAudit` | Capability 创建、更新、启停和删除审计 |
| `SkillConfig` | Skill 配置、启停与文件型属性 |
| `MCPServer` | MCP Server 配置、启停与运行上下文注入来源 |
| `AgentProfile` | 用户可维护 Agent Profile |
| `ModelProvider` | LLM Provider、模型、接口类型与凭据状态 |
| `EnvironmentConfig` | 运行参数、任务并发、Shell、Codex 命令等配置 |
| `CodexAccount` | 托管的 Codex `auth.json` 快照与活动账号映射 |
| `CodexLoginSession` | 独立 `codex login` 会话状态、日志与结果 |
| `RuntimeInstance` | 当前在线实例、启动时间与 commit hash |
| `WorkspaceServiceRegistration` | Session 短哈希域名到前端构建或 HTTP 测试服务的绑定关系 |
| `DeploymentBaseline` | systemd、Nginx、HOME、PATH、凭据与工具链要求 |
| `EngineeringPolicy` | TDD、文档同步、提交与验证规范 |

## Control API

### Channel

- 支持 Channel 创建、更新、删除与列表查询。
- Channel 配置至少包含类型、启停状态和稳定标识。
- Channels 入口归属 Settings 模块，旧直达路由保持兼容。

### Skill

- 支持 Skill 创建、更新、删除与列表查询。
- 默认提供 `default-nl`、`memory`、`deploy-test-service` 与 `frontend-design`。
- `deploy-test-service` 与 `frontend-design` 作为项目内置 file-backed Skill 由源码仓库直接承载并在启动时校验文件存在；当前路径分别为 `.alter0/skills/deploy-test-service/SKILL.md` 与 `docs/skills/frontend-design/SKILL.md`。
- Skill 协议支持文件路径与可写属性。
- Agent 私有 file-backed Skill 由运行时自动注入，不要求出现在控制面内置 Skill 列表。

### Capability 与 MCP

- Capability 统一接口支持按类型查询、创建或更新、删除 Skill、MCP 等能力配置。
- MCP 专用接口支持 MCP Server 创建或更新、列表查询、启用、禁用与删除。
- Capability 与 MCP 生命周期变更必须写入审计记录，审计列表支持按 capability type 查询。
- 旧 Skill/MCP 专用接口与统一 Capability 接口返回同一能力字段结构，避免前端维护两套协议。
- Capability 控制面只负责配置生命周期；实际是否注入执行链仍由 Agent Profile、会话选择和运行时上下文解析决定。

### Agent Profile

- 支持用户管理 Agent Profile 的创建、更新、启用、禁用与查询。
- 内置 Agent 由服务注册，控制面不可覆盖或删除。
- Agent Profile 的模型、工具、Skills、MCP 和 Memory Files 选择通过统一运行时上下文注入执行链。
- Agent Profile 编辑页中的短字段优先采用并排栅格布局，`Enabled` 使用显式开关控件，不再把名称、迭代次数和启用状态拆成长纵向表单链。

### Cron 与 Codex Accounts

- Cron Job 控制面接口用于配置与触发记录查看；调度执行归属 Runtime & Orchestration。
- Codex Accounts 控制面负责运行账户下的多账号快照管理、独立登录会话、状态查询与当前生效账号切换。
- Web Shell 桥接阶段由 React 直接承接 `agent`、`channels`、`skills`、`mcp`、`models`、`environments`、`codex-accounts` 与 `cron-jobs` 八类控制台路由体的读取、加载、空态与错误态渲染；其中 `agent` 路由已由 React 承接托管 Agent 列表、表单编辑、保存、删除与进入运行页入口；legacy runtime 不再托管这些页面的主要 DOM。

## 接口边界

- `GET /api/control/channels`、`PUT /api/control/channels/{channel_id}`、`DELETE /api/control/channels/{channel_id}` 管理 Channel。
- `GET /api/control/capabilities`、`PUT /api/control/capabilities/{type}/{capability_id}`、`DELETE /api/control/capabilities/{type}/{capability_id}` 管理统一 Capability。
- `GET /api/control/capabilities/audit` 查询 Capability 生命周期审计。
- `GET /api/control/skills`、`PUT /api/control/skills/{skill_id}`、`POST /api/control/skills/{skill_id}`、`DELETE /api/control/skills/{skill_id}` 管理 Skill 兼容接口。
- `GET /api/control/mcps`、`PUT /api/control/mcps/{mcp_id}`、`POST /api/control/mcps/{mcp_id}`、`DELETE /api/control/mcps/{mcp_id}` 管理 MCP 兼容接口。
- `GET /api/control/agents`、`POST /api/control/agents`、`GET /api/control/agents/{agent_id}`、`PUT /api/control/agents/{agent_id}`、`DELETE /api/control/agents/{agent_id}` 管理用户 Agent Profile。
- `GET /api/control/environments` 与 `PUT /api/control/environments` 读取和更新 Environment 配置。
- `GET /api/control/environments/audits` 查询 Environment 配置审计。
- `GET /api/control/workspace-services`、`GET /api/control/workspace-services/{session_id}`、`PUT /api/control/workspace-services/{session_id}`、`GET /api/control/workspace-services/{session_id}/{service_id}`、`PUT /api/control/workspace-services/{session_id}/{service_id}`、`DELETE /api/control/workspace-services/{session_id}/{service_id}` 管理 Session 级 workspace service 注册表。
- `GET /api/control/runtime` 读取在线实例信息。
- `POST /api/control/runtime/restart` 请求 supervisor 重启。
- `GET /api/control/codex/accounts`、`POST /api/control/codex/accounts`、`POST /api/control/codex/accounts/{account_name}/switch`、`POST /api/control/codex/accounts/login-sessions`、`GET /api/control/codex/accounts/login-sessions/{session_id}` 管理 Codex 多账号。
- `GET /api/control/llm/providers`、`POST /api/control/llm/providers`、`GET /api/control/llm/providers/{provider_id}`、`PUT /api/control/llm/providers/{provider_id}`、`POST /api/control/llm/providers/{provider_id}`、`DELETE /api/control/llm/providers/{provider_id}` 管理 Model Provider。

## Model Provider

### Provider 类型

- 支持 OpenAI Compatible Provider。
- 支持 OpenRouter Provider。
- Provider 支持启用、禁用、默认切换、模型列表、base URL 与 API type。

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

- Chat 发送区支持会话级 `Provider / Model` 选择。
- 复杂度评估、同步执行、Agent/ReAct 执行优先复用当前消息选择。
- 未显式选择时，回退到 Agent Profile，再回退到系统默认 Provider 与默认模型。

## Environments

### 配置管理

- Environments 模块支持关键运行参数可视化配置、校验与持久化。
- 可纳管配置包括任务并发、Terminal shell、Codex 命令、运行时路径与执行相关环境键。
- Environments 桌面端默认采用运行态工具栏 + 模块化配置卡片：顶部工具栏展示最近启动时间、commit hash、敏感值显隐、重载、保存与重启入口，下方按模块输出配置卡片、输入控件、更多信息折叠区与审计列表。
- Web & Queue 配置覆盖 `web_addr`、`web_bind_localhost_only`、`web_login_password`、`worker_pool_size`、`max_queue_size` 与 `queue_timeout`。
- Async Tasks 配置覆盖 `async_task_workers`、`async_task_timeout`、`async_task_max_retries`、`async_task_trigger_threshold` 与 `async_long_content_threshold`。
- Terminal 配置覆盖 `task_terminal_shell`；启动参数可继续提供 shell args 作为运行态输入。
- Session Memory 配置覆盖 `session_memory_turns`、`session_memory_ttl`、`context_compression_threshold`、`context_compression_summary_tokens` 与 `context_compression_retain_turns`。
- Persistent Memory 配置覆盖 `daily_memory_dir`、`long_term_memory_path`、`long_term_memory_write_policy`、`long_term_memory_writeback_flush`、`long_term_memory_token_budget` 与 `mandatory_context_file`。
- LLM 配置覆盖 `llm_temperature`、`llm_max_tokens` 与 `llm_react_max_iterations`，并按配置项声明决定立即生效或重启后生效。
- 配置更新不得破坏当前已运行会话的持久化状态。
- 配置更新必须记录 Environment audit，审计列表支持按时间倒序查看操作者、变更 key 与变更时间。
- 敏感配置默认隐藏，显式切换后才读取真实值；隐藏态下未编辑的敏感字段不得因为保存动作被误清空。

### 实例信息

- 工具栏展示当前在线实例最近启动时间。
- 工具栏展示当前在线实例对应 commit hash。
- 页面重连到新实例后以站内成功弹窗提示。

## Workspace Service Gateway

### 域名与注册

- 共享运行时统一接收 `alter0.cn` 与 `*.alter0.cn` 的反向代理流量。
- Session 短哈希域名固定使用 `sha1(session_id)` 的前 8 位短哈希。
- 默认 `web` 服务域名格式为 `https://<session_short_hash>.alter0.cn`。
- 附加服务域名格式为 `https://<service>.<session_short_hash>.alter0.cn`。
- `travel` 服务域名固定为 `https://<session_short_hash>.travel.alter0.cn`，且该 host 只读、免登录。
- 控制面通过 `PUT /api/control/workspace-services/{session_id}` 或 `PUT /api/control/workspace-services/{session_id}/{service_id}` 注册当前会话工作区的前端构建目录或 HTTP upstream。
- 删除或重绑服务时，通过 `DELETE /api/control/workspace-services/...` 或再次 `PUT` 完成更新。
- 预览 host 继续沿用共享工作台登录保护：默认 `web` 短哈希 host 需要能直接打开 `/login`，并与主域共享根域登录 cookie；用户不应因为切到 `https://<session_short_hash>.alter0.cn` 而丢失已建立的工作台登录态。

## Codex Accounts

### 运行目录

- 当前活动 `CODEX_HOME` 优先读取环境变量；未显式设置时，默认使用 `$HOME/.codex`。
- 托管账号、认证备份与登录会话目录统一持久化到 `<active_codex_home>/alter0-accounts`。
- 当前活动账号仍以 `<active_codex_home>/auth.json` 为准，切换账号只替换该文件，不改写其他 Codex 配置文件。
- 当前 Codex 运行时管理通过 Codex app-server 读取与更新用户配置；稳定支持 `model` 与 `model_reasoning_effort` 两项运行时能力，实际可选值必须来自 Codex 返回的能力列表。

### 新增与登录

- 面板支持导入已有 `auth.json` 内容创建托管账号。
- 面板支持启动独立的 `codex login` 会话；登录过程使用隔离的临时 `CODEX_HOME`，避免污染当前运行中的活动账号。
- 登录会话完成后，把新生成的 `auth.json` 保存为托管账号；是否生效由用户显式切换决定。
- 控制面默认展示运行时概览条、当前 Codex 管理区、托管账号高密度列表与导入/登录操作侧栏；概览区采用当前账号主身份区配合套餐、小时/周剩余额度与托管数量的紧凑指标列，小时/周额度必须以进度条呈现并显示 reset 时间，key/value 默认按同列对齐，概览本身不展示活动 auth 路径，当前 Codex 管理区需保留当前 profile、当前 model、当前思考深度和运行时状态，但 model / 思考深度只保留一套可编辑字段，不再在选择器下方重复展示当前值，CLI 命令、auth/config 状态、路径与配置来源统一收纳到 `Runtime Details` 折叠区，操作区需持续显示最近一次登录会话状态与日志。
- 当 `<active_codex_home>/auth.json` 已存在但尚未匹配任何托管快照时，概览区仍需显示当前 live 账号身份；若 quota 已成功获取，概览区继续直接展示该 live 账号的套餐与额度，托管列表空态需明确提示“当前 auth 已生效但未托管”。
- 控制面首次加载或刷新期间需保留概览、托管区与侧栏结构，用 skeleton 占位替代真实数据，避免整页收缩为单行提示。
- 当前 Codex 管理区需允许直接切换活动 model 与思考深度，并在保存后刷新同页状态；前端只允许提交当前所选 model 实际支持的思考深度。
- 托管账号区需按断点切换布局：大屏优先展示当前 Codex 管理区、行式账号列表与右侧操作栏，中屏切换为全宽账号区加下方双侧栏，窄屏回落为单列紧凑列表；套餐、小时/周额度进度条、reset 时间、当前 model、思考深度与切换按钮不得依赖横向滚动后才可见，状态展示与切换控件应避免使用多层胶囊式嵌套，并优先采用分隔线式而非内嵌方框布局。

### 状态与切换

- 账号列表展示托管快照里的账号名、邮箱、计划类型、当前激活标识与额度状态。
- OAuth 账号状态刷新允许先用 `refresh_token` 刷新 access token，再查询额度接口。
- 单个账号状态刷新失败不能阻断整个列表返回，错误信息按账号维度回显。
- 切换前若当前活动 `auth.json` 与目标账号不同，系统需先生成备份文件用于追溯。
- 若当前活动 `auth.json` 与目标账号属于同一身份且活动 token 更新更近，则优先用当前活动认证回写托管快照。
- 当前 Codex 管理接口需返回活动 `auth.json`、`config.toml`、CLI 命令、当前 profile、活动 model、思考深度、配置来源与可选 model 列表，供前端直接展示“Ready / Auth missing / Config default”等运行态文案，并构建真实可选项。

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
- `sync_remote_master=true` 时，先校验当前分支为 `master`，仅丢弃 Git 已跟踪的本地改动并保留未跟踪文件/目录，再执行 `git fetch --prune origin master` 与 `git merge --ff-only FETCH_HEAD`。
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
- Session 级测试服务的标准部署入口为 `scripts/deploy_test_service.sh`，它负责构建或注册工作区服务，并调用共享运行时的 workspace service 注册接口。默认 `scripts/deploy_test_service.sh <session_id>` 会先构建前端，再把当前分支 Web 后端的启动命令、工作目录、端口与健康检查路径注册给共享运行时托管，并把默认 `web` 短哈希域名绑定成 `http` 反代；如需纯静态 UI 预览，显式传 `--service-type frontend_dist`。

## 安全与认证

- Web 登录密码启用后，未认证访问不能进入受保护页面和 API。
- `/login` 使用统一登录态写入；`/logout` 清理登录态并返回登录流程。
- 共享运行时的主 Web child 必须继承同一份 `web_login_password`；只有 workspace service 托管的预览后端可以移除自身登录层并复用共享网关登录态。
- 服务账户缺少 Codex/OpenAI 认证时，Web 端快速返回认证失败，不长时间等待。
- Codex Accounts 控制面不提供导出接口，也不暴露本地文件浏览能力。
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
- Model Provider 属于控制与适配配置，具体执行错误收口归属 Agent 或 Runtime。
- 部署基线描述运行账户与宿主环境要求，不替代产品内鉴权和权限模型。

## 验收口径

- Channel、Capability、Skill、MCP、Agent、Cron、Models、Environments 控制面接口可用。
- Capability 与 MCP 生命周期审计可查询。
- 禁用默认 Provider 后自动收敛到可用配置。
- Runtime 重启成功后页面连接到新实例，失败时自动回滚。
- systemd 部署下服务账户可以访问 Codex、gh、node、npm、npx 与签名凭据。
- 文档类变更说明免测原因；代码类变更按 TDD 补测试并运行匹配测试集。
