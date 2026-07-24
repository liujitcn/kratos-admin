package message

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"
)

// AgenticMessage 表示 Eino Agentic 消息。
type AgenticMessage = schema.AgenticMessage

// ContentBlock 表示 Eino 多模态消息片段。
type ContentBlock = schema.ContentBlock

// StreamReader 表示 Eino 流式消息读取器。
type StreamReader = schema.StreamReader[*schema.AgenticMessage]

// TokenUsage 表示模型原始 token 统计。
type TokenUsage = schema.TokenUsage

// ToolCall 表示 Eino 函数工具调用。
type ToolCall = schema.ToolCall

// FunctionCall 表示 Eino 函数工具调用参数。
type FunctionCall = schema.FunctionCall

// ImageData 表示图片字节输入。
type ImageData struct {
	// Bytes 图片原始字节。
	Bytes []byte
	// MIMEType 图片 MIME 类型。
	MIMEType string
}

// SystemText 创建系统文本消息。
func SystemText(content string) *AgenticMessage {
	return schema.SystemAgenticMessage(content)
}

// UserText 创建用户文本消息。
func UserText(content string) *AgenticMessage {
	return schema.UserAgenticMessage(content)
}

// AIText 创建助手文本消息。
func AIText(content string) *AgenticMessage {
	return &schema.AgenticMessage{
		Role: schema.AgenticRoleTypeAssistant,
		ContentBlocks: []*schema.ContentBlock{
			schema.NewContentBlock(&schema.AssistantGenText{Text: content}),
		},
	}
}

// TextByRole 按对话角色创建文本消息。
func TextByRole(role string, content string) *AgenticMessage {
	// 历史消息角色需要还原到模型角色，未知角色按用户消息兼容旧数据。
	switch strings.ToLower(role) {
	case "ai":
		return AIText(content)
	case "system":
		return SystemText(content)
	default:
		return UserText(content)
	}
}

// UserParts 创建带多模态片段的用户消息。
func UserParts(parts []*ContentBlock) *AgenticMessage {
	return &schema.AgenticMessage{
		Role:          schema.AgenticRoleTypeUser,
		ContentBlocks: parts,
	}
}

// UserTextWithImages 创建包含正文、补充文本和图片的用户消息。
func UserTextWithImages(content string, textSections []string, images []ImageData) *AgenticMessage {
	textParts := []string{content}
	textParts = append(textParts, textSections...)
	blocks := make([]*ContentBlock, 0, 1+len(images))
	// 用户可能只上传图片不输入文本，空文本不传入 message parts。
	if text := strings.Join(textParts, "\n\n"); text != "" {
		blocks = append(blocks, TextPart(text))
	}
	for _, image := range images {
		// 没有原始字节的图片无法作为视觉输入，跳过避免构造无效多模态块。
		if len(image.Bytes) == 0 {
			continue
		}
		blocks = append(blocks, ImageDataPart(image.Bytes, image.MIMEType))
	}
	return UserParts(blocks)
}

// TextPart 构造文本输入片段。
func TextPart(text string) *ContentBlock {
	return schema.NewContentBlock(&schema.UserInputText{Text: text})
}

// ImageURLPart 构造远程图片输入片段。
func ImageURLPart(rawURL string) *ContentBlock {
	return schema.NewContentBlock(&schema.UserInputImage{
		URL:    rawURL,
		Detail: schema.ImageURLDetailAuto,
	})
}

// ImageDataPart 构造图片字节输入片段。
func ImageDataPart(data []byte, mimeType string) *ContentBlock {
	return schema.NewContentBlock(&schema.UserInputImage{
		Base64Data: base64.StdEncoding.EncodeToString(data),
		MIMEType:   mimeType,
		Detail:     schema.ImageURLDetailAuto,
	})
}

// FunctionToolResultMessage 构造函数工具执行结果消息。
func FunctionToolResultMessage(callID string, name string, content string) *AgenticMessage {
	return &schema.AgenticMessage{
		Role: schema.AgenticRoleTypeUser,
		ContentBlocks: []*schema.ContentBlock{
			schema.NewContentBlock(&schema.FunctionToolResult{
				CallID: callID,
				Name:   name,
				Content: []*schema.FunctionToolResultContentBlock{
					{
						Type: schema.FunctionToolResultContentBlockTypeText,
						Text: &schema.UserInputText{Text: content},
					},
				},
			}),
		},
	}
}

// Concat 合并流式 Agentic 消息片段，兼容服务端工具块与文本块混合返回。
func Concat(chunks []*AgenticMessage) (*AgenticMessage, error) {
	result, err := schema.ConcatAgenticMessages(chunks)
	if err == nil || !hasMixedStreamingBlocks(chunks) {
		return result, err
	}
	return concatMixedStreamingBlocks(chunks)
}

// hasMixedStreamingBlocks 判断消息流是否同时携带普通块和流式块。
func hasMixedStreamingBlocks(chunks []*AgenticMessage) bool {
	var hasStreaming bool
	var hasNonStreaming bool
	for _, chunk := range chunks {
		if chunk == nil {
			continue
		}
		for _, block := range chunk.ContentBlocks {
			if block == nil {
				continue
			}
			if block.StreamingMeta == nil {
				hasNonStreaming = true
			} else {
				hasStreaming = true
			}
		}
	}
	return hasStreaming && hasNonStreaming
}

