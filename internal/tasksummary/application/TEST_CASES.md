# Test Cases

## 覆盖范围

- RuntimeMarkdownStore 写入 daily 与 long-term 任务摘要。
- Rebuild upsert 行为。
- RecorderGroup fanout。
- StoreSnapshot 默认 recent window、deep retrieval、drill down、防误触发与 miss fallback。

## 边界

- Task 主生命周期由 `internal/task/application` 覆盖。
- Memory 注入和长期记忆策略由 `internal/orchestration/application` 覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/tasksummary/application`
