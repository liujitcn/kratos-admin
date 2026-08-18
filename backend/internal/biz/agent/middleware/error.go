package middleware

import (
	"encoding/json"
	"fmt"
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

// DisabledToolMessage 返回 Agent 工具禁用提示。
func DisabledToolMessage(name string) string {
	// 无工具名时返回泛化文案，避免把空名称展示给用户。
	if name == "" {
		return "该 Agent 工具已被禁用，无法继续调用。"
	}
	return fmt.Sprintf("Agent 工具 %s 已被禁用，无法继续调用。", name)
}
