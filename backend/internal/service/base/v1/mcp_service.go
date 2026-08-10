package base

import (
	"context"
	"fmt"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-core/pkg/errorsx"

	"github.com/go-kratos/kratos/v3/log"
	mcpserver "github.com/liujitcn/kratos-kit/transport/mcp"
	"google.golang.org/protobuf/types/known/emptypb"
)

// McpService Base MCP 服务。
type McpService struct {
	basev1.UnimplementedMcpServiceServer
	mcpCase *biz.McpCase
}

// NewMcpService 创建 Base MCP 服务。
func NewMcpService(
	mcpCase *biz.McpCase,
) *McpService {
	var ss = McpService{
		mcpCase: mcpCase,
	}
	return &ss
}

// RegisterMCP 将 MCP 请求处理器绑定到宿主服务。
func (s *McpService) RegisterMCP(server *mcpserver.Server) {
	if s == nil || s.mcpCase == nil {
		return
	}
	s.mcpCase.RegisterMCP(server)
}

// HandleMcp 处理 MCP Streamable HTTP 请求。
func (s *McpService) HandleMcp(ctx context.Context, req *basev1.HandleMcpRequest) (*emptypb.Empty, error) {
	res, err := s.mcpCase.HandleMcp(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("HandleMcp %v", err))
		return nil, errorsx.WrapInternal(err, "处理MCP请求失败")
	}
	return res, nil
}
