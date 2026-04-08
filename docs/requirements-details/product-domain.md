# Product Domain Requirements

> Last update: 2026-04-08

## 领域边界

Product Domain 负责平台内业务产品的建模、草稿生成、审核发布、Product 总 Agent 执行、Workspace、详情页空间、独立 HTML 页面和首个内置 `travel` 产品域。

## 核心对象

| 对象 | 职责 |
| --- | --- |
| `Product` | 产品定义、状态、入口、主 Agent、产物类型与知识源 |
| `ProductDraft` | Product 草稿、审核状态、生成模式与版本 |
| `ProductAgentDraft` | 草稿中的主 Agent 定义 |
| `ProductMasterAgent` | 已发布 Product 的唯一执行入口 |
| `ProductWorkspace` | Product 工作台摘要、主 Agent 对话与详情页空间 |
| `ProductSpace` | Product 下的具体详情页空间 |
| `TravelGuide` | travel 产品域中的结构化攻略 |
| `ProductDiscovery` | main Agent 对 Product 的发现、选择与调度元数据 |

## Product 目录

### 定义

- Product 是业务产品域的一等对象，不使用 `App` 作为同级领域模型命名。
- Product 最小字段包括 `product_id`、`name`、`slug`、`summary`、`status`、`visibility`、`owner_type`、`master_agent_id`、`entry_route`、`tags`、`version`、`created_at`、`updated_at`。
- Product 保留稳定 `product_id`，供 Agent、任务、产物、会话与路由统一引用。

### 来源

- 系统内置 Product 由服务注册，控制面只读展示，不允许删除或覆盖。
- 用户管理 Product 支持创建、编辑、停用、归档和删除。
- Product 更新由服务端维护版本。

### 展示

- `Products` 页面提供 `Workspace` 与 `Studio` 视图。
- Product 列表展示基础信息、状态、总 Agent、入口路由、产物类型与知识源摘要。
- Product 详情展示基础信息、Agent 结构、能力摘要、产物摘要、Workspace 摘要与详情页空间。
- Product 草稿、禁用态和空态必须有明确展示，避免误入生产执行入口。

## Draft Studio

### 草稿生成

- 平台提供内置 `product-builder`，用于根据产品目标和约束生成 Product 草稿。
- 输入至少覆盖名称、目标、目标用户、核心能力、约束、预期产物和集成要求。
- 输出至少覆盖 Product 基础定义草案、主 Agent 草案、可复用 Prompt/Skill 沉淀建议、能力边界、默认资料结构和产物类型定义。

### 单主 Agent 默认模型

- 新生成 Product 默认采用单主 Agent。
- 领域规则、章节顺序、稳定命名与呈现约束优先沉淀到 `master agent.system_prompt` 或私有 Skill。
- `matrix/generate` 保留为兼容增量扩展入口，但默认不自动补出 worker matrix。
- 历史 supporting agents 字段兼容读取，但新草稿不默认新增多 Agent 矩阵。

### 审核与发布

- 草案生成后进入草稿态，不自动发布。
- Draft Studio 支持查看、编辑、审核与发布草稿。
- 发布时自动物化主 Agent，并写入托管 Product 定义。
- 发布链路自动补齐稳定工具面、领域 Skill、`memory` Skill 与标准 Memory Files，确保旧草稿或手工编辑草稿回到稳定执行模型。

## Product 总 Agent

### 执行入口

- 每个已发布 Product 绑定且仅绑定一个 `master_agent_id`。
- Product 总 Agent 是该 Product 的统一需求入口和任务编排中心。
- Product 总 Agent 采用 Agent 协助 / Codex 执行模型，由总 Agent 直接收口结果。

### 职责

- 识别请求是否属于本 Product。
- 解析用户目标、约束和上下文。
- 直接产出领域结果与结构化产物。
- 决定是否兼容调用 supporting agents。
- 汇总最终结果、子任务摘要和产物引用。

### 委派边界

- Product 默认不依赖多 Agent 编排。
- 若存在 supporting agents，必须具备明确职责边界，不允许多个 Agent 对同一职责无约束重叠。
- 委派链路保留 `product_id`、`master_agent_id`、`worker_agent_id`、`task_id`、`delegation_reason` 与 `delegation_order`。
- 编排层限制递归深度、自委派和循环依赖。
- supporting agent 不直接越权调用其他 Product 的 supporting agents；跨 Product 调用回到 `Alter0/main` 或上层协调器裁决。

## 跨 Product 调度

### Product 发现

