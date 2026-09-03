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
	refreshTokenCookieName       = "kratos_refresh_token"
	refreshExpiryCookieName      = "kratos_refresh_exp"
	refreshTokenCookiePath       = "/api/v1/base/token"
	legacyRefreshTokenCookiePath = "/api/v1/base"
	refreshTokenTransportHeader  = "X-Refresh-Token-Transport"
	refreshTokenTransportCookie  = "cookie"
)

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

// clearRefreshTokenCookie 清除刷新令牌 Cookie。
func clearRefreshTokenCookie(ctx context.Context) {
	secure := requestUsesTLS(ctx)
	httpTransport.SetCookie(ctx, &http.Cookie{Name: refreshTokenCookieName, Value: "", Path: refreshTokenCookiePath, MaxAge: -1, Expires: time.Unix(1, 0), HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
	// 清理旧版本使用的父路径 Cookie，避免浏览器保留旧令牌并在刷新时恢复登录态。
	httpTransport.SetCookie(ctx, &http.Cookie{Name: refreshTokenCookieName, Value: "", Path: legacyRefreshTokenCookiePath, MaxAge: -1, Expires: time.Unix(1, 0), HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
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

// requestUsesTLS 判断当前请求是否使用 HTTPS 或由 HTTPS 代理转发。
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
	return request.TLS != nil || strings.EqualFold(request.Header.Get("X-Forwarded-Proto"), "https")
}
