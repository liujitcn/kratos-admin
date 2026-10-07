package kit

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-admin/backend/internal/config"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	_const "github.com/liujitcn/kratos-core/const"
	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"
	"github.com/liujitcn/kratos-kit/redact"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const policyCacheTTL = time.Minute

// defaultTenantID 表示默认租户编号（种子固定），默认租户的脱敏策略即全局策略。
const defaultTenantID = int64(1)

var (
	_ redact.PolicyResolver        = (*RedactPolicyResolver)(nil)
	_ redact.StoragePolicyResolver = (*RedactPolicyResolver)(nil)
)

// RedactPolicyResolver 将 Admin 入库和出库策略转换为运行时策略。
type RedactPolicyResolver struct {
	defaultDB               *gorm.DB
	store                   *StorageValueStore
	initializeMu            sync.Mutex
	refreshMu               sync.Mutex
	runtime                 *storageRuntime
	fieldCipher             *fieldCipher
	storagePolicyRepository *data.BaseRedactStoragePolicyRepository
	outputPolicyRepository  *data.BaseRedactOutputPolicyRepository
	ruleRepository          *data.BaseRedactRuleRepository
	mu                      sync.RWMutex
	outputPolicies          map[string]redact.FieldPolicy
	storagePolicies         map[string][]redact.StorageFieldPolicy
	loadedAt                time.Time
	refreshAttemptedAt      time.Time
}

// NewRedactPolicyResolver 构造默认数据库仓储和空缓存，迁移完成后须调用 Initialize。
func NewRedactPolicyResolver(databases map[string]*kitgorm.Client) (*RedactPolicyResolver, error) {
	d, err := data.NewData(databases)
	if err != nil {
		return nil, err
	}
	var store *StorageValueStore
	store, err = NewStorageValueStore(databases)
	if err != nil {
		return nil, err
	}
	return &RedactPolicyResolver{
		defaultDB:               databases[kitgorm.DefaultClientName].DB,
		store:                   store,
		storagePolicyRepository: data.NewBaseRedactStoragePolicyRepository(d),
		outputPolicyRepository:  data.NewBaseRedactOutputPolicyRepository(d),
		ruleRepository:          data.NewBaseRedactRuleRepository(d),
		outputPolicies:          make(map[string]redact.FieldPolicy),
		storagePolicies:         make(map[string][]redact.StorageFieldPolicy),
	}, nil
}

// Initialize 在迁移和密钥初始化完成后、数据库开始处理业务请求前加载策略并绑定回调。
// 同一实例成功后重复调用无副作用；失败可重试，同一数据库的初始化必须串行执行。
func (r *RedactPolicyResolver) Initialize(ctx context.Context) error {
	r.initializeMu.Lock()
	defer r.initializeMu.Unlock()
	if r.runtime != nil {
		return nil
	}
	err := r.Refresh(ctx)
	if err != nil {
		return err
	}
	var protector *redact.StorageProtector
	protector, err = config.NewRedactStorageProtector()
	if err != nil {
		return fmt.Errorf("创建脱敏存储保护器失败: %w", err)
	}
	var directCipher *fieldCipher
	directCipher, err = newRuntimeFieldCipher()
	if err != nil {
		return fmt.Errorf("创建直接字段加密器失败: %w", err)
	}
	r.fieldCipher = directCipher
	runtime := newStorageRuntime(r.store, r, protector, directCipher)
	err = runtime.registerCallbacks(r.defaultDB)
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.runtime = runtime
	r.mu.Unlock()
	return nil
}

// Refresh 从数据库刷新启用的入库和出库策略。
func (r *RedactPolicyResolver) Refresh(ctx context.Context) error {
	if r == nil {
		return fmt.Errorf("脱敏策略解析器未初始化")
	}
	r.refreshMu.Lock()
	defer r.refreshMu.Unlock()
	return r.refreshLocked(ctx)
}