- 默认 `main` Agent 具备 Product 目录读取与意图匹配能力。
- Product 信息查询优先返回公开摘要，不暴露内部草稿、凭据或审核记录。
- Product 调度元数据至少包括 `matched_product_ids`、`selected_product_id`、`selection_reason`、`master_agent_id`、`product_execution_mode`。

### Product 执行

- 执行型请求命中 Product 后，系统自动切换到目标 Product 总 Agent。
- 请求注入 `product_context` 与 `product_discovery`。
- 多 Product 请求可按顺序或并行调度多个 Product 总 Agent，最终结果必须区分来源。
- Product 被禁用、未发布或无可用总 Agent 时，返回明确原因并建议可用 Product 或退回通用处理路径。

## Workspace 与公开入口

- 已发布且公开 Product 提供公共目录与详情入口。
- Product Workspace 返回 Product 概览、主 Agent 摘要、详情页空间列表和具体空间详情。
- 每个详情页空间可映射独立 HTML 页面。
- Product 消息入口自动绑定目标 Product 总 Agent。
- 公开入口仅暴露 `active + public` Product。

## Travel 产品域

### 定位

- `travel` 是首个内置 Product。
- `travel` 绑定唯一 `travel-master`，作为旅游领域默认执行入口。
- `travel` 面向按城市聚合的旅游攻略、行程、地铁、美食和地图图层场景。

### 结构化攻略

- `TravelGuide` 至少包含 `city`、`days`、`travel_style`、`must_visit`、`avoid`、`pois`、`metro_lines`、`daily_routes`、`foods`、`notes`、`map_layers`。
- 攻略结果同时支持用户可读文案和结构化数据。
- 后续修订必须基于已有攻略，不丢弃上下文重建。
- 地图高亮、路线绘制和 POI 点位在结构中预留标准字段。

### Workspace Chat

- `travel` Workspace 支持与 `travel-master` 对话。
- 创建或修改请求同步到具体城市页空间。
- 用户选择城市页后，后续修改默认作用于当前城市页。
- `travel-master` 执行链不可用、模型响应异常或 CLI fallback 不可用时，服务端自动切换到本地规则解析。

### 私有 Skill

- `travel-master` 使用 `.alter0/agents/travel-master/SKILL.md` 作为城市页生成规则、章节组织与 HTML 呈现约定的规则簿。
- 稳定、可复用、影响后续多个城市页的偏好可写入该 Skill。
- 某次出行或某个城市的一次性约束只更新目标城市页，不写入 Skill。

## 接口边界

### 公共 Product

- `GET /api/products`
- `GET /api/products/{product_id}`
- `POST /api/products/{product_id}/messages`
- `POST /api/products/{product_id}/messages/stream`
- `GET /api/products/{product_id}/workspace`
- `GET /api/products/{product_id}/workspace/spaces/{space_id}`
- `GET /products/{product_id}/spaces/{space_id}.html`

### 控制面 Product

- `GET /api/control/products`
- `POST /api/control/products`
- `PUT /api/control/products/{product_id}`
- `DELETE /api/control/products/{product_id}`
- `GET /api/control/products/{product_id}`
- `POST /api/control/products/generate`
- `GET /api/control/products/drafts`
- `GET /api/control/products/drafts/{draft_id}`
- `PUT /api/control/products/drafts/{draft_id}`
- `POST /api/control/products/drafts/{draft_id}/publish`
- `POST /api/control/products/{product_id}/matrix/generate`

### Travel

- `GET /api/products/travel`
- `GET /api/products/travel/workspace`
- `GET /api/products/travel/workspace/spaces/{space_id}`
- `POST /api/products/travel/workspace/chat`
- `POST /api/products/travel/guides`
- `GET /api/products/travel/guides/{guide_id}`
- `POST /api/products/travel/guides/{guide_id}/revise`
- `GET /products/travel/spaces/{space_id}.html`

## 依赖与边界

- Product 总 Agent 复用 Agent Capability & Memory 领域执行模型。
- Workspace 与产物下载复用 Task, Terminal & Workspace 的产物交付底座。
- Product 控制面归属 Control 领域，但 Product 业务规则归属本领域。
- `travel` 是 Product 模式的领域实例，不应把 travel 规则反向写入通用 Agent 规则。

## 验收口径

- Products 页面可管理内置与用户 Product，并区分只读与可维护来源。
- Draft Studio 可生成、编辑、审核并发布单主 Agent Product 草稿。
- 已发布 Product 仅暴露唯一总 Agent 执行入口。
- `main` Agent 可发现并调度目标 Product。
- `travel` 可创建和修订城市页，提供独立 HTML 页面。
- `travel-master` 失败时 Workspace Chat 仍可通过本地解析完成创建或修订。
