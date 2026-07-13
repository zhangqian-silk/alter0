# Codex Runtime 配额解析与思考强度透传设计

## 背景

Codex quota 接口的 `primary_window` 与 `secondary_window` 是位置字段，不保证分别代表 5 小时窗口和周窗口。当前部分 Pro 账号临时只返回一个 7 天窗口，导致现有实现把该窗口错放到 `hourly`，再把缺失的第二窗口展示成 `0%` 和 Go 零时间。

Codex `model/list` 已返回模型支持的思考强度原始值与顺序。前端当前把 `xhigh` 转换成 `Max`，当模型同时返回 `max` 与 `xhigh` 时产生两个同名选项。

## 决策

### 配额契约

- 保留现有 API 字段 `quota.hourly` 与 `quota.weekly`，不引入新的通用窗口数组。
- 从 quota 响应读取 `limit_window_seconds`，按窗口时长归类：
  - `18000` 秒归入 `hourly`，保持 JSON 兼容；界面文案显示为 `5 Hours / 5 小时`。
  - `604800` 秒归入 `weekly`。
- 不再根据 `primary_window` 或 `secondary_window` 的位置推断窗口类型。
- 当前缺失的窗口使用可空字段表达，并从 JSON 与界面中省略；不得生成 `0%` 或零时间占位。
- 未识别时长的窗口不猜测、不覆盖已识别窗口。后续新增窗口类型另行扩展明确契约。
- 百分比与重置时间继续使用后端返回值，不改变现有计算和时区展示规则。

该方案同时覆盖正常返回 `5h + weekly`、临时只返回 `weekly`，以及未来恢复 5h 窗口的情况。

### 模型与思考强度

- 模型列表继续使用 `model/list` 返回的原始 model、display name 和顺序，不增加本地枚举或排序。
- 思考强度选项保留后端返回的原始 value 和顺序。
- 前端直接显示 `reasoning_effort` 原值，不再做大小写、标题化或 `xhigh -> Max` 等语义转换。
- 仅保留边界处的空白清理，避免无效空选项；不合并不同原始值。
- 保存设置时继续把所选原始值写回 `model_reasoning_effort`。

## 数据流

1. quota HTTP 响应解析出 primary/secondary 的百分比、重置时间与窗口秒数。
2. 应用层逐个按时长归入可空的 `hourly` 或 `weekly` 字段。
3. Web 接口序列化时省略缺失字段。
4. Runtime 页面只渲染实际存在的窗口，并使用固定的 `5 Hours / Weekly` 产品文案。
5. `model/list` 的模型与思考强度经过现有后端 DTO 透传；前端以原值构造 `<option>` 文案和值。

## 错误与兼容边界

- quota 请求、鉴权刷新与百分比边界处理保持不变。
- 响应没有 `limit_window_seconds` 或返回未知时长时，不使用位置兜底，避免再次错位。
- API 保留 `hourly / weekly` 名称，现有调用方无需迁移；字段改为可选后，调用方必须容忍任一窗口缺失。
- 本次不切换 quota 数据源，不扩展额度重置积分、credits 或其他 app-server 字段。
- app-server 初始化通知兼容问题不纳入本次变更，避免扩大当前修复范围。

## 最小测试范围

- 应用层一组表驱动测试验证：窗口顺序不影响 5h/周归类，只有周窗口时 `hourly` 为空。
- 现有 model/list 应用层测试加入一个原始思考强度值，验证 value、description 与顺序透传。
- 前端保留一条用户可见回归测试：仅渲染存在的 quota 窗口，并按原值显示 `xhigh`。

不为 primary/secondary 的所有排列、所有未知时长、大小写格式化函数或重复 option 单独增加测试。

## 文档同步

同步更新 README、稳定需求总览、Control 领域需求和技术方案，将 `hourly` 产品文案修正为 5 小时窗口，并明确额度按时长归类、缺失不展示，模型与思考强度按 Codex 能力列表原样呈现。
