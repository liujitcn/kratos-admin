# frontend/admin

管理后台采用 pnpm workspace，按“薄宿主 + core 底座 + 可选业务模块 + 工程工具”组织。宿主只负责组合和启动；页面、请求、RPC 类型和业务依赖归属对应模块包。

依赖方向固定为 `app -> business module -> core`。core 不依赖任何业务模块，业务模块之间默认也不互相引用；确需复用时，只能通过对方 `package.json#exports` 公开的 Interface。

## 目录职责

```text
frontend/admin
├── apps/admin                        # 装配当前全部 module 的默认宿主
├── packages/core                     # @liujitcn/kratos-admin-core 底座
│   ├── src/api/{base,system}         # 按 Proto 领域组织的底座请求
│   ├── src/components                # 公共组件
│   ├── src/layouts                   # 布局
│   ├── src/modules                   # 模块注册 interface
│   ├── src/rpc                       # 登录、菜单和底座能力 Proto 类型
│   └── src/views                     # 登录与静态错误页面
├── packages/modules/system           # @liujitcn/kratos-admin-system
│   └── src/{api,rpc,views}           # API 与 RPC 按 Proto 领域层级维护
├── packages/cli                      # @liujitcn/kratos-admin-cli
│   └── templates/business-workspace  # 完整 pnpm workspace 模板
├── internal/vite-config              # 当前仓库宿主构建配置
├── internal/tsconfig                 # 共享 TypeScript 配置
├── internal/lint-config              # 共享 Oxlint 配置
├── scripts/build-package.mjs          # core 与业务模块的 npm 发布构建器
├── package.json                       # workspace 命令与公共开发依赖
├── pnpm-workspace.yaml
├── tsconfig.json                      # workspace TypeScript 路径映射
└── turbo.json
```

`apps/admin/src/module-manifest.ts` 是默认宿主 module 配置的唯一来源，同时声明运行时加载器、Vite 扫描包和预构建依赖；`apps/admin/src/modules.ts` 加载并默认导出当前宿主的全部 module。core 的业务视图目录只保留登录页，403、404、500、Pending 由 core 提供默认静态实现；个人中心、AI 助手与系统管理页面均由 `systemAdminModule` 提供。个人中心和 AI 助手由后端菜单动态注册路由，不在宿主中声明静态业务路由。

## 根目录文件

| 路径                                   | 作用                                                              |
| -------------------------------------- | ----------------------------------------------------------------- |
| `apps/`                                | 可运行的管理端宿主集合；当前默认宿主是 `admin`。                  |
| `packages/core/`                       | 可发布的管理端运行时、布局、组件和基础页面。                      |
| `packages/modules/`                    | 可选业务模块集合，API 与页面在模块内一起维护。                    |
| `packages/cli/`                        | 可发布的独立业务 workspace 命令行工具和模板。                     |
| `internal/`                            | 仅供当前源码仓库使用的 Vite、TypeScript 和 lint 配置包。          |
| `scripts/build-package.mjs`            | 生成 npm 包源码副本和 TypeScript 声明，并转换 core 内部源码别名。 |
| `package.json`                         | 声明 workspace 级命令、工具依赖、Node 与 pnpm 版本。              |
| `pnpm-lock.yaml`                       | 固定整个 workspace 的依赖解析结果。                               |
| `pnpm-workspace.yaml`                  | 声明 `apps`、`packages` 和 `internal` 下的 workspace 包。         |
| `tsconfig.json`                        | 继承共享配置并声明本仓库源码路径映射。                            |
| `turbo.json`                           | 定义开发、构建、发布构建和类型检查任务关系。                      |
| `AGENTS.md`                            | 管理端目录的协作与代码约束。                                      |
| `.editorconfig`                        | 编辑器通用缩进、换行和字符集规则。                                |
| `.gitignore`                           | 管理端本地产物忽略规则。                                          |
| `.oxlintignore`                        | Oxlint 扫描排除规则。                                             |
| `.prettierignore`、`.prettierrc.cjs`   | Prettier 排除范围和格式配置。                                     |
| `.stylelintignore`、`.stylelintrc.cjs` | Stylelint 排除范围和样式规则。                                    |
| `lint-staged.config.cjs`               | 仓库级 Git hook 使用的管理端暂存文件检查配置。                    |
| `commitlint.config.cjs`                | Git 提交信息校验规则。                                            |
| `postcss.config.cjs`                   | PostCSS 与浏览器前缀处理配置。                                    |
| `README.md`                            | workspace 架构、公共命令和模块接入说明。                          |

每个包含 `package.json` 的子目录都有同级 `README.md`，用于说明该包内部文件和目录职责。

管理端发布 `@liujitcn/kratos-admin-core`、`@liujitcn/kratos-admin-system` 和
`@liujitcn/kratos-admin-cli`。默认宿主 `@liujitcn/kratos-admin` 为私有包，不进入
npm 打包和发布清单。

## 开发与构建

```bash
cd frontend/admin
pnpm install
pnpm dev
pnpm check:exports
pnpm test
pnpm type:check
pnpm lint:oxlint
pnpm build
pnpm build:package
```

在 `frontend/admin` 目录也可以通过上一级 Makefile 执行常用流程：

```bash
make -C .. run-admin
make -C .. check-admin
make -C .. build-admin
make -C .. package-admin
```

默认宿主地址为 `http://localhost:8848`。环境变量位于 `apps/admin/.env*`，开发模式的 API 代理和生产构建输出目录由宿主 Vite 配置统一管理；当前生产构建写入 `backend/data/admin`。

## 国际化

