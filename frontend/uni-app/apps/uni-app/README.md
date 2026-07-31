# @liujitcn/kratos-uni-app

`@liujitcn/kratos-uni-app` 是不发布的私有 uni-app 宿主，负责把 core 和业务模块装配成可运行的 H5、微信小程序应用。宿主只保留平台入口和固定 bootstrap 页面，不承载可复用业务实现。

## 目录结构

```text
apps/uni-app
├── src
│   ├── pages/bootstrap        # 唯一固定页面，启动后进入动态导航
│   ├── App.vue                # 应用根组件和全局样式入口
│   ├── main.ts                # Vue、Pinia 与模块启动入口
│   ├── manifest.json          # uni-app 多端配置
│   ├── module-manifest.ts     # 宿主唯一模块清单
│   ├── pages.json             # 仅提交 bootstrap 固定路由
│   └── uni.scss               # 转发 core 的全局 Sass 变量
├── index.html                 # H5 HTML 入口
├── vite.config.ts             # 页面装配插件和 uni-app 插件
├── tsconfig.json              # 宿主类型检查配置
└── package.json
```

构建期间，core 的 Vite 插件会扫描模块页面，临时生成页面 wrapper、合并 `pages.json` 并复制静态资源；构建或开发服务退出后会恢复宿主。不要把生成页面当作宿主源码提交。

## 功能

- 通过 `module-manifest.ts` 注册 core、system 及后续业务模块。
- 使用 `bootstrapKratosApp()` 初始化 Vue、Pinia、身份状态和动态导航。
- 为 H5 提供 hash 路由，为微信小程序提供对应页面产物。
- 将 H5 构建结果写入后端静态目录，将微信产物写入本包 `dist`。

## 命令

命令统一从 `frontend/uni-app` workspace 根目录执行：

```bash
pnpm dev:h5
pnpm dev:mp-weixin
pnpm build:h5
pnpm build:mp-weixin
```

模块开发和动态导航约定见 [workspace 文档](../../README.md)。
