# alter0 Web Frontend

`internal/interfaces/web/frontend` 是新的 Web 前端工程入口，负责把浏览器端构建、测试与静态产物发布收敛到统一工程目录。

当前阶段采用兼容桥切换：

- `index.html` 仅保留前端启动容器、字体与 legacy 样式入口
- React 负责渲染当前 Web Shell 的 legacy DOM 契约，并启动兼容桥接层
- `features/shell` 当前已把主导航当前路由高亮、导航折叠态与语言感知文案，以及 Session Pane、ChatWorkspace 头部动作区、Session 历史空态提示/可访问标签、路由页头部标题/副标题和主工作区的历史折叠、菜单、会话抽屉、`page-mode / data-route / chatView / routeView` 路由壳层状态前移到 React；会话列表、消息区、route body、runtime host 与 `sessionHeading / sessionSubheading` 标题挂载点继续保持静态 DOM 契约，避免桥接期 rerender 清空 legacy runtime 注入内容
- `LegacyWebShell` 会同步保留 `appShell` 上由 legacy runtime 直接切换的 transient classes，如 `nav-open`、`panel-open`、`overlay-open` 与 `runtime-sheet-open`，避免 React 因 hash 路由或语言变化重渲染时覆盖移动端当前打开的壳层状态
- `public/legacy/chat.js` 当前不再负责回写 `newChatButton`、`sessionToggle`、`mobileNewChatButton`、`routeTitle`、`routeSubtitle`、`sessionEmpty` 和 `sessionList` 可访问标签的显示文案，也不再切换 `info-mode`、`page-mode`、`data-route` 和 `chatView / routeView` 显隐；legacy runtime 仅保留这些节点的事件绑定、路由加载与业务流转，避免与 React 壳层发生状态竞争
- 旧版运行时脚本与样式通过 `public/legacy` 纳入 Vite 构建产物，并作为唯一 legacy 资源来源输出到 `static/dist/legacy`
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
