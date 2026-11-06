# @liujitcn/kratos-admin-bi

kratos-admin 管理端 BI 分析模块，提供数据源建模、智能问数、查询记录、数据卡片、仪表盘设计与个人工作台。

## 安装与接入

在 Vue 3 + Vite 管理端项目中安装，与 core、System 保持相同版本：

```bash
pnpm add @liujitcn/kratos-admin-core @liujitcn/kratos-admin-system @liujitcn/kratos-admin-bi
```

将模块加入宿主的模块加载清单，在 System 模块之后加载：

```ts
{
  packageName: "@liujitcn/kratos-admin-bi",
  load: async () => (await import("@liujitcn/kratos-admin-bi")).biAdminModule
}
```

模块依赖 `@liujitcn/kratos-admin-core` 和 `@liujitcn/kratos-admin-system`。宿主需使用 core 的 Vite 配置扫描模块页面，并连接提供 BI API、菜单和权限的 kratos-admin 后端。

## 目录与公开入口

- `src/api/bi/admin/v1`：BI 请求封装，对应 `./api/*`。
- `src/rpc`：由 Proto 生成的协议类型，对应 `./rpc/*`。
- `src/views`：数据源、问数、卡片、仪表盘和工作台页面，对应 `./views/*`。
- `src/components`：模块内部的图表、建模和仪表盘组件。
- `src/locales`：zh-CN、zh-TW、en-US、ja-JP 语言包，对应 `./locales/*`。
- `src/module.ts`：`biAdminModule` 定义，通过主入口或 `./module` 导出。
- `tests`：查询、图表、布局与响应处理测试。

## 验证与打包

在 `frontend/admin` 目录执行：

```bash
pnpm --filter @liujitcn/kratos-admin-bi test
pnpm --filter @liujitcn/kratos-admin-bi type:check
pnpm --filter @liujitcn/kratos-admin-bi pack
```

打包自动执行现有 `build:package` 命令，生成 `dist/package` 中的源码和类型声明；发布时由 pnpm 将 workspace 依赖替换为对应版本。
