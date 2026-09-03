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

// BaseTenantService Admin租户管理服务。
type BaseTenantService struct {
	adminv1.UnimplementedBaseTenantServiceServer
	baseTenantCase *biz.BaseTenantCase
}

// NewBaseTenantService 创建Admin租户管理服务。
func NewBaseTenantService(baseTenantCase *biz.BaseTenantCase) *BaseTenantService {
	return &BaseTenantService{
		baseTenantCase: baseTenantCase,
	}
}

// OptionBaseTenant 查询租户下拉选择。
func (s *BaseTenantService) OptionBaseTenant(ctx context.Context, req *adminv1.OptionBaseTenantRequest) (*commonv1.SelectOptionResponse, error) {
	list, err := s.baseTenantCase.OptionBaseTenant(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("OptionBaseTenant %v", err))
		return nil, errorsx.WrapInternal(err, "查询租户下拉选择失败")
	}
	return list, nil
}

// PageBaseTenant 查询租户分页列表。
func (s *BaseTenantService) PageBaseTenant(ctx context.Context, req *adminv1.PageBaseTenantRequest) (*adminv1.PageBaseTenantResponse, error) {
	page, err := s.baseTenantCase.PageBaseTenant(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseTenant %v", err))
		return nil, errorsx.WrapInternal(err, "查询租户分页列表失败")
	}
	return page, nil
}

// GetBaseTenant 查询租户。
func (s *BaseTenantService) GetBaseTenant(ctx context.Context, req *adminv1.GetBaseTenantRequest) (*adminv1.BaseTenantForm, error) {
	baseTenant, err := s.baseTenantCase.GetBaseTenant(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseTenant %v", err))
		return nil, errorsx.WrapInternal(err, "查询租户失败")
	}
	return baseTenant, nil
}

// CreateBaseTenant 创建租户。
func (s *BaseTenantService) CreateBaseTenant(ctx context.Context, req *adminv1.CreateBaseTenantRequest) (*emptypb.Empty, error) {
	err := s.baseTenantCase.CreateBaseTenant(ctx, req.GetBaseTenant())
	if err != nil {
		log.Error(fmt.Sprintf("CreateBaseTenant %v", err))
		return nil, errorsx.WrapInternal(err, "创建租户失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseTenant 更新租户。
func (s *BaseTenantService) UpdateBaseTenant(ctx context.Context, req *adminv1.UpdateBaseTenantRequest) (*emptypb.Empty, error) {
	err := s.baseTenantCase.UpdateBaseTenant(ctx, req.GetBaseTenant())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseTenant %v", err))
		return nil, errorsx.WrapInternal(err, "更新租户失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseTenant 删除租户。
func (s *BaseTenantService) DeleteBaseTenant(ctx context.Context, req *adminv1.DeleteBaseTenantRequest) (*emptypb.Empty, error) {
	err := s.baseTenantCase.DeleteBaseTenant(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBaseTenant %v", err))
		return nil, errorsx.WrapInternal(err, "删除租户失败")
	}
	return new(emptypb.Empty), nil
}

// SetBaseTenantStatus 设置状态。
func (s *BaseTenantService) SetBaseTenantStatus(ctx context.Context, req *adminv1.SetBaseTenantStatusRequest) (*emptypb.Empty, error) {
	err := s.baseTenantCase.SetBaseTenantStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetBaseTenantStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置状态失败")
	}
	return new(emptypb.Empty), nil
}
