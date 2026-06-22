# Skill & Memory Requirements

> Last update: 2026-06-08

## 领域边界

Skill & Memory 负责 CLI Runtime 的上下文注入、Skill 仓库、Markdown 记忆文件、会话摘要、长期记忆、天级记忆、项目记忆、任务摘要与记忆可视化。

该领域维护“执行前给 runtime 什么上下文”和“执行后沉淀什么记忆”。具体任务推理、会话内压缩、工具调用与执行策略由 Claude Code 或 Codex CLI 运行时完成。

## 核心对象

| 对象 | 职责 |
| --- | --- |
| `CLIRuntime` | 表示一次 Agent 请求使用的 CLI runtime 执行器 |
| `RuntimeProfile` | 表示本轮运行时、模型来源、工作区、Skill 与 Memory 注入配置 |
| `Skill` | 可复用产品能力说明、执行规则、交付要求与检查清单 |
| `SkillRepository` | 管理 `docs/skills/<skill_id>/SKILL.md` 及其脚本、参考文件和资产 |
| `MCPServer` | 可选外部 MCP Server 配置与 CLI 运行时注入来源 |
| `MemoryFile` | 可注入、可读写或只读的 Markdown 记忆文件 |
| `MemoryContext` | 执行前解析出的记忆文件、召回摘要和注入预算 |
| `ConversationSummary` | 会话归档摘要，作为跨会话记忆整理输入 |
| `ProjectMemory` | 面向项目或仓库的长期事实、约束与交付偏好 |
| `LongTermMemory` / `DailyMemory` | 用户级长期记忆与天级记忆 |

## CLI Runtime

### 运行时选择

- Agent 请求统一进入 CLI Runtime。
- 存在启用且健康的 Model Provider 时，运行时优先启动 `Claude Code + provider profile`。
- Model Provider 未配置或不可用时，运行时使用 `Codex Direct`；已进入 Claude 后执行失败直接返回错误，不自动改走 Codex。
- Web 对话显式选择 `Codex` 或消息 metadata 声明 `alter0.execution.engine=codex` 时，本轮直接使用 `Codex Direct`。
- Cron 任务、会话归档、系统记忆维护与普通用户消息复用同一运行时选择规则。

### 运行时注入

- 每个会话都有独立工作区，运行时只在当前工作区注入本轮所需上下文。
- Claude Code 运行时生成 `CLAUDE.md`、provider profile 环境、Skill 文件副本、Memory 文件副本或引用、MCP 配置和工作区事实。
- Codex Direct 运行时生成 `AGENTS.md`、独立 `CODEX_HOME/config.toml`、Skill 文件副本、Memory 文件副本或引用、MCP 配置和工作区事实。
- 注入内容必须包含会话身份、工作区路径、仓库路径、附件路径、产物路径、可写边界、选中 Skill、命中的 Memory 摘要和交付要求。
- 会话内上下文压缩由实际 CLI runtime 自身处理；alter0 保存原始消息、运行日志、最终结果和归档摘要。

### 会话入口

- Chat 是通用自然语言入口。
- Chat 是选择 Skill 组合的任务入口；代码开发、旅行攻略、结构化写作等业务能力以 Skill 组合和交付规则表达。
- Terminal 是前端手动选择的独立终端入口；自然语言 Auto 模式固定进入 Chat。
- 当前稳定入口模型面向一对一用户会话。

## Skill Repository

### Skill 配置

- Skill 配置包含 `skill_id`、显示名、描述、启停状态、排序、文件路径、可写属性和可选标签。
- 标准 Skill 使用 `docs/skills/<skill_id>/SKILL.md` 作为 file-backed 入口。
- Skill 目录可包含 `scripts/`、`references/`、`assets/` 等附属资源，运行时按当前任务需要复制到会话工作区。
- 默认公有 Skill 覆盖 `memory`、`preview-publish`、`frontend-design`、`doc-coauthoring`、`fullstack-developer`、`code-reviewer`、`webapp-testing`、`find-skills`、`test-driven-development`、`ui-ux-pro-max`、`code-simplifier`、`code-review`、`brainstorming` 与 `travel`；`memory-maintenance` 作为系统维护专用私有 Skill 保留。

### 业务 Skill

- 编码类任务通过用户选择全栈开发、测试驱动、评审、重构、预览发布与协作文档等现有 Skill 执行。
- `travel` Skill 维护城市行程、分类推荐池、路线表达、Codex 生成的行程地图图片、移动端 HTML 攻略、公开只读 `travel` 服务发布与交付检查。
- `memory-maintenance` 私有 Skill 维护系统记忆整理规则，包括摘要合并、长期事实提炼、过期信息处理、冲突消解和项目记忆更新，只由维护任务注入。
- 会话可以选择一组 Skill；运行时直接以 Claude Code CLI 或 Codex CLI 执行当前任务。

## Memory Files

### 文件结构

```text
AGENTS.md
SOUL.md
memory/
  USER.md
  MEMORY.md
  daily/YYYY-MM-DD.md
  projects/<project>.md
  conversations/<conversation_id>/summary.md
```

