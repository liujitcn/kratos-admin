package agent

import (
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/message"
)

func TestHasUnparsedToolCall(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		content string
		want    bool
	}{
		{name: "标准标签", content: "<tool_calls>{\"name\":\"list\"}</tool_calls>", want: true},
		{name: "DeepSeek标签", content: "<｜tool▁calls▁begin｜>", want: true},
		{name: "普通文本", content: "已完成统计", want: false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if got := hasUnparsedToolCall(message.AIText(testCase.content)); got != testCase.want {
				t.Fatalf("hasUnparsedToolCall() = %v, want %v", got, testCase.want)
			}
		})
	}
}
