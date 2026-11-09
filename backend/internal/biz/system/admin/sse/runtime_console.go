package sse

import (
	"context"

	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	authdata "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/database/gorm"
	"github.com/liujitcn/kratos-kit/transport/sse"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/logstream"
)

var _ sse.SSEStream = (*RuntimeConsole)(nil)

// RuntimeConsole 描述用户隔离的实时控制台 SSE 流。
type RuntimeConsole struct {
	hub *logstream.Hub
}

// NewRuntimeConsole 创建实时控制台 SSE 流。
func NewRuntimeConsole(hub *logstream.Hub) *RuntimeConsole {
	return &RuntimeConsole{hub: hub}
}

// ID 返回实时控制台 SSE 流标识。
func (*RuntimeConsole) ID() string {
	return logstream.SSEStreamRuntimeConsole
}

// ResolveContext 在可信身份下校验订阅权限，保留原有频道与任务归属校验。
func (r *RuntimeConsole) ResolveContext(_ context.Context, channelID string, identity *authdata.UserTokenPayload) (string, error) {
	if identity.TenantCode != gorm.DefaultTenantCode || (identity.RoleCode != _const.BASE_ROLE_CODE_SUPER && identity.RoleCode != _const.BASE_ROLE_CODE_ADMIN) {
		return "", errorsx.PermissionDenied("只有平台管理员可以订阅实时控制台")
	}
	return r.Resolve(channelID, identity.UserId)
}

// Resolve 校验频道归属后返回隔离的底层传输流标识。
func (r *RuntimeConsole) Resolve(channelID string, userID int64) (string, error) {
	if channelID == "" {
		return "", errorsx.InvalidArgument("实时控制台频道不能为空")
	}
	if r == nil || r.hub == nil || !r.hub.IsSessionOwner(channelID, userID) {
		return "", errorsx.PermissionDenied("无权订阅实时控制台频道")
	}
	return logstream.SSEStreamRuntimeConsole + ":" + channelID, nil
}

// Stop 清理实时控制台发布能力和内存会话。
func (r *RuntimeConsole) Stop() {
	if r == nil || r.hub == nil {
		return
	}
	r.hub.ClearPublisher()
}
