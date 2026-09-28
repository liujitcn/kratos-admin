package kit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/server/middleware/ratelimit"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

const rateLimitPolicyCacheTTL = 30 * time.Second

const (
	minRateLimitRate          = 0.001
	maxRateLimitRate          = 1000000.0
	maxRateLimitBurst         = 1000000
	maxRateLimitLimit         = 1000000
	maxRateLimitWindowSeconds = 86400
	maxRateLimitLogLimit      = 10000
)

const (
	rateLimitAlgorithmTokenBucket          = string(ratelimit.AlgorithmTokenBucket)
	rateLimitAlgorithmFixedWindow          = string(ratelimit.AlgorithmFixedWindow)
	rateLimitAlgorithmSlidingWindowCounter = string(ratelimit.AlgorithmSlidingWindowCounter)
	rateLimitAlgorithmSlidingWindowLog     = string(ratelimit.AlgorithmSlidingWindowLog)
	rateLimitAlgorithmLeakyBucket          = string(ratelimit.AlgorithmLeakyBucket)
)

// RateLimitParams 描述所选限流算法对应的规则默认参数或策略参数快照。
type RateLimitParams struct {
	// Algorithm 是数据库中保存的算法编码。
	Algorithm string
	// TokensPerSecond 是令牌桶每秒生成的令牌数。
	TokensPerSecond float64
	// Burst 是令牌桶容量。
	Burst int
	// Limit 是单个时间窗口允许的请求数。
	Limit int
	// WindowSeconds 是滑动或固定窗口的秒数。
	WindowSeconds int
	// LeakRatePerSecond 是漏桶每秒漏出的水量。
	LeakRatePerSecond float64
	// Capacity 是漏桶容量。
	Capacity int
}

// RateLimitPolicyResolver 将数据库中的接口限流策略转换为 Core 运行时策略。
type RateLimitPolicyResolver struct {
	policyRepository *data.BaseAPIRateLimitPolicyRepository
	ruleRepository   *data.BaseRateLimitRuleRepository
	refreshMu        sync.Mutex
	mu               sync.RWMutex
	policies         map[string][]ratelimit.Policy
	loadedAt         time.Time
	refreshAttempted time.Time
	refreshErr       error
}

var _ ratelimit.PolicyResolver = (*RateLimitPolicyResolver)(nil)

// NewRateLimitPolicyResolver 创建接口限流策略解析器。
func NewRateLimitPolicyResolver(databases map[string]*gorm.Client) (*RateLimitPolicyResolver, error) {
	d, err := data.NewData(databases)
	if err != nil {
		return nil, err
	}
	return &RateLimitPolicyResolver{
		policyRepository: data.NewBaseAPIRateLimitPolicyRepository(d),
		ruleRepository:   data.NewBaseRateLimitRuleRepository(d),
		policies:         make(map[string][]ratelimit.Policy),
	}, nil
}

// Initialize 在服务处理请求前加载限流策略。
func (r *RateLimitPolicyResolver) Initialize(ctx context.Context) error {
	return r.Refresh(ctx)
}

// Refresh 从数据库刷新启用的接口限流策略。
func (r *RateLimitPolicyResolver) Refresh(ctx context.Context) error {
	r.refreshMu.Lock()
	defer r.refreshMu.Unlock()
	return r.refreshLocked(ctx)
}

// Resolve 按 RPC 操作名匹配策略，依策略 ID 顺序只返回首条启用策略。
func (r *RateLimitPolicyResolver) Resolve(ctx context.Context, operation string) ([]ratelimit.Policy, error) {
	err := r.refreshIfExpired(ctx)
	if err != nil {
		return nil, err
	}
	r.mu.RLock()
	policies := append([]ratelimit.Policy(nil), r.policies[operation]...)
	r.mu.RUnlock()
	if len(policies) > 1 {
		policies = policies[:1]
	}
	return policies, nil
}

