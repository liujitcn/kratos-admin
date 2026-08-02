# 后端错误国际化目录

本目录存放后端返回给客户端的结构化错误消息。文件由 `backend/internal/i18n/catalog.go` 嵌入并加载，`backend/internal/i18n/error.go` 的 `LocalizeError` 会根据请求语言区域渲染消息；HTTP 语言中间件在返回错误前调用这条链路。

| 文件 | 用途 | 使用位置 |
| --- | --- | --- |
| `zh-CN.json` | 中文基准目录；Proto 校验消息和 Go 显式消息键必须先在这里定义。 | `CheckCatalogFiles` 校验源文与消息键，未找到目标语言时作为默认回退。 |
| `zh-TW.json` | 繁体中文错误消息目录。 | `Catalog.Localize` 按 `Accept-Language` 请求头选择。 |
| `en-US.json` | 英文错误消息目录，与中文保持完全相同的消息键和占位符。 | `Catalog.Localize` 按 `Accept-Language` 请求头选择。 |
| `ja-JP.json` | 日文错误消息目录，与中文保持完全相同的消息键和占位符。 | `Catalog.Localize` 按 `Accept-Language` 请求头选择。 |
| `ko-KR.json` | 韩语错误消息目录。 | `Catalog.Localize` 按 `Accept-Language` 请求头选择。 |
| `fr-FR.json` | 法语错误消息目录。 | `Catalog.Localize` 按 `Accept-Language` 请求头选择。 |
| `es-ES.json` | 西班牙语错误消息目录。 | `Catalog.Localize` 按 `Accept-Language` 请求头选择。 |

这些文件必须保持标准 JSON，不能写 `//` 或 `/* ... */` 注释。修改后在 `backend` 目录执行 `go run ./internal/cmd/i18n -mode check -root .` 检查七语键集合和占位符。
