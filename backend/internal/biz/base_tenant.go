package biz

import (
	"context"
	"errors"

	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"

	"github.com/liujitcn/gorm-kit/repository"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/gorm"
)

// BaseTenantCase 提供启动期租户基础数据同步能力。
type BaseTenantCase struct {
	tx             data.Transaction
	baseRoleRepo   *data.BaseRoleRepository
	baseTenantRepo *data.BaseTenantRepository
}

// NewBaseTenantCase 创建启动期租户基础数据同步实例。
func NewBaseTenantCase(
	tx data.Transaction,
	baseRoleRepo *data.BaseRoleRepository,
	baseTenantRepo *data.BaseTenantRepository,
) *BaseTenantCase {
	return &BaseTenantCase{
		tx:             tx,
		baseRoleRepo:   baseRoleRepo,
		baseTenantRepo: baseTenantRepo,
	}
}

// FindDefault 查询默认租户。
func (c *BaseTenantCase) FindDefault(ctx context.Context) (*models.BaseTenant, error) {
	tenantQuery := c.baseTenantRepo.Query(ctx).BaseTenant
	tenantOpts := make([]repository.QueryOption, 0, 1)
	tenantOpts = append(tenantOpts, repository.Where(tenantQuery.Code.Eq(databaseGorm.DefaultTenantCode)))
	return c.baseTenantRepo.Find(ctx, tenantOpts...)
}

// SyncTenantRoleMenus 将默认租户管理员角色菜单同步到所有普通租户的角色副本。
//
// 该方法仅在服务启动时调用，必须位于 OpenAPI 接口同步之后、全量 Casbin 规则重建之前。
// 默认租户或角色模板尚未初始化时返回 nil，使首次导入初始化数据前的启动流程保持幂等。
func (c *BaseTenantCase) SyncTenantRoleMenus(ctx context.Context) error {
	tenantQuery := c.baseTenantRepo.Query(ctx).BaseTenant
	tenantOpts := make([]repository.QueryOption, 0, 1)
	tenantOpts = append(tenantOpts, repository.Where(tenantQuery.Code.Eq(databaseGorm.DefaultTenantCode)))
	defaultTenant, err := c.baseTenantRepo.Find(ctx, tenantOpts...)
	// 首次启动尚未导入初始化数据时没有默认租户，等待后续启动再同步。
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return errorsx.Internal("查询默认租户失败").WithCause(err)
	}

	roleQuery := c.baseRoleRepo.Query(ctx).BaseRole
	roleOpts := make([]repository.QueryOption, 0, 2)
	roleOpts = append(roleOpts, repository.Where(roleQuery.TenantID.Eq(defaultTenant.ID)))
	roleOpts = append(roleOpts, repository.Where(roleQuery.Code.Eq(_const.BASE_ROLE_CODE_TENANT)))
	var templateRole *models.BaseRole
	templateRole, err = c.baseRoleRepo.Find(ctx, roleOpts...)
	// 初始化数据尚未写入租户角色模板时无需执行同步。
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		allRoleOpts := make([]repository.QueryOption, 0, 1)
		allRoleOpts = append(allRoleOpts, repository.Where(roleQuery.Code.Eq(_const.BASE_ROLE_CODE_TENANT)))
		var baseRoleList []*models.BaseRole
		baseRoleList, err = c.baseRoleRepo.List(ctx, allRoleOpts...)
		if err != nil {
			return err
		}
		for _, item := range baseRoleList {
			// 默认租户模板和已同步副本无需重复写入。
			if item.ID == templateRole.ID || item.Menus == templateRole.Menus {
				continue
			}
			err = c.baseRoleRepo.UpdateByID(ctx, &models.BaseRole{
				ID:       item.ID,
				TenantID: item.TenantID,
				Menus:    templateRole.Menus,
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}
