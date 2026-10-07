package biz

import (
	"context"
	"errors"

	"github.com/liujitcn/kratos-admin/backend/adapter/kit"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	secretcrypto "github.com/liujitcn/kratos-kit/secretcrypto"
)

// SecretCryptoCase 敏感字段加密用例。
type SecretCryptoCase struct {
	secretCrypto *secretcrypto.Service
	resolver     *kit.RedactPolicyResolver
}

// NewSecretCryptoService 创建敏感字段临时密钥服务。
func NewSecretCryptoService(baseCase *biz.BaseCase) *secretcrypto.Service {
	return secretcrypto.NewService(baseCase.Cache)
}

// NewSecretCryptoCase 创建敏感字段加密用例。
func NewSecretCryptoCase(secretCrypto *secretcrypto.Service, resolver *kit.RedactPolicyResolver) *SecretCryptoCase {
	return &SecretCryptoCase{secretCrypto: secretCrypto, resolver: resolver}
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

// RevealSecretField 解密查看指定记录的密钥字段明文。
func (c *SecretCryptoCase) RevealSecretField(ctx context.Context, req *basev1.RevealSecretFieldRequest) (*basev1.RevealSecretFieldResponse, error) {
	var text string
	var err error
	text, err = c.resolver.RevealStorageField(ctx, req.GetResource(), req.GetField(), req.GetId())
	if err != nil {
		if errors.Is(err, kit.ErrUnprotectedStorageField) {
			return nil, errorsx.InvalidArgument("密钥字段不存在").WithCause(err)
		}
		return nil, errorsx.Internal("查看密钥字段失败").WithCause(err)
	}
	return &basev1.RevealSecretFieldResponse{Text: text}, nil
}
