package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	config "github.com/liujitcn/kratos-admin/backend/internal/config"
	kitConfig "github.com/liujitcn/kratos-kit/config"
)

// main 执行错误目录检查或缺失草稿生成。
func main() {
	mode := flag.String("mode", "check", "执行模式：check 或 draft")
	write := flag.Bool("write", false, "写入缺失草稿；未设置时只报告且不联网")
	root := flag.String("root", ".", "backend 根目录")
	configPath := flag.String("config", "", "配置目录；默认使用 backend 根目录下的 configs")
	flag.Parse()
	if *configPath == "" {
		*configPath = filepath.Join(*root, "configs")
	}

	var err error
	switch *mode {
	case "check":
		var result *CatalogCheckResult
		result, err = CheckCatalogFiles(*root)
		if err == nil {
			fmt.Printf("国际化目录检查通过：Proto消息 %d 条，所有语言目录各 %d 条\n", result.SourceCount, result.LocaleCount)
		}
	case "draft":
		var result *DraftResult
		if !*write {
			result, err = DraftCatalogFiles(context.Background(), *root, nil, false)
		} else {
			err = kitConfig.LoadBootstrapConfig(*configPath)
			if err != nil {
				break
			}
			translationConfig := kitConfig.GetBootstrapConfig().GetTranslator()
			var providerErr error
			provider, providerErr := config.NewDraftTranslator(translationConfig)
			if providerErr != nil {
				err = providerErr
				break
			}
			result, err = DraftCatalogFiles(context.Background(), *root, provider, true)
		}
		if err == nil {
			fmt.Printf("缺失草稿：%v，写入=%d\n", result.MissingByLocale, result.Written)
		}
	default:
		err = fmt.Errorf("不支持的执行模式 %s", *mode)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
