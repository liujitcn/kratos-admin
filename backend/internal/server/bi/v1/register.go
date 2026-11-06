package bi

import (
	"github.com/go-kratos/kratos/v3/transport/http"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	admin "github.com/liujitcn/kratos-admin/backend/internal/service/bi/admin/v1"
	"github.com/liujitcn/kratos-kit/transport/mcp"
	"google.golang.org/grpc"
)

// Services 汇总 bi.admin.v1 服务实现。
type Services struct {
	BiCard        *admin.BiCardService
	BiSource      *admin.BiSourceService
	BiQuery       *admin.BiQueryService
	BiQueryRecord *admin.BiQueryRecordService
	BiDashboard   *admin.BiDashboardService
}

// RegisterGRPC 注册 bi.admin.v1 gRPC 服务。
func (s Services) RegisterGRPC(server grpc.ServiceRegistrar) {
	adminv1.RegisterBiCardServiceServer(server, adminv1.RedactedBiCardServiceServer(s.BiCard))
	adminv1.RegisterBiSourceServiceServer(server, adminv1.RedactedBiSourceServiceServer(s.BiSource))
	adminv1.RegisterBiQueryServiceServer(server, adminv1.RedactedBiQueryServiceServer(s.BiQuery))
	adminv1.RegisterBiQueryRecordServiceServer(server, adminv1.RedactedBiQueryRecordServiceServer(s.BiQueryRecord))
	adminv1.RegisterBiDashboardServiceServer(server, adminv1.RedactedBiDashboardServiceServer(s.BiDashboard))
}

// RegisterHTTP 注册 bi.admin.v1 HTTP 服务。
func (s Services) RegisterHTTP(server *http.Server) {
	adminv1.RegisterBiCardServiceHTTPServer(server, adminv1.RedactedBiCardServiceServer(s.BiCard))
	adminv1.RegisterBiSourceServiceHTTPServer(server, adminv1.RedactedBiSourceServiceServer(s.BiSource))
	adminv1.RegisterBiQueryServiceHTTPServer(server, adminv1.RedactedBiQueryServiceServer(s.BiQuery))
	adminv1.RegisterBiQueryRecordServiceHTTPServer(server, adminv1.RedactedBiQueryRecordServiceServer(s.BiQueryRecord))
	adminv1.RegisterBiDashboardServiceHTTPServer(server, adminv1.RedactedBiDashboardServiceServer(s.BiDashboard))
}

// RegisterMCP 注册 bi.admin.v1 MCP 工具。
func (s Services) RegisterMCP(server *mcp.Server) {
	adminv1.RegisterBiCardServiceMCPTools(server.MCPServer(), s.BiCard)
	adminv1.RegisterBiSourceServiceMCPTools(server.MCPServer(), s.BiSource)
	adminv1.RegisterBiQueryServiceMCPTools(server.MCPServer(), s.BiQuery)
	adminv1.RegisterBiQueryRecordServiceMCPTools(server.MCPServer(), s.BiQueryRecord)
	adminv1.RegisterBiDashboardServiceMCPTools(server.MCPServer(), s.BiDashboard)
}
