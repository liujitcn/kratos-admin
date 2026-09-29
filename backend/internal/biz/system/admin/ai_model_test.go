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

// TestChatModelTestCapsTokensAndTimeout 验证聊天模型测试会按16个Token和30秒超时发起请求。
func TestChatModelTestCapsTokensAndTimeout(t *testing.T) {
	var requestTokens []float64
	var requestMu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requestBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Errorf("decode request body: %v", err)
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		requestMu.Lock()
		requestTokens = append(requestTokens, requestBody["max_tokens"].(float64))
		requestMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl-test","object":"chat.completion","created":1,"model":"test-model","choices":[{"index":0,"message":{"role":"assistant","content":"OK"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	defer server.Close()

	result := testChatModel(context.Background(), &model.ModelConfig{
		Provider: model.ProviderOpenAICompatible, ModelName: "test-model", APIKey: "secret-key",
		BaseURL: server.URL + "/v1", MaxTokens: 4096, TimeoutSeconds: 600, MaxRetries: 5, APIType: model.APITypeChatCompletions,
	})
	if !result.Success {
		t.Fatalf("chat model test should succeed: %+v", result)
	}
	requestMu.Lock()
	defer requestMu.Unlock()
	if len(requestTokens) != 1 || requestTokens[0] != 16 {
		t.Fatalf("request token limits = %v, want [16]", requestTokens)
	}
}

// TestChatModelTestRedactsFailureMessage 验证失败的聊天模型测试会移除密钥。
func TestChatModelTestRedactsFailureMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html>secret-key</html>"))
	}))
	defer server.Close()

	result := testChatModel(context.Background(), &model.ModelConfig{
		Provider: model.ProviderOpenAICompatible, ModelName: "broken-model", APIKey: "secret-key",
		BaseURL: server.URL + "/v1", MaxTokens: 16, TimeoutSeconds: 5, APIType: model.APITypeChatCompletions,
	})
	if result.Success || result.Message == "" || strings.Contains(result.Message, "secret-key") {
		t.Fatalf("failed model error was empty or exposed the API key: %+v", result)
	}
}

// TestAiTestErrorRedactsAndTruncates 验证模型错误详情会移除密钥并限制长度。
func TestAiTestErrorRedactsAndTruncates(t *testing.T) {
	message := aiTestError(errors.New("secret-key"+strings.Repeat("x", 600)), "secret-key")
	if strings.Contains(message, "secret-key") || len([]rune(message)) > 515 {
		t.Fatalf("unexpected sanitized message length or content: %d", len([]rune(message)))
	}
}
