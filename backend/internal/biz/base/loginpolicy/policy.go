package loginpolicy

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/liujitcn/kratos-kit/cache"
	"github.com/redis/go-redis/v9"
)

// CacheKey 是登录策略缓存键。
const CacheKey = "security:login-policy"

const (
	// DefaultMaxFailedAttempts 是兼容环境变量和新建策略表单的初始值。
	DefaultMaxFailedAttempts = int32(5)
	// DefaultLockDurationMinutes 是兼容环境变量和新建策略表单的初始值。
	DefaultLockDurationMinutes = int32(15)
	// DefaultPasswordMinLength 是兼容环境变量和新建策略表单的初始值。
	DefaultPasswordMinLength = int32(8)
	// MaxPasswordMinLength 是密码最小长度允许的上限。
	MaxPasswordMinLength = int32(128)
	// DefaultPasswordHistoryCount 是兼容环境变量和新建策略表单的初始值。
	DefaultPasswordHistoryCount = int32(3)
	// MaxPasswordHistoryCount 是历史密码数量允许的上限。
	MaxPasswordHistoryCount = int32(100)
	// DefaultPasswordMinComplexityClasses 是兼容环境变量和新建策略表单的初始值。
	DefaultPasswordMinComplexityClasses = int32(3)
	// DefaultPasswordMaxAgeDays 是兼容环境变量和新建策略表单的初始值。
	DefaultPasswordMaxAgeDays = int32(90)
	policyCacheTTL            = 10 * 365 * 24 * time.Hour
)

// 作用域类型常量。
const (
	// ScopeGlobal 表示全局作用域。
	ScopeGlobal = int32(1)
	// ScopeTenant 表示租户作用域。
	ScopeTenant = int32(2)
	// ScopeUser 表示用户作用域。
	ScopeUser = int32(3)
)

// 限制类型常量。
const (
	// RestrictionBlacklist 表示黑名单限制。
	RestrictionBlacklist = int32(1)
	// RestrictionWhitelist 表示白名单限制。
	RestrictionWhitelist = int32(2)
)

// 限制方式常量。
const (
	// MethodIP 表示 IP 地址限制。
	MethodIP = int32(1)
	// MethodMAC 表示 MAC 地址限制。
	MethodMAC = int32(2)
	// MethodRegion 表示地区限制。
	MethodRegion = int32(3)
	// MethodTime 表示时间限制。
	MethodTime = int32(4)
	// MethodDevice 表示设备限制。
	MethodDevice = int32(5)
)

// 状态常量与通用 Status 枚举保持一致。
const (
	// StatusEnable 表示启用状态。
	StatusEnable = int32(1)
	// StatusDisable 表示禁用状态。
	StatusDisable = int32(2)
)

// Policy 表示一个作用域的登录策略。
type Policy struct {
	ID                           int64  `json:"id"`                              // 登录策略ID。
	ScopeType                    int32  `json:"scope_type"`                      // 作用域类型。
	TenantID                     int64  `json:"tenant_id"`                       // 租户ID。
	UserID                       int64  `json:"user_id"`                         // 用户ID。
	Status                       int32  `json:"status"`                          // 状态。
	MaxFailedAttempts            int32  `json:"max_failed_attempts"`             // 最大登录失败次数。
	LockDurationMinutes          int32  `json:"lock_duration_minutes"`           // 锁定时长（分钟）。
	PasswordMinLength            int32  `json:"password_min_length"`             // 密码最小长度。
	PasswordHistoryCount         int32  `json:"password_history_count"`          // 禁止重复使用的历史密码数量，零表示不启用。
	PasswordMinComplexityClasses int32  `json:"password_min_complexity_classes"` // 密码至少满足的字符类别数量。
	PasswordMaxAgeDays           int32  `json:"password_max_age_days"`           // 密码有效期天数，零表示不启用。
	InitialPasswordHash          string `json:"initial_password_hash,omitempty"` // 初始化密码哈希，不向接口返回。
	Rules                        []Rule `json:"rules"`                           // 该作用域下的限制规则。
}

