# frontend/uni-app

登录保留主动协议确认与正文查看，确认摘要通过公共请求层随本次登录和 MFA 请求发送，成功后清除；确认框不自动恢复为勾选。

账号密码、OAuth 票据兑换和微信登录统一返回 `mfa_remember_days`，表示当前登录策略允许的 MFA 设备免验证天数；为 `0` 时不提供记住设备选项。

账号密码登录与管理端共用 `POST /api/v1/base/session`，三方跳转授权共用 `/api/v1/base/oauth/authorization`，认证、Token 权限和本人 AI 对话均共享，可跨端查看和继续。并发登录遵循现有登录策略：允许时可同时使用，禁止时后登录撤销已有会话。AI 对话继续使用已保存的工具配置，并校验当前权限与审批。

`frontend/uni-app` 是独立的 pnpm workspace，提供可直接运行的 uni-app x 宿主、可发布应用底座、system 业务模块和项目脚手架。技术栈为 `uni-app x + Vue 3 + TypeScript + Vite + Pinia + Sass`，当前支持 H5、微信小程序和 x App 构建。

应用端只复用管理端的分层思想，不依赖 `frontend/admin` 的源码或 workspace。当前保留首页、登录（含 TOTP/WebAuthn MFA）、协议、WebView、个人中心、设置（含 MFA 绑定）、个人资料、AI 助手和站内信收件箱，不包含商城、订单、支付或推荐业务。

## Workspace