// refreshLocked 从数据库刷新启用的入库和出库策略，调用方必须持有刷新锁。
func (r *RedactPolicyResolver) refreshLocked(ctx context.Context) error {
	if r == nil || r.storagePolicyRepository == nil || r.outputPolicyRepository == nil || r.ruleRepository == nil {
		return fmt.Errorf("脱敏策略仓储未完整初始化")
	}
	r.refreshAttemptedAt = time.Now()
	rules, err := r.ruleRepository.List(ctx)
	if err != nil {
		return fmt.Errorf("查询脱敏规则失败: %w", err)
	}
	ruleByID := make(map[int64]*models.BaseRedactRule, len(rules))
	for _, rule := range rules {
		err = ValidateRedactRule(rule.Code, rule.RuleType, rule.DefaultParams)
		if err != nil {
			return fmt.Errorf("解析脱敏规则 %d 失败: %w", rule.ID, err)
		}
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

// refreshIfExpired 在缓存过期时串行刷新，避免并发请求重复查询数据库。
func (r *RedactPolicyResolver) refreshIfExpired(ctx context.Context) error {
	r.refreshMu.Lock()
	defer r.refreshMu.Unlock()
	r.mu.RLock()
	loadedAt := r.loadedAt
	r.mu.RUnlock()
	if time.Since(loadedAt) < policyCacheTTL {
		return nil
	}
	if !r.refreshAttemptedAt.IsZero() && time.Since(r.refreshAttemptedAt) < policyCacheTTL {
		return nil
	}
	return r.refreshLocked(ctx)
}

// Resolve 按接口和 Proto 字段解析出库策略，仅在初始化成功后自动刷新缓存。
func (r *RedactPolicyResolver) Resolve(ctx context.Context, fieldRef string) (redact.FieldPolicy, bool) {
	if r == nil || redact.DirectionFromContext(ctx) != redact.DirectionResponse {
		return redact.FieldPolicy{}, false
	}
	r.mu.RLock()
	loadedAt := r.loadedAt
	initialized := r.runtime != nil
	policy, ok := r.lookupOutputPolicy(ctx, fieldRef)
	r.mu.RUnlock()
	if initialized && time.Since(loadedAt) >= policyCacheTTL {
		err := r.refreshIfExpired(ctx)
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

// ListStoragePolicies 返回默认数据源的入库策略：默认租户策略即全局策略，
// 其他租户的策略按字段覆盖默认租户策略，未携带租户或租户未配置时应用默认租户策略。
// 仅在初始化成功后自动刷新缓存。
func (r *RedactPolicyResolver) ListStoragePolicies(ctx context.Context, tenantID int64, tableName string) []redact.StorageFieldPolicy {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	loadedAt := r.loadedAt
	initialized := r.runtime != nil
	policies := r.mergedStoragePolicies(tenantID, tableName)
	r.mu.RUnlock()
	if initialized && time.Since(loadedAt) >= policyCacheTTL {
		err := r.refreshIfExpired(ctx)
		if err == nil {
			r.mu.RLock()
			policies = r.mergedStoragePolicies(tenantID, tableName)
			r.mu.RUnlock()
		}
	}
	return append([]redact.StorageFieldPolicy(nil), policies...)
}

// mergedStoragePolicies 合并默认租户和其他租户的入库策略，调用方必须持有读锁。
func (r *RedactPolicyResolver) mergedStoragePolicies(tenantID int64, tableName string) []redact.StorageFieldPolicy {
	global := r.storagePolicies[storagePolicyKey(defaultTenantID, kitgorm.DefaultClientName, tableName)]
	if tenantID <= 0 || tenantID == defaultTenantID {
		return global
	}
	tenant := r.storagePolicies[storagePolicyKey(tenantID, kitgorm.DefaultClientName, tableName)]
	if len(tenant) == 0 {
		return global
	}
	if len(global) == 0 {
		return tenant
	}
	return mergeStoragePolicies(global, tenant)
}

// HasStoragePolicies 判断物理表是否配置了任一租户存储脱敏策略。
func (r *RedactPolicyResolver) HasStoragePolicies(tableName string) bool {
	if r == nil || tableName == "" {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	suffix := "\x00" + tableName
	for key, policies := range r.storagePolicies {
		if len(policies) > 0 && strings.HasSuffix(key, suffix) {
			return true
		}
	}
	return false
}

// ListStoragePoliciesByTable 返回物理表在所有租户下的存储脱敏策略。
func (r *RedactPolicyResolver) ListStoragePoliciesByTable(tableName string) []redact.StorageFieldPolicy {
	if r == nil || tableName == "" {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	suffix := "\x00" + tableName
	policies := make([]redact.StorageFieldPolicy, 0)
	for key, items := range r.storagePolicies {
		if strings.HasSuffix(key, suffix) {
			policies = append(policies, items...)
		}
	}
	return policies
}

// lookupOutputPolicy 按精确接口和字段执行出库策略匹配：
// 优先使用响应数据所属租户的策略，未命中或未携带租户时回退默认租户的全局策略。
func (r *RedactPolicyResolver) lookupOutputPolicy(ctx context.Context, fieldRef string) (redact.FieldPolicy, bool) {
	operation := redact.OperationFromContext(ctx)
	if operation == "" {
		return redact.FieldPolicy{}, false
	}
	if authenticationResponseOperation(operation) {
		return redact.FieldPolicy{Mode: redact.PolicyModeFull}, true
	}
	tenantID := redact.TenantIDFromContext(ctx)
	if tenantID > 0 {
		policy, ok := r.outputPolicies[outputPolicyKey(tenantID, operation, fieldRef)]
		if ok {
			return policy, true
		}
	}
	policy, ok := r.outputPolicies[outputPolicyKey(defaultTenantID, operation, fieldRef)]
	if ok {
		return policy, true
	}
	return redact.FieldPolicy{}, false
}

// buildStoragePolicies 构建按物理表分组的入库策略，默认租户策略即全局策略。
func buildStoragePolicies(rows []*models.BaseRedactStoragePolicy, rules map[int64]*models.BaseRedactRule) (map[string][]redact.StorageFieldPolicy, error) {
	result := make(map[string][]redact.StorageFieldPolicy)
	var err error
	for _, row := range rows {
		if row.SourceName == "" || row.TableName_ == "" || row.ColumnName == "" {
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
		if row.TenantID <= 0 {
			return nil, fmt.Errorf("入库脱敏策略 %d 缺少租户", row.ID)
		}
		key := storagePolicyKey(row.TenantID, row.SourceName, row.TableName_)
		result[key] = append(result[key], redact.StorageFieldPolicy{
			ID:         row.ID,
			TenantID:   row.TenantID,
			TableName:  row.TableName_,
			ColumnName: row.ColumnName,
			Rule:       fieldPolicy,
		})
	}
	return result, nil
}

// buildOutputPolicies 构建精确接口字段出库策略，默认租户策略即全局策略。
func buildOutputPolicies(rows []*models.BaseRedactOutputPolicy, rules map[int64]*models.BaseRedactRule) (map[string]redact.FieldPolicy, error) {
	result := make(map[string]redact.FieldPolicy, len(rows))
	var err error
	for _, row := range rows {
		if row.TenantID <= 0 {
			return nil, fmt.Errorf("响应脱敏策略 %d 缺少租户", row.ID)
		}
		if row.ServiceName == "" || row.Operation == "" || row.MessageRef == "" || row.FieldPath == "" {
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
		result[outputPolicyKey(row.TenantID, row.Operation, row.MessageRef+"."+row.FieldPath)] = policy
	}
	return result, nil
}

// ErrUnprotectedStorageField 表示字段未配置直接加密入库策略，不能通过受控接口查看明文。
var ErrUnprotectedStorageField = errors.New("敏感字段未配置直接加密入库策略")

// RevealStorageField 读取默认数据源受保护表的直接加密字段明文，仅供受控查询接口使用。
// 未配置直接加密策略、旁表脱敏字段和跨数据源字段一律拒绝。
func (r *RedactPolicyResolver) RevealStorageField(ctx context.Context, tableName, columnName string, recordID int64) (string, error) {
	if r == nil || r.defaultDB == nil || r.fieldCipher == nil {
		return "", fmt.Errorf("脱敏策略解析器未初始化")
	}
	algorithm, ok := r.directEncryptionAlgorithm(kitgorm.DefaultClientName, tableName, columnName)
	if !ok {
		return "", ErrUnprotectedStorageField
	}
	var text string
	db := r.defaultDB.WithContext(ctx)
	err := db.Table(tableName).
		Select(db.Statement.Quote(clause.Column{Name: columnName})).
		Where("id = ?", recordID).Row().Scan(&text)
	if err != nil {
		return "", fmt.Errorf("读取敏感字段失败: %w", err)
	}
	if text == "" {
		return text, nil
	}
	plaintext, err := r.fieldCipher.Decrypt(algorithm, text)
	if err != nil {
		// 存量旧格式或明文按原值返回，等待下次保存时经策略转为策略密文。
		return text, nil
	}
	return plaintext, nil
}

// directEncryptionAlgorithm 判断默认数据源物理表字段是否配置了直接加密策略，返回加密算法。
func (r *RedactPolicyResolver) directEncryptionAlgorithm(sourceName, tableName, columnName string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for key, policies := range r.storagePolicies {
		parts := strings.SplitN(key, "\x00", 3)
		if len(parts) != 3 || parts[1] != sourceName || parts[2] != tableName {
			continue
		}
		for _, policy := range policies {
			if strings.EqualFold(policy.ColumnName, columnName) && isDirectEncryptionPolicy(policy) {
				return policy.Rule.EncryptAlgorithm, true
			}
		}
	}
	return "", false
}

// storagePolicyKey 返回数据源和物理表组成的策略键。
func storagePolicyKey(tenantID int64, sourceName, tableName string) string {
	return fmt.Sprintf("%d\x00%s\x00%s", tenantID, sourceName, tableName)
}

// effectiveRuleParams 返回策略实际使用的完整规则参数。
func effectiveRuleParams(policyParams, ruleDefaultParams string) string {
	if policyParams != "" && policyParams != "{}" {
		return policyParams
	}
	return ruleDefaultParams
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
func outputPolicyKey(tenantID int64, operation, fieldRef string) string {
	return fmt.Sprintf("%d\x00%s\x00%s", tenantID, operation, fieldRef)
}

// mergeStoragePolicies 合并全局和租户入库策略，租户策略按字段覆盖全局策略。
func mergeStoragePolicies(global, tenant []redact.StorageFieldPolicy) []redact.StorageFieldPolicy {
	overridden := make(map[string]struct{}, len(tenant))
	for _, policy := range tenant {
		overridden[strings.ToLower(policy.ColumnName)] = struct{}{}
	}
	merged := make([]redact.StorageFieldPolicy, 0, len(global)+len(tenant))
	for _, policy := range global {
		if _, ok := overridden[strings.ToLower(policy.ColumnName)]; ok {
			continue
		}
		merged = append(merged, policy)
	}
	return append(merged, tenant...)
}
