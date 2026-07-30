package biz

import (
	"context"

	basebiz "github.com/liujitcn/kratos-admin/backend/internal/biz"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"

	"github.com/liujitcn/gorm-kit/repository"
)

// BaseRoleCase 处理基础角色业务。
type BaseRoleCase struct {
	*data.BaseRoleRepository
	baseTenantCase *basebiz.BaseTenantCase
}

// NewBaseRoleCase 创建基础角色业务实例。
func NewBaseRoleCase(
	baseRoleRepo *data.BaseRoleRepository,
	baseTenantCase *basebiz.BaseTenantCase,
) *BaseRoleCase {
	return &BaseRoleCase{
		BaseRoleRepository: baseRoleRepo,
		baseTenantCase:     baseTenantCase,
	}
}

// FindDefaultUser 查询默认租户的普通用户角色。
func (c *BaseRoleCase) FindDefaultUser(ctx context.Context) (*models.BaseRole, error) {
	defaultTenant, err := c.baseTenantCase.FindDefault(ctx)
	if err != nil {
		return nil, err
	}
	roleQuery := c.Query(ctx).BaseRole
	roleOpts := make([]repository.QueryOption, 0, 2)
	roleOpts = append(roleOpts, repository.Where(roleQuery.TenantID.Eq(defaultTenant.ID)))
	roleOpts = append(roleOpts, repository.Where(roleQuery.Code.Eq(_const.BASE_ROLE_CODE_USER)))
	return c.Find(ctx, roleOpts...)
}
