# Alter0 Features（OpenClaw 对齐版，2026-03）

## 0. 文档定位

- 本文维护"当前待交付"与"刚完成"的能力状态，不重复展开历史细节。
- 已交付能力矩阵请看 `docs/features.md`。
- OpenClaw 版本对齐清单请看 `docs/openclaw-alignment.md`。

## 1. 阶段目标与约束

- 部署形态：继续保持自部署优先，先做单节点高可用，再考虑分布式扩展。
- 安全边界：默认私有网络可用，但新增能力必须带最小可行权限控制与审计。
- 兼容策略：保留现有 Task 编排主链路（`route -> execute -> close`），新增能力以可插拔方式接入。
- 阶段目标：在核心能力已对齐 OpenClaw 基线后，优先补齐研究报告第 5.2 章剩余缺口（场景基准对比 + 持续竞品追踪）。

## 2. 当前未完成需求（Active Gaps）

### N18 场景基准矩阵与竞品追踪自动化（P2）

- 目标：在 N17 已交付的风险基准能力上，补齐研究报告第 5.2 章剩余建议，形成长期可执行的月度追踪机制。
- 交付边界：
  - 新增三类 workload（个人助理 / 团队协作 / 自动巡检）基准输入与结果模板。
  - 建立竞品追踪清单（活跃度、发布频率、核心 feature 变化）自动化抓取或半自动更新脚本。
  - 将场景基准与竞品追踪结果接入运维检查命令或 release-gate 的可选扩展项。

## 3. 优先级与执行队列

### P0（已完成）

1. [x] N9 异步任务取消语义修复（真实中断 + 状态一致性）
2. [x] N10 存储迁移安全升级（非破坏迁移 + 备份回滚）
3. [x] N14 测试稳定性加固（`make test-stability` + release-gate 接入）

### P1（已完成）

1. [x] N13 配置层依赖解耦（移除反向依赖 + `make check-config-boundary`）
2. [x] N15 服务模块化（`core -> service -> infra`，补充 `make check-service-boundary`）

### P2（当前主线）

1. [x] N16 外部策略与供应链风险监测（provider/channel 策略漂移 + skill/plugin 来源审计）
2. [x] N17 风险执行基准与漂移处置手册（`make risk-benchmark` + runbook + release-gate 接入）
3. [ ] N18 场景基准矩阵与竞品追踪自动化

当前下一项：`P2/N18 场景基准矩阵与竞品追踪自动化`。

## 4. 与 OpenClaw 研究报告对比（2026-03-02）

对照 `../cs-note/ai/agent/openclaw_research_report.md`：

- 已对齐：多通道网关、会话/子代理编排、工具协议与安全门禁、memory 检索、release-gate 基线、服务分层边界、N16 风险 watchlist 自动告警、N17 风险巡检 benchmark + 漂移分级 runbook。
- 当前缺口：研究报告第 5.2 章中的“场景基准对比”与“按月竞品追踪清单”尚未形成工程化更新链路。
- 下一步：推进 N18，补齐 workload benchmark matrix 与竞品追踪自动化，并定义可持续维护频率。

## 5. 执行规则

- 单需求闭环：每次只推进一个需求到"编码 + 测试 + 文档"完成态。
- 文档同步：需求状态变化后，必须同步 `docs/features.md` 与本文件。
- 变更可回滚：涉及存储、会话、任务调度的改动必须附带回滚说明。

## 6. 失败记录与优先重试

- 2026-03-01（UTC）：`git push -u origin feat/p2-n16-risk-watchlist-automation` 连续两次超时失败（无法连接 `github.com:443`），导致 PR/merge 链路阻塞；已在后续轮次恢复并完成 N16 合并。
