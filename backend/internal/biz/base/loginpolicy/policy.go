// Package loginpolicy 提供登录来源策略匹配能力。
package loginpolicy

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/liujitcn/kratos-kit/cache"
	"github.com/redis/go-redis/v9"
)

// CacheKey 是登录来源策略缓存键。
const CacheKey = "security:login-policy"

const policyCacheTTL = 10 * 365 * 24 * time.Hour

// Policy 表示一条登录来源策略。
// 策略由平台管理员维护并缓存在运行时；任一黑名单命中或白名单未命中都会拒绝登录。
type Policy struct {
	Enabled         bool     `json:"enabled"`          // 是否启用来源策略校验。
	IPBlacklist     []string `json:"ip_blacklist"`     // 禁止访问的 IP 或 CIDR 列表。
	IPWhitelist     []string `json:"ip_whitelist"`     // 允许访问的 IP 或 CIDR 列表。
	TimeWindows     []string `json:"time_windows"`     // 禁止登录时间窗口，格式为 HH:MM-HH:MM。
	DeviceBlacklist []string `json:"device_blacklist"` // 禁止访问的设备标识或 User-Agent 匹配项。
	DeviceWhitelist []string `json:"device_whitelist"` // 允许访问的设备标识或 User-Agent 匹配项。
	Rules           []Rule   `json:"rules"`            // 按租户编码或用户名匹配的定向规则。
}

// Rule 表示一条按目标对象匹配的登录来源规则。
type Rule struct {
	TargetType      string   `json:"target_type"`      // 目标类型：TENANT 或 USER。
	TargetValue     string   `json:"target_value"`     // 目标值：租户编码或用户名。
	Enabled         bool     `json:"enabled"`          // 是否启用规则。
	IPBlacklist     []string `json:"ip_blacklist"`     // 禁止访问的 IP 或 CIDR 列表。
	IPWhitelist     []string `json:"ip_whitelist"`     // 允许访问的 IP 或 CIDR 列表。
	TimeWindows     []string `json:"time_windows"`     // 禁止登录时间窗口。
	DeviceBlacklist []string `json:"device_blacklist"` // 禁止访问的设备标识或 User-Agent 匹配项。
	DeviceWhitelist []string `json:"device_whitelist"` // 允许访问的设备标识或 User-Agent 匹配项。
}

// Load 从环境变量读取登录来源策略。
func Load() Policy {
	policy := Policy{
		IPBlacklist:     split(os.Getenv("LOGIN_POLICY_IP_BLACKLIST")),
		IPWhitelist:     split(os.Getenv("LOGIN_POLICY_IP_WHITELIST")),
		TimeWindows:     split(os.Getenv("LOGIN_POLICY_TIME_WINDOWS")),
		DeviceBlacklist: split(os.Getenv("LOGIN_POLICY_DEVICE_BLACKLIST")),
		DeviceWhitelist: split(os.Getenv("LOGIN_POLICY_DEVICE_WHITELIST")),
	}
	policy.Enabled = len(policy.IPBlacklist)+len(policy.IPWhitelist)+len(policy.TimeWindows)+len(policy.DeviceBlacklist)+len(policy.DeviceWhitelist) > 0
	return policy
}

// LoadFromCache 从运行时缓存读取策略，未配置时回退环境变量。
func LoadFromCache(store cache.Cache) Policy {
	policy, err := LoadFromCacheStrict(store)
	if err != nil {
		return Load()
	}
	return policy
}

// LoadFromCacheStrict 从运行时缓存读取策略，缓存故障时返回错误而不是降级放行。
func LoadFromCacheStrict(store cache.Cache) (Policy, error) {
	if store == nil {
		return Policy{}, errors.New("登录策略缓存未配置")
	}
	raw, err := store.Get(CacheKey)
	if err != nil {
		if isCacheMiss(err) {
			return Load(), nil
		}
		return Policy{}, fmt.Errorf("读取登录策略缓存失败: %w", err)
	}
	if raw == "" {
		return Load(), nil
	}
	policy := Policy{}
	if err = json.Unmarshal([]byte(raw), &policy); err != nil {
		return Policy{}, fmt.Errorf("解析登录策略缓存失败: %w", err)
	}
	return policy, nil
}

// SaveToCache 保存登录来源策略到运行时缓存。
func SaveToCache(store cache.Cache, policy Policy) error {
	if store == nil {
		return fmt.Errorf("登录策略缓存未配置")
	}
	payload, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("序列化登录策略失败: %w", err)
	}
	return store.Set(CacheKey, string(payload), policyCacheTTL)
}