// PasswordConfig 表示当前账号生效的密码策略。
type PasswordConfig struct {
	MinLength            int32  `json:"min_length"`                      // 密码最小长度。
	HistoryCount         int32  `json:"history_count"`                   // 禁止重复使用的历史密码数量，零表示不启用。
	MinComplexityClasses int32  `json:"min_complexity_classes"`          // 至少满足的字符类别数量。
	MaxAgeDays           int32  `json:"max_age_days"`                    // 密码有效期天数，零表示不启用。
	InitialPasswordHash  string `json:"initial_password_hash,omitempty"` // 初始化密码哈希。
}

// Rule 表示一条登录来源限制规则。
type Rule struct {
	ID                int64  `json:"id"`                 // 登录策略规则ID。
	PolicyID          int64  `json:"policy_id"`          // 登录策略ID。
	RestrictionType   int32  `json:"restriction_type"`   // 限制类型。
	RestrictionMethod int32  `json:"restriction_method"` // 限制方式。
	RestrictionValue  string `json:"restriction_value"`  // 限制值。
	Reason            string `json:"reason"`             // 限制原因。
	Status            int32  `json:"status"`             // 状态。
}

// PolicySet 表示缓存中的全部登录策略记录。
type PolicySet struct {
	Policies         []Policy       `json:"policies"`          // 登录策略记录集合。
	PasswordFallback PasswordConfig `json:"password_fallback"` // 无数据库策略时的兼容密码配置。
}

// Load 返回数据库策略为空时使用的默认密码策略。
func Load() PolicySet {
	return PolicySet{PasswordFallback: PasswordConfig{
		MinLength:            DefaultPasswordMinLength,
		HistoryCount:         DefaultPasswordHistoryCount,
		MinComplexityClasses: DefaultPasswordMinComplexityClasses,
		MaxAgeDays:           DefaultPasswordMaxAgeDays,
	}}
}

// LoadFromCache 从运行时缓存读取策略，缓存未配置时回退默认策略。
func LoadFromCache(store cache.Cache) PolicySet {
	policy, err := LoadFromCacheStrict(store)
	if err != nil {
		return Load()
	}
	return policy
}

// LoadFromCacheStrict 从运行时缓存读取策略，缓存故障时返回错误而不是降级放行。
func LoadFromCacheStrict(store cache.Cache) (PolicySet, error) {
	if store == nil {
		return PolicySet{}, errors.New("登录策略缓存未配置")
	}
	raw, err := store.Get(CacheKey)
	if err != nil {
		if isCacheMiss(err) {
			return Load(), nil
		}
		return PolicySet{}, fmt.Errorf("读取登录策略缓存失败: %w", err)
	}
	if raw == "" {
		return Load(), nil
	}
	policy := PolicySet{}
	if err = json.Unmarshal([]byte(raw), &policy); err != nil {
		return PolicySet{}, fmt.Errorf("解析登录策略缓存失败: %w", err)
	}
	if len(policy.Policies) == 0 {
		return Load(), nil
	}
	return policy, nil
}

// LoadPasswordConfig 从运行时缓存读取当前账号生效的密码策略。
func LoadPasswordConfig(store cache.Cache, tenantID, userID int64) (PasswordConfig, error) {
	policySet, err := LoadFromCacheStrict(store)
	if err != nil {
		return PasswordConfig{}, err
	}
	return policySet.PasswordConfigFor(tenantID, userID), nil
}

// SaveToCache 保存全部登录策略到运行时缓存。
func SaveToCache(store cache.Cache, policy PolicySet) error {
	if store == nil {
		return fmt.Errorf("登录策略缓存未配置")
	}
	payload, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("序列化登录策略失败: %w", err)
	}
	return store.Set(CacheKey, string(payload), policyCacheTTL)
}

