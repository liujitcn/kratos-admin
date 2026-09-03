# 管理端 System 语言包

本目录提供 System 管理业务模块的页面、AI、个人中心安全设置、开放授权客户端、基础资料和代码生成等业务文案。语言包由同级 `../module.ts` 导入，随 `systemAdminModule` 注册；管理端启动时由 core 的 `registerLocaleMessages` 合并，System 页面和组件通过 `t()` 或 `useLocaleStore()` 使用 `system.*` 键。

| 文件 | 用途 | 使用位置 |
| --- | --- | --- |
| `zh-CN.json` | System 模块简体中文固定 UI 默认语言包，也是所有语言键集合和占位符校验基准。 | 管理端固定文案默认语言及缺失翻译回退。 |
| `zh-TW.json` | System 模块繁体中文语言包。 | 管理端切换为繁体中文时由 Vue I18n 使用。 |
| `en-US.json` | System 模块英文语言包。 | 管理端切换为英文时由 Vue I18n 使用。 |
| `ja-JP.json` | System 模块日文语言包。 | 管理端切换为日文时由 Vue I18n 使用。 |

## 命名空间与使用位置

| 命名空间 | 文案范围 | 主要使用位置 |
| --- | --- | --- |
| `common.*` | 跨页面复用的操作、字段、弹窗、状态和校验文案。 | `src/views/**` 下的列表页、表单和确认弹窗。 |
| `system.ai.*` | AI 会话、消息流、重试和错误提示。 | `src/views/ai/**`、`src/api/base/v1/ai_message.ts`。 |
| `system.profile.*` | 个人资料、安全设置和用户中心导航。 | `src/views/profile/**`、`src/module.ts` 的用户菜单。 |
| `system.base.*` | 区域、配置、部门、字典、岗位、角色、租户、用户等基础管理页面。 | `src/views/base/**`。 |
| `system.base.oauth_client.*` | 开放授权客户端字段、凭据和 API 选择。 | `src/views/base/oauth-client/**`。 |
| `system.code.gen.*` | 代码生成配置、预览、列配置和多语言编辑。 | `src/views/tool/code-gen/**`。 |
| `system.cache.*` | 运行时缓存查询、分页结果和条目详情。 | `src/views/tool/cache/**`。 |
| `system.i18n.*` | 动态翻译编辑、草稿生成和语言包状态。 | `src/components/DynamicI18nEditor.vue`、代码生成语言编辑组件。 |

语言包由生成的 `generated.ts` 传入 `systemAdminModule.messages`，由 core 的 `registerLocaleMessages` 校验语言键集合、命名空间和占位符后注册到 Vue I18n；业务代码统一使用 `t('system....')` 或 `useLocaleStore()` 读取。

语言键按文案归属使用 `common.` 或 `system.` 命名空间；语言包保持标准 JSON，不在文件内写注释。

新增语言的完整文件清单和迁移流程见[国际化语言扩展指南](../../../../../../../docs/国际化语言扩展指南.md)。
