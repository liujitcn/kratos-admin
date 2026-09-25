package oauth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/i18n"
	corei18n "github.com/liujitcn/kratos-core/resource/i18n"
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

// TestWriteOauthCryptoErrorLocalizesAllProtocolFailures 验证绕过 Kratos 路由的加解密错误使用请求语言。
func TestWriteOauthCryptoErrorLocalizesAllProtocolFailures(t *testing.T) {
	catalog, err := corei18n.NewI18n("oauth-crypto-error-test", i18n.Assets())
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		key    string
		status int
		want   map[string]string
	}{
		{
			key:    "system.oauth.crypto.error.client_token_invalid",
			status: http.StatusUnauthorized,
			want:   map[string]string{"zh-CN": "客户端访问令牌无效", "en-US": "The client access token is invalid", "zh-TW": "客戶端存取權杖無效", "ja-JP": "クライアントアクセストークンが無効です"},
		},
		{
			key:    "system.oauth.crypto.error.config_invalid",
			status: http.StatusUnauthorized,
			want:   map[string]string{"zh-CN": "客户端加密配置无效", "en-US": "The client encryption configuration is invalid", "zh-TW": "客戶端加密設定無效", "ja-JP": "クライアントの暗号化設定が無効です"},
		},
		{
			key:    "system.oauth.crypto.error.request_decrypt_failed",
			status: http.StatusBadRequest,
			want:   map[string]string{"zh-CN": "请求数据解密失败", "en-US": "Failed to decrypt the request data", "zh-TW": "請求資料解密失敗", "ja-JP": "リクエストデータの復号に失敗しました"},
		},
		{
			key:    "system.oauth.crypto.error.response_too_large",
			status: http.StatusRequestEntityTooLarge,
			want:   map[string]string{"zh-CN": "响应数据超过大小限制", "en-US": "The response exceeds the size limit", "zh-TW": "回應資料超過大小限制", "ja-JP": "レスポンスサイズが上限を超えています"},
		},
		{
			key:    "system.oauth.crypto.error.response_encrypt_failed",
			status: http.StatusInternalServerError,
			want:   map[string]string{"zh-CN": "响应数据加密失败", "en-US": "Failed to encrypt the response", "zh-TW": "回應資料加密失敗", "ja-JP": "レスポンスの暗号化に失敗しました"},
		},
	}
	for _, test := range tests {
		for locale, want := range test.want {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/oauth/test", nil)
			acceptLanguage := locale
			if locale == "en-US" {
				acceptLanguage = "en-US,en;q=0.9"
			}
			request.Header.Set("Accept-Language", acceptLanguage)
			writer := httptest.NewRecorder()
			writeOauthCryptoError(writer, request, test.status, catalog, test.key, "fallback")
			if writer.Code != test.status {
				t.Errorf("%s / %s: status = %d, want %d", test.key, locale, writer.Code, test.status)
				continue
			}
			var response struct {
				Code int    `json:"code"`
				Msg  string `json:"msg"`
			}
			if err = json.Unmarshal(writer.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.Msg != want {
				t.Errorf("%s / %s: message = %q, want %q", test.key, locale, response.Msg, want)
			}
		}
	}
}
