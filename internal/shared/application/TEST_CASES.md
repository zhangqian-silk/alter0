# Test Cases

## 覆盖范围

- IDGenerator 应用端口方法契约。
- Telemetry 应用端口计数与耗时观测方法契约。

## 边界

- 具体 ID 生成由 `internal/shared/infrastructure/id` 覆盖。
- 具体 metrics 输出由 `internal/shared/infrastructure/observability` 覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/shared/application`
