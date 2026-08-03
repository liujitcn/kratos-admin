package base

import (
	"context"
	"fmt"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base/v1"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/protobuf/types/known/emptypb"
)

// LoginService 登录公共服务
type LoginService struct {
	basev1.UnimplementedLoginServiceServer
	loginCase *biz.LoginCase
}

// NewLoginService 创建登录公共服务
func NewLoginService(
	loginCase *biz.LoginCase,
) *LoginService {
	var ss = LoginService{
		loginCase: loginCase,
	}

	return &ss
}

// Captcha 验证码
func (s *LoginService) Captcha(ctx context.Context, req *basev1.CaptchaRequest) (*basev1.CaptchaResponse, error) {
	res, err := s.loginCase.Captcha(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("Captcha %v", err))
		return nil, errorsx.WrapInternal(err, "获取验证码失败")
	}
	return res, nil
}

// VerifyCaptcha 验证码预校验
func (s *LoginService) VerifyCaptcha(ctx context.Context, req *basev1.VerifyCaptchaRequest) (*basev1.VerifyCaptchaResponse, error) {
	res, err := s.loginCase.VerifyCaptcha(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("VerifyCaptcha %v", err))
		return nil, errorsx.WrapInternal(err, "验证码预校验失败")
	}
	return res, nil
}

// PasswordPublicKey 获取密码临时公钥
func (s *LoginService) PasswordPublicKey(ctx context.Context, req *basev1.PasswordPublicKeyRequest) (*basev1.PasswordPublicKeyResponse, error) {
	res, err := s.loginCase.PasswordPublicKey(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PasswordPublicKey %v", err))
		return nil, errorsx.WrapInternal(err, "获取密码临时公钥失败")
	}
	return res, nil
}

// Logout 登出
func (s *LoginService) Logout(ctx context.Context, req *basev1.LogoutRequest) (*emptypb.Empty, error) {
	err := s.loginCase.Logout(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("Logout %v", err))
		return nil, errorsx.WrapInternal(err, "退出登录失败")
	}
	return &emptypb.Empty{}, nil
}

// RefreshToken 刷新认证令牌
func (s *LoginService) RefreshToken(ctx context.Context, req *basev1.RefreshTokenRequest) (*basev1.RefreshTokenResponse, error) {
	res, err := s.loginCase.RefreshToken(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("RefreshToken %v", err))
		return nil, errorsx.WrapInternal(err, "刷新认证令牌失败")
	}
	return res, nil
}

// Login 登录
func (s *LoginService) Login(ctx context.Context, req *basev1.LoginRequest) (*basev1.LoginResponse, error) {
	res, err := s.loginCase.Login(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("Login %v", err))
		return nil, errorsx.WrapInternal(err, "登录失败")
	}
	return res, nil
}
