package biz

import (
	"context"

	"github.com/liujitcn/go-utils/crypto"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/loginpolicy"
	passwordPolicy "github.com/liujitcn/kratos-admin/backend/internal/biz/base/password"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/utils"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	corebiz "github.com/liujitcn/kratos-core/biz"
	coreconst "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	"google.golang.org/protobuf/types/known/emptypb"
)

// BaseLoginPolicyCase 提供平台管理员登录策略的逐条管理能力。
type BaseLoginPolicyCase struct {
	*corebiz.BaseCase
	tx                      data.Transaction
	baseLoginPolicyRepo     *data.BaseLoginPolicyRepository
	baseLoginPolicyRuleRepo *data.BaseLoginPolicyRuleRepository
	baseTenantRepo          *data.BaseTenantRepository
	baseUserRepo            *data.BaseUserRepository
}

// NewBaseLoginPolicyCase 创建登录策略业务实例。
func NewBaseLoginPolicyCase(
	baseCase *corebiz.BaseCase,
	tx data.Transaction,
	baseLoginPolicyRepo *data.BaseLoginPolicyRepository,
	baseLoginPolicyRuleRepo *data.BaseLoginPolicyRuleRepository,
	baseTenantRepo *data.BaseTenantRepository,
	baseUserRepo *data.BaseUserRepository,
) *BaseLoginPolicyCase {
	return &BaseLoginPolicyCase{BaseCase: baseCase, tx: tx, baseLoginPolicyRepo: baseLoginPolicyRepo, baseLoginPolicyRuleRepo: baseLoginPolicyRuleRepo, baseTenantRepo: baseTenantRepo, baseUserRepo: baseUserRepo}
}

