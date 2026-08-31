package base

import (
	"context"
	"fmt"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/protobuf/types/known/emptypb"
)

// MfaService 多因素认证公共服务。
type MfaService struct {
	basev1.UnimplementedMfaServiceServer
	loginCase *biz.LoginCase
	mfaCase   *biz.MfaCase
}

// NewMfaService 创建多因素认证公共服务。
func NewMfaService(loginCase *biz.LoginCase, mfaCase *biz.MfaCase) *MfaService {
	return &MfaService{loginCase: loginCase, mfaCase: mfaCase}
}

// VerifyMfa 校验登录阶段的多因素认证。
func (s *MfaService) VerifyMfa(ctx context.Context, req *basev1.VerifyMfaRequest) (*basev1.LoginResponse, error) {
	res, err := s.loginCase.VerifyMfa(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("VerifyMfa %v", err))
		return nil, errorsx.WrapInternal(err, "校验多因素认证失败")
	}
	setRefreshTokenCookie(ctx, res.GetRefreshToken(), s.loginCase.RefreshTokenExpiresIn())
	return res, nil
}

// BeginMfaEnrollment 开始登录阶段的强制 MFA 绑定。
func (s *MfaService) BeginMfaEnrollment(ctx context.Context, req *basev1.BeginMfaEnrollmentRequest) (*basev1.BeginMfaEnrollmentResponse, error) {
	res, err := s.loginCase.BeginMfaEnrollment(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("BeginMfaEnrollment %v", err))
		return nil, errorsx.WrapInternal(err, "开始强制绑定多因素认证失败")
	}
	return res, nil
}

// ConfirmMfaEnrollment 确认登录阶段的强制 MFA 绑定。
func (s *MfaService) ConfirmMfaEnrollment(ctx context.Context, req *basev1.ConfirmMfaEnrollmentRequest) (*basev1.ConfirmMfaEnrollmentResponse, error) {
	res, err := s.loginCase.ConfirmMfaEnrollment(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("ConfirmMfaEnrollment %v", err))
		return nil, errorsx.WrapInternal(err, "确认强制绑定多因素认证失败")
	}
	return res, nil
}

// GetMfaStatus 查询当前用户多因素认证状态。
func (s *MfaService) GetMfaStatus(ctx context.Context, req *basev1.GetMfaStatusRequest) (*basev1.MfaStatusResponse, error) {
	res, err := s.mfaCase.GetMfaStatus(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("GetMfaStatus %v", err))
		return nil, errorsx.WrapInternal(err, "查询多因素认证状态失败")
	}
	return res, nil
}

// BeginMfaSetup 开始绑定当前用户的多因素认证方式。
func (s *MfaService) BeginMfaSetup(ctx context.Context, req *basev1.BeginMfaSetupRequest) (*basev1.BeginMfaSetupResponse, error) {
	res, err := s.mfaCase.BeginMfaSetup(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("BeginMfaSetup %v", err))
		return nil, errorsx.WrapInternal(err, "开始绑定多因素认证失败")
	}
	return res, nil
}

// ConfirmMfaSetup 确认绑定当前用户的多因素认证方式。
func (s *MfaService) ConfirmMfaSetup(ctx context.Context, req *basev1.ConfirmMfaSetupRequest) (*basev1.ConfirmMfaSetupResponse, error) {
	res, err := s.mfaCase.ConfirmMfaSetup(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("ConfirmMfaSetup %v", err))
		return nil, errorsx.WrapInternal(err, "确认绑定多因素认证失败")
	}
	return res, nil
}

// BeginMfaDisable 开始禁用 WebAuthn 多因素认证的 Passkey 验证。
func (s *MfaService) BeginMfaDisable(ctx context.Context, req *basev1.BeginMfaDisableRequest) (*basev1.BeginMfaDisableResponse, error) {
	res, err := s.mfaCase.BeginMfaDisable(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("BeginMfaDisable %v", err))
		return nil, errorsx.WrapInternal(err, "开始禁用多因素认证验证失败")
	}
	return res, nil
}

// DisableMfa 禁用当前用户的多因素认证。
func (s *MfaService) DisableMfa(ctx context.Context, req *basev1.DisableMfaRequest) (*emptypb.Empty, error) {
	err := s.mfaCase.DisableMfa(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("DisableMfa %v", err))
		return nil, errorsx.WrapInternal(err, "禁用多因素认证失败")
	}
	return &emptypb.Empty{}, nil
}

// RegenerateMfaRecoveryCodes 重新生成当前用户的多因素认证恢复码。
func (s *MfaService) RegenerateMfaRecoveryCodes(ctx context.Context, req *basev1.RegenerateMfaRecoveryCodesRequest) (*basev1.RecoveryCodesResponse, error) {
	res, err := s.mfaCase.RegenerateRecoveryCodes(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("RegenerateMfaRecoveryCodes %v", err))
		return nil, errorsx.WrapInternal(err, "重新生成多因素认证恢复码失败")
	}
	return res, nil
}
