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

// BaseJobService Admin定时任务服务
type BaseJobService struct {
	adminv1.UnimplementedBaseJobServiceServer
	baseJobCase *biz.BaseJobCase
}

// NewBaseJobService 创建Admin定时任务服务
func NewBaseJobService(baseJobCase *biz.BaseJobCase) *BaseJobService {
	return &BaseJobService{baseJobCase: baseJobCase}
}

// OptionBaseJob 查询定时任务下拉选择
func (s *BaseJobService) OptionBaseJob(ctx context.Context, req *adminv1.OptionBaseJobRequest) (*commonv1.SelectOptionResponse, error) {
	list, err := s.baseJobCase.OptionBaseJob(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("OptionBaseJob %v", err))
		return nil, errorsx.WrapInternal(err, "查询定时任务下拉列表失败")
	}
	return list, nil
}

// PageBaseJob 查询定时任务分页列表
func (s *BaseJobService) PageBaseJob(ctx context.Context, req *adminv1.PageBaseJobRequest) (*adminv1.PageBaseJobResponse, error) {
	page, err := s.baseJobCase.PageBaseJob(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseJob %v", err))
		return nil, errorsx.WrapInternal(err, "查询定时任务分页列表失败")
	}
	return page, nil
}

// GetBaseJob 查询定时任务
func (s *BaseJobService) GetBaseJob(ctx context.Context, req *adminv1.GetBaseJobRequest) (*adminv1.BaseJobForm, error) {
	baseJob, err := s.baseJobCase.GetBaseJob(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseJob %v", err))
		return nil, errorsx.WrapInternal(err, "查询定时任务失败")
	}
	return baseJob, nil
}

// CreateBaseJob 创建定时任务
func (s *BaseJobService) CreateBaseJob(ctx context.Context, req *adminv1.CreateBaseJobRequest) (*emptypb.Empty, error) {
	err := s.baseJobCase.CreateBaseJob(ctx, req.GetBaseJob())
	if err != nil {
		log.Error(fmt.Sprintf("CreateBaseJob %v", err))
		return nil, errorsx.WrapInternal(err, "创建定时任务失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseJob 更新定时任务
func (s *BaseJobService) UpdateBaseJob(ctx context.Context, req *adminv1.UpdateBaseJobRequest) (*emptypb.Empty, error) {
	err := s.baseJobCase.UpdateBaseJob(ctx, req.GetBaseJob())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseJob %v", err))
		return nil, errorsx.WrapInternal(err, "更新定时任务失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseJob 删除定时任务
func (s *BaseJobService) DeleteBaseJob(ctx context.Context, req *adminv1.DeleteBaseJobRequest) (*emptypb.Empty, error) {
	err := s.baseJobCase.DeleteBaseJob(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBaseJob %v", err))
		return nil, errorsx.WrapInternal(err, "删除定时任务失败")
	}
	return new(emptypb.Empty), nil
}

// SetBaseJobStatus 设置状态
func (s *BaseJobService) SetBaseJobStatus(ctx context.Context, req *adminv1.SetBaseJobStatusRequest) (*emptypb.Empty, error) {
	err := s.baseJobCase.SetBaseJobStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetBaseJobStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置状态失败")
	}
	return new(emptypb.Empty), nil
}

// StartBaseJob 启动任务
func (s *BaseJobService) StartBaseJob(ctx context.Context, req *adminv1.StartBaseJobRequest) (*emptypb.Empty, error) {
	err := s.baseJobCase.StartBaseJob(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("StartBaseJob %v", err))
		return nil, errorsx.WrapInternal(err, "启动任务失败")
	}
	return new(emptypb.Empty), nil
}

// StopBaseJob 停止任务
func (s *BaseJobService) StopBaseJob(ctx context.Context, req *adminv1.StopBaseJobRequest) (*emptypb.Empty, error) {
	err := s.baseJobCase.StopBaseJob(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("StopBaseJob %v", err))
		return nil, errorsx.WrapInternal(err, "停止任务失败")
	}
	return new(emptypb.Empty), nil
}

// ExecuteBaseJob 执行任务
func (s *BaseJobService) ExecuteBaseJob(ctx context.Context, req *adminv1.ExecuteBaseJobRequest) (*emptypb.Empty, error) {
	err := s.baseJobCase.ExecuteBaseJob(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("ExecuteBaseJob %v", err))
		return nil, errorsx.WrapInternal(err, "执行任务失败")
	}
	return new(emptypb.Empty), nil
}
