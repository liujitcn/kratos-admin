package agent

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	modelclient "github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	modelconfig "github.com/liujitcn/kratos-kit/ai/model"
)

// TestRuntimePrepareAttachmentsUploadsSupportedFiles 验证图片和 PDF 上传后只在附件副本写入文件编号。
func TestRuntimePrepareAttachmentsUploadsSupportedFiles(t *testing.T) {
	var uploadCount atomic.Int32
	var deleteCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/v1/files":
			if err := request.ParseMultipartForm(1 << 20); err != nil {
				t.Errorf("parse upload form: %v", err)
				writer.WriteHeader(http.StatusBadRequest)
				return
			}
			if request.FormValue("purpose") != "user_data" {
				t.Errorf("upload purpose = %q, want user_data", request.FormValue("purpose"))
			}
			file, header, err := request.FormFile("file")
			if err != nil {
				t.Errorf("read uploaded file: %v", err)
				writer.WriteHeader(http.StatusBadRequest)
				return
			}
			_ = file.Close()
			if header.Filename != "diagram.png" && header.Filename != "report.pdf" {
				t.Errorf("uploaded filename = %q", header.Filename)
			}
			fileID := "file_test_" + strconv.Itoa(int(uploadCount.Add(1)))
			writer.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(writer, `{"id":"`+fileID+`","bytes":9,"created_at":1,"filename":"attachment","object":"file","purpose":"user_data","status":"processed"}`)
		case request.Method == http.MethodDelete && strings.HasPrefix(request.URL.Path, "/v1/files/"):
			deleteCount.Add(1)
			writer.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(writer, `{"id":"file_test","deleted":true,"object":"file"}`)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	input := RuntimeInput{Attachments: []Attachment{
		{Name: "diagram.png", MIMEType: "image/png", Bytes: []byte("image bytes")},
		{Name: "report.pdf", MIMEType: "application/pdf", Bytes: []byte("pdf bytes")},
	}}
	client := newFileUploadTestClient(server, modelconfig.ProviderOpenAICompatible, modelconfig.APITypeResponses)
	prepared, cleanup := (&Runtime{}).prepareAttachments(context.Background(), input, client)
	if cleanup == nil {
		t.Fatal("cleanup is nil after successful uploads")
	}
	if prepared.Attachments[0].FileID != "file_test_1" || prepared.Attachments[1].FileID != "file_test_2" {
		t.Fatalf("prepared file IDs = %q, %q", prepared.Attachments[0].FileID, prepared.Attachments[1].FileID)
	}
	if input.Attachments[0].FileID != "" || input.Attachments[1].FileID != "" {
		t.Fatal("preparing attachments mutated the original input")
	}
	cleanup()
	if uploadCount.Load() != 2 || deleteCount.Load() != 2 {
		t.Fatalf("upload count = %d, delete count = %d, want 2 each", uploadCount.Load(), deleteCount.Load())
	}
}

// TestRuntimePrepareAttachmentsFallsBackOnUploadError 验证文件 API 不可用时保留原附件处理路径。
func TestRuntimePrepareAttachmentsFallsBackOnUploadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodPost && request.URL.Path == "/v1/files" {
			http.Error(writer, "files are unavailable", http.StatusNotFound)
			return
		}
		http.NotFound(writer, request)
	}))
	defer server.Close()

	input := RuntimeInput{Content: "分析这些附件", Attachments: []Attachment{
		{Name: "diagram.png", MIMEType: "image/png", Bytes: []byte("image bytes")},
		{Name: "report.pdf", MIMEType: "application/pdf", FileID: "client_file_id", Bytes: []byte("pdf bytes")},
	}}
	prepared, cleanup := (&Runtime{}).prepareAttachments(context.Background(), input, newFileUploadTestClient(server, modelconfig.ProviderOpenAICompatible, modelconfig.APITypeResponses))
	if cleanup != nil {
		t.Fatal("cleanup should be nil when upload failed")
	}
	if prepared.Attachments[0].FileID != "" || prepared.Attachments[1].FileID != "" || input.Attachments[0].FileID != "" || input.Attachments[1].FileID != "client_file_id" {
		t.Fatal("failed upload assigned or mutated a file ID")
	}
	message := (&Runtime{}).buildUserMessage(context.Background(), prepared)
	var hasImageData bool
	var hasPDFMetadata bool
	var hasPDFFileBlock bool
	for _, block := range message.ContentBlocks {
		if block.UserInputImage != nil && block.UserInputImage.Base64Data != "" {
			hasImageData = true
		}
		if block.UserInputText != nil && strings.Contains(block.UserInputText.Text, "report.pdf") {
			hasPDFMetadata = true
		}
		if block.UserInputFile != nil {
			hasPDFFileBlock = true
		}
	}
	if !hasImageData || !hasPDFMetadata || hasPDFFileBlock {
		t.Fatalf("fallback blocks: image base64=%t PDF metadata=%t PDF file block=%t", hasImageData, hasPDFMetadata, hasPDFFileBlock)
	}
}

