# Test Cases

## 覆盖范围

- Scheduler Manager 到点触发 Cron UnifiedMessage。
- 每次 Cron 触发使用独立会话。

## 边界

- Cron 表达式和 timezone 规则由 `internal/scheduler/domain` 覆盖。
- Control Cron API 由 `internal/interfaces/web` 覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/scheduler/application`
