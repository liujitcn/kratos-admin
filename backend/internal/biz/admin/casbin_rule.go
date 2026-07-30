package biz

import (
	"context"

	"github.com/liujitcn/kratos-admin/backend/internal/biz"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"

	_set "github.com/liujitcn/go-utils/set"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
)

// CasbinRuleCase 权限规则业务实例
type CasbinRuleCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.CasbinRuleRepository
	baseMenuRepo   *data.BaseMenuRepository
	baseRoleRepo   *data.BaseRoleRepository
	baseTenantRepo *data.BaseTenantRepository
	baseAPICase    *BaseAPICase
}

// NewCasbinRuleCase 创建权限规则业务实例
func NewCasbinRuleCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	casbinRuleRepo *data.CasbinRuleRepository,
	baseMenuRepo *data.BaseMenuRepository,
	baseRoleRepo *data.BaseRoleRepository,
	baseTenantRepo *data.BaseTenantRepository,
	baseAPICase *BaseAPICase,
) (*CasbinRuleCase, error) {
	return &CasbinRuleCase{
		BaseCase:             baseCase,
		tx:                   tx,
		CasbinRuleRepository: casbinRuleRepo,
		baseMenuRepo:         baseMenuRepo,
		baseRoleRepo:         baseRoleRepo,
		baseTenantRepo:       baseTenantRepo,
		baseAPICase:          baseAPICase,
	}, nil
}

// DeleteCasbinRuleByMenuIDs 按菜单批量删除角色权限
func (c *CasbinRuleCase) DeleteCasbinRuleByMenuIDs(ctx context.Context, menuIDs []int64) error {
	baseRoleList, err := c.baseRoleRepo.List(ctx)
	if err != nil {
		return err
	}

	menuIDSet := _set.NewThreadUnsafeSet(menuIDs...)
	for _, item := range baseRoleList {
		menus := _string.ConvertJsonStringToInt64Array(item.Menus)
		// 角色菜单未命中待删除菜单时，无需重建该角色权限。
		if !menuIDSet.ContainsAny(menus...) {
			continue
		}
		err = c.rebuildCasbinRuleByRole(ctx, item)
		if err != nil {
			return err
		}
	}
	return c.RebuildPolicyRule(ctx)
}

// DeleteCasbinRuleByRoleIDs 按角色批量删除权限规则
func (c *CasbinRuleCase) DeleteCasbinRuleByRoleIDs(ctx context.Context, roleIDs []int64) error {
	baseRoleList, err := c.baseRoleRepo.ListByIDs(ctx, roleIDs)
	if err != nil {
		return err
	}

	// 角色集合为空时，只需要刷新内存权限策略。
	if len(baseRoleList) == 0 {
		return c.RebuildPolicyRule(ctx)
	}

	query := c.Query(ctx).CasbinRule
	for _, item := range baseRoleList {
		var baseTenant *models.BaseTenant
		baseTenant, err = c.baseTenantRepo.FindByID(ctx, item.TenantID)
		if err != nil {
			return err
		}
		opts := make([]repository.QueryOption, 0, 2)
		opts = append(opts, repository.Where(query.V0.Eq(baseTenant.Code)))
		opts = append(opts, repository.Where(query.V1.Eq(item.Code)))
		err = c.Delete(ctx, opts...)
		if err != nil {
			return err
		}
	}
	return c.RebuildPolicyRule(ctx)
}

// RebuildCasbinRuleByMenuID 按菜单重建角色权限
func (c *CasbinRuleCase) RebuildCasbinRuleByMenuID(ctx context.Context, menuID int64) error {
	baseRoleList, err := c.baseRoleRepo.List(ctx)
	if err != nil {
		return err
	}

	for _, item := range baseRoleList {
		menus := _string.ConvertJsonStringToInt64Array(item.Menus)
		// 当前角色未配置目标菜单时，无需重建该角色权限。
		if !_set.NewThreadUnsafeSet(menus...).ContainsOne(menuID) {
			continue
		}
		err = c.rebuildCasbinRuleByRole(ctx, item)
		if err != nil {
			return err
		}
	}
	return c.RebuildPolicyRule(ctx)
}

// RebuildCasbinRuleByRole 按角色重建权限规则
func (c *CasbinRuleCase) RebuildCasbinRuleByRole(ctx context.Context, baseRole *models.BaseRole) error {
	err := c.rebuildCasbinRuleByRole(ctx, baseRole)
	if err != nil {
		return err
	}
	return c.RebuildPolicyRule(ctx)
}

// RebuildCasbinRuleByTenantRole 按指定租户和角色模板重建权限规则。
func (c *CasbinRuleCase) RebuildCasbinRuleByTenantRole(ctx context.Context, tenantCode string, baseRole *models.BaseRole) error {
	err := c.rebuildCasbinRuleByTenantRole(ctx, tenantCode, baseRole)
	if err != nil {
		return err
	}
	return c.RebuildPolicyRule(ctx)
}

// rebuildCasbinRuleByRole 按角色重建数据库权限规则。
func (c *CasbinRuleCase) rebuildCasbinRuleByRole(ctx context.Context, baseRole *models.BaseRole) error {
	baseTenant, err := c.baseTenantRepo.FindByID(ctx, baseRole.TenantID)
	if err != nil {
		return err
	}
	return c.rebuildCasbinRuleByTenantRole(ctx, baseTenant.Code, baseRole)
}

// rebuildCasbinRuleByTenantRole 按指定租户编码和角色模板重建数据库权限规则。
func (c *CasbinRuleCase) rebuildCasbinRuleByTenantRole(ctx context.Context, tenantCode string, baseRole *models.BaseRole) error {
	query := c.Query(ctx).CasbinRule
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.V0.Eq(tenantCode)))
	opts = append(opts, repository.Where(query.V1.Eq(baseRole.Code)))
	err := c.Delete(ctx, opts...)
	if err != nil {
		return err
	}

	menuIDs := _string.ConvertJsonStringToInt64Array(baseRole.Menus)
	// 角色未配置菜单时，只清理数据库权限规则。
	if len(menuIDs) == 0 {
		return nil
	}

	var baseMenuList []*models.BaseMenu
	baseMenuList, err = c.baseMenuRepo.ListByIDs(ctx, menuIDs)
	if err != nil {
		return err
	}

	operations := make([]string, 0)
	for _, item := range baseMenuList {
		operations = append(operations, _string.ConvertJsonStringToStringArray(item.API)...)
	}
	// 菜单未配置接口权限时，只清理数据库权限规则。
	if len(operations) == 0 {
		return nil
	}

	operationSet := _set.NewThreadUnsafeSet(operations...)
	var allAPIList []*models.BaseAPI
	allAPIList, err = c.baseAPICase.List(ctx)
	if err != nil {
		return err
	}

	casbinRuleList := make([]*models.CasbinRule, 0)
	for _, item := range allAPIList {
		// 非当前角色菜单命中的接口不参与规则生成。
		if !operationSet.ContainsOne(item.Operation) {
			continue
		}
		casbinRuleList = append(casbinRuleList, &models.CasbinRule{
			Ptype: "p",
			V0:    tenantCode,
			V1:    baseRole.Code,
			V2:    item.Operation,
			V3:    item.Method,
			V4:    "*",
		})
	}
	// 命中接口规则时，批量写入角色权限规则。
	if len(casbinRuleList) > 0 {
		err = c.BatchCreate(ctx, casbinRuleList)
		if err != nil {
			return err
		}
	}
	return nil
}
