# Codex 规则（frontend/uni-app）

## 项目概览
- 技术栈：`uni-app + Vue 3 + TypeScript + Vite + Pinia + Sass`，独立 pnpm workspace，当前支持 H5 和微信小程序。
- 依赖方向固定为 `apps/uni-app -> packages/modules/* -> packages/core`；宿主只负责组合和启动，core 不依赖业务模块。

## 目录与职责
- `apps/uni-app` 是私有薄宿主，只维护入口、manifest、模块清单、固定 bootstrap 页面和 Vite 配置。
- `packages/core` 是可发布底座，维护启动、认证、请求、状态、动态导航、基础页面、公共静态资源和构建插件。
- `packages/modules/system` 是可发布业务模块，维护个人中心、资料、设置和 AI 的 API、RPC 与页面。
- `packages/cli` 是可发布脚手架，负责创建同结构的独立 uni-app workspace。
- 页面放在所属包的 `src/views`；页面私有组件放页面目录下 `components`，至少两处复用才提升为包级公共组件。

## 页面与模块
- 每个模块通过 `defineKratosAppModule()` 声明 `pages`、`views` 和 `icons`；新增页面同时更新所属包的 `pages.ts` 和模块 `views` 映射。
- `apps/uni-app/src/module-manifest.ts` 是宿主唯一模块清单，顺序决定静态页面和 `viewKey` 的覆盖优先级。
- 宿主 `pages.json` 只提交固定 bootstrap 页面；模块页面 wrapper 和 static 由 core Vite 插件临时生成，禁止提交临时事务文件或手工维护生成页。
- 动态菜单使用稳定 `viewKey`，接口不能直接下发任意组件路径；tabBar 使用 core 自绘组件，不配置原生 tabBar。

## 接口与状态
- 业务请求统一通过所属包 `src/api` 的 service 发起，不在页面、store 或组件直接写 `uni.request`。
- 请求、鉴权、刷新 token、错误提示统一复用 core 的 `src/utils/http.ts`；token 读写统一走 core 的 `src/utils/auth.ts`。
- 业务模块只能通过 `@liujitcn/kratos-uni-app-core` 的公开 exports 复用底座，不相对引用其他包源码。
- 全局共享状态放 core 的 `src/stores`；业务模块状态优先保留在所属模块，页面局部状态留在页面内。
- RPC 按真实消费者归属 core 或业务模块，生成文件保留 Proto 完整层级，不手写、不扁平化。

## RPC 生成
- Buf 配置统一位于 `backend/api`，frontend 下不维护第二份生成模板。
- uni-app RPC 统一执行 `pnpm generate:rpc` 或 `make -C backend ts-app`，分别生成 core 与 system 的 `src/rpc`。
- 生成产物只能通过项目命令更新，不手写或复制等价类型。

## 样式与多端兼容
- 全局样式和 Sass 变量统一复用 core 的 `src/styles/base.scss` 与 `src/uni.scss`。
- 平台差异显式使用 `#ifdef`/`#ifndef` 条件编译；修改登录、路由、存储、上传和图片预览时同时检查 `MP-WEIXIN` 与 `H5`。
- 页面样式优先保证微信小程序可用，同时兼顾 H5，不在单页发明第二套全局视觉规则。

## 校验与文档
- 本次任务全部改动完成后统一执行一次 `pnpm lint` 与 `pnpm tsc`；涉及模块协议、构建插件、CLI 或包边界时，同时执行 `pnpm test`、`pnpm check:exports` 和对应构建命令。
- 新增页面、目录调整、构建方式、环境变量或公开 Interface 变化后，同步更新 workspace 和所属包的 `README.md`。
