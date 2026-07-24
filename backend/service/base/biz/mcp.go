package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	commonv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-admin/backend/pkg/errorsx"
	"github.com/liujitcn/kratos-admin/backend/pkg/gen/data"
	"github.com/liujitcn/kratos-admin/backend/pkg/gen/models"

	"github.com/go-kratos/kratos/v3/log"
	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/bootstrap"
	mcpserver "github.com/liujitcn/kratos-kit/transport/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	mcpDefaultEndpointPath = "/mcp"
	mcpMethodCallTool      = "tools/call"
	mcpMethodListTools     = "tools/list"
	mcpTerminalHeader      = "X-Admin-Mcp-Terminal"
)

type mcpTerminalContextKey struct{}

// McpCase 处理 MCP 公共业务。
type McpCase struct {
	http.Handler

	baseAPIRepo *data.BaseAPIRepository
	handlerPath string
}

// NewMcpCase 创建 MCP 业务实例，并挂载工具过滤中间件。
func NewMcpCase(ctx *bootstrap.Context, baseAPIRepo *data.BaseAPIRepository, mcpSrv *mcpserver.Server) (*McpCase, error) {
	h := &McpCase{
		baseAPIRepo: baseAPIRepo,
	}
	cfg := ctx.GetConfig()
	// 未启用 HTTP 服务时，不创建 MCP HTTP 处理器。
	if cfg == nil || cfg.Server == nil || cfg.Server.Http == nil {
		return h, nil
	}

	h.handlerPath = mcpDefaultEndpointPath
	if cfg.Server.Mcp != nil && cfg.Server.Mcp.GetPath() != "" {
		h.handlerPath = cfg.Server.Mcp.GetPath()
	}

	mcpSrv.MCPServer().AddReceivingMiddleware(h.filterToolsMiddleware)
	handler, err := mcpSrv.HTTPHandler()
	if err != nil {
		return nil, err
	}
	h.Handler = handler
	if h.Handler == nil {
		return nil, nil
	}
	return h, nil
}

// HandleMcp 处理 MCP Streamable HTTP 请求。
func (h *McpCase) HandleMcp(ctx context.Context, req *basev1.HandleMcpRequest) (*emptypb.Empty, error) {
	if h == nil || h.Handler == nil {
		return nil, errorsx.Internal("MCP服务未初始化")
	}
	terminal := req.GetTerminal()
	w, ok := kratosHTTP.ResponseWriterFromServerContext(ctx)
	if !ok || w == nil {
		return nil, errorsx.InvalidArgument("MCP请求仅支持HTTP访问")
	}
	var r *http.Request
	r, ok = kratosHTTP.RequestFromServerContext(ctx)
	if !ok || r == nil {
		return nil, errorsx.InvalidArgument("MCP请求仅支持HTTP访问")
	}
	clonedRequest := r.Clone(context.WithValue(r.Context(), mcpTerminalContextKey{}, terminal))
	clonedRequest.Header = r.Header.Clone()
	clonedRequest.Header.Set(mcpTerminalHeader, terminal)
	urlCopy := *r.URL
	urlCopy.Path = h.handlerPath
	urlCopy.RawPath = ""
	clonedRequest.URL = &urlCopy
	h.Handler.ServeHTTP(w, clonedRequest)
	return &emptypb.Empty{}, nil
}

// filterToolsMiddleware 在官方 MCP SDK 接收链路中校验工具调用权限。
func (h *McpCase) filterToolsMiddleware(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		if method == mcpMethodCallTool {
			return h.filterToolCall(ctx, req, next)
		}
		if method == mcpMethodListTools {
			return h.filterToolList(ctx, req, next)
		}
		return next(ctx, method, req)
	}
}

// filterToolList 按当前服务关键字返回已注册工具。
func (h *McpCase) filterToolList(ctx context.Context, req mcp.Request, next mcp.MethodHandler) (mcp.Result, error) {
	result, err := next(ctx, mcpMethodListTools, req)
	if err != nil {
		return nil, err
	}
	listResult, ok := result.(*mcp.ListToolsResult)
	if !ok || listResult == nil {
		return result, nil
	}
	terminal := mcpTerminal(ctx, req)
	tools := h.filterMcpToolsByTerminal(terminal, listResult.Tools)
	listResult.Tools = tools
	err = h.applyMcpToolPrompts(ctx, tools)
	if err != nil {
		log.Error(fmt.Sprintf("查询 MCP 工具提示词失败 err=%v", err))
		return listResult, nil
	}
	return listResult, nil
}

