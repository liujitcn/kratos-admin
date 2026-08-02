// Package base 注册 base.v1 传输层服务。
package base

import (
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	host "github.com/liujitcn/kratos-admin/backend/internal/server"
	baseService "github.com/liujitcn/kratos-admin/backend/internal/service/base"

	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/google/wire"
	mcpserver "github.com/liujitcn/kratos-kit/transport/mcp"
	"google.golang.org/grpc"
)

// Services 汇总 base.v1 的服务实现。
type Services struct {
	AiSession *baseService.AiSessionService
	AiTool    *baseService.AiToolService
	AiMessage *baseService.AiMessageService
	Config    *baseService.ConfigService
	Language  *baseService.LanguageService
	File      *baseService.FileService
	Login     *baseService.LoginService
	Oauth     *baseService.OauthService
	Mcp       *baseService.McpService
	Sse       *baseService.SseService
}

var _ host.Module = Services{}

// ProviderSet 汇总 base.v1 传输模块依赖注入提供者。
var ProviderSet = wire.NewSet(wire.Struct(new(Services), "*"))

// RegisterGRPC 注册 base.v1 的 gRPC 服务。
func (s Services) RegisterGRPC(srv grpc.ServiceRegistrar) {
	basev1.RegisterAiSessionServiceServer(srv, s.AiSession)
	basev1.RegisterAiToolServiceServer(srv, s.AiTool)
	basev1.RegisterAiMessageServiceServer(srv, s.AiMessage)
	basev1.RegisterConfigServiceServer(srv, s.Config)
	basev1.RegisterLanguageServiceServer(srv, s.Language)
	basev1.RegisterFileServiceServer(srv, s.File)
	basev1.RegisterLoginServiceServer(srv, s.Login)
	basev1.RegisterOauthServiceServer(srv, s.Oauth)
	basev1.RegisterMcpServiceServer(srv, s.Mcp)
	basev1.RegisterSseServiceServer(srv, s.Sse)
}

// RegisterHTTP 注册 base.v1 的 HTTP 服务。
func (s Services) RegisterHTTP(srv *kratosHTTP.Server) {
	basev1.RegisterAiSessionServiceHTTPServer(srv, s.AiSession)
	basev1.RegisterAiToolServiceHTTPServer(srv, s.AiTool)
	// AI 助手消息发送使用直连 SSE，避免占用工作台共用 /events 流。
	baseService.RegisterAiMessageServiceHTTPServer(srv, s.AiMessage)
	basev1.RegisterConfigServiceHTTPServer(srv, s.Config)
	basev1.RegisterLanguageServiceHTTPServer(srv, s.Language)
	// 文件上传需要兼容 uni.uploadFile 的 multipart/form-data 请求，使用自定义 HTTP 适配器。
	baseService.RegisterFileServiceHTTPServer(srv, s.File)
	basev1.RegisterLoginServiceHTTPServer(srv, s.Login)
	basev1.RegisterOauthServiceHTTPServer(srv, s.Oauth)
	// MCP 需要保留 Streamable HTTP 的原始请求体和流式响应，使用自定义 HTTP 适配器。
	baseService.RegisterMcpServiceHTTPServer(srv, s.Mcp)
	// SSE 需要直接写入事件流响应，使用自定义 HTTP 适配器避免默认 JSON 响应。
	baseService.RegisterSseServiceHTTPServer(srv, s.Sse)
}

// RegisterMCP 注册 base.v1 的 MCP 工具。
func (s Services) RegisterMCP(server *mcpserver.Server) {
	mcpSrv := server.MCPServer()
	basev1.RegisterAiSessionServiceMCPTools(mcpSrv, s.AiSession)
	basev1.RegisterAiToolServiceMCPTools(mcpSrv, s.AiTool)
	basev1.RegisterAiMessageServiceMCPTools(mcpSrv, s.AiMessage)
	basev1.RegisterConfigServiceMCPTools(mcpSrv, s.Config)
	basev1.RegisterLanguageServiceMCPTools(mcpSrv, s.Language)
	basev1.RegisterFileServiceMCPTools(mcpSrv, s.File)
	basev1.RegisterLoginServiceMCPTools(mcpSrv, s.Login)
}
