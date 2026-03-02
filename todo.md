# Alter0 Features（OpenClaw 对齐版，2026-03）

## 0. 文档定位

- 本文维护"当前待交付"与"刚完成"的能力状态，不重复展开历史细节。
- 已交付能力矩阵请看 `docs/features.md`。
- OpenClaw 版本对齐清单请看 `docs/openclaw-alignment.md`。

## 1. 阶段目标与约束

- 部署形态：继续保持自部署优先，先做单节点高可用，再考虑分布式扩展。
- 安全边界：默认私有网络可用，但新增能力必须带最小可行权限控制与审计。
- 兼容策略：保留现有 Task 编排主链路（`route -> execute -> close`），新增能力以可插拔方式接入。
- 阶段目标：在核心能力已对齐 OpenClaw 基线后，固化第 5.2 章建议，并把第 5.1 风险项推进到“可观测 + 演练 + 周期复盘”闭环。

## 2. 当前未完成需求（Active Gaps）

- 运行时成本治理主链路已闭环，并新增会话压缩质量漂移画像（`/status.cost.compaction_quality` + `alerts.session_compaction_quality_drift`），当前无阻塞型功能缺口。
- 通道韧性治理已补齐“降级观测 + chaos drill 演练门禁 + 阈值分层抑噪 + trace 抽样候选 + 周校准复盘指标 + source_candidate 覆盖/采纳归因画像”，当前重点转为稳定执行周节奏并推动候选场景采纳。
- 自适应阈值回写已具备“提案 + 周归档 + 命中率观测”能力，仍保持人工确认后落地（默认不自动改配置）。
- 研究报告 5.1 中“外部模型/渠道政策突变”已补齐 trace 侧 `provider_policy_incidents` 观测与 `alerts.provider_policy_drift` 告警归因，可按类别/通道定位异常源。
- 研究报告 5.1 中“长会话 token 成本与 compaction 质量权衡”已补齐聚合观测，支持按重负载会话识别 `drift_share`、`top_drift_sessions` 并触发质量漂移告警。
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
4. [x] N34 外部策略异常可观测性补齐（`/status.provider_policy_incidents` + `alerts.provider_policy_drift` 输出类别/通道归因与样本）
5. [x] N35 外部策略告警阈值治理（`runtime.observability.provider_policy.critical_signal_threshold` 控制 quota-only 信号升级阈值）

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
7. [x] N36 会话压缩质量漂移治理（`/status.cost.compaction_quality` 聚合重负载会话 `drift_share`/`top_drift_sessions`，新增 `alerts.session_compaction_quality_drift`）

### P5（多通道韧性治理）

1. [x] N27 通道降级观测与回退建议（`/status.channel_degradation` 输出每通道退化等级、fallback 候选与恢复建议，并新增 `alerts.channel_degradation`）
2. [x] N28 通道 chaos drill 门禁（`make channel-chaos-drill` 基于 `config/channel-chaos-matrix.json` 执行断连/错误尖峰场景回归，输出 `output/channel-chaos/drill-latest.json`）
3. [x] N29 通道降级阈值分层治理（新增 `runtime.observability.channel_degradation` 配置与 channel override，`/status.channel_degradation` 输出 `thresholds` / `suppressed_channels` / `threshold_profile`，抑制低样本误报）
4. [x] N30 通道 chaos drill 阈值画像回归（`config/channel-chaos-matrix.json` 支持 `thresholds.default/overrides`、`expect.min_suppressed_channels`、`expect.required_threshold_profile`，把阈值策略命中纳入回归门禁）
5. [x] N31 真实故障样本自动抽样候选（`make channel-chaos-candidates` 从 `output/trace` 采样异常通道，输出 `output/channel-chaos/candidates-latest.json` 供 review 后回填 matrix）
6. [x] N32 周度校准复盘指标（`make channel-chaos-calibration` 关联候选归档 + 阈值节奏，输出 `output/channel-chaos/calibration-latest.json` 的候选采纳率/误报回落率）
7. [x] N33 候选采纳归因画像（`make channel-chaos-calibration` 新增 `tag_coverage`、`missing_scenario_tags`、`adoption_by_channel`、`matrix_unseen_candidates`，可直接识别 matrix 标记缺口与通道采纳分布）
8. [x] N37 候选采纳优先级画像（`make channel-chaos-calibration` 新增 `pending_by_channel`、`priority_candidates`，按严重度/断连/复发频次评分并给出推荐回填动作）

当前状态：N37 已闭环，运行时治理具备“成本热点 + 压缩压力 + 压缩质量漂移 + 降级观测 + 演练门禁 + 阈值分层抑噪 + 阈值画像回归 + trace 抽样候选 + 周度校准指标 + 采纳归因画像 + 采纳优先级画像 + 外部策略异常归因 + 策略异常阈值治理”闭环能力。

