# Test Cases

## 覆盖范围

- 并发服务：同会话串行、多会话并发、队列等待、超时、取消、过载拒绝。
- Orchestration：命令优先、Agent 执行、metadata 透传、ChatRuntime Task 记忆跳过。
- Long-term Memory、Mandatory Context、任务摘要召回，以及 CLI runtime 自管的会话上下文边界。
- Session 持久化、Cron/Skill 来源字段、长期记忆分层验收。

## 边界

- 命令与意图领域端口由 `internal/orchestration/domain` 覆盖。
- 命令分类基础设施由 `internal/orchestration/infrastructure` 覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/orchestration/application`
