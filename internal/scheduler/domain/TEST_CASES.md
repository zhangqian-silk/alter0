# Test Cases

## 覆盖范围

- Visual cron mode 到 cron expression 的解析与往返。
- ScheduleSpec timezone 下次触发时间。
- Job legacy interval 归一兼容。

## 边界

- Scheduler 触发和回注编排由 `internal/scheduler/application` 覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/scheduler/domain`