管理端支持的语言由 core 与 System JSON 语言包自动发现，模块注册时校验语言键和占位符集合；登录页和顶部工具栏共用 locale store，切换语言不刷新页面，并保留当前路由、查询参数和未提交表单。

语言偏好保存为 `kratos-admin:locale`。Axios、刷新令牌、原生 fetch、SSE 和 Swagger 请求统一发送 `Accept-Language`；动态菜单和字典由后端按 locale 返回，缺少当前语言译文时回退主语言。新增语言需要同步后端错误目录、三个 workspace 的六个前端语言包目录和代码生成 `catalog.json`，再执行仓库根目录的 `make i18n-sync`；注册文件和 Day.js 映射由脚本生成。具体流程见 [国际化语言扩展指南](../../docs/国际化语言扩展指南.md)。

API 按 Proto 一级领域组织为 `api/base`、`api/system` 等目录；RPC 保留 `rpc/base/v1`、`rpc/system/admin/v1` 等完整 Proto 层级。RPC 类型按真实消费者归属放置：core 保留登录、菜单、用户信息和启动期能力所需服务类型；System 自包含系统管理、个人中心、AI 及其依赖类型。修改 Proto 后在 `backend` 执行 `make ts-admin`，命令会按两份 Buf 配置分别清理并生成 core 与 System 的 RPC；需要一次生成三个前端的 RPC 时执行 `make ts`。服务端契约尚未完成细粒度拆分时，同一生成文件可能暂时包含当前包未调用的方法，不手写生成文件。

core 内部源码使用 `@/*`；业务模块使用 `@liujitcn/kratos-admin-core/*` 和自身包名。模块间页面跳转使用 Vue Router，代码复用禁止跨目录相对引用。

根 TypeScript 路径和宿主 Vite 源码别名只为各 npm 包映射 `package.json#exports` 声明的入口。core 实现内部使用的 `@/*` 因源码传递编译而保留，但 `pnpm check:exports` 会扫描应用、core、业务模块和内部构建配置中的 import，拒绝未公开子路径、非 core 的 `@/*` 引用以及业务模块跨包相对引用；仓库级 pre-commit 会先处理暂存文件，再执行 core 依赖方向和包导出边界检查。

## 模块 interface

core 导出 `bootstrapAdminApp`、`defineAdminModule`、视图注册表以及顶部工具、用户菜单和路由行为扩展。业务模块入口声明名称和页面加载器即可接入动态页面：

```ts
import type { Component } from "vue";
import { defineAdminModule } from "@liujitcn/kratos-admin-core";

const views = import.meta.glob<{ default: Component }>("./views/**/*.vue");

export const orderAdminModule = defineAdminModule({
  name: "order",
  views
});
```

业务页面只登记 module 前缀路径，因此 `views/list/index.vue` 统一由 `order/list/index` 解析，不提供 `list/index` 无前缀别名。后端菜单的 `component` 必须写完整 module 路径；不同 module 即使包含同名 `views` 页面也不会互相覆盖。宿主在 `src/module-manifest.ts` 中声明 module，`src/modules.ts` 加载并默认导出全部 module，Vite 构建配置从 manifest 派生：

```ts
export const adminModuleManifest = [
  {
    packageName: "@order/admin-module",
    load: async () => (await import("@order/admin-module")).orderAdminModule
  }
];
```

### 替换静态页面

core 通过 `ADMIN_STATIC_VIEWS` 公开全部静态页面的固定视图键：

| 属性           | 视图键          |
| -------------- | --------------- |
| `LOGIN`        | `login/index`   |
| `FORBIDDEN`    | `error/403`     |
| `NOT_FOUND`    | `error/404`     |
| `SERVER_ERROR` | `error/500`     |
| `PENDING`      | `error/pending` |

业务模块通过 `staticViews` 把自己的任意页面显式映射到固定视图键，后注册模块替换先注册实现：

```ts
import { ADMIN_STATIC_VIEWS, defineAdminModule } from "@liujitcn/kratos-admin-core";

export const orderAdminModule = defineAdminModule({
  name: "order",
  staticViews: {
    [ADMIN_STATIC_VIEWS.NOT_FOUND]: () => import("./components/NotFound.vue"),
    [ADMIN_STATIC_VIEWS.PENDING]: () => import("./components/Pending.vue")
  }
});
```

普通 `views` 永远按模块名隔离，只有 `staticViews` 是可替换 Seam。core 的 npm Interface 只公开模块接入、组件白名单以及 `request`、`navigation`、`table`、`security`、`stores/runtime` 等稳定入口；业务模块不得依赖 core 的源码内部路径。

## 创建业务项目

CLI 生成的业务项目本身也是 pnpm workspace，包含独立宿主和可发布业务模块包：

```bash
pnpm dlx @liujitcn/kratos-admin-cli create shop-admin --module shop
pnpm dlx @liujitcn/kratos-admin-cli create shop-admin --module shop,order

# 当前仓库开发
pnpm module:create ../shop-admin --module shop
pnpm module:create ../shop-admin --module shop,order
pnpm module:create ../shop-admin --module shop --module order
```

生成结果：

```text
shop-admin
├── apps/admin
│   └── README.md
├── packages/modules/shop
│   └── README.md
├── packages/modules/order
│   └── README.md
├── scripts/build-package.mjs
├── package.json
├── pnpm-workspace.yaml
├── README.md
├── tsconfig.json
└── turbo.json
```

CLI 默认先引入 `@liujitcn/kratos-admin-system`，再按 `--module` 参数顺序引入并创建自有 module。`--module` 可重复使用，也接受逗号分隔名称；`--with` 只把额外的已发布 module 加入宿主组合，不会创建其源码，也不会制造业务 module 间的隐式依赖。CLI 拒绝覆盖已存在目录。
