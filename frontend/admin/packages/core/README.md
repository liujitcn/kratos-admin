# @liujitcn/kratos-admin-core

kratos-admin 前端底座包。它不包含具体业务模块，提供登录、菜单、用户信息、应用启动、动态路由、布局、运行状态、公共组件和静态状态页。宿主可以只安装 core；个人中心、AI 助手和系统管理能力必须由 System 模块提供。

## 目录结构

```text
packages/core
├── build/                    # Vite 配置辅助函数和插件组合
├── src
│   ├── api/                  # 按 Proto 领域组织的底座请求
│   │   ├── base/             # 登录、OAuth、配置和文件请求
│   │   └── system/           # 用户、菜单、字典和租户运行请求
│   ├── assets/               # 字体、图标和图片资源
│   ├── components/           # 公共 Vue 组件
│   ├── config/               # 前端运行配置
│   ├── directives/           # 全局指令
│   ├── enums/                # 公共枚举
│   ├── hooks/                # 组合式函数
│   ├── layouts/              # 布局和导航框架
│   ├── modules/              # 业务模块 interface 与视图注册表
│   ├── routers/              # 静态、动态路由
│   ├── rpc/                  # 底座运行和共享协议的 TypeScript 类型
│   ├── stores/               # Pinia 状态
│   ├── styles/               # 全局样式和主题
│   ├── typings/              # 手写全局声明
│   ├── utils/                # 公共工具与请求客户端
│   └── views/                # 登录与静态错误页面
├── types/generated/          # 自动导入和组件声明
├── package.json
├── README.md
├── tsconfig.json
├── tsconfig.package.json
└── vite.config.ts
```

## 根文件

| 路径                    | 作用                                                               |
| ----------------------- | ------------------------------------------------------------------ |
| `package.json`          | 声明 core 依赖、公开子路径、发布文件和构建命令。                   |
| `README.md`             | core 的目录、文件组、公共入口和边界说明。                          |
| `tsconfig.json`         | core 开发态类型检查配置。                                          |
| `tsconfig.package.json` | npm 发布声明文件的生成配置。                                       |
| `vite.config.ts`        | 导出 `defineAdminViteConfig`，供宿主组合模块扫描、插件和构建参数。 |

## 构建文件

| 路径               | 作用                                                       |
| ------------------ | ---------------------------------------------------------- |
| `build/getEnv.ts`  | 读取并规范化宿主的 Vite 环境变量。                         |
| `build/plugins.ts` | 组合 Vue、自动导入、组件解析、SVG、PWA、压缩等 Vite 插件。 |
| `build/proxy.ts`   | 根据环境变量生成开发服务器代理配置。                       |

## 源码文件组

