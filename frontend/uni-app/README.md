# frontend/uni-app

`frontend/uni-app` 是独立的 pnpm workspace，提供可直接运行的 uni-app 宿主、可发布应用底座、system 业务模块和项目脚手架。技术栈为 `uni-app + Vue 3 + TypeScript + Vite + Pinia + Sass`，当前支持 H5 和微信小程序。

应用端只复用管理端的分层思想，不依赖 `frontend/admin` 的源码或 workspace。当前保留首页、登录、协议、WebView、个人中心、设置、个人资料和 AI 助手，不包含商城、订单、支付或推荐业务。

## Workspace

```text
frontend/uni-app
├── apps
│   └── uni-app                # @liujitcn/kratos-uni-app，默认宿主，不发布
│       ├── src
│       │   ├── pages/bootstrap
│       │   ├── module-manifest.ts
│       │   ├── manifest.json
│       │   └── pages.json
│       ├── README.md
│       └── package.json
├── packages
│   ├── core                   # @liujitcn/kratos-uni-app-core v0.0.17
│   │   ├── src                # 底座运行时、页面、状态和构建插件
│   │   ├── test
│   │   ├── README.md
│   │   └── package.json
│   ├── modules
│   │   └── system             # @liujitcn/kratos-uni-app-system v0.0.17
│   │       ├── src            # 个人中心、设置和 AI
│   │       ├── README.md
│   │       └── package.json
│   └── cli                    # @liujitcn/kratos-uni-app-cli v0.0.17
│       ├── bin
│       ├── src
│       ├── test
│       ├── README.md
│       └── package.json
├── scripts                    # 包导出边界检查
├── README.md                  # workspace 总览和公共命令
├── package.json               # workspace 脚本与公共开发依赖
├── pnpm-workspace.yaml        # workspace 包范围
└── turbo.json                 # 跨包任务依赖与缓存配置
```

依赖方向固定为：

```text
@liujitcn/kratos-uni-app
  -> @liujitcn/kratos-uni-app-system
  -> @liujitcn/kratos-uni-app-core
```

- `core` 负责认证、请求、配置、Pinia、基础页面、状态页、动态导航和构建插件。
- `system` 负责个人中心、资料、设置和 AI 页面，只通过 `core` 的公开 exports 复用底座能力。
- `apps/uni-app` 只维护宿主入口、manifest、模块清单、bootstrap 页面和 Vite 配置。
- `cli` 创建同结构的独立 pnpm workspace。

各包的目录边界、功能和公开入口见：

- [宿主应用](apps/uni-app/README.md)
- [应用底座](packages/core/README.md)
- [system 模块](packages/modules/system/README.md)
- [项目脚手架](packages/cli/README.md)

workspace 根目录不再保留旧单体应用的 `src`、`bin` 或空模板目录，业务源码只能位于对应的宿主或 package 内。

## 页面装配

宿主在 `apps/uni-app/src/module-manifest.ts` 维护唯一模块清单，顺序决定静态视图覆盖优先级：

```ts
export const moduleManifest = [coreModule, systemModule]
```

每个模块使用 `defineKratosAppModule()` 声明：

- `pages`：页面样式和可选物理路由覆盖。
- `views`：稳定 `viewKey` 到物理页面的映射。
- `icons`：动态导航可引用的模块内图标。

构建插件自动扫描各模块的 `src/views/**/*.vue`，忽略任意层级的 `components` 目录。后注册模块可以用相同物理路由或 `viewKey` 替换前面模块的静态页面。

宿主 `pages.json` 只提交固定 bootstrap 页面。开发或构建期间，插件会临时生成页面 wrapper、合并页面配置，并把模块 static 逐文件合并到宿主；同名宿主文件优先，事务只记录和清理插件实际写入的文件。正常退出后自动恢复。若进程被强制结束，下次启动会先根据 `.kratos-uni-app-pages-state.json` 恢复上一次事务。同一宿主同时只允许一个 H5 或小程序开发/构建进程持有页面事务，切换平台前需要先停止当前进程。

## 动态导航

默认菜单接口与管理端 `base-menu` 服务保持一致：

```text
GET /api/v1/app/base/menu
```

请求 service path 为 `/v1/app/base/menu`，由统一 `/api` base 拼接。`base_menu.id = 999` 是隐藏的移动端固定根目录，后端逐层查询它下面的启用页面并保持扁平响应，core 再按 `parent_id` 构建菜单树；也可以通过 `setAppNavigationAdapter()` 接入兼容契约。根目录只关联移动菜单查询，每个页面菜单分别关联该页面实际调用的受保护 API。

菜单配置约束：

