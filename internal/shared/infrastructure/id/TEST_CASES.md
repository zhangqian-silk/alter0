# Test Cases

## 覆盖范围

- RandomIDGenerator 输出 UTC 时间戳前缀。
- 随机后缀可用时采用 8 字节十六进制格式。
- 随机源不可用时允许仅时间戳 fallback。

## 边界

- 跨进程全局唯一性不通过单测穷举证明，由时间戳、随机熵和调用侧业务键共同保障。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/shared/infrastructure/id`
