package biz

import (
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

// TestIsBaseUserManagementRoleProtectedAllowsDefaultTenantReset 验证默认租户可以重置普通租户管理员密码。
func TestIsBaseUserManagementRoleProtectedAllowsDefaultTenantReset(t *testing.T) {
	tests := []struct {
		name string
		auth *data.UserTokenPayload
		role *models.BaseRole
		want bool
	}{
		{
			name: "default tenant ordinary tenant admin",
			auth: &data.UserTokenPayload{TenantId: 1, TenantCode: gorm.DefaultTenantCode},
			role: &models.BaseRole{TenantID: 1000, Code: _const.BASE_ROLE_CODE_TENANT},
		},
		{
			name: "default tenant own tenant admin",
			auth: &data.UserTokenPayload{TenantId: 1, TenantCode: gorm.DefaultTenantCode},
			role: &models.BaseRole{TenantID: 1, Code: _const.BASE_ROLE_CODE_TENANT},
			want: true,
		},
		{
			name: "ordinary tenant tenant admin",
			auth: &data.UserTokenPayload{TenantId: 1000, TenantCode: "1000"},
			role: &models.BaseRole{TenantID: 1000, Code: _const.BASE_ROLE_CODE_TENANT},
			want: true,
		},
		{
			name: "default tenant custom role",
			auth: &data.UserTokenPayload{TenantId: 1, TenantCode: gorm.DefaultTenantCode},
			role: &models.BaseRole{TenantID: 1000, Code: "custom"},
		},
		{
			name: "missing auth",
			role: &models.BaseRole{TenantID: 1000, Code: _const.BASE_ROLE_CODE_TENANT},
			want: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isBaseUserManagementRoleProtected(test.auth, test.role); got != test.want {
				t.Fatalf("isBaseUserManagementRoleProtected() = %v, want %v", got, test.want)
			}
		})
	}
}

// TestIsBaseUserDeletionProtectedKeepsTenantAdmin 验证内置 tenant 管理员账号始终不能删除。
func TestIsBaseUserDeletionProtectedKeepsTenantAdmin(t *testing.T) {
	if !isBaseUserDeletionProtected(&models.BaseRole{Code: _const.BASE_ROLE_CODE_TENANT}) {
		t.Fatal("tenant 管理员账号不应允许删除")
	}
	if isBaseUserDeletionProtected(&models.BaseRole{Code: "custom"}) {
		t.Fatal("普通角色账号应允许进入删除校验")
	}
}
