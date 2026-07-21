# Test Cases

## 覆盖范围

- 登录保护、Chat 页面、静态资源、Chat runtime session input、Skill 消息入口。
- Control API：Channel、Capability、Skill 生命周期、Alter0/Codex Skill 只读目录聚合、MCP、Cron、Environment、Runtime、LLM Provider、Codex Accounts、Codex device-code login sessions、Maintenance。
- Workspace service 网关：`/api/control/workspace-services` 注册表、短哈希 Host 命中的 `frontend_dist` 构建分发，以及默认 `web` / 其他服务的 `http` 类型反向代理。
- Session、Task、Memory Task、Chat runtime API 与产物下载/预览；Session 置顶接口、维护会话清理接口、置顶跳过、queued/running 任务保护、runtime busy 保护、workspace 删除、维护执行器不可用和清理资源删除失败。
- 前端模板/静态资源中的移动端、侧边栏、Composer、Cron 可观测标识，Runtime 重启确认错误码、候选 commit 列表与目标 commit 透传，以及 Runtime 页 Codex device-code 登录、Claude Code Provider Console 多 Provider 连续注册、查看、编辑、默认远端同步和按需二次确认。

## 边界

- 浏览器真实交互由 `internal/interfaces/web/e2e` Playwright 套件覆盖。
- 领域规则不在 Web handler 中重复断言，归属对应 `domain` 与 `application` 包。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/interfaces/web`
