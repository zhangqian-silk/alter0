# Test Cases

## 覆盖范围

- 登录保护、Chat 页面、静态资源、消息 JSON/SSE、Agent/Product 消息入口。
- Control API：Channel、Capability、Skill、MCP、Agent、Product、Draft、Cron、Environment、Runtime、LLM Provider。
- Workspace service 网关：`/api/control/workspace-services` 注册表、短哈希 Host 命中的 `frontend_dist` 构建分发，以及 `http` 类型服务的反向代理。
- Session、Task、Memory Task、Terminal API 与产物下载/预览。
- 前端模板/静态资源中的移动端、侧边栏、Composer、Terminal、Cron 可观测标识。

## 边界

- 浏览器真实交互由 `internal/interfaces/web/e2e` Playwright 套件覆盖。
- 领域规则不在 Web handler 中重复断言，归属对应 `domain` 与 `application` 包。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/interfaces/web`
