package biz

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/transport"
	httpTransport "github.com/go-kratos/kratos/v3/transport/http"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/sessionstate"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	authData "github.com/liujitcn/kratos-kit/auth/data"
)

// BaseSessionCase 提供当前登录会话查询和批量撤销能力。
// 当前实现以 UserToken 的用户级令牌槽为边界，查询时校验请求令牌，撤销时同时清理访问和刷新令牌。
type BaseSessionCase struct {
	*biz.BaseCase
	userToken *authData.UserToken
}

// NewBaseSessionCase 创建会话管理业务实例。
func NewBaseSessionCase(baseCase *biz.BaseCase, userToken *authData.UserToken) *BaseSessionCase {
	return &BaseSessionCase{BaseCase: baseCase, userToken: userToken}
}

// GetCurrentBaseSession 查询当前用户会话信息。
func (c *BaseSessionCase) GetCurrentBaseSession(ctx context.Context) (*adminv1.BaseSession, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	if c.userToken == nil || !c.userToken.IsExistAccessToken(authInfo.UserId) {
		return nil, errorsx.Unauthenticated("当前会话已失效")
	}
	if accessToken := sessionAccessToken(ctx); accessToken != "" && c.userToken.GetAccessToken(authInfo.UserId) != accessToken {
		return nil, errorsx.Unauthenticated("当前会话已失效")
	}
	clientIP, userAgent := sessionRequestMeta(ctx)
	var state sessionstate.State
	state, err = sessionstate.Read(c.Cache, authInfo.UserId)
	if err != nil {
		if !errors.Is(err, sessionstate.ErrStateNotFound) {
			return nil, errorsx.Internal("读取会话状态失败").WithCause(err)
		}
		state, err = sessionstate.Start(c.Cache, authInfo.UserId, clientIP, userAgent, time.Now())
		if err != nil {
			return nil, errorsx.Internal("创建会话状态失败").WithCause(err)
		}
	}
	if state.ClientIP != "" {
		clientIP = state.ClientIP
	}
	if state.Device != "" {
		userAgent = state.Device
	}
	issuedAt := ""
	if !state.TokenIssuedAt.IsZero() {
		issuedAt = state.TokenIssuedAt.Format(time.RFC3339)
	}
	return &adminv1.BaseSession{
		UserId:     authInfo.UserId,
		UserName:   authInfo.UserName,
		TenantCode: authInfo.TenantCode,
		ClientIp:   clientIP,
		Device:     userAgent,
		UserAgent:  userAgent,
		IssuedAt:   issuedAt,
		ExpiresIn:  c.userToken.GetAccessTokenExpires(),
	}, nil
}

// RevokeAllBaseSessions 撤销当前用户的全部访问和刷新令牌。
func (c *BaseSessionCase) RevokeAllBaseSessions(ctx context.Context) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	if c.userToken == nil {
		return errorsx.Internal("会话令牌管理器未配置")
	}
	if err = c.userToken.RemoveToken(authInfo.UserId); err != nil {
		return errorsx.Internal("撤销全部会话失败").WithCause(err)
	}
	if err = sessionstate.Clear(c.Cache, authInfo.UserId); err != nil {
		return errorsx.Internal("清理会话状态失败").WithCause(err)
	}
	return nil
}

// sessionRequestMeta 提取当前请求的对端地址和用户代理信息。
func sessionRequestMeta(ctx context.Context) (string, string) {
	info, ok := transport.FromServerContext(ctx)
	if !ok {
		return "", ""
	}
	httpInfo, ok := info.(*httpTransport.Transport)
	if !ok || httpInfo.Request() == nil {
		return "", ""
	}
	remoteAddr := httpInfo.Request().RemoteAddr
	clientIP, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		clientIP = remoteAddr
	}
	return clientIP, httpInfo.RequestHeader().Get("User-Agent")
}

// sessionAccessToken 从当前 HTTP 请求提取访问令牌，用于确认会话详情对应当前设备。
func sessionAccessToken(ctx context.Context) string {
	info, ok := transport.FromServerContext(ctx)
	if !ok {
		return ""
	}
	httpInfo, ok := info.(*httpTransport.Transport)
	if !ok || httpInfo.Request() == nil {
		return ""
	}
	parts := strings.Fields(httpInfo.RequestHeader().Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}
