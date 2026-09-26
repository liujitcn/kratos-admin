package kit

import (
	"context"
	"encoding/json"
	"fmt"
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
	minRateLimitRate  = 0.001
	maxRateLimitRate  = 1000000.0
	maxRateLimitBurst = 1000000
)

// RateLimitParams 描述令牌桶规则的默认参数和接口策略参数。
type RateLimitParams struct {
	// TokensPerSecond 是令牌生成速率。
	TokensPerSecond float64 `json:"tokens_per_second"`
	// Burst 是令牌桶容量。
	Burst int `json:"burst"`
}

// RateLimitPolicyResolver 将数据库中的接口限流策略转换为 Core 运行时策略。
type RateLimitPolicyResolver struct {
	policyRepository *data.BaseAPIRateLimitPolicyRepository
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
	policies := make(map[string][]ratelimit.Policy)
	for _, row := range rows {
		var params RateLimitParams
		params, err = ParseRateLimitParams("TOKEN_BUCKET", row.RuleParams)
		if err != nil {
			err = fmt.Errorf("接口限流策略 %d 参数无效: %w", row.ID, err)
			r.recordRefreshError(err)
			return err
		}
		if !validRateLimitDimension(row.Dimension) {
			err = fmt.Errorf("接口限流策略 %d 维度无效", row.ID)
			r.recordRefreshError(err)
			return err
		}
		var operations []string
		err = json.Unmarshal([]byte(row.Operations), &operations)
		if err != nil {
			err = fmt.Errorf("接口限流策略 %d 接口列表无效: %w", row.ID, err)
			r.recordRefreshError(err)
			return err
		}
		if len(operations) == 0 {
			err = fmt.Errorf("接口限流策略 %d 未配置接口", row.ID)
			r.recordRefreshError(err)
			return err
		}
		seenOperations := make(map[string]struct{}, len(operations))
		for _, operation := range operations {
			if operation == "" {
				err = fmt.Errorf("接口限流策略 %d 包含空接口操作", row.ID)
				r.recordRefreshError(err)
				return err
			}
			if _, exists := seenOperations[operation]; exists {
				err = fmt.Errorf("接口限流策略 %d 包含重复接口操作", row.ID)
				r.recordRefreshError(err)
				return err
			}
			seenOperations[operation] = struct{}{}
			policies[operation] = append(policies[operation], ratelimit.Policy{
				Dimension:       ratelimit.Dimension(row.Dimension),
				TokensPerSecond: params.TokensPerSecond,
				Burst:           params.Burst,
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

// ParseRateLimitParams 校验并解析令牌桶参数。
func ParseRateLimitParams(ruleType, raw string) (RateLimitParams, error) {
	if ruleType != "TOKEN_BUCKET" {
		return RateLimitParams{}, fmt.Errorf("不支持的限流算法类型 %q", ruleType)
	}
	params := RateLimitParams{}
	err := json.Unmarshal([]byte(raw), &params)
	if err != nil {
		return RateLimitParams{}, fmt.Errorf("限流规则参数不是有效JSON: %w", err)
	}
	if math.IsNaN(params.TokensPerSecond) || math.IsInf(params.TokensPerSecond, 0) || params.TokensPerSecond < minRateLimitRate || params.TokensPerSecond > maxRateLimitRate {
		return RateLimitParams{}, fmt.Errorf("tokens_per_second 必须在 %g 到 %g 之间", minRateLimitRate, maxRateLimitRate)
	}
	if params.Burst < 1 || params.Burst > maxRateLimitBurst {
		return RateLimitParams{}, fmt.Errorf("burst 必须在 1 到 %d 之间", maxRateLimitBurst)
	}
	return params, nil
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
