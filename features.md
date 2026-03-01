# Alter0 Features（OpenClaw 对齐版，2026-03）

## 0. 文档定位

- 本文维护"当前待交付"与"刚完成"的能力状态，不重复展开历史细节。
- 已交付能力矩阵请看 `docs/features.md`。
- OpenClaw 版本对齐清单请看 `docs/openclaw-alignment.md`。

## 1. 阶段目标与约束

- 部署形态：继续保持自部署优先，先做单节点高可用，再考虑分布式扩展。
- 安全边界：默认私有网络可用，但新增能力必须带最小可行权限控制与审计。
- 兼容策略：保留现有 Task 编排主链路（`route -> execute -> close`），新增能力以可插拔方式接入。
- 阶段目标：在核心能力已对齐 OpenClaw 基线后，优先补齐架构可维护性缺口（分层依赖、服务模块化）。

## 2. 当前未完成需求（Active Gaps）

### N17 风险执行基准与漂移处置手册（P2）

- 目标：将 N16 的风险监测能力升级为“可演练、可量化、可复盘”的执行基准，补齐研究报告第 5.2 章建议。
- 交付边界：
  - 覆盖 `provider_policy` / `supply_chain` 的风险巡检基准脚本。
  - 漂移事件分级处置 runbook（检测 -> 分级 -> 通知 -> 恢复）。
  - 基准结果接入 release-gate 或运维检查命令的最小闭环。

## 3. 优先级与执行队列

### P0（已完成）

1. [x] N9 异步任务取消语义修复（真实中断 + 状态一致性）
2. [x] N10 存储迁移安全升级（非破坏迁移 + 备份回滚）
3. [x] N14 测试稳定性加固（`make test-stability` + release-gate 接入）

### P1（已完成）

1. [x] N13 配置层依赖解耦（移除反向依赖 + `make check-config-boundary`）
2. [x] N15 服务模块化（`core -> service -> infra`，补充 `make check-service-boundary`）

### P2（当前主线）

1. [ ] N16 外部策略与供应链风险监测（provider/channel 策略漂移 + skill/plugin 来源审计，代码/测试已完成，待远端发布链路）
2. [ ] N17 风险执行基准与漂移处置手册（风险巡检基准 + 漂移分级 runbook）

当前下一项：`P2/N16 发布链路重试（push -> PR -> merge）`。

## 4. 与 OpenClaw 研究报告对比（2026-03-01）

对照 `../cs-note/ai/agent/openclaw_research_report.md`：

- 已对齐：多通道网关、会话/子代理编排、工具协议与安全门禁、memory 检索、release-gate 基线、服务分层边界；N16 风险 watchlist 自动告警闭环已在功能分支实现，待合并主线。
- 当前缺口：研究报告第 5.2 章“场景基准对比 + 持续竞品/风险追踪”尚未形成工程化执行基准与处置手册。
- 下一步：推进 N17，落地风险巡检 benchmark 与漂移分级 runbook，并把结果纳入 release-gate/运维检查流程。

## 5. 执行规则

- 单需求闭环：每次只推进一个需求到"编码 + 测试 + 文档"完成态。
- 文档同步：需求状态变化后，必须同步 `docs/features.md` 与本文件。
- 变更可回滚：涉及存储、会话、任务调度的改动必须附带回滚说明。

## 6. 失败记录与优先重试

- 2026-03-01（UTC）：`git push -u origin feat/p2-n16-risk-watchlist-automation` 连续两次超时失败（无法连接 `github.com:443`），导致 PR/merge 链路阻塞；下一轮优先重试发布链路。
