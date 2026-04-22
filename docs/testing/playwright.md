# Playwright 浏览器自动化测试

## 定位

Playwright 用于 `alter0` Web 控制台的端到端交互回归，覆盖页面打开、登录保护、输入法合成、草稿持久化、会话切换、轮询刷新与恢复行为。

当前重点覆盖：

1. 聊天输入框的未发送内容保护
2. 输入法合成态下的稳定输入
3. 草稿持久化、会话隔离与恢复
4. Terminal 会话创建、轮询刷新、关闭与中断恢复
5. 受保护路由与 API 的登录态接入

## 代码位置

- 配置文件：`internal/interfaces/web/playwright.config.js`
- 测试目录：`internal/interfaces/web/e2e/`
- 测试产物：`output/playwright/`

当前已落地文件：

- `internal/interfaces/web/e2e/chat.spec.ts`
- `internal/interfaces/web/e2e/cron.spec.ts`
- `internal/interfaces/web/e2e/runtime-workspace.spec.ts`
- `internal/interfaces/web/e2e/terminal.spec.ts`
- `internal/interfaces/web/e2e/helpers/asserts/composer.ts`
- `internal/interfaces/web/e2e/helpers/asserts/input.ts`
- `internal/interfaces/web/e2e/helpers/components/composer.ts`
- `internal/interfaces/web/e2e/helpers/components/session-list.ts`
- `internal/interfaces/web/e2e/helpers/flows/auth.ts`
- `internal/interfaces/web/e2e/helpers/flows/chat-draft.ts`
- `internal/interfaces/web/e2e/helpers/flows/chat-session.ts`
- `internal/interfaces/web/e2e/helpers/flows/routes.ts`
- `internal/interfaces/web/e2e/helpers/flows/terminal-runtime.ts`
- `internal/interfaces/web/e2e/helpers/flows/terminal-session.ts`
- `internal/interfaces/web/e2e/helpers/guards/app-ready.ts`
- `internal/interfaces/web/e2e/helpers/guards/login.ts`
- `internal/interfaces/web/e2e/helpers/guards/unsaved.ts`
- `internal/interfaces/web/e2e/helpers/interactions/ime.ts`
- `internal/interfaces/web/e2e/helpers/pages/chat.ts`
- `internal/interfaces/web/e2e/helpers/pages/cron.ts`
- `internal/interfaces/web/e2e/helpers/pages/login.ts`
- `internal/interfaces/web/e2e/helpers/pages/app-shell.ts`
- `internal/interfaces/web/e2e/helpers/pages/terminal.ts`
- `internal/interfaces/web/e2e/helpers/scenarios/chat.ts`
- `internal/interfaces/web/e2e/helpers/scenarios/cron.ts`
- `internal/interfaces/web/e2e/helpers/scenarios/terminal.ts`

相关输入组件说明：

- `docs/testing/composer.md`

## 执行方式

在 `internal/interfaces/web` 目录执行：

- 安装依赖：`npm ci`
- 首次安装浏览器：`npx playwright install chromium`
- 执行全部 E2E：`npm run test:e2e`
- 执行聊天专项：`npx playwright test -c playwright.config.js e2e/chat.spec.ts`
- 执行 Cron 专项：`npx playwright test -c playwright.config.js e2e/cron.spec.ts`
- 执行 Terminal 专项：`npx playwright test -c playwright.config.js e2e/terminal.spec.ts`

若通过 `alter0` 服务内的 `Codex CLI` 执行上述命令，需先在宿主机 root 下运行一次：

- `sudo ./scripts/setup_alter0_runtime_node.sh`

该初始化会把带 `npm`/`npx` 的 Node 运行时安装到服务运行账户的 `HOME` 下，并默认预装当前仓库的 Web E2E 依赖与 Chromium，避免 `Codex CLI` 在非交互式运行态中因为缺少 `npm` 而跳过 Playwright/E2E。

默认路径选择为 `/var/lib/alter0/.local`，目的是把 Node/Playwright 依赖收敛在 `alter0` 服务账户自己的运行时目录内，不污染系统全局安装；脚本会把 `node`、`npm`、`npx`、`corepack` 统一暴露到 `/var/lib/alter0/.local/bin`，只要服务 `PATH` 包含这一目录，`Codex CLI` 与测试子进程就能稳定复用同一套工具链。

