package middleware

import (
	"context"

	coreI18n "github.com/liujitcn/kratos-admin/backend/core/pkg/i18n"
	coreLocale "github.com/liujitcn/kratos-admin/backend/core/pkg/locale"

	kratosMiddleware "github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
)

// NewLocaleMiddleware 创建统一的请求语言解析和错误本地化中间件。
func NewLocaleMiddleware() kratosMiddleware.Middleware {
	return func(handler kratosMiddleware.Handler) kratosMiddleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			localeValue := coreLocale.Default
			if transporter, ok := transport.FromServerContext(ctx); ok {
				localeValue = coreLocale.ResolveAcceptLanguage(transporter.RequestHeader().Get("Accept-Language"))
			}
			ctx = coreLocale.WithContext(ctx, localeValue)
			reply, err := handler(ctx, req)
			if err != nil {
				return nil, coreI18n.LocalizeError(defaultCatalog, localeValue, err)
			}
			return reply, nil
		}
	}
}
