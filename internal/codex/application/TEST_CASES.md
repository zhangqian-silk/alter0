# Test Cases

## 覆盖范围

- Codex 账号导入、当前活动账号识别、账号切换与备份生成。
- OAuth 账号状态刷新、额度查询结果回写与当前账号标记。
- 独立 `codex login` 会话启动、日志收集、成功保存与失败收口。

## 边界

- `auth.json` 结构解析与身份推导由 `internal/codex/domain` 覆盖。
- 本地文件持久化契约由 `internal/codex/infrastructure/localfile` 覆盖。
- Web API 与前端交互契约由 `internal/interfaces/web` 与前端测试覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/codex/application`