// refreshIfExpired 在策略缓存过期时串行刷新并限制失败重试频率。
func (r *RateLimitPolicyResolver) refreshIfExpired(ctx context.Context) error {
	r.mu.RLock()
	loadedAt := r.loadedAt
	refreshAttempted := r.refreshAttempted
	refreshErr := r.refreshErr
	r.mu.RUnlock()
	if time.Since(loadedAt) < rateLimitPolicyCacheTTL {
		return nil
	}
	if !refreshAttempted.IsZero() && time.Since(refreshAttempted) < 5*time.Second {
		return refreshErr
	}
	r.refreshMu.Lock()
	defer r.refreshMu.Unlock()
	r.mu.RLock()
	loadedAt = r.loadedAt
	refreshAttempted = r.refreshAttempted
	refreshErr = r.refreshErr
	r.mu.RUnlock()
	if time.Since(loadedAt) < rateLimitPolicyCacheTTL {
		return nil
	}
	if !refreshAttempted.IsZero() && time.Since(refreshAttempted) < 5*time.Second {
		return refreshErr
	}
	return r.refreshLocked(ctx)
}

// refreshLocked 从数据库构建新策略快照，调用方必须持有刷新锁。
func (r *RateLimitPolicyResolver) refreshLocked(ctx context.Context) error {
	r.mu.Lock()
	r.refreshAttempted = time.Now()
	r.refreshErr = nil
	r.mu.Unlock()
	query := r.policyRepository.Query(ctx).BaseAPIRateLimitPolicy
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.Status.Eq(_const.STATUS_STATUS_ENABLE)))
	// 同一接口按策略 ID 升序匹配，首条启用策略优先。
	opts = append(opts, repository.Order(query.ID.Asc()))
	var rows []*models.BaseAPIRateLimitPolicy
	var err error
	rows, err = r.policyRepository.List(ctx, opts...)
	if err != nil {
		err = fmt.Errorf("查询启用的接口限流策略失败: %w", err)
		r.recordRefreshError(err)
		return err
	}
	ruleIDs := make([]int64, 0, len(rows))
	ruleIDSet := make(map[int64]struct{}, len(rows))
	for _, row := range rows {
		if _, exists := ruleIDSet[row.RuleID]; !exists {
			ruleIDs = append(ruleIDs, row.RuleID)
			ruleIDSet[row.RuleID] = struct{}{}
		}
	}
	var rules []*models.BaseRateLimitRule
	if len(ruleIDs) > 0 {
		rules, err = r.ruleRepository.ListByIDs(ctx, ruleIDs)
		if err != nil {
			err = fmt.Errorf("查询接口限流规则失败: %w", err)
			r.recordRefreshError(err)
			return err
		}
	}
	ruleByID := make(map[int64]*models.BaseRateLimitRule, len(rules))
	for _, rule := range rules {
		ruleByID[rule.ID] = rule
	}
	policies := make(map[string][]ratelimit.Policy)
	for _, row := range rows {
		rule := ruleByID[row.RuleID]
		if rule == nil || rule.Status != _const.STATUS_STATUS_ENABLE {
			log.Warn("接口限流策略引用的规则不存在或未启用，跳过该策略", "policy_id", row.ID, "rule_id", row.RuleID)
			continue
		}
		var params RateLimitParams
		params, err = ParseRateLimitParams(rule.RuleType, row.RuleParams)
		if err != nil {
			log.Warn("接口限流策略参数无效，跳过该策略", "policy_id", row.ID, "error", err)
			continue
		}
		if !validRateLimitDimension(row.Dimension) {
			log.Warn("接口限流策略维度无效，跳过该策略", "policy_id", row.ID)
			continue
		}
		var operations []string
		err = json.Unmarshal([]byte(row.Operations), &operations)
		if err != nil {
			log.Warn("接口限流策略接口列表无效，跳过该策略", "policy_id", row.ID, "error", err)
			continue
		}
		if len(operations) == 0 {
			log.Warn("接口限流策略未配置接口，跳过该策略", "policy_id", row.ID)
			continue
		}
		seenOperations := make(map[string]struct{}, len(operations))
		for _, operation := range operations {
			if operation == "" {
				log.Warn("接口限流策略包含空接口操作，跳过该操作", "policy_id", row.ID)
				continue
			}
			if _, exists := seenOperations[operation]; exists {
				log.Warn("接口限流策略包含重复接口操作，跳过该操作", "policy_id", row.ID, "operation", operation)
				continue
			}
			seenOperations[operation] = struct{}{}
			policies[operation] = append(policies[operation], ratelimit.Policy{
				Algorithm:         ratelimit.Algorithm(rule.RuleType),
				Dimension:         ratelimit.Dimension(row.Dimension),
				TokensPerSecond:   params.TokensPerSecond,
				Burst:             params.Burst,
				Limit:             params.Limit,
				Window:            time.Duration(params.WindowSeconds) * time.Second,
				LeakRatePerSecond: params.LeakRatePerSecond,
				Capacity:          params.Capacity,
			})
		}
	}
	r.mu.Lock()
	r.policies = policies
	r.loadedAt = time.Now()
	r.refreshErr = nil
	r.mu.Unlock()
	return nil
}

