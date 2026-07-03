# Test Cases

## 覆盖范围

- Chat runtime session status 归一。
- 会话 open 状态与输入可用性判断。

## 边界

- Chat runtime 持久化、Codex 执行和恢复由 `internal/chatruntime/application` 覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/chatruntime/domain`
