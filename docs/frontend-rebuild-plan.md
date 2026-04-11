# Frontend Rebuild Plan

> Last update: 2026-04-11

## 目标

本方案定义 `alter0` Web 前端的一次性彻底重构路径。目标不是继续在现有原生静态前端上叠加修补，而是用新的前端工程体系完整替换当前 `chat.html + chat.js + chat-*.css` 实现，并在一次切换后成为唯一稳定前端。

重构完成后，Web 前端应满足以下目标：

1. 使用现代组件化前端框架，替换当前单文件原生 SPA。
2. 保留现有 Go 后端与稳定 API，不重写主业务后端。
3. 支持 Chat、Agent、Terminal、Control、Product 等全部现有 Web 能力。
4. 把移动端视口、桌面宽屏、SSE、轮询、Terminal turn/step、运行时设置等高复杂交互收敛到可维护架构。
5. 建立组件测试、集成测试、E2E 测试三级回归体系，替代当前高度依赖端到端兜底的模式。
6. 旧前端在新前端完成切换后完全下线，不保留长期双实现。

## 范围

本次重构覆盖以下内容：

1. `internal/interfaces/web/static/chat.html`
2. `internal/interfaces/web/static/assets/chat.js`
3. `internal/interfaces/web/static/assets/chat-core.css`
4. `internal/interfaces/web/static/assets/chat-routes.css`
5. `internal/interfaces/web/static/assets/chat-terminal.css`
6. 由上述静态资源驱动的 Chat、Agent、Terminal、Control、Product、Memory、Sessions、Tasks、Models、Environments 等全部 Web 界面
7. Go Web 静态资源分发方式
8. Web 端登录态、SSE、轮询、路由切换、视口管理、前端缓存、时间格式化、运行时设置面板、移动端 bottom sheet、Terminal 交互层

不在本次重构中重写以下内容：

1. `internal/orchestration`、`internal/execution`、`internal/task`、`internal/terminal` 等核心后端领域逻辑
2. 现有稳定 API 的业务语义
3. 登录鉴权模型
4. Go 侧 Task、Terminal、Session、Product 的领域数据结构

## 技术选型

### 前端基础栈

1. 框架：`React 19 + TypeScript`
2. 构建工具：`Vite`
3. 路由：`TanStack Router`
4. 服务端数据：`TanStack Query`
5. 全局客户端状态：`Zustand`
6. 表单：`react-hook-form + zod`
7. 组件测试：`Vitest + React Testing Library`
8. E2E：`Playwright`
9. 样式：`CSS Modules + 设计令牌（CSS variables）`

### 状态建模策略

1. 页面级远程数据统一走 Query，不允许在组件中零散 `fetch`
2. 会话激活态、侧边栏开合、移动端视口、运行时设置面板、Terminal 交互态统一收敛到 feature store
3. Chat SSE、Terminal 轮询、移动端键盘/VisualViewport 采用独立 driver 层，不直接散落到页面组件里
4. 高复杂交互优先使用显式状态流，不允许继续用 DOM 即状态的方式补丁式维护

### 不采用的方案

1. 不采用 `Next.js / Nuxt / Remix` 等 SSR 全家桶
2. 不采用“继续用原生 JS，但拆更多文件”的折中方案
3. 不长期维持新旧两套前端并存

## 最终目标架构

```text
internal/interfaces/web/
  frontend/
    src/
      app/
        providers/
        router/
        store/
        bootstrap/
      features/
        shell/
        auth/
        chat/
        agent-runtime/
        terminal/
        sessions/
        tasks/
        products/
        memory/
        skills/
        mcp/
        channels/
        models/
        environments/
        cron-jobs/
      shared/
        api/
        sse/
        polling/
        viewport/
        i18n/
        time/
        markdown/
        components/
        layout/
        styles/
        testing/
        utils/
    index.html
    vite.config.ts
    tsconfig.json
    package.json
  static/
    dist/
```

Go 侧最终形态：

1. embed `static/dist/` 编译产物，而不是直接 embed 手写 HTML/CSS/JS
2. Web handler 负责：
   - 登录页与登录态检查
   - API
   - SSE
   - 前端编译产物分发
   - SPA fallback

## 重构原则

1. 一次切换，以新前端替换旧前端，不做长期增量并存
2. 保持后端 API 契约优先稳定，前端适配当前 API
3. 所有页面都在统一 App Shell 中运行，不允许重复实现侧边栏、会话栏、底部输入栏等基础壳层
4. Chat 与 Terminal 为最高优先级核心区，必须按独立 feature 彻底拆分
5. 设计令牌统一管理断点、阴影、圆角、层级、主内容宽度，禁止散落硬编码
6. 先建测试骨架，再落业务页面，最后切换
7. 切换前必须完成全量回归，不允许以“后续补测试”方式上线

## 工作分解结构

## 1. 项目基建与切换准备

### 1.1 新前端工程初始化

任务：

