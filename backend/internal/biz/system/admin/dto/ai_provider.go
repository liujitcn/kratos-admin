package dto

// AiProviderModelConfig 表示AI供应商JSON数组中的单个模型配置。
type AiProviderModelConfig struct {
	// ModelName 是供应商提供的模型名称。
	ModelName string `json:"model_name"`
	// APIType 是模型接口协议。
	APIType string `json:"api_type"`
	// Temperature 是生成温度。
	Temperature float32 `json:"temperature"`
	// MaxTokens 是最大生成Token数。
	MaxTokens int32 `json:"max_tokens"`
	// TimeoutSeconds 是模型请求超时秒数。
	TimeoutSeconds int32 `json:"timeout_seconds"`
	// MaxRetries 是请求最大重试次数。
	MaxRetries int32 `json:"max_retries"`
}
