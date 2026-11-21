package middleware

import (
	"fmt"
	"slices"
	"strings"

	"github.com/cloudwego/eino/schema"
)

// removeVerifiedToolCallFragments 仅移除与同一 Responses 条目完整调用逐字一致的参数碎片。
func removeVerifiedToolCallFragments(blocks []*schema.ContentBlock) ([]*schema.ContentBlock, error) {
	complete := make(map[string]*schema.FunctionToolCall)
	fragments := make(map[string]*strings.Builder)
	for _, block := range blocks {
		if block == nil || block.FunctionToolCall == nil {
			continue
		}
		call := block.FunctionToolCall
		itemID := ""
		if value := block.Extra["openai-item-id"]; value != nil {
			itemID = fmt.Sprint(value)
		}
		if call.Name == "" && call.CallID == "" {
			if itemID == "" {
				return nil, fmt.Errorf("模型返回无来源的工具参数碎片，已停止执行")
			}
			if fragments[itemID] == nil {
				fragments[itemID] = &strings.Builder{}
			}
			fragments[itemID].WriteString(call.Arguments)
			continue
		}
		if call.Name == "" || call.CallID == "" {
			return nil, fmt.Errorf("模型返回不完整的工具调用，已停止执行")
		}
		if itemID != "" {
			if previous, ok := complete[itemID]; ok && (previous.CallID != call.CallID || previous.Name != call.Name || previous.Arguments != call.Arguments) {
				return nil, fmt.Errorf("模型返回冲突的工具来源条目，已停止执行")
			}
			complete[itemID] = call
		}
	}
	for itemID, arguments := range fragments {
		call := complete[itemID]
		if call == nil || arguments.String() != call.Arguments {
			return nil, fmt.Errorf("模型工具参数碎片与完整调用不一致，已停止执行")
		}
	}
	return slices.DeleteFunc(blocks, func(block *schema.ContentBlock) bool {
		return block != nil && block.FunctionToolCall != nil && block.FunctionToolCall.Name == "" && block.FunctionToolCall.CallID == ""
	}), nil
}
