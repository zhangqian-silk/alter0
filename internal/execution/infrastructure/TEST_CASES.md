# Test Cases

## 覆盖范围

- Codex CLI 同步与流式执行、错误、空输出、认证失败、心跳。
- runtime、product、skill、MCP、memory、agent context prompt 与 metadata 组装。
- 工作区解析、session repo clone、repo root 模式。
- Hybrid NL Agent/ReAct/Codex 执行源、模型选择、委派、Product Context、Memory 工具。
- `deploy_test_service` 工具参数解析、Session 级服务部署请求转发与脚本型 deployer 调用。

## 边界

- 运行时上下文选择规则由 `internal/execution/application` 覆盖。
- Web 流式协议由 `internal/interfaces/web` 覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/execution/infrastructure`
