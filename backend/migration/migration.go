package migration

import (
	"io/fs"
	"os"
)

// ModuleName 是 Admin 迁移在 Core 资源注册表中的稳定模块名。
const ModuleName = "admin"

// Assets 返回部署目录中的迁移文件系统，可通过 ADMIN_MIGRATION_DIR 指定绝对路径。
func Assets() fs.FS {
	directory := os.Getenv("ADMIN_MIGRATION_DIR")
	if directory == "" {
		directory = "migration/assets"
	}
	return os.DirFS(directory)
}