## 4. 与 OpenClaw 研究报告对比（2026-03-02 UTC）

对照 `../cs-note/ai/agent/openclaw_research_report.md`：

- 已对齐：多通道网关、会话/子代理编排、工具协议与安全门禁、memory 检索、release-gate 基线、服务分层边界、N16 风险 watchlist 自动告警、N17 风险巡检 benchmark + 漂移分级 runbook、N18 场景基准矩阵与竞品月度追踪链路、N19 参数级配置治理门禁、N20 月度治理节奏自动化。
- 本轮新增对齐：补齐 N37，`make channel-chaos-calibration` 新增 `pending_by_channel` 与 `priority_candidates`，可按严重度/断连/复发频次对待采纳候选排序并输出推荐回填动作。
- 当前缺口：研究报告 5.2 建议项保持全量落地；5.1 已形成“成本治理 + 压缩质量漂移治理 + 通道降级观测 + chaos drill 门禁 + 阈值分层抑噪 + 阈值画像回归 + trace 抽样候选 + 周度校准指标 + 采纳归因画像 + 采纳优先级画像 + 外部策略异常归因 + 策略异常阈值治理”十二段闭环，当前无阻塞级功能缺口。
- 下一步：持续周度执行 `make channel-chaos-calibration` 与策略异常巡检，优先处理 `priority_candidates` 高分条目并回填 `config/channel-chaos-matrix.json` 的 `source_candidate`，并结合 `cost.compaction_quality.drift_share` 与 `provider_policy_incidents.by_category` 热点继续校准相关阈值。

## 5. 执行规则

- 单需求闭环：每次只推进一个需求到"编码 + 测试 + 文档"完成态。
- 文档同步：需求状态变化后，必须同步 `docs/features.md` 与本文件。
- 变更可回滚：涉及存储、会话、任务调度的改动必须附带回滚说明。

## 6. 失败记录与优先重试

- 2026-03-02（UTC）：本轮首次执行 `gh pr create` 误用 `-C` 参数（当前 gh 版本不支持），导致 PR 创建失败；已切换为在仓库工作目录执行 `gh pr create --body-file` 重试成功，后续统一避免依赖 `gh -C`。
- 2026-03-02（UTC）：本轮 `git push -u origin feat/n33-channel-chaos-source-coverage` 前 3 次失败（`Failure when receiving data from the peer` / `Failed to connect to github.com:443`），第 4 次重试成功；后续同类网络抖动继续按“至少 4 次重试 + 间隔连通性探测”策略执行。
- 2026-03-02（UTC）：`gh pr create --body` 初次执行因 markdown 反引号被 shell 命令替换导致 PR 描述注入失败；已改为 `--body-file` 重试修复（PR #99），后续创建 PR 统一使用 body 文件避免复发。
- 2026-03-01（UTC）：本轮 `git fetch origin --prune` 前 2 次失败（`Failure when receiving data from the peer`），第 4 次重试恢复；后续已完成远端同步与开发链路。
- 2026-03-01（UTC）：`git push -u origin feat/p2-n16-risk-watchlist-automation` 连续两次超时失败（无法连接 `github.com:443`），导致 PR/merge 链路阻塞；已在后续轮次恢复并完成 N16 合并。
- 2026-03-02（UTC）：`make risk-benchmark` 初次执行因缺少 N18 基准文件失败（`config/scenario-benchmark-matrix.json`、`config/competitor-tracking.json`）；已在同轮补齐并通过门禁，无待重试项。
- 2026-03-02（UTC）：`make competitor-tracking-refresh` 在未配置 `GH_TOKEN/GITHUB_TOKEN` 时触发 GitHub API 403 rate limit；已改为默认降级不中断并记录 warning，下一轮优先在带 token 环境执行一次完整刷新。
- 2026-03-01（UTC）：本轮执行 `git fetch origin` 连续超时（无输出后被超时终止），无法确认远端 `master` 最新提交；已改为基于本地 `master` 继续开发，下一轮优先重试网络连通后再同步远端。
- 2026-03-01（UTC）：`git push -u origin feat/n21-session-cost-pressure-alerts` 曾两次失败（`Failure when receiving data from the peer` / `Failed to connect to github.com:443`）；已在 2026-03-01 晚间恢复并完成 PR #89 合并。
- 2026-03-02（UTC）：本轮首次执行 `go test ./...` 失败（本机默认 `GOTOOLCHAIN=local` 且 `GOSUMDB=off`，无法拉取 `go1.25.7` 工具链）；已切换为 `GOTOOLCHAIN=auto GOSUMDB=sum.golang.org` 重试并通过，后续测试统一沿用该环境参数。
- 2026-03-02（UTC）：`gh pr merge` 初次执行失败（仓库禁用 merge commit / auto-merge，且分支策略阻止普通 squash）；已在同轮改用 `gh pr merge --squash --admin` 完成合并，后续轮次优先按该策略执行。
