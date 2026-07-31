<!-- prettier-ignore -->
# __PROJECT_NAME__

基于 kratos-admin 的独立业务管理端 workspace。默认包含一个薄宿主、System module 和自有 module（**MODULE_NAMES**）；依赖方向为 `apps/admin -> business module -> @liujitcn/kratos-admin-core`。

## 目录与文件

```text
__PROJECT_NAME__
├── apps/admin
│   ├── src
│   ├── package.json
│   └── README.md
__MODULE_TREE__
├── scripts
│   └── build-package.mjs
├── .gitignore
├── package.json
├── pnpm-workspace.yaml
├── README.md
├── tsconfig.json
└── turbo.json
```

| 路径          | 作用                                                |
| ------------- | --------------------------------------------------- |
| `apps/admin/` | 可运行的薄宿主，只负责启动和选择启用的业务 module。 |

**MODULE_TABLE_ROWS**
| `scripts/build-package.mjs` | 生成自有 module 的发布源码和 TypeScript 声明。 |
| `.gitignore` | 忽略依赖、缓存和构建产物。 |
| `package.json` | 声明 workspace 公共命令、工具依赖和运行时版本。 |
| `pnpm-workspace.yaml` | 声明宿主和业务 module 的 workspace 范围。 |
| `README.md` | 当前 workspace 的目录、文件和开发说明。 |
| `tsconfig.json` | TypeScript 公共配置以及全部自有 module 的源码映射。 |
| `turbo.json` | 定义开发、构建、发布构建和类型检查任务关系。 |

每个含 `package.json` 的目录都有同级 README，可继续向对应文件查看更细的职责说明。

## 开发

```bash
pnpm install
pnpm dev
pnpm type:check
pnpm build
pnpm build:package
pnpm package
```

| 命令                 | 作用                                  |
| -------------------- | ------------------------------------- |
| `pnpm dev`           | 启动 `apps/admin` 开发服务器。        |
| `pnpm type:check`    | 检查宿主和全部业务 module 类型。      |
| `pnpm build`         | 构建宿主应用。                        |
| `pnpm build:package` | 生成全部自有 module 的 npm 发布目录。 |
| `pnpm package`       | 构建全部自有 module 并生成 npm 包。   |

业务 API、RPC、页面和业务组件分别放在 `packages/modules/<module>/src`。业务页面对应的后端菜单组件路径必须使用 module 名称前缀，例如 `shop/index/index`。

宿主 module manifest 位于 `apps/admin/src/module-manifest.ts`，默认先加载 `@liujitcn/kratos-admin-system`，再按创建参数顺序加载自有 module。`apps/admin/src/modules.ts` 加载并默认导出当前全部 module；安装其他 module 后，只需更新宿主依赖和 manifest。

跨模块跳转使用 Vue Router；跨模块代码复用只允许引用对方 `package.json#exports` 公开的 Interface，不使用跨目录相对路径。
