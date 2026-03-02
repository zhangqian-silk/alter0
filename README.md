# alter0

一个按 DDD 方式拆分的个人部署版 agent 骨架，目标是实现类似 openclaw 的三段式链路：

1. `Gateway`：CLI/Web 接收输入并转换为统一消息。
2. `Orchestration`：判断命令还是自然语言并做路由。
3. `Execution`：执行自然语言请求。

当前版本按你的约束实现了无鉴权、单机可运行、带可观测能力的最小闭环。

## 模块结构

```text
cmd/alter0                         # 程序入口（web/cli 模式）
internal/interfaces/cli            # CLI 网关
internal/interfaces/web            # Web 网关
internal/orchestration/domain      # 编排领域模型（Intent/Command）
internal/orchestration/application # 编排应用服务
internal/orchestration/infrastructure
internal/execution/domain          # 执行领域接口
internal/execution/application     # 执行应用服务
internal/execution/infrastructure  # NL 执行器实现（示例）
internal/shared/domain             # UnifiedMessage / OrchestrationResult
internal/shared/infrastructure     # ID 生成、日志、metrics
```

## 核心流程

1. CLI/Web 输入都转换成 `UnifiedMessage`。
2. 编排模块 `IntentClassifier` 判断：
   1. 命令：`/` 前缀，或首 token 命中已注册命令。
   2. 自然语言：其他输入。
3. 命令走 `CommandRegistry` 分发到 `CommandHandler`。
4. 自然语言走 `ExecutionPort`（当前示例输出 `nl_executor: <content>`）。

## 内置命令

1. `/help`：查看命令列表
2. `/echo ...`：回显参数
3. `/time`（别名 `/now`）：输出 UTC 时间（RFC3339）

## 可观测性

1. 结构化日志（JSON）
2. `/metrics`：Prometheus 文本格式指标
3. `/healthz`：活性检查
4. `/readyz`：就绪检查
5. 关键字段贯穿日志：`trace_id`、`session_id`、`message_id`、`route`

## 运行方式

### Web 模式

```bash
go run ./cmd/alter0 -mode=web -addr=127.0.0.1:8088
```

调用示例：

```bash
curl -X POST http://127.0.0.1:8088/api/messages \
  -H "Content-Type: application/json" \
  -d '{"session_id":"s1","content":"/help"}'
```

### CLI 模式

```bash
go run ./cmd/alter0 -mode=cli
```

输入 `/quit` 或 `/exit` 退出。

## 测试

```bash
go test ./...
```
