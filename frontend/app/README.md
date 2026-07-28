# frontend/app

`frontend/app` 是 `@liujitcn/kratos-app` 的源码包，基于 `uni-app + Vue 3 + TypeScript + Vite + Pinia + Sass` 提供通用应用宿主。它不绑定具体行业业务，只承载框架、登录态、账户资料、协议页面和基础 AI 会话。

## 目录职责

```text
frontend/app
├── public                 # H5 静态资源目录
├── src
│   ├── api/base           # 基础接口 service
│   ├── api/system         # system.app 接口 service
│   ├── pages              # 首页、账户、登录、协议和 WebView
│   ├── pagesMember        # 设置、个人资料和 AI 会话
│   ├── rpc                # proto 生成的 TypeScript 类型与客户端
│   ├── stores             # Pinia 状态
│   ├── styles             # 全局样式
│   ├── types              # 手写类型
│   ├── utils              # 请求、鉴权、路由和文件工具
│   ├── manifest.json      # uni-app 应用配置
│   └── pages.json         # 页面、分包和 tabBar 配置
├── package.json
└── vite.config.ts
```

## 页面结构

主包 `src/pages`：

- 首页：`pages/index/index`
- 账户中心：`pages/my/my`
- 登录与协议：`pages/login`
- WebView：`pages/webview/webview`

会员分包 `src/pagesMember`：

- 设置：`pagesMember/settings/settings`
- 个人资料：`pagesMember/profile/profile`
- 智能助手：`pagesMember/ai/index`

## 环境要求

- `Node.js >= 16.18.0`
- `pnpm`
- 后端服务默认运行在 `http://localhost:7001`
- 微信开发者工具（调试微信小程序时需要）

安装依赖：

```bash
cd frontend/app
pnpm install
```

## 启动与构建

启动 H5：

```bash
cd frontend/app
pnpm dev:h5
```

默认地址：`http://localhost:5002`。

启动微信小程序：

```bash
cd frontend/app
pnpm dev:mp-weixin
```

也可以从仓库根目录执行：

```bash
make -C frontend run-app
```

构建 H5：

```bash
cd frontend/app
pnpm build:h5
```

构建产物输出到 `backend/data/app`，后端启动后可通过 `/app` 访问。

## 接口、状态与生成代码

- 请求统一通过 `src/api` 下的 service 发起。
- `src/api/base` 对应基础协议，`src/api/system` 对应系统应用协议。
- `src/rpc` 是生成产物，由后端 `make ts-app` 通过 `ts-proto` 生成，生成文件保留默认相对导入，不手工维护等价类型；人工源码使用 `@/*` 引用包内模块。
- 全局状态放在 `src/stores/modules`，并通过 `src/stores/index.ts` 汇总。
- 请求封装、鉴权、刷新令牌和错误提示集中在 `src/utils/http.ts`。
- 令牌读写统一走 `src/utils/auth.ts`。

## 宿主接入

- `bootstrapKratosApp` 创建 uni-app 实例并注册公共模块，业务宿主传入自己的 `App.vue`、Pinia 实例和模块列表。
- `defineKratosAppModule` / `registerKratosAppModule` 提供业务模块注册边界。
- `kratosApp()` 是 Vite 页面注册插件，在构建期合并宿主与公共 `pages.json`；宿主同路径页面优先，缺失页面生成临时 wrapper 指向公共页面，构建结束后清理，不复制源码。
- `kratos-app-uni` 是共享包提供的 Uni CLI wrapper，底座依赖、Vite 插件和版本由本包统一管理，商城等业务宿主无需重复声明。
- 包通过 `@liujitcn/kratos-app/api/*`、`@liujitcn/kratos-app/rpc/*`、`@liujitcn/kratos-app/utils/*` 等子路径导出公共能力；`@liujitcn/kratos-app/vite` 仅供构建配置加载页面插件。

后端接口、应用端页面和生成代码的完整接入顺序见仓库的[服务接入指南](../../docs/服务接入指南.md)。

## 多端兼容

页面默认优先保证微信小程序端可用，同时兼顾 H5。涉及登录、路由、存储、上传和预览等平台敏感逻辑时，使用 uni-app 条件编译标记区分平台。新增页面和接口必须属于基础壳子、`base` 或 `system.app` 范围。

## 校验

```bash
cd frontend/app
pnpm lint
pnpm tsc
```
