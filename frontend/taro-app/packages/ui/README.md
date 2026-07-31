# @liujitcn/kratos-taro-app-ui

`@liujitcn/kratos-taro-app-ui` 是 Kratos Taro 应用的 UI 基础包，统一维护 NutUI 适配、按需图标和应用主题变量。包内不承载业务页面，也不依赖业务模块。

## 公开入口

- `@liujitcn/kratos-taro-app-ui`：版本标识及按需封装的图标。
- `@liujitcn/kratos-taro-app-ui/icons`：图标专用入口。
- `@liujitcn/kratos-taro-app-ui/styles/theme.scss`：Kratos 与 NutUI 主题变量。

宿主应在全局样式中引入主题：

```scss
@use '@liujitcn/kratos-taro-app-ui/styles/theme.scss';
```

业务代码只从本包公开入口引用图标。不要直接依赖 NutUI 的内部路径；按需加载细节由本包统一维护。
