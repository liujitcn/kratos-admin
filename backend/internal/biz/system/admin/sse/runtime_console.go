package sse

import (
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/logstream"
	"github.com/liujitcn/kratos-core/errorsx"
	sseTransport "github.com/liujitcn/kratos-kit/transport/sse"
)

var _ sseTransport.SSEStream = (*RuntimeConsole)(nil)

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
