# Codex 规则（frontend/admin）

## 页面开发

- 列表页统一使用“`ProTable + FormDialog + ProForm`”结构；能通过 `columns`、`headerActions`、`cellType`、`actions` 配置表达的内容统一走配置，不堆砌具名插槽。
- 图片列用 `cellType: "image"`，状态列用 `cellType: "status"`；弹窗优先 `FormDialog`，仅当表单结构明显不适合 `ProForm` 时才用 `ProDialog` 或 `el-dialog`。
- 页面样式优先复用 `packages/core/src/styles/common.scss` 与 `packages/core/src/styles/element-dark.scss` 的主题变量，不写死浅色常量；必须同时兼容亮色、暗黑、灰色、色弱四种模式（灰色/色弱走全局滤镜）。
- 需要新增页面级颜色变量时，先补充到全局主题变量再消费，不在单页 `html.dark` 零散覆盖。

## 列表与弹窗

- 数据流分层清晰：表格请求封装为 `requestXxxTable` 并用 `buildPageRequest` 处理分页；弹窗开关、表单重置、提交、删除、状态切换分别独立方法。
- 批量删除与单项删除复用同一方法，兼容对象、对象数组、ID、ID 数组入参。
- 确认弹窗文案优先展示 `name`、`label`、`code` 字段，格式“字段中文名：字段值”；弹窗关闭时显式重置表单和校验状态。
- 编辑态回填、下拉预加载、提交后刷新表格在页面方法里显式处理，不隐式耦合到基础组件。

## 组件扩展

- `ProTable`/`FormDialog` 只沉淀高复用、低业务耦合的能力（如图片列、状态列、通用按钮）；页面级业务流程、请求编排、权限分支不下沉，不为抽象而抽象。

## 自动导入与类型

- Element Plus 运行时 API 和图标走 `unplugin-auto-import`，不重复手写 import；类型（`FormRules` 等）保持显式导入。
- core 手写源码使用 `@/*`，业务模块通过 `@liujitcn/kratos-admin-core/*` 和自身 npm 包名引用；RPC 生成文件保留生成器输出的相对路径。npm 发布产物由 `pnpm build:package` 生成，禁止手改任意 `dist/package`。
- API 按 Proto 一级领域放入 `src/api/base`、`src/api/system` 等目录；RPC 保留 Proto 的完整生成层级，不把 API 或 RPC 扁平化。
- RPC 按前端能力归属放置：登录、菜单、用户信息等运行契约属于 core；个人中心、AI 和 System 管理契约属于 System；其他业务模块契约放在对应模块的 `src/rpc`。业务模型优先引用所属包的生成类型，不重复定义等价类型。
- 自动生成文件放 `packages/core/types/generated`（`auto-imports.d.ts`、`components.d.ts`）；`packages/core/src/typings` 只放手写声明。调整自动导入配置时同步确认 `packages/core/build/plugins.ts`、`internal/lint-config/oxlint.json`、根 `tsconfig.json` 与包级 `tsconfig.json`。

## 校验与文档

- 本次任务全部改动完成后统一执行一次 `pnpm lint:oxlint`（涉及类型改动时同时执行 `pnpm type:check`）并处理报错，作为收尾步骤；改动过程中间不要逐次编辑逐次校验，避免重复全量扫描。因历史遗留无法全量通过时，说明执行范围、报错文件和阻塞原因。
- 新增或修改业务功能后同步检查更新 `README.md`。

## 注释补充

- 新增或修改 `interface`、`type` 等类型定义时补充中文类型注释；字段语义不直观时字段也补注释。
