package i18n

import (
	"embed"
)

// LocalesData 提供宿主语言目录的内嵌文件系统。
//
//go:embed assets/*.json
var LocalesData embed.FS
