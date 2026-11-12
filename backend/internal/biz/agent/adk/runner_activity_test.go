package adk

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/callback"
)

func TestConsumeEventsEmitsReasoningSummaryAndAnswerActivity(t *testing.T) {
	iterator, generator := adk.NewAsyncIteratorPair[*adk.TypedAgentEvent[*schema.AgenticMessage]]()
	generator.Send(&adk.TypedAgentEvent[*schema.AgenticMessage]{
		Output: &adk.TypedAgentOutput[*schema.AgenticMessage]{
			MessageOutput: &adk.TypedMessageVariant[*schema.AgenticMessage]{
				IsStreaming: true,
				AgenticRole: schema.AgenticRoleTypeAssistant,
				MessageStream: schema.StreamReaderFromArray([]*schema.AgenticMessage{{
					Role: schema.AgenticRoleTypeAssistant,
					ContentBlocks: []*schema.ContentBlock{
						schema.NewContentBlock(&schema.Reasoning{Text: "正在分析"}),
						schema.NewContentBlock(&schema.AssistantGenText{Text: "完成"}),
					},
				}}),
			},
		},
	})
	generator.Close()

	var activities []callback.Activity
	ctx := callback.WithActivityEmitter(context.Background(), func(activity callback.Activity) {
		activities = append(activities, activity)
	})
	_, _, err := consumeEvents(ctx, iterator, func(string) {}, true)
	if err != nil {
		t.Fatalf("consumeEvents() error = %v", err)
	}
	if len(activities) != 2 {
		t.Fatalf("activity count = %d, want 2", len(activities))
	}
	if activities[0].Phase != "reasoning" || activities[0].Summary != "正在分析" {
		t.Fatalf("reasoning activity = %#v", activities[0])
	}
	if activities[1].Phase != "answering" || activities[1].Status != "running" {
		t.Fatalf("answer activity = %#v", activities[1])
	}
}
