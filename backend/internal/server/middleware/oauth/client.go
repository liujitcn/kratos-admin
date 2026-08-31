package oauth

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-kit/auth"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"gorm.io/gorm"
)

const oauthDevelopmentPathPrefix = "/api/v1/oauth/"

// NewClientMiddleware 创建开放授权客户端 operation 拦截器。
//
// 拦截器只依赖客户端仓储和 API Case，不注入完整的开放授权业务 Case。
func NewClientMiddleware(clientRepo *data.OauthClientRepository, baseAPICase *biz.BaseAPICase) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			request, ok := kratosHTTP.RequestFromServerContext(ctx)
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
			ctx = withResolvedOauthClient(ctx, client)
			if err = authorizeOauthClientOperation(ctx, baseAPICase, client, request.Method); err != nil {
				return nil, err
			}
			return handler(ctx, req)
		}
	}
}

// resolveOauthClientRequest 解析并校验当前请求对应的开放授权客户端。
func resolveOauthClientRequest(ctx context.Context, clientRepo *data.OauthClientRepository) (*models.OauthClient, bool, error) {
	var authInfo *authData.UserTokenPayload
	var err error
	authInfo, err = auth.FromContext(ctx)
	if err != nil || authInfo.RoleCode == "" {
		return nil, false, nil
	}
	query := clientRepo.Query(ctx).OauthClient
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ClientID.Eq(authInfo.RoleCode)))
	var client *models.OauthClient
	client, err = clientRepo.Find(ctx, opts...)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if authInfo.UserId < 0 {
				return nil, true, errorsx.PermissionDenied("客户端不属于当前租户")
			}
			return nil, false, nil
		}
		return nil, true, errorsx.Internal("读取开放授权客户端配置失败").WithCause(err)
	}
	if authInfo.UserId != -client.ID {
		return nil, true, errorsx.PermissionDenied("客户端令牌身份不匹配")
	}
	if client.Status != int32(_const.STATUS_STATUS_ENABLE) {
		return nil, true, errorsx.PermissionDenied("客户端已停用")
	}
	return client, true, nil
}

// authorizeOauthClientOperation 校验当前 HTTP operation 是否在客户端 API 白名单中。
func authorizeOauthClientOperation(ctx context.Context, baseAPICase *biz.BaseAPICase, client *models.OauthClient, method string) error {
	operations := _string.ConvertJsonStringToStringArray(client.API)
	if len(operations) == 0 {
		return errorsx.PermissionDenied("客户端未授权任何接口")
	}
	apiQuery := baseAPICase.Query(ctx).BaseAPI
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(apiQuery.Operation.In(operations...)))
	apiList, err := baseAPICase.List(ctx, opts...)
	if err != nil {
		return errorsx.Internal("读取客户端授权接口失败").WithCause(err)
	}
	operation, ok := transport.FromServerContext(ctx)
	if !ok || operation.Operation() == "" {
		return errorsx.PermissionDenied("无法识别当前开发授权接口")
	}
	for _, api := range apiList {
		if !isOauthDevelopmentAPI(api.Path) {
			return errorsx.InvalidArgument("只能授权开发授权接口")
		}
		if api.Operation == operation.Operation() && strings.EqualFold(api.Method, method) {
			return nil
		}
	}
	return errorsx.PermissionDenied("客户端未授权当前开发接口")
}

// isOauthDevelopmentAPI 判断接口是否属于外部开发授权路由。
func isOauthDevelopmentAPI(path string) bool {
	return strings.HasPrefix(path, oauthDevelopmentPathPrefix)
}

// oauthClientRemoteIP 读取 TCP 对端地址，不信任未经代理配置的转发头。
func oauthClientRemoteIP(request *http.Request) string {
	if host, _, err := net.SplitHostPort(request.RemoteAddr); err == nil {
		return host
	}
	return request.RemoteAddr
}
