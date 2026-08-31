package oauth

import (
	"context"
	"net"
	"strings"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport/http"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/errorsx"
)

// oauthClientContextKey 是上下文中已解析 OAuth 客户端对象的私有键类型。
type oauthClientContextKey struct{}

// NewIPMiddleware 创建开放授权客户端 IP 白名单拦截器。
//
// 客户端身份由前置认证链路解析，仓储负责读取客户端配置；本拦截器只处理
// 请求来源地址与客户端白名单的匹配，并将已解析客户端传给后续 OAuth middleware。
func NewIPMiddleware(clientRepo *data.OauthClientRepository) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			request, ok := http.RequestFromServerContext(ctx)
			if !ok || request == nil {
				return handler(ctx, req)
			}
			if !isOauthDevelopmentAPI(request.URL.Path) {
				return handler(ctx, req)
			}
			var client *models.OauthClient
			client, ok = resolvedOauthClient(ctx)
			var err error
			if !ok {
				client, ok, err = resolveOauthClientRequest(ctx, clientRepo)
			}
			if err != nil {
				return nil, err
			}
			if !ok {
				return handler(ctx, req)
			}
			if !MatchClientIP(oauthClientRemoteIP(request), client.IPWhitelist) {
				return nil, errorsx.PermissionDenied("客户端来源 IP 未授权")
			}
			return handler(withResolvedOauthClient(ctx, client), req)
		}
	}
}

// MatchClientIP 判断客户端地址是否命中 IP 白名单。
func MatchClientIP(ip string, whitelist string) bool {
	if strings.TrimSpace(whitelist) == "" {
		return true
	}
	parsedIP := net.ParseIP(strings.TrimSpace(ip))
	if parsedIP == nil {
		return false
	}
	for _, raw := range strings.Split(whitelist, ",") {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			continue
		}
		if strings.Contains(entry, "/") {
			_, network, parseErr := net.ParseCIDR(entry)
			if parseErr == nil && network.Contains(parsedIP) {
				return true
			}
			continue
		}
		if parsedIP.Equal(net.ParseIP(entry)) {
			return true
		}
	}
	return false
}

// withResolvedOauthClient 将已解析客户端放入当前请求上下文，避免重复查询。
func withResolvedOauthClient(ctx context.Context, client *models.OauthClient) context.Context {
	return context.WithValue(ctx, oauthClientContextKey{}, client)
}

// resolvedOauthClient 从请求上下文读取已解析的客户端。
func resolvedOauthClient(ctx context.Context) (*models.OauthClient, bool) {
	client, ok := ctx.Value(oauthClientContextKey{}).(*models.OauthClient)
	return client, ok && client != nil
}
