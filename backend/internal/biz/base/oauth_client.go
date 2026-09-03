package biz

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/transport"
	httpTransport "github.com/go-kratos/kratos/v3/transport/http"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/oauthsecret"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	coreconst "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/redis/go-redis/v9"

	"github.com/liujitcn/gorm-kit/repository"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"gorm.io/gorm"
)

const oauthClientGrantType = "client_credentials"

const (
	oauthClientFailurePrefix  = "oauth_client_failure:"
	oauthClientFailureWindow  = 5 * time.Minute
	oauthClientFailureMaximum = 10
)

// OauthClientTokenCase 处理开放授权客户端令牌签发。
type OauthClientTokenCase struct {
	*biz.BaseCase
	oauthClientRepo *data.OauthClientRepository
	baseTenantRepo  *data.BaseTenantRepository
	userToken       *authData.UserToken
	protector       *oauthsecret.Protector
}

// NewOauthClientTokenCase 创建开放授权客户端令牌业务实例。
func NewOauthClientTokenCase(baseCase *biz.BaseCase, oauthClientRepo *data.OauthClientRepository, baseTenantRepo *data.BaseTenantRepository, userToken *authData.UserToken, protector *oauthsecret.Protector) *OauthClientTokenCase {
	return &OauthClientTokenCase{BaseCase: baseCase, oauthClientRepo: oauthClientRepo, baseTenantRepo: baseTenantRepo, userToken: userToken, protector: protector}
}

// IssueOauthClientToken 使用客户端凭据签发访问令牌。
func (c *OauthClientTokenCase) IssueOauthClientToken(ctx context.Context, req *basev1.IssueOauthClientTokenRequest) (*basev1.IssueOauthClientTokenResponse, error) {
	if req.GetGrantType() != "" && req.GetGrantType() != oauthClientGrantType {
		return nil, errorsx.InvalidArgument("授权类型必须为 client_credentials")
	}
	if c.userToken == nil {
		return nil, errorsx.Internal("访问令牌服务未初始化")
	}
	locked, err := c.oauthClientFailureLocked(ctx, req.GetClientId())
	if err != nil {
		return nil, errorsx.Internal("读取客户端认证失败状态失败").WithCause(err)
	}
	if locked {
		return nil, errorsx.PermissionDenied("客户端认证失败次数过多，请稍后再试")
	}
	query := c.oauthClientRepo.Query(ctx).OauthClient
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ClientID.Eq(req.GetClientId())))
	var item *models.OauthClient
	item, err = c.oauthClientRepo.Find(ctx, opts...)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err = c.recordOauthClientFailure(ctx, req.GetClientId()); err != nil {
				return nil, errorsx.Internal("记录客户端认证失败状态失败").WithCause(err)
			}
			return nil, errorsx.Unauthenticated("客户端凭据无效")
		}
		return nil, errorsx.Internal("读取客户端凭据失败").WithCause(err)
	}
	if c.protector == nil {
		return nil, errorsx.Internal("OAuth 凭据保护器未初始化")
	}
	var clientSecret string
	clientSecret, err = c.protector.Unprotect(item.ClientSecret)
	if err != nil || subtle.ConstantTimeCompare([]byte(clientSecret), []byte(req.GetClientSecret())) != 1 {
		if err = c.recordOauthClientFailure(ctx, req.GetClientId()); err != nil {
			return nil, errorsx.Internal("记录客户端认证失败状态失败").WithCause(err)
		}
		return nil, errorsx.Unauthenticated("客户端凭据无效")
	}
	if !oauthClientIPAllowed(ctx, item.IPWhitelist) {
		return nil, errorsx.PermissionDenied("客户端来源 IP 未授权")
	}
	if err = c.clearOauthClientFailures(ctx, req.GetClientId()); err != nil {
		return nil, errorsx.Internal("清理客户端认证失败状态失败").WithCause(err)
	}
	if item.Status != coreconst.STATUS_STATUS_ENABLE {
		return nil, errorsx.PermissionDenied("客户端已停用")
	}
	var tenant *models.BaseTenant
	tenant, err = c.baseTenantRepo.FindByID(ctx, item.TenantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("客户端绑定租户不存在")
		}
		return nil, errorsx.Internal("读取客户端绑定租户失败").WithCause(err)
	}
	if tenant.Status != coreconst.STATUS_STATUS_ENABLE {
		return nil, errorsx.PermissionDenied("客户端绑定租户已停用")
	}
	authInfo := &authData.UserTokenPayload{
		TenantId:   item.TenantID,
		TenantCode: _const.OAuthClientTenantCode(tenant.Code, item.ClientID),
		// OAuth 客户端使用负数身份空间，避免与真实用户共用 uat_<id> Redis 键。
		UserId:   -item.ID,
		UserCode: item.ClientID,
		UserName: item.ClientName,
		RoleId:   -item.ID,
		RoleCode: item.ClientID,
		RoleName: item.ClientName,
		DeptName: "oauth-client",
	}
	var accessToken string
	accessToken, err = c.userToken.GenerateAccessToken(authInfo)
	if err != nil {
		return nil, errorsx.Internal("签发客户端访问令牌失败").WithCause(err)
	}
	return &basev1.IssueOauthClientTokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   c.userToken.GetAccessTokenExpires(),
	}, nil
}

