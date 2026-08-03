# frontend/taro-app

`frontend/taro-app` 是独立的 pnpm workspace，以 React 18 和 Taro 4 实现与 `frontend/uni-app` 相同的应用功能和视觉。当前支持 H5 与微信小程序，包含首页、登录、协议、WebView、个人中心、设置、个人资料和 AI 助手，不包含商城、订单、支付或推荐业务。

## Workspace

```text
frontend/taro-app
├── apps/taro-app                 # 私有 Taro 宿主，不发布
├── packages/core                 # 运行时、基础页面和构建 runner
├── packages/ui                   # NutUI 主题与按需图标适配
├── packages/modules/system       # 个人中心、设置、资料和 AI
├── packages/cli                  # 独立 workspace 脚手架
├── scripts                       # package exports 边界检查
├── package.json                  # 公共命令和开发依赖
├── pnpm-workspace.yaml           # workspace 包范围
└── turbo.json                    # 跨包任务依赖
```

依赖方向固定为：

```text
apps/taro-app -> packages/modules/* -> packages/ui -> packages/core
```

core 不依赖 UI 或业务模块；system 只通过 core 和 UI 的公开 exports 复用能力；宿主只负责组合模块和平台配置。

详细说明：

- [私有宿主](apps/taro-app/README.md)
- [应用底座](packages/core/README.md)
- [UI 基础包](packages/ui/README.md)
- [system 模块](packages/modules/system/README.md)
- [项目脚手架](packages/cli/README.md)

## 页面装配

`apps/taro-app/src/module-manifest.ts` 是唯一模块清单，声明顺序决定页面、稳定 `viewKey` 和图标的覆盖优先级。模块通过 `defineKratosTaroModule()` 声明运行时能力，通过 `defineKratosTaroBuildModule()` 暴露构建期页面描述。

宿主只提交固定的 `pages/bootstrap` 页面。core runner 在开发或构建开始前完成以下操作：

- 扫描各模块 `src/views/**/*.tsx`，忽略任意层级的 `components`。
- 生成页面 wrapper 和页面 config，统一挂载自绘 `KratosTabBar`，并为 H5 默认导航页补齐与 uni-app 一致的顶部栏。
- 按 `pages*` 根目录拆分主包和分包。
- 合并各模块 `src/static` 到宿主，宿主已有文件优先，模块同名文件后注册优先。
- 构建退出后恢复原始 `app.config.ts` 并删除临时文件。

异常退出后，下次命令会根据 `.kratos-taro-app-pages-state.json` 自动恢复。H5 和微信小程序开发进程可以同时持有同一份页面装配事务；所有进程退出后，runner 才恢复宿主文件。两个目标的默认产物目录分别是 `dist/h5` 和 `dist/mp-weixin`。

## 开发与构建

```bash
cd frontend/taro-app
pnpm install
pnpm dev:h5
pnpm dev:mp-weixin
pnpm build:h5
pnpm build:mp-weixin
```

- H5 开发地址默认是 `http://localhost:5002`，`/api` 和 `/events` 代理到 `http://localhost:7001`。
- H5 生产产物写入 `backend/data/taro-app`，访问地址是 `/taro-app/`。
- 微信小程序开发和生产产物统一写入 `apps/taro-app/dist/mp-weixin`，微信开发者工具可直接导入该目录。
- `KRATOS_TARO_API_URL` 配置后端地址；`KRATOS_TARO_API_BASE`、`KRATOS_TARO_STATIC_URL`、`KRATOS_TARO_PUBLIC_PATH` 可分别覆盖 API 前缀、静态资源地址和 H5 公共路径。

设计稿宽度为 750。迁移自 uni-app 的 `rpx` 直接写作 Taro 设计稿 `px`；必须保持物理像素的原 `px` 使用 Taro 不转换写法。

模块内需要固定路径的打包资源统一放在 `src/static`。页面直接展示的图片通过所属包的 `static/*` export 静态导入，例如 `@liujitcn/kratos-taro-app-core/static/images/avatar.png`，确保 Webpack 在 H5 与微信页面 chunk 中生成正确引用；`resolveBundledAsset('static/...')` 只用于动态菜单图标等无法静态导入的运行时路径。

## 国际化

Taro 支持的语言由 core 和 System JSON 语言包自动发现，模块注册时校验 key 与占位符集合；登录、首页、状态页、WebView、个人中心、设置、资料和 AI 页面都通过 `t(key)` 使用固定文案。语言偏好保存为 `kratos-app:locale`，切换后不改变稳定路由和业务字段。

所有 `Taro.request`、文件上传和 SSE 请求统一发送 `Accept-Language`。动态菜单沿用后端解析后的标题，缺少当前语言译文时回退主语言；新增语言需要同步后端错误目录、三个 workspace 的六个前端语言包目录和代码生成 `catalog.json`，再执行仓库根目录的 `make i18n-sync`。具体流程见 [国际化语言扩展指南](../../docs/国际化语言扩展指南.md)。

## RPC 生成

Taro RPC 模板统一位于 `backend/api`：

- `buf.taro-app.typescript.gen.yaml` 生成 core RPC。
- `buf.taro-app.core.typescript.gen.yaml` 生成 system RPC。

```bash
pnpm generate:rpc
# 等价于
make -C ../../backend ts-taro-app
```

RPC 是生成产物，不得手工修改。

## CLI

```bash
pnpm dlx @liujitcn/kratos-taro-app-cli create my-app
pnpm dlx @liujitcn/kratos-taro-app-cli create shop-app --module shop,order
pnpm dlx @liujitcn/kratos-taro-app-cli create my-app --with @acme/customer-module
```

生成项目沿用相同的 React/Taro 技术栈、模块清单、runner 事务协议和 H5/微信构建方式。本地模块独立维护 `pages`、运行时入口与构建期入口，适合后续按业务域加入商城等能力。

## 包与验证

4 个公开包版本必须一致：

- `@liujitcn/kratos-taro-app-core`
- `@liujitcn/kratos-taro-app-ui`
- `@liujitcn/kratos-taro-app-system`
- `@liujitcn/kratos-taro-app-cli`

```bash
pnpm lint
pnpm tsc
pnpm test
pnpm check:exports
pnpm build:packages
pnpm build:h5
pnpm build:mp-weixin
```

`check:exports` 验证公开目标、版本一致性和跨包导入边界；`test` 覆盖模块覆盖优先级、导航、runner 事务、CLI 脚手架和 AI SSE；`build:packages` 在 `dist/npm` 生成 4 个 tarball。
