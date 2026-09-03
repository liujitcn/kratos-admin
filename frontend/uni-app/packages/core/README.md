# @liujitcn/kratos-uni-app-core

`@liujitcn/kratos-uni-app-core` 是应用底座包，提供模块协议、启动流程、动态导航、认证与请求、MFA 校验、共享状态、基础页面和构建期页面装配能力。业务模块只能通过本包公开 exports 使用这些能力。

## 目录结构

```text
packages/core
├── src
│   ├── api                    # 与 Proto 完整路径一致的接口 service
│   ├── components             # 启动状态、自绘 tabBar、公共设置页和 MFA 组件
│   ├── rpc                    # Buf 生成的 TypeScript 接口类型
│   ├── static                 # 底座图片和 tab 图标
│   ├── stores                 # Pinia 设置与用户状态
│   ├── styles                 # 全局基础样式
│   ├── types                  # uni-app 与第三方类型补充
│   ├── utils                  # HTTP、认证、文件、MFA/WebAuthn 和导航工具
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

`src/rpc` 是生成产物，只能通过 workspace 根目录的 `pnpm generate:rpc` 更新；该命令委托 Frontend 的 `make ts-uni-app` 和 `backend/api/buf.uni-app.typescript.gen.yaml` 生成。

## 功能

- 定义 `KratosAppModule`、`viewKey`、静态页面覆盖和模块图标协议。
- 从 `/v1/app/base/menu` 加载 `99000000` 下的扁平菜单，按 `parent_id` 构建树，以二级页面生成 tabBar，并原子更新动态路由和层级导航。
- 按匿名态、登录态缓存导航，并在远端不可用时回退到最近缓存或默认菜单。
- 统一认证、token、MFA、HTTP、上传、配置、Pinia 和基础页面能力。
- 提供可复用的 `KratosSettingsPage`，统一语言、MFA、微信设置和退出登录，并允许产品在最上方追加业务设置。
- 公开通用密码弹窗、MFA 绑定面板与恢复码弹窗，供登录页和业务模块复用同一套绑定界面。
- 登录阶段支持 TOTP、WebAuthn 和一次性恢复码校验；WebAuthn 编解码封装在 `src/utils/webauthn.ts`。
- 构建时扫描模块 `src/views/**/*.vue`，逐文件合并模块 static，生成宿主页并在退出时按事务精确恢复。

## 公开入口

- `@liujitcn/kratos-uni-app-core`：运行时、导航、store 和组件。
- `@liujitcn/kratos-uni-app-core/module`：模块协议和 core 模块。
- `@liujitcn/kratos-uni-app-core/vite`：Vite 配置与页面装配插件。
- `@liujitcn/kratos-uni-app-core/api/*`、`components/*.vue`、`utils/*`、`views/*`：白名单子路径。

## 公共设置页

产品设置页通过公开子路径引用 `KratosSettingsPage`。`extensions` 是唯一业务扩展插槽，固定直接渲染在页面视口最上方；没有传入内容时不会生成容器或额外间距。

```vue
<script setup lang="ts">
import KratosSettingsPage from '@liujitcn/kratos-uni-app-core/components/KratosSettingsPage.vue'
</script>

<template>
  <KratosSettingsPage>
    <template #extensions>
      <ShopSettingsSection />
    </template>
  </KratosSettingsPage>
</template>
```

扩展内容由产品模块自行组织完整卡片。页面顺序固定为业务扩展、MFA 与语言、微信设置、退出登录；产品模块可以注册自己的物理页面，并通过后注册的 `SETTINGS` viewKey 覆盖 system 默认包装页。

构建与验证命令见 [workspace 文档](../../README.md)。
