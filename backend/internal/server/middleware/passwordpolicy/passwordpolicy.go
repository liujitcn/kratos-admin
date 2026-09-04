package passwordpolicy

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/loginpolicy"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/password"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	coreconst "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-kit/auth"
	"github.com/liujitcn/kratos-kit/cache"
)

const passwordChangeRequiredMetadataKey = "password_change_required"

// NewMiddleware 创建强制改密拦截器。
// 密码过期或被管理员标记为必须修改时，仅允许加载登录后初始化信息、退出登录和提交改密请求。
func NewMiddleware(baseUserRepo *data.BaseUserRepository, policyCache cache.Cache) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			operation, ok := operationFromContext(ctx)
			if !ok || operation == "" {
				return handler(ctx, req)
			}
			authInfo, err := auth.FromContext(ctx)
			if err != nil || authInfo == nil || authInfo.UserId <= 0 {
				return handler(ctx, req)
			}
			// 密码策略仅针对后台账号；应用端没有后台改密接口，不能把应用用户锁死。
			if authInfo.RoleCode == coreconst.BASE_ROLE_CODE_USER || authInfo.RoleCode == coreconst.BASE_ROLE_CODE_AUTHUSER {
				return handler(ctx, req)
			}
			var user *models.BaseUser
			user, err = baseUserRepo.FindByID(ctx, authInfo.UserId)
			if err != nil {
				return nil, errorsx.Internal("读取密码策略状态失败").WithCause(err)
			}
			var policySet loginpolicy.PolicySet
			policySet, err = loginpolicy.LoadFromCacheStrict(policyCache)
			if err != nil {
				return nil, errorsx.Internal("读取密码策略配置失败").WithCause(err)
			}
			if user.MustChangePassword != _const.BASE_USER_PASSWORD_CHANGE_STATUS_REQUIRED && !password.IsExpiredAtWithMaxAge(user.PasswordChangedAt, time.Now(), policySet.PasswordMaxAgeDaysFor(user.TenantID, user.ID)) {
				return handler(ctx, req)
			}
			if passwordChangeOperation(operation) {
				return handler(ctx, req)
			}
			passwordError := errorsx.PermissionDenied("密码已过期，请先修改密码")
			metadata := make(map[string]string, len(passwordError.Metadata)+1)
			for key, value := range passwordError.Metadata {
				metadata[key] = value
			}
			metadata[passwordChangeRequiredMetadataKey] = "true"
			return nil, passwordError.WithMetadata(metadata)
		}
	}
}

// operationFromContext 读取当前服务端请求的 operation。
func operationFromContext(ctx context.Context) (string, bool) {
	value, ok := transport.FromServerContext(ctx)
	if !ok || value == nil || value.Operation() == "" {
		return "", false
	}
	return value.Operation(), true
}

// passwordChangeOperation 判断强制改密会话允许访问的管理端接口。
func passwordChangeOperation(operation string) bool {
	switch operation {
	case "/system.admin.v1.AuthService/TreeUserMenu",
		"/system.admin.v1.AuthService/ListUserButton",
		"/system.admin.v1.AuthService/GetUserInfo",
		"/system.admin.v1.AuthService/GetUserProfile",
		"/system.admin.v1.AuthService/UpdateUserPassword":
		return true
	case "/base.v1.LoginService/Logout":
		return true
	default:
		return false
	}
}
