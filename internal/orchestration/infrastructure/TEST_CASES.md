# Test Cases

## 覆盖范围

- Slash command 分类。
- 已注册无 slash 命令分类。
- 自然语言 fallback 分类。

## 边界

- 命令执行与结果收口由 `internal/orchestration/application` 覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/orchestration/infrastructure`
