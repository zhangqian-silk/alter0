# Workbench UI Polish Implementation

> Last update: 2026-04-25

## 目标

本说明定义 `alter0` PC 端工作台参考图精修的落地范围、实现顺序、文件边界与验收口径。目标是在不改变现有 React 工作台信息架构和运行契约的前提下，吸收参考图中的清爽浅色控制台质感、稳定三栏布局、精致空态与紧凑 Composer 工具栏。

本轮优化保持以下稳定边界：

1. 不重写 Web Shell 架构。
2. 不改变 `/chat`、`chat / agent-runtime / terminal` 路由与 API 语义。
3. 不恢复 legacy runtime 脚本接管。
4. 不引入胶囊按钮、胶囊标签或 `border-radius: 999px` 控件。
5. 不破坏 `1100px` 与 `760px` 两档窄屏断点、移动端抽屉、Composer 贴底和 `VisualViewport` 键盘链路。

## 设计吸收原则

参考图可吸收的是工作台层级、视觉密度和控件组织方式，不直接复制为营销化界面。

1. 主界面继续是高密度工作台，不引入 hero 页面、宣传文案区或装饰卡片墙。
2. PC 端采用更干净的浅色底、弱蓝灰边框、轻阴影和 8-14px 低圆角。
3. 空态允许使用细网格、浅色弧线和低对比纹理增强空间感，但不得干扰消息阅读或输入。
4. 主要操作保持明确、克制和可扫描：导航、会话卡、工作区头部、Composer 工具栏各自承担独立职责。
5. 图标服务于识别与操作，不替代核心文本，也不增加无意义装饰。

## 落地范围

## Chat / Agent / Terminal 复用边界

当前三类运行页已经共用 React 工作台主骨架，不属于三套页面各自实现。`chat` 与 `agent-runtime` 通过 `ConversationWorkspace` 生成 `RuntimeWorkspacePageController`，`terminal` 通过 `ReactManagedTerminalRouteBody` 生成同一类型的 controller，最终都进入 `RuntimeWorkspacePage`，共享会话列、工作区头部、消息区、Details 和 Composer 的基础布局。

移动端视觉高度一致是预期结果。窄屏下三类页面统一折叠为单列工作台，并复用 `Menu / Sessions / New`、移动端抽屉、Composer fixed 贴底和 `VisualViewport` 键盘约束。移动端不应为了强调页面差异重新引入独立壳层。

PC 端仍存在差异，主要来自两类来源：

1. 合理的领域差异：Terminal 需要展示 turn、process step、命令输出、跳转控件、运行状态和最终输出，这些内容可以继续使用 `terminal-*` 领域 class。
2. 需要收敛的表现差异：会话卡、Header 右侧操作、空态画布、Composer 外框、按钮半径、边框阴影和面板间距应由共享 runtime surface 规则控制，不应由 `conversation` 与 `terminal` 分别形成不同视觉皮肤。

后续 PC 精修以共享壳层为主线推进：

1. `RuntimeWorkspacePage`、`RuntimeSessionList`、`RuntimeWorkspaceHeader`、`RuntimeTimeline`、`RuntimeComposer` 继续作为三页共同入口。
2. `data-runtime-view="conversation"` 与 `data-runtime-view="terminal"` 只用于领域内容、状态文案和必要的布局微调，不用于定义两套 shell chrome。
3. Terminal 的 `terminal-*` class 保留在 timeline 内容、process、step、output、jump controls 等领域块内；会话列表、Header、Composer 和主 surface 优先迁移到共享 runtime class。
4. PC 端差异收敛不改变路由、数据结构、终端日志语义、附件上传、发送链路和移动端行为。

### 主导航

目标状态：

- 品牌区展示 Alter0 字标与可识别图标位。
- 导航组保持 `Workspace / Agent Studio / Control / Settings` 信息结构。
- Active 项使用浅蓝底、蓝色图标与文字，不使用胶囊外形。
- 底部保留语言切换，新增用户身份区样式承载头像缩写、名称、邮箱和在线点。
- 折叠按钮维持低圆角方形图标按钮。

