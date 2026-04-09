# Test Cases

## 覆盖范围

- DraftService 生成单主 Travel Master 草稿与 matrix 兼容行为。
- Product Service 内置 travel、托管 Product 创建与内置覆盖拒绝。
- TravelGuideService 创建、修订、排序、重复城市拒绝、Unicode guide ID。

## 边界

- Product、Draft、TravelGuide 领域校验由 `internal/product/domain` 覆盖。
- Product Web/API/HTML 页面由 `internal/interfaces/web` 覆盖。

## 执行

- `GOCACHE=/tmp/alter0-go-build-cache go test ./internal/product/application`
