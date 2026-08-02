# 管理端 Core 语言包

本目录提供管理端底座和公共组件使用的稳定文案，包括通用操作、字段、状态、登录认证、上传、校验和错误提示。语言包由 `../modules/kratosAdmin.ts` 导入，作为 `kratos-admin` 模块注册到 `packages/core/src/locales/index.ts`，最终由 `bootstrap.ts` 合并到 Vue I18n；core 内组件、hooks、请求错误处理和登录页通过 `t()` 或 `useLocaleStore()` 使用。

| 文件 | 用途 | 使用位置 |
| --- | --- | --- |
| `zh-CN.json` | 简体中文固定 UI 默认语言包，也是七语键集合和占位符校验基准。 | 管理端固定文案默认语言及缺失翻译回退。 |
| `zh-TW.json` | 繁体中文公共语言包。 | 管理端切换为繁体中文时由 Vue I18n 使用。 |
| `en-US.json` | 英文公共语言包。 | 管理端切换为英文时由 Vue I18n 使用。 |
| `ja-JP.json` | 日文公共语言包。 | 管理端切换为日文时由 Vue I18n 使用。 |
| `ko-KR.json` | 韩语公共语言包。 | 管理端切换为韩语时由 Vue I18n 使用。 |
| `fr-FR.json` | 法语公共语言包。 | 管理端切换为法语时由 Vue I18n 使用。 |
| `es-ES.json` | 西班牙语公共语言包。 | 管理端切换为西班牙语时由 Vue I18n 使用。 |

语言包是标准 JSON，不能在文件内直接写注释；新增或修改键时七种语言必须同步更新。
