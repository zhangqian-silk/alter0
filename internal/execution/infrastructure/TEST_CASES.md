# Test Cases

## 覆盖范围

- Codex CLI 同步与流式执行、错误、空输出、认证失败、心跳，以及原生 Codex Runtime 下 thread id 持久化与同 Session resume。
- Claude Code 执行器的 provider 环境变量、`CLAUDE.md`、runtime/skill 文件注入与 file-backed Skill 工作区副本。
- Runtime Resolver 的执行器选择：显式 Codex、可用 Provider 优先 Claude Code、Claude 失败兜底 Codex。
- runtime、skill、MCP、memory、agent context prompt 与 metadata 组装。
- 工作区解析、session repo clone、repo root 模式。
- 既有 Hybrid NL Agent/ReAct/Codex 执行源、模型选择、委派、Memory 工具的兼容测试。
- `deploy_test_service` 工具参数解析、默认 `web` 全栈预览参数传递、Session 级服务部署请求转发与脚本型 deployer 调用。

## 边界

- 运行时上下文选择规则由 `internal/execution/application` 覆盖。
- Web 流式协议由 `internal/interfaces/web` 覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/execution/infrastructure`
