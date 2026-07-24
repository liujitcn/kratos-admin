package assets

import _ "embed"

// OpenAPIData 内嵌 OpenAPI 文档数据。
//
//go:embed openapi.yaml
var OpenAPIData []byte
