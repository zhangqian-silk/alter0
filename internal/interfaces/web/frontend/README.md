# alter0 Web Frontend

`internal/interfaces/web/frontend` 是新的 Web 前端工程入口，负责把浏览器端构建、测试与静态产物发布收敛到统一工程目录。

当前阶段采用兼容桥切换：

- `index.html` 继续承载现有 Web Shell DOM 结构
- React 负责启动前端桥接层并接管后续迁移入口
- 旧版运行时脚本与样式通过 `public/legacy` 纳入 Vite 构建产物
- `npm run build` 输出到 `internal/interfaces/web/static/dist`

常用命令：

- `npm install`
- `npm run dev`
- `npm run build`
- `npm run test`
