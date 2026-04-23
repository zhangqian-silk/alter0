# Test Cases

## 覆盖范围

- Control、Scheduler、Session、Task 本地文件往返。
- JSON/Markdown 兼容读取。
- Task artifact、日志追加去重、legacy 迁移、日志保留、删除目录清理。

## 边界

- 领域校验由对应 `domain` 包覆盖。
- HTTP 下载/预览协议由 `internal/interfaces/web` 覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/storage/infrastructure/localfile`