| 路径                                   | 作用                                                                           |
| -------------------------------------- | ------------------------------------------------------------------------------ |
| `src/index.ts`                         | npm 主入口，导出启动函数和模块 interface。                                     |
| `src/auth.ts`                          | 业务模块使用的按钮权限入口。                                                   |
| `src/format.ts`                        | 业务模块使用的格式化入口。                                                     |
| `src/navigation.ts`                    | 业务模块使用的路由与跳转入口。                                                 |
| `src/request.ts`                       | 业务模块使用的请求客户端入口。                                                 |
| `src/security.ts`                      | OAuth、密码加密和密码强度入口。                                                |
| `src/table.ts`                         | ProTable 请求和选择值处理入口。                                                |
| `src/tenant.ts`                        | 租户选项入口。                                                                 |
| `src/components/ProTable.ts`           | ProTable 对外 Adapter，稳定组件入口为 `components/ProTable`。                  |
| `src/bootstrap.ts`                     | 创建 Vue 应用，注册 core 与宿主选择的业务模块后挂载。                          |
| `src/App.vue`                          | 根组件和路由出口。                                                             |
| `src/vite-env.d.ts`                    | Vite、构建变量和 `import.meta.glob` 类型引用。                                 |
| `src/api/base/*.ts`                    | Proto `base` 领域的登录、OAuth、配置和文件请求。                               |
| `src/api/system/*.ts`                  | Proto `system` 领域的认证、字典和租户运行请求。                                |
| `src/assets/fonts/`                    | 应用字体及字体样式。                                                           |
| `src/assets/iconfont/`                 | 内置图标字体。                                                                 |
| `src/assets/images/`                   | Logo、登录、错误页、头像和 OAuth 图标。                                        |
| `src/components/Card/`                 | 数据概览卡片。                                                                 |
| `src/components/CronExpression/`       | Cron 表达式编辑器。                                                            |
| `src/components/Dialog/`               | 通用弹窗和表单弹窗。                                                           |
| `src/components/Dict/`                 | 字典选择与字典值展示。                                                         |
| `src/components/ECharts/`              | ECharts 容器和图表配置。                                                       |
| `src/components/ErrorMessage/`         | 403、404、500 页面复用的错误状态展示组件。                                     |
| `src/components/Grid/`                 | 响应式表单网格及类型。                                                         |
| `src/components/ImportExcel/`          | Excel 导入交互。                                                               |
| `src/components/Loading/`              | 局部和全屏加载状态。                                                           |
| `src/components/PasswordStrength/`     | 密码强度提示。                                                                 |
| `src/components/ProForm/`              | 配置驱动表单、动态列表、键值列表和类型。                                       |
| `src/components/ProTable/`             | 配置驱动表格、分页、列设置和类型。                                             |
| `src/components/SearchForm/`           | 搜索表单和搜索字段渲染。                                                       |
| `src/components/SelectFilter/`         | 筛选条件选择器。                                                               |
| `src/components/SelectIcon/`           | 图标选择器。                                                                   |
| `src/components/SvgIcon/`              | SVG 图标渲染。                                                                 |
| `src/components/SwitchDark/`           | 明暗主题切换。                                                                 |
| `src/components/TreeFilter/`           | 树形筛选器。                                                                   |
| `src/components/Upload/`               | 单/多文件和单/多图片上传组件。                                                 |
| `src/components/WangEditor/`           | 富文本编辑器封装。                                                             |
| `src/config/index.ts`                  | 默认应用配置。                                                                 |
| `src/config/nprogress.ts`              | 页面切换进度条配置。                                                           |
| `src/directives/index.ts`              | 全局指令安装入口。                                                             |
| `src/directives/modules/*.ts`          | 权限、复制、防抖、拖拽、长按、节流和水印指令。                                 |
| `src/enums/httpEnum.ts`                | HTTP 状态和请求相关枚举。                                                      |
| `src/hooks/interface/index.ts`         | hooks 共用类型。                                                               |
| `src/hooks/useAuthButtons.ts`          | 页面按钮权限计算。                                                             |
| `src/hooks/useDownload.ts`             | 文件下载流程。                                                                 |
| `src/hooks/useHandleData.ts`           | 删除、状态变更等确认交互。                                                     |
| `src/hooks/useOnline.ts`               | 网络在线状态监听。                                                             |
| `src/hooks/useSelection.ts`            | 表格选择状态。                                                                 |
| `src/hooks/useTable.ts`                | 表格请求、分页、搜索和加载状态。                                               |
| `src/hooks/useTheme.ts`                | 主题初始化和切换。                                                             |
| `src/hooks/useTime.ts`                 | 问候时间和时间展示。                                                           |
| `src/layouts/Layout*/`                 | 经典、分栏、横向和纵向布局实现。                                               |
| `src/layouts/components/`              | Header、Menu、Tabs、Main、Footer 和主题抽屉。                                  |
| `src/layouts/index.vue`                | 布局选择入口。                                                                 |
| `src/layouts/indexAsync.vue`           | 异步布局入口。                                                                 |
| `src/modules/index.ts`                 | `AdminModule` interface、静态视图键、模块注册、视图解析和通用扩展读取。        |
| `src/modules/kratosAdmin.ts`           | 注册 core 自带登录页和全部静态状态页的默认实现。                               |
| `src/routers/index.ts`                 | Vue Router 创建与路由守卫入口。                                                |
| `src/routers/modules/dynamicRouter.ts` | 把动态菜单解析为已注册模块页面，未注册组件统一显示 Pending。                   |
| `src/routers/modules/staticRouter.ts`  | 声明登录入口、布局和错误页；全部静态页面从模块视图注册表解析。                 |
| `src/rpc/**/*.ts`                      | core 独立运行和共享协议所需的生成类型，禁止手改。                              |
| `src/stores/index.ts`                  | Pinia 创建和持久化插件入口。                                                   |
| `src/stores/helper/persist.ts`         | 状态持久化辅助配置。                                                           |
| `src/stores/interface/index.ts`        | Store 共用类型。                                                               |
| `src/stores/modules/*.ts`              | 认证、配置、字典、全局设置、缓存页签、页签和用户状态。                         |
| `src/stores/runtime.ts`                | 业务模块可使用的运行期 Store 公开入口。                                        |
| `src/styles/*.scss`                    | Element Plus 覆盖、暗色主题、重置、公共样式和变量。                            |
| `src/styles/theme/*.ts`                | 侧栏、头部和菜单的运行时主题变量。                                             |
| `src/typings/*.d.ts`                   | 全局、Window、工具和第三方库的手写声明。                                       |
| `src/utils/*.ts`                       | 颜色、字典、校验、错误、事件、OAuth、密码、表格、请求、路由、SVG、租户等工具。 |
| `src/utils/is/index.ts`                | 常用类型与值判断。                                                             |
| `src/views/login/`                     | 登录页和登录表单。                                                             |
| `src/views/error/`                     | 403、404、500 和动态菜单 Pending 的默认路由页面。                              |
| `types/generated/auto-imports.d.ts`    | 自动导入 API 的工具生成声明。                                                  |
| `types/generated/components.d.ts`      | 自动导入 Vue 组件的工具生成声明。                                              |

