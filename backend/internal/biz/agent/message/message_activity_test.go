package message

import (
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/eino/schema/openai"
)

func TestReasoningSummariesSkipsRawReasoningExtension(t *testing.T) {
	value := &schema.AgenticMessage{
		ContentBlocks: []*schema.ContentBlock{
			schema.NewContentBlock(&schema.Reasoning{Text: "可展示摘要"}),
			schema.NewContentBlock(&schema.Reasoning{
				OpenAIExtension: &openai.ReasoningExtension{
					Content: []*openai.ReasoningContent{{Text: "原始推理"}},
				},
			}),
		},
	}

	parts := ReasoningSummaries(value)
	if len(parts) != 1 || parts[0].Text != "可展示摘要" {
		t.Fatalf("ReasoningSummaries() = %#v, want one summary part", parts)
	}
}
