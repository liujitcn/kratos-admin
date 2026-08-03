# uni-app Core 语言包

本目录提供 uni-app 底座和公共页面使用的稳定文案，包括登录、启动状态、导航、认证、文件处理和网络错误提示。语言包由 `../module.ts` 的 `coreModule` 导入并注册，应用启动时由 `bootstrapKratosApp` 调用 `registerLocaleMessages` 合并；页面、hooks、请求工具通过 `useI18n()` 或 `t()` 使用。

| 文件 | 用途 | 使用位置 |
| --- | --- | --- |
| `zh-CN.json` | 简体中文固定 UI 默认语言包，也是所有语言键集合和占位符校验基准。 | uni-app 固定文案默认语言及缺失翻译回退。 |
| `zh-TW.json` | 繁体中文 Core 语言包。 | uni-app 切换为繁体中文时使用。 |
| `en-US.json` | 英文 Core 语言包。 | uni-app 切换为英文时使用。 |
| `ja-JP.json` | 日文 Core 语言包。 | uni-app 切换为日文时使用。 |
| `ko-KR.json` | 韩语 Core 语言包。 | uni-app 切换为韩语时使用。 |
| `fr-FR.json` | 法语 Core 语言包。 | uni-app 切换为法语时使用。 |
| `es-ES.json` | 西班牙语 Core 语言包。 | uni-app 切换为西班牙语时使用。 |

语言包是标准 JSON，不能在文件内直接写注释；公共键使用 `common.` 或 `core.` 命名空间。新增语言后执行仓库根目录的 `make i18n-sync`。语言切换选项固定显示后端 `base_language.native_name`，接口不可用时使用代码内置的原生名称；`common.language.*` 仅用于普通文案和语言迁移初始数据。新增语言的完整文件清单见[国际化语言扩展指南](../../../../../../docs/国际化语言扩展指南.md)。
