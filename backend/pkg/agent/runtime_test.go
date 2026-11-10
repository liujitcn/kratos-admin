package agent

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/message"
	agentmodel "github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	modelconfig "github.com/liujitcn/kratos-kit/ai/model"
)

// pdfInputModel 记录实际交给模型的附件输入。
type pdfInputModel struct{ messages []*schema.AgenticMessage }

// Generate 保存输入并返回文本回复。
func (m *pdfInputModel) Generate(_ context.Context, input []*schema.AgenticMessage, _ ...model.Option) (*schema.AgenticMessage, error) {
	m.messages = input
	return message.AIText("done"), nil
}

// Stream 通过同一输入路径返回流式回复。
func (m *pdfInputModel) Stream(ctx context.Context, input []*schema.AgenticMessage, opts ...model.Option) (*schema.StreamReader[*schema.AgenticMessage], error) {
	reply, err := m.Generate(ctx, input, opts...)
	return schema.StreamReaderFromArray([]*schema.AgenticMessage{reply}), err
}

// pdfClientResolver 在请求选择时覆盖默认客户端。
type pdfClientResolver struct {
	*agentmodel.Registry
	client *AssistantClient
}

// ResolveAssistantClient 返回本轮选择的客户端。
func (r pdfClientResolver) ResolveAssistantClient(int64, string) (*AssistantClient, error) {
	return r.client, nil
}

// TestRuntimePDFProtocol 验证普通与流式调用按所选协议使用 PDF 文本或内联文件。
func TestRuntimePDFProtocol(t *testing.T) {
	for _, protocol := range []struct {
		name     string
		apiType  modelconfig.APIType
		wantFile bool
	}{{"chat", modelconfig.APITypeChatCompletions, false}, {"responses", modelconfig.APITypeResponses, true}} {
		for _, stream := range []bool{false, true} {
			name := protocol.name
			if stream {
				name += "_stream"
			}
			t.Run(name, func(t *testing.T) {
				model := &pdfInputModel{}
				client := NewAssistantClient(&modelconfig.ModelConfig{APIType: protocol.apiType})
				client.AgenticModel = model
				// 默认客户端使用相反协议，防止附件判断绕过本轮模型选择。
				defaultAPI := modelconfig.APITypeResponses
				if protocol.wantFile {
					defaultAPI = modelconfig.APITypeChatCompletions
				}
				runtime := NewRuntime(RuntimeConfig{Client: NewAssistantClient(&modelconfig.ModelConfig{APIType: defaultAPI}), ClientResolver: pdfClientResolver{client: client}})
				input := RuntimeInput{Content: "Read the report", CheckPointID: "run", Attachments: []Attachment{{Name: "report.pdf", MIMEType: "application/pdf", Bytes: []byte("%PDF report"), Content: "Revenue: 42"}}}
				var err error
				if stream {
					_, err = runtime.RunStream(t.Context(), input, nil)
				} else {
					_, err = runtime.Run(t.Context(), input)
				}
				if err != nil {
					t.Fatal(err)
				}
				user := model.messages[len(model.messages)-1]
				files := 0
				for _, block := range user.ContentBlocks {
					if block.UserInputFile != nil {
						files++
						if !protocol.wantFile || block.UserInputFile.Base64Data != base64.StdEncoding.EncodeToString([]byte("%PDF report")) {
							t.Fatalf("错误的 PDF 文件输入: %+v", block.UserInputFile)
						}
					}
				}
				if protocol.wantFile && files != 1 {
					t.Fatalf("Responses 应包含一份内联 PDF，实际 %d", files)
				}
				if !protocol.wantFile && !strings.Contains(message.Text(user), "Revenue: 42") {
					t.Fatal("Chat Completions 丢失提取的 PDF 文本")
				}
			})
		}
	}
}
