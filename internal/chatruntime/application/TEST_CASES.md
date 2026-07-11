# Test Cases

## 覆盖范围

- Codex command/args/options 解析。
- Chat runtime create/recover 工作区分配、状态、会话共享和 persisted session 加载。
- 输入启动/恢复 Codex 线程、标题升级、并发拒绝、busy snapshot、认证失败落为 failed 终态并在原会话重试恢复。
- close/delete、置顶持久化、置顶优先排序、shutdown interrupted notice、conversation updated_at 排序、helper process。

## 边界

- Chat runtime 领域状态归一由 `internal/chatruntime/domain` 覆盖。
- Web API 和前端静态资源由 `internal/interfaces/web` 与 Playwright E2E 覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/chatruntime/application`
