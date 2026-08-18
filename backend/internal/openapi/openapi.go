package openapi

import (
	"embed"
	"io/fs"
)

// openAPIAssets 内嵌 OpenAPI 文件资源。
//
//go:embed assets
var openAPIAssets embed.FS

// Assets 返回 Admin 迁移脚本文件系统，交由 Core 统一注册和执行。
func Assets() fs.FS {
	value, err := fs.Sub(openAPIAssets, "assets")
	if err != nil {
		panic(err)
	}
	return value
}
