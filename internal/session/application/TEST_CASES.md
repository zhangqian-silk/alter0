# Test Cases

## 覆盖范围

- 消息追加、按时间范围读取。
- 会话列表分页、来源字段过滤、从 store 加载索引。
- store 失败回滚。
- 删除会话记录、索引更新和失败回滚。

## 边界

- MessageRecord 与 MessageSource 领域校验由 `internal/session/domain` 覆盖。
- Web 会话 API 与前端入口由 `internal/interfaces/web` 覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/session/application`
