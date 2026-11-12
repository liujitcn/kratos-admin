# @liujitcn/kratos-uni-app

`@liujitcn/kratos-uni-app` 是不发布的私有 uni-app x 宿主，负责把 core 和业务模块装配成可运行的 H5、微信小程序和 App 应用。宿主只保留平台入口和页面生成入口，不承载可复用业务实现。

## 目录结构

```text
apps/uni-app
├── App.uvue                    # 应用根组件和全局样式入口
├── main.ts                     # Vue、Pinia 与模块启动入口
├── manifest.json               # uni-app x 配置
├── module-manifest.ts          # 宿主唯一模块清单
├── pages                       # x 页面 wrapper
├── pagesMember                 # x 分包页面 wrapper
├── scripts/generate-pages.mjs  # 页面和静态资源生成器
├── pages.json                  # x 页面路由
├── uni.scss                    # 转发 core 的全局 Sass 变量
├── favicon.ico                # H5 宿主浏览器图标
├── index.html                 # H5 HTML 入口
├── vite.config.ts             # uni-app x 页面生成器和 uni-app 插件
├── tsconfig.json              # 宿主类型检查配置
└── package.json
```

运行和构建前执行 `pnpm run prepare:x`，生成 x 页面 wrapper、`pages.json` 和静态资源。页面位于 x 宿主根目录，不再使用旧 `src` 输入目录和临时页面事务。

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
pnpm --filter @liujitcn/kratos-uni-app dev:app
pnpm --filter @liujitcn/kratos-uni-app build:app
```

通过局域网 IP 访问 H5 时，可复用仓库根 `certs` 下的共享证书，并在 `.env.development-h5.local` 中设置 `VITE_APP_HTTPS=true`；后端使用 HTTPS 时同步设置 `VITE_APP_API_URL=https://127.0.0.1:7001`。

模块开发和动态导航约定见 [workspace 文档](../../README.md)。
