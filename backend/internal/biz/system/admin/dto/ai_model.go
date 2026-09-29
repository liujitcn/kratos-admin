package dto

// AiModelChatConfig 表示聊天模型分类的个性化配置。
type AiModelChatConfig struct {
	// APIType 是模型接口协议。
	APIType string `json:"api_type"`
	// Temperature 是生成温度。
	Temperature float32 `json:"temperature"`
	// MaxTokens 是单次请求最大输出Token数。
	MaxTokens int32 `json:"max_tokens"`
	// TimeoutSeconds 是模型请求超时秒数。
	TimeoutSeconds int32 `json:"timeout_seconds"`
	// MaxRetries 是请求最大重试次数。
	MaxRetries int32 `json:"max_retries"`
}
