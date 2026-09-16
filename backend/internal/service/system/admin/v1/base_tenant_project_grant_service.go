package admin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"
	"google.golang.org/protobuf/types/known/emptypb"
)

// BaseTenantProjectGrantService 提供项目授权管理接口。
type BaseTenantProjectGrantService struct {
	adminv1.UnimplementedBaseTenantProjectGrantServiceServer
	projectCase *biz.BaseTenantProjectGrantCase
}

// NewBaseTenantProjectGrantService 创建项目授权管理服务。
func NewBaseTenantProjectGrantService(projectCase *biz.BaseTenantProjectGrantCase) *BaseTenantProjectGrantService {
	return &BaseTenantProjectGrantService{projectCase: projectCase}
}

// PageBaseTenantProjectGrant 查询项目授权分页列表。
func (s *BaseTenantProjectGrantService) PageBaseTenantProjectGrant(ctx context.Context, req *adminv1.PageBaseTenantProjectGrantRequest) (*adminv1.PageBaseTenantProjectGrantResponse, error) {
	result, err := s.projectCase.PageBaseTenantProjectGrant(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseTenantProjectGrant %v", err))
		return nil, errorsx.WrapInternal(err, "查询项目授权分页列表失败")
	}
	return result, nil
}

// GetBaseTenantProjectGrant 查询项目授权详情。
func (s *BaseTenantProjectGrantService) GetBaseTenantProjectGrant(ctx context.Context, req *adminv1.GetBaseTenantProjectGrantRequest) (*adminv1.BaseTenantProjectGrant, error) {
	result, err := s.projectCase.GetBaseTenantProjectGrant(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseTenantProjectGrant %v", err))
		return nil, errorsx.WrapInternal(err, "查询项目授权详情失败")
	}
	return result, nil
}

// CreateBaseTenantProjectGrant 新增项目授权。
func (s *BaseTenantProjectGrantService) CreateBaseTenantProjectGrant(ctx context.Context, req *adminv1.CreateBaseTenantProjectGrantRequest) (*emptypb.Empty, error) {
	err := s.projectCase.CreateBaseTenantProjectGrant(ctx, req.GetBaseTenantProjectGrant())
	if err != nil {
		log.Error(fmt.Sprintf("CreateBaseTenantProjectGrant %v", err))
		return nil, errorsx.WrapInternal(err, "新增项目授权失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseTenantProjectGrant 更新项目授权的项目范围。
func (s *BaseTenantProjectGrantService) UpdateBaseTenantProjectGrant(ctx context.Context, req *adminv1.UpdateBaseTenantProjectGrantRequest) (*emptypb.Empty, error) {
	err := s.projectCase.UpdateBaseTenantProjectGrant(ctx, req.GetBaseTenantProjectGrant())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseTenantProjectGrant %v", err))
		return nil, errorsx.WrapInternal(err, "更新项目授权失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseTenantProjectGrant 删除项目授权。
func (s *BaseTenantProjectGrantService) DeleteBaseTenantProjectGrant(ctx context.Context, req *adminv1.DeleteBaseTenantProjectGrantRequest) (*emptypb.Empty, error) {
	err := s.projectCase.DeleteBaseTenantProjectGrant(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBaseTenantProjectGrant %v", err))
		return nil, errorsx.WrapInternal(err, "删除项目授权失败")
	}
	return new(emptypb.Empty), nil
}
