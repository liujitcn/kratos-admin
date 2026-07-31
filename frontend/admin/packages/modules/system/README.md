# @liujitcn/kratos-admin-system

System 管理端业务模块。系统管理、个人中心、AI 助手的请求、RPC、页面与模块定义在同一个 npm 包内维护；宿主只有安装并注册本包后，才会提供这些能力。

## 目录结构

```text
packages/modules/system
├── src
│   ├── api
│   │   ├── base
│   │   └── system
│   ├── components
│   ├── rpc
│   ├── typings
│   ├── views
│   │   ├── ai
│   │   ├── base
│   │   ├── profile
│   │   └── tool
│   ├── ai.ts
│   ├── index.ts
│   └── module.ts
├── package.json
├── README.md
├── tsconfig.json
└── tsconfig.package.json
```

## 根文件

| 路径                    | 作用                                                    |
| ----------------------- | ------------------------------------------------------- |
| `src/index.ts`          | npm 主入口，导出 System 模块和 AI 扩展契约。            |
| `src/module.ts`         | 注册 System 页面、AI 顶部入口、个人中心菜单和路由行为。 |
| `src/ai.ts`             | 定义 AI 流程卡片扩展名称、类型和读取入口。              |
| `src/components/Ai.vue` | System 提供的 AI 顶部工具入口。                         |
| `src/rpc/**/*.ts`       | System 页面与 API 自包含的 RPC 类型。                   |
| `src/typings/*.d.ts`    | 声明 System 页面使用的 Markdown 和 Swagger 模块。       |
| `package.json`          | 声明依赖以及公开的模块入口、API 和 RPC 子路径。         |
| `README.md`             | System 模块的目录、页面和 API 文件说明。                |
| `tsconfig.json`         | 开发态类型检查配置。                                    |
| `tsconfig.package.json` | npm 发布声明文件生成配置。                              |

## API 文件

| 路径                          | 作用                                      |
| ----------------------------- | ----------------------------------------- |
| `src/api/base/ai_message.ts`  | AI 消息请求。                             |
| `src/api/base/ai_session.ts`  | AI 会话请求。                             |
| `src/api/base/ai_tool.ts`     | AI 工具请求。                             |
| `src/api/base/oauth.ts`       | 个人中心账号绑定请求。                    |
| `src/api/base/sse.ts`         | AI 与业务流式响应订阅。                   |
| `src/api/system/auth.ts`      | 个人中心认证请求。                        |
| `src/api/system/base_*.ts`    | System 基础资源管理请求。                 |
| `src/api/system/code_gen*.ts` | 代码生成、字段、Proto、表和进度订阅请求。 |
| `src/api/system/project_document.ts` | 多项目文档树与详情请求。           |

## 页面文件组

