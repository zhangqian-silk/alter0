# Test Cases

## 覆盖范围

- ModelProvider 默认值、API type、Provider type、OpenRouter 字段、默认模型和启用模型约束。
- ModelConfig Provider 新增、更新、重命名、重复名称、默认项收敛。
- ReAct Agent 多轮用户消息注入与迭代上限 fallback。

## 边界

- Provider 持久化和 HTTP 请求格式由 `internal/llm/infrastructure` 覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/llm/domain`
