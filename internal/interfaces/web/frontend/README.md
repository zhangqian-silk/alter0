# alter0 Web Frontend

`internal/interfaces/web/frontend` 是新的 Web 前端工程入口，负责把浏览器端构建、测试与静态产物发布收敛到统一工程目录。

当前阶段采用兼容桥切换：

- `index.html` 仅保留前端启动容器、字体与 legacy 样式入口
- React 负责渲染当前 Web Shell 的 legacy DOM 契约，并启动兼容桥接层
- 旧版运行时脚本与样式通过 `public/legacy` 纳入 Vite 构建产物
- `npm run build` 输出到 `internal/interfaces/web/static/dist`

常用命令：

- `npm install`
- `npm run dev`
- `npm run build`
- `npm run test`
