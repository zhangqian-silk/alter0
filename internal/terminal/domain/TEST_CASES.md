# Test Cases

## 覆盖范围

- Terminal session status 归一。
- 会话 open 状态与输入可用性判断。

## 边界

- Terminal 持久化、Codex 执行和恢复由 `internal/terminal/application` 覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/terminal/domain`
