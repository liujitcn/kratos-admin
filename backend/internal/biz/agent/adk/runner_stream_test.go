package adk

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/message"
)

type streamModeProbeModel struct {
	generateCalls int
	streamCalls   int
}

func (m *streamModeProbeModel) Generate(context.Context, []*schema.AgenticMessage, ...model.Option) (*schema.AgenticMessage, error) {
	m.generateCalls++
	return message.AIText("完整回复"), nil
}

func (m *streamModeProbeModel) Stream(context.Context, []*schema.AgenticMessage, ...model.Option) (*schema.StreamReader[*schema.AgenticMessage], error) {
	m.streamCalls++
	return schema.StreamReaderFromArray([]*schema.AgenticMessage{message.AIText("流式"), message.AIText("回复")}), nil
}

// TestRunnerUsesRequestStreamingMode 验证 Runner 按请求模式选择模型调用方式，
// SSE 场景由模型层的流归一化保证混合块序列不破坏 ADK 内部合并。
func TestRunnerUsesRequestStreamingMode(t *testing.T) {
	tests := []struct {
		name           string
		stream         bool
		modelStreaming bool
		wantGenerate   int
		wantStream     int
		wantMessage    string
		wantDeltaText  string
	}{
		// stream 用例同时覆盖“SSE 透传但模型禁用流式”的组合：整段文本仍要透传给前端。
		// 非 SSE 场景（自动化任务）不启用模型流式，恒走 Generate。
		{name: "stream", stream: true, modelStreaming: true, wantStream: 1, wantMessage: "流式回复", wantDeltaText: "流式回复"},
		{name: "generate", stream: false, modelStreaming: false, wantGenerate: 1, wantMessage: "完整回复", wantDeltaText: ""},
		{name: "sse-with-generate", stream: true, modelStreaming: false, wantGenerate: 1, wantMessage: "完整回复", wantDeltaText: "完整回复"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			probe := &streamModeProbeModel{}
			var deltaText string
			runner := NewRunner(Config{Model: probe, Name: "test", Description: "test"})
			result, err := runner.Run(context.Background(), Request{
				Messages:       []*message.AgenticMessage{message.UserText("测试")},
				Stream:         test.stream,
				ModelStreaming: test.modelStreaming,
				OnDelta: func(delta string) {
					deltaText += delta
				},
			})
			if err != nil {
				t.Fatalf("Runner.Run() error = %v", err)
			}
			if probe.generateCalls != test.wantGenerate {
				t.Fatalf("Generate() calls = %d, want %d", probe.generateCalls, test.wantGenerate)
			}
			if probe.streamCalls != test.wantStream {
				t.Fatalf("Stream() calls = %d, want %d", probe.streamCalls, test.wantStream)
			}
			if got := message.AITextOnly(result.Message); got != test.wantMessage {
				t.Fatalf("Runner.Run() message = %q, want %q", got, test.wantMessage)
			}
			if deltaText != test.wantDeltaText {
				t.Fatalf("OnDelta() text = %q, want %q", deltaText, test.wantDeltaText)
			}
		})
	}
}
