package biz

import (
	"context"
	"fmt"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/liujitcn/go-utils/set"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/auth/authz/engine"
	"github.com/liujitcn/kratos-kit/auth/authz/engine/casbin"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

// CasbinRuleCase 权限规则业务实例
type CasbinRuleCase struct {
	*data.CasbinRuleRepository
	baseMenuRepo   *data.BaseMenuRepository
	baseRoleRepo   *data.BaseRoleRepository
	baseTenantRepo *data.BaseTenantRepository
	baseAPICase    *BaseAPICase
	policyWriter   engine.PolicyWriter
}

// NewCasbinRuleCase 创建权限规则业务实例。
func NewCasbinRuleCase(
	casbinRuleRepo *data.CasbinRuleRepository,
	baseMenuRepo *data.BaseMenuRepository,
	baseRoleRepo *data.BaseRoleRepository,
	baseTenantRepo *data.BaseTenantRepository,
	baseAPICase *BaseAPICase,
	authorizer engine.Engine,
) (*CasbinRuleCase, error) {
	policyWriter, ok := authorizer.(engine.PolicyWriter)
	if !ok {
		return nil, errorsx.Internal("鉴权引擎不支持策略写入").WithCause(fmt.Errorf("authz engine %T does not implement PolicyWriter", authorizer))
	}
	return &CasbinRuleCase{
		CasbinRuleRepository: casbinRuleRepo,
		baseMenuRepo:         baseMenuRepo,
		baseRoleRepo:         baseRoleRepo,
		baseTenantRepo:       baseTenantRepo,
		baseAPICase:          baseAPICase,
		policyWriter:         policyWriter,
	}, nil
}

// RebuildPolicyRule 重建内存中的 Casbin 策略。
func (c *CasbinRuleCase) RebuildPolicyRule(ctx context.Context) error {
	policyRules := make([]casbin.PolicyRule, 0)
	baseAPIList, err := c.baseAPICase.List(ctx)
	if err != nil {
		return err
	}
	for _, item := range baseAPIList {
		policyRules = append(policyRules, casbin.PolicyRule{PType: "p", V0: gorm.DefaultTenantCode, V1: _const.BASE_ROLE_CODE_SUPER, V2: item.Operation, V3: item.Method, V4: "*"})
	}
	var casbinRuleList []*models.CasbinRule
	casbinRuleList, err = c.List(ctx)
	if err != nil {
		return err
	}
	for _, item := range casbinRuleList {
		if item.Ptype == "" || item.V0 == "" || item.V1 == "" || item.V2 == "" || item.V3 == "" || item.V4 == "" {
			continue
		}
		policyRules = append(policyRules, casbin.PolicyRule{PType: item.Ptype, V0: item.V0, V1: item.V1, V2: item.V2, V3: item.V3, V4: item.V4})
	}
	return c.policyWriter.SetPolicies(ctx, engine.PolicyMap{"policies": policyRules}, engine.RoleMap{})
}

// DeleteCasbinRuleByMenuIDs 按菜单批量删除角色权限
func (c *CasbinRuleCase) DeleteCasbinRuleByMenuIDs(ctx context.Context, menuIDs []int64) error {
	baseRoleList, err := c.baseRoleRepo.List(ctx)
	if err != nil {
		return err
	}

	menuIDSet := set.NewThreadUnsafeSet(menuIDs...)
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
		if !set.NewThreadUnsafeSet(menus...).ContainsOne(menuID) {
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

	operationSet := set.NewThreadUnsafeSet(operations...)
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
