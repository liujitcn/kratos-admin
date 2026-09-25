package conflictmessage

import (
	"context"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	"github.com/liujitcn/kratos-core/errorsx"
)

// NewConflictMessageKeyMiddleware 为冲突错误补充原始文案键，并在缺少资源时附加接口路径。
func NewConflictMessageKeyMiddleware() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, request any) (any, error) {
			reply, err := handler(ctx, request)
			if err == nil {
				return reply, nil
			}
			structured := errors.FromError(err)
			if structured == nil || structured.Reason != errorsx.ReasonConflict || structured.Message == "" {
				return reply, err
			}
			localized := structured
			if localized.Metadata[errorsx.METADATA_KEY_MESSAGE_KEY] == "" {
				localized = errorsx.WithMessageKey(localized, errorsx.MessageKey(localized.Message), nil)
			}
			if localized.Metadata[errorsx.METADATA_KEY_RESOURCE] == "" && localized.Metadata[errorsx.METADATA_KEY_CHILD_RESOURCE] == "" && localized.Metadata["operation"] == "" {
				if transporter, ok := transport.FromServerContext(ctx); ok && transporter.Operation() != "" {
					metadata := make(map[string]string, len(localized.Metadata)+1)
					for key, value := range localized.Metadata {
						metadata[key] = value
					}
					metadata["operation"] = transporter.Operation()
					localized = localized.WithMetadata(metadata)
				}
			}
			return reply, localized.WithCause(err)
		}
	}
}
