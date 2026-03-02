# OpenClaw Issue 日报（2026-03-02 UTC）

- 数据源：`openclaw/openclaw` GitHub Issues API（created desc，过滤 PR）
- 统计口径：当天新增且与 alter0 对齐价值高的 Issue
- 说明：当天有效 Issue 超过 5 条，本文件优先收录 6 条可直接驱动工程动作的条目

## Issue #31222

- 仓库：openclaw/openclaw
- 编号：#31222
- 标题：WhatsApp voice note media download produces truncated/corrupt files
- URL：https://github.com/openclaw/openclaw/issues/31222
- 作者：emilywinslow-source
- 标签：无
- 时间：created 2026-03-02T03:17:11Z
- 状态：open
- 核心问题点：
  - WhatsApp 语音消息下载后文件尺寸异常偏小（约 1KB），内容疑似截断。
  - ffmpeg 解码后幅度接近 0，导致转写为空或幻觉文本。
  - 文本与图片消息正常，问题集中在 voice media 链路。
  - 影响范围覆盖群聊与私聊，非单 sender 偶发。
- 对 alter0 的具体影响：
  - alter0 若引入语音工作流，现有“下载成功即可用”判断不够，需完整性校验。
- 建议落地动作：
  - 增加媒体质量探针（最小字节阈值、时长/波形校验、转写置信度下限）并接入告警。

## Issue #31219

- 仓库：openclaw/openclaw
- 编号：#31219
- 标题：[Bug]: Browser control fails when HTTP_PROXY / HTTPS_PROXY / ALL_PROXY is set
- URL：https://github.com/openclaw/openclaw/issues/31219
- 作者：BUGKillerKing
- 标签：bug, regression
- 时间：created 2026-03-02T03:09:52Z
- 状态：open
- 核心问题点：
  - 配置代理环境变量后，CDP 探测被错误走代理导致本地 loopback 连接失败。
  - 浏览器已启动但 `openclaw browser start` 超时，表现为“可启动不可控”。
  - 期望框架自动处理 `no_proxy` 里的 `127.0.0.1/localhost/::1`。
  - 属于回归问题，容易在企业网络环境放大。
- 对 alter0 的具体影响：
  - alter0 的 browser 自动化在代理环境同样可能出现“假离线”故障。
- 建议落地动作：
  - 在启动自检中增加 loopback + proxy 组合探测，发现异常时给出修复建议并降级到非 browser 路径。

## Issue #31212

- 仓库：openclaw/openclaw
- 编号：#31212
- 标题：message:received hook does not fire for Discord guild channel messages
- URL：https://github.com/openclaw/openclaw/issues/31212
- 作者：wave-workflow
- 标签：无
- 时间：created 2026-03-02T03:08:01Z
- 状态：open
- 核心问题点：
  - `message:received` hook 在 Discord guild 场景缺失。
  - DM/其他渠道可触发，guild 直达 session 时事件旁路。
  - 导致依赖 hook 的审计/聚合/自动化逻辑出现“静默漏数”。
- 对 alter0 的具体影响：
  - 若 alter0 依赖统一 hook 汇总指标，会出现通道偏差与错误决策。
- 建议落地动作：
  - 建立“入口事件覆盖率”监控，按渠道/会话类型核对 hook 触发率并设定最小覆盖门槛。

## Issue #31216

- 仓库：openclaw/openclaw
- 编号：#31216
- 标题：Avatar image from IDENTITY.md not loading in webchat
- URL：https://github.com/openclaw/openclaw/issues/31216
- 作者：xishandong
- 标签：无
- 时间：created 2026-03-02T03:09:21Z
- 状态：open
- 核心问题点：
  - `IDENTITY.md` 中相对路径头像在 webchat 未正常加载。
  - 文件存在且可读，但前端资源映射/静态路由疑似缺失。
  - 影响用户对 agent 身份识别与体验一致性。
- 对 alter0 的具体影响：
  - alter0 后续若提供控制台身份可视化，也会遇到 workspace 静态资源映射一致性问题。
- 建议落地动作：
  - 增加资源路径规范检查（相对路径解析、静态映射、404 采样）并在 UI 健康检查中暴露。

## Issue #31214

- 仓库：openclaw/openclaw
- 编号：#31214
- 标题：Security: Possible phishing/typosquatting domain (og-openclaw.com)
- URL：https://github.com/openclaw/openclaw/issues/31214
- 作者：AZRIELA
- 标签：无
- 时间：created 2026-03-02T03:08:26Z
- 状态：open
- 核心问题点：
  - 出现疑似仿冒域名，可能用于钓鱼或恶意投放。
  - 与官方域名近似，用户易混淆。
  - 社区请求官方确认授权状态并发布风险提示。
- 对 alter0 的具体影响：
  - alter0 的外链白名单与来源可信判断需要覆盖品牌仿冒域风险。
- 建议落地动作：
  - 在风险 watchlist 增加“域名仿冒”类目并落地告警模板（含隔离与通告流程）。

## Issue #31208

- 仓库：openclaw/openclaw
- 编号：#31208
- 标题：Feature: tools.byChannel for channel-specific tool profiles
- URL：https://github.com/openclaw/openclaw/issues/31208
- 作者：aspenas
- 标签：无
- 时间：created 2026-03-02T02:58:18Z
- 状态：open
- 核心问题点：
  - 当前工具集全量注入所有会话，造成上下文 token 负担。
  - 非交互会话（如 cron）携带 browser/canvas 等重工具价值低。
  - 提议 `tools.byChannel` 支持通道级 profile/deny 合并策略。
- 对 alter0 的具体影响：
  - alter0 的成本治理可直接受益于“会话类型分层工具注入”。
- 建议落地动作：
  - 设计 `tools.profile.by_surface`（cron/chat/web）并对 token 成本、成功率、平均时延做 A/B 验证。
