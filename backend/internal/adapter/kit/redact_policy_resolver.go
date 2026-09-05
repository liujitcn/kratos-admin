package kit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-kit/redact"
)

const policyCacheTTL = time.Minute

var (
	_ redact.PolicyResolver        = (*RedactPolicyResolver)(nil)
	_ redact.StoragePolicyResolver = (*RedactPolicyResolver)(nil)
)

// RedactPolicyResolver 将 Admin 入库和出库策略转换为运行时策略。
type RedactPolicyResolver struct {
	storagePolicyRepository *data.BaseRedactStoragePolicyRepository
	outputPolicyRepository  *data.BaseRedactOutputPolicyRepository
	ruleRepository          *data.BaseRedactRuleRepository
	mu                      sync.RWMutex
	outputPolicies          map[string]redact.FieldPolicy
	storagePolicies         map[string][]redact.StorageFieldPolicy
	loadedAt                time.Time
}

// NewRedactPolicyResolver 创建 Admin 脱敏策略解析器。
func NewRedactPolicyResolver(
	storagePolicyRepository *data.BaseRedactStoragePolicyRepository,
	outputPolicyRepository *data.BaseRedactOutputPolicyRepository,
	ruleRepository *data.BaseRedactRuleRepository,
) *RedactPolicyResolver {
	return &RedactPolicyResolver{
		storagePolicyRepository: storagePolicyRepository,
		outputPolicyRepository:  outputPolicyRepository,
		ruleRepository:          ruleRepository,
		outputPolicies:          make(map[string]redact.FieldPolicy),
		storagePolicies:         make(map[string][]redact.StorageFieldPolicy),
	}
}

// Refresh 从数据库刷新启用的入库和出库策略。
func (r *RedactPolicyResolver) Refresh(ctx context.Context) error {
	if r == nil || r.storagePolicyRepository == nil || r.outputPolicyRepository == nil || r.ruleRepository == nil {
		return fmt.Errorf("脱敏策略仓储未完整初始化")
	}
	rules, err := r.ruleRepository.List(ctx)
	if err != nil {
		return fmt.Errorf("查询脱敏规则模板失败: %w", err)
	}
	ruleByID := make(map[int64]*models.BaseRedactRule, len(rules))
	for _, rule := range rules {
		ruleByID[rule.ID] = rule
	}

	storageQuery := r.storagePolicyRepository.Query(ctx).BaseRedactStoragePolicy
	storageOpts := make([]repository.QueryOption, 0, 2)
	storageOpts = append(storageOpts, repository.Where(storageQuery.Status.Eq(_const.STATUS_STATUS_ENABLE)))
	storageOpts = append(storageOpts, repository.Order(storageQuery.ID.Asc()))
	var storageRows []*models.BaseRedactStoragePolicy
	storageRows, err = r.storagePolicyRepository.List(ctx, storageOpts...)
	if err != nil {
		return fmt.Errorf("查询入库脱敏策略失败: %w", err)
	}
	var storagePolicies map[string][]redact.StorageFieldPolicy
	storagePolicies, err = buildStoragePolicies(storageRows, ruleByID)
	if err != nil {
		return err
	}

	outputQuery := r.outputPolicyRepository.Query(ctx).BaseRedactOutputPolicy
	outputOpts := make([]repository.QueryOption, 0, 2)
	outputOpts = append(outputOpts, repository.Where(outputQuery.Status.Eq(_const.STATUS_STATUS_ENABLE)))
	outputOpts = append(outputOpts, repository.Order(outputQuery.ID.Asc()))
	var outputRows []*models.BaseRedactOutputPolicy
	outputRows, err = r.outputPolicyRepository.List(ctx, outputOpts...)
	if err != nil {
		return fmt.Errorf("查询出库脱敏策略失败: %w", err)
	}
	var outputPolicies map[string]redact.FieldPolicy
	outputPolicies, err = buildOutputPolicies(outputRows, ruleByID)
	if err != nil {
		return err
	}

	r.mu.Lock()
	r.outputPolicies = outputPolicies
	r.storagePolicies = storagePolicies
	r.loadedAt = time.Now()
	r.mu.Unlock()
	return nil
}

// Resolve 按接口和Proto字段解析出库策略。
func (r *RedactPolicyResolver) Resolve(ctx context.Context, fieldRef string) (redact.FieldPolicy, bool) {
	if r == nil || redact.DirectionFromContext(ctx) != redact.DirectionResponse {
		return redact.FieldPolicy{}, false
	}
	r.mu.RLock()
	loadedAt := r.loadedAt
	policy, ok := r.lookupOutputPolicy(ctx, fieldRef)
	r.mu.RUnlock()
	if time.Since(loadedAt) >= policyCacheTTL {
		err := r.Refresh(ctx)
		if err != nil {
			log.Error(fmt.Sprintf("刷新脱敏策略失败: %v", err))
			return policy, ok
		}
		r.mu.RLock()
		policy, ok = r.lookupOutputPolicy(ctx, fieldRef)
		r.mu.RUnlock()
	}
	return policy, ok
}

// ListStoragePolicies 按物理表返回入库脱敏策略。
func (r *RedactPolicyResolver) ListStoragePolicies(ctx context.Context, tableName string) []redact.StorageFieldPolicy {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	loadedAt := r.loadedAt
	policies := append([]redact.StorageFieldPolicy(nil), r.storagePolicies[tableName]...)
	r.mu.RUnlock()
	if time.Since(loadedAt) >= policyCacheTTL {
		err := r.Refresh(ctx)
		if err == nil {
			r.mu.RLock()
			policies = append([]redact.StorageFieldPolicy(nil), r.storagePolicies[tableName]...)
			r.mu.RUnlock()
		}
	}
	return policies
}

