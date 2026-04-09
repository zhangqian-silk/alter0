# Test Cases

## 覆盖范围

- MessageRole 合法值。
- MessageSource trigger/channel 允许值与拒绝值。
- MessageRecord 身份、会话、角色、内容、时间戳、来源校验。

## 边界

- 会话持久化、索引和删除由 `internal/session/application` 覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/session/domain`
