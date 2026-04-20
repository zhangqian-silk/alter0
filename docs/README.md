# docs

`alter0` 的工程文档入口，用于长期维护以下内容：

- 架构设计与边界
- 领域化技术方案
- 领域化需求清单与细化文档
- 部署、测试、Memory 等专项说明

> Baseline: 2026-03-02

## 文档索引

- [架构设计](./architecture.md)
- [领域化技术方案](./technical-solution.md)
- [领域化需求清单](./requirements.md)
- [前端设计与实现规范](./frontend-design-guidelines.md)
- [前端彻底重构方案](./frontend-rebuild-plan.md)
- [Memory 模块统一说明与需求文档](./memory/persistent-memory-module-spec.md)
- [研究报告](./research/README.md)
- [Nginx 反向代理与登录保护](./deployment/nginx.md)
- [Go 单元测试用例说明](./testing/unit-test-cases.md)
- [Playwright 浏览器自动化测试](./testing/playwright.md)

## 维护约定

- 功能改动涉及领域边界或主链路时，必须同步更新 `architecture.md`。
- 需求变更按领域更新 `requirements.md` 与 `requirements-details/*.md`；涉及架构、接口、数据、执行、存储、部署或测试策略时，同步更新 `technical-solution.md`；后续不再新增线性编号需求文件。
- 每次里程碑发版前，更新 `requirements.md` 的状态列（`supported` / `planned` / `in-progress`）。
- 所有临时产物统一落在 `output/`，包括测试结果、截图、排查日志、临时导出与本地自动化输出。
- 禁止在仓库根目录或业务目录新增一次性临时文件；需要共享的正式产物按对应文档或源码目录维护。
- 功能新增、缺陷修复、行为调整与重构默认遵循 TDD；纯文档、注释、格式化或无法自动化验证的变更需在交付说明中明确免测原因与替代验证。
