# Test Cases

## 覆盖范围

- 自然语言执行运行时 metadata 注入与执行来源回传。
- Skill include/exclude、优先级、Agent 私有 Skill、冲突解析。
- Memory Files 选择、自动召回、Agent 私有 AGENTS.md、Agent Session Profile。
- MCP 全局、会话、请求范围选择，transport、timeout、隔离、审计。

## 边界

- Codex CLI、ReAct 和工具执行由 `internal/execution/infrastructure` 覆盖。
- 心跳领域契约由 `internal/execution/domain` 覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/execution/application`
