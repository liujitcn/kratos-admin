# @liujitcn/kratos-admin-app

默认管理端宿主。该包不发布、不承载业务实现，只负责选择业务模块、加载 core 并提供本地开发和生产构建入口。

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

| 路径                     | 作用                                                |
| ------------------------ | --------------------------------------------------- |
| `src/main.ts`            | 应用入口，将默认导出的全部 module 交给 core 启动。  |
| `src/module-manifest.ts` | 统一声明 module 加载器、包名和预构建依赖。          |
| `src/modules.ts`         | 加载并默认导出当前宿主启用的全部 module。           |
| `src/vite-env.d.ts`      | 引入 Vite 客户端和 core 全局类型。                  |
| `.env`                   | 所有模式共享的应用标题、端口等环境变量。            |
| `.env.development`       | 开发模式 API 地址和代理配置。                       |
| `.env.production`        | 生产模式 API 地址和构建配置。                       |
| `index.html`             | Vite HTML 入口和应用挂载节点。                      |
| `package.json`           | 声明宿主命令以及 core、System、Vite 配置包依赖。    |
| `README.md`              | 当前宿主的职责和文件说明。                          |
| `tsconfig.json`          | 宿主源码的 TypeScript 检查范围和继承配置。          |
| `vite.config.ts`         | 将 module manifest 派生的构建参数交给共享宿主配置。 |

## 模块组合

`src/module-manifest.ts` 是宿主 module 配置的唯一来源。每个 manifest 项同时声明业务 module npm 包名、运行时加载器和可选预构建依赖；`src/modules.ts` 加载并默认导出全部 `AdminModule`，`vite.config.ts` 从 manifest 派生 `modulePackages` 与 `optimizeDependencies`。新增或删除 module 时，只需维护宿主依赖和这一份 manifest。

manifest 顺序就是模块注册顺序，只影响显式 `staticViews` 替换：后注册模块覆盖先注册模块的同一静态视图键，普通业务页面始终使用模块前缀隔离。

模块间页面跳转使用 Vue Router，跨模块代码引用只使用对方公开的 npm 子路径。宿主不得直接引用模块的 `src` 目录。

## 环境与输出

- `.env`：应用标题、端口和开发工具开关。
- `.env.development`：`/api`、`/events` 开发代理和路由模式。
- `.env.production`：`/admin/` 公共路径、PWA 和压缩设置。
- 生产构建输出：`backend/data/admin`，由内部 Vite 配置统一指定。

## 命令

```bash
pnpm --filter @liujitcn/kratos-admin-app dev
pnpm --filter @liujitcn/kratos-admin-app type:check
pnpm --filter @liujitcn/kratos-admin-app build
```
