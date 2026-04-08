# Control, Operations & Governance Requirements

> Last update: 2026-04-08

## 领域边界

Control, Operations & Governance 负责运行时配置管理、模型 Provider、环境配置、部署基线、运行时重启、认证凭据、工具链初始化与研发流程约束。它维护系统可治理性，不直接定义业务对话或 Product 领域行为。

## 核心对象

| 对象 | 职责 |
| --- | --- |
| `ChannelConfig` | 输入通道配置 |
| `SkillConfig` | Skill 配置、启停与文件型属性 |
| `AgentProfile` | 用户可维护 Agent Profile |
| `ModelProvider` | LLM Provider、模型、接口类型与凭据状态 |
| `EnvironmentConfig` | 运行参数、任务并发、Shell、Codex 命令等配置 |
| `RuntimeInstance` | 当前在线实例、启动时间与 commit hash |
| `DeploymentBaseline` | systemd、Nginx、HOME、PATH、凭据与工具链要求 |
| `EngineeringPolicy` | TDD、文档同步、提交与验证规范 |

## Control API

### Channel

- 支持 Channel 创建、更新、删除与列表查询。
- Channel 配置至少包含类型、启停状态和稳定标识。
- Channels 入口归属 Settings 模块，旧直达路由保持兼容。

### Skill

- 支持 Skill 创建、更新、删除与列表查询。
- 默认提供 `default-nl` 与 `memory`。
- Skill 协议支持文件路径与可写属性。
- Agent 私有 file-backed Skill 由运行时自动注入，不要求出现在控制面内置 Skill 列表。

### Agent Profile

- 支持用户管理 Agent Profile 的创建、更新、启用、禁用与查询。
- 内置 Agent 由服务注册，控制面不可覆盖或删除。
- Agent Profile 的模型、工具、Skills、MCP 和 Memory Files 选择通过统一运行时上下文注入执行链。

### Product 与 Cron

- Product 控制面接口用于目录管理、草稿生成、审核和发布；业务规则归属 Product Domain。
- Cron Job 控制面接口用于配置与触发记录查看；调度执行归属 Runtime & Orchestration。

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
- 配置更新不得破坏当前已运行会话的持久化状态。

### 实例信息

- 工具栏展示当前在线实例最近启动时间。
- 工具栏展示当前在线实例对应 commit hash。
- 页面重连到新实例后以站内成功弹窗提示。

## Runtime Restart

### 重启流程

- Web 控制台发起重启时，由 supervisor 托管子进程切换。
- 重启确认使用单一站内弹窗。
- “同步远端 master 最新改动”作为弹窗内勾选项展示，默认勾选。

### 同步远端

- `sync_remote_master=false` 时，基于当前仓库状态构建候选二进制并切换。
- `sync_remote_master=true` 时，先校验当前分支为 `master`、工作区干净且已跟踪远端，再执行 `git fetch --prune origin master` 与 `git merge --ff-only FETCH_HEAD`。
- Git、构建或快进失败直接返回到 Web 控制台，便于定位权限与凭据问题。

### 切换与回滚

- 候选版本只有在 `/readyz` 探活通过后才成为当前运行版本。
- 候选启动失败时自动恢复上一运行版本。
- 重启完成后页面自动刷新到新实例。

## 部署基线

### Web 暴露

- 公网部署建议通过 Nginx 反向代理。
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
- 服务启动时补齐 `/var/lib/alter0/.local/bin` 到 PATH，使 Codex CLI、Web 子进程和手工切换到账户后的执行环境一致。

## 安全与认证

- Web 登录密码启用后，未认证访问不能进入受保护页面和 API。
- 服务账户缺少 Codex/OpenAI 认证时，Web 端快速返回认证失败，不长时间等待。
- Product 公开目录只暴露 `active + public` 内容，不暴露控制面草稿、凭据或审核记录。
- 本地路径不直接作为 Web 产物交付方式暴露给用户。

## 研发治理

### TDD

- 功能新增、缺陷修复、行为调整或重构默认遵循 TDD。
- 缺陷修复优先补充回归测试。
- 功能新增覆盖核心领域规则、应用服务行为或外部接口契约中受影响的最小可验证面。
- 领域规则测试放在 domain，用例编排测试放在 application，Web/CLI 入口契约放在 interfaces，存储、LLM、Codex CLI、外部进程适配契约放在 infrastructure。
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

- Channel、Skill、Agent、Product、Cron、Models、Environments 控制面接口可用。
- 禁用默认 Provider 后自动收敛到可用配置。
- Runtime 重启成功后页面连接到新实例，失败时自动回滚。
- systemd 部署下服务账户可以访问 Codex、gh、node、npm、npx 与签名凭据。
- 文档类变更说明免测原因；代码类变更按 TDD 补测试并运行匹配测试集。
