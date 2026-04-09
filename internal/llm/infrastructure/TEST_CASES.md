# Test Cases

## 覆盖范围

- ModelConfigStorage 加载、clone、legacy 默认 Provider 收敛、缺 API Key 恢复。
- OpenAI client Responses API、Chat Completions API、streaming、OpenRouter headers 与 routing options。

## 边界

- 模型配置业务规则由 `internal/llm/domain` 覆盖。
- Web 控制面请求和响应由 `internal/interfaces/web` 覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/llm/infrastructure`
