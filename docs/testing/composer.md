# Web 输入组件复用说明

## 范围

Web 侧所有需要文本输入、草稿恢复、提交控制与输入法兼容的场景，统一遵循同一组 composer 契约。

当前接入范围：

- 主聊天输入框
- Cron 提示词输入框
- Control 任务终端输入框
- Terminal 路由输入框

当前实现已拆到各自 React 入口：

- `internal/interfaces/web/frontend/src/features/conversation-runtime/ConversationRuntimeProvider.tsx` 负责 Chat / Agent 主输入框的草稿恢复、字数限制、会话级草稿持久化与提交编排
- `internal/interfaces/web/frontend/src/features/conversation-runtime/ConversationWorkspace.tsx` 与 `internal/interfaces/web/frontend/src/features/shell/components/RuntimeComposer.tsx` 负责 Chat / Agent 共享 Composer 的聚焦、附件、提交与移动端视口协调
- `internal/interfaces/web/frontend/src/features/shell/components/ReactManagedTerminalRouteBody.tsx` 负责 Terminal 路由输入框的草稿恢复、提交与输入态保持
- Cron 与 Control 任务终端继续遵循同一 `data-composer-*` DOM 契约与 E2E 断言接口

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

- 新增文本输入场景时，优先复用现有 composer 契约与 `data-composer-*` 标记
- 涉及会话切换时，必须提供稳定的 `draftKey`
- 涉及路由切换或面板关闭时，必须接入未发送内容保护
- 涉及轮询刷新或异步渲染时，必须验证焦点与草稿不会丢失

## 测试要求

- 输入法合成态场景必须有 E2E 回归
- 草稿恢复、会话切换、删除会话后的行为必须有 E2E 回归
- 路由切换或关闭行为必须覆盖确认与取消两条路径
