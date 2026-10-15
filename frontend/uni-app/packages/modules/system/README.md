# @liujitcn/kratos-uni-app-system

`@liujitcn/kratos-uni-app-system` 是内置 system 业务模块。个人中心、个人资料、默认设置页装配和 AI 能力都归属本模块，并沿用 core 的登录态、请求、导航和页面装配协议。

## 目录结构

```text
packages/modules/system
├── src
│   ├── api/base/v1            # AI 消息、会话、快捷入口和Provider模型选项接口
│   ├── components             # 可复用的 KratosMyPage 公共个人中心
│   ├── rpc                    # system 相关 Buf 生成类型
│   ├── views
│   │   ├── pages/my           # 个人中心主页
│   │   │   └── components     # MyExtensions 业务扩展的空实现
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

## 个人中心扩展

公共组件 `KratosMyPage` 保留顶部标题与设置、个人信息和 AI 助手入口；AI 卡片继续由应用的 `aiEnabled` 配置控制显隐。唯一业务插槽 `extensions` 位于这些固定内容之后，与它们共用滚动区域、主题和卡片间距。未传入内容时不增加容器或留白，业务卡片、数据请求和权限由项目模块维护。

内置 [我的页面](src/views/pages/my/my.vue) 已使用该组件，并在插槽中挂载 [MyExtensions 空实现](src/views/pages/my/components/MyExtensions.vue)。空实现保留一个 `view` 作为业务内容容器，不显示占位卡片；项目可以按这个结构组织自己的业务内容。

当前模块的 `pages/my/my.vue` 通过公开组件入口装配个人中心：

```vue
<script setup lang="ts">
import KratosMyPage from '@liujitcn/kratos-uni-app-system/components/KratosMyPage.vue'
</script>

<template>
  <KratosMyPage />
</template>
```

项目模块的 `pages` 声明 `'pages/my/my': { style: { navigationStyle: 'custom', navigationBarTitleText: '' } }`，`views` 声明 `PROFILE_HOME: 'pages/my/my'`，并在宿主 `module-manifest.ts` 中排在 system 模块之后。使用相同的物理路由可以保留微信原生 tab 切换；页面 wrapper 和底部 tabBar 继续由构建插件生成，不在业务页面重复挂载。

当前页面保留空的 `extensions` 插槽，宿主可以在本仓库的模块清单中装配业务内容；公共页面、顶部入口和导航逻辑仍由 System 维护。需要登录态、用户资料或导航时，通过 core 的公开 store 与导航 API 获取。

模块装配与动态导航约定见 [workspace 文档](../../../README.md)。
