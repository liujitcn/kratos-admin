package utils

import (
	"net/http"
	"os"
	"strings"
)

// trustProxy 判断部署是否在可信反向代理之后终止 TLS。
// 通过环境变量 SERVER_TRUST_PROXY=true 开启；默认关闭，避免直接暴露在公网时被伪造 X-Forwarded-Proto 绕过。
func trustProxy() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("SERVER_TRUST_PROXY")))
	return value == "true" || value == "1" || value == "yes" || value == "on"
}

// IsSecureRequest 判断请求是否应视为 HTTPS：直接 TLS，或可信代理上报的 X-Forwarded-Proto 为 https。
func IsSecureRequest(request *http.Request) bool {
	if request == nil {
		return false
	}
	if request.TLS != nil {
		return true
	}
	return trustProxy() && strings.EqualFold(request.Header.Get("X-Forwarded-Proto"), "https")
}
