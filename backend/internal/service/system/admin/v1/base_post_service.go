package admin

import (
	"context"
	"fmt"

	"github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/protobuf/types/known/emptypb"
)

// BasePostService Admin岗位管理服务。
type BasePostService struct {
	adminv1.UnimplementedBasePostServiceServer
	basePostCase *biz.BasePostCase
}

// NewBasePostService 创建Admin岗位管理服务。
func NewBasePostService(basePostCase *biz.BasePostCase) *BasePostService {
	return &BasePostService{basePostCase: basePostCase}
}

// OptionBasePost 查询岗位下拉选择。
func (s *BasePostService) OptionBasePost(ctx context.Context, req *adminv1.OptionBasePostRequest) (*commonv1.SelectOptionResponse, error) {
	list, err := s.basePostCase.OptionBasePost(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("OptionBasePost %v", err))
		return nil, errorsx.WrapInternal(err, "查询岗位下拉列表失败")
	}
	return list, nil
}

// PageBasePost 查询岗位分页列表。
func (s *BasePostService) PageBasePost(ctx context.Context, req *adminv1.PageBasePostRequest) (*adminv1.PageBasePostResponse, error) {
	page, err := s.basePostCase.PageBasePost(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBasePost %v", err))
		return nil, errorsx.WrapInternal(err, "查询岗位分页列表失败")
	}
	return page, nil
}

// GetBasePost 查询岗位。
func (s *BasePostService) GetBasePost(ctx context.Context, req *adminv1.GetBasePostRequest) (*adminv1.BasePostForm, error) {
	basePost, err := s.basePostCase.GetBasePost(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBasePost %v", err))
		return nil, errorsx.WrapInternal(err, "查询岗位失败")
	}
	return basePost, nil
}

// CreateBasePost 创建岗位。
func (s *BasePostService) CreateBasePost(ctx context.Context, req *adminv1.CreateBasePostRequest) (*emptypb.Empty, error) {
	err := s.basePostCase.CreateBasePost(ctx, req.GetBasePost())
	if err != nil {
		log.Error(fmt.Sprintf("CreateBasePost %v", err))
		return nil, errorsx.WrapInternal(err, "创建岗位失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateBasePost 更新岗位。
func (s *BasePostService) UpdateBasePost(ctx context.Context, req *adminv1.UpdateBasePostRequest) (*emptypb.Empty, error) {
	err := s.basePostCase.UpdateBasePost(ctx, req.GetBasePost())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBasePost %v", err))
		return nil, errorsx.WrapInternal(err, "更新岗位失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteBasePost 删除岗位。
func (s *BasePostService) DeleteBasePost(ctx context.Context, req *adminv1.DeleteBasePostRequest) (*emptypb.Empty, error) {
	err := s.basePostCase.DeleteBasePost(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBasePost %v", err))
		return nil, errorsx.WrapInternal(err, "删除岗位失败")
	}
	return new(emptypb.Empty), nil
}

// SetBasePostStatus 设置岗位状态。
func (s *BasePostService) SetBasePostStatus(ctx context.Context, req *adminv1.SetBasePostStatusRequest) (*emptypb.Empty, error) {
	err := s.basePostCase.SetBasePostStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetBasePostStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置岗位状态失败")
	}
	return new(emptypb.Empty), nil
}
