package sse

import (
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/codegen"
	"github.com/liujitcn/kratos-core/errorsx"
	sseTransport "github.com/liujitcn/kratos-kit/transport/sse"
)

var _ sseTransport.SSEStream = (*Codegen)(nil)

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
