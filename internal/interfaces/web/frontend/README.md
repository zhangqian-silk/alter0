# alter0 Web Frontend

`internal/interfaces/web/frontend` 是新的 Web 前端工程入口，负责把浏览器端构建、测试与静态产物发布收敛到统一工程目录。

前端页面的视觉与实现约束统一维护在 [docs/frontend-design-guidelines.md](../../../../docs/frontend-design-guidelines.md)，涉及页面、组件、布局或交互调整时需一并遵守。

当前稳定结构如下：

- `index.html` 仅保留前端启动容器、字体与 legacy 样式入口
- `src/app` 承载单一 React 工作台壳层：`WorkbenchApp` 负责 `/chat`、`/terminal` 与其他控制页 canonical path 路由、语言切换、主导航折叠/抽屉和运行页/控制页分派；`/chat` 仅作为旧 URL alias 映射到 `/chat`。`routeState.ts` 负责路由解析、派发与 `session_id` 短 hash query 协调，`WorkbenchContext.tsx` 暴露当前 route/language/navigate 以及移动端主导航状态
- `features/conversation-runtime` 承载 `chat` 运行态：`ConversationRuntimeProvider` 复用 Terminal session API 的数据模型和状态机，并通过 `scope=chat` 使用独立会话存储来负责会话、输入提交、会话恢复轮询、草稿恢复、短 hash URL 恢复和模型/能力项选择；历史 `chat` 本地 snapshot 与 Terminal session `turns` 在加载时迁移到当前 Chat 会话结构，不再维护 Chat 专用消息发送链路。`ConversationRuntimeProvider` 额外维护同一 SPA 工作台内的 8 小时运行态内存缓存，用于从其他路由返回 Chat 时先恢复未过期的会话列表、活动会话和最新消息，再由接口响应合并更新。`ConversationWorkspace` 接入共享 `RuntimeWorkspacePage` 组装「会话列 + 主时间线工作区 + Composer + Inspector」运行页，并在窄屏下统一输出 `Menu / 标题 / New` 操作行；不再渲染 Skill 选择器、Deliverables、Session Profile 或独立 Skill 面板
- `features/shell` 负责主导航、共享 copy、运行页骨架与 React 管理页：`components/RuntimeWorkspacePage.tsx` 提供 `chat / terminal` 复用的运行页骨架、导航持有的会话列表、工作区 body 与 slot 化头部/正文/底部区域，并把当前运行页的 count、列表 body 与 `New` 动作绑定给主导航；`PrimaryNav.tsx` 稳定持有 `Sessions / New` 公共 chrome，`WorkbenchApp.tsx` 在运行页 rail 数据尚未注册时提供禁用态 `New` fallback rail，并按 route 缓存最近一次有效 rail body；`RuntimeWorkspaceHeader.tsx` 统一输出运行页的标题、状态按钮与 `Details` 按钮，Terminal 只注入状态值和详情内容；`ReactManagedTerminalRouteBody.tsx` 额外维护同一 SPA 工作台内的 8 小时运行态内存缓存，用于从其他路由返回 Terminal 时先恢复未过期的会话列表和最新 turns，再由接口响应合并更新；`ReactManagedRouteBody` 统一分派管理页和 Terminal/Skill Studio 等功能页；Terminal 失败、退出与附件错误提示进入共享 Composer 工具栏 meta，不再额外增加外层 note row
- 根壳层通过 `app-shell[data-workbench-route]` 暴露当前路由，运行页继续通过 `data-route / data-conversation-*` 暴露稳定锚点；`legacy` 资源仅保留兼容样式，不再通过 `/legacy/chat.js`、bridge 或 snapshot store 驱动业务运行时
- `public/legacy` 当前仅保留兼容样式资源，并作为 legacy 样式来源输出到 `static/dist/legacy`
- `npm run build` 输出到 `internal/interfaces/web/static/dist`；服务二进制的正式构建入口为仓库根目录的 `scripts/build_alter0_service.sh` / `make build`，会先运行该前端构建再执行 Go 构建
- `npm run dev` 默认把 `/api`、`/login`、`/logout`、`/healthz`、`/readyz`、`/metrics` 代理到 `http://127.0.0.1:18088`；可通过 `ALTER0_WEB_BACKEND_ORIGIN` 覆盖后端地址
- `src/shared/api/client.ts` 提供统一 JSON 请求、`204` 空响应、结构化错误与 `401` 登录失效钩子，后续 React 页面迁移统一复用该入口
- `src/shared/time/format.ts` 提供统一北京时间格式化与默认时区常量，后续 Chat / Terminal / Task / Cron 页面统一复用该入口
- `src/shared/time/sessionListGroups.ts` 提供运行页会话列表的最近时间分组逻辑，供 Chat / Terminal 统一复用
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
