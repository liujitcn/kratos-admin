# @liujitcn/kratos-uni-app-system

`@liujitcn/kratos-uni-app-system` 是内置 system 业务模块。个人中心、个人资料、默认设置页装配和 AI 能力都归属本模块，并沿用 core 的登录态、请求、导航和页面装配协议。

## 目录结构

```text
packages/modules/system
├── src
│   ├── api/base/v1            # AI 消息、会话和工具接口
│   ├── rpc                    # system 相关 Buf 生成类型
│   ├── views
│   │   ├── pages/my           # 个人中心主页
│   │   └── pagesMember        # AI、个人资料和默认设置包装页
│   ├── index.ts               # systemModule 定义和公开入口
│   └── pages.ts               # 页面样式与分包声明
├── tsconfig.json
└── package.json
```

`src/rpc` 是生成产物，只能通过 workspace 根目录的 `pnpm generate:rpc` 更新；该命令委托 Frontend 的 `make ts-uni-app` 和 `backend/api/buf.uni-app.core.typescript.gen.yaml` 生成。MFA 类型和请求统一从 `@liujitcn/kratos-uni-app-core/rpc/base/v1/mfa` 与 `@liujitcn/kratos-uni-app-core/api/base/v1/mfa` 复用。

## 功能

- 注册 `PROFILE_HOME`、`PROFILE`、`SETTINGS` 和 `AI` 四个稳定 `viewKey`。
- 提供个人中心、资料维护（昵称、邮箱和证件信息）、默认应用设置包装和 AI 助手页面。
- 默认设置页直接装配 core 的 `KratosSettingsPage`；产品模块可用同一公共设置页的顶部 `extensions` 插槽追加业务设置。
- 通过 core 的公开 service、store 和工具复用底座能力。
- 允许后续模块按相同物理路由或 `viewKey` 覆盖 system 静态视图。
- npm 根入口使用 `dist/index.mjs`；workspace 的 Turbo 任务和 `prepare:modules` 会先构建 system 及其依赖的 core Node 入口，干净安装后不依赖历史 dist。

模块装配与动态导航约定见 [workspace 文档](../../../README.md)。
