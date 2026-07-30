<!-- prettier-ignore -->
# __APP_PACKAGE__

`__PROJECT_NAME__` 的管理端宿主。该包私有且不实现业务，默认先组合 `@liujitcn/kratos-admin-system`，再组合自有 module **MODULE_PACKAGES** 和所选的其他业务 module。

## 目录与文件

```text
apps/admin
├── src
│   ├── main.ts
│   ├── module-manifest.ts
│   ├── modules.ts
│   └── vite-env.d.ts
├── .env
├── .env.development
├── .env.production
├── index.html
├── package.json
├── README.md
├── tsconfig.json
└── vite.config.ts
```

| 路径                     | 作用                                              |
| ------------------------ | ------------------------------------------------- |
| `src/main.ts`            | 将默认导出的全部 module 交给 core 启动 Vue 应用。 |
| `src/module-manifest.ts` | 统一声明 module 加载器、包名和预构建依赖。        |
| `src/modules.ts`         | 加载并默认导出当前宿主启用的全部 module。         |
| `src/vite-env.d.ts`      | 引入 Vite 客户端和 core 全局类型。                |
| `.env`                   | 所有模式共享的应用标题、端口等环境变量。          |
| `.env.development`       | 开发模式 API 地址和代理配置。                     |
| `.env.production`        | 生产模式 API 地址和构建配置。                     |
| `index.html`             | Vite HTML 入口和应用挂载节点。                    |
| `package.json`           | 声明宿主命令、core、当前模块和额外模块依赖。      |
| `README.md`              | 当前宿主的目录和文件说明。                        |
| `tsconfig.json`          | 宿主 TypeScript 检查配置。                        |
| `vite.config.ts`         | 组合 core 配置与模块 manifest 派生的构建参数。    |

## 模块组合

`src/module-manifest.ts` 是宿主 module 配置的唯一来源，每个 manifest 项同时声明 npm 包名、运行时加载器和可选预构建依赖。`src/modules.ts` 加载并默认导出全部 module，`vite.config.ts` 从 manifest 派生构建扫描参数。新增或删除 module 时，只需修改宿主依赖和这一份 manifest。

模块注册顺序只影响显式 `staticViews` 替换：后注册模块覆盖先注册模块的同一静态视图键。普通业务页面按 `<module>/<view>` 隔离，不允许通过同名页面覆盖其他模块。

模块之间使用 Vue Router 跳转；跨模块复用代码时，只引用模块公开的 npm 子路径，宿主不得引用模块的 `src` 目录。

## 命令

```bash
pnpm --filter __APP_PACKAGE__ dev
pnpm --filter __APP_PACKAGE__ type:check
pnpm --filter __APP_PACKAGE__ build
```