// recordRefreshError 保存策略加载失败状态，避免每个请求都重复访问数据库。
func (r *RateLimitPolicyResolver) recordRefreshError(err error) {
	r.mu.Lock()
	r.refreshErr = err
	r.mu.Unlock()
	log.Error("刷新接口限流策略失败", "error", err)
}

// ParseRateLimitParams 按限流算法严格校验并解析对应参数。
func ParseRateLimitParams(ruleType, raw string) (RateLimitParams, error) {
	params := RateLimitParams{Algorithm: ruleType}
	var err error
	switch ruleType {
	case rateLimitAlgorithmTokenBucket:
		var value struct {
			TokensPerSecond float64 `json:"tokens_per_second"`
			Burst           int     `json:"burst"`
		}
		err = decodeRateLimitParams(raw, &value)
		if err != nil {
			return RateLimitParams{}, err
		}
		if math.IsNaN(value.TokensPerSecond) || math.IsInf(value.TokensPerSecond, 0) || value.TokensPerSecond < minRateLimitRate || value.TokensPerSecond > maxRateLimitRate {
			return RateLimitParams{}, fmt.Errorf("tokens_per_second 必须在 %g 到 %g 之间", minRateLimitRate, maxRateLimitRate)
		}
		if value.Burst < 1 || value.Burst > maxRateLimitBurst {
			return RateLimitParams{}, fmt.Errorf("burst 必须在 1 到 %d 之间", maxRateLimitBurst)
		}
		params.TokensPerSecond = value.TokensPerSecond
		params.Burst = value.Burst
	case rateLimitAlgorithmFixedWindow, rateLimitAlgorithmSlidingWindowCounter, rateLimitAlgorithmSlidingWindowLog:
		var value struct {
			Limit         int `json:"limit"`
			WindowSeconds int `json:"window_seconds"`
		}
		err = decodeRateLimitParams(raw, &value)
		if err != nil {
			return RateLimitParams{}, err
		}
		if value.Limit < 1 || value.Limit > maxRateLimitLimit {
			return RateLimitParams{}, fmt.Errorf("limit 必须在 1 到 %d 之间", maxRateLimitLimit)
		}
		if ruleType == rateLimitAlgorithmSlidingWindowLog && value.Limit > maxRateLimitLogLimit {
			return RateLimitParams{}, fmt.Errorf("滑动窗口日志 limit 不能超过 %d", maxRateLimitLogLimit)
		}
		if value.WindowSeconds < 1 || value.WindowSeconds > maxRateLimitWindowSeconds {
			return RateLimitParams{}, fmt.Errorf("window_seconds 必须在 1 到 %d 之间", maxRateLimitWindowSeconds)
		}
		params.Limit = value.Limit
		params.WindowSeconds = value.WindowSeconds
	case rateLimitAlgorithmLeakyBucket:
		var value struct {
			LeakRatePerSecond float64 `json:"leak_rate_per_second"`
			Capacity          int     `json:"capacity"`
		}
		err = decodeRateLimitParams(raw, &value)
		if err != nil {
			return RateLimitParams{}, err
		}
		if math.IsNaN(value.LeakRatePerSecond) || math.IsInf(value.LeakRatePerSecond, 0) || value.LeakRatePerSecond < minRateLimitRate || value.LeakRatePerSecond > maxRateLimitRate {
			return RateLimitParams{}, fmt.Errorf("leak_rate_per_second 必须在 %g 到 %g 之间", minRateLimitRate, maxRateLimitRate)
		}
		if value.Capacity < 1 || value.Capacity > maxRateLimitBurst {
			return RateLimitParams{}, fmt.Errorf("capacity 必须在 1 到 %d 之间", maxRateLimitBurst)
		}
		params.LeakRatePerSecond = value.LeakRatePerSecond
		params.Capacity = value.Capacity
	default:
		return RateLimitParams{}, fmt.Errorf("不支持的限流算法类型 %q", ruleType)
	}
	return params, nil
}

