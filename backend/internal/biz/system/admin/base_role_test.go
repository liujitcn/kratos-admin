package biz

import (
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

// TestIsBaseRoleProtectedAllowsDefaultTenantManageCopies 验证默认租户可以操作普通租户的 tenant 角色副本。
func TestIsBaseRoleProtectedAllowsDefaultTenantManageCopies(t *testing.T) {
	role := &models.BaseRole{TenantID: 1000, Code: _const.BASE_ROLE_CODE_TENANT}
	tests := []struct {
		name string
		auth *data.UserTokenPayload
		want bool
	}{
		{name: "default tenant", auth: &data.UserTokenPayload{TenantId: 1, TenantCode: gorm.DefaultTenantCode}},
		{name: "ordinary tenant", auth: &data.UserTokenPayload{TenantId: 1000, TenantCode: "1000"}, want: true},
		{name: "missing auth", want: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isBaseRoleProtected(test.auth, role); got != test.want {
				t.Fatalf("isBaseRoleProtected() = %v, want %v", got, test.want)
			}
		})
	}
}

// TestIsBaseRoleDeletionProtectedKeepsBuiltinRoles 验证内置角色始终不能删除。
func TestIsBaseRoleDeletionProtectedKeepsBuiltinRoles(t *testing.T) {
	tests := []struct {
		name string
		code string
		want bool
	}{
		{name: "super", code: _const.BASE_ROLE_CODE_SUPER, want: true},
		{name: "tenant", code: _const.BASE_ROLE_CODE_TENANT, want: true},
		{name: "admin", code: _const.BASE_ROLE_CODE_ADMIN, want: true},
		{name: "custom", code: "custom"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			role := &models.BaseRole{Code: test.code}
			if got := isBaseRoleDeletionProtected(role); got != test.want {
				t.Fatalf("isBaseRoleDeletionProtected() = %v, want %v", got, test.want)
			}
		})
	}
}
