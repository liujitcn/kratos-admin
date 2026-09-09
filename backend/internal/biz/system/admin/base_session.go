package biz

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/sessionregistry"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/sessionstate"
	"github.com/liujitcn/kratos-core/biz"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

// BaseSessionCase 提供当前登录会话查询和批量撤销能力。
type BaseSessionCase struct {
	*biz.BaseCase
	userToken *data.UserToken
}

// NewBaseSessionCase 创建会话管理业务实例。
func NewBaseSessionCase(baseCase *biz.BaseCase, userToken *data.UserToken) *BaseSessionCase {
	return &BaseSessionCase{BaseCase: baseCase, userToken: userToken}
}

// ListCurrentBaseSessions 只查询认证用户本人的有效会话并标记当前设备。
func (c *BaseSessionCase) ListCurrentBaseSessions(ctx context.Context) (*adminv1.ListCurrentBaseSessionsResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var sessions []data.TokenSession
	sessions, err = c.userToken.GetTokenSessions(authInfo.UserId)
	if err != nil {
		return nil, errorsx.Internal("查询本人会话失败").WithCause(err)
	}
	currentToken := sessionregistry.AccessToken(ctx)
	items := make([]*adminv1.BaseSession, 0, len(sessions))
	for _, session := range sessions {
		var record sessionregistry.Record
		record, err = sessionregistry.Get(c.Cache, session.SessionID)
		if err != nil {
			return nil, errorsx.Internal("读取本人会话失败").WithCause(err)
		}
		if !c.userToken.IsAccessTokenValid(authInfo.UserId, session.AccessToken) && !c.userToken.IsRefreshTokenValid(authInfo.UserId, session.RefreshToken) {
			continue
		}
		if authInfo.RoleCode != _const.BASE_ROLE_CODE_USER && authInfo.RoleCode != _const.BASE_ROLE_CODE_AUTHUSER {
			_, err = sessionstate.Validate(c.Cache, session.SessionID, time.Now())
			if errors.Is(err, sessionstate.ErrIdleExpired) || errors.Is(err, sessionstate.ErrMaxLifetimeExpired) || errors.Is(err, sessionstate.ErrStateNotFound) {
				continue
			}
			if err != nil {
				return nil, errorsx.Internal("校验本人会话失败").WithCause(err)
			}
		}
		item := c.toBaseSession(record, session.AccessToken == currentToken)
		item.ExpiresIn = max(0, int64(time.Until(time.Unix(0, session.AccessExpiresAt)).Seconds()))
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Current != items[j].Current {
			return items[i].Current
		}
		return items[i].IssuedAt > items[j].IssuedAt
	})
	return &adminv1.ListCurrentBaseSessionsResponse{Sessions: items}, nil
}

// GetCurrentBaseSession 查询当前用户会话信息。
func (c *BaseSessionCase) GetCurrentBaseSession(ctx context.Context) (*adminv1.BaseSession, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var record sessionregistry.Record
	record, err = sessionregistry.FindByAccessToken(c.Cache, c.userToken, authInfo.UserId, sessionregistry.AccessToken(ctx))
	if err != nil {
		return nil, errorsx.Unauthenticated("当前会话已失效").WithCause(err)
	}
	return c.toBaseSession(record, true), nil
}

// PageOnlineBaseSessions 分页查询当前在线用户会话。
func (c *BaseSessionCase) PageOnlineBaseSessions(ctx context.Context, req *adminv1.PageOnlineBaseSessionsRequest) (*adminv1.PageOnlineBaseSessionsResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	if authInfo.RoleCode != _const.BASE_ROLE_CODE_SUPER {
		return nil, errorsx.PermissionDenied("只有平台管理员可以查看在线用户")
	}
	var records []sessionregistry.Record
	records, err = sessionregistry.List(c.Cache, c.userToken)
	if err != nil {
		return nil, errorsx.Internal("查询在线用户失败").WithCause(err)
	}
	tenantCode := req.GetTenantCode()
	if authInfo.TenantCode != gorm.DefaultTenantCode {
		tenantCode = authInfo.TenantCode
	}
	keyword := strings.ToLower(req.GetKeyword())
	filtered := make([]sessionregistry.Record, 0, len(records))
	for _, record := range records {
		if tenantCode != "" && record.TenantCode != tenantCode {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(record.UserName+" "+record.TenantCode+" "+record.ClientIP+" "+record.Device), keyword) {
			continue
		}
		filtered = append(filtered, record)
	}
	pageNum := req.GetPageNum()
	pageSize := req.GetPageSize()
	if pageNum <= 0 {
		pageNum = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	start := int((pageNum - 1) * pageSize)
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + int(pageSize)
	if end > len(filtered) {
		end = len(filtered)
	}
	items := make([]*adminv1.BaseSession, 0, end-start)
	for _, record := range filtered[start:end] {
		items = append(items, c.toBaseSession(record, false))
	}
	return &adminv1.PageOnlineBaseSessionsResponse{Sessions: items, Total: int32(len(filtered))}, nil
}

// RevokeBaseSession 下线指定用户会话。
func (c *BaseSessionCase) RevokeBaseSession(ctx context.Context, req *adminv1.RevokeBaseSessionRequest) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	if authInfo.RoleCode != _const.BASE_ROLE_CODE_SUPER {
		return errorsx.PermissionDenied("只有平台管理员可以下线在线用户")
	}
	var record sessionregistry.Record
	record, err = sessionregistry.Get(c.Cache, req.GetSessionId())
	if err != nil {
		return errorsx.ResourceNotFound("在线会话不存在").WithCause(err)
	}
	if err = sessionregistry.Remove(c.Cache, c.userToken, record); err != nil {
		return errorsx.Internal("下线在线用户失败").WithCause(err)
	}

	return nil
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
	if err = sessionregistry.RemoveAll(c.Cache, c.userToken, authInfo.UserId); err != nil {
		return errorsx.Internal("撤销全部会话失败").WithCause(err)
	}

	return nil
}

// toBaseSession 转换在线会话的展示信息。
func (c *BaseSessionCase) toBaseSession(record sessionregistry.Record, current bool) *adminv1.BaseSession {
	return &adminv1.BaseSession{SessionId: record.SessionID, UserId: record.UserID, UserName: record.UserName, TenantCode: record.TenantCode, ClientIp: record.ClientIP, Device: record.Device, UserAgent: record.UserAgent, IssuedAt: record.IssuedAt.Format(time.RFC3339), ExpiresIn: c.userToken.GetAccessTokenExpires(), Current: current}
}
