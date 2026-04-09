# Test Cases

## 覆盖范围

- RuntimeHeartbeat reporter 注入、发送、UTC 时间戳补齐。
- 显式时间戳保留。
- nil context、缺少回调与 nil 回调的静默边界。

## 边界

- 心跳持久化和任务超时续租由 Task 应用层覆盖。
- 执行上下文 payload 组装由 `internal/execution/application` 覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/execution/domain`
