package sse

import (
	"context"

	"github.com/liujitcn/kratos-core/errorsx"
	authdata "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/transport/sse"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/codegen"
)

var _ sse.SSEStream = (*Codegen)(nil)

// Codegen 描述代码生成任务的 SSE 订阅流。
type Codegen struct {
	manager *codegen.Manager
}

// NewCodegen 创建代码生成任务 SSE 流。
func NewCodegen(manager *codegen.Manager) *Codegen {
	return &Codegen{manager: manager}
}

// ID 返回代码生成 SSE 流标识。
func (c *Codegen) ID() string {
	return codegen.SSEStreamCodeGen
}

// ResolveContext 在可信身份下校验订阅权限，保留原有频道与任务归属校验。
func (c *Codegen) ResolveContext(_ context.Context, channelID string, identity *authdata.UserTokenPayload) (string, error) {
	return c.Resolve(channelID, identity.UserId)
}

// Resolve 校验任务归属后返回隔离的传输流标识。
func (c *Codegen) Resolve(channelID string, userID int64) (string, error) {
	if channelID == "" {
		return "", errorsx.InvalidArgument("代码生成任务ID不能为空")
	}
	if c.manager == nil || !c.manager.IsOwner(channelID, userID) {
		return "", errorsx.PermissionDenied("无权订阅代码生成任务")
	}
	return codegen.StreamID(channelID), nil
}

// SetPublisher 设置代码生成任务的 SSE 发布能力。
func (c *Codegen) SetPublisher(publisher codegen.Publisher) {
	if c == nil || c.manager == nil {
		return
	}
	c.manager.SetPublisher(publisher)
}
