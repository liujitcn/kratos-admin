package oauth

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestDecryptOauthRequestRejectsOversizedBody 验证开放授权请求体超过上限时不会被完整读入内存。
func TestDecryptOauthRequestRejectsOversizedBody(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/oauth/test", bytes.NewReader(make([]byte, maxOAuthCryptoBodyBytes+1)))
	if err := decryptOauthRequest(request, nil); err == nil {
		t.Fatal("超大开放授权请求体应被拒绝")
	}
}

// TestOauthCryptoResponseWriterRejectsOversizedBody 验证加密响应缓存超过上限时停止增长。
func TestOauthCryptoResponseWriterRejectsOversizedBody(t *testing.T) {
	writer := newOauthCryptoResponseWriter(httptest.NewRecorder())
	if _, err := writer.Write(make([]byte, maxOAuthCryptoBodyBytes+1)); err == nil {
		t.Fatal("超大开放授权响应应被拒绝")
	}
	if !writer.overflow || writer.body.Len() != 0 {
		t.Fatalf("超大响应缓存状态错误: overflow=%v size=%d", writer.overflow, writer.body.Len())
	}
}
