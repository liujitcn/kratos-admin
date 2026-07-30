package middleware

import (
	"context"
	"errors"

	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"

	"buf.build/go/protovalidate"
	kratosMiddleware "github.com/go-kratos/kratos/v3/middleware"
	"google.golang.org/protobuf/proto"
)

// NewValidateMiddleware 创建基于 Proto 声明的请求参数校验中间件。
func NewValidateMiddleware() kratosMiddleware.Middleware {
	return func(handler kratosMiddleware.Handler) kratosMiddleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			if message, ok := req.(proto.Message); ok {
				if err := protovalidate.Validate(message); err != nil {
					return nil, validationError(err)
				}
			}
			return handler(ctx, req)
		}
	}
}

// validationError 将 protovalidate 的首条违规转换为统一业务错误。
func validationError(err error) error {
	if validationErr, ok := errors.AsType[*protovalidate.ValidationError](err); ok && len(validationErr.Violations) > 0 {
		if message := validationErr.Violations[0].Proto.GetMessage(); message != "" {
			return errorsx.InvalidArgument(message)
		}
	}
	return errorsx.InvalidArgument(err.Error()).WithCause(err)
}
