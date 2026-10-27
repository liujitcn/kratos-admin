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

// WorkflowDefinitionService Admin工作流流程定义服务。
type WorkflowDefinitionService struct {
	adminv1.UnimplementedWorkflowDefinitionServiceServer
	definitionCase *biz.WorkflowDefinitionCase
}

// NewWorkflowDefinitionService 创建Admin工作流流程定义服务。
func NewWorkflowDefinitionService(definitionCase *biz.WorkflowDefinitionCase) *WorkflowDefinitionService {
	return &WorkflowDefinitionService{definitionCase: definitionCase}
}

// PageWorkflowScript 查询流程可使用的已启用受限脚本。
func (s *WorkflowDefinitionService) PageWorkflowScript(ctx context.Context, req *adminv1.PageBaseScriptRequest) (*adminv1.PageBaseScriptResponse, error) {
	response, err := s.definitionCase.PageWorkflowScript(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageWorkflowScript %v", err))
		return nil, errorsx.WrapInternal(err, "查询流程脚本失败")
	}
	return response, nil
}

// TestWorkflowScript 试运行已保存的受限流程脚本。
func (s *WorkflowDefinitionService) TestWorkflowScript(ctx context.Context, req *adminv1.TestWorkflowScriptRequest) (*adminv1.TestRunBaseScriptResponse, error) {
	response, err := s.definitionCase.TestWorkflowScript(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("TestWorkflowScript %v", err))
		return nil, errorsx.WrapInternal(err, "试运行流程脚本失败")
	}
	return response, nil
}

// OptionWorkflowDefinition 查询启用流程定义下拉选择。
func (s *WorkflowDefinitionService) OptionWorkflowDefinition(ctx context.Context, req *adminv1.OptionWorkflowDefinitionRequest) (*commonv1.SelectOptionResponse, error) {
	options, err := s.definitionCase.OptionWorkflowDefinition(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("OptionWorkflowDefinition %v", err))
		return nil, errorsx.WrapInternal(err, "查询流程定义下拉列表失败")
	}
	return options, nil
}

// GetWorkflowApplyForm 查询启用流程定义的申请表单。
func (s *WorkflowDefinitionService) GetWorkflowApplyForm(ctx context.Context, req *adminv1.GetWorkflowApplyFormRequest) (*adminv1.GetWorkflowApplyFormResponse, error) {
	form, err := s.definitionCase.GetWorkflowApplyForm(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("GetWorkflowApplyForm %v", err))
		return nil, errorsx.WrapInternal(err, "查询申请表单失败")
	}
	return form, nil
}

// PageWorkflowDefinition 查询流程定义分页列表。
func (s *WorkflowDefinitionService) PageWorkflowDefinition(ctx context.Context, req *adminv1.PageWorkflowDefinitionRequest) (*adminv1.PageWorkflowDefinitionResponse, error) {
	page, err := s.definitionCase.PageWorkflowDefinition(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageWorkflowDefinition %v", err))
		return nil, errorsx.WrapInternal(err, "查询流程定义失败")
	}
	return page, nil
}

// GetWorkflowDefinition 查询流程定义。
func (s *WorkflowDefinitionService) GetWorkflowDefinition(ctx context.Context, req *adminv1.GetWorkflowDefinitionRequest) (*adminv1.WorkflowDefinitionForm, error) {
	form, err := s.definitionCase.GetWorkflowDefinition(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetWorkflowDefinition %v", err))
		return nil, errorsx.WrapInternal(err, "查询流程定义失败")
	}
	return form, nil
}

// CreateWorkflowDefinition 创建流程定义。
func (s *WorkflowDefinitionService) CreateWorkflowDefinition(ctx context.Context, req *adminv1.CreateWorkflowDefinitionRequest) (*emptypb.Empty, error) {
	err := s.definitionCase.CreateWorkflowDefinition(ctx, req.GetDefinition())
	if err != nil {
		log.Error(fmt.Sprintf("CreateWorkflowDefinition %v", err))
		return nil, errorsx.WrapInternal(err, "创建流程定义失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateWorkflowDefinition 更新流程定义。
func (s *WorkflowDefinitionService) UpdateWorkflowDefinition(ctx context.Context, req *adminv1.UpdateWorkflowDefinitionRequest) (*emptypb.Empty, error) {
	err := s.definitionCase.UpdateWorkflowDefinition(ctx, req.GetDefinition())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateWorkflowDefinition %v", err))
		return nil, errorsx.WrapInternal(err, "更新流程定义失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteWorkflowDefinition 删除流程定义。
func (s *WorkflowDefinitionService) DeleteWorkflowDefinition(ctx context.Context, req *adminv1.DeleteWorkflowDefinitionRequest) (*emptypb.Empty, error) {
	err := s.definitionCase.DeleteWorkflowDefinition(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteWorkflowDefinition %v", err))
		return nil, errorsx.WrapInternal(err, "删除流程定义失败")
	}
	return new(emptypb.Empty), nil
}

// SetWorkflowDefinitionStatus 设置流程定义状态。
func (s *WorkflowDefinitionService) SetWorkflowDefinitionStatus(ctx context.Context, req *adminv1.SetWorkflowDefinitionStatusRequest) (*emptypb.Empty, error) {
	err := s.definitionCase.SetWorkflowDefinitionStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetWorkflowDefinitionStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置流程定义状态失败")
	}
	return new(emptypb.Empty), nil
}
