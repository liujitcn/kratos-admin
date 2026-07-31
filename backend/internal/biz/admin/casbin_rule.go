package biz

import (
	"context"
	"fmt"

	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"

	_set "github.com/liujitcn/go-utils/set"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	authzEngine "github.com/liujitcn/kratos-kit/auth/authz/engine"
	"github.com/liujitcn/kratos-kit/auth/authz/engine/casbin"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
)

// CasbinRuleCase 权限规则业务实例
type CasbinRuleCase struct {
	*data.CasbinRuleRepository
	tx             data.Transaction
	baseMenuRepo   *data.BaseMenuRepository
	baseRoleRepo   *data.BaseRoleRepository
	baseTenantRepo *data.BaseTenantRepository
	baseAPICase    *BaseAPICase
	policyWriter   authzEngine.PolicyWriter
}

// NewCasbinRuleCase 创建权限规则业务实例。
func NewCasbinRuleCase(
	casbinRuleRepo *data.CasbinRuleRepository,
	tx data.Transaction,
	baseMenuRepo *data.BaseMenuRepository,
	baseRoleRepo *data.BaseRoleRepository,
	baseTenantRepo *data.BaseTenantRepository,
	baseAPICase *BaseAPICase,
	authorizer authzEngine.Engine,
) (*CasbinRuleCase, error) {
	policyWriter, ok := authorizer.(authzEngine.PolicyWriter)
	if !ok {
		return nil, errorsx.Internal("鉴权引擎不支持策略写入").WithCause(fmt.Errorf("authz engine %T does not implement PolicyWriter", authorizer))
	}
	return &CasbinRuleCase{
		CasbinRuleRepository: casbinRuleRepo,
		tx:                   tx,
		baseMenuRepo:         baseMenuRepo,
		baseRoleRepo:         baseRoleRepo,
		baseTenantRepo:       baseTenantRepo,
		baseAPICase:          baseAPICase,
		policyWriter:         policyWriter,
	}, nil
}

// RebuildAllCasbinRules 按全部角色、菜单和接口重新初始化 Casbin 规则与内存策略。
//
// 该方法仅在服务启动时调用，必须位于 OpenAPI 接口同步和租户管理员菜单同步之后。它会以当前
// 数据库数据覆盖 casbin_rule 表，并在规则写入成功后加载 Casbin 内存策略。
func (c *CasbinRuleCase) RebuildAllCasbinRules(ctx context.Context) error {
	baseRoleList, err := c.baseRoleRepo.List(ctx)
	if err != nil {
		return err
	}
	var baseTenantList []*models.BaseTenant
	baseTenantList, err = c.baseTenantRepo.List(ctx)
	if err != nil {
		return err
	}

	// 仅查询角色实际关联的菜单，减少无关菜单参与规则构建。
	menuIDSet := make(map[int64]struct{})
	for _, item := range baseRoleList {
		for _, menuID := range _string.ConvertJsonStringToInt64Array(item.Menus) {
			menuIDSet[menuID] = struct{}{}
		}
	}
	menuIDs := make([]int64, 0, len(menuIDSet))
	for menuID := range menuIDSet {
		menuIDs = append(menuIDs, menuID)
	}
	var baseMenuList []*models.BaseMenu
	baseMenuList, err = c.baseMenuRepo.ListByIDs(ctx, menuIDs)
	if err != nil {
		return err
	}
	var baseAPIList []*models.BaseAPI
	baseAPIList, err = c.baseAPICase.List(ctx)
	if err != nil {
		return err
	}

	// 根据读取到的角色、租户、菜单和 API 数据构造完整规则快照。
	casbinRuleList := buildCasbinRuleList(baseRoleList, baseTenantList, baseMenuList, baseAPIList)
	query := c.Query(ctx).CasbinRule
	// 策略完全根据当前角色、菜单和接口重建，清空表并重置自增 ID 后重新生成。
	if err = query.WithContext(ctx).UnderlyingDB().Exec("TRUNCATE TABLE `casbin_rule`").Error; err != nil { //nolint:forbidigo // TRUNCATE 重置自增 ID，gorm/gen 无法表达
		return err
	}
	err = c.tx.Transaction(ctx, func(ctx context.Context) error {
		return c.BatchCreate(ctx, casbinRuleList)
	})
	if err != nil {
		return err
	}
	// 数据库规则提交后再刷新内存策略，确保鉴权引擎看到完整且一致的规则集合。
	return c.RebuildPolicyRule(ctx)
}