| 路径                                                           | 作用                      |
| -------------------------------------------------------------- | ------------------------- |
| `src/views/ai/chat/`                                           | AI 会话、消息和附件页面。 |
| `src/views/base/api/index.vue`                                 | API 资源管理页。          |
| `src/views/base/area/index.vue`                                | 行政区域管理页。          |
| `src/views/base/config/index.vue`                              | 系统配置管理页。          |
| `src/views/base/dept/index.vue`                                | 部门管理页。              |
| `src/views/base/dict/index.vue`                                | 字典类型管理页。          |
| `src/views/base/dict/item.vue`                                 | 字典项管理页。            |
| `src/views/base/job/index.vue`                                 | 定时任务管理页。          |
| `src/views/base/job/log.vue`                                   | 定时任务执行日志页。      |
| `src/views/base/log/index.vue`                                 | 系统日志页。              |
| `src/views/base/menu/index.vue`                                | 菜单管理页。              |
| `src/views/base/migration/index.vue`                           | 数据迁移管理页。          |
| `src/views/base/post/index.vue`                                | 岗位管理页。              |
| `src/views/base/role/index.vue`                                | 角色管理页。              |
| `src/views/base/tenant/index.vue`                              | 租户管理页。              |
| `src/views/base/user/index.vue`                                | 用户管理页。              |
| `src/views/base/user/components/dept-tree.vue`                 | 用户页的部门树筛选组件。  |
| `src/views/profile/`                                           | 个人中心与安全设置页面。  |
| `src/views/tool/api-doc/index.vue`                             | OpenAPI/Swagger 文档页。  |
| `src/views/tool/project-doc/index.vue`                         | 多项目 Markdown 树形导航与阅读页。 |
| `src/views/tool/code-gen/table/index.vue`                      | 代码生成数据表列表页。    |
| `src/views/tool/code-gen/columns/index.vue`                    | 代码生成字段配置页。      |
| `src/views/tool/code-gen/columns/option-copy.ts`               | 字段选项复制规则。        |
| `src/views/tool/code-gen/proto/index.vue`                      | Proto 生成配置页。        |
| `src/views/tool/code-gen/preview/index.vue`                    | 代码生成预览入口。        |
| `src/views/tool/code-gen/preview/capabilities.ts`              | 预览文件能力和操作规则。  |
| `src/views/tool/code-gen/preview/data.ts`                      | 预览树和文件数据转换。    |
| `src/views/tool/code-gen/code-preview/index.vue`               | 单个文件代码预览页。      |
| `src/views/tool/code-gen/components/CodeGenProgressDialog.vue` | 代码生成进度弹窗。        |
| `src/views/tool/code-gen/components/CodePreviewPane.vue`       | 代码内容预览面板。        |
| `src/views/tool/code-gen/config.ts`                            | 代码生成页面的公共配置。  |

## 接入

宿主安装本包后，在模块清单中注册：

```ts
import { systemAdminModule } from "@liujitcn/kratos-admin-system";

export const adminModules = [systemAdminModule];
```

动态菜单组件路径必须包含模块前缀，例如：

| 页面     | `component`                 |
| -------- | --------------------------- |
| 用户管理 | `system/base/user/index`    |
| 个人中心 | `system/profile/index`      |
| AI 助手  | `system/ai/chat/index`      |
| API 文档 | `system/tool/api-doc/index` |
| 项目文档 | `system/tool/project-doc/index` |

不再兼容 `base/user/index`、`profile/index` 等无模块前缀路径。不同业务模块可以拥有同名 `views`，core 会按 `<module>/<view>` 解析，不发生覆盖。

跨模块跳转使用 Vue Router；复用 System 代码时只引用 `package.json#exports` 公开的 npm 子路径。

System 页面只通过 `systemAdminModule.views` 注册，不作为 npm 子路径公开。模块内组件当前也不对外开放；出现真实跨模块复用需求时，按具体组件文件增加显式导出，不提供 `components/*` 通配入口。

登录、菜单和用户信息等运行底座属于 core；个人中心、AI 助手及其顶部入口属于 System。System 可以引用 core 的公共导出，core 不得反向依赖 System；其他业务模块只能通过 `AdminModule.staticViews` 显式替换默认登录页或状态页。

System 的 API 与 RPC 都按 Proto 层级维护。当前服务端契约尚未完成细粒度拆分，因此 System 与 core 可能分别包含同一服务生成文件；这些文件由项目命令生成，不在本包手工去重或改写。

## AI 扩展

System 导出 `ADMIN_AI_EXTENSION`、`AdminAiExtension` 和 `getAdminAiExtension()`。其他业务模块可以在 `AdminModule.extensions` 中使用该扩展名提供 `flowBlocks` 组件，AI 会话页会读取并渲染它。

仓库当前没有内置流程卡片组件，也没有默认注册该扩展；未提供时 AI 会话仍按普通消息展示。扩展组件由实际业务模块自行实现，不属于 System 的内置能力。

## 命令

```bash
pnpm --filter @liujitcn/kratos-admin-system type:check
pnpm --filter @liujitcn/kratos-admin-system build:package
pnpm --filter @liujitcn/kratos-admin-system pack
```
