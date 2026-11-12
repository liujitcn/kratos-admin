package adk

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/message"
)

func TestConsumeEventsKeepsStreamTextWhenTrailingAssistantMessageIsEmpty(t *testing.T) {
	iterator, generator := adk.NewAsyncIteratorPair[*adk.TypedAgentEvent[*schema.AgenticMessage]]()
	generator.Send(&adk.TypedAgentEvent[*schema.AgenticMessage]{
		Output: &adk.TypedAgentOutput[*schema.AgenticMessage]{
			MessageOutput: &adk.TypedMessageVariant[*schema.AgenticMessage]{
				IsStreaming:   true,
				AgenticRole:   schema.AgenticRoleTypeAssistant,
				MessageStream: schema.StreamReaderFromArray([]*schema.AgenticMessage{message.AIText("完整回复")}),
			},
		},
	})
	generator.Send(&adk.TypedAgentEvent[*schema.AgenticMessage]{
		Output: &adk.TypedAgentOutput[*schema.AgenticMessage]{
			MessageOutput: &adk.TypedMessageVariant[*schema.AgenticMessage]{
				AgenticRole: schema.AgenticRoleTypeAssistant,
				Message:     message.AIText(""),
			},
		},
	})
	generator.Close()

	finalMessage, _, err := consumeEvents(context.Background(), iterator, nil, true)
	if err != nil {
		t.Fatalf("consumeEvents() error = %v", err)
	}
	if got := message.AITextOnly(finalMessage); got != "完整回复" {
		t.Fatalf("consumeEvents() text = %q, want %q", got, "完整回复")
	}
}
