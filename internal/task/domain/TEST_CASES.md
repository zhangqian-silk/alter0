# Test Cases

## 覆盖范围

- TaskStatus 与 TaskLogLevel 合法值和终态判断。
- TaskSummary 空值、必填字段、状态、完成时间。
- Task 身份、来源消息 fallback、状态、阶段、进度、队列、重试、超时、时间、请求内容。
- 日志、产物、任务摘要、message link 一致性。

## 边界

- 生命周期编排和后台执行由 `internal/task/application` 覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/task/domain`