// Validate 校验登录策略记录集合格式。
func (p PolicySet) Validate() error {
	seenPolicies := make(map[string]struct{}, len(p.Policies))
	for _, policy := range p.Policies {
		if policy.ScopeType != ScopeGlobal && policy.ScopeType != ScopeTenant && policy.ScopeType != ScopeUser {
			return fmt.Errorf("登录策略作用域类型无效: %d", policy.ScopeType)
		}
		policyKey := fmt.Sprintf("%d:%d:%d", policy.ScopeType, policy.TenantID, policy.UserID)
		if _, exists := seenPolicies[policyKey]; exists {
			return fmt.Errorf("登录策略作用域重复: %s", policyKey)
		}
		seenPolicies[policyKey] = struct{}{}
		if policy.Status != StatusEnable && policy.Status != StatusDisable {
			return fmt.Errorf("登录策略状态无效: %d", policy.Status)
		}
		if policy.MaxFailedAttempts <= 0 {
			return fmt.Errorf("最大登录失败次数必须大于零")
		}
		if policy.LockDurationMinutes <= 0 {
			return fmt.Errorf("锁定时长必须大于零")
		}
		if policy.PasswordMinLength == 0 {
			policy.PasswordMinLength = DefaultPasswordMinLength
		}
		if policy.PasswordMinComplexityClasses == 0 {
			policy.PasswordMinComplexityClasses = DefaultPasswordMinComplexityClasses
		}
		if policy.PasswordMinLength <= 0 || policy.PasswordMinLength > MaxPasswordMinLength {
			return fmt.Errorf("密码最小长度必须在一到%d之间", MaxPasswordMinLength)
		}
		if policy.PasswordHistoryCount < 0 || policy.PasswordHistoryCount > MaxPasswordHistoryCount {
			return fmt.Errorf("历史密码数量必须在零到%d之间", MaxPasswordHistoryCount)
		}
		if policy.PasswordMinComplexityClasses <= 0 || policy.PasswordMinComplexityClasses > 4 {
			return fmt.Errorf("密码复杂度字符类别数量必须在一到四之间")
		}
		if policy.PasswordMaxAgeDays < 0 {
			return fmt.Errorf("密码有效期不能小于零")
		}
		seenRules := make(map[string]struct{}, len(policy.Rules))
		for _, rule := range policy.Rules {
			if rule.RestrictionType != RestrictionBlacklist && rule.RestrictionType != RestrictionWhitelist {
				return fmt.Errorf("限制类型无效: %d", rule.RestrictionType)
			}
			if rule.RestrictionMethod < MethodIP || rule.RestrictionMethod > MethodDevice {
				return fmt.Errorf("限制方式无效: %d", rule.RestrictionMethod)
			}
			if rule.RestrictionValue == "" {
				return fmt.Errorf("限制值不能为空")
			}
			ruleKey := fmt.Sprintf("%d:%d:%s", rule.RestrictionType, rule.RestrictionMethod, rule.RestrictionValue)
			if _, exists := seenRules[ruleKey]; exists {
				return fmt.Errorf("登录策略规则重复: %s", ruleKey)
			}
			seenRules[ruleKey] = struct{}{}
			if rule.Status != StatusEnable && rule.Status != StatusDisable {
				return fmt.Errorf("登录策略规则状态无效: %d", rule.Status)
			}
			if err := validateRuleValue(rule.RestrictionMethod, rule.RestrictionValue); err != nil {
				return err
			}
		}
	}
	return nil
}

// PasswordConfigFor 返回当前账号匹配的密码策略，未匹配数据库策略时使用兼容配置。
func (p PolicySet) PasswordConfigFor(tenantID, userID int64) PasswordConfig {
	config := p.PasswordFallback
	if config.MinLength <= 0 {
		config.MinLength = DefaultPasswordMinLength
	}
	if config.HistoryCount < 0 {
		config.HistoryCount = DefaultPasswordHistoryCount
	}
	if config.MinComplexityClasses <= 0 {
		config.MinComplexityClasses = DefaultPasswordMinComplexityClasses
	}
	for _, scope := range []int32{ScopeUser, ScopeTenant, ScopeGlobal} {
		for _, policy := range p.Policies {
			if policy.Status != StatusEnable || policy.ScopeType != scope {
				continue
			}
			if scope == ScopeUser && (policy.UserID == 0 || policy.UserID != userID) {
				continue
			}
			if scope == ScopeTenant && (policy.TenantID == 0 || policy.TenantID != tenantID) {
				continue
			}
			config.MinLength = normalizedPasswordMinLength(policy.PasswordMinLength)
			config.HistoryCount = normalizedPasswordHistoryCount(policy.PasswordHistoryCount)
			config.MinComplexityClasses = normalizedPasswordMinComplexityClasses(policy.PasswordMinComplexityClasses)
			config.MaxAgeDays = policy.PasswordMaxAgeDays
			config.InitialPasswordHash = policy.InitialPasswordHash
			return config
		}
	}
	return config
}

