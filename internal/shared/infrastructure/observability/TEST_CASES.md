# Test Cases

## 覆盖范围

- Telemetry counters、route duration、queue wait、queue depth、worker in-flight 的 Prometheus 文本输出。
- 空样本场景零值指标。
- 标签按字典序稳定输出。

## 边界

- slog JSON handler 使用标准库封装，不锁定所有日志字段细节。
- 部署侧 Prometheus 抓取和告警规则不在本路径覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/shared/infrastructure/observability`
