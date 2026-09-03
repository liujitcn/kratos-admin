package biz

import (
	"context"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"

	"github.com/liujitcn/gorm-kit/repository"
	_const "github.com/liujitcn/kratos-core/const"
)

// BaseRoleCase 处理基础角色业务。
type BaseRoleCase struct {
	*biz.BaseCase
	*data.BaseRoleRepository
}

// NewBaseRoleCase 创建基础角色业务实例。
func NewBaseRoleCase(
	baseCase *biz.BaseCase,
	baseRoleRepo *data.BaseRoleRepository,
) *BaseRoleCase {
	return &BaseRoleCase{
		BaseCase:           baseCase,
		BaseRoleRepository: baseRoleRepo,
	}
}

// FindDefaultUser 查询默认租户的普通用户角色。
func (c *BaseRoleCase) FindDefaultUser(ctx context.Context) (*models.BaseRole, error) {
	return c.FindDefaultByCode(ctx, _const.BASE_ROLE_CODE_USER)
}

// FindDefaultByCode 按编码查询默认租户的基础角色。
func (c *BaseRoleCase) FindDefaultByCode(ctx context.Context, roleCode string) (*models.BaseRole, error) {
	roleQuery := c.Query(ctx).BaseRole
	return c.Find(ctx, repository.Where(roleQuery.Code.Eq(roleCode)))
}
