package base

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/transport"
	httpTransport "github.com/go-kratos/kratos/v3/transport/http"
)

const (
	accessTokenCookieName       = "kratos_access_token"
	refreshTokenCookieName      = "kratos_refresh_token"
	refreshExpiryCookieName     = "kratos_refresh_exp"
	refreshTokenCookiePath      = "/api/v1/base/token"
	refreshTokenTransportHeader = "X-Refresh-Token-Transport"
	refreshTokenTransportCookie = "cookie"
)

// setAccessTokenCookie 写入供原生 src 静态资源请求使用的访问令牌 HttpOnly Cookie。
func setAccessTokenCookie(ctx context.Context, token string, expiresIn int64) {
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

// setAuthTokenCookies 同步写入访问令牌和刷新令牌 Cookie。
func setAuthTokenCookies(ctx context.Context, accessToken string, accessExpiresIn int64, refreshToken string, refreshExpiresIn int64) {
	setAccessTokenCookie(ctx, accessToken, accessExpiresIn)
	setRefreshTokenCookie(ctx, refreshToken, refreshExpiresIn)
}

// setRefreshTokenCookie 写入刷新令牌 HttpOnly Cookie 和非敏感过期时间提示 Cookie。
func setRefreshTokenCookie(ctx context.Context, token string, expiresIn int64) {
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

// hideRefreshTokenFromResponse 判断当前管理端浏览器是否只允许通过 HttpOnly Cookie 接收刷新令牌。
func hideRefreshTokenFromResponse(ctx context.Context) bool {
	serverTransport, ok := transport.FromServerContext(ctx)
	if !ok {
		return false
	}
	httpServerTransport, ok := serverTransport.(*httpTransport.Transport)
	if !ok || httpServerTransport.Request() == nil {
		return false
	}
	return strings.EqualFold(httpServerTransport.Request().Header.Get(refreshTokenTransportHeader), refreshTokenTransportCookie)
}

// clearAuthTokenCookies 清除访问令牌、刷新令牌和过期提示 Cookie。
func clearAuthTokenCookies(ctx context.Context) {
	secure := requestUsesTLS(ctx)
	httpTransport.SetCookie(ctx, &http.Cookie{Name: accessTokenCookieName, Value: "", Path: "/", MaxAge: -1, Expires: time.Unix(1, 0), HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
	httpTransport.SetCookie(ctx, &http.Cookie{Name: refreshTokenCookieName, Value: "", Path: refreshTokenCookiePath, MaxAge: -1, Expires: time.Unix(1, 0), HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
	httpTransport.SetCookie(ctx, &http.Cookie{Name: refreshExpiryCookieName, Value: "", Path: "/", MaxAge: -1, Expires: time.Unix(1, 0), Secure: secure, SameSite: http.SameSiteLaxMode})
}

// refreshTokenFromCookie 从当前 HTTP 请求提取刷新令牌。
func refreshTokenFromCookie(ctx context.Context) string {
	serverTransport, ok := transport.FromServerContext(ctx)
	if !ok {
		return ""
	}
	httpServerTransport, ok := serverTransport.(*httpTransport.Transport)
	if !ok || httpServerTransport.Request() == nil {
		return ""
	}
	cookie, err := httpServerTransport.Request().Cookie(refreshTokenCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// requestUsesTLS 判断当前请求是否直接使用 HTTPS。
func requestUsesTLS(ctx context.Context) bool {
	serverTransport, ok := transport.FromServerContext(ctx)
	if !ok {
		return false
	}
	httpServerTransport, ok := serverTransport.(*httpTransport.Transport)
	if !ok || httpServerTransport.Request() == nil {
		return false
	}
	request := httpServerTransport.Request()
	return request.TLS != nil
}
