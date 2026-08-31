package ai

import (
	"encoding/json"
	"fmt"
	"strings"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
)

// InputContentPayload 表示 AI 助手用户输入内容 JSON 结构。
type InputContentPayload struct {
	// Kind 输入内容类型。
	Kind string `json:"kind"`
	// Content 输入正文。
	Content string `json:"content"`
}

// OutputContentPayload 表示 AI 助手输出内容 JSON 结构。
type OutputContentPayload struct {
	// Kind 输出内容类型。
	Kind string `json:"kind"`
	// Content 输出正文。
	Content string `json:"content"`
	// ReplySource 回复来源。
	ReplySource string `json:"reply_source"`
	// Model 使用的模型名称。
	Model string `json:"model"`
	// Fallback 表示是否使用降级回复。
	Fallback bool `json:"fallback"`
	// FallbackReason 记录降级原因。
	FallbackReason string `json:"fallback_reason"`
	// Flow 固定流程标识。
	Flow string `json:"flow"`
	// Step 固定流程步骤。
	Step string `json:"step"`
	// BlocksJSON 结构化内容 JSON。
	BlocksJSON string `json:"blocks_json"`
}

// BuildUserContent 生成用户消息落库正文。
func BuildUserContent(content string, attachments []*basev1.AiAttachment) string {
	// 有用户文本时保留文本作为主问题，附件内容通过附件字段独立保存。
	if content != "" {
		return content
	}
	// 文本和附件都为空时交给上层参数校验处理。
	if len(attachments) == 0 {
		return ""
	}
	return "请结合附件内容继续分析"
}

// MarshalInputContentPayload 序列化 AI 助手输入内容。
func MarshalInputContentPayload(payload InputContentPayload) string {
	if payload.Kind == "" {
		payload.Kind = KindText
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return `{"kind":"text","content":""}`
	}
	return string(raw)
}

// ParseInputContent 解析 AI 助手输入内容。
func ParseInputContent(raw string) InputContentPayload {
	payload := InputContentPayload{Kind: KindText}
	if raw == "" {
		return payload
	}
	err := json.Unmarshal([]byte(raw), &payload)
	if err != nil {
		payload.Content = raw
		return payload
	}
	if payload.Kind == "" {
		payload.Kind = KindText
	}
	return payload
}

// MarshalInputContent 序列化用户输入内容。
func MarshalInputContent(content string, attachments []*basev1.AiAttachment) string {
	return MarshalInputContentPayload(InputContentPayload{
		Kind:    KindText,
		Content: BuildUserContent(content, attachments),
	})
}

// MarshalOutputContentPayload 序列化 AI 助手输出内容。
func MarshalOutputContentPayload(payload OutputContentPayload) string {
	if payload.Kind == "" {
		payload.Kind = KindText
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return `{"kind":"text","content":""}`
	}
	return string(raw)
}

// ParseOutputContent 解析 AI 助手输出内容。
func ParseOutputContent(raw string) OutputContentPayload {
	payload := OutputContentPayload{Kind: KindText}
	if raw == "" {
		return payload
	}
	err := json.Unmarshal([]byte(raw), &payload)
	if err != nil {
		payload.Content = raw
		return payload
	}
	if payload.Kind == "" {
		payload.Kind = KindText
	}
	return payload
}

// BuildFallbackReply 生成模型不可用时的本地降级回复。
func BuildFallbackReply(content string, attachments []*basev1.AiAttachment) string {
	// 附件场景不回显文件名，避免降级文案过长或暴露不必要的文件路径。
	if len(attachments) > 0 {
		return fmt.Sprintf("已收到你的问题和 %d 个附件，但当前大模型暂时不可用，无法生成完整回复，请稍后再试。", len(attachments))
	}
	return fmt.Sprintf("已收到你的问题：%s。但当前大模型暂时不可用，无法生成完整回复，请稍后再试。", NormalizePreview(content))
}

// BuildDefaultSummary 生成新会话默认摘要。
func BuildDefaultSummary() string {
	return "新对话"
}

// BuildDynamicSummary 根据本轮用户文本或附件数量生成会话摘要。
func BuildDynamicSummary(content string, attachments []*basev1.AiAttachment) string {
	preview := NormalizePreview(content)
	// 只有附件没有文本时，用附件数量表达本轮会话主题。
	if preview == "" && len(attachments) > 0 {
		preview = fmt.Sprintf("%d 个附件", len(attachments))
	}
	// 兜底默认摘要，避免会话列表出现空白标题区域。
	if preview == "" {
		return BuildDefaultSummary()
	}
	return preview
}

// NormalizePreview 将用户输入整理为适合会话列表展示的短文本。
func NormalizePreview(content string) string {
	trimmed := strings.ReplaceAll(content, "\n", " ")
	// 空输入没有可展示摘要，交给调用方决定默认值。
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	// 按 rune 截断，避免中文内容被字节截断成乱码。
	if len(runes) <= previewSize {
		return trimmed
	}
	return string(runes[:previewSize]) + "..."
}

// MarshalReplyContent 序列化助手回复正文和元信息。
func MarshalReplyContent(response *Response) string {
	// nil 回复通常表示上游未生成内容，直接落空字符串避免 panic。
	if response == nil {
		return ""
	}
	payload := OutputContentPayload{
		Kind:           KindText,
		Content:        response.Content,
		ReplySource:    response.Source,
		Model:          response.Model,
		Fallback:       response.Fallback,
		FallbackReason: response.FallbackReason,
		Flow:           response.Flow,
		Step:           response.Step,
		BlocksJSON:     response.BlocksJSON,
	}
	raw, err := json.Marshal(payload)
	// 极端情况下 JSON 序列化失败时保留正文，避免用户完全看不到回复。
	if err != nil {
		return response.Content
	}
	return string(raw)
}

// MarshalEmptyOutputContent 序列化空助手输出内容。
func MarshalEmptyOutputContent() string {
	return MarshalOutputContentPayload(OutputContentPayload{Kind: KindText})
}

// MarshalTools 序列化 AI 助手工具使用记录。
func MarshalTools(tools []ToolUsage) string {
	if len(tools) == 0 {
		return "[]"
	}
	raw, err := json.Marshal(tools)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

// ParseTools 解析 AI 助手工具使用记录。
func ParseTools(raw string) []ToolUsage {
	if raw == "" {
		return []ToolUsage{}
	}
	var tools []ToolUsage
	err := json.Unmarshal([]byte(raw), &tools)
	if err != nil {
		return []ToolUsage{}
	}
	result := make([]ToolUsage, 0, len(tools))
	for _, item := range tools {
		if item.Name == "" {
			continue
		}
		result = append(result, item)
	}
	return result
}

// MarshalTokenUsage 序列化 AI 助手 token 统计。
func MarshalTokenUsage(token TokenUsage) string {
	raw, err := json.Marshal(token)
	if err != nil {
		return `{"input":0,"output":0,"cache":0,"total":0}`
	}
	return string(raw)
}

// ParseTokenUsage 解析 AI 助手 token 统计。
func ParseTokenUsage(raw string) TokenUsage {
	if raw == "" {
		return TokenUsage{}
	}
	token := TokenUsage{}
	err := json.Unmarshal([]byte(raw), &token)
	if err != nil {
		return TokenUsage{}
	}
	return token
}
