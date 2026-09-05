package base

import (
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/service/base/v1"

	"github.com/go-kratos/kratos/v3/transport/http"
	"github.com/liujitcn/kratos-kit/transport/mcp"
	"google.golang.org/grpc"
)

// Services 汇总 base.v1 的服务实现。
type Services struct {
	AiSession    *base.AiSessionService
	AiTool       *base.AiToolService
	AiMessage    *base.AiMessageService
	Config       *base.ConfigService
	Language     *base.LanguageService
	File         *base.FileService
	Login        *base.LoginService
	Mfa          *base.MfaService
	Oauth        *base.OauthService
	OauthClient  *base.OauthClientService
	Mcp          *base.McpService
	Notification *base.NotificationService
	Sse          *base.SseService
}

// RegisterGRPC 注册 base.v1 的 gRPC 服务。
func (s Services) RegisterGRPC(srv grpc.ServiceRegistrar) {
	basev1.RegisterAiSessionServiceServer(srv, basev1.RedactedAiSessionServiceServer(s.AiSession))
	basev1.RegisterAiToolServiceServer(srv, basev1.RedactedAiToolServiceServer(s.AiTool))
	basev1.RegisterAiMessageServiceServer(srv, basev1.RedactedAiMessageServiceServer(s.AiMessage))
	basev1.RegisterConfigServiceServer(srv, basev1.RedactedConfigServiceServer(s.Config))
	basev1.RegisterLanguageServiceServer(srv, basev1.RedactedLanguageServiceServer(s.Language))
	basev1.RegisterFileServiceServer(srv, basev1.RedactedFileServiceServer(s.File))
	basev1.RegisterLoginServiceServer(srv, basev1.RedactedLoginServiceServer(s.Login))
	basev1.RegisterMfaServiceServer(srv, basev1.RedactedMfaServiceServer(s.Mfa))
	basev1.RegisterOauthServiceServer(srv, basev1.RedactedOauthServiceServer(s.Oauth))
	basev1.RegisterOauthClientServiceServer(srv, basev1.RedactedOauthClientServiceServer(s.OauthClient))
	basev1.RegisterMcpServiceServer(srv, basev1.RedactedMcpServiceServer(s.Mcp))
	basev1.RegisterNotificationServiceServer(srv, basev1.RedactedNotificationServiceServer(s.Notification))
	basev1.RegisterSseServiceServer(srv, basev1.RedactedSseServiceServer(s.Sse))
}

// RegisterHTTP 注册 base.v1 的 HTTP 服务。
func (s Services) RegisterHTTP(srv *http.Server) {
	basev1.RegisterAiSessionServiceHTTPServer(srv, basev1.RedactedAiSessionServiceServer(s.AiSession))
	basev1.RegisterAiToolServiceHTTPServer(srv, basev1.RedactedAiToolServiceServer(s.AiTool))
	// AI 助手消息发送使用直连 SSE，避免占用工作台共用 /events 流。
	base.RegisterAiMessageServiceHTTPServer(srv, s.AiMessage)
	basev1.RegisterConfigServiceHTTPServer(srv, basev1.RedactedConfigServiceServer(s.Config))
	basev1.RegisterLanguageServiceHTTPServer(srv, basev1.RedactedLanguageServiceServer(s.Language))
	// 文件上传需要兼容 uni.uploadFile 的 multipart/form-data 请求，使用自定义 HTTP 适配器。
	base.RegisterFileServiceHTTPServer(srv, s.File)
	basev1.RegisterLoginServiceHTTPServer(srv, basev1.RedactedLoginServiceServer(s.Login))
	basev1.RegisterMfaServiceHTTPServer(srv, basev1.RedactedMfaServiceServer(s.Mfa))
	basev1.RegisterOauthServiceHTTPServer(srv, basev1.RedactedOauthServiceServer(s.Oauth))
	basev1.RegisterOauthClientServiceHTTPServer(srv, basev1.RedactedOauthClientServiceServer(s.OauthClient))
	// MCP 需要保留 Streamable HTTP 的原始请求体和流式响应，使用自定义 HTTP 适配器。
	base.RegisterMcpServiceHTTPServer(srv, s.Mcp)
	basev1.RegisterNotificationServiceHTTPServer(srv, basev1.RedactedNotificationServiceServer(s.Notification))
	// SSE 订阅保留 Base 协议兼容路由，统一运行时由 Core SSE 服务承载。
	base.RegisterSseServiceHTTPServer(srv, s.Sse)
}

// RegisterMCP 注册 base.v1 的 MCP 工具。
func (s Services) RegisterMCP(server *mcp.Server) {
	mcpSrv := server.MCPServer()
	basev1.RegisterAiSessionServiceMCPTools(mcpSrv, s.AiSession)
	basev1.RegisterAiToolServiceMCPTools(mcpSrv, s.AiTool)
	basev1.RegisterAiMessageServiceMCPTools(mcpSrv, s.AiMessage)
	basev1.RegisterConfigServiceMCPTools(mcpSrv, s.Config)
	basev1.RegisterLanguageServiceMCPTools(mcpSrv, s.Language)
	basev1.RegisterFileServiceMCPTools(mcpSrv, s.File)
	basev1.RegisterLoginServiceMCPTools(mcpSrv, s.Login)
	basev1.RegisterMfaServiceMCPTools(mcpSrv, s.Mfa)
}
