# Test Cases

## 覆盖范围

- Codex/OpenAI 复杂度预测解析、提示词、timeout EWMA/backoff/fallback。
- 异步规则分流、文档型请求保留流式、预测失败 fallback。
- worker 并发上限、任务提交/完成、幂等、队列位置/等待、终端日志捕获。
- retry/cancel/timeout/heartbeat 续租、Task summary 精简、Terminal timeout 默认值。

## 边界

- Task 领域字段校验由 `internal/task/domain` 覆盖。
- Web 任务接口、日志 SSE、产物下载/预览由 `internal/interfaces/web` 覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/task/application`
