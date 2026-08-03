package base

import (
	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/rpc"
	sseserver "github.com/liujitcn/kratos-kit/transport/sse"
	"google.golang.org/protobuf/proto"
)

// NewSSEHandler 创建供 Backend 模块内部 HTTP 路由使用的 SSE 服务。
func NewSSEHandler(ctx *bootstrap.Context) (*sseserver.Server, error) {
	cfg := ctx.GetConfig()
	if cfg == nil || cfg.Server == nil || cfg.Server.Sse == nil || cfg.Server.Sse.GetTransport() != bootstrapConfigv1.Server_Sse_HTTP {
		return rpc.CreateSseHandler(cfg)
	}
	moduleConfig := proto.Clone(cfg).(*bootstrapConfigv1.Bootstrap)
	moduleConfig.Server.Sse.Transport = bootstrapConfigv1.Server_Sse_IN_PROCESS
	return rpc.CreateSseHandler(moduleConfig)
}

// NewSSEServer 创建独立部署形态使用的 SSE 服务。
func NewSSEServer(ctx *bootstrap.Context) (*sseserver.Server, error) {
	cfg := ctx.GetConfig()
	if cfg != nil && cfg.Server != nil && cfg.Server.Sse != nil && cfg.Server.Sse.GetTransport() != bootstrapConfigv1.Server_Sse_IN_PROCESS {
		return rpc.CreateSseServer(cfg)
	}
	return NewSSEHandler(ctx)
}
