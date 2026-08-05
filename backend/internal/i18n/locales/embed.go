package locales

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"

	coreLocale "github.com/liujitcn/kratos-admin/backend/core/pkg/locale"
)

// Files 提供宿主语言目录的内嵌文件系统。
//
//go:embed *.json
var Files embed.FS

type manifestConfig struct {
	Default string   `json:"default"`
	Locales []string `json:"locales"`
}

func init() {
	data, err := fs.ReadFile(Files, "manifest.json")
	if err != nil {
		panic(fmt.Errorf("读取语言 manifest: %w", err))
	}
	var config manifestConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		panic(fmt.Errorf("解析语言 manifest: %w", err))
	}
	err = coreLocale.Configure(coreLocale.Config{Default: config.Default, Supported: config.Locales})
	if err != nil {
		panic(err)
	}
}
