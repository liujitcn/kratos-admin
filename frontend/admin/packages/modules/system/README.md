# @liujitcn/kratos-admin-system

System 管理端业务模块。系统管理、个人中心（含 MFA 与三方账号绑定）、AI 助手、开放授权客户端的请求、RPC、页面与模块定义在同一个 npm 包内维护；宿主只有安装并注册本包后，才会提供这些能力。

## 目录结构

```text
packages/modules/system
├── src
│   ├── api
│   │   │   ├── base/v1
│   │   │   └── system/admin/v1
│   ├── config
│   ├── utils
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
| `src/index.ts`          | npm 主入口，导出 System 模块、AI 扩展契约和租户项目管理组件。 |
| `src/module.ts`         | 注册 System 页面、AI 顶部入口、个人中心菜单和路由行为。 |
| `src/ai.ts`             | 定义 AI 流程卡片扩展名称、类型和读取入口。              |
| `src/components/ai/Ai.vue` | System 提供的 AI 顶部工具入口。 |
| `src/components/account/ChangePasswordForm.vue` | 个人中心与强制改密弹窗共用的修改密码表单。 |
| `src/components/account/ForcedPasswordDialog.vue` | 登录成功后在当前业务页面展示的强制改密弹窗。 |
| `src/components/i18n/DynamicI18nCell.vue` | 翻译资源列表单元格，悬停预览当前语言之外的内容。 |
| `src/components/i18n/DynamicI18nEditor.vue` | 动态资源多语言编辑器。 |
| `src/components/i18n/dynamicI18n.ts` | 动态资源翻译状态和表单辅助类型。 |
| `src/components/log/LogTable.vue` | 日志列表的通用表格组件。 |
| `src/components/log/log.ts` | 日志列表列和查询辅助函数。 |
| `src/components/notification/Notification.vue` | System 提供的站内信顶部工具入口。 |
| `src/components/tenant-project/TenantProjectSelect.vue` | 租户项目树选择器，默认租户显示“租户 / 项目”，普通租户显示当前租户项目。 |
| `src/components/tenant-project/TenantProjectText.vue` | 租户项目列表展示组件，按当前租户身份自动组合名称。 |
| `src/components/tenant-project/TenantProjectManager.vue` | 可复用的租户项目管理组件，支持追加列、业务数据加载、操作和作用域插槽。 |
| `src/components/tenant-project/tenant-project-manager.ts` | 租户项目管理组件的扩展类型、双键上下文和数据合并辅助函数。 |
| `src/components/tenant-project-grant/ProjectGrantDialog.vue` | 岗位、角色、部门和用户的项目授权弹窗。 |
| `src/rpc/**/*.ts`       | System 页面与 API 自包含的 RPC 类型。                   |
| `src/typings/*.d.ts`    | 声明 System 页面使用的 Markdown 和 Swagger 模块。       |
| `package.json`          | 声明依赖以及公开的模块入口、API 和 RPC 子路径。         |
| `README.md`             | System 模块的目录、页面和 API 文件说明。                |
| `tsconfig.json`         | 开发态类型检查配置。                                    |
| `tsconfig.package.json` | npm 发布声明文件生成配置。                              |

## API 文件

| 路径                                         | 作用                                      |
| -------------------------------------------- | ----------------------------------------- |
| `src/api/base/v1/ai_message.ts`              | AI 消息服务请求。                         |
| `src/api/base/v1/ai_session.ts`              | AI 会话服务请求。                         |
| `src/api/base/v1/ai_tool.ts`                 | AI 工具服务请求。                         |
| `src/api/base/v1/notification.ts`             | 站内信服务请求。                          |
| `src/api/base/v1/oauth.ts`                   | 个人中心三方账号绑定服务请求。            |
| `src/api/base/v1/sse.ts`                     | SSE 服务请求。                            |
| `src/api/system/admin/v1/auth.ts`            | 个人中心认证服务请求。                    |
| `src/api/system/admin/v1/base_*.ts`          | System 基础服务请求。                     |
| `src/api/system/admin/v1/base_i18n_custom.ts` | 管理端固定文案自定义翻译请求。             |
| `src/api/system/admin/v1/code_gen*.ts`       | 代码生成服务请求。                        |
| `src/api/system/admin/v1/oauth_client.ts`    | 开放授权客户端服务请求。                  |
| `src/api/system/admin/v1/ops_monitoring.ts`  | 运维监控服务请求。                        |
| `src/api/system/admin/v1/runtime_log.ts`     | 运行日志服务请求。                        |
| `src/api/system/admin/v1/cache.ts`           | 运行时缓存服务请求。                      |
| `@liujitcn/kratos-admin-core/api/base/v1/mfa` | 当前用户 MFA 服务请求。                   |

`src/config` 维护运行配置定义，`src/utils` 维护 SSE 订阅、响应归一化和代码生成序列化等内部辅助；这些文件不属于 Proto 服务 API。

## 页面文件组

| 路径                                                           | 作用                               |
| -------------------------------------------------------------- | ---------------------------------- |
| `src/views/ai/chat/`                                           | AI 会话、消息和附件页面。          |
| `src/views/base/api/index.vue`                                 | API 资源管理页。                   |
| `src/views/base/area/index.vue`                                | 行政区域管理页。                   |
| `src/views/base/config/index.vue`                              | 系统配置管理页。                   |
| `src/views/base/i18n-custom/index.vue`                         | 国际化自定义翻译管理页。           |
| `src/views/base/runtime-config/index.vue`                      | 日志入库回退配置页。               |
| `src/views/base/backup-management/archive-config/index.vue`    | 数据归档配置页。                   |
| `src/views/base/backup-management/archive-record/index.vue`    | 数据归档执行记录页。               |
| `src/views/base/backup-management/archive-restore/index.vue`   | 数据归档人工恢复与恢复记录页。     |
| `src/views/base/backup-management/backup-config/index.vue`    | 数据备份配置页。                   |
| `src/views/base/backup-management/backup-record/index.vue`    | 数据备份执行记录页。               |
| `src/views/base/backup-management/backup-restore/index.vue`   | 数据备份人工恢复与恢复记录页。     |
| `src/views/base/dept/index.vue`                                | 部门管理页。                       |
| `src/views/base/dict/index.vue`                                | 字典类型管理页。                   |
| `src/views/base/dict/item.vue`                                 | 字典项管理页。                     |
| `src/views/base/job/index.vue`                                 | 定时任务管理页。                   |
| `src/views/base/job/log.vue`                                   | 定时任务执行日志页。               |
| `src/views/base/log/index.vue`                                 | 系统日志页。                       |
| `src/views/base/menu/index.vue`                                | 菜单管理页。                       |
| `src/views/base/migration/index.vue`                           | 数据迁移管理页。                   |
| `src/views/base/post/index.vue`                                | 岗位管理页。                       |
| `src/views/base/role/index.vue`                                | 角色管理页。                       |
| `src/views/base/tenant/index.vue`                              | 租户管理页。                       |
| `src/views/base/tenant-project-grant/index.vue`                  | 项目授权管理页。                   |
| `src/views/base/user/index.vue`                                | 用户管理页。                       |
| `src/views/base/user/components/dept-tree.vue`                 | 用户页的部门树筛选组件。           |
| `src/views/profile/`                                           | 个人中心与安全设置页面。           |
| `src/views/tool/api-doc/index.vue`                             | OpenAPI/Swagger 文档页。           |
| `src/views/tool/code-gen/table/index.vue`                      | 代码生成数据表列表页。             |
| `src/views/tool/code-gen/columns/index.vue`                    | 代码生成字段配置页。               |
| `src/views/tool/code-gen/columns/option-copy.ts`               | 字段选项复制规则。                 |
| `src/views/tool/code-gen/proto/index.vue`                      | Proto 生成配置页。                 |
| `src/views/tool/code-gen/preview/index.vue`                    | 代码生成预览入口。                 |
| `src/views/tool/code-gen/preview/capabilities.ts`              | 预览文件能力和操作规则。           |
| `src/views/tool/code-gen/preview/data.ts`                      | 预览树和文件数据转换。             |
| `src/views/tool/code-gen/code-preview/index.vue`               | 单个文件代码预览页。               |
| `src/views/tool/code-gen/components/CodeGenProgressDialog.vue` | 代码生成进度弹窗。                 |
| `src/views/tool/code-gen/components/CodePreviewPane.vue`       | 代码内容预览面板。                 |
| `src/views/tool/code-gen/config.ts`                            | 代码生成页面的公共配置。           |

| `src/views/base/language/index.vue` | 语言配置管理页。 |
| `src/views/base/oauth-client/index.vue` | 开放授权客户端管理页。 |
| `src/views/tool/runtime-log/index.vue` | 运行日志控制台、历史文件和下载页。 |
| `src/views/tool/cache/index.vue` | 运行时缓存查询页，展示键、值、TTL 和时间元数据。 |

租户管理页创建租户时会自动初始化并启用 `admin` 管理员账号；有全局初始化密码时复用该密码，没有配置时通过创建响应一次性展示随机初始密码。

## 接入

宿主安装本包后，在模块清单中注册：

```ts
import { systemAdminModule } from "@liujitcn/kratos-admin-system";

export const adminModules = [systemAdminModule];
```

动态菜单组件路径必须包含模块前缀，例如：

| 页面     | `component`                     |
| -------- | ------------------------------- |
| 用户管理 | `system/base/user/index`        |
| 个人中心 | `system/profile/index`          |
| AI 助手  | `system/ai/chat/index`          |
| API 文档 | `system/tool/api-doc/index`     |

不再兼容 `base/user/index`、`profile/index` 等无模块前缀路径。不同业务模块可以拥有同名 `views`，core 会按 `<module>/<view>` 解析，不发生覆盖。

跨模块跳转使用 Vue Router；复用 System 代码时只引用 `package.json#exports` 公开的 npm 子路径。

租户项目管理组件通过 `TenantProjectManager` 公开。默认租户创建项目时在表单中选择目标租户，普通租户由服务端自动使用当前租户；默认租户查询全部项目，普通租户按项目授权范围查询。外部业务模块可以传入 `extraColumns`、`extraActions` 和 `loadExtraData`：扩展列可通过 `after` 指定插入到哪个基础字段后面，列的同名表格插槽会沿用该位置，未配置时默认插入备注后面，例如 `{ prop: "address", after: "name" }` 配合 `#address` 插槽即可把内容放到项目名称后面；数据加载器接收当前页的 `tenant_id + project_id` 集合，单条操作回调接收同一组双键以及项目基础数据。组件也会透传表格具名插槽，并向行插槽增加 `tenantProject` 上下文；保存业务配置后可通过组件实例的 `refresh()` 刷新列表。按外部业务字段搜索或排序时，应由外部接口参与分页查询，不能只在当前页加载后过滤。

外部项目可以在自己的页面中包装该组件，并通过菜单将项目管理页面的 `component` 指向包装页面：

```vue
<template>
  <TenantProjectManager
    ref="manager"
    :extra-columns="columns"
    :extra-actions="actions"
    :load-extra-data="loadExtraData"
  />
  <ProjectConfigDialog v-model="configVisible" :project="selectedProject" @saved="manager?.refresh()" />
</template>

<script setup lang="ts">
import { TenantProjectManager, type TenantProjectAction, type TenantProjectExtraDataLoader } from "@liujitcn/kratos-admin-system";
</script>
```

包装页面、业务 API 和弹窗仍属于外部模块；`apps/admin` 只在模块清单中加载该模块。

System 页面只通过 `systemAdminModule.views` 注册，不作为 npm 子路径公开。模块内组件默认不对外开放；`TenantProjectManager` 是显式公开的跨业务复用组件，其他组件仍保持内部使用。

登录、菜单和用户信息等运行底座属于 core；个人中心、AI 助手及其顶部入口属于 System。System 可以引用 core 的公共导出，core 不得反向依赖 System；其他业务模块只能通过 `AdminModule.staticViews` 显式替换默认登录页或状态页。

System 的 API 与 RPC 都按 Proto 完整层级维护，API 文件名与对应 Proto 服务文件名保持一致。当前服务端契约尚未完成细粒度拆分，因此 System 与 core 可能分别包含同一服务生成文件；这些文件由项目命令生成，不在本包手工去重或改写。

## 新增页面

- `system/base/dashboard/index`：后台工作台概览和趋势统计。
- `system/base/file/index`：文件资产元数据查询和删除。
- `system/base/i18n-custom/index`：按位置、语言和语言键维护管理端固定文案覆盖。

## AI 扩展

System 导出 `ADMIN_AI_EXTENSION`、`AdminAiExtension` 和 `getAdminAiExtension()`。其他业务模块可以在 `AdminModule.extensions` 中使用该扩展名提供 `flowBlocks` 组件，AI 会话页会读取并渲染它。

仓库当前没有内置流程卡片组件，也没有默认注册该扩展；未提供时 AI 会话仍按普通消息展示。扩展组件由实际业务模块自行实现，不属于 System 的内置能力。

## 命令

```bash
pnpm --filter @liujitcn/kratos-admin-system type:check
pnpm --filter @liujitcn/kratos-admin-system build:package
pnpm --filter @liujitcn/kratos-admin-system pack
```

### 扩展系统配置表单

系统配置类型选择“表单”后，按配置 Key 从 `runtimeConfigDefinitions` 查找定义，使用 `createModel()` 创建默认模型，再用后台 `value` 中的 JSON 对象覆盖默认字段，最终由 `ProForm` 渲染。切换 Key 会重新创建模型和表单控件；保存前执行字段校验，再将模型序列化为 JSON。未注册的 Key 显示不可用提示，不能提交表单值。

新增表单时，在业务模块中定义 `RuntimeConfigDefinition`，并在模块初始化时调用 `registerRuntimeConfig(definition)` 注册一次，无需修改系统配置页面。System 内置定义放在 `src/config` 并加入 `runtimeConfigDefinitions`。注册必须在打开配置页面之前完成，Key 必须唯一，并与后端消费配置时使用的 Key 一致。

```ts
import { registerRuntimeConfig } from "@liujitcn/kratos-admin-system/config";
import { notificationConfig } from "./config/notification";

registerRuntimeConfig(notificationConfig);
```

每个定义维护 `key`、`titleKey`、`descriptionKey`、`icon`、`createModel` 和 `fields`，可参考 `src/config/base-log-fallback.ts`。字段通过 `prop` 绑定模型，`component` 指定控件，`props` 传递参数，`options` 提供选项；单选用 `radio-group` 或 `select`，多选用 `checkbox-group` 或 `select` 配合 `props.multiple: true`，数组字段默认值应为 `[]`。服务器目录路径目前使用 `input`，保存的路径由后端解释。

标题、提示和校验分别通过 `labelKey`、`labelTooltipKey`、`rules.messageKey` 翻译，相关 Key 需加入模块语言包。选项文案通过 `options: () => [...]` 在函数中调用 `t()`，不要在模块初始化时固定翻译结果。新增定义需要构建发布前端；后端仍需实现对应配置的读取和业务处理。

字段国际化默认入口使用 Element Plus 标准按钮搭配 Languages 翻译图标（与顶部语言切换一致），保留悬停说明和无障碍名称；ProForm 单行输入框通过 el-input 的 append 插槽展示，多行及独立场景使用普通按钮，不使用自定义拼接样式。自定义 trigger 插槽（如列表文字入口）保持原有展示。

国际化编辑弹窗使用 Element Plus 横向表单，语言标签在左侧统一占 100px，输入框及翻译操作位于右侧；单行和多行编辑均保持左右布局。
代码生成进度弹窗统一补齐 HTTP 和 SSE 快照中被 ProtoJSON 省略的零值计数，任务及表级进度在生成初期显示 0，避免出现 NaN。

数据库迁移记录的模块、数据源、版本和时间仅在左侧列表展示，右侧直接展示文件 Tab。详情按文件展示：说明、升级和回退文件各占一个 Tab，标签显示原始文件名，面板显示路径；Markdown 独立渲染，SQL 独立查看和复制，内容区域填满 Tab 下方的剩余高度并在内部滚动。切换记录后选中第一个文件，仅渲染当前 Tab 的正文。
