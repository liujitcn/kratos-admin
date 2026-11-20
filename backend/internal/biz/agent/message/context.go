package message

import (
	"encoding/json"
	"math"
	"unicode/utf8"

	"github.com/cloudwego/eino/schema"
)

// EstimateTokens 保守估算消息、工具和多模态载荷；不是供应商精确分词，调用方还需保留安全余量。
func EstimateTokens(messages []*AgenticMessage, tools []*schema.ToolInfo) int {
	textMessages := make([]*AgenticMessage, 0, len(messages))
	mediaTokens := 0
	for _, item := range messages {
		if item == nil {
			continue
		}
		copyMessage := *item
		copyMessage.ContentBlocks = make([]*schema.ContentBlock, 0, len(item.ContentBlocks))
		for _, block := range item.ContentBlocks {
			if block == nil {
				continue
			}
			if block.UserInputImage != nil {
				// shortcut: 图像每张保守预留8192 Token；接入供应商精确计数后替换，Base64不是模型文本。
				mediaTokens += 8192
				continue
			}
			copyMessage.ContentBlocks = append(copyMessage.ContentBlocks, block)
		}
		textMessages = append(textMessages, &copyMessage)
	}
	value, err := json.Marshal(struct {
		Messages []*AgenticMessage
		Tools    []*schema.ToolInfo
	}{textMessages, tools})
	if err != nil {
		// 无法序列化的上下文必须被预算闸门拒绝，不能作为零成本继续请求。
		return math.MaxInt / 1024
	}
	ascii := 0
	nonASCII := 0
	for len(value) > 0 {
		r, size := utf8.DecodeRune(value)
		if r < utf8.RuneSelf {
			ascii++
		} else {
			nonASCII += 2
		}
		value = value[size:]
	}
	return (ascii+2)/3 + nonASCII + 16*len(messages) + 512 + mediaTokens
}
