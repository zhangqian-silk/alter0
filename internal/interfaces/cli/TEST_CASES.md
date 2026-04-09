# Test Cases

## 覆盖范围

- CLI stdin 输入到 `UnifiedMessage` 的转换。
- 空输入跳过、`/quit` 与 `/exit` 退出。
- 编排成功输出、编排错误输出、gateway 指标记录。

## 边界

- CLI 测试不验证具体命令行为；命令行为由 Orchestration 测试覆盖。
- stdin/stdout 使用 pipe 注入，不依赖交互式终端。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/interfaces/cli`