- `AGENTS.md` 保存仓库/工作区运行规则、工具纪律、路径边界和交付约束；它是规则型上下文，不是事实型记忆。
- `SOUL.md` 保存最高优先级强约束；启动参数 `mandatory-context-file` 可把该文件解析到自定义位置。
- `USER.md` 保存稳定用户偏好、身份信息、长期协作习惯和跨项目约束。
- `MEMORY.md` 保存跨会话长期事实、常用约定、已确认偏好和可复用经验；实际路径可由 `long-term-memory-path` 覆盖。
- `daily/<YYYY-MM-DD>.md` 保存当天重要上下文、阶段性进展和待整理事实；实际目录可由 `daily-memory-dir` 覆盖。天级记忆不是原始会话副本，只保留 CLI Runtime 筛选后的活跃上下文、候选事实和待验证事项，不保存逐轮 transcript、日志或任务输出原文。
- `projects/<project>.md` 保存项目级规则与阶段性上下文。
- `conversations/<conversation_id>/summary.md` 保存会话归档摘要。

用户可见 Markdown 保持可读结构，不写入 confidence、source、status、sensitivity 等机器元数据。需要检索加速时，可生成派生索引；Markdown 文件仍是真相源。

### 注入协议

- 执行前根据会话、项目、用户输入和选中 Skill 解析 `MemoryContext`。
- 执行注入、Web Memory 只读页面、任务摘要运行时和系统维护任务使用同一组已解析 root instructions、`daily-memory-dir`、`long-term-memory-path` 与 `mandatory-context-file`。
- 注入内容包含文件路径、存在状态、可写性、内容摘要、召回片段和截断标记。
- 单文件与总注入体积必须设置上限；超出预算时优先注入摘要、最近事实、强相关片段和项目关键约束。
- 本轮命中长期记忆时，执行前生成 Active Recall 摘要，作为隐藏上下文注入 CLI runtime。

### 写入时机

- 用户显式表达“记住”“以后都按这个来”“把这个作为偏好”时，当前 CLI runtime 可直接更新目标记忆文件。
- 会话完成或归档时，系统生成 `ConversationSummary`，记录目标、关键决策、已完成结果、未完成事项、文件/链接/任务引用和记忆候选。
- 持久记忆 Markdown 不由服务直接写入。服务侧会话记忆只保留在运行态和 `ConversationSummary` 中，用于恢复、召回和维护任务输入；不会把每轮会话、压缩片段或任务摘要直接写入天级记忆或长期候选 Markdown。
- 系统维护任务以 Scheduler 内置 Job 形式每日定时启动 CLI Runtime 并加载 `memory-maintenance` Skill，把会话摘要、天级记忆和项目记忆合并到长期记忆。该任务使用固定默认策略，不向用户暴露复杂调度或文件选择配置；内置 Job 不能删除，但可在 Scheduler 控制面停用或重新启用。
- 任务完成后生成或刷新任务摘要，供 Memory 页面展示和后续召回。

### 自动维护

- 记忆维护任务固定作为系统维护能力运行，并作为 Scheduler 内置 Job 注册，默认读取长期记忆、当日/昨日天级记忆、用户记忆、项目记忆与会话摘要候选。
- 维护结果需记录运行状态、开始/完成时间、下次运行时间、变更文件、失败错误；CLI Runtime 不可用时必须记录为失败，失败后可从 `Settings > Maintenance` 手动重试。
- 手动执行记忆维护与每日自动维护走同一执行链路，均通过 CLI Runtime 注入 `memory-maintenance` Skill，不提供额外文件范围或整理策略配置项。

### 摘要格式

会话摘要使用稳定 Markdown 结构：

```markdown
# Conversation Summary

## Goal

## Decisions

## Results

## Open Items

## Memory Candidates

## References
```

长期记忆写入使用自然语言短条目，优先记录稳定、可复用、会影响后续行为的信息。一次性任务参数、临时路径、过期状态和未确认推测只进入会话摘要或日记忆。

## Memory 页面

- 前端 `Skill -> Memory` 提供 `AGENTS.md`、`SOUL.md`、长期记忆、天级记忆、项目记忆、会话摘要、任务历史与说明文档入口。
- Memory 页面按安全 Markdown 渲染正文；路径、任务 ID、时间、状态等元数据保持纯文本或等宽字段展示。
- `GET /api/memory/context` 返回 `AGENTS.md` root instructions、`SOUL.md` 强约束、长期记忆、天级记忆、项目记忆和说明文档的只读聚合视图。
- `GET /api/memory/tasks` 与相关详情接口提供任务摘要、任务日志、产物引用和摘要重建。
- Memory 聚合接口默认只读展示；写入由 CLI Runtime 或记忆维护任务完成。

## 依赖与边界

- Runtime & Orchestration 负责把 Agent 请求送入 Runtime Resolver，并把执行结果回写会话。
- Conversation 负责消息持久化、历史恢复、结构化过程和最终回复展示。
- Task 负责后台执行、日志、产物和任务摘要。
- Control 负责 Skill、MCP、Model Provider、Codex Runtime 与 Cron 配置生命周期。

## 验收口径

- Agent 请求能按 Provider 优先级进入 Claude Code 或 Codex Direct。
- 每次运行前都能在会话工作区看到对应 CLI runtime 的上下文注入文件。
- Skill 文件从独立仓库目录注入，不依赖固定业务 Skill 实现。
- Memory Files 以 Markdown 主存存在，并能按会话、项目和日期被召回。
- 用户显式记忆、会话归档摘要和系统记忆维护三条写入路径可追踪。
- Memory 页面能只读展示 `AGENTS.md`、`SOUL.md`、长期记忆、日记忆、项目记忆、会话摘要和任务摘要。
