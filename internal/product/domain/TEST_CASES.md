# Test Cases

## 覆盖范围

- Product 归一化、ID/slug/version/status/visibility/owner 校验、active 主 Agent 约束。
- ProductDraft 输入、草稿、主 Agent、worker matrix 归一化与校验。
- TravelGuide 创建、修订、travel product 绑定、天数上限、内容与偏好列表归一化。

## 边界

- Product CRUD、发布和 Travel 服务编排由 `internal/product/application` 覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/product/domain`
