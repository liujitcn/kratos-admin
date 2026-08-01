# @liujitcn/kratos-taro-app-core

`@liujitcn/kratos-taro-app-core` 是 Taro 应用底座，负责模块协议、启动流程、认证请求、共享状态、动态导航、基础页面、公共静态资源和构建期页面装配。core 不依赖 UI 或业务模块。

## 主要能力

- 账号密码、行为验证码、微信登录、OAuth、Token 刷新和退出。
- 统一 HTTP、上传、下载、文件预览与 RSA/AES 密码加密。
- Zustand 用户、配置和导航状态。
- 接口驱动的逻辑路由、访问模式、自绘 tabBar 和稳定 `viewKey`。
- 页面静态资源 export、动态资源公共路径解析，以及 H5 默认导航栏兼容框架。
- 首页、登录、协议、WebView、启动/错误状态页面。
- H5 与微信小程序共用的模块页面装配 runner。

## 公开入口

- `@liujitcn/kratos-taro-app-core`：启动、模块、导航、状态和常用工具，包含动态路径使用的 `resolveBundledAsset()`。
- `@liujitcn/kratos-taro-app-core/static/*`：页面图片等可由 Webpack 静态分析的打包资源。
- `@liujitcn/kratos-taro-app-core/build`：构建期模块描述。
- `@liujitcn/kratos-taro-app-core/runner`：Taro 构建装配入口。
- `api/*`、`components/*`、`rpc/*`、`styles/*`、`utils/*`、`views/*`：白名单子路径。

业务模块只能通过这些 exports 使用底座，不得相对引用 core 源码。RPC 由 `make -C backend ts-taro-app` 生成，不能手工编辑。

## Runner 协议

模块必须同时提供运行时定义和 `./build` 导出。构建入口返回模块名称、包根目录与页面清单；runner 根据宿主的静态 `module-manifest.ts` 解析模块顺序，扫描页面并创建事务。

runner 支持命名导入和默认导入的模块，后注册模块可以覆盖相同物理路由。页面私有 `components` 不会成为路由，模块 static 会先合并到宿主再由 Taro 编译。H5 下，runner 根据页面 `navigationStyle` 和 `navigationBar*` 配置生成默认顶部导航栏；微信小程序继续使用原生导航栏。构建失败、正常退出和收到终止信号时都会执行恢复；传入 `--prepare-only` 时只装配页面供类型检查使用，配合 `--cleanup-only` 在检查结束后恢复，不启动 Taro 编译。

## 验证

```bash
pnpm --filter @liujitcn/kratos-taro-app-core tsc
pnpm test
pnpm check:exports
```
