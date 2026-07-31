# @liujitcn/kratos-app-core

`@liujitcn/kratos-app-core` 是应用底座包，提供模块协议、启动流程、动态导航、认证与请求、共享状态、基础页面和构建期页面装配能力。业务模块只能通过本包公开 exports 使用这些能力。

## 目录结构

```text
packages/core
├── src
│   ├── api                    # base、system 基础接口 service
│   ├── components             # 启动状态和自绘 tabBar
│   ├── rpc                    # Buf 生成的 TypeScript 接口类型
│   ├── static                 # 底座图片和 tab 图标
│   ├── stores                 # Pinia 设置与用户状态
│   ├── styles                 # 全局基础样式
│   ├── types                  # uni-app 与第三方类型补充
│   ├── utils                  # HTTP、认证、文件和导航工具
│   ├── views                  # 首页、登录、状态页和 WebView
│   ├── bootstrap.ts           # 应用初始化
│   ├── module.ts              # 模块定义与注册表
│   ├── navigation.ts          # base-menu 导航解析与缓存
│   ├── pages.ts               # core 页面声明
│   └── vite.ts                # 自动页面装配插件
├── test                       # 导航匹配和 Vite 装配测试
├── tsconfig.json
└── package.json
```

`src/rpc` 是生成产物，只能通过 workspace 根目录的 `pnpm generate:rpc` 更新；该命令委托 backend 的 `make ts-app` 和 `backend/api/buf.app.typescript.gen.yaml` 生成。

## 功能

- 定义 `KratosAppModule`、`viewKey`、静态页面覆盖和模块图标协议。
- 从 `/v1/app/base/menu` 加载扁平菜单，校验后原子更新动态路由和 tabBar。
- 按匿名态、登录态缓存导航，并在远端不可用时回退到最近缓存或默认菜单。
- 统一认证、token、HTTP、上传、配置、Pinia 和基础页面能力。
- 构建时扫描模块 `src/views/**/*.vue`，生成宿主页并在退出时事务恢复。

## 公开入口

- `@liujitcn/kratos-app-core`：运行时、导航、store 和组件。
- `@liujitcn/kratos-app-core/module`：模块协议和 core 模块。
- `@liujitcn/kratos-app-core/vite`：Vite 配置与页面装配插件。
- `@liujitcn/kratos-app-core/api/*`、`utils/*`、`views/*`：白名单子路径。

构建与验证命令见 [workspace 文档](../../README.md)。
