# alter0 Web Frontend

`internal/interfaces/web/frontend` 是新的 Web 前端工程入口，负责把浏览器端构建、测试与静态产物发布收敛到统一工程目录。

当前稳定结构如下：

- `index.html` 仅保留前端启动容器、字体与 legacy 样式入口
- React 负责渲染当前 Web Shell 的 legacy DOM 契约与完整运行时，不再通过 `/legacy/chat.js` 启动旧脚本
- `features/shell` 当前已把主导航当前路由高亮、导航折叠态、导航 tooltip 与语言感知文案，以及 Session Pane、会话卡片列表、ChatWorkspace 头部动作区、欢迎区文案、欢迎区 target picker、欢迎区/消息区显隐、消息列表 DOM、运行时 controls/note/sheet 原生渲染、Session 历史空态提示/可访问标签、路由页头部标题/副标题和主工作区的历史折叠、菜单、会话抽屉、`page-mode / data-route / chatView / routeView` 路由壳层状态前移到 React；`agent / terminal / memory / channels / skills / mcp / models / environments / cron-jobs / sessions / tasks / products` 十二类路由页已由 `ReactManagedRouteBody` 接管，terminal 也已改为 React 原生实现；`routeBody` 会继续暴露当前页的 `data-react-managed-route` ownership 标记，同时 `appShell` 会输出稳定的 `data-react-managed-routes` 路由清单，避免壳层 rerender 清空页面主体
- `ReactRuntimeFacade` 负责 Chat / Agent 的会话创建、切换、删除、消息流、草稿恢复、runtime panel/sheet 状态、移动端 runtime sheet 叠层、共享 runtime API 与 `alter0:legacy-shell:*` bridge 事件处理
- `legacyRuntimeSnapshotStore` 继续作为共享结构化状态通道，供壳层读取 chat workspace、session pane、message region 与 chat runtime 四类快照；这些快照由 React runtime 直接发布，不再依赖 document 级 CustomEvent 广播
- `LegacyWebShell` 会同步保留 `appShell` 上的 transient classes，如 `nav-open`、`panel-open`、`overlay-open` 与 `runtime-sheet-open`，避免 React 因 hash 路由或语言变化重渲染时覆盖移动端当前打开的壳层状态
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
