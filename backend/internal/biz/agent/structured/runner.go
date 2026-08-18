package structured

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/message"
	einoModel "github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
)

// Runner 封装结构化输出模型调用。
type Runner struct {
	client *einoModel.ChatClient
}

// NewRunner 创建结构化输出运行器。
func NewRunner(client *einoModel.ChatClient) *Runner {
	return &Runner{client: client}
}

// DecodeContent 解码模型返回的结构化 JSON 文本。
func DecodeContent(content string, out any) error {
	cleanContent := content
	// 大部分模型在配置 JSON Schema 后会直接返回纯 JSON，先走最快路径。
	err := json.Unmarshal([]byte(cleanContent), out)
	if err == nil {
		return nil
	}
	// 少数模型仍可能包一层说明文字或 Markdown 围栏，这里只提取可被 JSON decoder 接受的片段。
	for _, jsonCandidate := range findJSONCandidates(cleanContent) {
		// 任一候选能成功解析即可返回，保留原始错误仅用于全部失败后的排障。
		if json.Unmarshal([]byte(jsonCandidate), out) == nil {
			return nil
		}
	}
	return err
}

// Enabled 判断结构化输出运行器是否可用。
func (r *Runner) Enabled() bool {
	return r != nil && r.client != nil && r.client.AgenticModel != nil
}

// Model 返回结构化输出当前使用的模型名称。
func (r *Runner) Model() string {
	if !r.Enabled() {
		return ""
	}
	return r.client.Name()
}

// Generate 按 JSON Schema 调用模型并反序列化结构化结果。
func (r *Runner) Generate(ctx context.Context, instruction string, parts []*Part, outputSchema *Schema, out any) error {
	// 模型客户端未初始化时，调用方无法继续发起大模型请求。
	if !r.Enabled() {
		return fmt.Errorf("agent chat client is not configured")
	}
	// 结构化任务必须配置系统提示词，避免用空规则调用大模型。
	if instruction == "" {
		return fmt.Errorf("agent instruction is empty")
	}
	// 输出目标为空时，无法承载结构化响应。
	if out == nil {
		return fmt.Errorf("agent structured output is nil")
	}

	// 系统消息放结构化约束，用户消息只放业务输入，避免模型把业务 payload 当作规则覆盖。
	messages := []*message.AgenticMessage{
		message.SystemText(instruction + "\n\n" + SchemaPrompt(outputSchema)),
		message.UserParts(parts),
	}
	response, err := r.client.Generate(ctx, messages)
	if err != nil {
		return fmt.Errorf("request agent structured output: %w", err)
	}
	// 服务商返回空消息时，无法解析结构化结果。
	if response == nil {
		return fmt.Errorf("agent structured response is empty")
	}

	content := message.AITextOnly(response)
	// 模型未返回 JSON 文本时，直接返回错误供调用方重试或降级。
	if content == "" {
		return fmt.Errorf("agent structured response content is empty")
	}
	err = DecodeContent(content, out)
	if err != nil {
		return fmt.Errorf("decode agent structured response: %w", err)
	}
	return nil
}

// findJSONCandidates 从模型额外说明文本中提取可能的 JSON 片段。
func findJSONCandidates(content string) []string {
	result := make([]string, 0, 1)
	for index, value := range content {
		// JSON 结构只可能从对象或数组起始符开始，跳过其他字符可以减少无意义 decoder 尝试。
		if value != '{' && value != '[' {
			continue
		}
		decoder := json.NewDecoder(strings.NewReader(content[index:]))
		var rawMessage json.RawMessage
		// RawMessage 会在第一个完整 JSON 值结束处停止，适合从“说明文字 + JSON + 说明文字”中截出候选值。
		if decoder.Decode(&rawMessage) == nil {
			result = append(result, string(rawMessage))
		}
	}
	return result
}