// EvaluateFor 判断当前登录来源是否被匹配的启用策略拒绝。
func (p PolicySet) EvaluateFor(tenantID, userID int64, clientIP, mac, region, device string, now time.Time) (bool, string) {
	for _, policy := range p.Policies {
		if policy.Status != StatusEnable || !policyMatches(policy, tenantID, userID) {
			continue
		}
		if blocked, reason := evaluateRules(policy.Rules, clientIP, mac, region, device, now); blocked {
			return true, reason
		}
	}
	return false, ""
}

// FailureConfig 返回当前账号使用的失败锁定参数，未匹配启用策略时返回零值。
func (p PolicySet) FailureConfig(tenantID, userID int64) (int, time.Duration) {
	for _, scope := range []int32{ScopeUser, ScopeTenant, ScopeGlobal} {
		for _, policy := range p.Policies {
			if policy.Status != StatusEnable || policy.ScopeType != scope {
				continue
			}
			if scope == ScopeUser && (policy.UserID == 0 || policy.UserID != userID) {
				continue
			}
			if scope == ScopeTenant && (policy.TenantID == 0 || policy.TenantID != tenantID) {
				continue
			}
			if policy.MaxFailedAttempts <= 0 || policy.LockDurationMinutes <= 0 {
				return 0, 0
			}
			return int(policy.MaxFailedAttempts), time.Duration(policy.LockDurationMinutes) * time.Minute
		}
	}
	return 0, 0
}

// PasswordMaxAgeDaysFor 返回当前账号使用的密码有效期，未匹配启用策略时返回零值。
func (p PolicySet) PasswordMaxAgeDaysFor(tenantID, userID int64) int32 {
	return p.PasswordConfigFor(tenantID, userID).MaxAgeDays
}

// normalizedPasswordMinLength 为旧策略记录补充密码最小长度默认值。
func normalizedPasswordMinLength(value int32) int32 {
	if value <= 0 {
		return DefaultPasswordMinLength
	}
	return value
}

// normalizedPasswordHistoryCount 为异常负值策略记录补充历史密码数量默认值。
func normalizedPasswordHistoryCount(value int32) int32 {
	if value < 0 {
		return DefaultPasswordHistoryCount
	}
	return value
}

// normalizedPasswordMinComplexityClasses 为旧策略记录补充复杂度字符类别默认值。
func normalizedPasswordMinComplexityClasses(value int32) int32 {
	if value <= 0 || value > 4 {
		return DefaultPasswordMinComplexityClasses
	}
	return value
}

// policyMatches 判断策略作用域是否匹配当前登录对象。
func policyMatches(policy Policy, tenantID, userID int64) bool {
	switch policy.ScopeType {
	case ScopeGlobal:
		return true
	case ScopeTenant:
		return tenantID > 0 && policy.TenantID == tenantID
	case ScopeUser:
		return userID > 0 && policy.UserID == userID
	default:
		return false
	}
}

// evaluateRules 按限制方式聚合黑白名单并判断登录来源。
func evaluateRules(rules []Rule, clientIP, mac, region, device string, now time.Time) (bool, string) {
	for _, method := range []int32{MethodIP, MethodMAC, MethodRegion, MethodTime, MethodDevice} {
		blacklist := make([]Rule, 0)
		whitelist := make([]Rule, 0)
		for _, rule := range rules {
			if rule.Status != StatusEnable || rule.RestrictionMethod != method {
				continue
			}
			if rule.RestrictionType == RestrictionBlacklist {
				blacklist = append(blacklist, rule)
			} else {
				whitelist = append(whitelist, rule)
			}
		}
		value := restrictionTarget(method, clientIP, mac, region, device, now)
		for _, rule := range blacklist {
			if matchesRestriction(method, value, rule.RestrictionValue, now) {
				return true, ruleReason(rule, blacklistReason(method))
			}
		}
		if len(whitelist) > 0 {
			matched := false
			for _, rule := range whitelist {
				if matchesRestriction(method, value, rule.RestrictionValue, now) {
					matched = true
					break
				}
			}
			if !matched {
				return true, whitelistReason(method)
			}
		}
	}
	return false, ""
}

