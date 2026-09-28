package admin

import (
	"context"
	"fmt"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/protobuf/types/known/emptypb"
)

// BaseMessageProviderService 消息发送 Provider 管理服务。
type BaseMessageProviderService struct {
	adminv1.UnimplementedBaseMessageProviderServiceServer
	baseMessageProviderCase *biz.BaseMessageProviderCase
}

// NewBaseMessageProviderService 创建消息发送 Provider 管理服务。
func NewBaseMessageProviderService(baseMessageProviderCase *biz.BaseMessageProviderCase) *BaseMessageProviderService {
	return &BaseMessageProviderService{baseMessageProviderCase: baseMessageProviderCase}
}

// OptionBaseMessageProvider 查询消息发送 Provider 选项。
func (s *BaseMessageProviderService) OptionBaseMessageProvider(ctx context.Context, req *adminv1.OptionBaseMessageProviderRequest) (*commonv1.SelectOptionResponse, error) {
	result, err := s.baseMessageProviderCase.OptionBaseMessageProvider(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("OptionBaseMessageProvider %v", err))
		return nil, errorsx.WrapInternal(err, "查询消息 Provider 选项失败")
	}
	return result, nil
}

// PageBaseMessageProvider 分页查询消息发送 Provider。
func (s *BaseMessageProviderService) PageBaseMessageProvider(ctx context.Context, req *adminv1.PageBaseMessageProviderRequest) (*adminv1.PageBaseMessageProviderResponse, error) {
	result, err := s.baseMessageProviderCase.PageBaseMessageProvider(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseMessageProvider %v", err))
		return nil, errorsx.WrapInternal(err, "查询消息 Provider 失败")
	}
	return result, nil
}

// GetBaseMessageProvider 查询消息发送 Provider 详情。
func (s *BaseMessageProviderService) GetBaseMessageProvider(ctx context.Context, req *adminv1.GetBaseMessageProviderRequest) (*adminv1.BaseMessageProviderForm, error) {
	result, err := s.baseMessageProviderCase.GetBaseMessageProvider(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseMessageProvider %v", err))
		return nil, errorsx.WrapInternal(err, "查询消息 Provider 失败")
	}
	return result, nil
}

// CreateBaseMessageProvider 创建消息发送 Provider。
func (s *BaseMessageProviderService) CreateBaseMessageProvider(ctx context.Context, req *adminv1.CreateBaseMessageProviderRequest) (*emptypb.Empty, error) {
	err := s.baseMessageProviderCase.CreateBaseMessageProvider(ctx, req.GetBaseMessageProvider())
	if err != nil {
		log.Error(fmt.Sprintf("CreateBaseMessageProvider %v", err))
		return nil, errorsx.WrapInternal(err, "创建消息 Provider 失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseMessageProvider 更新消息发送 Provider。
func (s *BaseMessageProviderService) UpdateBaseMessageProvider(ctx context.Context, req *adminv1.UpdateBaseMessageProviderRequest) (*emptypb.Empty, error) {
	err := s.baseMessageProviderCase.UpdateBaseMessageProvider(ctx, req.GetBaseMessageProvider())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseMessageProvider %v", err))
		return nil, errorsx.WrapInternal(err, "更新消息 Provider 失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseMessageProvider 删除消息发送 Provider。
func (s *BaseMessageProviderService) DeleteBaseMessageProvider(ctx context.Context, req *adminv1.DeleteBaseMessageProviderRequest) (*emptypb.Empty, error) {
	err := s.baseMessageProviderCase.DeleteBaseMessageProvider(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBaseMessageProvider %v", err))
		return nil, errorsx.WrapInternal(err, "删除消息 Provider 失败")
	}
	return new(emptypb.Empty), nil
}

// SetBaseMessageProviderStatus 设置消息发送 Provider 状态。
func (s *BaseMessageProviderService) SetBaseMessageProviderStatus(ctx context.Context, req *adminv1.SetBaseMessageProviderStatusRequest) (*emptypb.Empty, error) {
	err := s.baseMessageProviderCase.SetBaseMessageProviderStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetBaseMessageProviderStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置消息 Provider 状态失败")
	}
	return new(emptypb.Empty), nil
}
