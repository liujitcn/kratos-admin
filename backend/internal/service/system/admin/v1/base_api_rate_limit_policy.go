package admin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"
	"google.golang.org/protobuf/types/known/emptypb"
)

// BaseApiRateLimitPolicyService 提供接口限流策略管理接口。
type BaseApiRateLimitPolicyService struct {
	adminv1.UnimplementedBaseApiRateLimitPolicyServiceServer
	baseApiRateLimitPolicyCase *biz.BaseApiRateLimitPolicyCase
}

// NewBaseApiRateLimitPolicyService 创建接口限流策略服务。
func NewBaseApiRateLimitPolicyService(baseApiRateLimitPolicyCase *biz.BaseApiRateLimitPolicyCase) *BaseApiRateLimitPolicyService {
	return &BaseApiRateLimitPolicyService{baseApiRateLimitPolicyCase: baseApiRateLimitPolicyCase}
}

// PageBaseApiRateLimitPolicy 分页查询接口限流策略。
func (s *BaseApiRateLimitPolicyService) PageBaseApiRateLimitPolicy(ctx context.Context, req *adminv1.PageBaseApiRateLimitPolicyRequest) (*adminv1.PageBaseApiRateLimitPolicyResponse, error) {
	result, err := s.baseApiRateLimitPolicyCase.PageBaseApiRateLimitPolicy(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseApiRateLimitPolicy %v", err))
		return nil, errorsx.WrapInternal(err, "分页查询接口限流策略失败")
	}
	return result, nil
}

// GetBaseApiRateLimitPolicy 查询接口限流策略详情。
func (s *BaseApiRateLimitPolicyService) GetBaseApiRateLimitPolicy(ctx context.Context, req *adminv1.GetBaseApiRateLimitPolicyRequest) (*adminv1.BaseApiRateLimitPolicyForm, error) {
	result, err := s.baseApiRateLimitPolicyCase.GetBaseApiRateLimitPolicy(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseApiRateLimitPolicy %v", err))
		return nil, errorsx.WrapInternal(err, "查询接口限流策略详情失败")
	}
	return result, nil
}

// CreateBaseApiRateLimitPolicy 创建接口限流策略。
func (s *BaseApiRateLimitPolicyService) CreateBaseApiRateLimitPolicy(ctx context.Context, req *adminv1.CreateBaseApiRateLimitPolicyRequest) (*emptypb.Empty, error) {
	err := s.baseApiRateLimitPolicyCase.CreateBaseApiRateLimitPolicy(ctx, req.GetBaseApiRateLimitPolicy())
	if err != nil {
		log.Error(fmt.Sprintf("CreateBaseApiRateLimitPolicy %v", err))
		return nil, errorsx.WrapInternal(err, "创建接口限流策略失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseApiRateLimitPolicy 更新接口限流策略。
func (s *BaseApiRateLimitPolicyService) UpdateBaseApiRateLimitPolicy(ctx context.Context, req *adminv1.UpdateBaseApiRateLimitPolicyRequest) (*emptypb.Empty, error) {
	err := s.baseApiRateLimitPolicyCase.UpdateBaseApiRateLimitPolicy(ctx, req.GetBaseApiRateLimitPolicy())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseApiRateLimitPolicy %v", err))
		return nil, errorsx.WrapInternal(err, "更新接口限流策略失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseApiRateLimitPolicy 删除接口限流策略。
func (s *BaseApiRateLimitPolicyService) DeleteBaseApiRateLimitPolicy(ctx context.Context, req *adminv1.DeleteBaseApiRateLimitPolicyRequest) (*emptypb.Empty, error) {
	err := s.baseApiRateLimitPolicyCase.DeleteBaseApiRateLimitPolicy(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBaseApiRateLimitPolicy %v", err))
		return nil, errorsx.WrapInternal(err, "删除接口限流策略失败")
	}
	return new(emptypb.Empty), nil
}

// SetBaseApiRateLimitPolicyStatus 设置接口限流策略状态。
func (s *BaseApiRateLimitPolicyService) SetBaseApiRateLimitPolicyStatus(ctx context.Context, req *adminv1.SetBaseApiRateLimitPolicyStatusRequest) (*emptypb.Empty, error) {
	err := s.baseApiRateLimitPolicyCase.SetBaseApiRateLimitPolicyStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetBaseApiRateLimitPolicyStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置接口限流策略状态失败")
	}
	return new(emptypb.Empty), nil
}
