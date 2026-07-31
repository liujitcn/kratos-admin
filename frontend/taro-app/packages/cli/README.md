# @liujitcn/kratos-taro-app-cli

`@liujitcn/kratos-taro-app-cli` 用于创建模块化的 Taro React workspace。生成项目默认装配 Kratos core、UI 和 system 包，支持 H5 与微信小程序。

## 使用

```bash
pnpm dlx @liujitcn/kratos-taro-app-cli create customer-app
pnpm dlx @liujitcn/kratos-taro-app-cli create shop-app --module shop,order
pnpm dlx @liujitcn/kratos-taro-app-cli create customer-app --with @acme/customer-module
```

- `--module`：创建 workspace 内的本地业务模块，可重复使用或用逗号分隔。
- `--with`：装配一个已发布的 Taro 业务模块，可重复使用。

本地模块包含运行时入口、页面清单和构建期入口。模块页面由 core runner 在构建期间装配到私有宿主，不需要提交生成的页面包装器。

JavaScript 调用方也可以直接使用 `scaffoldKratosTaroApp(target, options)`。