```text
frontend/uni-app
├── apps
│   └── uni-app                # @liujitcn/kratos-uni-app，默认宿主，不发布
│       ├── App.uvue
│       ├── main.ts
│       ├── pages
│       ├── pagesMember
│       ├── scripts/generate-pages.mjs
│       ├── module-manifest.ts
│       ├── manifest.json
│       ├── pages.json
│       ├── README.md
│       └── package.json
├── packages
│   ├── core                   # @liujitcn/kratos-uni-app-core v0.0.55
│   │   ├── src                # 底座运行时、页面、状态和 x 运行入口
│   │   ├── test
│   │   ├── README.md
│   │   └── package.json
│   ├── modules
│   │   └── system             # @liujitcn/kratos-uni-app-system v0.0.55
│   │       ├── src            # 个人中心、默认设置包装和 AI
│   │       ├── README.md
│   │       └── package.json
│   └── cli                    # @liujitcn/kratos-uni-app-cli v0.0.55
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

- `core` 负责认证、请求、配置、Pinia、公共设置页、基础页面、状态页、动态导航和 uni-app x 页面生成器。
- `system` 负责个人中心、资料、默认设置包装和 AI 页面，只通过 `core` 的公开 exports 复用底座能力。
- `apps/uni-app` 只维护宿主入口、manifest、模块清单、x 页面生成器和 Vite 配置。
- `cli` 创建同结构的独立 pnpm workspace。

各包的目录边界、功能和公开入口见：

- [宿主应用](apps/uni-app/README.md)
- [应用底座](packages/core/README.md)
- [system 模块](packages/modules/system/README.md)
- [项目脚手架](packages/cli/README.md)

workspace 根目录不再保留旧单体应用的 `src`、`bin` 或空模板目录，业务源码只能位于对应的宿主或 package 内。

普通矩形控件统一使用 `--kratos-page-radius`（默认 `12rpx`），由 core 全局主题维护；页面不再单独设置圆角数值，使用规则见 [应用底座](packages/core/README.md#统一圆角)。

## 其他 workspace 迁移

`cloud-scp`、`kratos-shop` 等引用本 workspace 公共包的项目统一按 x 迁移：宿主改为根目录 `App.uvue`、`main.ts`、`pages/`、`pagesMember/` 和 `manifest.json`，业务页面与公共组件统一使用 `.uvue`，Vite 只接入 `createKratosUniPlugin()`，页面由各自的 `prepare:x` 生成器写入。公共包名和 CLI 名保持不变，但不保留旧 `.vue` 入口；迁移顺序是先升级 core/system，再迁移业务模块，最后切换宿主构建命令。

## 页面装配

宿主在 `apps/uni-app/module-manifest.ts` 维护唯一模块清单，顺序决定静态视图覆盖优先级：

```ts
export const moduleManifest = [coreModule, systemModule]
```

每个模块使用 `defineKratosAppModule()` 声明：

- `pages`：页面样式和可选物理路由覆盖。
- `views`：稳定 `viewKey` 到物理页面的映射。
- `icons`：动态导航可引用的模块内图标。

`pnpm run prepare:x` 扫描各模块的 `src/views/**/*.uvue`，把页面 wrapper、`pages.json` 和静态资源生成到 x 宿主根目录；开发和构建命令都会先执行该步骤。后注册模块可以用相同物理路由或 `viewKey` 替换前面模块的静态页面。

自定义顶部导航统一使用 core 的 `KratosNavBar`：普通页、tabBar 页和有无操作按钮共用相同标题栏高度，组件负责安全区、返回行为、按钮尺寸与微信胶囊避让。新页面只传标题和操作，外层不要重复添加顶部安全区或导航栏高度；具体用法见 [通用顶部导航](packages/core/README.md#通用顶部导航)。

登录页同样复用公共导航栏，通过 `action` 命名插槽保留原生语言选择。H5 系统弹窗的外层页面保持透明，遮罩下继续显示来源页面；确认框尺寸和排版与 Taro H5 保持一致。

System 的 `KratosMyPage` 负责个人信息、AI 助手和顶部设置入口；默认“我的”页面保留空的 `extensions` 扩展区，当前宿主不预置其他业务卡片。页面覆盖规则见 [system 个人中心扩展](packages/modules/system/README.md#个人中心扩展)。

公共设置页通过 `KratosSettingsPage` 的 `extensions` 和 `security` 插槽扩展页面顶部与账户安全分组；System 的修改密码入口仅在 H5 装配，并调用 Base 账户安全服务。

宿主按 uni-app x 根目录组织源码：页面位于 `pages/` 和 `pagesMember/`，组件和业务实现位于公开 package。页面生成是显式的 `prepare:x` 步骤，不再依赖临时改写 `pages.json` 的旧页面事务。

## 动态导航

默认菜单接口与管理端 `base-menu` 服务保持一致：

```text
GET /api/v1/app/base/menu
```

请求 service path 为 `/v1/app/base/menu`，由统一 `/api` base 拼接。`base_menu.id = 99000000` 是隐藏的移动端固定根目录，后端逐层查询它下面的启用页面并保持扁平响应，core 再按 `parent_id` 构建菜单树；也可以通过 `setAppNavigationAdapter()` 接入自定义菜单数据源。根目录只关联移动菜单查询，每个页面菜单分别关联该页面实际调用的受保护 API。

菜单配置约束：

- `path` 使用 `app/` 前缀，支持 `:参数` 和 query。
- `name` 使用 `App` 前缀。
- 标题和主图标使用 `meta.title`、`meta.icon`，图标可以使用 HTTPS 地址或模块 `icons` 中注册的键。
- 移动端配置统一放在 `meta.app`，不增加独立表字段。
- `meta.app.view_key` 必须已由模块注册，接口不能直接下发任意组件路径。
- `meta.app.access` 支持 `PUBLIC`、`GUEST_ONLY`、`AUTHENTICATED`，与管理端登录权限语义一致。
- 根目录的二级页面固定作为 tab，首页使用 `99010000`、我的使用 `99090000`，`99020000` 至 `99080000` 预留给业务 tab。
- 二级 tab 的 `meta.app.in_tab_bar` 固定为真，数量只能为 0 或 2–5 项；选中图标使用 `meta.app.selected_icon`，下级页面固定为非 tab。

管理端菜单表单递归识别 `99000000` 的整棵移动端子树，显示逻辑路径、逻辑名称、视图键、访问模式、移动端图标及该页面 API 权限；提交时根据层级自动设置 tab，并把移动端配置统一包装到现有 `meta.app`。

导航配置按匿名态和登录态分别缓存。新配置会整份校验后原子切换；远端失败时使用当前身份最后一次成功缓存，没有缓存时使用本地默认菜单。

## 国际化

uni-app x 支持的语言由 core 和 System JSON 语言包自动发现，模块注册时校验 key 与占位符集合；登录、首页、状态页、WebView、个人中心、设置、资料和 AI 页面都通过 `t(key)` 使用固定文案。语言偏好保存为 `kratos-app:locale`，切换后不改变稳定路由和业务字段。

所有 `uni.request`、文件上传和 SSE 请求统一发送 `Accept-Language`。动态菜单沿用后端解析后的标题，缺少当前语言译文时回退主语言；新增语言需要同步后端国际化目录、三个 workspace 的六个前端语言包目录，再执行仓库根目录的 `make i18n`。具体流程见 [国际化设计与语言扩展](../../docs/国际化设计与语言扩展.md)。

页面 wrapper 统一挂载自绘 `KratosTabBar`，宿主不声明原生 `tabBar`。首页、消息、我的和动态扩展 tab 均使用普通页面路由。普通页面优先使用 `navigateTo`，下级页面会沿父级关系归属并高亮对应 tab。因此接口内容变化后可以调整菜单层级、逻辑路径和 `viewKey`，无需把每个业务路由写死在宿主。

H5 和微信端自绘底部 tab 都优先使用 `redirectTo` 替换当前页面，微信端仅在替换失败时回退到页面栈复用。tab 切换不使用 `switchTab` 或 `reLaunch`，避免原生 tab 配置冲突和页面入栈/出栈动画造成闪烁。普通页面仍优先使用 `navigateTo`。

上述策略由可发布的 core 包统一提供，业务模块可以覆盖 `HOME` 的物理页面或扩展至 2–5 个 tab。宿主通过 `prepare:x` 生成页面 wrapper 和静态资源，运行时继续使用自绘 `tabBar`；引用项目升级 core 后重新执行页面生成器。

## 开发与构建

```bash
cd frontend/uni-app
pnpm install

