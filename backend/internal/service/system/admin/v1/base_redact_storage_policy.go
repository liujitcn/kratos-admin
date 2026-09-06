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

// BaseRedactStoragePolicyService 提供入库脱敏策略管理接口。
type BaseRedactStoragePolicyService struct {
	adminv1.UnimplementedBaseRedactStoragePolicyServiceServer
	baseRedactStoragePolicyCase *biz.BaseRedactStoragePolicyCase
}

// NewBaseRedactStoragePolicyService 创建入库脱敏策略服务。
func NewBaseRedactStoragePolicyService(baseRedactStoragePolicyCase *biz.BaseRedactStoragePolicyCase) *BaseRedactStoragePolicyService {
	return &BaseRedactStoragePolicyService{baseRedactStoragePolicyCase: baseRedactStoragePolicyCase}
}

// PageBaseRedactStoragePolicy 分页查询入库脱敏策略。
func (s *BaseRedactStoragePolicyService) PageBaseRedactStoragePolicy(ctx context.Context, req *adminv1.PageBaseRedactStoragePolicyRequest) (*adminv1.PageBaseRedactStoragePolicyResponse, error) {
	result, err := s.baseRedactStoragePolicyCase.PageBaseRedactStoragePolicy(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseRedactStoragePolicy %v", err))
		return nil, errorsx.WrapInternal(err, "分页查询入库脱敏策略失败")
	}
	return result, nil
}

// GetBaseRedactStoragePolicy 查询入库脱敏策略详情。
func (s *BaseRedactStoragePolicyService) GetBaseRedactStoragePolicy(ctx context.Context, req *adminv1.GetBaseRedactStoragePolicyRequest) (*adminv1.BaseRedactStoragePolicyForm, error) {
	result, err := s.baseRedactStoragePolicyCase.GetBaseRedactStoragePolicy(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseRedactStoragePolicy %v", err))
		return nil, errorsx.WrapInternal(err, "查询入库脱敏策略详情失败")
	}
	return result, nil
}

// CreateBaseRedactStoragePolicy 批量创建入库脱敏策略。
func (s *BaseRedactStoragePolicyService) CreateBaseRedactStoragePolicy(ctx context.Context, req *adminv1.CreateBaseRedactStoragePolicyRequest) (*emptypb.Empty, error) {
	err := s.baseRedactStoragePolicyCase.CreateBaseRedactStoragePolicy(ctx, req.GetBaseRedactStoragePolicy())
	if err != nil {
		log.Error(fmt.Sprintf("CreateBaseRedactStoragePolicy %v", err))
		return nil, errorsx.WrapInternal(err, "创建入库脱敏策略失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseRedactStoragePolicy 批量更新入库脱敏策略。
func (s *BaseRedactStoragePolicyService) UpdateBaseRedactStoragePolicy(ctx context.Context, req *adminv1.UpdateBaseRedactStoragePolicyRequest) (*emptypb.Empty, error) {
	err := s.baseRedactStoragePolicyCase.UpdateBaseRedactStoragePolicy(ctx, req.GetBaseRedactStoragePolicy())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseRedactStoragePolicy %v", err))
		return nil, errorsx.WrapInternal(err, "更新入库脱敏策略失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseRedactStoragePolicy 删除入库脱敏策略。
func (s *BaseRedactStoragePolicyService) DeleteBaseRedactStoragePolicy(ctx context.Context, req *adminv1.DeleteBaseRedactStoragePolicyRequest) (*emptypb.Empty, error) {
	err := s.baseRedactStoragePolicyCase.DeleteBaseRedactStoragePolicy(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBaseRedactStoragePolicy %v", err))
		return nil, errorsx.WrapInternal(err, "删除入库脱敏策略失败")
	}
	return new(emptypb.Empty), nil
}

// SetBaseRedactStoragePolicyStatus 设置入库脱敏策略状态。
func (s *BaseRedactStoragePolicyService) SetBaseRedactStoragePolicyStatus(ctx context.Context, req *adminv1.SetBaseRedactStoragePolicyStatusRequest) (*emptypb.Empty, error) {
	err := s.baseRedactStoragePolicyCase.SetBaseRedactStoragePolicyStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetBaseRedactStoragePolicyStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置入库脱敏策略状态失败")
	}
	return new(emptypb.Empty), nil
}
