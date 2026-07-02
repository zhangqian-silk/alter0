# Test Cases

## 覆盖范围

- Channel CRUD。
- Capability 生命周期、审计、Skill/MCP 统一存储。
- Control store 启动加载跳过历史遗留的非 Skill/MCP capability，不阻断合法 Skill/MCP 恢复。

## 边界

- 控制面 HTTP 协议由 `internal/interfaces/web` 覆盖。
- 领域字段归一和校验由 `internal/control/domain` 覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/control/application`
