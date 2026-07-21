# alter0 Web Frontend

`internal/interfaces/web/frontend` 是新的 Web 前端工程入口，负责把浏览器端构建、测试与静态产物发布收敛到统一工程目录。

前端页面的视觉与实现约束统一维护在 [docs/frontend-design-guidelines.md](../../../../docs/frontend-design-guidelines.md)，涉及页面、组件、布局或交互调整时需一并遵守。

当前稳定结构如下：

- `index.html` 仅保留前端启动容器、字体与 legacy 样式入口
- `src/app` 承载单一 React 工作台壳层：`WorkbenchApp` 负责 `/chat`、`/settings` 与登录相关 canonical path 路由、语言切换、主导航折叠/抽屉和Chat 工作区/控制页分派；`/chat runtime` 不再作为工作台 route 暴露。`routeState.ts` 负责路由解析、派发与 `session_id` 短 hash query 协调，`WorkbenchContext.tsx` 暴露当前 route/language/navigate 以及移动端主导航状态
- `features/conversation-runtime` 承载 Chat 对话运行态 UI：`ConversationRuntimeProvider` 复用 runtime session 数据模型和状态机，并通过 `/api/chat/sessions` 负责会话、输入提交、会话恢复轮询、草稿恢复、短 hash URL 恢复和模型/能力项选择。运行态只允许共享 runtime controller 提交会话状态，旧缓存和迟到请求不能反向覆盖已接受的新详情；`conversationSyncPolicy.ts` 维护 5 分钟前后台阈值、会话切换详情判定、请求代次、update 去重和内容缓存保留集合。24 小时浏览器缓存保留全部轻量会话信息，但完整内容只保留当前会话与最近 4 个会话，并剔除附件 data URL 与按需事件详情块。`ConversationWorkspace` 接入共享 `RuntimeWorkspacePage` 组装「会话列 + 主时间线工作区 + Composer + Inspector」Chat 工作区，并在窄屏下统一输出 `Menu / 标题 / New` 操作行；不再渲染 Skill 选择器、Deliverables、Session Profile 或独立 Skill 面板
- `features/shell` 负责主导航、共享 copy、Chat 工作区骨架与 React 管理页：`components/RuntimeWorkspacePage.tsx` 提供Chat 工作区骨架、导航持有的会话列表、工作区 body 与 slot 化头部/正文/底部区域，并把当前Chat 工作区的 count、列表 body 与 `New` 动作绑定给主导航；`PrimaryNav.tsx` 稳定持有 `Sessions / New` 公共 chrome，`WorkbenchApp.tsx` 在Chat 工作区 rail 数据尚未注册时提供禁用态 `New` fallback rail，并缓存最近一次有效 rail body；`RuntimeWorkspaceHeader.tsx` 统一输出Chat 工作区的标题、状态按钮与 `Details` 按钮；`ReactManagedRouteBody` 分派 Settings 管理页，其中 Skills 分区消费 `/api/control/skill-catalog` 并分别展示 Alter0 内置与 Codex 只读目录
- 根壳层通过 `app-shell[data-workbench-route]` 暴露当前路由，Chat 工作区继续通过 `data-route / data-conversation-*` 暴露稳定锚点；`legacy` 资源仅保留兼容样式，不再通过 `/legacy/chat.js`、bridge 或 snapshot store 驱动业务运行时
- `public/legacy` 当前仅保留兼容样式资源，并作为 legacy 样式来源输出到 `static/dist/legacy`
- `npm run build` 输出到 `internal/interfaces/web/static/dist`；服务二进制的正式构建入口为仓库根目录的 `scripts/build_alter0_service.sh` / `make build`，会先运行该前端构建再执行 Go 构建
- `npm run dev` 默认把 `/api`、`/login`、`/logout`、`/healthz`、`/readyz`、`/metrics` 代理到 `http://127.0.0.1:18088`；可通过 `ALTER0_WEB_BACKEND_ORIGIN` 覆盖后端地址
- `src/shared/api/client.ts` 提供统一 JSON 请求、`204` 空响应、结构化错误与 `401` 登录失效钩子，后续 React 页面迁移统一复用该入口
- `src/shared/time/format.ts` 提供统一北京时间格式化与默认时区常量，后续 Chat / Task / Cron 页面统一复用该入口
- `src/shared/time/sessionListGroups.ts` 提供Chat 会话列表的最近时间分组逻辑，供 Chat 复用
- `src/shared/viewport/mobileViewport.ts` 提供移动端断点、键盘阈值与 viewport baseline 纯计算逻辑，供 Chat 的 viewport driver 与 hook 复用

常用命令：

- `npm install`
- `npm run dev`
- `npm run build`
- `npm run test`

开发态联调：

- 运行时一：`ALTER0_WEB_FRONTEND_DEV_ORIGIN=http://127.0.0.1:5173 go run ./cmd/alter0`
- 运行时二：`ALTER0_WEB_BACKEND_ORIGIN=http://127.0.0.1:18088 npm run dev`
- 浏览器既可以直接访问 Go 服务的 `http://127.0.0.1:18088/chat`，由 Go 反向代理到 Vite dev server；也可以直接访问 Vite 的 `http://127.0.0.1:5173/chat`，由 Vite 把后端请求代理回 Go 服务
