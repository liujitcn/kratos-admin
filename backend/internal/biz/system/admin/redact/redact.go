package redact

import (
	"context"

	"github.com/liujitcn/kratos-admin/backend/adapter/kit"
	"github.com/liujitcn/kratos-core/biz"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

// EnsureRedactPlatformOperator 校验当前操作者具备平台级脱敏管理权限。
func EnsureRedactPlatformOperator(ctx context.Context, baseCase *biz.BaseCase) error {
	authInfo, err := baseCase.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	if (authInfo.RoleCode != _const.BASE_ROLE_CODE_SUPER && authInfo.RoleCode != _const.BASE_ROLE_CODE_ADMIN) || authInfo.TenantCode != gorm.DefaultTenantCode {
		return errorsx.PermissionDenied("只有平台管理员可以管理脱敏策略")
	}
	return nil
}

// RefreshRedactRuntime 刷新脱敏运行时缓存。
func RefreshRedactRuntime(ctx context.Context, resolver *kit.RedactPolicyResolver) error {
	if resolver == nil {
		return nil
	}
	return resolver.Refresh(ctx)
}
