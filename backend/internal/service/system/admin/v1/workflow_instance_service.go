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

// WorkflowInstanceService Admin工作流流程实例服务。
type WorkflowInstanceService struct {
	adminv1.UnimplementedWorkflowInstanceServiceServer
	instanceCase *biz.WorkflowInstanceCase
}

// NewWorkflowInstanceService 创建Admin工作流流程实例服务。
func NewWorkflowInstanceService(instanceCase *biz.WorkflowInstanceCase) *WorkflowInstanceService {
	return &WorkflowInstanceService{instanceCase: instanceCase}
}

// SubmitWorkflowInstance 提交审批申请。
func (s *WorkflowInstanceService) SubmitWorkflowInstance(ctx context.Context, req *adminv1.SubmitWorkflowInstanceRequest) (*adminv1.SubmitWorkflowInstanceResponse, error) {
	instanceID, err := s.instanceCase.SubmitWorkflowInstance(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SubmitWorkflowInstance %v", err))
		return nil, errorsx.WrapInternal(err, "提交审批申请失败")
	}
	return &adminv1.SubmitWorkflowInstanceResponse{InstanceId: instanceID}, nil
}

// WithdrawWorkflowInstance 撤回审批申请。
func (s *WorkflowInstanceService) WithdrawWorkflowInstance(ctx context.Context, req *adminv1.WithdrawWorkflowInstanceRequest) (*emptypb.Empty, error) {
	err := s.instanceCase.WithdrawWorkflowInstance(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("WithdrawWorkflowInstance %v", err))
		return nil, errorsx.WrapInternal(err, "撤回审批申请失败")
	}
	return new(emptypb.Empty), nil
}

// PageWorkflowInstance 查询流程实例分页列表。
func (s *WorkflowInstanceService) PageWorkflowInstance(ctx context.Context, req *adminv1.PageWorkflowInstanceRequest) (*adminv1.PageWorkflowInstanceResponse, error) {
	page, err := s.instanceCase.PageWorkflowInstance(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageWorkflowInstance %v", err))
		return nil, errorsx.WrapInternal(err, "查询流程实例失败")
	}
	return page, nil
}

// GetWorkflowInstance 查询流程实例详情。
func (s *WorkflowInstanceService) GetWorkflowInstance(ctx context.Context, req *adminv1.GetWorkflowInstanceRequest) (*adminv1.GetWorkflowInstanceResponse, error) {
	detail, err := s.instanceCase.GetWorkflowInstance(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetWorkflowInstance %v", err))
		return nil, errorsx.WrapInternal(err, "查询流程实例详情失败")
	}
	return detail, nil
}

// RetryWorkflowScript 重试失败的工作流脚本。
func (s *WorkflowInstanceService) RetryWorkflowScript(ctx context.Context, req *adminv1.RetryWorkflowScriptRequest) (*emptypb.Empty, error) {
	err := s.instanceCase.RetryWorkflowScript(ctx, req)
	if err != nil {
		return nil, errorsx.WrapInternal(err, "重试工作流脚本失败")
	}
	return new(emptypb.Empty), nil
}
