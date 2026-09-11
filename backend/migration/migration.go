package migration

import (
	"embed"
	"io/fs"
)

// ModuleName 是 Admin 迁移在 Core 资源注册表中的稳定模块名。
const ModuleName = "admin"

//go:embed assets/*
var baseMigrationFS embed.FS

// Assets 返回内置的 Admin 迁移文件系统。
func Assets() fs.FS {
	value, err := fs.Sub(baseMigrationFS, "assets")
	if err != nil {
		panic(err)
	}
	return value
}
