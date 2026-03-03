# Daily Research Agent Task (alter0)

请执行每日研究报告任务，直接由你（Agent）完成，而不是调用 `scripts/research/daily_research_report.py` 脚本生成。

## 目标

在 `alter0/docs/research` 下更新两份独立文档（当天日期）：

1. `docs/research/hotspots/YYYY-MM-DD.md`
2. `docs/research/comparison/YYYY-MM-DD.md`

并更新索引：

- `docs/research/README.md`

## 数据来源要求

- 论坛：如 Hacker News、Reddit、技术社区
- 博客：如 OpenAI、Google AI、Hugging Face 等官方博客
- 技术网站/媒体：如 The Verge、MIT Technology Review 等
- GitHub 仓库可作为补充，不得作为唯一来源

## 内容要求

- 全文中文总结
- 热点追踪与功能对比必须是两份不同文档
- 对比文档需包含：OpenClaw 当前能力、社区需求信号、与 alter0 需求覆盖对照、优先建议

## 约束

- 只改研究报告相关文件，不改业务代码
- 创建 PR 并自动合并（若策略允许）
- 自动合并类 PR 默认不分配 assignee/reviewer
