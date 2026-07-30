package base

import (
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/rpc"
	sseserver "github.com/liujitcn/kratos-kit/transport/sse"
)

// NewSSEHandler 创建进程内 SSE 服务。
func NewSSEHandler(ctx *bootstrap.Context) (*sseserver.Server, error) {
	return rpc.CreateSseHandler(ctx.GetConfig())
}