1. 创建 `internal/interfaces/web/frontend`
2. 初始化 `package.json`、`tsconfig.json`、`vite.config.ts`
3. 配置 React、TypeScript、ESLint、Prettier、Vitest
4. 配置构建输出到 `internal/interfaces/web/static/dist`
5. 配置开发态代理到 Go Web 服务

交付物：

1. 可运行的 Vite React 工程
2. `npm run dev`、`npm run build`、`npm run test`、`npm run test:e2e`

验收：

1. 本地能启动空白 React App
2. Go 服务能分发编译产物

### 1.2 前端工程规范落地

任务：

1. 定义 feature 目录结构
2. 定义组件、hook、api、store、types 命名规范
3. 定义样式模块命名规范
4. 定义 Query key、错误处理、日志上报约定

交付物：

1. `frontend/README.md`
2. 工程 lint 与目录规范

验收：

1. 新增 feature 时无需再讨论基础目录结构

## 2. 设计系统与壳层重建

### 2.1 设计令牌

任务：

1. 提炼色板、间距、圆角、阴影、层级、断点、内容宽度变量
2. 统一 `desktop / tablet / narrow / handset` 断点口径
3. 统一消息主列、Composer、Terminal 工作区最大宽度策略

交付物：

1. `shared/styles/tokens.css`
2. 断点与内容宽度设计规范

验收：

1. 不再在 feature 样式中硬编码 `860px / 760px / 420px`

### 2.2 App Shell

任务：

1. 重建主导航
2. 重建会话侧栏容器
3. 重建主内容容器
4. 重建移动端 overlay、nav drawer、panel drawer
5. 统一页头和 route 容器

交付物：

1. `features/shell`
2. App Shell Story/测试页

验收：

1. 桌面、平板、手机三档下壳层稳定
2. Chat/Terminal/Control 页面切换不影响壳层状态

## 3. 共享基础能力

### 3.1 API Client

任务：

1. 统一 JSON 请求封装
2. 统一错误码与错误消息处理
3. 统一登录失效处理

交付物：

1. `shared/api/client.ts`

### 3.2 SSE Client

任务：

1. 封装 Chat/Agent 流式连接
2. 支持 `start / delta / process / done / keepalive / error`
3. 统一重试、取消和断流收口

交付物：

1. `shared/sse/chat-stream.ts`

### 3.3 轮询与实时刷新驱动

任务：

1. 封装 Terminal 轮询
2. 封装 Task 轮询
3. 支持页面可见性降频
4. 支持输入聚焦与滚动活跃降频

交付物：

1. `shared/polling/*`

### 3.4 视口与键盘管理

任务：

1. 重建 VisualViewport 管理器
2. 区分桌面窄窗、平板、小屏手机
3. 管理移动端 bottom sheet、keyboard inset、App Shell 高度
4. 统一 Chat/Terminal 共用视口状态

交付物：

1. `shared/viewport/*`

验收：

1. 不再在多个 feature 内重复维护 viewport 逻辑

## 4. Chat 与 Agent Runtime 重建

### 4.1 Chat 数据模型

任务：

1. 建立 Session、Message、ProcessStep、RuntimeSelection 前端类型
2. 收敛本地缓存格式
3. 收敛 active session、draft、runtime selection、target selection

交付物：

1. `features/chat/types.ts`
2. `features/chat/store/*`

### 4.2 Chat 页面组件化

任务：

1. 会话列表组件
2. 消息列表组件
3. 用户消息组件
4. 助手消息组件
5. Process 组件
6. Composer 组件
7. Runtime settings 组件
8. Empty state / Welcome 组件

交付物：

1. `features/chat/components/*`

### 4.3 Chat 交互重建

任务：

1. 新建会话、空白会话唯一性
2. 自动标题升级
3. 草稿持久化
4. SSE 流式消息更新
5. 断流失败态恢复
6. 运行时设置面板
7. Agent 目标切换
8. 移动端 session settings bottom sheet
9. 桌面宽屏消息列与 Composer 对齐

验收：

1. 当前 Chat/Agent Web 行为全量兼容
2. 大屏、移动端、断流、空白会话、草稿、多会话回归全部通过

## 5. Terminal 重建

### 5.1 Terminal 数据模型

任务：

1. 建立 Terminal session、turn、step、detail、draft 类型
2. 建立轮询状态与滚动状态模型
3. 建立移动端 session sheet 状态模型

### 5.2 Terminal 页面组件化

任务：

1. 会话列表
2. 工作区头部
3. turn 列表
4. process 面板
5. step detail
6. 输入区
7. jump controls
8. 空态

### 5.3 Terminal 行为重建

任务：

1. 创建、恢复、关闭、删除会话
2. 轮询更新 turn/step
3. 输入聚焦时保护 DOM 不重建
4. 移动端 session sheet
5. 宽屏、窄屏、真手机宽度布局
6. jump navigation
7. 复制最终输出

验收：

1. Terminal 当前行为全量兼容
2. 移动端与桌面回归全部通过

## 6. Control 与 Product 页面重建

### 6.1 低复杂度页面

任务：

