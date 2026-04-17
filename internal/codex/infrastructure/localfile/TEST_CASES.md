# Test Cases

## 覆盖范围

- Codex 账号记录的保存、读取与列表加载。
- 托管账号目录、备份目录与登录会话目录创建。

## 边界

- 账号业务编排、切换策略与登录会话状态由 `internal/codex/application` 覆盖。
- `auth.json` 结构和身份解析由 `internal/codex/domain` 覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/codex/infrastructure/localfile`
