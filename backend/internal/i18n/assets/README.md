# 后端错误国际化目录

本目录存放后端返回给客户端的结构化错误消息。`embed.go` 只负责把本项目语言文件提供给宿主，通用目录加载和错误本地化由 `backend/core/pkg/i18n` 提供；HTTP 语言中间件在返回错误前调用这条链路。

| 文件 | 用途 | 使用位置 |
| --- | --- | --- |
| `zh-CN.json` | 中文基准目录；Proto 校验消息和 Go 显式消息键必须先在这里定义。 | `CheckCatalogFiles` 校验源文与消息键，未找到目标语言时作为默认回退。 |
| `zh-TW.json` | 繁体中文错误消息目录。 | `Catalog.Localize` 按 `Accept-Language` 请求头选择。 |
| `en-US.json` | 英文错误消息目录，与中文保持完全相同的消息键和占位符。 | `Catalog.Localize` 按 `Accept-Language` 请求头选择。 |

这些 JSON 文件必须保持标准 JSON，不能写 `//` 或 `/* ... */` 注释。修改后从仓库根目录执行 `make i18n-sync` 和 `make -C backend i18n-check`，检查所有语言的键集合和占位符。

新增语言的完整文件清单见[国际化语言扩展指南](../../../../docs/国际化语言扩展指南.md)。