- `path` 使用 `app/` 前缀，支持 `:参数` 和 query。
- `name` 使用 `App` 前缀。
- 标题和主图标使用 `meta.title`、`meta.icon`，图标可以使用 HTTPS 地址或模块 `icons` 中注册的键。
- 移动端配置统一放在 `meta.app`，不增加独立表字段。
- `meta.app.view_key` 必须已由模块注册，接口不能直接下发任意组件路径。
- `meta.app.access` 支持 `PUBLIC`、`GUEST_ONLY`、`AUTHENTICATED`，与管理端登录权限语义一致。
- 根目录的二级页面固定作为 tab，首页使用 `99901`、我的使用 `99909`，`99902` 至 `99908` 预留给业务 tab。
- 二级 tab 的 `meta.app.in_tab_bar` 固定为真，数量只能为 0 或 2–5 项；选中图标使用 `meta.app.selected_icon`，下级页面固定为非 tab。

管理端菜单表单递归识别 `999` 的整棵移动端子树，显示逻辑路径、逻辑名称、视图键、访问模式、移动端图标及该页面 API 权限；提交时根据层级自动设置 tab，并把移动端配置统一包装到现有 `meta.app`。

导航配置按匿名态和登录态分别缓存。新配置会整份校验后原子切换；远端失败时使用当前身份最后一次成功缓存，没有缓存时使用本地默认菜单。

项目不配置原生 `tabBar`。页面 wrapper 统一挂载 `KratosTabBar`，tab 路由使用 `reLaunch`，普通页面优先使用 `navigateTo`；下级页面会沿父级关系归属并高亮对应 tab。因此接口内容变化后可以调整菜单层级、逻辑路径和 `viewKey`，无需把每个业务路由写死在宿主。

## 开发与构建

```bash
cd frontend/uni-app
pnpm install

pnpm dev:h5
pnpm dev:mp-weixin
```

开发和构建命令由 `turbo.json` 保证先执行依赖包的 `build:entries`；独立测试和
exports 检查则通过 `prepare:modules` 完成相同准备，因此不依赖仓库中的历史
`dist`。H5 默认地址为 `http://localhost:5002`，开发 API 默认代理到
`http://localhost:7001`。
根命令由 Turbo 按 workspace 依赖图调度；启动或构建宿主前会先生成 core 和
system 的运行入口，命令名称和产物目录保持不变。

```bash
pnpm build:h5
pnpm build:mp-weixin
```

- H5 产物写入 `backend/data/app`，后端通过 `/app/` 挂载。
- 微信小程序产物写入 `apps/uni-app/dist/build/mp-weixin`，使用微信开发者工具导入。

## RPC 生成

uni-app 与 admin 一样，由 backend API 契约目录统一维护 Buf 生成范围：

- `backend/api/buf.app.typescript.gen.yaml` 生成 core RPC。
- `backend/api/buf.app.core.typescript.gen.yaml` 生成 system RPC。

```bash
pnpm generate:rpc
# 等价于
make -C ../../backend ts-app
```

命令分别清理并生成 `packages/core/src/rpc` 和 `packages/modules/system/src/rpc`。frontend 不保存第二份 Buf 配置；RPC 文件是生成产物，不得手工修改。

## CLI

CLI 包提供 `kratos-uni-app` 命令：

```bash
pnpm dlx @liujitcn/kratos-uni-app-cli create my-app
pnpm dlx @liujitcn/kratos-uni-app-cli create my-app --module orders
pnpm dlx @liujitcn/kratos-uni-app-cli create my-app --with @acme/pay
```

- 默认包含 system 模块。
- `--module` 创建 workspace 内本地模块。
- `--with` 添加已发布模块依赖。
- 目标目录已存在时拒绝覆盖。
- workspace 根目录、`apps/uni-app` 和每个本地模块都会生成 `README.md`，保证每个
  `package.json` 都有同级的目录结构与功能说明。

生成的宿主包含 Vue 3 `createSSRApp` 入口、模块清单、bootstrap 页面、manifest、Vite 配置、TypeScript 配置和 H5 HTML。

## 包与验证

三个公开包版本必须保持一致：

- `@liujitcn/kratos-uni-app-core`
- `@liujitcn/kratos-uni-app-system`
- `@liujitcn/kratos-uni-app-cli`

```bash
pnpm lint
pnpm tsc
pnpm test
pnpm check:exports
pnpm build:packages
```

`check:exports` 会检查 export target、跨包源码导入和相对路径越界。`build:packages` 在 `dist/npm` 生成三个 tarball，发布入口位于各包的 `dist`，TypeScript 源码和类型通过 exports 白名单公开。

仓库级 `make -C frontend package-uni-app` 会依次执行 lint、类型检查、测试、exports
检查和三个包的构建；`make -C frontend publish-uni-app` 发布 core、system 和 CLI，默认
宿主 `@liujitcn/kratos-uni-app` 保持私有。

完整接口接入顺序见[服务接入指南](../../docs/服务接入指南.md)。