func decodeRateLimitParams(raw string, target any) error {
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(target)
	if err != nil {
		return fmt.Errorf("限流规则参数不是有效JSON: %w", err)
	}
	var extra json.RawMessage
	err = decoder.Decode(&extra)
	if err != io.EOF {
		if err == nil {
			err = fmt.Errorf("限流规则参数包含多个JSON值")
		}
		return fmt.Errorf("限流规则参数不是有效JSON: %w", err)
	}
	return nil
}

// RateLimitAlgorithmFromProto 将限流算法枚举转换为数据库编码。
func RateLimitAlgorithmFromProto(algorithm adminv1.BaseRateLimitAlgorithm) (string, error) {
	switch algorithm {
	case adminv1.BaseRateLimitAlgorithm_BASE_RATE_LIMIT_ALGORITHM_TOKEN_BUCKET:
		return rateLimitAlgorithmTokenBucket, nil
	case adminv1.BaseRateLimitAlgorithm_BASE_RATE_LIMIT_ALGORITHM_FIXED_WINDOW:
		return rateLimitAlgorithmFixedWindow, nil
	case adminv1.BaseRateLimitAlgorithm_BASE_RATE_LIMIT_ALGORITHM_SLIDING_WINDOW_COUNTER:
		return rateLimitAlgorithmSlidingWindowCounter, nil
	case adminv1.BaseRateLimitAlgorithm_BASE_RATE_LIMIT_ALGORITHM_SLIDING_WINDOW_LOG:
		return rateLimitAlgorithmSlidingWindowLog, nil
	case adminv1.BaseRateLimitAlgorithm_BASE_RATE_LIMIT_ALGORITHM_LEAKY_BUCKET:
		return rateLimitAlgorithmLeakyBucket, nil
	default:
		return "", fmt.Errorf("限流算法未指定或无效")
	}
}

// RateLimitAlgorithmToProto 将数据库算法编码转换为限流算法枚举。
func RateLimitAlgorithmToProto(algorithm string) adminv1.BaseRateLimitAlgorithm {
	switch algorithm {
	case rateLimitAlgorithmTokenBucket:
		return adminv1.BaseRateLimitAlgorithm_BASE_RATE_LIMIT_ALGORITHM_TOKEN_BUCKET
	case rateLimitAlgorithmFixedWindow:
		return adminv1.BaseRateLimitAlgorithm_BASE_RATE_LIMIT_ALGORITHM_FIXED_WINDOW
	case rateLimitAlgorithmSlidingWindowCounter:
		return adminv1.BaseRateLimitAlgorithm_BASE_RATE_LIMIT_ALGORITHM_SLIDING_WINDOW_COUNTER
	case rateLimitAlgorithmSlidingWindowLog:
		return adminv1.BaseRateLimitAlgorithm_BASE_RATE_LIMIT_ALGORITHM_SLIDING_WINDOW_LOG
	case rateLimitAlgorithmLeakyBucket:
		return adminv1.BaseRateLimitAlgorithm_BASE_RATE_LIMIT_ALGORITHM_LEAKY_BUCKET
	default:
		return adminv1.BaseRateLimitAlgorithm_BASE_RATE_LIMIT_ALGORITHM_UNSPECIFIED
	}
}

