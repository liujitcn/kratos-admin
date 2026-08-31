package agent

import (
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/message"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/structured"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
)

// ChatClient 是结构化任务使用的聊天模型客户端。
type ChatClient = model.ChatClient

// StructuredRunner 是按 JSON Schema 生成并解析模型结果的运行器。
type StructuredRunner = structured.Runner

// Part 是结构化任务可传给模型的多模态输入片段。
type Part = structured.Part

// Schema 是结构化输出使用的 JSON Schema。
type Schema = structured.Schema

// NewChatClient 根据 Backend AI 模型配置创建聊天模型客户端。
func NewChatClient(modelConfig *configv1.AI_Model) *ChatClient {
	return model.NewChatClient(modelConfig)
}

// NewStructuredRunner 创建结构化输出运行器。
func NewStructuredRunner(client *ChatClient) *StructuredRunner {
	return structured.NewRunner(client)
}

// TextPart 构造文本输入片段。
func TextPart(content string) *Part {
	return structured.TextPart(content)
}

// ImageURLPart 构造远程图片输入片段。
func ImageURLPart(rawURL string) *Part {
	return message.ImageURLPart(rawURL)
}

// ImageDataPart 构造图片字节输入片段。
func ImageDataPart(data []byte, mimeType string) *Part {
	return message.ImageDataPart(data, mimeType)
}

// SchemaFor 根据结果类型生成 JSON Schema。
func SchemaFor[T any]() (*Schema, error) {
	return structured.SchemaFor[T]()
}

// SchemaPrompt 构造结构化输出的 JSON Schema 文本约束。
func SchemaPrompt(outputSchema *Schema) string {
	return structured.SchemaPrompt(outputSchema)
}

// DecodeContent 解码模型返回的结构化 JSON 文本。
func DecodeContent(content string, out any) error {
	return structured.DecodeContent(content, out)
}
