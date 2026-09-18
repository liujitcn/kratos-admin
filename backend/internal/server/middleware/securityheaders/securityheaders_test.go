package securityheaders

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestNewHandler 验证真实 HTTPS 响应带 HSTS，伪造代理头不会改变协议判断。
func TestNewHandler(t *testing.T) {
	handler := NewHandler(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "http://example.com/admin/", nil)
	request.Header.Set("X-Forwarded-Proto", "https")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	for _, header := range []string{"Content-Security-Policy", "Permissions-Policy", "Referrer-Policy", "X-Content-Type-Options", "X-Frame-Options"} {
		if recorder.Header().Get(header) == "" {
			t.Fatalf("缺少安全响应头 %s", header)
		}
	}
	if recorder.Header().Get("Strict-Transport-Security") != "" {
		t.Fatal("伪造 X-Forwarded-Proto 不应启用 HSTS")
	}

	request = httptest.NewRequest(http.MethodGet, "https://example.com/admin/", nil)
	request.TLS = &tls.ConnectionState{}
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Header().Get("Strict-Transport-Security") == "" {
		t.Fatal("真实 HTTPS 请求应启用 HSTS")
	}
}
