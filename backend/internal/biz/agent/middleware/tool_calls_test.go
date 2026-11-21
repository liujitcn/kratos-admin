package middleware

import (
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/message"
)

// TestToolFilterVerifiesResponseFragments 验证仅丢弃与同源完整调用逐字一致的匿名参数碎片。
func TestToolFilterVerifiesResponseFragments(t *testing.T) {
	for _, test := range []struct {
		name          string
		fragment      string
		itemID        string
		completeFirst bool
		wantErr       bool
	}{
		{name: "完整调用在后", fragment: "42}", itemID: "item-1"},
		{name: "完整调用在前", fragment: "42}", itemID: "item-1", completeFirst: true},
		{name: "碎片参数不一致", fragment: "43}", itemID: "item-1", wantErr: true},
		{name: "碎片来源不一致", fragment: "42}", itemID: "item-other", wantErr: true},
		{name: "碎片无来源", fragment: "42}", wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			placeholder := schema.NewContentBlock(&schema.FunctionToolCall{CallID: "call-1", Name: "read_counter"})
			placeholder.Extra = map[string]any{"openai-item-id": "item-1"}
			first := schema.NewContentBlock(&schema.FunctionToolCall{Arguments: `{"count":`})
			first.Extra = map[string]any{"openai-item-id": test.itemID}
			last := schema.NewContentBlock(&schema.FunctionToolCall{Arguments: test.fragment})
			last.Extra = map[string]any{"openai-item-id": test.itemID}
			complete := schema.NewContentBlock(&schema.FunctionToolCall{CallID: "call-1", Name: "read_counter", Arguments: `{"count":42}`})
			complete.Extra = map[string]any{"openai-item-id": "item-1"}
			blocks := []*schema.ContentBlock{placeholder, first, last, complete}
			if test.completeFirst {
				blocks = []*schema.ContentBlock{complete, first, last, placeholder}
			}
			original := &schema.AgenticMessage{Role: schema.AgenticRoleTypeAssistant, ContentBlocks: blocks}
			state := &adk.TypedChatModelAgentState[*schema.AgenticMessage]{Messages: []*schema.AgenticMessage{original}}
			handler := NewToolFilterHandler(nil)
			var err error
			_, state, err = handler.AfterModelRewriteState(t.Context(), state, nil)
			if (err != nil) != test.wantErr {
				t.Fatalf("wantErr=%v，err=%v", test.wantErr, err)
			}
			if test.wantErr {
				return
			}
			calls := message.ToolCalls(state.Messages[0])
			if len(calls) != 1 || calls[0].ID != "call-1" || calls[0].Function.Name != "read_counter" || calls[0].Function.Arguments != `{"count":42}` {
				t.Fatalf("匿名碎片未按同源完整调用校验: %+v", calls)
			}
			if len(original.ContentBlocks) != 4 || original.ContentBlocks[0] != blocks[0] {
				t.Fatal("校验修改了原始模型消息")
			}
		})
	}
}
