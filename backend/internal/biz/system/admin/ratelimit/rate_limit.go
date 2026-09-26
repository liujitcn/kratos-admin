package ratelimit

import (
	"context"

	"github.com/liujitcn/kratos-admin/backend/adapter/kit"
	"github.com/liujitcn/kratos-core/biz"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

// EnsureRateLimitPlatformOperator 校验当前操作者具备平台级限流管理权限。
func EnsureRateLimitPlatformOperator(ctx context.Context, baseCase *biz.BaseCase) error {
	authInfo, err := baseCase.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	if (authInfo.RoleCode != _const.BASE_ROLE_CODE_SUPER && authInfo.RoleCode != _const.BASE_ROLE_CODE_ADMIN) || authInfo.TenantCode != gorm.DefaultTenantCode {
		return errorsx.PermissionDenied("只有平台管理员可以管理限流策略")
	}
	return nil
}

// RefreshRateLimitRuntime 刷新限流运行时策略快照。
func RefreshRateLimitRuntime(ctx context.Context, resolver *kit.RateLimitPolicyResolver) error {
	if resolver == nil {
		return nil
	}
	return resolver.Refresh(ctx)
}