// lookupOutputPolicy 按精确接口和字段执行出库策略匹配。
func (r *RedactPolicyResolver) lookupOutputPolicy(ctx context.Context, fieldRef string) (redact.FieldPolicy, bool) {
	operation := redact.OperationFromContext(ctx)
	if operation == "" {
		return redact.FieldPolicy{}, false
	}
	if authenticationResponseOperation(operation) {
		return redact.FieldPolicy{Mode: redact.PolicyModeFull}, true
	}
	policy, ok := r.outputPolicies[outputPolicyKey(operation, fieldRef)]
	if ok {
		return policy, true
	}
	return redact.FieldPolicy{}, false
}

// buildStoragePolicies 构建按物理表分组的入库策略。
func buildStoragePolicies(rows []*models.BaseRedactStoragePolicy, rules map[int64]*models.BaseRedactRule) (map[string][]redact.StorageFieldPolicy, error) {
	result := make(map[string][]redact.StorageFieldPolicy)
	var err error
	for _, row := range rows {
		if row.TableName_ == "" || row.ColumnName == "" {
			return nil, fmt.Errorf("入库脱敏策略 %d 缺少数据库字段映射", row.ID)
		}
		rule, ok := rules[row.RuleID]
		if !ok || rule.Status != _const.STATUS_STATUS_ENABLE {
			return nil, fmt.Errorf("入库脱敏策略 %d 引用的规则不存在或未启用", row.ID)
		}
		params := effectiveRuleParams(row.RuleParams, rule.DefaultParams)
		var fieldPolicy redact.FieldPolicy
		fieldPolicy, err = redact.NewFieldPolicy(redact.PolicyModeApplyRule, rule.RuleType, params)
		if err != nil {
			return nil, fmt.Errorf("解析入库脱敏策略 %d 失败: %w", row.ID, err)
		}
		fieldPolicy.RuleID = rule.ID
		fieldPolicy.Fingerprint = redact.RuleFingerprint(rule.RuleType, params)
		result[row.TableName_] = append(result[row.TableName_], redact.StorageFieldPolicy{
			ID:         row.ID,
			TableName:  row.TableName_,
			ColumnName: row.ColumnName,
			Rule:       fieldPolicy,
		})
	}
	return result, nil
}

// buildOutputPolicies 构建精确接口字段出库策略。
func buildOutputPolicies(rows []*models.BaseRedactOutputPolicy, rules map[int64]*models.BaseRedactRule) (map[string]redact.FieldPolicy, error) {
	result := make(map[string]redact.FieldPolicy, len(rows))
	var err error
	for _, row := range rows {
		if row.Operation == "" || row.MessageRef == "" || row.FieldPath == "" {
			return nil, fmt.Errorf("出库脱敏策略 %d 缺少接口或Proto字段", row.ID)
		}
		mode := redact.PolicyMode(row.Mode)
		if mode != redact.PolicyModeApplyRule && mode != redact.PolicyModeHide && mode != redact.PolicyModeFull {
			return nil, fmt.Errorf("出库脱敏策略 %d 模式无效: %d", row.ID, row.Mode)
		}
		policy := redact.FieldPolicy{Mode: mode}
		if mode == redact.PolicyModeApplyRule {
			rule, ok := rules[row.RuleID]
			if !ok || rule.Status != _const.STATUS_STATUS_ENABLE {
				return nil, fmt.Errorf("出库脱敏策略 %d 引用的规则不存在或未启用", row.ID)
			}
			params := effectiveRuleParams(row.RuleParams, rule.DefaultParams)
			policy, err = redact.NewFieldPolicy(mode, rule.RuleType, params)
			if err != nil {
				return nil, fmt.Errorf("解析出库脱敏策略 %d 失败: %w", row.ID, err)
			}
			policy.RuleID = rule.ID
			policy.Fingerprint = redact.RuleFingerprint(rule.RuleType, params)
		}
		result[outputPolicyKey(row.Operation, row.MessageRef+"."+row.FieldPath)] = policy
	}
	return result, nil
}

// effectiveRuleParams 返回策略实际使用的完整规则参数。
func effectiveRuleParams(policyParams, defaultParams string) string {
	if policyParams != "" && policyParams != "{}" {
		return policyParams
	}
	return defaultParams
}

// authenticationResponseOperation 判断必须保留认证响应原值的协议方法。
func authenticationResponseOperation(operation string) bool {
	switch operation {
	case "/base.v1.LoginService/VerifyCaptcha",
		"/base.v1.LoginService/Login",
		"/base.v1.LoginService/RefreshToken",
		"/base.v1.MfaService/VerifyMfa",
		"/base.v1.OauthService/CreateOauthSession",
		"/base.v1.OauthService/BindOauthSession",
		"/base.v1.OauthService/ExchangeOauthTicket",
		"/base.v1.OauthClientService/IssueOauthClientToken":
		return true
	default:
		return false
	}
}

// outputPolicyKey 生成接口和Proto字段组成的出库策略键。
func outputPolicyKey(operation, fieldRef string) string {
	return operation + "\x00" + fieldRef
}
