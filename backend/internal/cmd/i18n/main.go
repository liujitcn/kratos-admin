// Package main 提供后端国际化目录检查和草稿补齐命令。
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	backendI18n "github.com/liujitcn/kratos-admin/backend/internal/i18n"

	googleTranslator "github.com/liujitcn/go-utils/translator/google"
)

// main 执行错误目录检查或缺失草稿生成。
func main() {
	mode := flag.String("mode", "check", "执行模式：check 或 draft")
	write := flag.Bool("write", false, "写入缺失草稿；未设置时只报告且不联网")
	root := flag.String("root", ".", "backend 根目录")
	flag.Parse()

	var err error
	switch *mode {
	case "check":
		var result *backendI18n.CatalogCheckResult
		result, err = backendI18n.CheckCatalogFiles(*root)
		if err == nil {
			fmt.Printf("国际化目录检查通过：Proto消息 %d 条，所有语言目录各 %d 条\n", result.SourceCount, result.LocaleCount)
		}
	case "draft":
		var result *backendI18n.DraftResult
		if !*write {
			result, err = backendI18n.DraftCatalogFiles(context.Background(), *root, nil, false)
		} else {
			provider := googleTranslator.NewTranslator(
				googleTranslator.WithVersion("v1"),
				googleTranslator.WithHTTPClient(&http.Client{Timeout: 8 * time.Second}),
			)
			result, err = backendI18n.DraftCatalogFiles(context.Background(), *root, provider, true)
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
