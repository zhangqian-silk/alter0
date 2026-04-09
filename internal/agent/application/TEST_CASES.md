# Test Cases

## 覆盖范围

- 内置 Agent Catalog 中 Travel 相关 Agent 的 assistant/codex 执行模型约束。

## 边界

- 用户管理 Agent Profile、Capability 与运行时上下文注入由 `internal/control` 和 `internal/execution` 覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/agent/application`
