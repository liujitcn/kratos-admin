package server

import _ "embed"

// OpenAPIData 内嵌 OpenAPI 文档数据。
//
//go:embed assets/openapi.yaml
var OpenAPIData []byte
