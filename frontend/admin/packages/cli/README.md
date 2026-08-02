# @liujitcn/kratos-admin-cli

用于创建 kratos-admin 业务项目的公开命令行工具。生成结果是一个可独立安装、开发和构建的 pnpm workspace，其中宿主保持轻量，业务实现位于可独立发布的模块包。

## 目录与文件

```text
packages/cli
├── src
│   ├── index.ts
│   └── index.test.ts
├── templates/business-workspace
│   ├── apps/admin
│   ├── packages/modules/__MODULE_NAME__ # 可重复渲染的 module 模板
│   ├── scripts/build-package.mjs
│   ├── package.json
│   ├── pnpm-workspace.yaml
│   ├── README.md
│   ├── tsconfig.json
│   └── turbo.json
├── package.json
├── README.md
└── tsconfig.json
```

| 路径                                                             | 作用                                                        |
| ---------------------------------------------------------------- | ----------------------------------------------------------- |
| `src/index.ts`                                                   | 解析 `create` 命令并生成宿主与一个或多个业务 module。       |
| `src/index.test.ts`                                              | 验证多 module、默认 System、README 和依赖生成行为。         |
| `templates/business-workspace/`                                  | 完整业务 workspace 模板；目录和文本中的占位符由 CLI 替换。  |
| `templates/business-workspace/apps/admin/`                       | 生成项目的薄宿主模板。                                      |
| `templates/business-workspace/packages/modules/__MODULE_NAME__/` | 生成项目的可发布业务模块模板。                              |
| `templates/business-workspace/scripts/build-package.mjs`         | 生成业务模块发布源码和声明文件。                            |
| `package.json`                                                   | 声明 `kratos-admin` 可执行命令、脚本、Node 版本和发布配置。 |
| `README.md`                                                      | CLI 使用方法、模板结构和维护说明。                          |
| `tsconfig.json`                                                  | CLI 源码、测试的编译配置和 `dist` 输出规则。                |

模板内部每个含 `package.json` 的目录都有同级 README，生成时目录名、包名、模块变量和文档占位符会一起替换。

## 使用

```bash
pnpm dlx @liujitcn/kratos-admin-cli create shop-admin --module shop
pnpm dlx @liujitcn/kratos-admin-cli create shop-admin --module shop,order
pnpm dlx @liujitcn/kratos-admin-cli create shop-admin --module shop --with audit

# 当前仓库开发
pnpm module:create ../shop-admin --module shop
pnpm module:create ../shop-admin --module shop,order
pnpm module:create ../shop-admin --module shop --module order
pnpm module:create ../shop-admin --module shop,order --with audit
pnpm --filter @liujitcn/kratos-admin-cli test
```

CLI 始终先引入 `@liujitcn/kratos-admin-system`。`--module` 可重复使用，也接受逗号分隔名称，并为每个名称创建独立 module 包；`--with` 接收逗号分隔的额外已发布 module 名称，只装配依赖而不生成源码。CLI 拒绝覆盖已有目录；渲染失败时会清理本次创建的不完整目标目录。

生成结果遵守以下约束：

- 宿主通过 `src/module-manifest.ts` 维护装配清单，`src/modules.ts` 默认导出当前全部 module。
- 业务视图按 `<module>/<view>` 注册，后端菜单必须使用模块前缀。
- 业务模块只依赖 core 的公开 npm Interface，不引用 core 源码路径。
- API 按 Proto 一级领域组织，RPC 保留完整 Proto 目录层级。
- 静态页只能通过 `AdminModule.staticViews` 显式替换。

## 模板占位符

| 占位符                  | 生成内容                                  |
| ----------------------- | ----------------------------------------- |
| `__PROJECT_NAME__`      | 项目目录名称，例如 `shop-admin`。         |
| `__APP_PACKAGE__`       | 宿主包名，例如 `@shop/admin-app`。        |
| `__APP_DEPENDENCIES__`  | System、自有 module 和额外 module 依赖。  |
| `__MODULE_NAME__`       | kebab-case 模块名，例如 `shop`。          |
| `__MODULE_PASCAL__`     | PascalCase 模块名，例如 `Shop`。          |
| `__MODULE_PACKAGE__`    | 模块包名，例如 `@shop/admin-module`。     |
| `__MODULE_IDENTIFIER__` | 模块入口变量，例如 `shopAdminModule`。    |
| `__CORE_VERSION__`      | 当前 CLI 包版本对应的公开包 semver 范围。 |
| `__MODULE_FILTERS__`    | 全部自有 module 的 workspace 过滤参数。   |
| `__MODULE_MANIFEST__`   | 宿主模块加载器、包名和预构建依赖清单。    |
| `__MODULE_NAMES__`      | 文档中的全部自有 module 名称。            |
| `__MODULE_PACKAGES__`   | 文档中的全部自有 module 包名。            |
| `__MODULE_PATHS__`      | 全部自有 module 的 TypeScript 路径映射。  |
| `__MODULE_TREE__`       | 文档中的自有 module 目录树。              |
| `__MODULE_TABLE_ROWS__` | 文档中的自有 module 目录说明。            |

## 维护与验证

修改 CLI 或模板后执行：

```bash
pnpm --filter @liujitcn/kratos-admin-cli type:check
pnpm --filter @liujitcn/kratos-admin-cli test
```

测试会生成临时 workspace，并检查根、宿主、模块和 RPC README 中的占位符均已替换。模板新增包或改变目录职责时，必须同步更新对应目录的 README 和 `src/index.test.ts` 断言。

发布包包含编译后的 `dist/index.js`、类型声明、workspace 模板和本 README。仓库级
`make -C frontend package-admin` 会先执行测试和构建，再生成
`@liujitcn/kratos-admin-cli` tarball。
