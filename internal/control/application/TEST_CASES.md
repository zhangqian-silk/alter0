# Test Cases

## 覆盖范围

- Environment 默认定义、配置更新与审计记录。
- Channel CRUD。
- Capability 生命周期、审计、Skill/MCP 统一存储。

## 边界

- 控制面 HTTP 协议由 `internal/interfaces/web` 覆盖。
- 领域字段归一和校验由 `internal/control/domain` 覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/control/application`
