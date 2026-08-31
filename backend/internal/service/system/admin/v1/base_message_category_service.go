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

// BaseMessageCategoryService 消息分类管理服务。
type BaseMessageCategoryService struct {
	adminv1.UnimplementedBaseMessageCategoryServiceServer
	baseMessageCategoryCase *biz.BaseMessageCategoryCase
}

// NewBaseMessageCategoryService 创建消息分类管理服务。
func NewBaseMessageCategoryService(baseMessageCategoryCase *biz.BaseMessageCategoryCase) *BaseMessageCategoryService {
	return &BaseMessageCategoryService{baseMessageCategoryCase: baseMessageCategoryCase}
}

// OptionBaseMessageCategory 查询消息分类选项。
func (s *BaseMessageCategoryService) OptionBaseMessageCategory(ctx context.Context, req *adminv1.OptionBaseMessageCategoryRequest) (*commonv1.SelectOptionResponse, error) {
	result, err := s.baseMessageCategoryCase.OptionBaseMessageCategory(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("OptionBaseMessageCategory %v", err))
		return nil, errorsx.WrapInternal(err, "查询消息分类选项失败")
	}
	return result, nil
}

// PageBaseMessageCategory 分页查询消息分类。
func (s *BaseMessageCategoryService) PageBaseMessageCategory(ctx context.Context, req *adminv1.PageBaseMessageCategoryRequest) (*adminv1.PageBaseMessageCategoryResponse, error) {
	result, err := s.baseMessageCategoryCase.PageBaseMessageCategory(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseMessageCategory %v", err))
		return nil, errorsx.WrapInternal(err, "查询消息分类失败")
	}
	return result, nil
}

// GetBaseMessageCategory 查询消息分类详情。
func (s *BaseMessageCategoryService) GetBaseMessageCategory(ctx context.Context, req *adminv1.GetBaseMessageCategoryRequest) (*adminv1.BaseMessageCategoryForm, error) {
	result, err := s.baseMessageCategoryCase.GetBaseMessageCategory(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseMessageCategory %v", err))
		return nil, errorsx.WrapInternal(err, "查询消息分类失败")
	}
	return result, nil
}

// CreateBaseMessageCategory 创建消息分类。
func (s *BaseMessageCategoryService) CreateBaseMessageCategory(ctx context.Context, req *adminv1.CreateBaseMessageCategoryRequest) (*emptypb.Empty, error) {
	err := s.baseMessageCategoryCase.CreateBaseMessageCategory(ctx, req.GetBaseMessageCategory())
	if err != nil {
		log.Error(fmt.Sprintf("CreateBaseMessageCategory %v", err))
		return nil, errorsx.WrapInternal(err, "创建消息分类失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseMessageCategory 更新消息分类。
func (s *BaseMessageCategoryService) UpdateBaseMessageCategory(ctx context.Context, req *adminv1.UpdateBaseMessageCategoryRequest) (*emptypb.Empty, error) {
	err := s.baseMessageCategoryCase.UpdateBaseMessageCategory(ctx, req.GetBaseMessageCategory())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseMessageCategory %v", err))
		return nil, errorsx.WrapInternal(err, "更新消息分类失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseMessageCategory 删除消息分类。
func (s *BaseMessageCategoryService) DeleteBaseMessageCategory(ctx context.Context, req *adminv1.DeleteBaseMessageCategoryRequest) (*emptypb.Empty, error) {
	err := s.baseMessageCategoryCase.DeleteBaseMessageCategory(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBaseMessageCategory %v", err))
		return nil, errorsx.WrapInternal(err, "删除消息分类失败")
	}
	return new(emptypb.Empty), nil
}

// SetBaseMessageCategoryStatus 设置消息分类状态。
func (s *BaseMessageCategoryService) SetBaseMessageCategoryStatus(ctx context.Context, req *adminv1.SetBaseMessageCategoryStatusRequest) (*emptypb.Empty, error) {
	err := s.baseMessageCategoryCase.SetBaseMessageCategoryStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetBaseMessageCategoryStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置消息分类状态失败")
	}
	return new(emptypb.Empty), nil
}
