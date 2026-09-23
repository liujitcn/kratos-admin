package biz

import (
	"context"

	"github.com/liujitcn/go-utils/crypto"
	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/loginpolicy"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/password"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/utils"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	"google.golang.org/protobuf/types/known/emptypb"
)

// BaseLoginPolicyCase 提供平台管理员登录策略的逐条管理能力。
type BaseLoginPolicyCase struct {
	*biz.BaseCase
	tx                      data.Transaction
	baseLoginPolicyRepo     *data.BaseLoginPolicyRepository
	baseLoginPolicyRuleRepo *data.BaseLoginPolicyRuleRepository
	baseTenantRepo          *data.BaseTenantRepository
	baseUserRepo            *data.BaseUserRepository
	formMapper              *mapper.CopierMapper[adminv1.BaseLoginPolicyForm, models.BaseLoginPolicy]
	mapper                  *mapper.CopierMapper[adminv1.BaseLoginPolicy, models.BaseLoginPolicy]
	policyMapper            *mapper.CopierMapper[loginpolicy.Policy, models.BaseLoginPolicy]
	ruleMapper              *mapper.CopierMapper[adminv1.BaseLoginPolicyRule, models.BaseLoginPolicyRule]
	policyRuleMapper        *mapper.CopierMapper[loginpolicy.Rule, models.BaseLoginPolicyRule]
}

// NewBaseLoginPolicyCase 创建登录策略业务实例。
func NewBaseLoginPolicyCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	baseLoginPolicyRepo *data.BaseLoginPolicyRepository,
	baseLoginPolicyRuleRepo *data.BaseLoginPolicyRuleRepository,
	baseTenantRepo *data.BaseTenantRepository,
	baseUserRepo *data.BaseUserRepository,
) *BaseLoginPolicyCase {
	formMapper := mapper.NewCopierMapper[adminv1.BaseLoginPolicyForm, models.BaseLoginPolicy]()
	formMapper.AppendConverters(mapper.NewGenericTypeConverterPair(
		false,
		int32(0),
		func(value bool) int32 {
			if value {
				return 1
			}
			return 0
		},
		func(value int32) bool {
			return value == 1
		},
	))
	listMapper := mapper.NewCopierMapper[adminv1.BaseLoginPolicy, models.BaseLoginPolicy]()
	listMapper.AppendConverters(mapper.NewGenericTypeConverterPair(
		false,
		int32(0),
		func(value bool) int32 {
			if value {
				return 1
			}
			return 0
		},
		func(value int32) bool {
			return value == 1
		},
	))
	policyMapper := mapper.NewCopierMapper[loginpolicy.Policy, models.BaseLoginPolicy]()
	policyMapper.AppendConverters(mapper.NewGenericTypeConverterPair(
		false,
		int32(0),
		func(value bool) int32 {
			if value {
				return 1
			}
			return 0
		},
		func(value int32) bool {
			return value == 1
		},
	))
	return &BaseLoginPolicyCase{
		BaseCase:                baseCase,
		tx:                      tx,
		baseLoginPolicyRepo:     baseLoginPolicyRepo,
		baseLoginPolicyRuleRepo: baseLoginPolicyRuleRepo,
		baseTenantRepo:          baseTenantRepo,
		baseUserRepo:            baseUserRepo,
		formMapper:              formMapper,
		mapper:                  listMapper,
		policyMapper:            policyMapper,
		ruleMapper:              mapper.NewCopierMapper[adminv1.BaseLoginPolicyRule, models.BaseLoginPolicyRule](),
		policyRuleMapper:        mapper.NewCopierMapper[loginpolicy.Rule, models.BaseLoginPolicyRule](),
	}
}

