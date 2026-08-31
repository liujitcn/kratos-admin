// Package password 提供后台账号口令安全策略。
package password

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/liujitcn/go-utils/crypto"
	"github.com/liujitcn/kratos-kit/cache"
	"github.com/redis/go-redis/v9"
)

const passwordStateTTL = 10 * 365 * 24 * time.Hour

// ErrWeakPassword 表示口令未达到复杂度要求。
var ErrWeakPassword = errors.New("口令长度不足或复杂度不符合要求")

// MinLength 返回口令最小长度。
func MinLength() int {
	value := os.Getenv("PASSWORD_MIN_LENGTH")
	if value != "" {
		length, err := strconv.Atoi(value)
		if err == nil && length > 0 {
			return length
		}
	}
	return 8
}

// HistoryCount 返回需要避免重复使用的历史口令数量。
func HistoryCount() int {
	value := os.Getenv("PASSWORD_HISTORY_COUNT")
	if value != "" {
		count, err := strconv.Atoi(value)
		if err == nil && count >= 0 {
			return count
		}
	}
	return 3
}

// MaxAgeDays 返回口令有效期天数，零表示不启用有效期。
func MaxAgeDays() int {
	value := os.Getenv("PASSWORD_MAX_AGE_DAYS")
	if value != "" {
		days, err := strconv.Atoi(value)
		if err == nil && days >= 0 {
			return days
		}
	}
	return 90
}

// ValidateComplexity 校验口令长度和字符类别复杂度。
func ValidateComplexity(plain string) error {
	if len([]rune(plain)) < MinLength() {
		return ErrWeakPassword
	}
	var lower, upper, digit, symbol bool
	for _, char := range plain {
		switch {
		case unicode.IsLower(char):
			lower = true
		case unicode.IsUpper(char):
			upper = true
		case unicode.IsDigit(char):
			digit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			symbol = true
		}
	}
	classes := 0
	for _, present := range []bool{lower, upper, digit, symbol} {
		if present {
			classes++
		}
	}
	if classes < 3 {
		return ErrWeakPassword
	}
	return nil
}

// CheckHistory 判断新口令是否重复使用近期历史口令。
func CheckHistory(store cache.Cache, userID int64, plain string) error {
	if store == nil {
		return errors.New("口令历史缓存未配置")
	}
	if userID <= 0 || HistoryCount() <= 0 {
		return nil
	}
	raw, err := store.Get(historyKey(userID))
	if err != nil {
		if isCacheMiss(err) {
			return nil
		}
		return fmt.Errorf("读取口令历史失败: %w", err)
	}
	if raw == "" {
		return nil
	}
	return CheckHistoryJSON(raw, plain)
}

// CheckHistoryJSON 校验 JSON 编码的历史口令哈希列表。
func CheckHistoryJSON(raw, plain string) error {
	if raw == "" {
		return nil
	}
	var history []string
	if err := json.Unmarshal([]byte(raw), &history); err != nil {
		return fmt.Errorf("解析口令历史失败: %w", err)
	}
	for _, hash := range history {
		if err := crypto.Verify(plain, hash); err == nil {
			return errors.New("新口令不能与近期历史口令相同")
		}
	}
	return nil
}

// RecordHistory 记录用户修改前的口令哈希，不保存明文口令。
func RecordHistory(store cache.Cache, userID int64, oldHash string) error {
	if store == nil {
		return errors.New("口令历史缓存未配置")
	}
	if userID <= 0 || oldHash == "" || HistoryCount() <= 0 {
		return nil
	}
	var history []string
	raw, err := store.Get(historyKey(userID))
	if err == nil && raw != "" {
		if err = json.Unmarshal([]byte(raw), &history); err != nil {
			return fmt.Errorf("解析口令历史失败: %w", err)
		}
	} else if err != nil && !isCacheMiss(err) {
		return fmt.Errorf("读取口令历史失败: %w", err)
	}
	history = append([]string{oldHash}, history...)
	if len(history) > HistoryCount() {
		history = history[:HistoryCount()]
	}
	var payload []byte
	payload, err = json.Marshal(history)
	if err != nil {
		return fmt.Errorf("序列化历史口令失败: %w", err)
	}
	return store.Set(historyKey(userID), string(payload), passwordStateTTL)
}

