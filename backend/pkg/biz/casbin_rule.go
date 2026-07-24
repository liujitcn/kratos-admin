package biz

import (
	"context"

	_const "github.com/liujitcn/kratos-admin/backend/pkg/const"
	"github.com/liujitcn/kratos-admin/backend/pkg/gen/data"
	"github.com/liujitcn/kratos-admin/backend/pkg/gen/models"

	_string "github.com/liujitcn/go-utils/string"
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
	authzEngine    authzEngine.Engine
}

// NewCasbinRuleCase 创建权限规则业务实例。
func NewCasbinRuleCase(
	casbinRuleRepo *data.CasbinRuleRepository,
	tx data.Transaction,
	baseMenuRepo *data.BaseMenuRepository,
	baseRoleRepo *data.BaseRoleRepository,
	baseTenantRepo *data.BaseTenantRepository,
	baseAPICase *BaseAPICase,
	authzEngine authzEngine.Engine,
) (*CasbinRuleCase, error) {
	return &CasbinRuleCase{
		CasbinRuleRepository: casbinRuleRepo,
		tx:                   tx,
		baseMenuRepo:         baseMenuRepo,
		baseRoleRepo:         baseRoleRepo,
		baseTenantRepo:       baseTenantRepo,
		baseAPICase:          baseAPICase,
		authzEngine:          authzEngine,
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
	return c.rebuildPolicyRule(ctx)
}

// rebuildPolicyRule 重建内存权限策略。
func (c *CasbinRuleCase) rebuildPolicyRule(ctx context.Context) error {
	policyRule := make([]casbin.PolicyRule, 0)
	// 查询全部 API，默认给 super 配置
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
	// 查询 casbin
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
	return c.authzEngine.SetPolicies(ctx, policyMap, roleMap)
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
