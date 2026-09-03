package sessionstate

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/cache"
	kitconfig "github.com/liujitcn/kratos-kit/config"
	"github.com/redis/go-redis/v9"
)

const (
	defaultIdleTimeout = 30 * time.Minute
	defaultMaxLifetime = 12 * time.Hour
	stateTTLGrace      = 24 * time.Hour
)

// ErrStateNotFound 表示当前用户还没有服务端会话状态。
var ErrStateNotFound = errors.New("会话状态不存在")

// ErrIdleExpired 表示会话超过空闲时间限制。
var ErrIdleExpired = errors.New("会话空闲超时")

// ErrMaxLifetimeExpired 表示会话超过绝对生命周期。
var ErrMaxLifetimeExpired = errors.New("会话超过最大生命周期")

// Policy 定义服务端会话超时策略。
type Policy struct {
	IdleTimeout time.Duration
	MaxLifetime time.Duration
}

// State 保存用户级服务端会话事实。
type State struct {
	CreatedAt     time.Time `json:"created_at"`
	LastActiveAt  time.Time `json:"last_active_at"`
	TokenIssuedAt time.Time `json:"token_issued_at"`
	ClientIP      string    `json:"client_ip"`
	Device        string    `json:"device"`
}

// PolicyFromConfig 从引导配置读取会话超时策略。
func PolicyFromConfig() Policy {
	bootstrapConfig := kitconfig.GetBootstrapConfig()
	if bootstrapConfig == nil || bootstrapConfig.GetAuthn() == nil {
		return policyFromSessionConfig(nil)
	}
	return policyFromSessionConfig(bootstrapConfig.GetAuthn().GetSession())
}

func policyFromSessionConfig(sessionConfig *configv1.Authentication_Session) Policy {
	policy := Policy{IdleTimeout: defaultIdleTimeout, MaxLifetime: defaultMaxLifetime}
	if sessionConfig == nil {
		return policy
	}
	if idleTimeout := sessionConfig.GetIdleTimeout(); idleTimeout != nil && idleTimeout.AsDuration() > 0 {
		policy.IdleTimeout = idleTimeout.AsDuration()
	}
	if maxLifetime := sessionConfig.GetMaxLifetime(); maxLifetime != nil && maxLifetime.AsDuration() > 0 {
		policy.MaxLifetime = maxLifetime.AsDuration()
	}
	return policy
}

// Start 创建或覆盖用户的服务端会话状态。
func Start(store cache.Cache, userID int64, clientIP, device string, now time.Time) (State, error) {
	state := State{CreatedAt: now, LastActiveAt: now, TokenIssuedAt: now, ClientIP: clientIP, Device: device}
	return state, save(store, userID, state, PolicyFromConfig())
}

// Read 读取用户的服务端会话状态。
func Read(store cache.Cache, userID int64) (State, error) {
	if store == nil || userID <= 0 {
		return State{}, ErrStateNotFound
	}
	raw, err := store.Get(key(userID))
	if err != nil {
		if isCacheMiss(err) {
			return State{}, ErrStateNotFound
		}
		return State{}, fmt.Errorf("读取会话状态失败: %w", err)
	}
	state := State{}
	if err = json.Unmarshal([]byte(raw), &state); err != nil {
		return State{}, fmt.Errorf("解析会话状态失败: %w", err)
	}
	return state, nil
}

// Validate 校验用户会话是否仍在空闲和绝对生命周期内。
func Validate(store cache.Cache, userID int64, now time.Time) (State, error) {
	state, err := Read(store, userID)
	if err != nil {
		return State{}, err
	}
	if err = Evaluate(state, now, PolicyFromConfig()); err != nil {
		return State{}, err
	}
	return state, nil
}

// Touch 校验并更新用户会话的最后活动时间。
func Touch(store cache.Cache, userID int64, now time.Time) (State, error) {
	state, err := Validate(store, userID, now)
	if err != nil {
		return State{}, err
	}
	state.LastActiveAt = now
	return state, save(store, userID, state, PolicyFromConfig())
}

// MarkTokenIssued 更新最近一次访问令牌签发时间，不延长会话活动时间。
func MarkTokenIssued(store cache.Cache, userID int64, now time.Time) error {
	state, err := Validate(store, userID, now)
	if err != nil {
		return err
	}
	state.TokenIssuedAt = now
	return save(store, userID, state, PolicyFromConfig())
}

// Clear 删除用户服务端会话状态。
func Clear(store cache.Cache, userID int64) error {
	if store == nil || userID <= 0 {
		return nil
	}
	return store.Del(key(userID))
}

// Evaluate 仅根据给定状态和策略判断会话是否过期。
func Evaluate(state State, now time.Time, policy Policy) error {
	if state.CreatedAt.IsZero() || state.LastActiveAt.IsZero() {
		return ErrStateNotFound
	}
	if policy.MaxLifetime > 0 && !now.Before(state.CreatedAt.Add(policy.MaxLifetime)) {
		return ErrMaxLifetimeExpired
	}
	if policy.IdleTimeout > 0 && !now.Before(state.LastActiveAt.Add(policy.IdleTimeout)) {
		return ErrIdleExpired
	}
	return nil
}

// save 持久化服务端会话状态。
func save(store cache.Cache, userID int64, state State, policy Policy) error {
	if store == nil || userID <= 0 {
		return errors.New("会话缓存未配置")
	}
	payload, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("序列化会话状态失败: %w", err)
	}
	ttl := policy.MaxLifetime + stateTTLGrace
	if ttl <= stateTTLGrace {
		ttl = defaultMaxLifetime + stateTTLGrace
	}
	return store.Set(key(userID), string(payload), ttl)
}

// key 返回用户级会话状态缓存键。
func key(userID int64) string {
	return fmt.Sprintf("admin_session_state:%d", userID)
}

// isCacheMiss 判断缓存键不存在，而不是缓存服务故障。
func isCacheMiss(err error) bool {
	if errors.Is(err, redis.Nil) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "key not found") || strings.Contains(message, "key expired") || strings.Contains(message, "not found")
}