// concatMixedStreamingBlocks 分别合并流式块并保留服务端返回的普通工具块。
func concatMixedStreamingBlocks(chunks []*AgenticMessage) (*AgenticMessage, error) {
	streamingChunks := make([]*AgenticMessage, 0, len(chunks))
	nonStreamingBlocks := make([]*ContentBlock, 0)
	var role schema.AgenticRoleType
	for index, chunk := range chunks {
		if chunk == nil {
			return nil, fmt.Errorf("message at index %d is nil", index)
		}
		if chunk.Role != "" {
			if role == "" {
				role = chunk.Role
			} else if role != chunk.Role {
				return nil, fmt.Errorf("cannot concat messages with different roles: got '%s' and '%s'", role, chunk.Role)
			}
		}

		streamingChunk := &AgenticMessage{
			Role:         chunk.Role,
			ResponseMeta: chunk.ResponseMeta,
			Extra:        chunk.Extra,
		}
		for _, block := range chunk.ContentBlocks {
			if block == nil {
				continue
			}
			if block.StreamingMeta == nil {
				nonStreamingBlocks = append(nonStreamingBlocks, block)
				continue
			}
			streamingChunk.ContentBlocks = append(streamingChunk.ContentBlocks, block)
		}
		if len(streamingChunk.ContentBlocks) > 0 || streamingChunk.ResponseMeta != nil || streamingChunk.Extra != nil {
			streamingChunks = append(streamingChunks, streamingChunk)
		}
	}

	streamingResult, err := schema.ConcatAgenticMessages(streamingChunks)
	if err != nil {
		return nil, err
	}
	return &AgenticMessage{
		Role:          role,
		ResponseMeta:  streamingResult.ResponseMeta,
		ContentBlocks: append(nonStreamingBlocks, streamingResult.ContentBlocks...),
		Extra:         streamingResult.Extra,
	}, nil
}

// Text 提取 Agentic 消息中的文本内容。
func Text(value *AgenticMessage) string {
	// nil 消息通常来自模型空响应或流式合并失败，提取文本时返回空串即可。
	if value == nil {
		return ""
	}
	parts := make([]string, 0, len(value.ContentBlocks))
	for _, item := range value.ContentBlocks {
		// 空内容块没有文本语义。
		if item == nil {
			continue
		}
		// 助手生成文本与用户输入文本都可能需要进入历史上下文展示。
		switch {
		case item.AssistantGenText != nil && item.AssistantGenText.Text != "":
			parts = append(parts, item.AssistantGenText.Text)
		case item.UserInputText != nil && item.UserInputText.Text != "":
			parts = append(parts, item.UserInputText.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// AITextOnly 提取助手消息中的文本内容。
func AITextOnly(value *AgenticMessage) string {
	// 仅提取助手输出，工具结果和用户输入不应混入最终回复。
	if value == nil {
		return ""
	}
	parts := make([]string, 0, len(value.ContentBlocks))
	for _, item := range value.ContentBlocks {
		// 非助手文本块可能是工具调用或服务端工具事件，不参与用户可见回复。
		if item == nil || item.AssistantGenText == nil || item.AssistantGenText.Text == "" {
			continue
		}
		parts = append(parts, item.AssistantGenText.Text)
	}
	return strings.Join(parts, "")
}

// ToolCalls 从 Agentic 消息中提取函数工具调用。
func ToolCalls(value *AgenticMessage) []ToolCall {
	// 空消息不会触发函数工具调用。
	if value == nil {
		return nil
	}
	calls := make([]ToolCall, 0, len(value.ContentBlocks))
	for _, item := range value.ContentBlocks {
		// 只有 FunctionToolCall 内容块才需要交给工具节点执行。
		if item == nil || item.FunctionToolCall == nil {
			continue
		}
		call := item.FunctionToolCall
		calls = append(calls, schema.ToolCall{
			ID: call.CallID,
			Function: schema.FunctionCall{
				Name:      call.Name,
				Arguments: call.Arguments,
			},
		})
	}
	return calls
}

// ServerTool 表示模型服务端内置工具调用记录。
type ServerTool struct {
	// Name 工具名称。
	Name string
}

// ServerTools 从模型输出中提取服务端工具使用记录。
func ServerTools(value *AgenticMessage) []ServerTool {
	// 空消息不包含 Responses 服务端工具事件。
	if value == nil {
		return nil
	}
	tools := make([]ServerTool, 0, len(value.ContentBlocks))
	for _, item := range value.ContentBlocks {
		// 空块没有工具事件。
		if item == nil {
			continue
		}
		// 服务端工具调用事件和结果事件都会用于前端工具记录展示。
		if item.ServerToolCall != nil && item.ServerToolCall.Name != "" {
			tools = append(tools, ServerTool{Name: item.ServerToolCall.Name})
		}
		if item.ServerToolResult != nil && item.ServerToolResult.Name != "" {
			tools = append(tools, ServerTool{Name: item.ServerToolResult.Name})
		}
	}
	return tools
}

// Usage 提取单次模型响应 token 消耗。
func Usage(value *AgenticMessage) *TokenUsage {
	// 部分模型响应没有 ResponseMeta，调用方会用零值 token 兜底。
	if value == nil || value.ResponseMeta == nil {
		return nil
	}
	return value.ResponseMeta.TokenUsage
}