// Validate 校验登录来源策略配置格式。
func (p Policy) Validate() error {
	if err := validateRuleLists(p.IPBlacklist, p.IPWhitelist, p.TimeWindows); err != nil {
		return err
	}
	seenTargets := make(map[string]struct{}, len(p.Rules))
	for _, rule := range p.Rules {
		if rule.TargetType != "TENANT" && rule.TargetType != "USER" {
			return fmt.Errorf("定向规则目标类型无效: %s", rule.TargetType)
		}
		if rule.TargetValue == "" {
			return fmt.Errorf("定向规则目标值不能为空")
		}
		targetKey := rule.TargetType + "\x00" + rule.TargetValue
		if _, exists := seenTargets[targetKey]; exists {
			return fmt.Errorf("定向规则目标重复: %s/%s", rule.TargetType, rule.TargetValue)
		}
		seenTargets[targetKey] = struct{}{}
		if err := validateRuleLists(rule.IPBlacklist, rule.IPWhitelist, rule.TimeWindows); err != nil {
			return err
		}
	}
	return nil
}

// Evaluate 判断登录来源是否被策略拒绝，并返回可记录的原因。
func (p Policy) Evaluate(clientIP, device string, now time.Time) (bool, string) {
	return p.EvaluateFor("", "", clientIP, device, now)
}

// EvaluateFor 按全局策略和目标对象定向策略判断登录来源。
func (p Policy) EvaluateFor(tenantCode, userName, clientIP, device string, now time.Time) (bool, string) {
	if p.Enabled {
		if blocked, reason := evaluateRuleLists(p.IPBlacklist, p.IPWhitelist, p.TimeWindows, p.DeviceBlacklist, p.DeviceWhitelist, clientIP, device, now); blocked {
			return true, reason
		}
	}
	for _, rule := range p.Rules {
		if !rule.Enabled || !ruleMatches(rule, tenantCode, userName) {
			continue
		}
		if blocked, reason := evaluateRuleLists(rule.IPBlacklist, rule.IPWhitelist, rule.TimeWindows, rule.DeviceBlacklist, rule.DeviceWhitelist, clientIP, device, now); blocked {
			return true, reason
		}
	}
	return false, ""
}

// validateRuleLists 校验 IP 和时间窗口规则格式。
func validateRuleLists(ipBlacklist, ipWhitelist, timeWindows []string) error {
	var err error
	for _, value := range append(append([]string{}, ipBlacklist...), ipWhitelist...) {
		if strings.Contains(value, "/") {
			_, _, err = net.ParseCIDR(value)
			if err != nil {
				return fmt.Errorf("IP/CIDR 格式无效: %s", value)
			}
			continue
		}
		if net.ParseIP(value) == nil {
			return fmt.Errorf("IP 格式无效: %s", value)
		}
	}
	for _, value := range timeWindows {
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

// evaluateRuleLists 判断来源地址、时间和设备是否命中规则。
func evaluateRuleLists(ipBlacklist, ipWhitelist, timeWindows, deviceBlacklist, deviceWhitelist []string, clientIP, device string, now time.Time) (bool, string) {
	for _, value := range ipBlacklist {
		if matchIP(clientIP, value) {
			return true, "登录 IP 命中黑名单"
		}
	}
	if len(ipWhitelist) > 0 && !matchesIP(clientIP, ipWhitelist) {
		return true, "登录 IP 不在白名单"
	}
	for _, value := range timeWindows {
		if matchTime(now, value) {
			return true, "当前时间不允许登录"
		}
	}
	for _, value := range deviceBlacklist {
		if matchDevice(device, value) {
			return true, "登录设备命中黑名单"
		}
	}
	if len(deviceWhitelist) > 0 && !matchesDevice(deviceWhitelist, device) {
		return true, "登录设备不在白名单"
	}
	return false, ""
}

// ruleMatches 判断定向规则是否匹配当前租户或用户。
func ruleMatches(rule Rule, tenantCode, userName string) bool {
	if rule.TargetType == "TENANT" {
		return tenantCode != "" && rule.TargetValue == tenantCode
	}
	return userName != "" && rule.TargetValue == userName
}

// split 将逗号分隔配置解析为去除空项的字符串列表。
func split(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// matchesIP 判断地址是否匹配任一 IP 或 CIDR 规则。
func matchesIP(value string, policies []string) bool {
	for _, policy := range policies {
		if matchIP(value, policy) {
			return true
		}
	}
	return false
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

// matchTime 判断当前时间是否落在禁止时间窗口内。
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
	var err error
	var hour int
	hour, err = strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, false
	}
	var minute int
	minute, err = strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, false
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, false
	}
	return hour*60 + minute, true
}

// matchesDevice 判断设备标识是否匹配任一规则。
func matchesDevice(values []string, target string) bool {
	for _, value := range values {
		if matchDevice(target, value) {
			return true
		}
	}
	return false
}

// matchDevice 判断设备标识是否包含规则文本。
func matchDevice(device, policy string) bool {
	if device == "" || policy == "" {
		return false
	}
	return strings.Contains(strings.ToLower(device), strings.ToLower(policy))
}

// isCacheMiss 判断缓存键不存在，而不是缓存服务不可用。
func isCacheMiss(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, redis.Nil) || strings.Contains(strings.ToLower(err.Error()), "not found") || strings.Contains(strings.ToLower(err.Error()), "key expired")
}
