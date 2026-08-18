package i18n

import (
	"embed"
	"io/fs"
)

// LocalesData 提供宿主语言目录的内嵌文件系统。
//
//go:embed assets/*.json
var localesData embed.FS

// Assets 返回 Admin 后端语言文件系统，交由 Core 统一注册和执行。
func Assets() fs.FS {
	value, err := fs.Sub(localesData, "assets")
	if err != nil {
		panic(err)
	}
	return value
}
