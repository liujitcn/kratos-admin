# @liujitcn/kratos-taro-app

`apps/taro-app` 是私有 Taro React 宿主，组合 core、UI 和业务模块，提供 H5 与微信小程序入口，不承载可复用页面或业务请求。

## 目录

```text
apps/taro-app
├── config                       # Taro Webpack 5 配置
├── src
│   ├── pages/bootstrap          # 唯一固定页面
│   ├── app.config.base.json     # runner 使用的只读路由基线
│   ├── app.config.ts            # 构建期间临时改写并恢复
│   ├── app.scss                 # 全局主题和基础样式
│   ├── app.tsx                  # 模块注册与启动
│   └── module-manifest.ts       # 唯一模块清单
├── babel.config.cjs
├── package.json
├── project.config.json
└── tsconfig.json
```

模块清单顺序决定页面、`viewKey` 和图标的覆盖优先级。新增业务模块时把依赖加入宿主并在 `module-manifest.ts` 静态导入；不要把模块页面复制到宿主，也不要提交 runner 生成的 wrapper、config 或 static 文件。

H5 构建默认输出到 `backend/data/app` 并使用 `/app/` 公共路径。微信小程序开发和生产产物统一输出到 `apps/taro-app/dist`，微信开发者工具导入该目录即可。
