# frontend/app

`frontend/app` 是可直接运行的 uni-app 应用端，也是 npm 包 `@liujitcn/kratos-app` 的源码。当前只包含通用首页、账户与登录、协议、WebView、设置、个人资料和 AI 会话，不包含商城、商品、订单、支付或推荐业务。

技术栈为 `uni-app + Vue 3 + TypeScript + Vite + Pinia + Sass`，主要适配微信小程序和 H5。

## 目录

```text
frontend/app
├── bin/uni.mjs            # npm 包提供的 Uni CLI 启动包装器
├── public                 # H5 静态资源
├── scripts                # Vite 解析验证脚本
├── src
│   ├── api/base           # 登录、配置、文件、OAuth 和 AI service
│   ├── api/system         # system.app 认证、地区和字典 service
│   ├── modules            # 具名模块注册表
│   ├── pages              # 主包页面
│   ├── pagesMember        # 设置、资料和 AI 分包
│   ├── rpc                # Proto 生成的 TypeScript 类型
│   ├── static             # 应用内静态资源
│   ├── stores             # Pinia 状态
│   ├── styles             # 全局样式
│   ├── types              # 手写类型声明
│   ├── utils              # 请求、鉴权、路由、密码和文件工具
│   ├── manifest.json      # uni-app 应用配置
│   └── pages.json         # 页面、分包和 tabBar
├── package.json
├── vite.cjs               # npm 的 `./vite` 公共入口
├── vite.config.ts         # 当前应用的 Vite 配置
└── vite.d.ts
```

## 当前页面

| 页面 | 路由 |
| --- | --- |
| 首页 | `pages/index/index` |
| 我的 | `pages/my/my` |
| 登录 | `pages/login/login` |
| 协议详情 | `pages/login/protocal` |
| WebView | `pages/webview/webview` |
| 设置 | `pagesMember/settings/settings` |
| 个人资料 | `pagesMember/profile/profile` |
| AI 助手 | `pagesMember/ai/index` |

主包只有 `src/pages`，当前唯一分包是 `src/pagesMember`。新增页面时以 `src/pages.json` 为准，不在文档中预留尚不存在的业务分包。

## 开发与构建

```bash
cd frontend/app
pnpm install
pnpm dev:h5
```

H5 开发端口由 `.env.development-h5` 配置，当前为 `http://localhost:5002`；开发 API 默认代理到 `http://localhost:7001`。

微信小程序开发：

```bash
pnpm dev:mp-weixin
```

也可以从仓库根目录执行：

```bash
make -C frontend run-app
```

H5 生产构建：

```bash
pnpm build:h5
```

该命令把产物写入 `backend/data/app`。目录中存在 `index.html` 时，后端通过 `/app/` 挂载应用。

其他 App、小程序和快应用平台的命令以 `package.json#scripts` 为准；是否可用取决于对应平台工具链和 uni-app 支持情况。

## 接口与状态

- 页面通过 `src/api` 的 service 调用后端，不直接调用 `uni.request`。
- `src/api/base` 使用 `base.v1` 契约；`src/api/system` 使用 `system.app.v1` 契约。
- `src/utils/http.ts` 统一处理 API 地址、认证头、Token 刷新、上传拦截和错误提示。
- Token 读写集中在 `src/utils/auth.ts`，登录跳转和原路返回集中在 `src/utils/navigation.ts`。
- 全局状态位于 `src/stores/modules`，由 `src/stores/index.ts` 汇总。
- `src/rpc` 由后端 `make ts-app` 生成，禁止手工修改或复制等价类型。

## npm 包边界

包根入口导出 `bootstrapKratosApp`、`defineKratosAppModule`、注册表函数和内置 `kratosAppModule`。当前 `KratosAppModule` 只有 `name` 字段，因此它只提供具名注册边界，不会自动注册页面、API、状态或组件。

包当前公开以下源码子路径：

- `@liujitcn/kratos-app/api/*`
- `@liujitcn/kratos-app/rpc/*`
- `@liujitcn/kratos-app/pages/*`
- `@liujitcn/kratos-app/pagesMember/*`
- `@liujitcn/kratos-app/utils/*`
- `@liujitcn/kratos-app/stores/*`
- `@liujitcn/kratos-app/static/*`

`kratos-app-uni` 是包提供的 Uni CLI 包装器。`@liujitcn/kratos-app/vite` 当前只公开 Vite 配置类型、环境加载和 `createKratosUniPlugin()`；页面合并函数 `kratosApp()` 仍是本仓库 `src/vite.ts` 的内部构建能力，没有作为 npm 子路径导出。

当前应用自己的 `vite.config.ts` 使用该内部插件：构建时合并公共与宿主 `pages.json`，同路径页面以宿主配置为准；缺失页面临时生成 wrapper，构建结束后恢复宿主文件。插件的 `modules` 参数目前只声明模块边界，不读取模块页面。

## 多端约束

平台差异使用 uni-app 的 `#ifdef`/`#ifndef` 条件编译。修改登录、路由、存储、上传或图片预览时，至少检查 `MP-WEIXIN`、`H5`、`APP-PLUS` 分支；当前页面优先保证微信小程序和 H5 可用。

完整接口接入顺序见[服务接入指南](../../docs/服务接入指南.md)。

## 验证

```bash
cd frontend/app
pnpm test:vite
pnpm lint
pnpm tsc
```
