# Test Cases

## 覆盖范围

- UnifiedMessage 必填身份、会话、通道、触发类型、内容、trace 和接收时间。
- TriggerType 仅允许 user、cron、system。

## 边界

- 输入源到 UnifiedMessage 的转换由 CLI/Web/Scheduler 接口或应用层覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/shared/domain`
