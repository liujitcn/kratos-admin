# frontend/admin

账号密码、OAuth 票据兑换和微信登录统一返回 `mfa_remember_days`，表示当前登录策略允许的 MFA 设备免验证天数；为 `0` 时不提供记住设备选项。

管理后台采用 pnpm workspace，按“薄宿主 + core 底座 + 可选业务模块 + 工程工具”组织。宿主只负责组合和启动；页面、请求、RPC 类型和业务依赖归属对应模块包。
System 模块包含消息管理、消息分类和个人收件箱，并通过顶部工具显示未读数。个人中心展示本人的全部有效登录会话并标记当前设备，登录记录通过专用接口仅查询本人数据；登录管理下的在线会话页面仅供平台超级管理员查询和下线会话，租户名称显示在第一列，只有默认租户显示可搜索的租户下拉筛选，选项展示租户名称并按 `tenant_code` 精确匹配；账号搜索使用接口的 `keyword` 参数。
用户管理的新增、编辑表单采用双列布局，小屏自动切换单列；新增时密码与强度提示并排，备注独占一行。
代码生成工具的父级菜单仅展示目录，按后端八位菜单编号规则允许选择一至三级目录；页面、按钮和外链不作为候选父级。
代码生成等 SSE 订阅收到 403 时展示权限错误并停止重试，不触发重新登录提示；401 仍按认证失效处理。
代码生成表配置弹窗最大宽度为 1440px，生成后端与生成前端、生成 SQL 与状态均按等宽双列展示，与业务模块和父级菜单对齐；窄屏下自动改为单列。
Core 布局在头像菜单中提供锁定屏幕能力，锁屏密码摘要仅在当前锁屏会话期间持久化。

依赖方向固定为 `app -> business module -> core`。core 不依赖任何业务模块，业务模块之间默认也不互相引用；确需复用时，只能通过对方 `package.json#exports` 公开的 Interface。

## 目录职责

```text
frontend/admin
├── apps/admin                        # 装配当前全部 module 的默认宿主
├── packages/core                     # @liujitcn/kratos-admin-core 底座
│   ├── src/api/{base/v1,system/admin/v1} # 按 Proto 完整路径组织的底座请求
│   ├── src/components                # 公共组件
│   ├── src/layouts                   # 布局
│   ├── src/modules                   # 模块注册 interface
│   ├── src/rpc                       # 登录、菜单和底座能力 Proto 类型
│   └── src/views                     # 登录与静态错误页面
├── packages/modules/system           # @liujitcn/kratos-admin-system
│   └── src/{api,config,rpc,utils,views} # API 与 RPC 按 Proto 层级维护
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

管理端登录密码在安全上下文中通过 Web Crypto 加密；通过局域网 HTTP 地址访问且浏览器没有 Web Crypto 时，会回退到纯 JavaScript 实现并保持后端密码密文协议不变。该回退只解决运行兼容性，HTTP 仍可能暴露 Token 或遭受主动篡改，生产环境请使用 HTTPS。

本机通过局域网 IP 启用 HTTPS 开发服务：

```bash
cd ../..
bash scripts/generate-dev-cert.sh 192.168.1.100
```

将上面的 IP 替换为本机实际局域网 IP。脚本会在仓库根 `certs` 生成共享证书；然后在 `apps/admin/.env.development.local` 中加入 `VITE_HTTPS=true`。重启 `pnpm dev` 后访问 `https://192.168.1.100:8848`，首次访问需在浏览器中接受本地自签名证书；其他设备访问时也需要分别信任该证书。后端、Taro 和 uni-app 可复用同一目录下的证书。

管理端认证状态采用浏览器 Cookie-only 模式：刷新令牌由后端写入 Path 收窄的 HttpOnly Cookie，访问令牌只保存在页面内存，不写入 `localStorage` 或 `sessionStorage`。应用启动会主动清理旧版本遗留的持久化访问令牌，并通过非敏感的过期时间 Cookie 判断是否需要静默恢复会话。

## 国际化

管理端支持的语言由 core 与 System JSON 语言包自动发现，模块注册时校验语言键和占位符集合；登录页和顶部工具栏共用 locale store，切换语言不刷新页面，并保留当前路由、查询参数和未提交表单。

登录流程中的行为验证码、MFA 验证、首次绑定与恢复码弹窗统一在当前登录表单卡片区域内水平、垂直居中，中心跟随卡片位置变化；验证码与 MFA 验证使用相同宽度，窄屏限制宽度，矮屏允许弹窗内部滚动。

登录页租户输入框由 `base_config.showTenantCode` 控制，值为 `false` 或 `0` 时隐藏；语言切换入口在后端只返回一种 `language_pack` 时自动隐藏；其他登录方式只有在 `base/oauth/provider` 返回非空 `providers` 时显示。

