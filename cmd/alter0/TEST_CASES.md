# Test Cases

## 覆盖范围

- 内置 Skill 初始化与 file-backed Skill 文件处理，包括私有 `memory-maintenance`、统一预览/部署 `preview-publish`、`frontend-design` 与 `travel` 路径约束。
- 运行时 PATH、NO_PROXY、Web 登录密码环境变量和内部启动参数过滤。
- Codex CLI 自动版本解析覆盖运行账户托管路径、NVM 稳定入口、历史版本路径、语义版本比较、同版本稳定入口优先级与显式 pinned 锁版。
- supervisor client 重启错误、结构化重启错误码透传、探活地址归一、空响应错误，以及 `sync_remote_master` 遇到 tracked 改动时要求二次确认，确认后才丢弃 tracked 改动。
- Runtime 重启候选 commit 读取覆盖当前运行 commit 之后全部 `origin/master` 提交，以及当前运行 commit 向前 10 个历史提交；指定 `target_commit` 时必须留在 `master` 分支并重置到目标 master 提交。
- Runtime supervisor 候选二进制构建必须调用统一前端感知构建脚本，并通过 `ALTER0_BUILD_OUTPUT` 指定候选输出路径。

## 边界

- 本路径测试只覆盖启动命令与 runtime supervisor 边界，不启动真实长期服务。
- 部署脚本、systemd、Nginx 与宿主签名凭据由 `docs/deployment` 与脚本级验证维护。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./cmd/alter0`
