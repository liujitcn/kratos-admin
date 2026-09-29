package rag

import (
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/rag/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/service/rag/admin/v1"

	"github.com/go-kratos/kratos/v3/transport/http"
	"github.com/liujitcn/kratos-kit/transport/mcp"
	"google.golang.org/grpc"
)

// Services 汇总 rag.admin.v1 的服务实现。
type Services struct {
	AiKnowledge      *admin.AiKnowledgeService
	AiKnowledgeDoc   *admin.AiKnowledgeDocService
	AiKnowledgeChunk *admin.AiKnowledgeChunkService
}

// RegisterGRPC 注册 rag.admin.v1 的 gRPC 服务。
func (s Services) RegisterGRPC(srv grpc.ServiceRegistrar) {
	adminv1.RegisterAiKnowledgeServiceServer(srv, adminv1.RedactedAiKnowledgeServiceServer(s.AiKnowledge))
	adminv1.RegisterAiKnowledgeDocServiceServer(srv, adminv1.RedactedAiKnowledgeDocServiceServer(s.AiKnowledgeDoc))
	adminv1.RegisterAiKnowledgeChunkServiceServer(srv, adminv1.RedactedAiKnowledgeChunkServiceServer(s.AiKnowledgeChunk))
}

// RegisterHTTP 注册 rag.admin.v1 的 HTTP 服务。
func (s Services) RegisterHTTP(srv *http.Server) {
	adminv1.RegisterAiKnowledgeServiceHTTPServer(srv, adminv1.RedactedAiKnowledgeServiceServer(s.AiKnowledge))
	adminv1.RegisterAiKnowledgeDocServiceHTTPServer(srv, adminv1.RedactedAiKnowledgeDocServiceServer(s.AiKnowledgeDoc))
	adminv1.RegisterAiKnowledgeChunkServiceHTTPServer(srv, adminv1.RedactedAiKnowledgeChunkServiceServer(s.AiKnowledgeChunk))
}

// RegisterMCP 注册 rag.admin.v1 的 MCP 工具。
func (s Services) RegisterMCP(server *mcp.Server) {
	mcpSrv := server.MCPServer()
	adminv1.RegisterAiKnowledgeServiceMCPTools(mcpSrv, s.AiKnowledge)
	adminv1.RegisterAiKnowledgeDocServiceMCPTools(mcpSrv, s.AiKnowledgeDoc)
	adminv1.RegisterAiKnowledgeChunkServiceMCPTools(mcpSrv, s.AiKnowledgeChunk)
}