// RebuildPolicyRule 重建内存权限策略。
func (c *CasbinRuleCase) RebuildPolicyRule(ctx context.Context) error {
	policyRule := make([]casbin.PolicyRule, 0)
	// 查询全部 API，默认给 super 配置。
	baseAPIList, err := c.baseAPICase.List(ctx)
	if err != nil {
		return err
	}
	for _, item := range baseAPIList {
		policyRule = append(policyRule, casbin.PolicyRule{
			PType: "p",
			V0:    databaseGorm.DefaultTenantCode,
			V1:    _const.BASE_ROLE_CODE_SUPER,
			V2:    item.Operation,
			V3:    item.Method,
			V4:    "*",
		})
	}
	var casbinRuleList []*models.CasbinRule
	casbinRuleList, err = c.List(ctx)
	if err != nil {
		return err
	}
	for _, item := range casbinRuleList {
		// 旧版本策略缺少租户或项目占位字段时会被 Casbin 识别为 4 段规则，启动阶段直接跳过等待角色权限重建修复。
		if item.Ptype == "" || item.V0 == "" || item.V1 == "" || item.V2 == "" || item.V3 == "" || item.V4 == "" {
			continue
		}
		policyRule = append(policyRule, casbin.PolicyRule{
			PType: item.Ptype,
			V0:    item.V0,
			V1:    item.V1,
			V2:    item.V2,
			V3:    item.V3,
			V4:    item.V4,
		})
	}
	policyMap := make(authzEngine.PolicyMap)
	policyMap["policies"] = policyRule
	roleMap := make(authzEngine.RoleMap)
	return c.policyWriter.SetPolicies(ctx, policyMap, roleMap)
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

// buildCasbinRuleList 根据角色菜单、租户和接口关联构造去重后的 Casbin 策略。
func buildCasbinRuleList(baseRoleList []*models.BaseRole, baseTenantList []*models.BaseTenant, baseMenuList []*models.BaseMenu, baseAPIList []*models.BaseAPI) []*models.CasbinRule {
	tenantCodeByID := make(map[int64]string, len(baseTenantList))
	for _, item := range baseTenantList {
		tenantCodeByID[item.ID] = item.Code
	}
	menuOperationsByID := make(map[int64][]string, len(baseMenuList))
	for _, item := range baseMenuList {
		menuOperationsByID[item.ID] = _string.ConvertJsonStringToStringArray(item.API)
	}
	apiByOperation := make(map[string]*models.BaseAPI, len(baseAPIList))
	for _, item := range baseAPIList {
		if _, ok := apiByOperation[item.Operation]; !ok {
			apiByOperation[item.Operation] = item
		}
	}

	rules := make([]*models.CasbinRule, 0)
	ruleSet := make(map[string]struct{})
	for _, baseRole := range baseRoleList {
		tenantCode, ok := tenantCodeByID[baseRole.TenantID]
		// 角色所属租户不存在时，不生成无效策略。
		if !ok {
			continue
		}
		for _, menuID := range _string.ConvertJsonStringToInt64Array(baseRole.Menus) {
			for _, operation := range menuOperationsByID[menuID] {
				baseAPI, ok := apiByOperation[operation]
				// 菜单关联的接口已失效时，不生成无效策略。
				if !ok {
					continue
				}
				ruleKey := tenantCode + "\x00" + baseRole.Code + "\x00" + baseAPI.Operation + "\x00" + baseAPI.Method
				if _, ok = ruleSet[ruleKey]; ok {
					continue
				}
				ruleSet[ruleKey] = struct{}{}
				rules = append(rules, &models.CasbinRule{
					Ptype: "p",
					V0:    tenantCode,
					V1:    baseRole.Code,
					V2:    baseAPI.Operation,
					V3:    baseAPI.Method,
					V4:    "*",
				})
			}
		}
	}
	return rules
}
