package biz

import (
	"context"

	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"

	"github.com/liujitcn/gorm-kit/repository"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
)

// BaseRoleCase 处理基础角色业务。
type BaseRoleCase struct {
	*data.BaseRoleRepository
	baseTenantRepo *data.BaseTenantRepository
}

// NewBaseRoleCase 创建基础角色业务实例。
func NewBaseRoleCase(
	baseRoleRepo *data.BaseRoleRepository,
	baseTenantRepo *data.BaseTenantRepository,
) *BaseRoleCase {
	return &BaseRoleCase{
		BaseRoleRepository: baseRoleRepo,
		baseTenantRepo:     baseTenantRepo,
	}
}

// FindDefaultUser 查询默认租户的普通用户角色。
func (c *BaseRoleCase) FindDefaultUser(ctx context.Context) (*models.BaseRole, error) {
	return c.FindDefaultByCode(ctx, _const.BASE_ROLE_CODE_USER)
}

// FindDefaultByCode 按编码查询默认租户的基础角色。
func (c *BaseRoleCase) FindDefaultByCode(ctx context.Context, roleCode string) (*models.BaseRole, error) {
	tenantQuery := c.baseTenantRepo.Query(ctx).BaseTenant
	tenantOpts := make([]repository.QueryOption, 0, 1)
	tenantOpts = append(tenantOpts, repository.Where(tenantQuery.Code.Eq(databaseGorm.DefaultTenantCode)))
	defaultTenant, err := c.baseTenantRepo.Find(ctx, tenantOpts...)
	if err != nil {
		return nil, err
	}
	roleQuery := c.Query(ctx).BaseRole
	roleOpts := make([]repository.QueryOption, 0, 2)
	roleOpts = append(roleOpts, repository.Where(roleQuery.TenantID.Eq(defaultTenant.ID)))
	roleOpts = append(roleOpts, repository.Where(roleQuery.Code.Eq(roleCode)))
	return c.Find(ctx, roleOpts...)
}