// TestRuntimePrepareAttachmentsSkipsUnsupportedInputs 验证不支持的 MIME 和 API 类型不会调用 Files API。
func TestRuntimePrepareAttachmentsSkipsUnsupportedInputs(t *testing.T) {
	var uploadCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodPost && request.URL.Path == "/v1/files" {
			uploadCount.Add(1)
		}
		http.NotFound(writer, request)
	}))
	defer server.Close()

	for _, testCase := range []struct {
		name     string
		provider modelconfig.Provider
		apiType  modelconfig.APIType
		mimeType string
		fileID   string
	}{
		{name: "unsupported MIME", provider: modelconfig.ProviderOpenAICompatible, apiType: modelconfig.APITypeResponses, mimeType: "application/zip", fileID: "client_file_id"},
		{name: "chat completions", provider: modelconfig.ProviderOpenAICompatible, apiType: modelconfig.APITypeChatCompletions, mimeType: "image/png"},
		{name: "ollama", provider: modelconfig.ProviderOllama, apiType: modelconfig.APITypeResponses, mimeType: "application/pdf"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			input := RuntimeInput{Attachments: []Attachment{{Name: "attachment", FileID: testCase.fileID, MIMEType: testCase.mimeType, Bytes: []byte("data")}}}
			prepared, cleanup := (&Runtime{}).prepareAttachments(context.Background(), input, newFileUploadTestClient(server, testCase.provider, testCase.apiType))
			if cleanup != nil || prepared.Attachments[0].FileID != "" {
				t.Fatal("unsupported input was uploaded")
			}
			if input.Attachments[0].FileID != testCase.fileID {
				t.Fatal("preparing attachments mutated the original input")
			}
		})
	}
	if uploadCount.Load() != 0 {
		t.Fatalf("Files API upload count = %d, want 0", uploadCount.Load())
	}
}

// TestRuntimeCleansAttachmentsAfterRequestCancellation 验证 Run 和 RunStream 取消后仍删除已上传文件。
func TestRuntimeCleansAttachmentsAfterRequestCancellation(t *testing.T) {
	for _, testCase := range []struct {
		name string
		run  func(*Runtime, context.Context, RuntimeInput) error
	}{
		{name: "Run", run: func(runtime *Runtime, ctx context.Context, input RuntimeInput) error {
			_, err := runtime.Run(ctx, input)
			return err
		}},
		{name: "RunStream", run: func(runtime *Runtime, ctx context.Context, input RuntimeInput) error {
			_, err := runtime.RunStream(ctx, input, nil)
			return err
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			responseStarted := make(chan struct{}, 1)
			var deleteCount atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				switch {
				case request.Method == http.MethodPost && request.URL.Path == "/v1/files":
					writer.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(writer, `{"id":"file_test","bytes":9,"created_at":1,"filename":"report.pdf","object":"file","purpose":"user_data","status":"processed"}`)
				case request.Method == http.MethodPost && request.URL.Path == "/v1/responses":
					body, err := io.ReadAll(request.Body)
					if err != nil {
						t.Errorf("read Responses request: %v", err)
					}
					if !strings.Contains(string(body), `"file_id":"file_test"`) {
						t.Errorf("Responses request does not contain uploaded file ID: %s", body)
					}
					responseStarted <- struct{}{}
					<-request.Context().Done()
				case request.Method == http.MethodDelete && request.URL.Path == "/v1/files/file_test":
					deleteCount.Add(1)
					writer.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(writer, `{"id":"file_test","deleted":true,"object":"file"}`)
				default:
					http.NotFound(writer, request)
				}
			}))
			defer server.Close()

			runtime := NewRuntime(RuntimeConfig{Client: newFileUploadTestClient(server, modelconfig.ProviderOpenAICompatible, modelconfig.APITypeResponses)})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			result := make(chan error, 1)
			go func() {
				result <- testCase.run(runtime, ctx, RuntimeInput{Attachments: []Attachment{{Name: "report.pdf", MIMEType: "application/pdf", Bytes: []byte("pdf bytes")}}})
			}()
			select {
			case <-responseStarted:
				cancel()
			case <-time.After(5 * time.Second):
				t.Fatal("Responses request did not start")
			}
			select {
			case err := <-result:
				if err == nil {
					t.Fatal("request cancellation returned no error")
				}
			case <-time.After(5 * time.Second):
				t.Fatal("runtime did not return after cancellation")
			}
			if deleteCount.Load() != 1 {
				t.Fatalf("provider file delete count = %d, want 1", deleteCount.Load())
			}
		})
	}
}

func newFileUploadTestClient(server *httptest.Server, provider modelconfig.Provider, apiType modelconfig.APIType) *modelclient.AssistantClient {
	return modelclient.NewAssistantClientWithName(&modelconfig.ModelConfig{
		Provider:  provider,
		ModelName: "test-model",
		APIKey:    "test-key",
		BaseURL:   server.URL + "/v1",
		APIType:   apiType,
	}, "test model")
}
