<!-- prettier-ignore -->
# __MODULE_PACKAGE__

<!-- prettier-ignore -->
`__PROJECT_NAME__` 的 __MODULE_PASCAL__ 管理端业务模块。API、RPC、页面和业务组件同包维护，可以发布到 npm 后被不同宿主组合使用。

## 目录与文件

```text
packages/modules/__MODULE_NAME__
├── src
│   ├── api                       # 按 Proto 完整路径新增服务目录和同名请求文件
│   ├── rpc
│   │   └── README.md             # Proto 生成目录说明
│   ├── components                # 可选的模块内共享组件
│   ├── views
│   │   └── index
│   │       └── index.vue
│   ├── index.ts
│   └── module.ts
├── package.json
├── README.md
├── tsconfig.json
└── tsconfig.package.json
```

| 路径                        | 作用                                                            |
| --------------------------- | --------------------------------------------------------------- |
| `src/index.ts`              | npm 主入口，导出 `__MODULE_IDENTIFIER__`。                      |
| `src/module.ts`             | 收集 `src/views/**/*.vue` 并声明名为 `__MODULE_NAME__` 的模块。 |
| `src/api/<proto-path>/*.ts`  | 与 Proto 服务文件同路径、同名的请求封装。                       |
| `src/rpc/README.md`         | 当前模块 RPC 生成目录说明，生成类型通过模块包子路径公开。       |
| `src/views/index/index.vue` | 模板自带的模块首页。                                            |
| `package.json`              | 声明依赖、模块入口、API/RPC 子路径和 npm 发布配置。             |
| `README.md`                 | 当前业务模块的目录、文件和接入说明。                            |
| `tsconfig.json`             | 开发态类型检查配置。                                            |
| `tsconfig.package.json`     | npm 发布声明文件生成配置。                                      |

## 开发约束

- API 放在与 Proto 完整路径一致的 `src/api/<proto-path>`，例如 `src/api/base/v1`、`src/api/system/admin/v1`；文件名与服务文件名保持一致。RPC 保留 `src/rpc/<proto-domain>/<version>` 等完整生成层级，不扁平化。
- 页面放在 `src/views`。页面私有组件就近放置，需要模块内多个页面复用时再创建 `src/components`。
- 底座能力通过 `@liujitcn/kratos-admin-core` 的公开子路径引用，不要依赖 core 源码目录。
- 跨模块页面跳转使用 Vue Router；跨模块代码复用需要对方先提供 npm 公开导出。
- 新增页面后，动态菜单组件路径必须使用 `__MODULE_NAME__/` 模块前缀；模板首页对应 `__MODULE_NAME__/index/index`，不兼容 `index/index`。
- 需要替换 core 静态页面时，使用 core 导出的 `ADMIN_STATIC_VIEWS` 在模块 `staticViews` 中显式映射页面。

模块运行时 Interface 是 `src/index.ts` 导出的 `__MODULE_IDENTIFIER__`。API 和 RPC 通过 `package.json#exports` 公开；页面只通过 `AdminModule.views` 注册，不作为 npm 子路径导出。组件需要真实跨模块复用时，按具体文件增加显式导出，不提供 `components/*` 通配入口。模块内部文件不因位于 `src` 下自动成为公共 Interface。

宿主接入：

```ts
export const adminModuleManifest = [
  {
    packageName: "__MODULE_PACKAGE__",
    load: async () => (await import("__MODULE_PACKAGE__")).__MODULE_IDENTIFIER__
  }
];
```

## 命令

```bash
pnpm --filter __MODULE_PACKAGE__ type:check
pnpm --filter __MODULE_PACKAGE__ build:package
pnpm --filter __MODULE_PACKAGE__ pack
```
