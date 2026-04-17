# Test Cases

## 覆盖范围

- `auth.json` 的 OAuth / API Key 两类快照解析。
- 账号身份键、推荐名称、过期时间与刷新时间推导。
- OAuth 账号名、邮箱、用户 ID、账号 ID 与套餐信息提取。

## 边界

- 额度接口调用、token 刷新与登录会话编排由 `internal/codex/application` 覆盖。
- 文件持久化与登录工作目录由 `internal/codex/infrastructure/localfile` 覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/codex/domain`
