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

// BaseRedactOutputPolicyService 提供出库脱敏策略管理接口。
type BaseRedactOutputPolicyService struct {
	adminv1.UnimplementedBaseRedactOutputPolicyServiceServer
	baseRedactOutputPolicyCase *biz.BaseRedactOutputPolicyCase
}

// NewBaseRedactOutputPolicyService 创建出库脱敏策略服务。
func NewBaseRedactOutputPolicyService(baseRedactOutputPolicyCase *biz.BaseRedactOutputPolicyCase) *BaseRedactOutputPolicyService {
	return &BaseRedactOutputPolicyService{baseRedactOutputPolicyCase: baseRedactOutputPolicyCase}
}

// PageBaseRedactOutputPolicy 分页查询出库脱敏策略。
func (s *BaseRedactOutputPolicyService) PageBaseRedactOutputPolicy(ctx context.Context, req *adminv1.PageBaseRedactOutputPolicyRequest) (*adminv1.PageBaseRedactOutputPolicyResponse, error) {
	result, err := s.baseRedactOutputPolicyCase.PageBaseRedactOutputPolicy(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseRedactOutputPolicy %v", err))
		return nil, errorsx.WrapInternal(err, "分页查询出库脱敏策略失败")
	}
	return result, nil
}

// GetBaseRedactOutputPolicy 查询出库脱敏策略详情。
func (s *BaseRedactOutputPolicyService) GetBaseRedactOutputPolicy(ctx context.Context, req *adminv1.GetBaseRedactOutputPolicyRequest) (*adminv1.BaseRedactOutputPolicyForm, error) {
	result, err := s.baseRedactOutputPolicyCase.GetBaseRedactOutputPolicy(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseRedactOutputPolicy %v", err))
		return nil, errorsx.WrapInternal(err, "查询出库脱敏策略详情失败")
	}
	return result, nil
}

// CreateBaseRedactOutputPolicy 批量创建出库脱敏策略。
func (s *BaseRedactOutputPolicyService) CreateBaseRedactOutputPolicy(ctx context.Context, req *adminv1.CreateBaseRedactOutputPolicyRequest) (*emptypb.Empty, error) {
	err := s.baseRedactOutputPolicyCase.CreateBaseRedactOutputPolicy(ctx, req.GetBaseRedactOutputPolicy())
	if err != nil {
		log.Error(fmt.Sprintf("CreateBaseRedactOutputPolicy %v", err))
		return nil, errorsx.WrapInternal(err, "创建出库脱敏策略失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseRedactOutputPolicy 批量更新出库脱敏策略。
func (s *BaseRedactOutputPolicyService) UpdateBaseRedactOutputPolicy(ctx context.Context, req *adminv1.UpdateBaseRedactOutputPolicyRequest) (*emptypb.Empty, error) {
	err := s.baseRedactOutputPolicyCase.UpdateBaseRedactOutputPolicy(ctx, req.GetBaseRedactOutputPolicy())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseRedactOutputPolicy %v", err))
		return nil, errorsx.WrapInternal(err, "更新出库脱敏策略失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseRedactOutputPolicy 删除出库脱敏策略。
func (s *BaseRedactOutputPolicyService) DeleteBaseRedactOutputPolicy(ctx context.Context, req *adminv1.DeleteBaseRedactOutputPolicyRequest) (*emptypb.Empty, error) {
	err := s.baseRedactOutputPolicyCase.DeleteBaseRedactOutputPolicy(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBaseRedactOutputPolicy %v", err))
		return nil, errorsx.WrapInternal(err, "删除出库脱敏策略失败")
	}
	return new(emptypb.Empty), nil
}

// SetBaseRedactOutputPolicyStatus 设置出库脱敏策略状态。
func (s *BaseRedactOutputPolicyService) SetBaseRedactOutputPolicyStatus(ctx context.Context, req *adminv1.SetBaseRedactOutputPolicyStatusRequest) (*emptypb.Empty, error) {
	err := s.baseRedactOutputPolicyCase.SetBaseRedactOutputPolicyStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetBaseRedactOutputPolicyStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置出库脱敏策略状态失败")
	}
	return new(emptypb.Empty), nil
}
