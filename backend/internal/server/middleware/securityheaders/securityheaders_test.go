package securityheaders

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestNewHandler 验证普通响应和代理 HTTPS 响应均带必要安全头。
func TestNewHandler(t *testing.T) {
	handler := NewHandler(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "http://example.com/admin/", nil)
	request.Header.Set("X-Forwarded-Proto", "https")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	for _, header := range []string{"Content-Security-Policy", "Permissions-Policy", "Referrer-Policy", "Strict-Transport-Security", "X-Content-Type-Options", "X-Frame-Options"} {
		if recorder.Header().Get(header) == "" {
			t.Fatalf("缺少安全响应头 %s", header)
		}
	}
}