// oauthClientFailureLocked 判断客户端和来源地址是否处于失败锁定窗口。
func (c *OauthClientTokenCase) oauthClientFailureLocked(ctx context.Context, clientID string) (bool, error) {
	value, err := c.Cache.Get(oauthClientFailureKey(clientID, oauthClientRequestIP(ctx)))
	if err == nil {
		var attempts int64
		attempts, err = strconv.ParseInt(value, 10, 64)
		if err != nil {
			return false, err
		}
		return attempts >= oauthClientFailureMaximum, nil
	}
	if isOauthCacheMiss(err) {
		return false, nil
	}
	return false, err
}

// recordOauthClientFailure 原子记录客户端认证失败次数。
func (c *OauthClientTokenCase) recordOauthClientFailure(ctx context.Context, clientID string) error {
	key := oauthClientFailureKey(clientID, oauthClientRequestIP(ctx))
	attempts, err := c.Cache.Incr(key)
	if err != nil {
		return err
	}
	if attempts == 1 {
		if err = c.Cache.Expire(key, oauthClientFailureWindow); err != nil {
			return err
		}
	}
	if attempts >= oauthClientFailureMaximum {
		return c.Cache.Expire(key, oauthClientFailureWindow)
	}
	return nil
}

// clearOauthClientFailures 清除客户端成功认证后的失败计数。
func (c *OauthClientTokenCase) clearOauthClientFailures(ctx context.Context, clientID string) error {
	return c.Cache.Del(oauthClientFailureKey(clientID, oauthClientRequestIP(ctx)))
}

// oauthClientFailureKey 生成客户端认证失败状态键。
func oauthClientFailureKey(clientID, clientIP string) string {
	digest := sha256.Sum256([]byte(clientID + "\x00" + clientIP))
	return oauthClientFailurePrefix + fmt.Sprintf("%x", digest[:])
}

// oauthClientRequestIP 读取当前请求的对端地址；无法识别请求时返回空地址。
func oauthClientRequestIP(ctx context.Context) string {
	value, ok := transport.FromServerContext(ctx)
	if !ok {
		return ""
	}
	httpValue, ok := value.(*httpTransport.Transport)
	if !ok || httpValue.Request() == nil {
		return ""
	}
	host, _, err := net.SplitHostPort(httpValue.Request().RemoteAddr)
	if err != nil {
		return httpValue.Request().RemoteAddr
	}
	return host
}

// oauthClientIPAllowed 判断客户端来源地址是否命中白名单。
func oauthClientIPAllowed(ctx context.Context, whitelist string) bool {
	if strings.TrimSpace(whitelist) == "" {
		return true
	}
	ip := net.ParseIP(oauthClientRequestIP(ctx))
	if ip == nil {
		return false
	}
	for _, raw := range strings.Split(whitelist, ",") {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			continue
		}
		if strings.Contains(entry, "/") {
			_, network, err := net.ParseCIDR(entry)
			if err == nil && network.Contains(ip) {
				return true
			}
			continue
		}
		if parsed := net.ParseIP(entry); parsed != nil && parsed.Equal(ip) {
			return true
		}
	}
	return false
}

// isOauthCacheMiss 判断客户端失败计数键不存在，而不是缓存服务不可用。
func isOauthCacheMiss(err error) bool {
	if err == nil || errors.Is(err, redis.Nil) {
		return err != nil
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "key not found") || strings.Contains(message, "key expired") || strings.Contains(message, "not found")
}
