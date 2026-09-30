package base

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-core/errorsx"
)

// SecretCryptoService 敏感字段加密公共服务
type SecretCryptoService struct {
	basev1.UnimplementedSecretCryptoServiceServer
	secretCryptoCase *biz.SecretCryptoCase
}

// NewSecretCryptoService 创建敏感字段加密公共服务
func NewSecretCryptoService(secretCryptoCase *biz.SecretCryptoCase) *SecretCryptoService {
	return &SecretCryptoService{secretCryptoCase: secretCryptoCase}
}

// GetSecretPublicKey 获取敏感字段临时公钥
func (s *SecretCryptoService) GetSecretPublicKey(ctx context.Context, req *basev1.GetSecretPublicKeyRequest) (*basev1.GetSecretPublicKeyResponse, error) {
	res, err := s.secretCryptoCase.GetSecretPublicKey()
	if err != nil {
		log.Error(fmt.Sprintf("GetSecretPublicKey %v", err))
		return nil, errorsx.WrapInternal(err, "获取敏感字段临时公钥失败")
	}
	return res, nil
}

// RevealSecretField 查看密钥字段明文
func (s *SecretCryptoService) RevealSecretField(ctx context.Context, req *basev1.RevealSecretFieldRequest) (*basev1.RevealSecretFieldResponse, error) {
	res, err := s.secretCryptoCase.RevealSecretField(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("RevealSecretField %v", err))
		return nil, errorsx.WrapInternal(err, "查看密钥字段失败")
	}
	return res, nil
}
