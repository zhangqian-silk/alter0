# OpenClaw PR 日报（2026-03-02 UTC）

- 数据源：`openclaw/openclaw` GitHub Pulls API（created desc）
- 统计口径：当天新增且与 alter0 对齐价值高的 PR
- 说明：当天有效 PR 远超 5 条，本文件优先收录 6 条“行为变化明显/可落地”条目

## PR #31221

- 仓库：openclaw/openclaw
- 编号：#31221
- 标题：feat(agents): add mtime + TTL invalidation to bootstrap file cache
- URL：https://github.com/openclaw/openclaw/pull/31221
- 作者：Sid-Qin
- 标签：agents, size: S, experienced-contributor
- 时间：created 2026-03-02T03:15:57Z
- 状态：open
- 核心变更点：
  - 为 bootstrap 文件缓存增加 mtime 校验，文件变化后自动失效重载。
  - 增加 5 分钟 TTL 兜底，避免“永不过期缓存”导致长期陈旧上下文。
  - 目标覆盖 AGENTS/SOUL/TOOLS/USER/MEMORY/HEARTBEAT 等工作区关键文件。
  - 不改现有 API 形态（`clearBootstrapSnapshot` 等保持兼容）。
- 对 alter0 的具体影响：
  - alter0 若长期保留 session 级缓存，也会遇到“文档已改、会话仍读旧快照”的一致性问题。
- 建议落地动作：
  - 在 alter0 session/bootstrap 缓存实现中引入 `mtime + TTL` 双失效策略并补回归测试。

## PR #31220

- 仓库：openclaw/openclaw
- 编号：#31220
- 标题：feat(config): add `openclaw config validate` and improve startup error messages
- URL：https://github.com/openclaw/openclaw/pull/31220
- 作者：Sid-Qin
- 标签：cli, size: S, experienced-contributor
- 时间：created 2026-03-02T03:13:29Z
- 状态：open
- 核心变更点：
  - 新增 `openclaw config validate`，可在启动前验证配置合法性。
  - 支持 `--json` 输出，便于 CI/自动化流程消费。
  - 启动期 `loadConfig` 错误信息附带具体 key path，定位效率提升。
  - 范围控制明确：不改变现有 `config set/patch` 写入路径校验语义。
- 对 alter0 的具体影响：
  - 可把“配置错误导致重启风暴”前移到发布前门禁，降低线上回滚压力。
- 建议落地动作：
  - 新增 `make config-validate-preflight`，并将结果写入 `output/config/validate-latest.json`。

## PR #31210

- 仓库：openclaw/openclaw
- 编号：#31210
- 标题：feat(web-search): add Metaso and Qwen as web_search providers
- URL：https://github.com/openclaw/openclaw/pull/31210
- 作者：barlowliu
- 标签：agents, size: L
- 时间：created 2026-03-02T03:02:54Z
- 状态：open
- 核心变更点：
  - 为 `web_search` 扩展 Metaso、Qwen 两个 provider。
  - 覆盖配置 schema、类型定义、运行期 provider 选择与测试。
  - 明确针对中国网络环境可达性与中文搜索质量问题。
  - 保持对既有 provider 的兼容，不引入破坏性 config 变更。
- 对 alter0 的具体影响：
  - alter0 的研究链路若只依赖单一搜索源，易受 API Key/区域网络波动影响。
- 建议落地动作：
  - 建立 provider 优先级与失败回退链（Brave -> Perplexity -> Qwen/Metaso -> web_fetch）。

## PR #31217

- 仓库：openclaw/openclaw
- 编号：#31217
- 标题：feat(memory): change QMD searchMode default from "search" to "query"
- URL：https://github.com/openclaw/openclaw/pull/31217
- 作者：Sid-Qin
- 标签：size: XS, experienced-contributor
- 时间：created 2026-03-02T03:09:23Z
- 状态：open
- 核心变更点：
  - QMD 默认检索模式由 BM25 `search` 改为混合检索 `query`。
  - 目标是修复多词自然语言查询召回空结果的问题。
  - 对显式配置 `search` 的用户保持兼容。
  - 在 `search` 模式空结果场景提供诊断警告。
- 对 alter0 的具体影响：
  - memory 召回质量可能显著提升，但首查延迟与资源占用特征会变化。
- 建议落地动作：
  - 增加 `query/search` 双模式样本回放基线，纳入周度成本与质量对比。

## PR #31218

- 仓库：openclaw/openclaw
- 编号：#31218
- 标题：test(web_search): lock Brave endpoint to /res/v1 and close #31142
- URL：https://github.com/openclaw/openclaw/pull/31218
- 作者：stakeswky
- 标签：agents, size: XS, trusted-contributor
- 时间：created 2026-03-02T03:09:44Z
- 状态：open
- 核心变更点：
  - 补充 Brave endpoint 回归测试，防止旧路径回退。
  - 明确 runtime 代码已经使用新路径，重点在“防倒退”。
  - 通过最小测试变更保障 web_search 稳定性。
- 对 alter0 的具体影响：
  - 研究同步强依赖 web_search，接口路径漂移会直接影响采集可用性。
- 建议落地动作：
  - 在 alter0 工具层增加 provider endpoint 健康检查与错误分类指标。

## PR #31213（状态变更样本）

- 仓库：openclaw/openclaw
- 编号：#31213
- 标题：fix(discord): prevent wildcard component registration collisions
- URL：https://github.com/openclaw/openclaw/pull/31213
- 作者：steipete
- 标签：channel: discord, maintainer, size: S
- 时间：created 2026-03-02T03:08:19Z
- 状态：closed（merged at 2026-03-02T03:08:32Z）
- 核心变更点：
  - 修复 Discord wildcard `customId="*"` 冲突导致的组件回调丢失。
  - 统一 sentinel helper，保持旧路径兼容解析。
  - 解决 Unknown component 类故障，提升交互稳定性。
- 对 alter0 的具体影响：
  - alter0 若后续扩展 Discord 交互组件，需避免 wildcard 标识复用冲突。
- 建议落地动作：
  - 在适配层规范 wildcard sentinel 生成策略并补集成测试。
