package model

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	modelconfig "github.com/liujitcn/kratos-kit/ai/model"
)

// TestAssistantClientFileUploadAndDelete 验证 Files API 的上传和删除请求。
func TestAssistantClientFileUploadAndDelete(t *testing.T) {
	var uploaded bool
	var deleted bool
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/v1/files":
			if request.Header.Get("Authorization") != "Bearer test-key" {
				t.Errorf("upload authorization = %q, want configured API key", request.Header.Get("Authorization"))
			}
			err := request.ParseMultipartForm(1 << 20)
			if err != nil {
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
			defer file.Close()
			if header.Filename != "report.pdf" {
				t.Errorf("uploaded filename = %q, want report.pdf", header.Filename)
			}
			data, err := io.ReadAll(file)
			if err != nil {
				t.Errorf("read uploaded bytes: %v", err)
				writer.WriteHeader(http.StatusBadRequest)
				return
			}
			if string(data) != "pdf bytes" {
				t.Errorf("uploaded bytes = %q, want pdf bytes", data)
			}
			uploaded = true
			writer.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(writer, `{"id":"file_test","bytes":9,"created_at":1,"filename":"report.pdf","object":"file","purpose":"user_data","status":"processed"}`)
		case request.Method == http.MethodDelete && request.URL.Path == "/v1/files/file_test":
			deleted = true
			writer.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(writer, `{"id":"file_test","deleted":true,"object":"file"}`)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewAssistantClientWithName(&modelconfig.ModelConfig{
		Provider:  modelconfig.ProviderOpenAICompatible,
		ModelName: "test-model",
		APIKey:    "test-key",
		BaseURL:   server.URL + "/v1",
		APIType:   modelconfig.APITypeResponses,
	}, "test model")
	if !client.SupportsFileUpload() {
		t.Fatal("Responses client should support provider file upload")
	}

	fileID, err := client.UploadFile(context.Background(), "report.pdf", []byte("pdf bytes"))
	if err != nil {
		t.Fatalf("upload file: %v", err)
	}
	if fileID != "file_test" {
		t.Fatalf("uploaded file ID = %q, want file_test", fileID)
	}
	if err = client.DeleteFile(context.Background(), fileID); err != nil {
		t.Fatalf("delete file: %v", err)
	}
	if !uploaded || !deleted {
		t.Fatalf("uploaded = %t, deleted = %t, want both true", uploaded, deleted)
	}
}

// TestAssistantClientFileUploadUsesConfiguredTimeout 验证文件上传使用模型配置时限且超时后不重试。
func TestAssistantClientFileUploadUsesConfiguredTimeout(t *testing.T) {
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount.Add(1)
		time.Sleep(1500 * time.Millisecond)
		writer.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(writer, `{"id":"file_test","bytes":9,"created_at":1,"filename":"report.pdf","object":"file","purpose":"user_data","status":"processed"}`)
	}))
	defer server.Close()

	client := NewAssistantClientWithName(&modelconfig.ModelConfig{
		Provider:       modelconfig.ProviderOpenAICompatible,
		ModelName:      "test-model",
		APIKey:         "test-key",
		BaseURL:        server.URL + "/v1",
		APIType:        modelconfig.APITypeResponses,
		TimeoutSeconds: 1,
	}, "test model")
	if _, err := client.UploadFile(context.Background(), "report.pdf", []byte("pdf bytes")); err == nil {
		t.Fatal("upload succeeded after the configured timeout")
	}
	if requestCount.Load() != 1 {
		t.Fatalf("upload request count = %d, want 1", requestCount.Load())
	}
}

// TestAssistantClientFileUploadDisabledForOtherModels 验证其他协议和 Ollama 不启用供应商文件上传。
func TestAssistantClientFileUploadDisabledForOtherModels(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		config *modelconfig.ModelConfig
	}{
		{
			name: "chat completions",
			config: &modelconfig.ModelConfig{
				Provider:  modelconfig.ProviderOpenAICompatible,
				ModelName: "test-model",
				APIKey:    "test-key",
				BaseURL:   "http://localhost/v1",
				APIType:   modelconfig.APITypeChatCompletions,
			},
		},
		{
			name: "ollama responses",
			config: &modelconfig.ModelConfig{
				Provider:  modelconfig.ProviderOllama,
				ModelName: "test-model",
				BaseURL:   "http://localhost:11434/v1",
				APIType:   modelconfig.APITypeResponses,
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			client := NewAssistantClientWithName(testCase.config, "test model")
			if client.SupportsFileUpload() {
				t.Fatal("file upload should be disabled for this model configuration")
			}
			if _, err := client.UploadFile(context.Background(), "report.pdf", []byte("pdf bytes")); err == nil || !strings.Contains(err.Error(), "not supported") {
				t.Fatalf("upload error = %v, want not supported", err)
			}
		})
	}
}
