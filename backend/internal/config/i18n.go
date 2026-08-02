package config

import (
	"net/http"
	"os"
	"strconv"
	"time"

	_const "github.com/liujitcn/kratos-admin/backend/internal/const"

	"github.com/liujitcn/go-utils/translator"
	googleTranslator "github.com/liujitcn/go-utils/translator/google"
	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
)

// TranslationDraftConfig 描述机器翻译草稿能力的部署开关。
type TranslationDraftConfig struct {
	Enabled bool
}

// NewTranslationDraftConfig 从应用元数据或环境变量解析翻译草稿开关。
func NewTranslationDraftConfig(appInfo *bootstrapConfigv1.AppInfo) TranslationDraftConfig {
	value := ""
	if appInfo != nil {
		value = appInfo.GetMetadata()[_const.I18N_TRANSLATION_DRAFT_CONFIG_KEY]
	}
	if value == "" {
		value = os.Getenv(_const.I18N_TRANSLATION_DRAFT_ENV_KEY)
	}
	enabled, err := strconv.ParseBool(value)
	if err != nil {
		enabled = false
	}
	return TranslationDraftConfig{Enabled: enabled}
}

// NewDraftTranslator 创建只用于显式草稿操作的 Google V1 翻译器。
func NewDraftTranslator() translator.Translator {
	httpClient := &http.Client{Timeout: 8 * time.Second}
	return googleTranslator.NewTranslator(
		googleTranslator.WithVersion("v1"),
		googleTranslator.WithHTTPClient(httpClient),
	)
}
