package agent

import (
	internalMessage "github.com/liujitcn/kratos-admin/backend/internal/biz/agent/message"
	internalModel "github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	internalStructured "github.com/liujitcn/kratos-admin/backend/internal/biz/agent/structured"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
)

// ChatClient 是结构化任务使用的聊天模型客户端。
type ChatClient = internalModel.ChatClient

// StructuredRunner 是按 JSON Schema 生成并解析模型结果的运行器。
type StructuredRunner = internalStructured.Runner

// Part 是结构化任务可传给模型的多模态输入片段。
type Part = internalStructured.Part

// Schema 是结构化输出使用的 JSON Schema。
type Schema = internalStructured.Schema

// NewChatClient 根据 Backend AI 模型配置创建聊天模型客户端。
func NewChatClient(modelConfig *configv1.AI_Model) *ChatClient {
	return internalModel.NewChatClient(modelConfig)
}

// NewStructuredRunner 创建结构化输出运行器。
func NewStructuredRunner(client *ChatClient) *StructuredRunner {
	return internalStructured.NewRunner(client)
}

// TextPart 构造文本输入片段。
func TextPart(content string) *Part {
	return internalStructured.TextPart(content)
}

// ImageURLPart 构造远程图片输入片段。
func ImageURLPart(rawURL string) *Part {
	return internalMessage.ImageURLPart(rawURL)
}

// ImageDataPart 构造图片字节输入片段。
func ImageDataPart(data []byte, mimeType string) *Part {
	return internalMessage.ImageDataPart(data, mimeType)
}

// SchemaFor 根据结果类型生成 JSON Schema。
func SchemaFor[T any]() (*Schema, error) {
	return internalStructured.SchemaFor[T]()
}

// SchemaPrompt 构造结构化输出的 JSON Schema 文本约束。
func SchemaPrompt(outputSchema *Schema) string {
	return internalStructured.SchemaPrompt(outputSchema)
}

// DecodeContent 解码模型返回的结构化 JSON 文本。
func DecodeContent(content string, out any) error {
	return internalStructured.DecodeContent(content, out)
}
