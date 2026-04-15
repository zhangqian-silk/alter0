# alter0 Web Frontend

`internal/interfaces/web/frontend` 是新的 Web 前端工程入口，负责把浏览器端构建、测试与静态产物发布收敛到统一工程目录。

当前阶段采用兼容桥切换：

- `index.html` 仅保留前端启动容器、字体与 legacy 样式入口
- React 负责渲染当前 Web Shell 的 legacy DOM 契约，并启动兼容桥接层
- `features/shell` 当前已把主导航当前路由高亮、导航折叠态、导航 tooltip 与语言感知文案，以及 Session Pane、会话卡片列表、ChatWorkspace 头部动作区、欢迎区文案、欢迎区 target picker、欢迎区/消息区显隐、消息列表 DOM、运行时 controls/note/sheet DOM、Session 历史空态提示/可访问标签、路由页头部标题/副标题和主工作区的历史折叠、菜单、会话抽屉、`page-mode / data-route / chatView / routeView` 路由壳层状态前移到 React；其中 `agent / terminal / memory / channels / skills / mcp / models / environments / cron-jobs / sessions / tasks / products` 十二类路由页已由 `ReactManagedRouteBody` 接管，其中 terminal 通过 React-managed host 挂载 legacy terminal controller，其余路由直接请求控制台或会话 API 并渲染；`routeBody` 会同步暴露 `data-react-managed-route` ownership 标记，避免桥接期 rerender 清空 legacy runtime 注入内容
- `features/shell` 现已由 React 接管导航抽屉、会话抽屉、关闭按钮与移动端遮罩的开合控制；legacy runtime 不再绑定这些壳层按钮事件，继续负责路由加载、会话业务流转与运行时快照计算
- `features/shell` 现在还会通过 `alter0:legacy-shell:*` bridge 事件把主导航跳转、新建会话、欢迎区快捷提示、会话聚焦、会话删除、语言切换、导航折叠同步与会话历史折叠同步交给 legacy runtime 处理；React 成为壳层按钮与快捷入口的唯一事件来源，legacy runtime 不再直接给这些按钮绑定重复监听
- `features/shell` 现在还会消费 legacy runtime 发出的 chat workspace snapshot bridge、session pane snapshot bridge、message region snapshot bridge 与 chat runtime snapshot bridge，用于渲染当前会话标题、副标题、欢迎区描述、欢迎区 target picker、欢迎区/消息区显隐、运行时 controls/note/sheet，以及会话卡片列表、消息列表 HTML、空态/加载错误提示与 runtime sheet 滚动位置；legacy runtime 负责计算业务态文案、消息 HTML 与运行时 HTML，React 负责稳定渲染节点，不再直接改写 `sessionHeading`、`sessionSubheading`、`welcomeHeading`、`welcomeDescription`、`welcomeTargetList`、`welcomeScreen`、`messageArea`、`sessionList`、`sessionEmpty`、`sessionLoadError`、`[data-runtime-controls-root]`、`[data-runtime-note-root]` 与 `[data-runtime-sheet-root]`
- `LegacyWebShell` 会同步保留 `appShell` 上由 legacy runtime 直接切换的 transient classes，如 `nav-open`、`panel-open`、`overlay-open` 与 `runtime-sheet-open`，避免 React 因 hash 路由或语言变化重渲染时覆盖移动端当前打开的壳层状态
- `public/legacy/chat.js` 当前不再负责回写 `newChatButton`、`sessionToggle`、`mobileNewChatButton`、`routeTitle`、`routeSubtitle`、`sessionEmpty` 和 `sessionList` 可访问标签的显示文案，也不再维护主导航 tooltip、直接改写 chat workspace 头部/欢迎区文案、欢迎区 target picker、欢迎区/消息区显隐、会话卡片列表、消息列表 DOM、运行时 controls/note/sheet DOM，以及 `agent / products / memory / channels / skills / mcp / models / environments / cron-jobs / sessions / tasks` 十一类路由页 DOM；terminal 路由主体也不再走 `renderRoute(routeBody)` 页面加载链，而是通过 React-managed host 调用 legacy terminal mount API 完成挂载。legacy runtime 仅保留业务路由加载、会话流转、workspace/session pane/message region/chat runtime 计算与 runtime / welcome target picker 交互委托，并通过 `routeBody[data-react-managed-route="true"]` 判断页面主体是否已由 React 接管，避免与 React 壳层发生状态竞争
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