语言偏好保存为 `kratos-admin:locale`。Axios、刷新令牌、原生 fetch、SSE 和 Swagger 请求统一发送 `Accept-Language`；动态菜单和字典由后端按 locale 返回，缺少当前语言译文时回退主语言。新增语言需要同步后端国际化目录、三个 workspace 的六个前端语言包目录，再执行仓库根目录的 `make i18n`；注册文件和 Day.js 映射由脚本生成。具体流程见 [国际化语言扩展指南](../../docs/国际化语言扩展指南.md)。

API 按 Proto 完整路径组织为 `api/base/v1`、`api/system/admin/v1` 等目录，`src/api` 只保留与服务文件同名的请求封装；运行配置和内部辅助实现分别放在 `src/config`、`src/utils`。RPC 保留 `rpc/base/v1`、`rpc/system/admin/v1` 等完整 Proto 层级。RPC 类型按真实消费者归属放置：core 保留登录、菜单、用户信息和启动期能力所需服务类型；System 自包含系统管理、个人中心、AI 及其依赖类型。修改 Proto 后在仓库根目录执行 `make -C frontend ts-admin`，命令会按两份 Buf 配置分别清理并生成 core 与 System 的 RPC；需要一次生成三个前端的 RPC 时执行 `make -C frontend ts`。服务端契约尚未完成细粒度拆分时，同一生成文件可能暂时包含当前包未调用的方法，不手写生成文件。

core 内部源码使用 `@/*`；业务模块使用 `@liujitcn/kratos-admin-core/*` 和自身包名。模块间页面跳转使用 Vue Router，代码复用禁止跨目录相对引用。

根 TypeScript 路径和宿主 Vite 源码别名只为各 npm 包映射 `package.json#exports` 声明的入口；core 实现内部使用 `@/*`，业务模块使用包名引用公开入口。

## 模块 interface

core 导出 `bootstrapAdminApp`、`defineAdminModule`、`setAdminDocumentTitle`、视图注册表以及顶部工具、用户菜单和路由行为扩展。浏览器标题优先使用公共配置接口返回的 `sysName`，宿主环境变量作为加载失败时的兜底。业务模块入口声明名称和页面加载器即可接入动态页面：

```ts
import type { Component } from "vue";
import { defineAdminModule } from "@liujitcn/kratos-admin-core";

const views = import.meta.glob<{ default: Component }>("./views/**/*.vue");

export const businessAdminModule = defineAdminModule({
  name: "business",
  views
});
```

业务页面只登记 module 前缀路径，因此 `views/list/index.vue` 统一由 `business/list/index` 解析，不提供 `list/index` 无前缀别名。后端菜单的 `component` 必须写完整 module 路径；不同 module 即使包含同名 `views` 页面也不会互相覆盖。宿主在 `src/module-manifest.ts` 中声明 module，`src/modules.ts` 加载并默认导出全部 module，Vite 构建配置从 manifest 派生：

```ts
export const adminModuleManifest = [
  {
    packageName: "@business/admin-module",
    load: async () => (await import("@business/admin-module")).businessAdminModule
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

export const businessAdminModule = defineAdminModule({
  name: "business",
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
pnpm dlx @liujitcn/kratos-admin-cli create business-admin --module business
pnpm dlx @liujitcn/kratos-admin-cli create business-admin --module business,report

# 当前仓库开发
pnpm module:create ../business-admin --module business
pnpm module:create ../business-admin --module business,report
pnpm module:create ../business-admin --module business --module report
```

生成结果：

```text
business-admin
├── apps/admin
│   └── README.md
├── packages/modules/business
│   └── README.md
├── packages/modules/report
│   └── README.md
├── scripts/build-package.mjs
├── package.json
├── pnpm-workspace.yaml
├── README.md
├── tsconfig.json
└── turbo.json
```

CLI 默认先引入 `@liujitcn/kratos-admin-system`，再按 `--module` 参数顺序引入并创建自有 module。`--module` 可重复使用，也接受逗号分隔名称；`--with` 只把额外的已发布 module 加入宿主组合，不会创建其源码，也不会制造业务 module 间的隐式依赖。CLI 拒绝覆盖已存在目录。


### CLI 生成边界

CLI 直接生成完整宿主、本地业务模块、四种语言源文件与注册入口、类型检查、lint 和打包配置。
支持本地 `system` 与其他模块一起创建；管理端本地 System 继承内置能力，运行时只注册一次，
构建仍扫描内置源码。应用端本地模块使用独立导入别名，保留内置 System。