1. Channels
2. Skills
3. MCP
4. Models
5. Environments
6. Cron Jobs

实现要求：

1. 统一表单层
2. 统一列表、详情、保存、错误提示
3. 统一 loading / empty / error 状态

### 6.2 中高复杂度页面

任务：

1. Sessions
2. Tasks
3. Products
4. Agent Profiles
5. Memory

实现要求：

1. 列表与详情拆分
2. 统一分页、筛选、复制、刷新策略

## 7. Go Web 分发层改造

### 7.1 静态资源分发改造

任务：

1. embed `static/dist`
2. 增加 SPA fallback
3. 增加静态资源缓存头策略
4. 保留登录页和 API handler

### 7.2 开发态联调支持

任务：

1. 开发模式下支持 Vite dev server
2. 本地可同时跑 Go API 和前端 dev server
3. 文档化开发命令

## 8. 测试体系重建

### 8.1 单元测试

任务：

1. 基础组件测试
2. 表单组件测试
3. Markdown 渲染测试
4. 时间格式化测试
5. viewport hook 测试

### 8.2 集成测试

任务：

1. Chat store + SSE 集成测试
2. Terminal polling 集成测试
3. Runtime settings 状态切换测试
4. Session draft/restore 测试

### 8.3 E2E 测试

任务：

1. 迁移现有 Playwright 用例
2. 保留移动端键盘/VisualViewport 回归
3. 保留桌面宽屏回归
4. 增补 Control、Product、Task 核心路径

最终验收门槛：

1. `go test ./...`
2. 前端单元与集成测试全绿
3. Playwright 核心场景全绿

## 9. 数据与兼容策略

### 9.1 API 兼容

原则：

1. 现有 API 路径和语义保持稳定
2. 前端优先适配后端，不在本次重构中大规模改 API
3. 若确需补接口，只允许做向后兼容增强

### 9.2 本地缓存兼容

原则：

1. 新前端重新定义本地缓存 schema
2. 切换时允许一次性迁移旧缓存
3. 迁移失败时直接清空前端缓存并从服务端恢复，不保留旧脚本兼容层

## 10. 切换策略

### 10.1 冻结规则

切换期间执行以下规则：

1. 旧前端停止新增功能
2. 旧前端只允许阻断级缺陷修复
3. 新需求默认落到新前端，不再给旧前端追加结构性能力

### 10.2 切换步骤

1. 新前端功能完成
2. 测试全绿
3. 预发环境联调
4. 静态资源切换到 `dist`
5. 删除旧前端入口引用
6. 下线旧前端静态文件

### 10.3 删除清单

切换完成后需删除：

1. `static/chat.html`
2. `static/assets/chat.js`
3. `static/assets/chat.css`
4. `static/assets/chat-core.css`
5. `static/assets/chat-routes.css`
6. `static/assets/chat-terminal.css`
7. 与旧静态前端直接耦合的无效测试与辅助逻辑

## 11. 任务排期与并行拆分

## 阶段 A：基建与壳层

负责人组：

1. 前端基建
2. 设计系统
3. Go Web 分发适配

完成标志：

1. 新工程可构建
2. App Shell 可运行
3. Go 可分发 `dist`

## 阶段 B：Chat/Agent/Terminal 核心区

负责人组：

1. Chat/Agent Runtime
2. Terminal
3. 共享 viewport / stream / polling

完成标志：

1. Chat 与 Terminal 主链路可用
2. 现有关键 Playwright 回归迁移完成

## 阶段 C：Control 与 Product

负责人组：

1. Control 表单页
2. Product/Memory/Tasks/Sessions

完成标志：

1. 旧前端所有页面在新前端中均有对应实现

## 阶段 D：测试、联调、切换

负责人组：

1. 测试治理
2. 联调验收
3. 切换与清理

完成标志：

1. 全量回归通过
2. 生产切换完成
3. 旧前端删除完成

## 12. 最终完成定义

满足以下条件时，重构才算真正完成：

1. 新前端已成为唯一 Web 前端实现
2. 旧静态前端代码已从仓库删除
3. Chat、Agent、Terminal、Control、Product 页面全部迁移完毕
4. Go 服务只分发新前端编译产物
5. 现有稳定 API 兼容保留
6. 桌面宽屏、移动端键盘、Terminal 轮询、SSE、会话恢复等高风险场景均有自动化回归
7. 文档、开发命令、测试命令、部署方式全部更新完成

## 13. 建议执行顺序

建议严格按以下顺序推进：

1. 新前端基建
2. App Shell
3. 共享层：API、SSE、viewport、polling
4. Chat / Agent Runtime
5. Terminal
6. Control 页面
7. Product / Sessions / Tasks / Memory
8. 测试收口
9. Go 分发切换
10. 删除旧前端

## 14. 免测说明

本次提交仅新增重构方案文档与索引，不涉及运行时代码与用户行为变更，因此不新增自动化测试。文档有效性通过与当前仓库结构、现有 Web 静态资源组织和现有测试体系对照校验。
