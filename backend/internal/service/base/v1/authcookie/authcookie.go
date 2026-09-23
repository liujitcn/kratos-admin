package authcookie

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/transport"
	httpTransport "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/liujitcn/kratos-admin/backend/internal/utils"
)

const (
	accessTokenCookieName       = "kratos_access_token"
	refreshTokenCookieName      = "kratos_refresh_token"
	refreshExpiryCookieName     = "kratos_refresh_exp"
	refreshTokenCookiePath      = "/api/v1/base/token"
	refreshTokenTransportHeader = "X-Refresh-Token-Transport"
	refreshTokenTransportCookie = "cookie"
)

// Set 同步写入访问令牌和刷新令牌 Cookie。
func Set(ctx context.Context, accessToken string, accessExpiresIn int64, refreshToken string, refreshExpiresIn int64) {
	setAccessToken(ctx, accessToken, accessExpiresIn)
	setRefreshToken(ctx, refreshToken, refreshExpiresIn)
}

// setAccessToken 写入供原生 src 静态资源请求使用的访问令牌 HttpOnly Cookie。
func setAccessToken(ctx context.Context, token string, expiresIn int64) {
	if token == "" || expiresIn <= 0 {
		return
	}
	secure := requestUsesTLS(ctx)
	httpTransport.SetCookie(ctx, &http.Cookie{
		Name:     accessTokenCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(expiresIn),
		Expires:  time.Now().Add(time.Duration(expiresIn) * time.Second),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// setRefreshToken 写入刷新令牌 HttpOnly Cookie 和非敏感过期时间提示 Cookie。
func setRefreshToken(ctx context.Context, token string, expiresIn int64) {
	if token == "" || expiresIn <= 0 {
		return
	}
	secure := requestUsesTLS(ctx)
	httpTransport.SetCookie(ctx, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    token,
		Path:     refreshTokenCookiePath,
		MaxAge:   int(expiresIn),
		Expires:  time.Now().Add(time.Duration(expiresIn) * time.Second),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
	httpTransport.SetCookie(ctx, &http.Cookie{
		Name:     refreshExpiryCookieName,
		Value:    strconv.FormatInt(time.Now().Add(time.Duration(expiresIn)*time.Second).Unix(), 10),
		Path:     "/",
		MaxAge:   int(expiresIn),
		Expires:  time.Now().Add(time.Duration(expiresIn) * time.Second),
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// HideRefreshTokenFromResponse 判断当前管理端浏览器是否只允许通过 HttpOnly Cookie 接收刷新令牌。
func HideRefreshTokenFromResponse(ctx context.Context) bool {
	request := requestFromContext(ctx)
	if request == nil {
		return false
	}
	return strings.EqualFold(request.Header.Get(refreshTokenTransportHeader), refreshTokenTransportCookie)
}

// Clear 清除访问令牌、刷新令牌和过期提示 Cookie。
func Clear(ctx context.Context) {
	secure := requestUsesTLS(ctx)
	httpTransport.SetCookie(ctx, &http.Cookie{Name: accessTokenCookieName, Value: "", Path: "/", MaxAge: -1, Expires: time.Unix(1, 0), HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
	httpTransport.SetCookie(ctx, &http.Cookie{Name: refreshTokenCookieName, Value: "", Path: refreshTokenCookiePath, MaxAge: -1, Expires: time.Unix(1, 0), HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
	httpTransport.SetCookie(ctx, &http.Cookie{Name: refreshExpiryCookieName, Value: "", Path: "/", MaxAge: -1, Expires: time.Unix(1, 0), Secure: secure, SameSite: http.SameSiteLaxMode})
}

// RefreshToken 从当前 HTTP 请求提取刷新令牌。
func RefreshToken(ctx context.Context) string {
	request := requestFromContext(ctx)
	if request == nil {
		return ""
	}
	cookie, err := request.Cookie(refreshTokenCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// requestUsesTLS 判断当前请求是否应视为 HTTPS。
func requestUsesTLS(ctx context.Context) bool {
	return utils.IsSecureRequest(requestFromContext(ctx))
}

// requestFromContext 从 Kratos 服务上下文提取 HTTP 请求。
func requestFromContext(ctx context.Context) *http.Request {
	serverTransport, ok := transport.FromServerContext(ctx)
	if !ok {
		return nil
	}
	httpServerTransport, ok := serverTransport.(*httpTransport.Transport)
	if !ok {
		return nil
	}
	return httpServerTransport.Request()
}
