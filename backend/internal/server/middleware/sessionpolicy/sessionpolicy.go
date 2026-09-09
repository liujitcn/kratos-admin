package sessionpolicy

import (
	"context"
	"errors"
	"time"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/sessionregistry"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/sessionstate"
	"github.com/liujitcn/kratos-core/biz"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-kit/auth"
	"github.com/liujitcn/kratos-kit/auth/data"
)

// NewMiddleware 创建服务端空闲超时和绝对生命周期拦截器。
func NewMiddleware(baseCase *biz.BaseCase, userToken *data.UserToken) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			authInfo, err := auth.FromContext(ctx)
			if err != nil || authInfo == nil || authInfo.UserId <= 0 {
				return handler(ctx, req)
			}
			var record sessionregistry.Record
			record, err = sessionregistry.FindByAccessToken(baseCase.Cache, userToken, authInfo.UserId, sessionregistry.AccessToken(ctx))
			if err != nil {
				return nil, errorsx.Unauthenticated("当前会话已失效").WithCause(err)
			}
			if authInfo.RoleCode == _const.BASE_ROLE_CODE_USER || authInfo.RoleCode == _const.BASE_ROLE_CODE_AUTHUSER {
				return handler(ctx, req)
			}
			_, err = sessionstate.Touch(baseCase.Cache, record.SessionID, time.Now())
			if err != nil {
				if errors.Is(err, sessionstate.ErrStateNotFound) || errors.Is(err, sessionstate.ErrIdleExpired) || errors.Is(err, sessionstate.ErrMaxLifetimeExpired) {
					if err = sessionregistry.Remove(baseCase.Cache, userToken, record); err != nil {
						return nil, errorsx.Internal("撤销过期会话失败").WithCause(err)
					}
					return nil, errorsx.Unauthenticated("会话已超时，请重新登录")
				}
				return nil, errorsx.Internal("校验会话状态失败").WithCause(err)
			}
			return handler(ctx, req)
		}
	}
}