通过 `--kratos-project` 生成与 Go 后端配套的前端：H5 输出到 `backend/data/<terminal>`，
管理端 CLI 同时在 workspace 父目录创建共享 `Makefile` 与 `scripts`。Go 调用方仅传参执行 CLI。
新增语言或修改语言文件后运行 `pnpm i18n:sync`；`pnpm i18n:check` 校验注册文件是否同步。

代码生成的国际化缺失警告持续显示在单项或批量生成确认弹窗中，确认或取消后关闭；缺失项按业务表折叠分组并显示数量，提示与明细分开呈现；明细区域限高滚动，确认按钮固定在内容区外。

开发模式下代码生成期间暂缓 Vite 热更新，任务结束并关闭进度弹窗后合并为一次刷新；开发服务器监听宿主和业务模块源码目录，自动发现生成的新页面；仅修改 Vite 配置或开发插件时需要重载配置或重启前端开发服务器。新增 Go 接口需要重新编译启动后端，重启后端不能替代前端热更新。

代码生成器的 API 文件与导入路径保留完整 Proto 目录，例如 `src/api/system/admin/v1/tenant_project.ts`。

菜单编辑弹窗最大宽度为 1440px，API 授权双栏等宽铺满可用空间，长名称自动换行；窄屏下表单及授权列表切换为单列。

## 系统管理菜单

菜单由后端初始化数据按当前语言动态下发，二级目录按下表排序，三级页面按表内顺序展示；菜单图标使用已全局注册的 Element Plus 图标。

| 二级菜单 ID | 二级菜单 | 三级菜单 |
| --- | --- | --- |
| `91010000` | 基础管理 | 字典管理、行政区划、语言管理、文件管理、系统配置 |
| `91020000` | 权限管理 | 菜单管理、接口管理、应用授权 |
| `91030000` | 登录管理 | 登录策略、在线会话 |
| `91040000` | 消息通知 | 消息管理、消息分类 |
| `91050000` | 数据脱敏 | 脱敏规则、存储脱敏策略、响应脱敏策略 |
| `91060000` | 数据备份 | 备份配置、备份记录、备份恢复记录 |
| `91070000` | 数据归档 | 归档配置、归档记录、归档恢复记录 |
| `91080000` | 日志审计 | 登录日志、操作日志、接口访问日志、数据访问日志、权限日志、策略评估日志 |
| `91090000` | 定时任务 | 任务管理、任务执行日志 |
| `91100000` | 系统运维 | 运行监控、运行日志、缓存查询、数据库迁移记录 |
| `91110000` | 开发工具 | 接口文档、代码生成 |

系统配置统一维护 Key、类型和值；类型为“表单”时在编辑弹窗加载注册的表单字段、提示与校验译文，不再提供独立系统参数页面。字典数据和代码生成配置、预览页继续作为隐藏路由。菜单 ID、角色引用及菜单翻译统一维护在后端 `v0.0.1/mysql` 初始化资源中；已执行过该版本的数据库不会自动重放修改。

系统配置编辑弹窗最大宽度为 1040px，仅表单类型将基础信息与动态配置表单拆分为两个 Tab，其他类型隐藏页签栏并直接展示编辑表单；编辑已有表单配置时默认打开配置表单。字段标签置顶、正文独立滚动，保存时统一校验并切换到错误所在 Tab。动态字段继续由 `registerRuntimeConfig` 注册定义：`component` 指定密码、数字、树选择等现有 ProForm 控件，`props`、`options`、`rules`、`visible` 控制交互和校验；`colSpan: 24` 可让长路径等字段占满整行，`rowBreakBefore` 控制换行，默认双列、窄屏单列。服务器路径使用文本输入，真实目录选择需业务模块提供服务器接口，不能用本机文件选择器替代。

国际化入口统一由 `base_language` 中启用的非主语言控制：仅主语言时列表显示普通文本、表单隐藏入口；启用其他语言时，列表字段或输入控件旁的国际化入口打开编辑窗口。空译文默认显示原值，保存主表单时将其补为原值；单项及批量翻译会替换目标语言的当前草稿；确认后表单译文随主表单保存，列表译文直接保存，取消不写入。代码生成的表描述、左树标题及字段描述复用同一入口。

构建日志使用 Turbo 普通文本界面，按任务分组输出并关闭颜色，适用于终端和 IDE 控制台。仓库统一构建入口会显示各平台的开始与完成阶段。

API 和组件自动导入声明统一通过 `pnpm generate:auto-import-types` 或 `make -C .. types-admin` 生成。生成器扫描宿主、core 和业务模块的 Vue 模板，使用组件解析器输出 `packages/core/types/generated/components.d.ts`；开发和生产构建不再改写此文件。新增自动导入组件后先运行该命令，再执行类型检查或构建。
