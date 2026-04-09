# Test Cases

## 覆盖范围

- 默认 LLM Provider 配置：OpenAI 与 OpenRouter 条目、默认禁用、空默认 Provider、API type 与 base URL。

## 边界

- Provider 领域校验由 `internal/llm/domain` 覆盖。
- Provider 文件存储与 OpenAI/OpenRouter HTTP 适配由 `internal/llm/infrastructure` 覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/llm/application`