// restrictionTarget 提取指定限制方式的当前请求值。
func restrictionTarget(method int32, clientIP, mac, region, device string, now time.Time) string {
	switch method {
	case MethodIP:
		return clientIP
	case MethodMAC:
		return mac
	case MethodRegion:
		return region
	case MethodTime:
		return now.Format("15:04")
	case MethodDevice:
		return device
	default:
		return ""
	}
}

// matchesRestriction 判断单条限制规则是否命中。
func matchesRestriction(method int32, target, value string, now time.Time) bool {
	if method == MethodTime {
		return matchTime(now, value)
	}
	if method == MethodIP {
		return matchIP(target, value)
	}
	if target == "" || value == "" {
		return false
	}
	if method == MethodDevice || method == MethodMAC {
		return strings.Contains(strings.ToLower(target), strings.ToLower(value))
	}
	return strings.EqualFold(target, value)
}

// validateRuleValue 校验限制值格式。
func validateRuleValue(method int32, value string) error {
	if method == MethodIP {
		if strings.Contains(value, "/") {
			if _, _, err := net.ParseCIDR(value); err != nil {
				return fmt.Errorf("IP/CIDR 格式无效: %s", value)
			}
		} else if net.ParseIP(value) == nil {
			return fmt.Errorf("IP 格式无效: %s", value)
		}
	}
	if method == MethodTime {
		parts := strings.Split(value, "-")
		if len(parts) != 2 {
			return fmt.Errorf("时间窗口格式无效: %s", value)
		}
		start, startOK := parseMinute(parts[0])
		end, endOK := parseMinute(parts[1])
		if !startOK || !endOK || start == end {
			return fmt.Errorf("时间窗口格式无效: %s", value)
		}
	}
	return nil
}

// matchIP 判断单个地址是否匹配精确 IP 或 CIDR。
func matchIP(value, policy string) bool {
	if value == "" || policy == "" {
		return false
	}
	if strings.Contains(policy, "/") {
		_, network, err := net.ParseCIDR(policy)
		if err != nil {
			return false
		}
		ip := net.ParseIP(value)
		return ip != nil && network.Contains(ip)
	}
	return value == policy
}

// matchTime 判断当前时间是否落在时间窗口内。
func matchTime(now time.Time, value string) bool {
	parts := strings.Split(value, "-")
	if len(parts) != 2 {
		return false
	}
	start, ok := parseMinute(parts[0])
	if !ok {
		return false
	}
	end, ok := parseMinute(parts[1])
	if !ok || start == end {
		return false
	}
	current := now.Hour()*60 + now.Minute()
	if start < end {
		return current >= start && current < end
	}
	return current >= start || current < end
}

// parseMinute 将 HH:MM 时间转换为当天分钟数。
func parseMinute(value string) (int, bool) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 2 {
		return 0, false
	}
	hour, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, false
	}
	minute, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, false
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, false
	}
	return hour*60 + minute, true
}

// ruleReason 返回规则自定义原因或默认原因。
func ruleReason(rule Rule, fallback string) string {
	if rule.Reason != "" {
		return rule.Reason
	}
	return fallback
}

// blacklistReason 返回黑名单命中的默认原因。
func blacklistReason(method int32) string {
	switch method {
	case MethodIP:
		return "登录 IP 命中黑名单"
	case MethodMAC:
		return "登录 MAC 命中黑名单"
	case MethodRegion:
		return "登录地区命中黑名单"
	case MethodTime:
		return "当前时间不允许登录"
	case MethodDevice:
		return "登录设备命中黑名单"
	default:
		return "登录来源命中黑名单"
	}
}

// whitelistReason 返回白名单未命中原因。
func whitelistReason(method int32) string {
	switch method {
	case MethodIP:
		return "登录 IP 不在白名单"
	case MethodMAC:
		return "登录 MAC 不在白名单"
	case MethodRegion:
		return "登录地区不在白名单"
	case MethodTime:
		return "当前时间不在允许窗口"
	case MethodDevice:
		return "登录设备不在白名单"
	default:
		return "登录来源不在白名单"
	}
}

// isCacheMiss 判断缓存键不存在，而不是缓存服务不可用。
func isCacheMiss(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, redis.Nil) || strings.Contains(strings.ToLower(err.Error()), "not found") || strings.Contains(strings.ToLower(err.Error()), "key expired")
}