## 模块边界

core 内部源码使用 `@/*`，不得引用任何 System 包或 System 源码。业务模块只通过 `@liujitcn/kratos-admin-core` 及 `package.json#exports` 声明的子路径引用底座；不得相对引用 `packages/core/src`。RPC 按能力归属放入 core 或对应业务模块，生成文件和 `types/generated` 禁止手工修改。

core 的 API 目录只保留底座运行时实际调用的请求封装。RPC 生成文件以服务为生成粒度；服务端契约尚未拆分完成时，core 所需的登录、菜单或用户服务文件可能暂含个人中心相关方法，这属于当前生成结果，不代表个人中心页面归属 core，也不在前端手工裁剪。

core 通过 `ADMIN_STATIC_VIEWS` 注册全部默认静态页面，后注册业务模块通过 `AdminModule.staticViews` 显式替换默认实现。固定视图键为 `login/index`、`error/403`、`error/404`、`error/500` 和 `error/pending`。普通业务页面只按 `<module>/<view>` 路径解析，不提供无前缀别名。个人中心与 AI 助手不在 core 中注册，使用这些能力的宿主必须安装并注册 System 模块。

## 公共 Interface

| 能力             | 入口                                                        |
| ---------------- | ----------------------------------------------------------- |
| 应用与模块注册   | `@liujitcn/kratos-admin-core`                               |
| 请求客户端       | `@liujitcn/kratos-admin-core/request`                       |
| 路由与跳转       | `@liujitcn/kratos-admin-core/navigation`                    |
| 权限             | `@liujitcn/kratos-admin-core/auth`                          |
| 格式化           | `@liujitcn/kratos-admin-core/format`                        |
| 密码与 OAuth     | `@liujitcn/kratos-admin-core/security`                      |
| 表格工具         | `@liujitcn/kratos-admin-core/table`                         |
| 租户选项         | `@liujitcn/kratos-admin-core/tenant`                        |
| 运行时 Store     | `@liujitcn/kratos-admin-core/stores/runtime`                |
| ProTable Adapter | `@liujitcn/kratos-admin-core/components/ProTable`           |
| ProTable 类型    | `@liujitcn/kratos-admin-core/components/ProTable/interface` |

其他 Vue 组件以 `package.json#exports` 中的明确白名单为准。禁止使用 `components/ProTable/index.vue`、`utils/*`、`hooks/*`、`stores/modules/*` 等实现路径。发布构建会把内部 `@/` 别名转换成包内相对路径，内部实现不占用公共子路径。

## 命令

```bash
pnpm --filter @liujitcn/kratos-admin-core test
pnpm --filter @liujitcn/kratos-admin-core type:check
pnpm --filter @liujitcn/kratos-admin-core build:package
pnpm --filter @liujitcn/kratos-admin-core pack
```
