package sessionregistry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v3/transport"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/sessionstate"

	"github.com/liujitcn/go-utils/id"
	"github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/cache"
	"github.com/redis/go-redis/v9"
)

const (
	indexKey        = "security:login-session:ids"
	recordKeyPrefix = "security:login-session:"
	recordTTL       = 365 * 24 * time.Hour
)

// indexMu 保护内存缓存返回的索引映射快照；跨实例写入仍使用缓存原子哈希操作。
var indexMu sync.Mutex

// Record 保存一个登录会话的展示信息和令牌关联信息。
type Record struct {
	SessionID    string    `json:"session_id"`
	UserID       int64     `json:"user_id"`
	UserName     string    `json:"user_name"`
	TenantCode   string    `json:"tenant_code"`
	ClientIP     string    `json:"client_ip"`
	Device       string    `json:"device"`
	UserAgent    string    `json:"user_agent"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	IssuedAt     time.Time `json:"issued_at"`
}

// NewRecord 创建一个新的登录会话记录。
func NewRecord(userID int64, userName, tenantCode, clientIP, device, userAgent, accessToken, refreshToken string, issuedAt time.Time) Record {
	return Record{SessionID: id.NewGUIDv4NoHyphen(), UserID: userID, UserName: userName, TenantCode: tenantCode, ClientIP: clientIP, Device: device, UserAgent: userAgent, AccessToken: accessToken, RefreshToken: refreshToken, IssuedAt: issuedAt}
}

// Register 保存登录会话记录并加入全局索引。
func Register(store cache.Cache, record Record) error {
	if store == nil || record.SessionID == "" || record.UserID <= 0 {
		return errors.New("登录会话参数无效")
	}
	if err := saveRecord(store, record); err != nil {
		return err
	}
	indexMu.Lock()
	defer indexMu.Unlock()
	return store.HSet(indexKey, record.SessionID, "1")
}

// UpdateTokens 更新会话轮换后的令牌关联。
func UpdateTokens(store cache.Cache, sessionID, accessToken, refreshToken string) error {
	record, err := Get(store, sessionID)
	if err != nil {
		return err
	}
	record.AccessToken = accessToken
	record.RefreshToken = refreshToken
	return saveRecord(store, record)
}

// Get 读取指定登录会话记录。
func Get(store cache.Cache, sessionID string) (Record, error) {
	raw, err := store.Get(recordKey(sessionID))
	if err != nil {
		return Record{}, err
	}
	record := Record{}
	if err = json.Unmarshal([]byte(raw), &record); err != nil {
		return Record{}, err
	}
	return record, nil
}

// List 返回当前仍有访问令牌或刷新令牌的会话记录。
func List(store cache.Cache, userToken *data.UserToken) ([]Record, error) {
	ids, err := loadIndex(store)
	if err != nil {
		return nil, err
	}
	result := make([]Record, 0, len(ids))
	for _, sessionID := range ids {
		var record Record
		record, err = Get(store, sessionID)
		if err != nil {
			if isCacheMiss(err) {
				if err = removeIndex(store, sessionID); err != nil && !isCacheMiss(err) {
					return nil, err
				}
				continue
			}
			return nil, err
		}
		// 令牌轮换与展示记录更新之间存在短暂间隔，查询不得据旧令牌删除会话。
		if userToken.IsAccessTokenValid(record.UserID, record.AccessToken) || userToken.IsRefreshTokenValid(record.UserID, record.RefreshToken) {
			result = append(result, record)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].IssuedAt.After(result[j].IssuedAt) })
	return result, nil
}

// FindByAccessToken 按访问令牌查找当前会话。
func FindByAccessToken(store cache.Cache, userToken *data.UserToken, userID int64, accessToken string) (Record, error) {
	sessions, err := userToken.GetTokenSessions(userID)
	if err != nil {
		return Record{}, err
	}
	for _, session := range sessions {
		if session.AccessToken == accessToken && userToken.IsAccessTokenValid(userID, accessToken) {
			return Get(store, session.SessionID)
		}
	}
	return Record{}, redis.Nil
}

// FindByRefreshToken 按刷新令牌查找当前会话。
func FindByRefreshToken(store cache.Cache, userToken *data.UserToken, userID int64, refreshToken string) (Record, error) {
	sessions, err := userToken.GetTokenSessions(userID)
	if err != nil {
		return Record{}, err
	}
	for _, session := range sessions {
		if session.RefreshToken == refreshToken && userToken.IsRefreshTokenValid(userID, refreshToken) {
			return Get(store, session.SessionID)
		}
	}
	return Record{}, redis.Nil
}

// Remove 撤销并删除指定登录会话。
func Remove(store cache.Cache, userToken *data.UserToken, record Record) error {
	err := userToken.RemoveTokenForSession(record.UserID, record.SessionID, record.AccessToken, record.RefreshToken)
	if err != nil {
		return err
	}
	if err = sessionstate.Clear(store, record.SessionID); err != nil {
		return err
	}
	if err = store.Del(RefreshTokenAuthKey(record.RefreshToken)); err != nil {
		return err
	}
	if err = store.Del(recordKey(record.SessionID)); err != nil {
		return err
	}
	if err = removeIndex(store, record.SessionID); err != nil && !isCacheMiss(err) {
		return err
	}
	return nil
}

// RemoveAll 撤销用户的全部登录会话。
func RemoveAll(store cache.Cache, userToken *data.UserToken, userID int64) error {
	sessions, err := userToken.GetTokenSessions(userID)
	if err != nil {
		return err
	}
	for _, session := range sessions {
		record := Record{SessionID: session.SessionID, UserID: userID, AccessToken: session.AccessToken, RefreshToken: session.RefreshToken}
		if err = Remove(store, userToken, record); err != nil {
			return err
		}
	}
	return userToken.RemoveToken(userID)
}

// RefreshTokenAuthKey 返回刷新令牌认证信息缓存键。
func RefreshTokenAuthKey(refreshToken string) string {
	return "refresh_token_auth:" + refreshToken
}

// saveRecord 持久化单个会话展示记录。
func saveRecord(store cache.Cache, record Record) error {
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return store.Set(recordKey(record.SessionID), string(payload), recordTTL)
}

// loadIndex 读取会话编号快照，缓存哈希以会话为单位原子增删。
func loadIndex(store cache.Cache) ([]string, error) {
	indexMu.Lock()
	defer indexMu.Unlock()
	entries, err := store.HGetAll(indexKey)
	if err != nil {
		if isCacheMiss(err) {
			return nil, nil
		}
		return nil, err
	}
	ids := make([]string, 0, len(entries))
	for sessionID := range entries {
		ids = append(ids, sessionID)
	}
	return ids, nil
}

// removeIndex 原子删除一个会话索引项，同时保护内存缓存快照。
func removeIndex(store cache.Cache, sessionID string) error {
	indexMu.Lock()
	defer indexMu.Unlock()
	return store.HDel(indexKey, sessionID)
}

// isCacheMiss 统一识别 Redis 与内存缓存的缺失键。
func isCacheMiss(err error) bool {
	return errors.Is(err, redis.Nil) || err != nil && (strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "key expired"))
}

// AccessToken 从 HTTP 或 gRPC 请求头提取当前会话的访问令牌。
func AccessToken(ctx context.Context) string {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return ""
	}
	parts := strings.Fields(tr.RequestHeader().Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

// recordKey 返回会话展示记录的缓存键。
func recordKey(sessionID string) string {
	return fmt.Sprintf("%s%s", recordKeyPrefix, sessionID)
}
