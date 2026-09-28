package biz

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/liujitcn/kratos-kit/ai/model"
)

// TestRunAiProviderModelTestsContinuesAfterFailure 验证模型失败后仍会测试后续模型。
func TestRunAiProviderModelTestsContinuesAfterFailure(t *testing.T) {
	requestModels := make([]string, 0, 2)
	requestTokens := make([]float64, 0, 2)
	var requestMu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requestBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Errorf("decode request body: %v", err)
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		requestMu.Lock()
		requestModels = append(requestModels, requestBody["model"].(string))
		requestTokens = append(requestTokens, requestBody["max_tokens"].(float64))
		requestMu.Unlock()
		if requestBody["model"] == "broken-model" {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("<html>secret-key</html>"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl-test","object":"chat.completion","created":1,"model":"test-model","choices":[{"index":0,"message":{"role":"assistant","content":"OK"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	defer server.Close()

	results := testAiProviderModels(context.Background(), []*model.ModelConfig{
		{Provider: model.ProviderOpenAICompatible, ModelName: "working-model", APIKey: "secret-key", BaseURL: server.URL + "/v1", MaxTokens: 4096, TimeoutSeconds: 600, MaxRetries: 5, APIType: model.APITypeChatCompletions},
		{Provider: model.ProviderOpenAICompatible, ModelName: "broken-model", APIKey: "secret-key", BaseURL: server.URL + "/v1", MaxTokens: 4096, TimeoutSeconds: 600, MaxRetries: 5, APIType: model.APITypeChatCompletions},
	})
	if len(results) != 2 || !results[0].Success || results[1].Success {
		t.Fatalf("unexpected test results: %+v", results)
	}
	requestMu.Lock()
	gotModels := append([]string(nil), requestModels...)
	gotTokens := append([]float64(nil), requestTokens...)
	requestMu.Unlock()
	if len(gotModels) != 2 || gotModels[0] != "working-model" || gotModels[1] != "broken-model" {
		t.Fatalf("tested models = %v, want both models in order", gotModels)
	}
	if len(gotTokens) != 2 || gotTokens[0] != 16 || gotTokens[1] != 16 {
		t.Fatalf("request token limits = %v, want [16 16]", gotTokens)
	}
	if results[1].Message == "" || strings.Contains(results[1].Message, "secret-key") {
		t.Fatalf("failed model error was empty or exposed the API key: %q", results[1].Message)
	}
}

// TestAiProviderTestErrorRedactsAndTruncates 验证模型错误详情会移除密钥并限制长度。
func TestAiProviderTestErrorRedactsAndTruncates(t *testing.T) {
	message := aiProviderTestError(errors.New("secret-key"+strings.Repeat("x", 600)), "secret-key")
	if strings.Contains(message, "secret-key") || len([]rune(message)) > 515 {
		t.Fatalf("unexpected sanitized message length or content: %d", len([]rune(message)))
	}
}
