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

// BaseTenantProjectService Admin项目管理服务。
type BaseTenantProjectService struct {
	adminv1.UnimplementedBaseTenantProjectServiceServer
	baseTenantProjectCase *biz.BaseTenantProjectCase
}

// NewBaseTenantProjectService 创建Admin项目管理服务。
func NewBaseTenantProjectService(baseTenantProjectCase *biz.BaseTenantProjectCase) *BaseTenantProjectService {
	return &BaseTenantProjectService{baseTenantProjectCase: baseTenantProjectCase}
}

// OptionBaseTenantProject 查询项目下拉选择。
func (s *BaseTenantProjectService) OptionBaseTenantProject(ctx context.Context, req *adminv1.OptionBaseTenantProjectRequest) (*commonv1.SelectOptionResponse, error) {
	list, err := s.baseTenantProjectCase.OptionBaseTenantProject(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("OptionBaseTenantProject %v", err))
		return nil, errorsx.WrapInternal(err, "查询项目下拉列表失败")
	}
	return list, nil
}

// PageBaseTenantProject 查询项目分页列表。
func (s *BaseTenantProjectService) PageBaseTenantProject(ctx context.Context, req *adminv1.PageBaseTenantProjectRequest) (*adminv1.PageBaseTenantProjectResponse, error) {
	page, err := s.baseTenantProjectCase.PageBaseTenantProject(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseTenantProject %v", err))
		return nil, errorsx.WrapInternal(err, "查询项目分页列表失败")
	}
	return page, nil
}

// TreeBaseTenantProject 查询当前账号可用的租户项目树。
func (s *BaseTenantProjectService) TreeBaseTenantProject(ctx context.Context, req *adminv1.TreeBaseTenantProjectRequest) (*adminv1.TreeBaseTenantProjectResponse, error) {
	tree, err := s.baseTenantProjectCase.TreeBaseTenantProject(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("TreeBaseTenantProject %v", err))
		return nil, errorsx.WrapInternal(err, "查询项目树失败")
	}
	return tree, nil
}

// GetBaseTenantProject 查询项目。
func (s *BaseTenantProjectService) GetBaseTenantProject(ctx context.Context, req *adminv1.GetBaseTenantProjectRequest) (*adminv1.BaseTenantProjectForm, error) {
	baseTenantProject, err := s.baseTenantProjectCase.GetBaseTenantProject(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseTenantProject %v", err))
		return nil, errorsx.WrapInternal(err, "查询项目失败")
	}
	return baseTenantProject, nil
}

// CreateBaseTenantProject 创建项目。
func (s *BaseTenantProjectService) CreateBaseTenantProject(ctx context.Context, req *adminv1.CreateBaseTenantProjectRequest) (*emptypb.Empty, error) {
	err := s.baseTenantProjectCase.CreateBaseTenantProject(ctx, req.GetBaseTenantProject())
	if err != nil {
		log.Error(fmt.Sprintf("CreateBaseTenantProject %v", err))
		return nil, errorsx.WrapInternal(err, "创建项目失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseTenantProject 更新项目。
func (s *BaseTenantProjectService) UpdateBaseTenantProject(ctx context.Context, req *adminv1.UpdateBaseTenantProjectRequest) (*emptypb.Empty, error) {
	err := s.baseTenantProjectCase.UpdateBaseTenantProject(ctx, req.GetBaseTenantProject())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseTenantProject %v", err))
		return nil, errorsx.WrapInternal(err, "更新项目失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseTenantProject 删除项目。
func (s *BaseTenantProjectService) DeleteBaseTenantProject(ctx context.Context, req *adminv1.DeleteBaseTenantProjectRequest) (*emptypb.Empty, error) {
	err := s.baseTenantProjectCase.DeleteBaseTenantProject(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBaseTenantProject %v", err))
		return nil, errorsx.WrapInternal(err, "删除项目失败")
	}
	return new(emptypb.Empty), nil
}

// SetBaseTenantProjectStatus 设置项目状态。
func (s *BaseTenantProjectService) SetBaseTenantProjectStatus(ctx context.Context, req *adminv1.SetBaseTenantProjectStatusRequest) (*emptypb.Empty, error) {
	err := s.baseTenantProjectCase.SetBaseTenantProjectStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetBaseTenantProjectStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置项目状态失败")
	}
	return new(emptypb.Empty), nil
}
