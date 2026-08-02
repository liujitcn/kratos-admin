# Taro Core 语言包

本目录提供 Taro 应用底座和公共页面使用的稳定文案，包括登录、启动状态、导航、认证、文件处理和网络错误提示。语言包由 `../index.ts` 的 `coreModule` 导入并注册，应用启动时由 `bootstrapKratosTaroApp` 调用 `registerLocaleMessages` 合并；页面、hooks、请求工具通过 `useI18n()` 或 `t()` 使用。

| 文件 | 用途 | 使用位置 |
| --- | --- | --- |
| `zh-CN.json` | 简体中文默认语言包，也是键集合和占位符校验基准。 | Taro 默认语言及缺失翻译回退。 |
| `en-US.json` | 英文 Core 语言包。 | Taro 切换为英文时使用。 |
| `ja-JP.json` | 日文 Core 语言包。 | Taro 切换为日文时使用。 |

语言包是标准 JSON，不能在文件内直接写注释；公共键使用 `common.` 或 `core.` 命名空间。
