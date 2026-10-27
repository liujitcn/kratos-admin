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

// WorkflowTaskService Admin工作流审批任务服务。
type WorkflowTaskService struct {
	adminv1.UnimplementedWorkflowTaskServiceServer
	taskCase *biz.WorkflowTaskCase
}

// NewWorkflowTaskService 创建Admin工作流审批任务服务。
func NewWorkflowTaskService(taskCase *biz.WorkflowTaskCase) *WorkflowTaskService {
	return &WorkflowTaskService{taskCase: taskCase}
}

// PageMyWorkflowTask 查询我的审批任务分页列表。
func (s *WorkflowTaskService) PageMyWorkflowTask(ctx context.Context, req *adminv1.PageMyWorkflowTaskRequest) (*adminv1.PageMyWorkflowTaskResponse, error) {
	page, err := s.taskCase.PageMyWorkflowTask(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageMyWorkflowTask %v", err))
		return nil, errorsx.WrapInternal(err, "查询我的审批任务失败")
	}
	return page, nil
}

// GetWorkflowTask 查询审批任务详情。
func (s *WorkflowTaskService) GetWorkflowTask(ctx context.Context, req *adminv1.GetWorkflowTaskRequest) (*adminv1.GetWorkflowTaskResponse, error) {
	detail, err := s.taskCase.GetWorkflowTask(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetWorkflowTask %v", err))
		return nil, errorsx.WrapInternal(err, "查询审批任务详情失败")
	}
	return detail, nil
}

// AuditWorkflowTask 审批任务办理。
func (s *WorkflowTaskService) AuditWorkflowTask(ctx context.Context, req *adminv1.AuditWorkflowTaskRequest) (*emptypb.Empty, error) {
	err := s.taskCase.AuditWorkflowTask(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("AuditWorkflowTask %v", err))
		return nil, errorsx.WrapInternal(err, "审批任务办理失败")
	}
	return new(emptypb.Empty), nil
}