通过 `alter0` 服务内的 `Codex CLI` 触发 E2E 时，运行时会把当前 `web_login_password` 同步为子进程可见的 `ALTER0_WEB_LOGIN_PASSWORD`，并自动确保 `127.0.0.1`、`localhost` 进入 `NO_PROXY`，避免 Playwright 本地探活被代理转发。

若本机已有其他实例占用默认测试端口 `18188`，可通过环境变量覆盖：

- `ALTER0_PLAYWRIGHT_PORT=18189 npm run test:e2e`

## Helper 分层

### 组件

- `helpers/components/composer.ts`：统一基于 `data-composer-*` 稳定标识组装输入、提交、计数与表单 locator
- `helpers/components/session-list.ts`：统一会话列表 item、按值选择与删除按钮 locator 组装

### 页面与路由

- `helpers/pages/*.ts`：页面元素定位与 `create*Page()` page object 接口
- `helpers/pages/login.ts`：登录页输入与提交按钮
- `helpers/pages/app-shell.ts`：全局导航菜单等壳层控件
- `helpers/guards/app-ready.ts`：统一等待前端进入可交互态
- `helpers/guards/login.ts`：受保护页面登录守卫
- `helpers/flows/routes.ts`：统一聊天、Cron、Terminal 路由打开
- `helpers/flows/auth.ts`：统一解析登录密码，并为 API 请求补登录态

### 输入与交互

- `helpers/asserts/composer.ts`：统一 composer 可交互态、值、计数与状态属性断言
- `helpers/asserts/input.ts`：统一输入框可用态、焦点和值断言
- `helpers/interactions/ime.ts`：统一输入法合成开始与提交流程
- `helpers/guards/unsaved.ts`：统一未发送草稿确认守卫
- 复用输入组件统一暴露 `data-composer-*` 稳定标识，包括输入、提交、计数、草稿态、合成态与 pending 态
- Terminal 运行区统一暴露 `data-terminal-workspace-status`、`data-terminal-workspace-live` 等稳定状态标识；当前会话态固定为 `ready / busy / exited / interrupted`

### 场景能力

- `helpers/scenarios/chat.ts`：统一 Chat 路由打开、草稿装载、双会话草稿准备与刷新后的可交互态恢复
- `helpers/scenarios/cron.ts`：统一 Cron 路由打开与 composer 可交互态准备
- `helpers/scenarios/terminal.ts`：统一 Terminal 客户端绑定、会话预置、路由打开与可交互态准备
- `helpers/flows/chat-session.ts`：聊天会话切换、删除与激活态流程
- `helpers/flows/chat-draft.ts`：聊天草稿准备与双会话草稿场景
- `helpers/flows/terminal-session.ts`：Terminal 会话创建、选择与预置数据
- `helpers/flows/terminal-runtime.ts`：Terminal 轮询等待与重绘等待

## 当前覆盖

### Chat

- 路由切换前的未发送草稿确认
- 刷新后的草稿与字符计数恢复
- 多会话草稿隔离
- 删除会话后的草稿清理与剩余会话恢复
- 会话级草稿跨刷新恢复
- 取消切换或删除非当前会话时保持当前草稿
- 输入法合成期间按回车不提交

### Cron

- 输入法合成期间保持提示词输入稳定

### Terminal

- 从页面创建会话
- 轮询刷新期间保持焦点与草稿
- 输入法合成期间不打断输入
- 关闭后禁用输入
- 多终端会话草稿隔离
- 运行时不可用时将存量会话标记为 `interrupted`

### Shared Runtime Workspace

- `chat` 首次点击发送即提交当前草稿
- `chat / agent-runtime / terminal` 的正文滚动区与底部 Composer 保持独立边界，不出现正文被输入区遮挡

## 维护约束

- 新增文本输入场景时，优先接入现有 helper 分层
- 涉及登录保护的页面或接口测试，统一复用 `helpers/flows/auth.ts`、`helpers/guards/login.ts`、`helpers/guards/app-ready.ts` 与 `helpers/flows/routes.ts`
- 涉及草稿、焦点、输入法合成、会话切换的修改，优先补真实交互路径
- 避免在 E2E 中使用固定时间等待，优先等待可观测状态、轮询响应或页面重绘
- 所有测试产物统一输出到 `output/`
