package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/agent"
	"github.com/liujitcn/kratos-core/biz"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-core/resource/i18n"
	"github.com/liujitcn/kratos-kit/auth"
	"github.com/liujitcn/kratos-kit/auth/authz/engine"

	kratosErrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/log"
	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/liujitcn/gorm-kit/repository"
	mcpserver "github.com/liujitcn/kratos-kit/transport/mcp"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	mcpDefaultEndpointPath = "/mcp"
	mcpTerminalHeader      = "X-Admin-Mcp-Terminal"
)

type mcpTerminalContextKey struct{}

// McpCase 处理 MCP 公共业务。
type McpCase struct {
	*biz.BaseCase
	http.Handler

	baseAPIRepo *data.BaseAPIRepository
	authorizer  engine.Engine
	catalog     *i18n.I18n
	handlerPath string
}

// NewMcpCase 创建 MCP 业务实例。
func NewMcpCase(baseCase *biz.BaseCase, baseAPIRepo *data.BaseAPIRepository, authorizer engine.Engine, catalog *i18n.I18n) (*McpCase, error) {
	h := &McpCase{
		BaseCase:    baseCase,
		baseAPIRepo: baseAPIRepo,
		authorizer:  authorizer,
		catalog:     catalog,
	}
	cfg := baseCase.GetConfig()
	// 未启用 HTTP 服务时，不创建 MCP HTTP 处理器。
	if cfg == nil || cfg.Server == nil || cfg.Server.Http == nil {
		return h, nil
	}

	h.handlerPath = mcpDefaultEndpointPath
	if cfg.Server.Mcp != nil && cfg.Server.Mcp.GetPath() != "" {
		h.handlerPath = cfg.Server.Mcp.GetPath()
	}
	return h, nil
}

// RegisterMCP 将 MCP 过滤器和 HTTP 处理器绑定到宿主服务。
func (h *McpCase) RegisterMCP(mcpSrv *mcpserver.Server) {
	if h == nil || mcpSrv == nil {
		return
	}
	hooks := mcpSrv.MCPServer().GetHooks()
	if hooks == nil {
		hooks = &server.Hooks{}
		server.WithHooks(hooks)(mcpSrv.MCPServer())
	}
	hooks.AddAfterListTools(h.filterToolList)
	mcpSrv.MCPServer().Use(h.filterToolCall)
	handler, err := mcpSrv.HTTPHandler()
	if err != nil {
		panic(fmt.Sprintf("创建 MCP HTTP 处理器失败: %v", err))
	}
	h.Handler = handler
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
	// 保留 Kratos 中间件已注入的认证、租户和语言信息。
	ctx = agent.ContextWithTerminal(ctx, terminal)
	clonedRequest := r.Clone(context.WithValue(ctx, mcpTerminalContextKey{}, terminal))
	clonedRequest.Header = r.Header.Clone()
	clonedRequest.Header.Set(mcpTerminalHeader, terminal)
	urlCopy := *r.URL
	urlCopy.Path = h.handlerPath
	urlCopy.RawPath = ""
	clonedRequest.URL = &urlCopy
	h.Handler.ServeHTTP(w, clonedRequest)
	return &emptypb.Empty{}, nil
}

// ToolConfigs 查询当前终端允许暴露给 Agent 的工具配置。
func (h *McpCase) ToolConfigs(ctx context.Context, terminal string, names []string) (map[string]ai.ToolConfig, error) {
	result := make(map[string]ai.ToolConfig)
	if len(names) == 0 {
		return result, nil
	}
	filteredNames := make([]string, 0, len(names))
	for _, name := range names {
		if matchToolTerminal(terminal, name) {
			filteredNames = append(filteredNames, name)
		}
	}
	if len(filteredNames) == 0 {
		return result, nil
	}
	query := h.baseAPIRepo.Query(ctx).BaseAPI
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ToolName.In(filteredNames...)))
	list, err := h.baseAPIRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	totalByName := make(map[string]int, len(filteredNames))
	enabledByName := make(map[string]int, len(filteredNames))
	promptsByName := make(map[string][]string, len(filteredNames))
	for _, item := range list {
		totalByName[item.ToolName]++
		if !agentToolBlocked(item) && item.AgentStatus == _const.STATUS_STATUS_ENABLE && (terminal != "app" || agentToolReadOnly(item)) {
			err = h.authorizeToolCall(ctx, item)
			if err != nil {
				if kratosErrors.FromError(err).Code == http.StatusForbidden || kratosErrors.FromError(err).Code == http.StatusUnauthorized {
					continue
				}
				return nil, err
			}
			enabledByName[item.ToolName]++
		}
		if len(promptsByName[item.ToolName]) == 0 {
			promptsByName[item.ToolName] = parseToolPrompts(item.ToolPrompts)
		}
	}
	for _, name := range filteredNames {
		if totalByName[name] > 0 {
			result[name] = ai.ToolConfig{Enabled: totalByName[name] == enabledByName[name], Prompts: promptsByName[name]}
		}
	}
	return result, nil
}

