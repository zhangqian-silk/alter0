# Test Cases

## 覆盖范围

- Intent 稳定类型：`command` 与 `nl`。
- CommandHandler 接收 UnifiedMessage、命令名和参数，并返回输出与 metadata。

## 边界

- CommandRegistry 具体注册、解析、别名处理由 application/infrastructure 覆盖。
- 本路径不绑定具体内置命令文案。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/orchestration/domain`
