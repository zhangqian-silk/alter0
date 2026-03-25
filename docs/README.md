# docs

`alter0` 的工程文档入口，用于长期维护以下内容：

1. 详细技术方案
2. 架构设计与边界
3. 已支持需求与规划需求

> Baseline: 2026-03-02

## 文档索引

1. [架构设计](./architecture.md)
2. [技术方案](./technical-solution.md)
3. [需求清单](./requirements.md)
4. [Memory 模块统一说明与需求文档](./memory/persistent-memory-module-spec.md)
5. [研究报告](./research/README.md)
6. [Nginx 反向代理与登录保护](./deployment/nginx.md)
7. [Playwright 浏览器自动化测试](./testing/playwright.md)

## 维护约定

1. 功能改动涉及领域边界或主链路时，必须同步更新 `architecture.md`。
2. 新增中大型能力（如存储、鉴权、多租户）时，先更新 `technical-solution.md` 再落代码。
3. 每次里程碑发版前，更新 `requirements.md` 的状态列（`supported` / `planned` / `in-progress`）。
4. 所有临时产物统一落在 `output/`，包括测试结果、截图、排查日志、临时导出与本地自动化输出。
5. 禁止在仓库根目录或业务目录新增一次性临时文件；需要共享的正式产物按对应文档或源码目录维护。
