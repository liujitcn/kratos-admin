package utils

import (
	"net/http"

	"github.com/liujitcn/kratos-admin/backend/pkg/securityconfig"
)

// IsSecureRequest 判断请求是否应视为 HTTPS：直接 TLS，或可信代理上报的 X-Forwarded-Proto 为 https。
func IsSecureRequest(request *http.Request) bool {
	return securityconfig.Current().IsSecureRequest(request)
}