pnpm dev:h5
pnpm dev:mp-weixin
pnpm --filter @liujitcn/kratos-uni-app dev:app
```

H5 通过局域网 IP 访问时，先在仓库根目录运行 `bash scripts/generate-dev-cert.sh 192.168.1.100` 生成共享证书，再在 `.env.development-h5.local` 中设置 `VITE_APP_HTTPS=true`；后端使用 `APP_ENV=https` 时同时将 `VITE_APP_API_URL` 改为 `https://127.0.0.1:7001`。证书默认读取仓库根 `certs/dev-key.pem` 和 `certs/dev-cert.pem`。

上一级 Frontend Makefile 提供对应的聚合入口：

```bash
make -C .. run-uni-app      # 微信小程序开发
make -C .. run-uni-app-app  # uni-app x App 开发
make -C .. check-uni-app
make -C .. build-uni-app    # H5 构建
make -C .. build-uni-app-app # uni-app x App 构建
make -C .. package-uni-app
```

开发和构建命令由 `turbo.json` 保证先执行依赖包的 `build:entries`；独立测试和
exports 检查则通过 `prepare:modules` 完成相同准备，因此不依赖仓库中的历史
`dist`。H5 默认地址为 `http://localhost:5004`，开发 API 默认代理到
`http://127.0.0.1:7001`。
根命令由 Turbo 按 workspace 依赖图调度；启动或构建宿主前会先生成 core 和
system 的运行入口，命令名称和产物目录保持不变。

```bash
pnpm build:h5
pnpm build:mp-weixin
pnpm --filter @liujitcn/kratos-uni-app build:app
```

