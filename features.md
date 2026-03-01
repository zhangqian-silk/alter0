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

### N16 外部策略与供应链风险监测（P2）

- 目标：针对研究报告中的“模型/渠道政策漂移 + skill/plugin 供应链风险”建立最小自动监测闭环。
- 交付边界：
  - provider/channel 策略变更 watchlist（含异常提示）。
  - skill/plugin 来源与版本漂移审计。
  - 风险事件落盘与告警分级策略。

## 3. 优先级与执行队列

### P0（已完成）

1. [x] N9 异步任务取消语义修复（真实中断 + 状态一致性）
2. [x] N10 存储迁移安全升级（非破坏迁移 + 备份回滚）
3. [x] N14 测试稳定性加固（`make test-stability` + release-gate 接入）

### P1（已完成）

1. [x] N13 配置层依赖解耦（移除反向依赖 + `make check-config-boundary`）
2. [x] N15 服务模块化（`core -> service -> infra`，补充 `make check-service-boundary`）

### P2（当前主线）

1. [ ] N16 外部策略与供应链风险监测（provider/channel 策略漂移 + skill/plugin 来源审计）

当前下一项：`P2/N16`。

## 4. 与 OpenClaw 研究报告对比（2026-03-01）

对照 `../cs-note/ai/agent/openclaw_research_report.md`：

- 已对齐：多通道网关、会话/子代理编排、工具协议与安全门禁、memory 检索、release-gate 基线，以及调度域的 service 分层边界。
- 当前缺口：研究报告第 5 章指出的外部模型/渠道政策变化与 skill/plugin 供应链风险监测仍缺自动化闭环。
- 下一步：推进 N16，落地策略漂移 watchlist 与供应链审计告警，继续保持单需求闭环（编码 -> 测试 -> PR -> merge）。

## 5. 执行规则

- 单需求闭环：每次只推进一个需求到"编码 + 测试 + 文档"完成态。
- 文档同步：需求状态变化后，必须同步 `docs/features.md` 与本文件。
- 变更可回滚：涉及存储、会话、任务调度的改动必须附带回滚说明。
