# Skill 与原生记忆

## 范围

本领域维护两个 Alter0 业务 Skill 的全局生命周期、Codex 实际发现 Skill 的只读目录，以及 Codex 原生 Memories 的全局设置与运行状态。第三方 Skill 生命周期、会话级 Skill/Memory 选择、alter0 Markdown MemoryContext、日级/长期记忆整理和记忆维护定时任务均不属于稳定能力。

## 核心对象

| 对象 | 职责 |
| --- | --- |
| `Skill` | Alter0 控制面中的业务 Skill 配置，记录启用状态、可见性和 `SKILL.md` 来源 |
| `NativeSkillSource` | 供 Codex 安装的标准 file-backed Skill 来源 |
| `NativeSkillReconciler` | 把有效公有 Skill 原子同步到 Codex 用户级 Skill 目录 |
| `NativeSkillCatalog` | Codex app-server 对固定 cwd 实际发现的 Skill、scope、路径类别、依赖和解析错误读模型 |
| `CodexHome` | 当前服务用户共享的 Codex 认证、配置、线程、Skills 与 Memories 根目录 |
| `NativeMemoriesSettings` | Codex `config.toml` 中的总开关、生成开关与召回开关 |

## 原生 Skill

- alter0 只维护 `preview-publish` 与 `travel`，入口固定为 `docs/skills/<skill_id>/SKILL.md`，并包含有效的 `name` 与 `description` frontmatter。
- 服务启动完成两个内置 Skill 注册后执行一次全量同步；Skill 启用、停用、更新或删除后立即重新同步。启动注册必须保留用户已明确设置的停用状态。
- 仅启用、非私有且 file-backed 的 Alter0 业务 Skill 可由 reconciler 安装。Codex 根据描述隐式匹配，用户可用 `$skill-name` 显式调用。
- Chat Composer、Details、输入 API 与 turn 持久化均不保存 Skill 选择；历史 `skill_context` 可解码，但新快照不再写入。
- 同步目标使用 alter0 管理标记、临时目录和原子 rename。停用或删除只清理带管理标记的目录；用户自行安装的 Skill 永远保留。
- 历史通用 Capability 和带有效 marker 的 `alter0-*` 通用副本在迁移时清理；无 marker 的 Agent、用户、仓库、管理员和系统 Skill 保留原状。
- 目标存在但不受 alter0 管理、来源不是 `SKILL.md`、frontmatter 无效或来源包含不安全 symlink 时，同步失败并拒绝对应生命周期修改，保留最后一个有效安装版本。

## Skill 目录

- `GET /api/control/skill-catalog` 返回 `project_skills`、`codex_skills` 与 `errors`。Alter0 区只包含两个业务 Skill；Codex 区不取得第三方生命周期所有权。
- Codex 目录必须调用当前活动 Codex CLI 的 app-server `skills/list`，请求固定服务 runtime root cwd 并设置 `forceReload=true`；不自行扫描目录或解析第三方 frontmatter。
- 绝对 path 只用于服务内部去重和分类。Web API 只返回 `alter0`、`user_agents`、`codex_home`、`repo`、`admin`、`system` 或 `other`，页面分别显示安全位置标签。
- 同名、同规范化 path 视为一项；同名、不同 path 全部保留并共享重复组。Alter0 业务 Skill 与外部同名项分别显示，平台不选择赢家。
- app-server 不可用时 Alter0 区继续返回，Codex 区报告 `catalog_unavailable`；单项解析错误只进入 `errors`，不得携带绝对路径或导致整个页面失败。

## 原生 Memories

- 服务启动通过 `codex features list` 检测 `memories` 能力；首次发现能力可用且配置键缺失时，把 `features.memories`、`memories.generate_memories` 与 `memories.use_memories` 默认写为 `true`，显式已有值必须保留。
- `Settings > Runtime` 提供用户级全局总开关、生成开关和召回开关，使用 Codex app-server `config/read` 与 `config/batchWrite` 直接读写活动 `config.toml`。
- 新建与续接 Codex Turn 只设置共享活动 `CODEX_HOME`，由 Codex 自行读取原生配置；alter0 不再追加 `--enable memories` 或 `-c memories.*` 命令行覆盖。
- 所有会话使用同一个活动 `CODEX_HOME`；Codex 自行选择可沉淀任务、提取事实、合并精炼并在后续任务中召回。
- alter0 不解析或写入 Codex 未公开的 Memories schema，不运行第二次模型整理，不保留 Markdown MemoryContext fallback。
- Runtime 状态同时返回可用性、当前开关、生成目录是否存在、生成文件数量和最近修改时间；响应不得包含活动 Home、绝对路径、认证信息或生成内容。功能缺失时明确显示兼容诊断，不启用旧记忆系统。
- 独立 Memory 设置分区和 `/api/memory/context` 不再保留；关闭总开关不会删除已有原生记忆文件，删除记忆必须是独立且明确确认的操作。
- Memory Maintenance 内置任务与 `memory` / `memory-maintenance` Skill 必须从历史控制和调度状态中清理；Session Cleanup 独立保留。

## 工作区与仓库边界

- 会话工作区保存附件、产物、repository checkout 与 alter0 会话状态；共享 `CODEX_HOME` 不放入会话目录。
- 绑定仓库后 Codex 的工作目录为 `<session workspace>/repo`，原生读取仓库自身 `AGENTS.md`；未绑定仓库时工作目录保持会话工作区。
- alter0 不在绑定仓库或会话根目录生成 Skill manifest、Memory 摘要或托管 `AGENTS.md`。
- 普通附件仍写入会话工作区。绑定仓库时，提示中的普通文件路径相对 `repo/` 计算；图片继续使用规范化绝对路径传给 Codex CLI。

## 验收

- 一个会话生成的 Codex Memory 可被另一个会话使用。
- 默认配置迁移只补齐缺失键，不覆盖用户已经明确关闭的总开关、生成或召回。
- 新建与 resume 命令都使用共享活动 Home，且不携带会话级 Memories 覆盖参数。
- Runtime 中四种组合配置可持久化并作用于后续所有 Turn；关闭总开关不删除已有文件。
- Chat 输入 payload 不含 `skill_ids`，界面没有会话级 Skill 入口。
- 启用的两个 Alter0 业务 Skill 可被 Codex 隐式或显式调用；停用后对应托管目录不可见，外部同名 Skill 不受影响。
- Skills 页面可以区分 Alter0 内置、`~/.agents/skills`、`$CODEX_HOME/skills`、repo、admin 与 system 来源；同名不同位置不折叠且不泄露服务器绝对路径。
- 启动 Turn 不修改仓库 `AGENTS.md`，不创建会话 `codex-home`、`skills.md` 或 Memory 注入目录。