// PageBaseLoginPolicy 分页查询登录策略及其限制规则。
func (c *BaseLoginPolicyCase) PageBaseLoginPolicy(ctx context.Context, req *adminv1.PageBaseLoginPolicyRequest) (*adminv1.PageBaseLoginPolicyResponse, error) {
	err := c.ensurePlatformOperator(ctx)
	if err != nil {
		return nil, err
	}
	query := c.baseLoginPolicyRepo.Query(ctx).BaseLoginPolicy
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Order(query.ScopeType.Asc()), repository.Order(query.ID.Asc()))
	if req.ScopeType != nil {
		opts = append(opts, repository.Where(query.ScopeType.Eq(int32(req.GetScopeType()))))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	list, total, err := c.baseLoginPolicyRepo.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	result := make([]*adminv1.BaseLoginPolicy, 0, len(list))
	for _, entity := range list {
		var item *adminv1.BaseLoginPolicy
		item, err = c.toBaseLoginPolicy(ctx, entity)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return &adminv1.PageBaseLoginPolicyResponse{BaseLoginPolicies: result, Total: int32(total)}, nil
}

// GetBaseLoginPolicy 查询登录策略详情及其限制规则。
func (c *BaseLoginPolicyCase) getBaseLoginPolicy(ctx context.Context, id int64) (*adminv1.BaseLoginPolicyForm, error) {
	err := c.ensurePlatformOperator(ctx)
	if err != nil {
		return nil, err
	}
	entity, err := c.baseLoginPolicyRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	rules, err := c.listRules(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.toBaseLoginPolicyForm(entity, rules)
}

// CreateBaseLoginPolicy 创建登录策略及其限制规则。
func (c *BaseLoginPolicyCase) createBaseLoginPolicy(ctx context.Context, input *adminv1.BaseLoginPolicyForm) error {
	err := c.ensurePlatformOperator(ctx)
	if err != nil {
		return err
	}
	policy, err := c.policyFromForm(ctx, input)
	if err != nil {
		return err
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	entity := policyToEntity(policy)
	entity.CreatedBy = authInfo.UserId
	entity.UpdatedBy = authInfo.UserId
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		err = c.baseLoginPolicyRepo.Create(txCtx, entity)
		if err != nil {
			if errorsx.IsDuplicateKey(err) {
				return errorsx.UniqueConflict("同一作用域的登录策略已存在", "base_login_policy", "scope_type,tenant_id,user_id", "unique_base_login_policy").WithCause(err)
			}
			return err
		}
		for _, rule := range policy.Rules {
			ruleEntity := ruleToEntity(rule)
			ruleEntity.ID = 0
			ruleEntity.PolicyID = entity.ID
			ruleEntity.CreatedBy = authInfo.UserId
			ruleEntity.UpdatedBy = authInfo.UserId
			err = c.baseLoginPolicyRuleRepo.Create(txCtx, ruleEntity)
			if err != nil {
				if errorsx.IsDuplicateKey(err) {
					return errorsx.UniqueConflict("登录策略限制规则重复", "base_login_policy_rule", "", "unique_base_login_policy_rule").WithCause(err)
				}
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return c.RefreshBaseLoginPolicy(ctx)
}

// UpdateBaseLoginPolicy 更新登录策略及其限制规则。
func (c *BaseLoginPolicyCase) updateBaseLoginPolicy(ctx context.Context, input *adminv1.BaseLoginPolicyForm) error {
	err := c.ensurePlatformOperator(ctx)
	if err != nil {
		return err
	}
	oldEntity, err := c.baseLoginPolicyRepo.FindByID(ctx, input.GetId())
	if err != nil {
		return err
	}
	policy, err := c.policyFromForm(ctx, input)
	if err != nil {
		return err
	}
	if input.GetInitialPassword() == nil {
		policy.InitialPasswordHash = oldEntity.InitialPasswordHash
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	entity := policyToEntity(policy)
	entity.ID = oldEntity.ID
	entity.CreatedBy = oldEntity.CreatedBy
	entity.CreatedAt = oldEntity.CreatedAt
	entity.UpdatedBy = authInfo.UserId
	oldRules, err := c.listRules(ctx, oldEntity.ID)
	if err != nil {
		return err
	}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		err = c.baseLoginPolicyRepo.UpdateByID(txCtx, entity)
		if err != nil {
			if errorsx.IsDuplicateKey(err) {
				return errorsx.UniqueConflict("同一作用域的登录策略已存在", "base_login_policy", "scope_type,tenant_id,user_id", "unique_base_login_policy").WithCause(err)
			}
			return err
		}
		if len(oldRules) > 0 {
			ruleIDs := make([]int64, 0, len(oldRules))
			for _, rule := range oldRules {
				ruleIDs = append(ruleIDs, rule.ID)
			}
			err = c.baseLoginPolicyRuleRepo.DeleteByIDs(txCtx, ruleIDs)
			if err != nil {
				return err
			}
		}
		for _, rule := range policy.Rules {
			ruleEntity := ruleToEntity(rule)
			ruleEntity.ID = 0
			ruleEntity.PolicyID = entity.ID
			ruleEntity.CreatedBy = oldEntity.CreatedBy
			ruleEntity.UpdatedBy = authInfo.UserId
			err = c.baseLoginPolicyRuleRepo.Create(txCtx, ruleEntity)
			if err != nil {
				if errorsx.IsDuplicateKey(err) {
					return errorsx.UniqueConflict("登录策略限制规则重复", "base_login_policy_rule", "", "unique_base_login_policy_rule").WithCause(err)
				}
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return c.RefreshBaseLoginPolicy(ctx)
}

// DeleteBaseLoginPolicy 删除登录策略及其限制规则。
func (c *BaseLoginPolicyCase) deleteBaseLoginPolicy(ctx context.Context, id string) error {
	err := c.ensurePlatformOperator(ctx)
	if err != nil {
		return err
	}
	ids := _string.ConvertStringToInt64Array(id)
	if len(ids) == 0 {
		return errorsx.InvalidArgument("登录策略ID不能为空")
	}
	list, err := c.baseLoginPolicyRepo.ListByIDs(ctx, ids)
	if err != nil {
		return err
	}
	if len(list) != len(ids) {
		return errorsx.ResourceNotFound("登录策略不存在")
	}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		var rules []*models.BaseLoginPolicyRule
		rules, err = c.listRulesByPolicyIDs(txCtx, ids)
		if err != nil {
			return err
		}
		if len(rules) > 0 {
			ruleIDs := make([]int64, 0, len(rules))
			for _, rule := range rules {
				ruleIDs = append(ruleIDs, rule.ID)
			}
			err = c.baseLoginPolicyRuleRepo.DeleteByIDs(txCtx, ruleIDs)
			if err != nil {
				return err
			}
		}
		return c.baseLoginPolicyRepo.DeleteByIDs(txCtx, ids)
	})
	if err != nil {
		return err
	}
	return c.RefreshBaseLoginPolicy(ctx)
}

// SetBaseLoginPolicyStatus 设置登录策略状态。
func (c *BaseLoginPolicyCase) setBaseLoginPolicyStatus(ctx context.Context, req *adminv1.SetBaseLoginPolicyStatusRequest) error {
	err := c.ensurePlatformOperator(ctx)
	if err != nil {
		return err
	}
	status := int32(req.GetStatus())
	if status != coreconst.STATUS_STATUS_ENABLE && status != coreconst.STATUS_STATUS_DISABLE {
		return errorsx.InvalidArgument("登录策略状态无效")
	}
	entity, err := c.baseLoginPolicyRepo.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	if entity.Status == status {
		return nil
	}
	err = c.baseLoginPolicyRepo.UpdateByID(ctx, &models.BaseLoginPolicy{ID: entity.ID, Status: status})
	if err != nil {
		return err
	}
	return c.RefreshBaseLoginPolicy(ctx)
}

// RefreshBaseLoginPolicy 从数据库加载全部登录策略到运行时缓存。
func (c *BaseLoginPolicyCase) RefreshBaseLoginPolicy(ctx context.Context) error {
	query := c.baseLoginPolicyRepo.Query(ctx).BaseLoginPolicy
	policies, err := c.baseLoginPolicyRepo.List(ctx, repository.Order(query.ID.Asc()))
	if err != nil {
		return err
	}
	policySet := loginpolicy.PolicySet{Policies: make([]loginpolicy.Policy, 0, len(policies))}
	for _, entity := range policies {
		var rules []*models.BaseLoginPolicyRule
		rules, err = c.listRules(ctx, entity.ID)
		if err != nil {
			return err
		}
		var policy loginpolicy.Policy
		policy = entityToPolicy(entity, rules)
		policySet.Policies = append(policySet.Policies, policy)
	}
	if len(policySet.Policies) == 0 {
		policySet = loginpolicy.Load()
	}
	return loginpolicy.SaveToCache(c.Cache, policySet)
}

// GetBaseLoginPolicy 查询登录策略详情。
func (c *BaseLoginPolicyCase) GetBaseLoginPolicy(ctx context.Context, req *adminv1.GetBaseLoginPolicyRequest) (*adminv1.BaseLoginPolicyForm, error) {
	return c.getBaseLoginPolicy(ctx, req.GetId())
}

// CreateBaseLoginPolicy 创建登录策略。
func (c *BaseLoginPolicyCase) CreateBaseLoginPolicy(ctx context.Context, req *adminv1.CreateBaseLoginPolicyRequest) (*emptypb.Empty, error) {
	if err := c.createBaseLoginPolicy(ctx, req.GetBaseLoginPolicy()); err != nil {
		return nil, err
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseLoginPolicy 更新登录策略。
func (c *BaseLoginPolicyCase) UpdateBaseLoginPolicy(ctx context.Context, req *adminv1.UpdateBaseLoginPolicyRequest) (*emptypb.Empty, error) {
	if err := c.updateBaseLoginPolicy(ctx, req.GetBaseLoginPolicy()); err != nil {
		return nil, err
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseLoginPolicy 删除登录策略。
func (c *BaseLoginPolicyCase) DeleteBaseLoginPolicy(ctx context.Context, req *adminv1.DeleteBaseLoginPolicyRequest) (*emptypb.Empty, error) {
	if err := c.deleteBaseLoginPolicy(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return new(emptypb.Empty), nil
}

// SetBaseLoginPolicyStatus 设置登录策略状态。
func (c *BaseLoginPolicyCase) SetBaseLoginPolicyStatus(ctx context.Context, req *adminv1.SetBaseLoginPolicyStatusRequest) (*emptypb.Empty, error) {
	if err := c.setBaseLoginPolicyStatus(ctx, req); err != nil {
		return nil, err
	}
	return new(emptypb.Empty), nil
}

// listRules 查询指定策略的限制规则。
func (c *BaseLoginPolicyCase) listRules(ctx context.Context, policyID int64) ([]*models.BaseLoginPolicyRule, error) {
	query := c.baseLoginPolicyRuleRepo.Query(ctx).BaseLoginPolicyRule
	return c.baseLoginPolicyRuleRepo.List(ctx, repository.Where(query.PolicyID.Eq(policyID)), repository.Order(query.ID.Asc()))
}

// listRulesByPolicyIDs 查询多条策略的限制规则。
func (c *BaseLoginPolicyCase) listRulesByPolicyIDs(ctx context.Context, policyIDs []int64) ([]*models.BaseLoginPolicyRule, error) {
	query := c.baseLoginPolicyRuleRepo.Query(ctx).BaseLoginPolicyRule
	return c.baseLoginPolicyRuleRepo.List(ctx, repository.Where(query.PolicyID.In(policyIDs...)))
}

// policyFromForm 将接口表单转换为策略领域记录并校验作用域。
func (c *BaseLoginPolicyCase) policyFromForm(ctx context.Context, input *adminv1.BaseLoginPolicyForm) (loginpolicy.Policy, error) {
	policy := loginpolicy.Policy{ID: input.GetId(), ScopeType: int32(input.GetScopeType()), TenantID: input.GetTenantId(), UserID: input.GetUserId(), Status: int32(input.GetStatus()), MaxFailedAttempts: input.GetMaxFailedAttempts(), LockDurationMinutes: input.GetLockDurationMinutes(), PasswordMinLength: input.GetPasswordMinLength(), PasswordHistoryCount: input.GetPasswordHistoryCount(), PasswordMinComplexityClasses: input.GetPasswordMinComplexityClasses(), PasswordMaxAgeDays: input.GetPasswordMaxAgeDays(), Rules: make([]loginpolicy.Rule, 0, len(input.GetRules()))}
	var err error
	if policy.Status == 0 {
		policy.Status = coreconst.STATUS_STATUS_ENABLE
	}
	if policy.MaxFailedAttempts == 0 {
		policy.MaxFailedAttempts = loginpolicy.DefaultMaxFailedAttempts
	}
	if policy.LockDurationMinutes == 0 {
		policy.LockDurationMinutes = loginpolicy.DefaultLockDurationMinutes
	}
	if input.PasswordMinLength != nil && policy.PasswordMinLength <= 0 {
		return loginpolicy.Policy{}, errorsx.InvalidArgument("密码最小长度必须大于零")
	}
	if input.PasswordMinLength == nil {
		policy.PasswordMinLength = loginpolicy.DefaultPasswordMinLength
	}
	if input.PasswordHistoryCount == nil {
		policy.PasswordHistoryCount = loginpolicy.DefaultPasswordHistoryCount
	}
	if input.PasswordMinComplexityClasses != nil && policy.PasswordMinComplexityClasses <= 0 {
		return loginpolicy.Policy{}, errorsx.InvalidArgument("密码复杂度字符类别数量必须大于零")
	}
	if input.PasswordMinComplexityClasses == nil {
		policy.PasswordMinComplexityClasses = loginpolicy.DefaultPasswordMinComplexityClasses
	}
	if input.PasswordMaxAgeDays == nil {
		policy.PasswordMaxAgeDays = loginpolicy.DefaultPasswordMaxAgeDays
	}
	if input.GetInitialPassword() != nil {
		var initialPassword string
		initialPassword, err = utils.DecryptPassword(c.Cache, input.GetInitialPassword(), basev1.PasswordCryptoScene_PASSWORD_CRYPTO_SCENE_CONFIGURE_PASSWORD_POLICY)
		if err != nil {
			return loginpolicy.Policy{}, err
		}
		config := loginpolicy.PasswordConfig{MinLength: policy.PasswordMinLength, MinComplexityClasses: policy.PasswordMinComplexityClasses}
		if err = passwordPolicy.ValidateComplexity(initialPassword, config); err != nil {
			return loginpolicy.Policy{}, errorsx.InvalidArgument("初始化密码长度或复杂度不符合安全策略").WithCause(err)
		}
		policy.InitialPasswordHash, err = crypto.Encrypt(initialPassword)
		if err != nil {
			return loginpolicy.Policy{}, errorsx.Internal("保存初始化密码失败").WithCause(err)
		}
	}
	for _, inputRule := range input.GetRules() {
		status := int32(inputRule.GetStatus())
		if status == 0 {
			status = coreconst.STATUS_STATUS_ENABLE
		}
		policy.Rules = append(policy.Rules, loginpolicy.Rule{ID: inputRule.GetId(), PolicyID: inputRule.GetPolicyId(), RestrictionType: int32(inputRule.GetRestrictionType()), RestrictionMethod: int32(inputRule.GetRestrictionMethod()), RestrictionValue: inputRule.GetRestrictionValue(), Reason: inputRule.GetReason(), Status: status})
	}
	err = validatePolicyTarget(ctx, policy, c.baseTenantRepo, c.baseUserRepo)
	if err != nil {
		return loginpolicy.Policy{}, err
	}
	err = (loginpolicy.PolicySet{Policies: []loginpolicy.Policy{policy}}).Validate()
	if err != nil {
		return loginpolicy.Policy{}, errorsx.InvalidArgument("登录策略格式无效").WithCause(err)
	}
	return policy, nil
}

// toBaseLoginPolicy 将数据库记录转换为列表项并补充目标名称。
func (c *BaseLoginPolicyCase) toBaseLoginPolicy(ctx context.Context, entity *models.BaseLoginPolicy) (*adminv1.BaseLoginPolicy, error) {
	rules, err := c.listRules(ctx, entity.ID)
	if err != nil {
		return nil, err
	}
	policy := entityToPolicy(entity, rules)
	result := &adminv1.BaseLoginPolicy{Id: policy.ID, ScopeType: adminv1.BaseLoginPolicyScopeType(policy.ScopeType), TenantId: policy.TenantID, UserId: policy.UserID, MaxFailedAttempts: policy.MaxFailedAttempts, LockDurationMinutes: policy.LockDurationMinutes, PasswordMinLength: policy.PasswordMinLength, PasswordHistoryCount: policy.PasswordHistoryCount, PasswordMinComplexityClasses: policy.PasswordMinComplexityClasses, PasswordMaxAgeDays: policy.PasswordMaxAgeDays, Status: commonv1.Status(policy.Status), CreatedAt: entity.CreatedAt.Format("2006-01-02 15:04:05"), UpdatedAt: entity.UpdatedAt.Format("2006-01-02 15:04:05")}
	result.Rules = make([]*adminv1.BaseLoginPolicyRule, 0, len(policy.Rules))
	for _, rule := range policy.Rules {
		result.Rules = append(result.Rules, toBaseLoginPolicyRule(rule))
	}
	if policy.TenantID > 0 {
		var tenant *models.BaseTenant
		tenant, err = c.baseTenantRepo.FindByID(ctx, policy.TenantID)
		if err != nil {
			return nil, err
		}
		result.TenantName = tenant.Name
	}
	if policy.UserID > 0 {
		var user *models.BaseUser
		user, err = c.baseUserRepo.FindByID(ctx, policy.UserID)
		if err != nil {
			return nil, err
		}
		result.UserName = user.UserName
	}
	return result, nil
}

// toBaseLoginPolicyForm 将数据库记录转换为编辑表单。
func (c *BaseLoginPolicyCase) toBaseLoginPolicyForm(entity *models.BaseLoginPolicy, rules []*models.BaseLoginPolicyRule) (*adminv1.BaseLoginPolicyForm, error) {
	policy := entityToPolicy(entity, rules)
	passwordMaxAgeDays := policy.PasswordMaxAgeDays
	passwordMinLength := policy.PasswordMinLength
	passwordHistoryCount := policy.PasswordHistoryCount
	passwordMinComplexityClasses := policy.PasswordMinComplexityClasses
	result := &adminv1.BaseLoginPolicyForm{Id: policy.ID, ScopeType: adminv1.BaseLoginPolicyScopeType(policy.ScopeType), TenantId: policy.TenantID, UserId: policy.UserID, MaxFailedAttempts: policy.MaxFailedAttempts, LockDurationMinutes: policy.LockDurationMinutes, Status: commonv1.Status(policy.Status), PasswordMinLength: &passwordMinLength, PasswordHistoryCount: &passwordHistoryCount, PasswordMinComplexityClasses: &passwordMinComplexityClasses, PasswordMaxAgeDays: &passwordMaxAgeDays}
	result.Rules = make([]*adminv1.BaseLoginPolicyRule, 0, len(policy.Rules))
	for _, rule := range policy.Rules {
		result.Rules = append(result.Rules, toBaseLoginPolicyRule(rule))
	}
	return result, nil
}

// ensurePlatformOperator 校验当前操作者具备平台级登录策略管理权限。
func (c *BaseLoginPolicyCase) ensurePlatformOperator(ctx context.Context) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	if authInfo.RoleCode != coreconst.BASE_ROLE_CODE_SUPER {
		return errorsx.PermissionDenied("只有平台管理员可以管理登录策略")
	}
	return nil
}

// validatePolicyTarget 校验策略作用域和目标记录。
func validatePolicyTarget(ctx context.Context, policy loginpolicy.Policy, tenantRepo *data.BaseTenantRepository, userRepo *data.BaseUserRepository) error {
	var err error
	switch policy.ScopeType {
	case loginpolicy.ScopeGlobal:
		if policy.TenantID != 0 || policy.UserID != 0 {
			return errorsx.InvalidArgument("全局策略不能设置租户或用户")
		}
	case loginpolicy.ScopeTenant:
		if policy.TenantID <= 0 || policy.UserID != 0 {
			return errorsx.InvalidArgument("租户策略目标无效")
		}
		_, err = tenantRepo.FindByID(ctx, policy.TenantID)
		if err != nil {
			return errorsx.ResourceNotFound("租户不存在").WithCause(err)
		}
	case loginpolicy.ScopeUser:
		if policy.UserID <= 0 || policy.TenantID <= 0 {
			return errorsx.InvalidArgument("用户策略目标无效")
		}
		var user *models.BaseUser
		user, err = userRepo.FindByID(ctx, policy.UserID)
		if err != nil {
			return errorsx.ResourceNotFound("用户不存在").WithCause(err)
		}
		if user.TenantID != policy.TenantID {
			return errorsx.Conflict("用户与租户不匹配")
		}
	default:
		return errorsx.InvalidArgument("登录策略作用域类型无效")
	}
	return nil
}

// policyToEntity 将策略领域记录转换为数据库模型。
func policyToEntity(policy loginpolicy.Policy) *models.BaseLoginPolicy {
	return &models.BaseLoginPolicy{ID: policy.ID, ScopeType: policy.ScopeType, TenantID: policy.TenantID, UserID: policy.UserID, Status: policy.Status, MaxFailedAttempts: policy.MaxFailedAttempts, LockDurationMinutes: policy.LockDurationMinutes, PasswordMinLength: policy.PasswordMinLength, PasswordHistoryCount: policy.PasswordHistoryCount, PasswordMinComplexityClasses: policy.PasswordMinComplexityClasses, PasswordMaxAgeDays: policy.PasswordMaxAgeDays, InitialPasswordHash: policy.InitialPasswordHash}
}

// ruleToEntity 将策略规则转换为数据库模型。
func ruleToEntity(rule loginpolicy.Rule) *models.BaseLoginPolicyRule {
	return &models.BaseLoginPolicyRule{ID: rule.ID, PolicyID: rule.PolicyID, RestrictionType: rule.RestrictionType, RestrictionMethod: rule.RestrictionMethod, RestrictionValue: rule.RestrictionValue, Reason: rule.Reason, Status: rule.Status}
}

// entityToPolicy 将数据库模型转换为策略领域记录。
func entityToPolicy(entity *models.BaseLoginPolicy, rules []*models.BaseLoginPolicyRule) loginpolicy.Policy {
	policy := loginpolicy.Policy{ID: entity.ID, ScopeType: entity.ScopeType, TenantID: entity.TenantID, UserID: entity.UserID, Status: entity.Status, MaxFailedAttempts: entity.MaxFailedAttempts, LockDurationMinutes: entity.LockDurationMinutes, PasswordMinLength: entity.PasswordMinLength, PasswordHistoryCount: entity.PasswordHistoryCount, PasswordMinComplexityClasses: entity.PasswordMinComplexityClasses, PasswordMaxAgeDays: entity.PasswordMaxAgeDays, InitialPasswordHash: entity.InitialPasswordHash, Rules: make([]loginpolicy.Rule, 0, len(rules))}
	if policy.PasswordMinLength == 0 {
		policy.PasswordMinLength = loginpolicy.DefaultPasswordMinLength
	}
	if policy.PasswordMinComplexityClasses == 0 {
		policy.PasswordMinComplexityClasses = loginpolicy.DefaultPasswordMinComplexityClasses
	}
	for _, entityRule := range rules {
		policy.Rules = append(policy.Rules, loginpolicy.Rule{ID: entityRule.ID, PolicyID: entityRule.PolicyID, RestrictionType: entityRule.RestrictionType, RestrictionMethod: entityRule.RestrictionMethod, RestrictionValue: entityRule.RestrictionValue, Reason: entityRule.Reason, Status: entityRule.Status})
	}
	return policy
}

// toBaseLoginPolicyRule 将策略规则转换为接口对象。
func toBaseLoginPolicyRule(rule loginpolicy.Rule) *adminv1.BaseLoginPolicyRule {
	return &adminv1.BaseLoginPolicyRule{Id: rule.ID, PolicyId: rule.PolicyID, RestrictionType: adminv1.BaseLoginPolicyRestrictionType(rule.RestrictionType), RestrictionMethod: adminv1.BaseLoginPolicyRestrictionMethod(rule.RestrictionMethod), RestrictionValue: rule.RestrictionValue, Reason: rule.Reason, Status: commonv1.Status(rule.Status)}
}
