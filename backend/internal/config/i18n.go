package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	_const "github.com/liujitcn/kratos-admin/backend/internal/const"

	"github.com/liujitcn/go-utils/translator"
	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	kitTranslator "github.com/liujitcn/kratos-kit/translator"
	"google.golang.org/protobuf/proto"
)

const defaultTranslationTimeout = 8 * time.Second

// TranslationDraftConfig 描述机器翻译草稿能力的部署配置。
type TranslationDraftConfig struct {
	Enabled  bool
	Provider string
	Timeout  time.Duration
}

// NewTranslationDraftConfig 从引导配置、应用元数据或环境变量解析翻译草稿配置。
func NewTranslationDraftConfig(cfg *bootstrapConfigv1.Translator, appInfo *bootstrapConfigv1.AppInfo) TranslationDraftConfig {
	provider := cfg.GetType()
	if provider == "" {
		provider = kitTranslator.Google
	}
	provider = strings.ToLower(provider)
	timeout := cfg.GetTimeout().AsDuration()
	if timeout <= 0 {
		timeout = defaultTranslationTimeout
	}

	value := ""
	if appInfo != nil {
		value = appInfo.GetMetadata()[_const.I18N_TRANSLATION_DRAFT_CONFIG_KEY]
	}
	if value == "" {
		value = os.Getenv(_const.I18N_TRANSLATION_DRAFT_ENV_KEY)
	}
	enabled := cfg.GetEnabled()
	if value != "" {
		parsed, err := strconv.ParseBool(value)
		if err == nil {
			enabled = parsed
		}
	}
	return TranslationDraftConfig{Enabled: enabled, Provider: provider, Timeout: timeout}
}

// NewDraftTranslator 创建只用于显式草稿操作的配置化翻译器。
func NewDraftTranslator(cfg *bootstrapConfigv1.Translator, draftConfig TranslationDraftConfig) (translator.Translator, error) {
	if !draftConfig.Enabled {
		return nil, nil
	}
	if cfg.GetEnabled() {
		return kitTranslator.NewTranslatorWithError(cfg)
	}
	cloneMessage := proto.Clone(cfg)
	clone, ok := cloneMessage.(*bootstrapConfigv1.Translator)
	if !ok {
		return nil, fmt.Errorf("翻译配置类型无效")
	}
	clone.Enabled = true
	return kitTranslator.NewTranslatorWithError(clone)
}
