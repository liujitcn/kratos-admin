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

// WorkflowDelegationService Admin工作流审批委托服务。
type WorkflowDelegationService struct {
	adminv1.UnimplementedWorkflowDelegationServiceServer
	delegationCase *biz.WorkflowDelegationCase
}

// NewWorkflowDelegationService 创建Admin工作流审批委托服务。
func NewWorkflowDelegationService(delegationCase *biz.WorkflowDelegationCase) *WorkflowDelegationService {
	return &WorkflowDelegationService{delegationCase: delegationCase}
}

// PageWorkflowDelegation 查询审批委托分页列表。
func (s *WorkflowDelegationService) PageWorkflowDelegation(ctx context.Context, req *adminv1.PageWorkflowDelegationRequest) (*adminv1.PageWorkflowDelegationResponse, error) {
	page, err := s.delegationCase.PageWorkflowDelegation(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageWorkflowDelegation %v", err))
		return nil, errorsx.WrapInternal(err, "查询审批委托失败")
	}
	return page, nil
}

// SetWorkflowDelegation 设置我的审批委托。
func (s *WorkflowDelegationService) SetWorkflowDelegation(ctx context.Context, req *adminv1.SetWorkflowDelegationRequest) (*emptypb.Empty, error) {
	err := s.delegationCase.SetWorkflowDelegation(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetWorkflowDelegation %v", err))
		return nil, errorsx.WrapInternal(err, "设置审批委托失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteWorkflowDelegation 删除审批委托。
func (s *WorkflowDelegationService) DeleteWorkflowDelegation(ctx context.Context, req *adminv1.DeleteWorkflowDelegationRequest) (*emptypb.Empty, error) {
	err := s.delegationCase.DeleteWorkflowDelegation(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteWorkflowDelegation %v", err))
		return nil, errorsx.WrapInternal(err, "删除审批委托失败")
	}
	return new(emptypb.Empty), nil
}
