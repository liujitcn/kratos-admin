package biz

import (
	"context"
	"errors"

	"github.com/liujitcn/kratos-admin/backend/adapter/kit"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-core/biz"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-kit/database/gorm"
	secretcrypto "github.com/liujitcn/kratos-kit/secretcrypto"
)

// SecretCryptoCase 敏感字段加密用例。
type SecretCryptoCase struct {
	*biz.BaseCase
	secretCrypto *secretcrypto.Service
	resolver     *kit.RedactPolicyResolver
}

// NewSecretCryptoService 创建敏感字段临时密钥服务。
func NewSecretCryptoService(baseCase *biz.BaseCase) *secretcrypto.Service {
	return secretcrypto.NewService(baseCase.Cache)
}

// NewSecretCryptoCase 创建敏感字段加密用例。
func NewSecretCryptoCase(baseCase *biz.BaseCase, secretCrypto *secretcrypto.Service, resolver *kit.RedactPolicyResolver) *SecretCryptoCase {
	return &SecretCryptoCase{BaseCase: baseCase, secretCrypto: secretCrypto, resolver: resolver}
}

// GetSecretPublicKey 签发敏感字段一次性临时公钥。
func (c *SecretCryptoCase) GetSecretPublicKey() (*basev1.GetSecretPublicKeyResponse, error) {
	info, err := c.secretCrypto.GeneratePublicKey()
	if err != nil {
		return nil, errorsx.Internal("获取敏感字段临时公钥失败").WithCause(err)
	}
	return &basev1.GetSecretPublicKeyResponse{
		KeyId:     info.KeyID,
		PublicKey: info.PublicKey,
		Algorithm: info.Algorithm,
		Nonce:     info.Nonce,
		ExpiresIn: info.ExpiresIn,
	}, nil
}

// RevealSecretField 仅允许平台管理员显式查看已登记的供应商密钥，访问由统一审计留痕。
func (c *SecretCryptoCase) RevealSecretField(ctx context.Context, req *basev1.RevealSecretFieldRequest) (*basev1.RevealSecretFieldResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	if authInfo.UserId <= 0 || authInfo.TenantCode != gorm.DefaultTenantCode || (authInfo.RoleCode != _const.BASE_ROLE_CODE_SUPER && authInfo.RoleCode != _const.BASE_ROLE_CODE_ADMIN) {
		return nil, errorsx.PermissionDenied("只有平台管理员可以查看供应商密钥")
	}
	var text string
	text, err = c.resolver.RevealStorageField(ctx, req.GetResource(), req.GetField(), req.GetId())
	if err != nil {
		if errors.Is(err, kit.ErrUnprotectedStorageField) {
			return nil, errorsx.InvalidArgument("密钥字段不存在").WithCause(err)
		}
		return nil, errorsx.Internal("查看密钥字段失败").WithCause(err)
	}
	return &basev1.RevealSecretFieldResponse{Text: text}, nil
}
