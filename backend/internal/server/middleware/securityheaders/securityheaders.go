package securityheaders

import (
	"net/http"

	"github.com/liujitcn/kratos-admin/backend/internal/utils"
)

const contentSecurityPolicy = "script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob: https:; font-src 'self' data:; connect-src 'self' https: wss:; base-uri 'self'; object-src 'none'; frame-ancestors 'none'; form-action 'self'"

// NewHandler 为静态资源、API 和错误响应统一增加安全响应头。
func NewHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Security-Policy", contentSecurityPolicy)
		writer.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("X-Frame-Options", "DENY")
		if utils.IsSecureRequest(request) {
			writer.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}
		next.ServeHTTP(writer, request)
	})
}
