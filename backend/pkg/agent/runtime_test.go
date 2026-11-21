package agent

import (
	"context"
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/message"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	modelconfig "github.com/liujitcn/kratos-kit/ai/model"
)

// TestBuildUserMessageKeepsCheckpointPDF 验证审批检查点模式仍将 PDF 内容作为可恢复的文件输入发送。
func TestBuildUserMessageKeepsCheckpointPDF(t *testing.T) {
	runtime := &Runtime{}
	client := model.NewAssistantClient(&modelconfig.ModelConfig{APIType: modelconfig.APITypeResponses})
	input := RuntimeInput{
		CheckPointID: "approval-1",
		Content:      "请阅读附件",
		Attachments: []Attachment{{
			Name:     "report.pdf",
			MIMEType: "application/pdf",
			Bytes:    []byte("%PDF-1.7"),
		}},
	}
	var cleanup func()
	input, cleanup = runtime.prepareAttachments(t.Context(), input, client)
	if cleanup != nil {
		t.Fatal("检查点输入不应依赖需要清理的供应商临时文件")
	}
	userMessage := runtime.buildUserMessage(t.Context(), input, client)
	for _, block := range userMessage.ContentBlocks {
		if file := block.UserInputFile; file != nil {
			if file.Name != "report.pdf" || file.FileID != "" || file.Base64Data != "JVBERi0xLjc=" {
				t.Fatalf("PDF 文件输入未保留名称与内联内容: %+v", file)
			}
			return
		}
	}
	t.Fatal("审批检查点模式丢失 PDF 文件内容")
}

// TestBuildUserMessageSkipsUnuploadedPDFBytes 验证供应商文件上传失败时不发送无效的 PDF 原始文件输入。
func TestBuildUserMessageSkipsUnuploadedPDFBytes(t *testing.T) {
	runtime := &Runtime{}
	client := model.NewAssistantClient(&modelconfig.ModelConfig{APIType: modelconfig.APITypeResponses})
	userMessage := runtime.buildUserMessage(context.Background(), RuntimeInput{
		Content: "请阅读附件",
		Attachments: []Attachment{{
			Name:     "report.pdf",
			MIMEType: "application/pdf",
			Bytes:    []byte("%PDF-1.7"),
		}},
	}, client)

	for _, block := range userMessage.ContentBlocks {
		if block.UserInputFile != nil {
			t.Fatal("未上传成功的 PDF 不应生成 UserInputFile")
		}
	}
}

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
