# @liujitcn/kratos-taro-app-ui

`@liujitcn/kratos-taro-app-ui` 是 Kratos Taro 应用的 UI 基础包，统一维护 NutUI 适配、按需图标和应用主题变量。包内不承载业务页面，也不依赖业务模块。

## 公开入口

- `@liujitcn/kratos-taro-app-ui`：版本标识、按需封装的 NutUI 图标和与 uni-app 对齐的 `UniIcon`；`UniIcon` 同时提供消息分类所需的通知、锁定、列表、会话等字形。
- `@liujitcn/kratos-taro-app-ui/icons`：图标专用入口。
- `@liujitcn/kratos-taro-app-ui/styles/theme.scss`：Kratos 与 NutUI 主题变量。

宿主应在全局样式中引入主题：

```scss
@use '@liujitcn/kratos-taro-app-ui/styles/theme.scss';
```

业务代码只从本包公开入口引用图标。不要直接依赖 NutUI 的内部路径；按需加载细节由本包统一维护。

## 统一圆角

主题提供 `--kratos-page-radius: 12px`（Taro 设计稿单位），按 750 设计宽度转换，与 uni-app 的 `12rpx` 对齐，在 375 逻辑宽度下约为 6px。普通卡片、按钮、输入框、消息气泡和弹窗统一使用 `var(--kratos-page-radius)`；底部面板使用 `var(--kratos-page-radius) var(--kratos-page-radius) 0 0`。NutUI 按钮圆角也映射到此变量。

圆形元素使用 `50%`，直角使用 `0`；CSS 绘制的图标和 AI 头像保留造型所需圆角。宿主可以在全局主题引入后，通过 `page`、`.theme-light` 和 `.theme-dark` 覆盖统一圆角，业务页面继续消费变量。
