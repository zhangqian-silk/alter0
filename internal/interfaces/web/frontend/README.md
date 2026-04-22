# alter0 Web Frontend

`internal/interfaces/web/frontend` 是新的 Web 前端工程入口，负责把浏览器端构建、测试与静态产物发布收敛到统一工程目录。

前端页面的视觉与实现约束统一维护在 [docs/frontend-design-guidelines.md](../../../../docs/frontend-design-guidelines.md)，涉及页面、组件、布局或交互调整时需一并遵守。

当前稳定结构如下：

- `index.html` 仅保留前端启动容器、字体与 legacy 样式入口
- `src/app` 承载单一 React 工作台壳层：`WorkbenchApp` 负责 hash 路由、语言切换、主导航折叠/抽屉与运行页/控制页分派，`routeState.ts` 负责路由解析与派发，`WorkbenchContext.tsx` 暴露当前 route/language/navigate 以及移动端主导航状态
- `features/conversation-runtime` 承载 `chat / agent-runtime` 的运行态：`ConversationRuntimeProvider` 负责会话、消息流、SSE 收口、任务轮询、草稿恢复和模型/能力项选择，`ConversationWorkspace` 通过共享 `RuntimeWorkspaceFrame` 组装 terminal-style 的「会话列 + 主时间线工作区 + Composer + Inspector」视图，并在窄屏下提供 `Menu / Sessions / New` 操作行与会话抽屉
- `features/shell` 负责主导航、共享 copy、运行页骨架与 React 管理页：`components/RuntimeWorkspaceFrame.tsx` 提供 `chat / agent-runtime / terminal` 复用的运行页骨架、侧栏/backdrop、工作区 body 与 slot 化头部/正文/底部区域；`ReactManagedRouteBody` 统一分派 `agent / terminal / memory / channels / skills / mcp / models / environments / cron-jobs / sessions / tasks / products / codex-accounts`，其中 `ReactManagedTerminalRouteBody` 通过该骨架输出 `Menu / Sessions / New` 顶部操作行并接入 workbench 导航状态
- 根壳层通过 `app-shell[data-workbench-route]` 暴露当前路由，运行页继续通过 `data-route / data-conversation-*` 暴露稳定锚点；`legacy` 资源仅保留兼容样式，不再通过 `/legacy/chat.js`、bridge 或 snapshot store 驱动业务运行时
- `public/legacy` 当前仅保留兼容样式资源，并作为 legacy 样式来源输出到 `static/dist/legacy`
- `npm run build` 输出到 `internal/interfaces/web/static/dist`
- `npm run dev` 默认把 `/api`、`/login`、`/logout`、`/healthz`、`/readyz`、`/metrics` 与 `/products` 代理到 `http://127.0.0.1:18088`；可通过 `ALTER0_WEB_BACKEND_ORIGIN` 覆盖后端地址
- `src/shared/api/client.ts` 提供统一 JSON 请求、`204` 空响应、结构化错误与 `401` 登录失效钩子，后续 React 页面迁移统一复用该入口
- `src/shared/time/format.ts` 提供统一北京时间格式化与默认时区常量，后续 Chat / Terminal / Task / Cron 页面统一复用该入口
- `src/shared/viewport/mobileViewport.ts` 提供移动端断点、键盘阈值与 viewport baseline 纯计算逻辑，后续 Chat / Terminal 的 viewport driver 与 hook 统一复用该入口

常用命令：

- `npm install`
- `npm run dev`
- `npm run build`
- `npm run test`

开发态联调：

- 终端一：`ALTER0_WEB_FRONTEND_DEV_ORIGIN=http://127.0.0.1:5173 go run ./cmd/alter0`
- 终端二：`ALTER0_WEB_BACKEND_ORIGIN=http://127.0.0.1:18088 npm run dev`
- 浏览器既可以直接访问 Go 服务的 `http://127.0.0.1:18088/chat`，由 Go 反向代理到 Vite dev server；也可以直接访问 Vite 的 `http://127.0.0.1:5173/chat`，由 Vite 把后端请求代理回 Go 服务