// validRateLimitDimension 检查限流维度是否在公开协议范围内。
func validRateLimitDimension(dimension string) bool {
	switch dimension {
	case string(ratelimit.DimensionGlobal), string(ratelimit.DimensionIP), string(ratelimit.DimensionTenant), string(ratelimit.DimensionUser), string(ratelimit.DimensionOauthClient):
		return true
	default:
		return false
	}
}

// ValidateRateLimitPolicyForm 校验策略表单参数和维度。
func ValidateRateLimitPolicyForm(ruleType, dimension, raw string) error {
	_, err := ParseRateLimitParams(ruleType, raw)
	if err != nil {
		return err
	}
	if !validRateLimitDimension(dimension) {
		return fmt.Errorf("限流维度无效")
	}
	return nil
}

// RateLimitDimensionFromProto 将 Proto 限流维度转换为数据库编码。
func RateLimitDimensionFromProto(dimension adminv1.BaseApiRateLimitDimension) (string, error) {
	switch dimension {
	case adminv1.BaseApiRateLimitDimension_BASE_API_RATE_LIMIT_DIMENSION_GLOBAL:
		return string(ratelimit.DimensionGlobal), nil
	case adminv1.BaseApiRateLimitDimension_BASE_API_RATE_LIMIT_DIMENSION_IP:
		return string(ratelimit.DimensionIP), nil
	case adminv1.BaseApiRateLimitDimension_BASE_API_RATE_LIMIT_DIMENSION_TENANT:
		return string(ratelimit.DimensionTenant), nil
	case adminv1.BaseApiRateLimitDimension_BASE_API_RATE_LIMIT_DIMENSION_USER:
		return string(ratelimit.DimensionUser), nil
	case adminv1.BaseApiRateLimitDimension_BASE_API_RATE_LIMIT_DIMENSION_OAUTH_CLIENT:
		return string(ratelimit.DimensionOauthClient), nil
	default:
		return "", fmt.Errorf("限流维度未指定或无效")
	}
}

// RateLimitDimensionToProto 将数据库限流维度转换为 Proto 枚举。
func RateLimitDimensionToProto(dimension string) adminv1.BaseApiRateLimitDimension {
	switch dimension {
	case string(ratelimit.DimensionGlobal):
		return adminv1.BaseApiRateLimitDimension_BASE_API_RATE_LIMIT_DIMENSION_GLOBAL
	case string(ratelimit.DimensionIP):
		return adminv1.BaseApiRateLimitDimension_BASE_API_RATE_LIMIT_DIMENSION_IP
	case string(ratelimit.DimensionTenant):
		return adminv1.BaseApiRateLimitDimension_BASE_API_RATE_LIMIT_DIMENSION_TENANT
	case string(ratelimit.DimensionUser):
		return adminv1.BaseApiRateLimitDimension_BASE_API_RATE_LIMIT_DIMENSION_USER
	case string(ratelimit.DimensionOauthClient):
		return adminv1.BaseApiRateLimitDimension_BASE_API_RATE_LIMIT_DIMENSION_OAUTH_CLIENT
	default:
		return adminv1.BaseApiRateLimitDimension_BASE_API_RATE_LIMIT_DIMENSION_UNSPECIFIED
	}
}
