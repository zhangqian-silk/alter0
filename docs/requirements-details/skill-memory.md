# Skill 与原生记忆

## 范围

本领域维护 Codex 原生 Skill 的全局生命周期，以及 Codex 原生 Memories 的全局设置与运行状态。会话级 Skill/Memory 选择、alter0 Markdown MemoryContext、日级/长期记忆整理和记忆维护定时任务均不属于稳定能力。

## 核心对象

| 对象 | 职责 |
| --- | --- |
| `Skill` | 控制面中的全局能力配置，记录启用状态、可见性和 `SKILL.md` 来源 |
| `NativeSkillSource` | 供 Codex 安装的标准 file-backed Skill 来源 |
| `NativeSkillReconciler` | 把有效公有 Skill 原子同步到 Codex 用户级 Skill 目录 |
| `CodexHome` | 当前服务用户共享的 Codex 认证、配置、线程、Skills 与 Memories 根目录 |
| `NativeMemoriesSettings` | Codex `config.toml` 中的总开关、生成开关与召回开关 |

## 原生 Skill

- 每个可安装 Skill 必须以 `docs/skills/<skill_id>/SKILL.md` 为入口，并包含有效的 `name` 与 `description` frontmatter。
- 服务启动完成内置 Skill 注册后执行一次全量同步；Skill 启用、停用、更新或删除后立即重新同步。
- 仅启用、非私有且 file-backed 的 Skill 可安装。Codex 根据描述隐式匹配，用户可用 `$skill-name` 显式调用。
- Chat Composer、Details、输入 API 与 turn 持久化均不保存 Skill 选择；历史 `skill_context` 可解码，但新快照不再写入。
- 同步目标使用 alter0 管理标记、临时目录和原子 rename。停用或删除只清理带管理标记的目录；用户自行安装的 Skill 永远保留。
- 目标存在但不受 alter0 管理、来源不是 `SKILL.md`、frontmatter 无效或来源包含不安全 symlink 时，同步失败并拒绝对应生命周期修改，保留最后一个有效安装版本。

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
- 启用的公有 Skill 可被 Codex 隐式或显式调用；停用后新 Codex 进程不可见。
- 启动 Turn 不修改仓库 `AGENTS.md`，不创建会话 `codex-home`、`skills.md` 或 Memory 注入目录。
