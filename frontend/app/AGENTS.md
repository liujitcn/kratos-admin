# Codex 规则（frontend/app）

## 项目概览
- 技术栈：`uni-app + Vue 3 + TypeScript + Vite + Pinia + Sass`；主包 `src/pages`，会员分包 `src/pagesMember`，订单分包 `src/pagesOrder`。

## 目录与职责
- 新增页面放入最接近的包目录，并同步更新 `src/pages.json`（路由、分包、导航）。
- 页面私有组件放页面目录下 `components`；通用组件放 `src/components`（≥2 处复用才提升）；组合式逻辑放 `src/composables`；工具放 `src/utils`；全局状态只放 `src/stores`。

## 页面开发
- `script setup` + TypeScript；按“状态定义 → 计算属性 → 数据加载 → 事件方法”组织，初始化请求收敛为 `loadData`/`getXxxData` 这类语义方法。
- 需要登录态的操作先检查 `useUserStore()` 再决定跳转登录或继续；已有页面的 URL 参数名不随意改写。

## 接口与状态
- 业务请求统一通过 `src/api` 的 service 类发起（命名沿用 `XxxServiceImpl`/`defXxxService`），不在页面、store、组件直接写 `uni.request`；新增接口先补 service 再调用。
- 请求封装、鉴权、刷新 token、错误提示统一复用 `src/utils/http.ts`；token 读写统一走 `src/utils/auth.ts`。
- 业务类型优先从 `src/rpc` 导入；`src/rpc` 视为生成产物，不随意手改。
- store 模块放 `src/stores/modules` 并在 `src/stores/index.ts` 统一导出；页面局部临时状态留在页面内。

## 样式与多端兼容
- 全局样式复用 `src/styles/base.scss`、`src/styles/fonts.scss`、`src/uni.scss`；页面样式沿用现有色值与间距，不单页发明视觉规则。
- 平台差异必须显式用 `#ifdef`/`#ifndef` 条件编译，不写成运行时分支；修改登录、路由、存储、分享、支付、预览图片等平台敏感逻辑时，同时检查 `MP-WEIXIN`、`H5`、`APP-PLUS`；仅支持单端的能力在代码与注释中明确说明。
- 样式优先保证微信小程序端可用，同时兼顾 H5。

## 变更边界
- 改动优先限制在页面、私有组件、service、store 边界内；影响路由结构、分包、请求封装、登录态、全局 store 或通用组件接口时，先梳理影响范围再动手。

## 校验与文档
- 本次任务全部改动完成后统一执行一次 `pnpm lint` 与 `pnpm tsc`，作为收尾步骤；改动过程中间不要逐次编辑逐次校验，避免重复全量扫描。因历史问题无法全量通过时，说明执行了什么、失败在哪、是否由本次改动引起。
- 新增页面、目录调整、构建方式或环境变量变化后同步更新 `README.md`；发现 README 与实际不符时顺手修正。
