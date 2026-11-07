package biz

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v3/errors"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	"github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

// TestRevealSecretFieldRequiresPlatformAdmin 验证明文出口在读取密钥前校验平台身份。
func TestRevealSecretFieldRequiresPlatformAdmin(t *testing.T) {
	for _, tc := range []struct {
		name string
		user *data.UserTokenPayload
		code int
	}{
		{name: "匿名", code: 401},
		{name: "其他租户管理员", user: &data.UserTokenPayload{UserId: 2, TenantId: 2, TenantCode: "tenant2", RoleCode: _const.BASE_ROLE_CODE_ADMIN}, code: 403},
		{name: "普通平台用户", user: &data.UserTokenPayload{UserId: 3, TenantId: 1, TenantCode: gorm.DefaultTenantCode, RoleCode: _const.BASE_ROLE_CODE_USER}, code: 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("未授权调用已进入密钥读取: %v", recovered)
				}
			}()
			ctx := context.Background()
			if tc.user != nil {
				ctx = engine.ContextWithAuthClaims(ctx, tc.user.MakeAuthClaims())
			}
			secretCase := &SecretCryptoCase{}
			_, err := secretCase.RevealSecretField(ctx, &basev1.RevealSecretFieldRequest{Resource: "ai_provider", Field: "api_key", Id: 1})
			if errors.Code(err) != tc.code {
				t.Fatalf("预期HTTP %d，实际 %v", tc.code, err)
			}
		})
	}
}