// filterMcpToolsByTerminal 按当前服务关键字筛选 MCP 工具。
func (h *McpCase) filterMcpToolsByTerminal(terminal string, values []*mcp.Tool) []*mcp.Tool {
	tools := make([]*mcp.Tool, 0, len(values))
	for _, tool := range values {
		if tool == nil {
			continue
		}
		if !matchMcpToolPrefix(terminal, tool.Name) {
			continue
		}
		toolCopy := *tool
		tools = append(tools, &toolCopy)
	}
	return tools
}

// applyMcpToolPrompts 使用 base_api.tool_prompts 覆盖 MCP 工具描述。
func (h *McpCase) applyMcpToolPrompts(ctx context.Context, tools []*mcp.Tool) error {
	if h == nil || h.baseAPIRepo == nil || len(tools) == 0 {
		return nil
	}
	toolNames := make([]string, 0, len(tools))
	for _, tool := range tools {
		if tool == nil || tool.Name == "" {
			continue
		}
		toolNames = append(toolNames, tool.Name)
	}
	if len(toolNames) == 0 {
		return nil
	}
	query := h.baseAPIRepo.Query(ctx).BaseAPI
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ToolName.In(toolNames...)))
	list, err := h.baseAPIRepo.List(ctx, opts...)
	if err != nil {
		return err
	}
	promptDescByName := make(map[string]string, len(list))
	for _, item := range list {
		if item.ToolName == "" || promptDescByName[item.ToolName] != "" {
			continue
		}
		promptDescByName[item.ToolName] = toolPromptsDescription(item.ToolPrompts)
	}
	for _, tool := range tools {
		if tool == nil || promptDescByName[tool.Name] == "" {
			continue
		}
		tool.Description = promptDescByName[tool.Name]
	}
	return nil
}

// filterToolCall 拦截未启用或不属于当前终端的工具调用。
func (h *McpCase) filterToolCall(ctx context.Context, req mcp.Request, next mcp.MethodHandler) (mcp.Result, error) {
	callReq, ok := req.(*mcp.CallToolRequest)
	if !ok || callReq == nil || callReq.Params == nil {
		return next(ctx, mcpMethodCallTool, req)
	}
	baseAPI, err := h.findEnabledBaseAPI(ctx, req, callReq.Params.Name)
	if err != nil {
		log.Error(fmt.Sprintf("查询 MCP 工具状态失败 err=%v", err))
		return newMcpToolResultError(fmt.Errorf("查询 MCP 工具状态失败: %w", err).Error()), nil
	}
	if baseAPI == nil {
		return newMcpToolResultError(fmt.Sprintf("MCP 工具 %s 未注册", callReq.Params.Name)), nil
	}
	return next(ctx, mcpMethodCallTool, req)
}

// findEnabledBaseAPI 查询当前工具是否允许调用。
func (h *McpCase) findEnabledBaseAPI(ctx context.Context, req mcp.Request, toolName string) (*models.BaseAPI, error) {
	terminal := mcpTerminal(ctx, req)
	if !matchMcpToolPrefix(terminal, toolName) {
		return nil, nil
	}
	query := h.baseAPIRepo.Query(ctx).BaseAPI
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.McpStatus.Eq(int32(commonv1.Status_ENABLE))))
	opts = append(opts, repository.Where(query.ToolName.Eq(toolName)))
	opts = append(opts, repository.Limit(1))
	list, err := h.baseAPIRepo.List(ctx, opts...)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return list[0], nil
}

// toolPromptsDescription 将多条工具提示词合并为运行时工具描述。
func toolPromptsDescription(value string) string {
	if value == "" {
		return ""
	}
	var prompts []string
	err := json.Unmarshal([]byte(value), &prompts)
	if err != nil {
		return ""
	}
	values := make([]string, 0, len(prompts))
	for _, item := range prompts {
		if item == "" {
			continue
		}
		values = append(values, item)
	}
	return strings.Join(values, "\n")
}

// matchMcpToolPrefix 判断工具名是否匹配当前服务前缀。
func matchMcpToolPrefix(terminal, toolName string) bool {
	return toolName != "" && (terminal == "" || strings.HasPrefix(toolName, terminal+"_") || strings.HasPrefix(toolName, "base_"))
}

// mcpTerminal 获取当前 MCP 请求的服务筛选关键字。
func mcpTerminal(ctx context.Context, req mcp.Request) string {
	if req != nil && req.GetExtra() != nil {
		return req.GetExtra().Header.Get(mcpTerminalHeader)
	}
	terminal, _ := ctx.Value(mcpTerminalContextKey{}).(string)
	return terminal
}

// newMcpToolResultError 构造 MCP 工具级错误结果，避免返回协议级错误。
func newMcpToolResultError(message string) *mcp.CallToolResult {
	result := &mcp.CallToolResult{}
	result.SetError(errors.New(message))
	return result
}
