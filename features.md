# Alter0 Features（OpenClaw 对齐版，2026-03）

## 0. 文档定位

- 本文维护"当前待交付"与"刚完成"的能力状态，不重复展开历史细节。
- 已交付能力矩阵请看 `docs/features.md`。
- OpenClaw 版本对齐清单请看 `docs/openclaw-alignment.md`。

## 1. 阶段目标与约束

- 部署形态：继续保持自部署优先，先做单节点高可用，再考虑分布式扩展。
- 安全边界：默认私有网络可用，但新增能力必须带最小可行权限控制与审计。
- 兼容策略：保留现有 Task 编排主链路（`route -> execute -> close`），新增能力以可插拔方式接入。
- 阶段目标：在核心能力已对齐 OpenClaw 基线后，固化第 5.2 章建议并补齐第 5.1 章提出的长会话成本漂移监测。

## 2. 当前未完成需求（Active Gaps）

- 运行时成本治理主链路已闭环，当前无阻塞型功能缺口。
- 自适应阈值回写已具备“提案 + 周归档 + 命中率观测”能力，仍保持人工确认后落地（默认不自动改配置）。
- 外部依赖仍需保持可用（GitHub 网络与 token），避免阻塞 PR/merge 链路。

## 3. 优先级与执行队列

### P0（已完成）

1. [x] N9 异步任务取消语义修复（真实中断 + 状态一致性）
2. [x] N10 存储迁移安全升级（非破坏迁移 + 备份回滚）
3. [x] N14 测试稳定性加固（`make test-stability` + release-gate 接入）

### P1（已完成）

1. [x] N13 配置层依赖解耦（移除反向依赖 + `make check-config-boundary`）
2. [x] N15 服务模块化（`core -> service -> infra`，补充 `make check-service-boundary`）

### P2（已完成）

1. [x] N16 外部策略与供应链风险监测（provider/channel 策略漂移 + skill/plugin 来源审计）
2. [x] N17 风险执行基准与漂移处置手册（`make risk-benchmark` + runbook + release-gate 接入）
3. [x] N18 场景基准矩阵与竞品追踪自动化（`config/scenario-benchmark-matrix.json`、`config/competitor-tracking.json`、`make competitor-tracking-refresh`）

### P3（治理深化，已完成）

1. [x] N19 配置模型参数级解剖与门禁（`make config-governance` + `output/config/governance-latest.json`）
2. [x] N20 配置治理月度巡检自动化（`make config-governance-cadence` + `output/config/governance-cadence-latest.json` + `output/config/governance-history/`）

### P4（运行时成本治理）

1. [x] N21 长会话成本热点与压缩压力告警（`/status` 成本热点 + alerts `session_cost_hotspot` / `session_compaction_pressure`）
2. [x] N22 成本阈值校准指引（`/status.cost.threshold_guidance` 给出 p90 建议阈值与偏移量）
3. [x] N23 成本阈值历史回归自动化（`make cost-threshold-history` 周归档 + 告警命中率回归 + `make risk-benchmark` 联动校验）
4. [x] N24 分层阈值建议（`/status.cost.threshold_guidance.workload_tiers` 按 token 量级给出分层建议）
5. [x] N25 阈值回写协调器（`make cost-threshold-reconcile` 生成阈值调优提案，支持 `--apply` 受控写回 `config/config.json`）
6. [x] N26 阈值回写节奏归档（`make cost-threshold-reconcile` 周归档 `output/cost/threshold-reconcile/<ISO-week>/`，并输出 `cadence.ready_rate/applied_rate/ready_streak`）

### P5（多通道韧性治理）

1. [x] N27 通道降级观测与回退建议（`/status.channel_degradation` 输出每通道退化等级、fallback 候选与恢复建议，并新增 `alerts.channel_degradation`）

当前状态：N27 已闭环，运行时治理从“成本阈值”扩展到“多通道降级可观测 + 回退指引”。

## 4. 与 OpenClaw 研究报告对比（2026-03-01 UTC）

对照 `../cs-note/ai/agent/openclaw_research_report.md`：

- 已对齐：多通道网关、会话/子代理编排、工具协议与安全门禁、memory 检索、release-gate 基线、服务分层边界、N16 风险 watchlist 自动告警、N17 风险巡检 benchmark + 漂移分级 runbook、N18 场景基准矩阵与竞品月度追踪链路、N19 参数级配置治理门禁、N20 月度治理节奏自动化。
- 本轮新增对齐：针对研究报告 5.1 剩余项“多通道故障时的降级策略与可观测性完整度”，新增 N27 通道降级观测层，`/status.channel_degradation` 现可输出每通道 error/disconnect 严重度、fallback 候选与恢复建议，并通过 `alerts.channel_degradation` 给出统一告警。
- 当前缺口：研究报告 5.2 建议项保持全量落地；5.1 现已具备“成本治理 + 多通道降级观测”双闭环，暂无阻塞级功能缺口。
- 下一步：在生产样本累积后补充通道级 chaos drill（按通道注入断连）并回填 `channel_degradation` 阈值参数，验证告警噪声与误报率。

## 5. 执行规则

- 单需求闭环：每次只推进一个需求到"编码 + 测试 + 文档"完成态。
- 文档同步：需求状态变化后，必须同步 `docs/features.md` 与本文件。
- 变更可回滚：涉及存储、会话、任务调度的改动必须附带回滚说明。

## 6. 失败记录与优先重试

- 2026-03-01（UTC）：本轮 `git fetch origin --prune` 前 2 次失败（`Failure when receiving data from the peer`），第 4 次重试恢复；后续已完成远端同步与开发链路。
- 2026-03-01（UTC）：`git push -u origin feat/p2-n16-risk-watchlist-automation` 连续两次超时失败（无法连接 `github.com:443`），导致 PR/merge 链路阻塞；已在后续轮次恢复并完成 N16 合并。
- 2026-03-02（UTC）：`make risk-benchmark` 初次执行因缺少 N18 基准文件失败（`config/scenario-benchmark-matrix.json`、`config/competitor-tracking.json`）；已在同轮补齐并通过门禁，无待重试项。
- 2026-03-02（UTC）：`make competitor-tracking-refresh` 在未配置 `GH_TOKEN/GITHUB_TOKEN` 时触发 GitHub API 403 rate limit；已改为默认降级不中断并记录 warning，下一轮优先在带 token 环境执行一次完整刷新。
- 2026-03-01（UTC）：本轮执行 `git fetch origin` 连续超时（无输出后被超时终止），无法确认远端 `master` 最新提交；已改为基于本地 `master` 继续开发，下一轮优先重试网络连通后再同步远端。
- 2026-03-01（UTC）：`git push -u origin feat/n21-session-cost-pressure-alerts` 曾两次失败（`Failure when receiving data from the peer` / `Failed to connect to github.com:443`）；已在 2026-03-01 晚间恢复并完成 PR #89 合并。
