# Test Cases

## 覆盖范围

- 内置 Skill 初始化与 file-backed Skill 文件处理，包括 `deploy-test-service`、`frontend-design` 与 Agent 私有 `SKILL.md` 路径约束。
- 运行时 PATH、NO_PROXY、Web 登录密码环境变量和内部启动参数过滤。
- supervisor client 重启错误、探活地址归一、空响应错误，以及 `sync_remote_master` 仅丢弃 tracked 改动并保留 untracked 工作区内容。

## 边界

- 本路径测试只覆盖启动命令与 runtime supervisor 边界，不启动真实长期服务。
- 部署脚本、systemd、Nginx 与宿主签名凭据由 `docs/deployment` 与脚本级验证维护。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./cmd/alter0`
