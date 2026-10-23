# @liujitcn/kratos-uni-app-core

账号密码、OAuth 票据兑换和微信登录统一返回 `mfa_remember_days`，表示当前登录策略允许的 MFA 设备免验证天数；为 `0` 时不提供记住设备选项。

`@liujitcn/kratos-uni-app-core` 是应用底座包，提供模块协议、启动流程、动态导航、认证与请求、MFA 校验、共享状态、基础页面和构建期页面装配能力。业务模块只能通过本包公开 exports 使用这些能力。

## 目录结构

```text
packages/core
├── src
│   ├── api                    # 与 Proto 完整路径一致的接口 service
│   ├── components             # 通用导航栏、自绘 tabBar、公共设置页和 MFA 组件
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
- tabBar 统一自绘，宿主不声明原生 `tabBar`；H5 替换当前路由，微信端复用普通页面栈，栈满时替换栈顶，切换不重建整个页面栈。
- 页面装配插件在编译配置中移除宿主的原生 `tabBar`，退出后恢复宿主原文件。导航按模块注册的 `viewKey` 解析物理页面，支持业务模块覆盖首页和扩展 tab；引用项目升级 core 包后即可使用相同策略。
- 按匿名态、登录态缓存导航，并在远端不可用时回退到最近缓存或默认菜单。
- 统一认证、token、MFA、HTTP、上传、配置、Pinia 和基础页面能力。
- 提供可复用的 `KratosSettingsPage`，统一语言、MFA、微信设置和退出登录，并允许产品在最上方追加业务设置。
- 公开通用密码弹窗、MFA 绑定面板与恢复码弹窗，供登录页和业务模块复用同一套绑定界面。
- 登录阶段支持 TOTP、WebAuthn 和一次性恢复码校验；WebAuthn 编解码封装在 `src/utils/webauthn.ts`。
- 构建时扫描模块 `src/views/**/*.vue`，逐文件合并模块 static，生成宿主页并在退出时按事务精确恢复。

主题 store 公开 `themeStore.colors` 和 `registerThemeColors`。页面脚本使用语义颜色（例如 `colors.primary`、`colors.textMuted`），业务模块可以在初始化时传入 `light`、`dark` 的局部字段覆盖 Core 默认值；未传入的字段保持默认值，覆盖会立即作用于当前主题。

```ts
import { registerThemeColors } from '@liujitcn/kratos-uni-app-core/stores'

registerThemeColors({
  light: { primary: '#e1251b' },
  dark: { primary: '#ff4d4f' },
})
```

## 公开入口

登录页返回会清理本次回跳并替换到首页；直接打开登录页清理旧 `lastRoute`，登录成功默认进入首页。主动登录、路由守卫和会话过期重登通过 `redirect=1` 标记保留本次业务目标及查询参数，成功跳转后才清理目标。登录页每次显示都会检查有效登录态，已登录则替换回首页，避免浏览器回退重新停留在登录页。H5 与微信小程序统一使用 `reLaunch` 替换登录入口并重置应用页面栈。

- `@liujitcn/kratos-uni-app-core`：运行时、导航、store 和组件。
- `@liujitcn/kratos-uni-app-core/module`：模块协议和 core 模块。
- `@liujitcn/kratos-uni-app-core/vite`：Vite 配置、页面装配插件和构建错误文案格式化器。
- `@liujitcn/kratos-uni-app-core/api/*`、`components/*.vue`、`utils/*`、`views/*`：白名单子路径。

## 通用顶部导航

所有自定义顶部导航统一使用公开组件 `KratosNavBar`，页面只提供标题和业务操作，不再定义导航高度、安全区、标题居中、按钮尺寸或微信胶囊避让。普通页与 tabBar 页的标题栏使用相同高度，按钮显隐不会改变标题位置。页面外层只维护水平与底部留白，顶部留白由组件处理；tabBar 页的导航栏还统一预留内容间距。

```vue
<script setup lang="ts">
import KratosNavBar from '@liujitcn/kratos-uni-app-core/components/KratosNavBar.vue'
</script>

<template>
  <!-- 普通页：默认带返回，无法返回时回到首页 -->
  <KratosNavBar title="个人信息" />
  <!-- tabBar 页：默认不带返回，也可以不传操作按钮 -->
  <KratosNavBar title="首页" variant="tab" />
  <KratosNavBar
    title="我的"
    variant="tab"
    :action="{ icon: 'settings', label: '设置', onClick: openSettings }"
  />
</template>
```

`back` 可显式控制返回按钮，`onBack` 可覆盖默认返回操作。`action` 包含 `label`、可选 `icon`、可选 `display` 和 `onClick`；`display` 支持 `icon`（纯图标）、`text`（纯文字）和 `both`（图标加文字），默认有图标时显示图标，否则显示文字。图标支持内置 `settings`、`read-all`、`history`，以及自定义图片地址。不传 `action` 时不渲染操作按钮。三种按钮形式共用相同高度；纯图标保持方形，文字与组合按钮按内容增宽，空间不足时省略文字。组件通过公开的 `components/KratosNavBar.vue` 子路径导入；`KratosNavBarProps`、`KratosNavBarAction` 类型从包根公开导出。小程序的 Vue 组件必须直接导入 `.vue` 文件，避免经 TypeScript 入口重导出导致编译产物漏登记 `usingComponents`；业务模块应传入翻译后的标题和按钮名称。

```ts
import type { KratosNavBarAction } from '@liujitcn/kratos-uni-app-core'

// 纯图标
const iconAction: KratosNavBarAction = { icon: 'settings', label: '设置', display: 'icon', onClick: openSettings }
// 纯文字
const textAction: KratosNavBarAction = { label: '全部已读', display: 'text', onClick: markAllRead }
// 图标加文字
const combinedAction: KratosNavBarAction = { icon: 'history', label: '历史', display: 'both', onClick: openHistory }
```

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

应用配置加载成功后，core 会将 `i18n_customs` 按 `locale + key` 覆盖已注册的本地文案，并刷新页面翻译。仅覆盖已有 key 的非空文案；未配置或已删除的覆盖项恢复本地默认值，未知语言和未知 key 忽略。切换语言时沿用配置重载流程。
