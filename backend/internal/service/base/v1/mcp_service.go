package base

import (
	"context"
	"fmt"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"

	"github.com/go-kratos/kratos/v3/log"
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

// HandleMcp 处理 MCP Streamable HTTP 请求。
func (s *McpService) HandleMcp(ctx context.Context, req *basev1.HandleMcpRequest) (*emptypb.Empty, error) {
	res, err := s.mcpCase.HandleMcp(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("HandleMcp %v", err))
		return nil, errorsx.WrapInternal(err, "处理MCP请求失败")
	}
	return res, nil
}
