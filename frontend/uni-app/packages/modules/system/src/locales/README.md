# uni-app System 语言包

本目录提供 uni-app System 模块的个人中心、设置、AI 和相关业务文案。语言包由同级 `../index.ts` 导入并挂载到 `systemModule`，应用启动时由 uni-app Core 的 `registerLocaleMessages` 合并；System 页面和业务组件通过 `useI18n()` 或 `t()` 使用 `system.*` 键。

| 文件 | 用途 | 使用位置 |
| --- | --- | --- |
| `zh-CN.json` | System 模块简体中文默认语言包，也是三语键集合和占位符校验基准。 | uni-app 默认语言及缺失翻译回退。 |
| `en-US.json` | System 模块英文语言包。 | uni-app 切换为英文时使用。 |
| `ja-JP.json` | System 模块日文语言包。 | uni-app 切换为日文时使用。 |

语言键必须使用 `system.` 命名空间；语言包保持标准 JSON，不在文件内写注释。
