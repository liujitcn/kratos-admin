package middleware

import (
	"encoding/json"
)

// MarshalToolError 将工具错误转换成稳定 JSON 文本。
func MarshalToolError(message string) string {
	raw, err := json.Marshal(map[string]string{"error": message})
	// 理论上 map[string]string 不会序列化失败；保留兜底是为了稳定工具协议。
	if err != nil {
		return `{"error":"tool execution failed"}`
	}
	return string(raw)
}