// AppendHistoryJSON 将旧口令哈希追加到持久化历史列表并限制保留数量。
func AppendHistoryJSON(raw, oldHash string) (string, error) {
	if oldHash == "" || HistoryCount() <= 0 {
		return raw, nil
	}
	var history []string
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &history); err != nil {
			return "", fmt.Errorf("解析口令历史失败: %w", err)
		}
	}
	history = append([]string{oldHash}, history...)
	if len(history) > HistoryCount() {
		history = history[:HistoryCount()]
	}
	payload, err := json.Marshal(history)
	if err != nil {
		return "", fmt.Errorf("序列化历史口令失败: %w", err)
	}
	return string(payload), nil
}

// MarkChanged 标记用户最近一次修改口令的时间。
func MarkChanged(store cache.Cache, userID int64) error {
	if store == nil {
		return errors.New("口令状态缓存未配置")
	}
	if userID <= 0 {
		return nil
	}
	return store.Set(changedAtKey(userID), strconv.FormatInt(time.Now().Unix(), 10), passwordStateTTL)
}

// EnsureChanged 为历史账号补充首次口令更新时间，不覆盖已有时间。
func EnsureChanged(store cache.Cache, userID int64) error {
	if store == nil {
		return errors.New("口令状态缓存未配置")
	}
	if userID <= 0 {
		return nil
	}
	raw, err := store.Get(changedAtKey(userID))
	if err == nil && raw != "" {
		return nil
	}
	if err != nil && !isCacheMiss(err) {
		return fmt.Errorf("读取口令更新时间失败: %w", err)
	}
	return MarkChanged(store, userID)
}

// CheckExpiry 判断用户口令是否超过配置的有效期，并区分缓存故障。
func CheckExpiry(store cache.Cache, userID int64, now time.Time) (bool, error) {
	maxAge := MaxAgeDays()
	if store == nil {
		return false, errors.New("口令状态缓存未配置")
	}
	if userID <= 0 || maxAge <= 0 {
		return false, nil
	}
	raw, err := store.Get(changedAtKey(userID))
	if err != nil {
		if isCacheMiss(err) {
			return false, nil
		}
		return false, fmt.Errorf("读取口令更新时间失败: %w", err)
	}
	if raw == "" {
		return false, nil
	}
	var changedAt int64
	changedAt, err = strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return false, fmt.Errorf("口令更新时间无效: %w", err)
	}
	if changedAt <= 0 {
		return false, errors.New("口令更新时间无效")
	}
	return now.Sub(time.Unix(changedAt, 0)) >= time.Duration(maxAge)*24*time.Hour, nil
}

// IsExpiredAt 根据数据库中的口令修改时间判断是否过期。
func IsExpiredAt(changedAt, now time.Time) bool {
	maxAge := MaxAgeDays()
	if maxAge <= 0 || changedAt.IsZero() {
		return false
	}
	return now.Sub(changedAt) >= time.Duration(maxAge)*24*time.Hour
}

// IsExpired 判断用户口令是否超过配置的有效期。
func IsExpired(store cache.Cache, userID int64, now time.Time) bool {
	expired, err := CheckExpiry(store, userID, now)
	return err == nil && expired
}

// isCacheMiss 判断缓存键不存在，而不是缓存服务故障。
func isCacheMiss(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, redis.Nil) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "key not found") || strings.Contains(message, "key expired") || strings.Contains(message, "not found")
}

// historyKey 返回用户口令历史的缓存键。
func historyKey(userID int64) string {
	return fmt.Sprintf("password_history:%d", userID)
}

// changedAtKey 返回用户口令修改时间的缓存键。
func changedAtKey(userID int64) string {
	return fmt.Sprintf("password_changed_at:%d", userID)
}