涉及文件：

- `internal/interfaces/web/frontend/src/features/shell/components/PrimaryNav.tsx`
- `internal/interfaces/web/frontend/src/features/shell/components/NavIcon.tsx`
- `internal/interfaces/web/frontend/src/features/shell/legacyShellCopy.ts`
- `internal/interfaces/web/frontend/src/styles/shell.css`

### 会话列表

目标状态：

- 会话列表面板保持独立滚动与最近时间分组。
- 当前会话卡使用左侧蓝色竖线、浅蓝选中底和轻阴影。
- 删除入口从显性大按钮收敛为尾侧轻量操作；PC 端可进一步切换为三点菜单。
- 短 hash 使用低对比图标 + monospace 文本，不使用胶囊底。
- 卡片内容顺序稳定为：标题、摘要、短 hash、尾侧操作。

涉及文件：

- `internal/interfaces/web/frontend/src/features/shell/components/RuntimeSessionList.tsx`
- `internal/interfaces/web/frontend/src/features/conversation-runtime/ConversationWorkspace.tsx`
- `internal/interfaces/web/frontend/src/features/shell/components/ReactManagedTerminalRouteBody.tsx`
- `internal/interfaces/web/frontend/src/styles/shell.css`

### 工作区头部

目标状态：

- 左侧显示当前会话标题，标题旁可显示轻量状态点和下拉提示。
- 右侧操作固定为运行状态、`Workspace Flow`、`Details` 三组。
- `Workspace Flow` 作为 Process / Timeline / workspace 流程视图入口；首版可映射到现有 `Details` 面板中的运行摘要或作为禁用态预留入口。
- `Ready / Details / Workspace Flow` 均为低圆角矩形按钮，不使用胶囊。
- Header 在 PC 端保持单行，在窄屏允许换行或收缩。

涉及文件：

- `internal/interfaces/web/frontend/src/features/shell/components/RuntimeWorkspaceHeader.tsx`
- `internal/interfaces/web/frontend/src/features/shell/components/ReactManagedTerminalRouteBody.tsx`
- `internal/interfaces/web/frontend/src/features/conversation-runtime/ConversationWorkspace.tsx`
- `internal/interfaces/web/frontend/src/styles/shell.css`

### 空态工作区

目标状态：

- 空态主画布使用浅色网格、轻弧线和低对比背景纹理。
- 标题居中，说明文案控制在两行以内。
- 空态不渲染额外 route hero、workspace hero、统计条或徽标云。
- 空态容器仍由运行页滚动区承载，移动端不得因背景纹理造成重绘卡顿。

涉及文件：

- `internal/interfaces/web/frontend/src/features/shell/components/RuntimeTimeline.tsx`
- `internal/interfaces/web/frontend/src/features/shell/components/ChatMessageRegion.tsx`
- `internal/interfaces/web/frontend/src/styles/shell.css`

### Composer

目标状态：

- Composer 改为上方输入区 + 下方工具栏的两层结构。
- 左侧工具区支持图标按钮：快捷工具、mention、工具网格。
- 右侧保留 `Attach` 文本按钮与蓝色 icon submit 按钮。
- 输入框、工具按钮和发送按钮保持低圆角矩形。
- 移动端继续复用当前 fixed composer、键盘贴底和真实遮挡高度同步链路。

涉及文件：

- `internal/interfaces/web/frontend/src/features/shell/components/RuntimeComposer.tsx`
- `internal/interfaces/web/frontend/src/features/conversation-runtime/ConversationRuntimeProvider.tsx`
- `internal/interfaces/web/frontend/src/features/shell/components/ReactManagedTerminalRouteBody.tsx`
- `internal/interfaces/web/frontend/src/styles/shell.css`

## 实现顺序

### 第一阶段：视觉令牌与壳层基线

交付内容：

1. 调整 `shell.css` 中背景、边框、阴影、半径和主 surface 令牌。
2. 固化 `border-radius: 999px` 禁用契约。
3. 统一导航、会话面板、工作区面板和 Composer 的低圆角节奏。

测试要求：

