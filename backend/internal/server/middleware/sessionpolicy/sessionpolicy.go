package sessionpolicy

import (
	"context"
	"errors"
	"time"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/sessionstate"
	coreBiz "github.com/liujitcn/kratos-core/biz"
	coreconst "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-kit/auth"
	authData "github.com/liujitcn/kratos-kit/auth/data"
)

// NewMiddleware 创建服务端空闲超时和绝对生命周期拦截器。
func NewMiddleware(baseCase *coreBiz.BaseCase, userToken *authData.UserToken) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			authInfo, err := auth.FromContext(ctx)
			if err != nil || authInfo == nil || authInfo.UserId <= 0 {
				return handler(ctx, req)
			}
			if authInfo.RoleCode == coreconst.BASE_ROLE_CODE_USER || authInfo.RoleCode == coreconst.BASE_ROLE_CODE_AUTHUSER {
				return handler(ctx, req)
			}
			_, err = sessionstate.Touch(baseCase.Cache, authInfo.UserId, time.Now())
			if errors.Is(err, sessionstate.ErrStateNotFound) {
				_, err = sessionstate.Start(baseCase.Cache, authInfo.UserId, "", "", time.Now())
			}
			if err != nil {
				if errors.Is(err, sessionstate.ErrIdleExpired) || errors.Is(err, sessionstate.ErrMaxLifetimeExpired) {
					_ = userToken.RemoveToken(authInfo.UserId)
					_ = sessionstate.Clear(baseCase.Cache, authInfo.UserId)
					return nil, errorsx.Unauthenticated("会话已超时，请重新登录")
				}
				return nil, errorsx.Internal("校验会话状态失败").WithCause(err)
			}
			return handler(ctx, req)
		}
	}
}