// PageBaseLoginPolicy 分页查询登录策略及其限制规则。
func (c *BaseLoginPolicyCase) PageBaseLoginPolicy(ctx context.Context, req *adminv1.PageBaseLoginPolicyRequest) (*adminv1.PageBaseLoginPolicyResponse, error) {
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
func (c *BaseLoginPolicyCase) GetBaseLoginPolicy(ctx context.Context, req *adminv1.GetBaseLoginPolicyRequest) (*adminv1.BaseLoginPolicyForm, error) {
	entity, err := c.baseLoginPolicyRepo.FindByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	var rules []*models.BaseLoginPolicyRule
	rules, err = c.listRules(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return c.toBaseLoginPolicyForm(entity, rules)
}

// CreateBaseLoginPolicy 创建登录策略及其限制规则。
func (c *BaseLoginPolicyCase) CreateBaseLoginPolicy(ctx context.Context, req *adminv1.CreateBaseLoginPolicyRequest) (*emptypb.Empty, error) {
	input := req.GetBaseLoginPolicy()
	policy, err := c.policyFromForm(ctx, input)
	if err != nil {
		return nil, err
	}
	entity := c.policyMapper.ToEntity(&policy)
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		err = c.baseLoginPolicyRepo.Create(txCtx, entity)
		if err != nil {
			if errorsx.IsDuplicateKey(err) {
				return errorsx.UniqueConflict("同一作用域的登录策略已存在", "base_login_policy", "scope_type,tenant_id,user_id", "unique_base_login_policy").WithCause(err)
			}
			return err
		}
		for _, rule := range policy.Rules {
			ruleEntity := c.policyRuleMapper.ToEntity(&rule)
			ruleEntity.ID = 0
			ruleEntity.PolicyID = entity.ID
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
		return nil, err
	}
	if err = c.RefreshBaseLoginPolicy(ctx); err != nil {
		return nil, err
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseLoginPolicy 更新登录策略及其限制规则。
func (c *BaseLoginPolicyCase) UpdateBaseLoginPolicy(ctx context.Context, req *adminv1.UpdateBaseLoginPolicyRequest) (*emptypb.Empty, error) {
	input := req.GetBaseLoginPolicy()
	oldEntity, err := c.baseLoginPolicyRepo.FindByID(ctx, input.GetId())
	if err != nil {
		return nil, err
	}
	policy, err := c.policyFromForm(ctx, input)
	if err != nil {
		return nil, err
	}
	if input.GetInitialPassword() == nil {
		policy.InitialPasswordHash = oldEntity.InitialPasswordHash
	}
	// 未显式指定状态时保留原状态，避免静默重新启用已停用的策略。
	if input.GetStatus() == 0 {
		policy.Status = oldEntity.Status
	}
	entity := c.policyMapper.ToEntity(&policy)
	entity.ID = oldEntity.ID
	oldRules, err := c.listRules(ctx, oldEntity.ID)
	if err != nil {
		return nil, err
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
			ruleEntity := c.policyRuleMapper.ToEntity(&rule)
			ruleEntity.ID = 0
			ruleEntity.PolicyID = entity.ID
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
		return nil, err
	}
	if err = c.RefreshBaseLoginPolicy(ctx); err != nil {
		return nil, err
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseLoginPolicy 删除登录策略及其限制规则。
func (c *BaseLoginPolicyCase) DeleteBaseLoginPolicy(ctx context.Context, req *adminv1.DeleteBaseLoginPolicyRequest) (*emptypb.Empty, error) {
	ids := _string.ConvertStringToInt64Array(req.GetId())
	if len(ids) == 0 {
		return nil, errorsx.InvalidArgument("登录策略ID不能为空")
	}
	list, err := c.baseLoginPolicyRepo.ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	if len(list) != len(ids) {
		return nil, errorsx.ResourceNotFound("登录策略不存在")
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
		return nil, err
	}
	if err = c.RefreshBaseLoginPolicy(ctx); err != nil {
		return nil, err
	}
	return new(emptypb.Empty), nil
}

// SetBaseLoginPolicyStatus 设置登录策略状态。
func (c *BaseLoginPolicyCase) SetBaseLoginPolicyStatus(ctx context.Context, req *adminv1.SetBaseLoginPolicyStatusRequest) (*emptypb.Empty, error) {
	status := int32(req.GetStatus())
	if status != _const.STATUS_STATUS_ENABLE && status != _const.STATUS_STATUS_DISABLE {
		return nil, errorsx.InvalidArgument("登录策略状态无效")
	}
	entity, err := c.baseLoginPolicyRepo.FindByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if entity.Status == status {
		return new(emptypb.Empty), nil
	}
	entity.Status = status
	err = c.baseLoginPolicyRepo.UpdateByID(ctx, entity)
	if err != nil {
		return nil, err
	}
	if err = c.RefreshBaseLoginPolicy(ctx); err != nil {
		return nil, err
	}
	return new(emptypb.Empty), nil
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
		policy := c.policyMapper.ToDTO(entity)
		if policy.PasswordMinLength == 0 {
			policy.PasswordMinLength = loginpolicy.DefaultPasswordMinLength
		}
		if policy.PasswordMinComplexityClasses == 0 {
			policy.PasswordMinComplexityClasses = loginpolicy.DefaultPasswordMinComplexityClasses
		}
		policy.Rules = make([]loginpolicy.Rule, 0, len(rules))
		for _, ruleEntity := range rules {
			policy.Rules = append(policy.Rules, *c.policyRuleMapper.ToDTO(ruleEntity))
		}
		policySet.Policies = append(policySet.Policies, *policy)
	}
	if len(policySet.Policies) == 0 {
		policySet = loginpolicy.Load()
	}
	return loginpolicy.SaveToCache(c.Cache, policySet)
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
	policy := *c.policyMapper.ToDTO(c.formMapper.ToEntity(input))
	policy.Rules = make([]loginpolicy.Rule, 0, len(input.GetRules()))
	var err error
	if policy.Status == 0 {
		policy.Status = _const.STATUS_STATUS_ENABLE
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
	if input.MfaRememberDays != nil && (policy.MfaRememberDays < 0 || policy.MfaRememberDays > loginpolicy.MaxMfaRememberDays) {
		return loginpolicy.Policy{}, errorsx.InvalidArgument("MFA设备免验证天数必须在零到90之间")
	}
	if input.GetInitialPassword() != nil {
		var initialPassword string
		initialPassword, err = utils.DecryptPassword(c.Cache, input.GetInitialPassword(), basev1.PasswordCryptoScene_PASSWORD_CRYPTO_SCENE_CONFIGURE_PASSWORD_POLICY)
		if err != nil {
			return loginpolicy.Policy{}, err
		}
		config := loginpolicy.PasswordConfig{MinLength: policy.PasswordMinLength, MinComplexityClasses: policy.PasswordMinComplexityClasses}
		if err = password.ValidateComplexity(initialPassword, config); err != nil {
			return loginpolicy.Policy{}, errorsx.InvalidArgument("初始化密码长度或复杂度不符合安全策略").WithCause(err)
		}
		policy.InitialPasswordHash, err = crypto.Encrypt(initialPassword)
		if err != nil {
			return loginpolicy.Policy{}, errorsx.Internal("保存初始化密码失败").WithCause(err)
		}
	}
	for _, inputRule := range input.GetRules() {
		ruleEntity := c.ruleMapper.ToEntity(inputRule)
		rule := c.policyRuleMapper.ToDTO(ruleEntity)
		status := rule.Status
		if status == 0 {
			status = _const.STATUS_STATUS_ENABLE
		}
		rule.Status = status
		policy.Rules = append(policy.Rules, *rule)
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
	result := c.mapper.ToDTO(entity)
	if result.PasswordMinLength == 0 {
		result.PasswordMinLength = loginpolicy.DefaultPasswordMinLength
	}
	if result.PasswordMinComplexityClasses == 0 {
		result.PasswordMinComplexityClasses = loginpolicy.DefaultPasswordMinComplexityClasses
	}
	result.Rules = make([]*adminv1.BaseLoginPolicyRule, 0, len(rules))
	for _, ruleEntity := range rules {
		result.Rules = append(result.Rules, c.ruleMapper.ToDTO(ruleEntity))
	}
	if result.TenantId > 0 {
		var tenant *models.BaseTenant
		tenant, err = c.baseTenantRepo.FindByID(ctx, result.TenantId)
		if err != nil {
			return nil, err
		}
		result.TenantName = tenant.Name
	}
	if result.UserId > 0 {
		var user *models.BaseUser
		query := c.baseUserRepo.Query(ctx).BaseUser
		user, err = c.baseUserRepo.Find(ctx,
			repository.Select(query.ID, query.TenantID, query.UserName),
			repository.Where(query.ID.Eq(result.UserId)),
		)
		if err != nil {
			return nil, err
		}
		result.UserName = user.UserName
	}
	return result, nil
}

// toBaseLoginPolicyForm 将数据库记录转换为编辑表单。
func (c *BaseLoginPolicyCase) toBaseLoginPolicyForm(entity *models.BaseLoginPolicy, rules []*models.BaseLoginPolicyRule) (*adminv1.BaseLoginPolicyForm, error) {
	result := c.formMapper.ToDTO(entity)
	if result.PasswordMinLength == nil || result.GetPasswordMinLength() == 0 {
		result.PasswordMinLength = new(int32)
		*result.PasswordMinLength = loginpolicy.DefaultPasswordMinLength
	}
	if result.PasswordMinComplexityClasses == nil || result.GetPasswordMinComplexityClasses() == 0 {
		result.PasswordMinComplexityClasses = new(int32)
		*result.PasswordMinComplexityClasses = loginpolicy.DefaultPasswordMinComplexityClasses
	}
	result.Rules = make([]*adminv1.BaseLoginPolicyRule, 0, len(rules))
	for _, ruleEntity := range rules {
		result.Rules = append(result.Rules, c.ruleMapper.ToDTO(ruleEntity))
	}
	return result, nil
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
		query := userRepo.Query(ctx).BaseUser
		user, err = userRepo.Find(ctx,
			repository.Select(query.ID, query.TenantID),
			repository.Where(query.ID.Eq(policy.UserID)),
		)
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