- 更新 `shellLayoutStyles.test.ts`，覆盖低圆角令牌和无胶囊半径。
- 运行 `npm run test -- shellLayoutStyles.test.ts`。

### 第二阶段：导航与会话列表

交付内容：

1. 增加品牌图标位和用户身份区。
2. 优化 active 导航项、分组标题和语言切换。
3. 会话卡增加左侧 active 竖线与尾侧轻量操作。

测试要求：

- 更新 `WorkbenchApp.test.tsx` 或相关 shell 测试，覆盖导航与用户区存在性。
- 更新 `ConversationWorkspace.test.tsx` 或 `ReactManagedTerminalRouteBody.test.tsx`，覆盖会话列表语义不变。

### 第三阶段：工作区头部与空态

交付内容：

1. Header 右侧补齐 `Workspace Flow` 入口。
2. 空态背景改为轻网格与细线纹理。
3. 空态文案和居中布局收敛到参考图的视觉重心。

测试要求：

- 更新 `RuntimeWorkspaceFrame.test.tsx` 或 `ConversationWorkspace.test.tsx`。
- 桌面截图验证标题、状态、详情、流程入口不重叠。

### 第四阶段：Composer 工具栏

交付内容：

1. Composer DOM 明确拆为输入区与工具栏。
2. 左侧工具按钮与右侧附件/发送按钮按参考图重排。
3. 上传附件与发送行为继续复用现有 handler。
4. 移动端保持按钮可触达，不遮挡输入区。

测试要求：

- 更新 `ConversationWorkspace.test.tsx`。
- 更新 `ReactManagedTerminalRouteBody.test.tsx`。
- 保留图片/文件附件相关测试。

## 验收标准

### 自动化

1. `npm run test -- shellLayoutStyles.test.ts`
2. `npm run test`
3. `npm run build`
4. 涉及 Go handler 或静态分发变更时运行对应 `go test`。

### 浏览器验证

桌面视口：

- `1440 x 900`
- `1280 x 800`

移动视口：

- `390 x 844`
- `360 x 740`

验证项目：

1. PC 端三栏布局稳定，导航、会话列、主工作区边界清晰。
2. PC 端按钮、标签、短 hash、上传、发送、详情与跳转控件均不是胶囊形态。
3. 空态背景不影响标题和说明阅读。
4. Composer 工具栏不遮挡输入内容，附件与发送按钮可见。
5. 移动端 `Menu / Sessions / New`、workspace header 与 Composer 不重叠。
6. 打开/关闭 Details 后，消息区不被重新排版。

## 文档同步

完成实际代码改造时必须同步以下文档：

1. `README.md`：更新用户可见的工作台视觉与交互说明。
2. `docs/requirements.md`：更新 Conversation / Terminal 相关稳定需求口径。
3. `docs/requirements-details/conversation-session-experience.md`：更新 Chat / Agent Runtime 壳层、空态与 Composer 细则。
4. `docs/requirements-details/task-terminal-workspace.md`：更新 Terminal 共享工作台控件与 Composer 细则。
5. `docs/technical-solution.md`：更新涉及组件、样式令牌、测试契约和静态构建产物的实现说明。
6. `docs/frontend-design-guidelines.md`：若新增可复用视觉规则，同步沉淀为长期设计规范。

## 风险与边界

1. 不在本轮引入新的图标库；若需要新增图标，优先复用现有 `NavIcon` 或轻量内联图标模式。
2. 空态背景使用 CSS 实现，避免新增大位图资源和加载闪烁。
3. `Workspace Flow` 首版不强行新增复杂业务面板；先明确入口、状态和可扩展位置。
4. Composer DOM 调整必须保持附件上传、草稿恢复、发送、移动端 touch submit 和 keyboard inset 行为不变。
5. legacy CSS 仅保持兼容，不作为新视觉主实现入口。

## 免测说明

本次提交仅新增落地说明文档与文档索引，不修改运行时代码、接口契约或构建产物，因此不新增自动化测试。文档有效性通过与当前 React 工作台源码结构、前端设计规范和已建立的测试入口对照校验。
