# Web 输入组件复用说明

## 范围

Web 侧所有需要文本输入、草稿恢复、提交控制与输入法兼容的场景，统一接入复用输入组件。

当前接入范围：

- 主聊天输入框
- Cron 提示词输入框
- Control 任务终端输入框
- Terminal 路由输入框

核心实现位于 `internal/interfaces/web/static/assets/chat.js` 中的 `createReusableComposer()`。

E2E 侧统一通过 `internal/interfaces/web/e2e/helpers/components/composer.ts` 与
`internal/interfaces/web/e2e/helpers/asserts/composer.ts` 接入该能力。

## 统一能力

- 输入法合成态保护，避免中文输入过程中被回车或刷新打断
- 草稿持久化，按场景选择 `sessionStorage` 或 `localStorage`
- 提交中禁用输入与提交按钮
- 字符计数同步
- 未发送内容导航守卫
- 会话级草稿切换与恢复

## 草稿策略

### 主聊天

- 存储介质：`sessionStorage`
- 草稿粒度：按聊天会话隔离
- 键规则：`chat.main:<session-id>`

### Cron

- 存储介质：`sessionStorage`
- 草稿粒度：单输入框

### Control 任务终端

- 存储介质：`localStorage`
- 草稿粒度：按任务终端锚点隔离

### Terminal 路由

- 存储介质：`localStorage`
- 草稿粒度：按终端会话隔离

## 接入要求

- 新增文本输入场景时，优先复用 `createReusableComposer()`
- 涉及会话切换时，必须提供稳定的 `draftKey`
- 涉及路由切换或面板关闭时，必须接入未发送内容保护
- 涉及轮询刷新或异步渲染时，必须验证焦点与草稿不会丢失

## 测试要求

- 输入法合成态场景必须有 E2E 回归
- 草稿恢复、会话切换、删除会话后的行为必须有 E2E 回归
- 路由切换或关闭行为必须覆盖确认与取消两条路径
