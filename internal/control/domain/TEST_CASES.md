# Test Cases

## 覆盖范围

- memory capability 与 runtime catalog 字段往返。
- Capability 默认值、版本规则与类型校验。
- Skill 统一模型校验。

## 边界

- 持久化和审计写入由 `internal/control/application` 与 `internal/storage/infrastructure/localfile` 覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/control/domain`