// filterToolList 按当前服务关键字返回已注册工具。
func (h *McpCase) filterToolList(ctx context.Context, _ any, req *mcp.ListToolsRequest, result *mcp.ListToolsResult) {
	terminal := mcpTerminal(ctx, req.Header)
	result.Tools = h.filterMcpToolsByTerminal(terminal, result.Tools)
	err := h.applyMcpToolPrompts(ctx, result.Tools)
	if err != nil {
		log.Error(fmt.Sprintf("查询 MCP 工具提示词失败 err=%v", err))
	}
}

// filterMcpToolsByTerminal 按当前服务关键字筛选 MCP 工具。
func (h *McpCase) filterMcpToolsByTerminal(terminal string, values []mcp.Tool) []mcp.Tool {
	tools := make([]mcp.Tool, 0, len(values))
	for _, tool := range values {
		if !matchMcpToolPrefix(terminal, tool.Name) {
			continue
		}
		tools = append(tools, tool)
	}
	return tools
}

// applyMcpToolPrompts 使用 base_api.tool_prompts 覆盖 MCP 工具描述。
func (h *McpCase) applyMcpToolPrompts(ctx context.Context, tools []mcp.Tool) error {
	if h == nil || h.baseAPIRepo == nil || len(tools) == 0 {
		return nil
	}
	toolNames := make([]string, 0, len(tools))
	for _, tool := range tools {
		if tool.Name == "" {
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
	for i := range tools {
		if promptDescByName[tools[i].Name] == "" {
			continue
		}
		tools[i].Description = promptDescByName[tools[i].Name]
	}
	return nil
}

// filterToolCall 在 mcp-go 执行链路中校验工具状态、终端、登录权限和只读限制。
func (h *McpCase) filterToolCall(next server.ToolHandlerFunc) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		baseAPI, err := h.findEnabledBaseAPI(ctx, req, req.Params.Name)
		if err != nil {
			log.Error(fmt.Sprintf("查询 MCP 工具状态失败 err=%v", err))
			return mcp.NewToolResultError(h.localizeMcpError(ctx, err)), nil
		}
		if baseAPI == nil {
			message := h.localizeMcpMessage(ctx, "system.mcp.tool_not_registered", map[string]any{"Name": req.Params.Name}, "MCP tool {{.Name}} is not registered.")
			return mcp.NewToolResultError(message), nil
		}
		if err = h.authorizeToolCall(ctx, baseAPI); err != nil {
			return mcp.NewToolResultError(h.localizeMcpError(ctx, err)), nil
		}
		if agentToolBlocked(baseAPI) || !agentToolReadOnly(baseAPI) {
			return mcp.NewToolResultError("写入操作须通过 AI 助手人工确认，执行类工具暂不开放"), nil
		}
		return next(ctx, req)
	}
}

// findEnabledBaseAPI 查询当前工具是否允许调用。
func (h *McpCase) findEnabledBaseAPI(ctx context.Context, req mcp.CallToolRequest, toolName string) (*models.BaseAPI, error) {
	if h == nil || h.baseAPIRepo == nil {
		return nil, errorsx.Internal("MCP权限仓储未初始化")
	}
	terminal := mcpTerminal(ctx, req.Header)
	if !matchMcpToolPrefix(terminal, toolName) {
		return nil, nil
	}
	query := h.baseAPIRepo.Query(ctx).BaseAPI
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.McpStatus.Eq(_const.STATUS_STATUS_ENABLE)))
	opts = append(opts, repository.Where(query.ToolName.Eq(toolName)))
	opts = append(opts, repository.Limit(1))
	list, err := h.baseAPIRepo.List(ctx, opts...)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return list[0], nil
}

