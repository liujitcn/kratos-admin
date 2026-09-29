package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

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
