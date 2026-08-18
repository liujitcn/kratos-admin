package migration

import (
	"embed"
	"io/fs"
)

// ModuleName 是 Admin 迁移在 Core 资源注册表中的稳定模块名。
const ModuleName = "admin"

//go:embed assets/*
var baseMigrationFS embed.FS

// Assets 返回 Admin 迁移脚本文件系统，交由 Core 统一注册和执行。
func Assets() fs.FS {
	value, err := fs.Sub(baseMigrationFS, "assets")
	if err != nil {
		panic(err)
	}
	return value
}
