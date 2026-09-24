package admin

import (
	"context"
	"fmt"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/protobuf/types/known/emptypb"
)

// BaseOauthProviderService OAuth第三方登录方式管理服务。
type BaseOauthProviderService struct {
	adminv1.UnimplementedBaseOauthProviderServiceServer
	baseOauthProviderCase *biz.BaseOauthProviderCase
}

// NewBaseOauthProviderService 创建 OAuth 第三方登录方式管理服务。
func NewBaseOauthProviderService(baseOauthProviderCase *biz.BaseOauthProviderCase) *BaseOauthProviderService {
	return &BaseOauthProviderService{baseOauthProviderCase: baseOauthProviderCase}
}

// PageBaseOauthProvider 分页查询 OAuth 登录方式。
func (s *BaseOauthProviderService) PageBaseOauthProvider(ctx context.Context, req *adminv1.PageBaseOauthProviderRequest) (*adminv1.PageBaseOauthProviderResponse, error) {
	result, err := s.baseOauthProviderCase.PageBaseOauthProvider(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseOauthProvider %v", err))
		return nil, errorsx.WrapInternal(err, "查询OAuth登录方式失败")
	}
	return result, nil
}

// GetBaseOauthProvider 查询 OAuth 登录方式详情。
func (s *BaseOauthProviderService) GetBaseOauthProvider(ctx context.Context, req *adminv1.GetBaseOauthProviderRequest) (*adminv1.BaseOauthProviderForm, error) {
	result, err := s.baseOauthProviderCase.GetBaseOauthProvider(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseOauthProvider %v", err))
		return nil, errorsx.WrapInternal(err, "查询OAuth登录方式失败")
	}
	return result, nil
}

// CreateBaseOauthProvider 创建 OAuth 登录方式。
func (s *BaseOauthProviderService) CreateBaseOauthProvider(ctx context.Context, req *adminv1.CreateBaseOauthProviderRequest) (*emptypb.Empty, error) {
	err := s.baseOauthProviderCase.CreateBaseOauthProvider(ctx, req.GetBaseOauthProvider())
	if err != nil {
		log.Error(fmt.Sprintf("CreateBaseOauthProvider %v", err))
		return nil, errorsx.WrapInternal(err, "创建OAuth登录方式失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseOauthProvider 更新 OAuth 登录方式。
func (s *BaseOauthProviderService) UpdateBaseOauthProvider(ctx context.Context, req *adminv1.UpdateBaseOauthProviderRequest) (*emptypb.Empty, error) {
	err := s.baseOauthProviderCase.UpdateBaseOauthProvider(ctx, req.GetBaseOauthProvider())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseOauthProvider %v", err))
		return nil, errorsx.WrapInternal(err, "更新OAuth登录方式失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseOauthProvider 删除 OAuth 登录方式。
func (s *BaseOauthProviderService) DeleteBaseOauthProvider(ctx context.Context, req *adminv1.DeleteBaseOauthProviderRequest) (*emptypb.Empty, error) {
	err := s.baseOauthProviderCase.DeleteBaseOauthProvider(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBaseOauthProvider %v", err))
		return nil, errorsx.WrapInternal(err, "删除OAuth登录方式失败")
	}
	return new(emptypb.Empty), nil
}

// SetBaseOauthProviderStatus 设置 OAuth 登录方式状态。
func (s *BaseOauthProviderService) SetBaseOauthProviderStatus(ctx context.Context, req *adminv1.SetBaseOauthProviderStatusRequest) (*emptypb.Empty, error) {
	err := s.baseOauthProviderCase.SetBaseOauthProviderStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetBaseOauthProviderStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置OAuth登录方式状态失败")
	}
	return new(emptypb.Empty), nil
}
