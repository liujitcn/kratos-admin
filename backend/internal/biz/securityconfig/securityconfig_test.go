package securityconfig

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestLiveAIHTTPClientAppliesRevocation 验证已创建的客户端在后续请求中使用更新后的出站策略。
func TestLiveAIHTTPClientAppliesRevocation(t *testing.T) {
	previous := Current()
	t.Cleanup(func() { Set(previous) })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	Set(Config{AIAllowedCIDRs: []string{"127.0.0.1/32"}})
	client := NewLiveAIHTTPClient(time.Second)
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("允许访问时状态 = %d，预期 204", response.StatusCode)
	}

	Set(Default())
	response, err = client.Do(request)
	if err == nil {
		_ = response.Body.Close()
		t.Fatal("撤销白名单后，已有客户端仍能访问私网")
	}
}
