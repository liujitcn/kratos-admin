package ai

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// embeddingDimensionOptions 是未声明维度的embedding模型提供的常用维度集合。
var embeddingDimensionOptions = []int32{512, 768, 1024, 1536, 2048, 3072}

// EmbeddingDimensionOptions 返回embedding模型可选的向量维度列表：
// 模型声明了维度时唯一可选，未声明时提供常用维度集合。
func EmbeddingDimensionOptions(declared int32) []int32 {
	if declared > 0 {
		return []int32{declared}
	}
	return slices.Clone(embeddingDimensionOptions)
}

// ResolveEmbeddingDimensions 解析知识库应使用的向量维度：
// 表单未传（0）时按模型声明维度或系统默认；显式传入的值必须在模型可选范围内。
func ResolveEmbeddingDimensions(declared int32, requested int32) (int32, error) {
	if requested == 0 {
		if declared > 0 {
			return declared, nil
		}
		return DefaultEmbeddingDimensions, nil
	}
	for _, option := range EmbeddingDimensionOptions(declared) {
		if option == requested {
			return requested, nil
		}
	}
	return 0, fmt.Errorf("向量维度 %d 不在当前模型可选范围内", requested)
}

// EmbeddingModelConfig 表示embedding模型分类的个性化配置。
type EmbeddingModelConfig struct {
	// Dimensions 是embedding输出向量维度，0表示未声明。
	Dimensions int32 `json:"dimensions"`
	// TimeoutSeconds 是单次请求超时秒数，0表示使用默认值。
	TimeoutSeconds int32 `json:"timeout_seconds"`
	// MaxRetries 是请求最大重试次数，0表示使用默认值。
	MaxRetries int32 `json:"max_retries"`
}

// ParseEmbeddingModelConfig 解析embedding模型的个性化配置JSON对象。
func ParseEmbeddingModelConfig(raw string) (*EmbeddingModelConfig, error) {
	if strings.TrimSpace(raw) == "" {
		return &EmbeddingModelConfig{}, nil
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	var value EmbeddingModelConfig
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("embedding模型配置必须是JSON对象: %w", err)
	}
	return &value, nil
}

// TranslationModelConfig 表示翻译模型分类的个性化配置。
type TranslationModelConfig struct {
	// TimeoutSeconds 是单次请求超时秒数，0表示使用默认值。
	TimeoutSeconds int32 `json:"timeout_seconds"`
	// MaxRetries 是请求最大重试次数，0表示使用默认值。
	MaxRetries int32 `json:"max_retries"`
}

// ParseTranslationModelConfig 解析翻译模型的个性化配置JSON对象。
func ParseTranslationModelConfig(raw string) (*TranslationModelConfig, error) {
	if strings.TrimSpace(raw) == "" {
		return &TranslationModelConfig{}, nil
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	var value TranslationModelConfig
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("翻译模型配置必须是JSON对象: %w", err)
	}
	return &value, nil
}
