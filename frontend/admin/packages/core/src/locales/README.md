# 管理端 Core 语言包

本目录提供管理端底座和公共组件使用的稳定文案，包括通用操作、字段、状态、登录认证、上传、校验和错误提示。语言包由 `../modules/kratosAdmin.ts` 导入，作为 `kratos-admin` 模块注册到 `packages/core/src/locales/index.ts`，最终由 `bootstrap.ts` 合并到 Vue I18n；core 内组件、hooks、请求错误处理和登录页通过 `t()` 或 `useLocaleStore()` 使用。

| 文件 | 用途 | 使用位置 |
| --- | --- | --- |
| `zh-CN.json` | 简体中文固定 UI 默认语言包，也是所有语言键集合和占位符校验基准。 | 管理端固定文案默认语言及缺失翻译回退。 |
| `zh-TW.json` | 繁体中文公共语言包。 | 管理端切换为繁体中文时由 Vue I18n 使用。 |
| `en-US.json` | 英文公共语言包。 | 管理端切换为英文时由 Vue I18n 使用。 |

语言包是标准 JSON，不能在文件内直接写注释；新增语言或修改键后执行仓库根目录的 `make i18n-sync`，脚本会校验所有语言并生成注册产物。语言切换选项固定显示后端 `base_language.native_name`，接口不可用时使用代码内置的原生名称；`common.language.*` 仅用于普通文案和语言迁移初始数据。新增语言的完整文件清单见[国际化语言扩展指南](../../../../../../docs/国际化语言扩展指南.md)。
