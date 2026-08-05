package admin

import (
	"context"
	"fmt"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	commonv1 "github.com/liujitcn/kratos-admin/backend/core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/protobuf/types/known/emptypb"
)

// BaseRoleService Admin角色管理服务
type BaseRoleService struct {
	systemadminv1.UnimplementedBaseRoleServiceServer
	baseRoleCase *biz.BaseRoleCase
}

// NewBaseRoleService 创建Admin角色管理服务
func NewBaseRoleService(
	roleCase *biz.BaseRoleCase,
) *BaseRoleService {
	return &BaseRoleService{
		baseRoleCase: roleCase,
	}
}

// OptionBaseRole 查询角色下拉选择
func (s *BaseRoleService) OptionBaseRole(ctx context.Context, req *systemadminv1.OptionBaseRoleRequest) (*commonv1.SelectOptionResponse, error) {
	list, err := s.baseRoleCase.OptionBaseRole(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("OptionBaseRole %v", err))
		return nil, errorsx.WrapInternal(err, "查询角色下拉列表失败")
	}
	return list, nil
}

// PageBaseRole 查询角色分页列表
func (s *BaseRoleService) PageBaseRole(ctx context.Context, req *systemadminv1.PageBaseRoleRequest) (*systemadminv1.PageBaseRoleResponse, error) {
	page, err := s.baseRoleCase.PageBaseRole(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseRole %v", err))
		return nil, errorsx.WrapInternal(err, "查询角色分页列表失败")
	}
	return page, nil
}

// GetBaseRole 查询角色
func (s *BaseRoleService) GetBaseRole(ctx context.Context, req *systemadminv1.GetBaseRoleRequest) (*systemadminv1.BaseRoleForm, error) {
	baseRole, err := s.baseRoleCase.GetBaseRole(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseRole %v", err))
		return nil, errorsx.WrapInternal(err, "查询角色失败")
	}
	return baseRole, nil
}

// CreateBaseRole 创建角色
func (s *BaseRoleService) CreateBaseRole(ctx context.Context, req *systemadminv1.CreateBaseRoleRequest) (*emptypb.Empty, error) {
	err := s.baseRoleCase.CreateBaseRole(ctx, req.GetBaseRole())
	if err != nil {
		log.Error(fmt.Sprintf("CreateBaseRole %v", err))
		return nil, errorsx.WrapInternal(err, "创建角色失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseRole 更新角色
func (s *BaseRoleService) UpdateBaseRole(ctx context.Context, req *systemadminv1.UpdateBaseRoleRequest) (*emptypb.Empty, error) {
	err := s.baseRoleCase.UpdateBaseRole(ctx, req.GetBaseRole())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseRole %v", err))
		return nil, errorsx.WrapInternal(err, "更新角色失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseRole 删除角色
func (s *BaseRoleService) DeleteBaseRole(ctx context.Context, req *systemadminv1.DeleteBaseRoleRequest) (*emptypb.Empty, error) {
	err := s.baseRoleCase.DeleteBaseRole(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBaseRole %v", err))
		return nil, errorsx.WrapInternal(err, "删除角色失败")
	}
	return new(emptypb.Empty), nil
}

// SetBaseRoleStatus 设置状态
func (s *BaseRoleService) SetBaseRoleStatus(ctx context.Context, req *systemadminv1.SetBaseRoleStatusRequest) (*emptypb.Empty, error) {
	err := s.baseRoleCase.SetBaseRoleStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetBaseRoleStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置状态失败")
	}
	return new(emptypb.Empty), nil
}

// SetBaseRoleMenu 设置角色菜单权限
func (s *BaseRoleService) SetBaseRoleMenu(ctx context.Context, req *systemadminv1.SetBaseRoleMenuRequest) (*emptypb.Empty, error) {
	err := s.baseRoleCase.SetBaseRoleMenu(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetBaseRoleMenu %v", err))
		return nil, errorsx.WrapInternal(err, "设置角色菜单权限失败")
	}
	return new(emptypb.Empty), nil
}
