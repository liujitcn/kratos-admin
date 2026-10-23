# @liujitcn/kratos-taro-app-core

账号密码、OAuth 票据兑换和微信登录统一返回 `mfa_remember_days`，表示当前登录策略允许的 MFA 设备免验证天数；为 `0` 时不提供记住设备选项。

`@liujitcn/kratos-taro-app-core` 是 Taro 应用底座，负责模块协议、启动流程、认证请求、共享状态、动态导航、基础页面、公共静态资源和构建期页面装配。core 不依赖 UI 或业务模块。

## 主要能力

- 账号密码、行为验证码、微信登录、OAuth、MFA（TOTP/WebAuthn/恢复码）、Token 刷新和退出。
- 统一 HTTP、上传、下载、文件预览、RSA/AES 密码加密和 WebAuthn 工具。
- Zustand 用户、配置和导航状态，H5 浏览器页签标题使用宿主 HTML 声明的应用名称。
- 登录阶段支持 TOTP、WebAuthn 和一次性恢复码校验。
- 公开 Base MFA 请求及通用密码弹窗、绑定面板和恢复码弹窗，供登录页与业务模块复用同一套绑定界面。
- 接口驱动的逻辑路由、访问模式、自绘 tabBar 和稳定 `viewKey`。
- tab 切换统一使用普通页面路由：H5 替换当前路由，微信端复用页面栈，栈满时替换栈顶，避免整栈重建。
- runner 在编译配置中移除宿主的原生 `tabBar`，退出后恢复宿主原文件；导航按模块注册的 `viewKey` 解析物理页面，支持业务模块覆盖首页和扩展 tab。
- 页面静态资源 export、动态资源公共路径解析，以及 H5 默认导航栏兼容框架。
- 首页、登录、协议、WebView、启动/错误状态页面。
- H5 与微信小程序共用的模块页面装配 runner。

主题 store 公开 `useThemeStore().colors` 和 `registerThemeColors`。页面脚本使用语义颜色（例如 `colors.primary`、`colors.textMuted`），业务模块可以在初始化时传入 `light`、`dark` 的局部字段覆盖 Core 默认值；未传入的字段保持默认值，覆盖会立即作用于当前主题。

```ts
import { registerThemeColors } from '@liujitcn/kratos-taro-app-core/stores'

registerThemeColors({
  light: { primary: '#e1251b' },
  dark: { primary: '#ff4d4f' },
})
```

## 公开入口

登录页返回会清理本次回跳并替换到首页；直接打开登录页清理旧 `lastRoute`，登录成功默认进入首页。主动登录、路由守卫和会话过期重登通过 `redirect=1` 标记保留本次业务目标及查询参数，成功跳转后才清理目标。登录页每次显示都会检查有效登录态，已登录则替换回首页，避免浏览器回退重新停留在登录页。H5 与微信小程序统一使用 `reLaunch` 替换登录入口并重置应用页面栈。

- `@liujitcn/kratos-taro-app-core`：启动、模块、导航、状态和常用工具，包含动态路径使用的 `resolveBundledAsset()`。
- `@liujitcn/kratos-taro-app-core/static/*`：页面图片等可由 Webpack 静态分析的打包资源。
- `@liujitcn/kratos-taro-app-core/build`：构建期模块描述和双语构建错误文案格式化器。
- `@liujitcn/kratos-taro-app-core/runner`：Taro 构建装配入口。
- `api/*`、`components/*`、`rpc/*`、`styles/*`、`utils/*`、`views/*`：白名单子路径。

业务模块只能通过这些 exports 使用底座，不得相对引用 core 源码。RPC 由 `make -C frontend ts-taro-app` 生成，不能手工编辑。

## 通用顶部导航

所有自定义顶部导航统一使用公开组件 `KratosNavBar`，页面只提供标题和业务操作，不再定义导航高度、安全区、标题居中、按钮尺寸或微信胶囊避让。普通页与 tabBar 页的标题栏使用相同高度，按钮显隐不会改变标题位置。页面外层只维护水平与底部留白，顶部留白由组件处理；tabBar 页的导航栏还统一预留内容间距。

```tsx
import { KratosNavBar } from '@liujitcn/kratos-taro-app-core'

// 普通页默认带返回，无法返回时回到首页。
<KratosNavBar title='个人信息' />
// tabBar 页默认不带返回，也可以不传操作按钮。
<KratosNavBar title='首页' variant='tab' />
<KratosNavBar
  title='我的'
  variant='tab'
  action={{ icon: 'settings', label: '设置', onClick: openSettings }}
/>
```

`back` 可显式控制返回按钮，`onBack` 可覆盖默认返回操作。`action` 包含 `label`、可选 `icon`、可选 `display` 和 `onClick`；`display` 支持 `icon`（纯图标）、`text`（纯文字）和 `both`（图标加文字），默认有图标时显示图标，否则显示文字。图标支持内置 `settings`、`read-all`、`history`，以及自定义图片地址。不传 `action` 时不渲染操作按钮。三种按钮形式共用相同高度；纯图标保持方形，文字与组合按钮按内容增宽，空间不足时省略文字。组件及 `KratosNavBarProps`、`KratosNavBarAction` 类型均从包根公开导出；业务模块应传入翻译后的标题和按钮名称。

```ts
import type { KratosNavBarAction } from '@liujitcn/kratos-taro-app-core'

// 纯图标
const iconAction: KratosNavBarAction = { icon: 'settings', label: '设置', display: 'icon', onClick: openSettings }
// 纯文字
const textAction: KratosNavBarAction = { label: '全部已读', display: 'text', onClick: markAllRead }
// 图标加文字
const combinedAction: KratosNavBarAction = { icon: 'history', label: '历史', display: 'both', onClick: openHistory }
```

## Runner 协议

模块必须同时提供运行时定义和 `./build` 导出。构建入口返回模块名称、包根目录与页面清单；runner 根据宿主的静态 `module-manifest.ts` 解析模块顺序，扫描页面并创建事务。

runner 支持命名导入和默认导入的模块，后注册模块可以覆盖相同物理路由。页面私有 `components` 不会成为路由，模块 static 会先合并到宿主再由 Taro 编译。H5 下，runner 根据页面 `navigationStyle` 和 `navigationBar*` 配置生成默认顶部导航栏；微信小程序继续使用原生导航栏。构建失败、正常退出和收到终止信号时都会执行恢复；传入 `--prepare-only` 时只装配页面供类型检查使用，配合 `--cleanup-only` 在检查结束后恢复，不启动 Taro 编译。

## 验证

```bash
pnpm --filter @liujitcn/kratos-taro-app-core tsc
pnpm test
pnpm check:exports
```

构建产物按模式与平台隔离：开发使用 `dist/dev/h5`、`dist/dev/mp-weixin`，生产默认使用 `dist/build/h5`、`dist/build/mp-weixin`。`KRATOS_TARO_OUTPUT_ROOT` 可覆盖默认目录；当前仓库宿主的 H5 生产构建输出到 `backend/web/taro-app`。微信开发者工具默认指向开发目录，发布时导入生产目录。

应用配置加载成功后，core 会将 `i18n_customs` 按 `locale + key` 覆盖已注册的本地文案，并刷新页面翻译。仅覆盖已有 key 的非空文案；未配置或已删除的覆盖项恢复本地默认值，未知语言和未知 key 忽略。切换语言时沿用配置重载流程。