- H5 产物写入 `backend/web/uni-app`，后端通过 `/uni-app/` 挂载。
- 微信小程序产物写入 `apps/uni-app/dist/build/mp-weixin`，使用微信开发者工具导入；构建后自动检查实际 WXML 引用的 `kratos-*` 组件是否注册到 `usingComponents`，防止组件漏注册却构建成功。

## RPC 生成

uni-app x 与 admin 一样，由 backend API 契约目录统一维护 Buf 生成范围：

- `backend/api/buf.app.typescript.gen.yaml` 生成 core RPC。
- `backend/api/buf.app.core.typescript.gen.yaml` 生成 system RPC。

```bash
pnpm generate:rpc
# 等价于
make -C .. ts-uni-app
```

命令分别清理并生成 `packages/core/src/rpc` 和 `packages/modules/system/src/rpc`。frontend 不保存第二份 Buf 配置；RPC 文件是生成产物，不得手工修改。

## CLI

CLI 包提供 `kratos-uni-app` 命令：

```bash
pnpm dlx @liujitcn/kratos-uni-app-cli create my-app
pnpm dlx @liujitcn/kratos-uni-app-cli create my-app --module business
pnpm dlx @liujitcn/kratos-uni-app-cli create my-app --with @acme/pay
```

- 默认包含 system 模块。
- `--module` 创建 workspace 内本地模块。
- `--with` 添加已发布模块依赖。
- 目标目录已存在时拒绝覆盖。
- workspace 根目录、`apps/uni-app` 和每个本地模块都会生成 `README.md`，保证每个
  `package.json` 都有同级的目录结构与功能说明。

生成的宿主包含 Vue 3 `createSSRApp` 入口、模块清单、bootstrap 页面、manifest、Vite 配置、TypeScript 配置和 H5 HTML。

CLI 生成的 `.env.production` 不包含本机 API 地址，发布前需要填写实际 API 和静态资源地址。

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

CLI 包通过 `bin`、`src` 和 `assets` 发布可执行入口、生成逻辑和宿主图标。`pnpm test`
会额外使用 `pnpm pack` 打包 CLI，在临时目录解压并执行创建命令，验证图标内容和模块生成，
防止源码测试通过但 npm 包缺少运行时资源。

仓库级 `make -C frontend package-uni-app` 会依次执行 lint、类型检查、测试、exports
检查和三个包的构建；`make -C frontend publish-uni-app` 发布 core、system 和 CLI，默认
宿主 `@liujitcn/kratos-uni-app` 保持私有。

完整接口接入顺序见[服务接入指南](../../docs/服务接入指南.md)。


### CLI 生成边界

CLI 直接生成完整宿主、本地业务模块、四种语言源文件与注册入口、类型检查、lint 和打包配置。
支持本地 `system` 与其他模块一起创建；管理端本地 System 继承内置能力，运行时只注册一次，
构建仍扫描内置源码。应用端本地模块使用独立导入别名，保留内置 System。

通过 `--kratos-project` 生成当前仓库约定的前端：H5 输出到 `backend/data/<terminal>`，
管理端 CLI 同时在 workspace 父目录创建共享 `Makefile` 与 `scripts`。Go 调用方仅传参执行 CLI。
新增语言或修改语言文件后运行 `pnpm i18n:sync`；`pnpm i18n:check` 校验注册文件是否同步。

构建日志使用 Turbo 普通文本界面，按任务分组输出并关闭颜色，适用于终端和 IDE 控制台。仓库统一构建入口会显示各平台的开始与完成阶段。

应用配置加载成功后，core 会将 `i18n_customs` 按 `locale + key` 覆盖已注册的本地文案，并刷新页面翻译。仅覆盖已有 key 的非空文案；未配置或已删除的覆盖项恢复本地默认值，未知语言和未知 key 忽略。切换语言时沿用配置重载流程。
