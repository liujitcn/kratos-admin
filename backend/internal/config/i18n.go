package config

import (
	"github.com/liujitcn/go-utils/translator"
	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	kitTranslator "github.com/liujitcn/kratos-kit/translator"
)

// NewDraftTranslator 创建只用于显式草稿操作的配置化翻译器。
func NewDraftTranslator(cfg *bootstrapConfigv1.Translator) (translator.Translator, error) {
	if cfg == nil || !cfg.GetEnabled() {
		return nil, nil
	}
	return kitTranslator.NewTranslatorWithError(cfg)
}
