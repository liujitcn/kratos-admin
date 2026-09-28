package model

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/liujitcn/kratos-kit/ai/model"
)

// TestConnectionUsesChatCompletionsMaxTokensField 验证模型连通测试发送标准Chat Completions请求。
func TestConnectionUsesChatCompletionsMaxTokensField(t *testing.T) {
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("request path = %q", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Errorf("decode request body: %v", err)
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl-test","object":"chat.completion","created":1,"model":"test-model","choices":[{"index":0,"message":{"role":"assistant","content":"OK"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	defer server.Close()

	err := TestConnection(context.Background(), &model.ModelConfig{
		Provider:  model.ProviderOpenAICompatible,
		ModelName: "test-model",
		APIKey:    "test-key",
		BaseURL:   server.URL + "/v1",
		MaxTokens: 16,
		APIType:   model.APITypeChatCompletions,
	})
	if err != nil {
		t.Fatal(err)
	}
	if requestBody["max_tokens"] != float64(16) {
		t.Fatalf("max_tokens = %v, want 16", requestBody["max_tokens"])
	}
	if _, exists := requestBody["max_completion_tokens"]; exists {
		t.Fatalf("request contains max_completion_tokens: %v", requestBody["max_completion_tokens"])
	}
}
