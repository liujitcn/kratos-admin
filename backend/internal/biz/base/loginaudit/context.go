package loginaudit

import (
	"context"

	"github.com/liujitcn/kratos-kit/auth/data"
)

type identityKey struct{}

// identityCapture 保存同一次登录请求中确认的身份，供外层审计读取。
type identityCapture struct {
	identity *data.UserTokenPayload
}

// WithCapture 创建请求范围内的登录身份接收器。
func WithCapture(ctx context.Context) context.Context {
	return context.WithValue(ctx, identityKey{}, &identityCapture{})
}

// Record 在成功签发令牌后记录可信身份，不保存访问令牌或刷新令牌。
func Record(ctx context.Context, identity *data.UserTokenPayload) {
	if capture, ok := ctx.Value(identityKey{}).(*identityCapture); ok {
		value := *identity
		capture.identity = &value
	}
}

// FromContext 返回当前登录请求已确认的身份。
func FromContext(ctx context.Context) *data.UserTokenPayload {
	if capture, ok := ctx.Value(identityKey{}).(*identityCapture); ok {
		return capture.identity
	}
	return nil
}
