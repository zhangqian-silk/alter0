# `.alter0` 任务历史与 Memory 查阅规范

> 版本：v1.0  
> 状态：Draft / Requirement Baseline  
> 归属需求：R-032

## 1. 目标

统一任务运行态数据的落盘结构、保留策略与前端查阅方式，确保：

- 运行态数据与业务代码隔离
- 任务摘要可跨会话检索
- 任务详情可按 `task_id` 下钻

## 2. 运行态目录规范

所有任务历史数据默认写入 `.alter0/`。

```text
.alter0/
  tasks/
    index.json
    {task_id}/
      meta.json
      logs.jsonl
      artifacts.json
  memory/
    YYYY-MM-DD.md
    long-term/
      YYYY-MM-DD.md
```

### 2.1 文件职责

- `tasks/index.json`：任务全局索引（用于检索与分页）
- `tasks/{task_id}/meta.json`：任务状态机快照与基础字段
- `tasks/{task_id}/logs.jsonl`：阶段日志（append-only，便于流式追踪）
- `tasks/{task_id}/artifacts.json`：产物列表与引用地址
- `memory/YYYY-MM-DD.md`：当日任务摘要（面向跨会话理解）
- `memory/long-term/YYYY-MM-DD.md`：长期候选摘要（面向长期召回）

## 3. 数据字段约束（最小集）

### 3.1 Task Meta

- `task_id`
- `session_id`
- `source_message_id`
- `task_type`
- `status`（`queued/running/success/failed/canceled`）
- `created_at`
- `finished_at`
- `retry_count`

### 3.2 Task Summary（写入 memory）

- `task_id`
- `goal`
- `result`
- `status`
- `finished_at`
- `tags`

## 4. 回链与一致性规范

1. 摘要记录必须包含 `task_id`，支持从 Memory 视图回查任务详情。
2. 任务详情需展示其对应摘要引用，支持反向定位。
3. 若摘要层与任务主数据冲突，以 `meta.json` 与 `logs.jsonl` 为准，并触发摘要重建。

## 5. 留存策略

- `logs.jsonl` 默认保留 90 天（可配置）。
- `index.json` 与 `meta.json` 保留全生命周期索引。
- 天级摘要可长期保留；归档后仍保留 `task_id` 映射。

## 6. 前端 Memory 模块查阅规范

在 `Agent -> Memory` 中新增“任务历史”查阅能力：

1. 默认显示摘要列表（按时间倒序）。
2. 支持筛选：`status`、`task_type`、`time_range`。
3. 点击摘要项后按需加载详情：
   - 基础信息（来自 `meta.json`）
   - 阶段日志（来自 `logs.jsonl`）
   - 产物引用（来自 `artifacts.json`）
4. 默认不自动展开原始日志，防止信息过载。

## 7. MVP 接口与字段拆解

### 7.1 后端接口（MVP）

- `GET /api/memory/tasks`：摘要列表 + 分页
- `GET /api/memory/tasks/{task_id}`：任务详情（meta + summary_refs）
- `GET /api/memory/tasks/{task_id}/logs`：日志游标读取
- `GET /api/memory/tasks/{task_id}/artifacts`：产物列表
- `POST /api/memory/tasks/{task_id}/rebuild-summary`：摘要重建

### 7.2 前端字段（MVP）

- 摘要列表：`task_id`、`task_type`、`goal`、`result`、`status`、`finished_at`、`tags`
- 详情基础：`session_id`、`source_message_id`、`progress`、`retry_count`、`created_at`
- 日志项：`seq`、`stage`、`level`、`message`、`created_at`
- 产物项：`artifact_type`、`uri`、`summary`、`created_at`
- 页面状态：`isLoading`、`isLogLoading`、`hasMoreLogs`、`errorCode`、`activeTaskId`

## 8. Git 隔离原则

`.alter0/` 属于运行态目录，默认由 `.gitignore` 忽略，不参与业务代码提交。

## 9. 验收口径

1. 任务执行后 `.alter0/tasks/*` 与 `.alter0/memory/*` 按规范生成。
2. Memory 页面可展示任务摘要并按 `task_id` 下钻详情。
3. 摘要与详情支持双向跳转。
4. 运行态数据不出现在常规 Git 提交中。
