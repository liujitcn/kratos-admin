package structured

import "github.com/liujitcn/kratos-admin/backend/pkg/agent/eino/message"

// Part 表示结构化任务可传给模型的多模态输入片段。
type Part = message.ContentBlock

// TextPart 构造文本输入片段。
func TextPart(text string) *Part {
	return message.TextPart(text)
}