// authorizeToolCall 按当前登录主体重新校验 MCP 工具对应的 API 权限。
func (h *McpCase) authorizeToolCall(ctx context.Context, api *models.BaseAPI) error {
	if h.authorizer == nil {
		return errorsx.Internal("MCP鉴权引擎未初始化")
	}
	authInfo, err := auth.FromContext(ctx)
	if err != nil || authInfo == nil || authInfo.RoleCode == "" || authInfo.TenantCode == "" {
		return errorsx.Unauthenticated("MCP用户认证失败")
	}
	tenant := engine.Tenant(authInfo.TenantCode)
	authzCtx := engine.ContextWithAuthClaims(ctx, &engine.AuthClaims{Tenant: &tenant})
	var allowed bool
	allowed, err = h.authorizer.IsAuthorized(authzCtx, engine.Subject(authInfo.RoleCode), engine.Action(strings.ToUpper(api.Method)), engine.Resource(api.Operation), "")
	if err != nil {
		return errorsx.Internal("MCP工具权限校验失败").WithCause(err)
	}
	if !allowed {
		return errorsx.PermissionDenied("无权调用 MCP 工具")
	}
	return nil
}

// localizeMcpError 将 MCP 工具鉴权和状态错误按当前请求语言本地化。
func (h *McpCase) localizeMcpError(ctx context.Context, err error) string {
	if h.catalog == nil || h.catalog.Empty() {
		return "MCP tool execution failed. Check the service logs."
	}
	localized := i18n.LocalizeError(h.catalog, biz.LocaleFromContext(ctx), "zh-CN", err)
	structured := kratosErrors.FromError(localized)
	if structured == nil || structured.Message == "" {
		return h.localizeMcpMessage(ctx, "system.mcp.tool_check_failed", nil, "Could not verify MCP tool access. Check the service logs.")
	}
	return structured.Message
}

// localizeMcpMessage 从统一国际化目录读取 MCP 工具提示。
func (h *McpCase) localizeMcpMessage(ctx context.Context, key string, args map[string]any, fallback string) string {
	if h.catalog == nil || h.catalog.Empty() {
		return fallback
	}
	return h.catalog.Localize(biz.LocaleFromContext(ctx), "zh-CN", key, args, fallback)
}

// toolPromptsDescription 将多条工具提示词合并为运行时工具描述。
func toolPromptsDescription(value string) string {
	return strings.Join(parseToolPrompts(value), "\n")
}

// parseToolPrompts 解析工具提示词 JSON 并过滤空值。
func parseToolPrompts(value string) []string {
	if value == "" {
		return nil
	}
	var prompts []string
	err := json.Unmarshal([]byte(value), &prompts)
	if err != nil {
		return nil
	}
	values := make([]string, 0, len(prompts))
	for _, item := range prompts {
		if item != "" {
			values = append(values, item)
		}
	}
	return values
}

// matchMcpToolPrefix 判断工具名是否匹配当前终端。
func matchMcpToolPrefix(terminal, toolName string) bool {
	return matchToolTerminal(terminal, toolName)
}

// matchToolTerminal 按终端筛选工具，并允许两端复用 BI 语义问数。
func matchToolTerminal(terminal, toolName string) bool {
	if toolName == "" {
		return false
	}
	if terminal == "" {
		return true
	}
	switch toolName {
	case "bi_admin_v1_bi_query_service_ask_bi_query", "bi_admin_v1_bi_query_service_list_bi_sources", "bi_admin_v1_bi_query_service_preview_bi_query":
		return true
	}
	return strings.HasPrefix(toolName, "base_") ||
		strings.HasPrefix(toolName, terminal+"_") ||
		strings.Contains(toolName, "_"+terminal+"_")
}

// mcpTerminal 获取当前 MCP 请求的服务筛选关键字。
func mcpTerminal(ctx context.Context, header http.Header) string {
	if terminal := header.Get(mcpTerminalHeader); terminal != "" {
		return terminal
	}
	terminal, _ := ctx.Value(mcpTerminalContextKey{}).(string)
	return terminal
}
