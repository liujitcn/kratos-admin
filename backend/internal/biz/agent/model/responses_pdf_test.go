package model

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/securityconfig"
	"github.com/liujitcn/kratos-kit/ai/model"
)

// TestResponsesInlinePDFRequest 验证真实模型客户端将内联 PDF 转为协议要求的 data URL。
func TestResponsesInlinePDFRequest(t *testing.T) {
	previous := securityconfig.Current()
	t.Cleanup(func() { securityconfig.Set(previous) })
	securityconfig.Set(securityconfig.Config{AIAllowedCIDRs: []string{"127.0.0.1/32"}})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Input []struct {
				Content []struct {
					Type     string `json:"type"`
					FileData string `json:"file_data"`
					Filename string `json:"filename"`
				} `json:"content"`
			} `json:"input"`
		}
		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if len(request.Input) != 1 || len(request.Input[0].Content) != 1 {
			t.Errorf("文件输入丢失: %+v", request)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		file := request.Input[0].Content[0]
		if file.Type != "input_file" || file.Filename != "report.pdf" || file.FileData != "data:application/pdf;base64,JVBERi0xLjc=" {
			t.Errorf("内联 PDF 请求格式错误: %+v", file)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, err = io.WriteString(w, `{"id":"resp_pdf","object":"response","status":"completed","model":"test","output":[{"id":"msg_pdf","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"PDF_OK","annotations":[]}]}]}`)
		if err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	client, err := newResponsesModel(t.Context(), &model.ModelConfig{
		Provider:  model.ProviderOpenAICompatible,
		APIType:   model.APITypeResponses,
		APIKey:    "test-key",
		BaseURL:   server.URL + "/v1",
		ModelName: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Generate(t.Context(), []*schema.AgenticMessage{{
		Role: schema.AgenticRoleTypeUser,
		ContentBlocks: []*schema.ContentBlock{schema.NewContentBlock(&schema.UserInputFile{
			Name: "report.pdf", MIMEType: "application/pdf", Base64Data: "JVBERi0xLjc=",
		})},
	}})
	if err != nil {
		t.Fatal(err)
	}
}

// TestResponsesPDFTransportPreservesInputs 验证适配不会改变文件引用、已有前缀或非 Responses 输入及大整数。
func TestResponsesPDFTransportPreservesInputs(t *testing.T) {
	for _, test := range []struct {
		name string
		path string
		body string
		want string
	}{
		{
			name: "PDF与大整数", path: "/v1/responses",
			body: `{"input":[{"role":"user","content":[{"type":"input_file","file_data":"JVBERi0xLjc="}]}],"metadata":{"sequence":9007199254740993}}`,
			want: `{"input":[{"role":"user","content":[{"type":"input_file","file_data":"data:application/pdf;base64,JVBERi0xLjc="}]}],"metadata":{"sequence":9007199254740993}}`,
		},
		{
			name: "已带前缀", path: "/v1/responses",
			body: `{"input":[{"content":[{"type":"input_file","file_data":"data:application/pdf;base64,JVBERi0xLjc="}]}]}`,
		},
		{
			name: "文件引用和图片", path: "/v1/responses",
			body: `{"input":[{"content":[{"type":"input_file","file_id":"file-1"},{"type":"input_file","file_url":"https://example.com/report.pdf"},{"type":"input_image","image_url":"data:image/png;base64,AA=="}]}]}`,
		},
		{
			name: "其他文件类型", path: "/v1/responses",
			body: `{"input":[{"content":[{"type":"input_file","file_data":"aGVsbG8="}]}]}`,
		},
		{
			name: "聊天补全", path: "/v1/chat/completions",
			body: `{"messages":[{"content":[{"type":"file","file_data":"JVBERi0xLjc="}]}]}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			captured := make(chan string, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				captured <- string(body)
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()
			client := &http.Client{Transport: responsesPDFTransport{RoundTripper: http.DefaultTransport}}
			request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, server.URL+test.path, strings.NewReader(test.body))
			if err != nil {
				t.Fatal(err)
			}
			var response *http.Response
			response, err = client.Do(request)
			if err != nil {
				t.Fatal(err)
			}
			err = response.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			want := test.want
			if want == "" {
				want = test.body
			}
			var decoded [2]any
			for index, body := range []string{<-captured, want} {
				decoder := json.NewDecoder(strings.NewReader(body))
				decoder.UseNumber()
				err = decoder.Decode(&decoded[index])
				if err != nil {
					t.Fatal(err)
				}
			}
			if !reflect.DeepEqual(decoded[0], decoded[1]) {
				t.Fatalf("请求内容被意外修改: got=%v want=%v", decoded[0], decoded[1])
			}
		})
	}
}
