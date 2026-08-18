package middleware

import (
	"context"
	"time"

	einoadk "github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/callback"
)

// ToolTitleResolver 根据工具名返回展示标题。
type ToolTitleResolver func(name string) string

// ToolFilterHandler 在模型调用前按本轮候选裁剪工具定义。
type ToolFilterHandler struct {
	*einoadk.TypedBaseChatModelAgentMiddleware[*schema.AgenticMessage]
	allowedNames map[string]bool
	enabled      bool
}

// NewToolFilterHandler 创建工具筛选中间件。
func NewToolFilterHandler(toolInfos []*schema.ToolInfo) *ToolFilterHandler {
	allowedNames := make(map[string]bool, len(toolInfos))
	for _, info := range toolInfos {
		// 工具定义缺失名称时无法被模型稳定引用，过滤掉避免误放行。
		if info == nil || info.Name == "" {
			continue
		}
		allowedNames[info.Name] = true
	}
	return &ToolFilterHandler{
		TypedBaseChatModelAgentMiddleware: &einoadk.TypedBaseChatModelAgentMiddleware[*schema.AgenticMessage]{},
		allowedNames:                      allowedNames,
		enabled:                           true,
	}
}

// NewToolMetricsHandler 创建工具统计中间件。
func NewToolMetricsHandler(titleResolver ToolTitleResolver) *ToolMetricsHandler {
	return &ToolMetricsHandler{
		TypedBaseChatModelAgentMiddleware: &einoadk.TypedBaseChatModelAgentMiddleware[*schema.AgenticMessage]{},
		titleResolver:                     titleResolver,
	}
}

// BeforeModelRewriteState 将工具列表限制在当前轮允许暴露的集合内。
func (h *ToolFilterHandler) BeforeModelRewriteState(
	ctx context.Context,
	state *einoadk.TypedChatModelAgentState[*schema.AgenticMessage],
	mc *einoadk.TypedModelContext[*schema.AgenticMessage],
) (context.Context, *einoadk.TypedChatModelAgentState[*schema.AgenticMessage], error) {
	// 未启用筛选或状态为空时保持 ADK 原始状态，避免影响无工具的普通问答。
	if h == nil || !h.enabled || state == nil {
		return ctx, state, nil
	}
	state.ToolInfos = h.filter(state.ToolInfos)
	// TypedModelContext 中也有一份工具定义，必须同步裁剪，否则模型仍可能看到被禁用工具。
	if mc != nil {
		mc.Tools = h.filter(mc.Tools)
	}
	return ctx, state, nil
}

// BeforeAgent 将本轮可执行工具池同步限制到允许暴露的集合内。
func (h *ToolFilterHandler) BeforeAgent(
	ctx context.Context,
	runCtx *einoadk.ChatModelAgentContext,
) (context.Context, *einoadk.ChatModelAgentContext, error) {
	// runCtx 为空时没有可执行工具池，保持原始上下文。
	if h == nil || !h.enabled || runCtx == nil {
		return ctx, runCtx, nil
	}
	tools := make([]tool.BaseTool, 0, len(runCtx.Tools))
	for _, item := range runCtx.Tools {
		// 空工具没有执行能力，跳过即可。
		if item == nil {
			continue
		}
		info, err := item.Info(ctx)
		// 读取工具定义失败时不能确认名称，出于安全边界不暴露给本轮 Agent。
		if err != nil || info == nil || info.Name == "" {
			continue
		}
		// 只保留本轮候选工具，避免模型绕过业务层筛选直接调用完整工具池。
		if h.allowedNames[info.Name] {
			tools = append(tools, item)
		}
	}
	runCtx.Tools = tools
	return ctx, runCtx, nil
}

// ToolMetricsHandler 统一记录函数工具调用、耗时、入参、出参和错误 JSON。
type ToolMetricsHandler struct {
	*einoadk.TypedBaseChatModelAgentMiddleware[*schema.AgenticMessage]
	titleResolver ToolTitleResolver
}

func (h *ToolFilterHandler) filter(infos []*schema.ToolInfo) []*schema.ToolInfo {
	// 没有工具定义时直接返回原切片，减少无意义分配。
	if len(infos) == 0 {
		return infos
	}
	result := make([]*schema.ToolInfo, 0, len(infos))
	for _, info := range infos {
		// 空名称工具无法被模型调用，也不能进入候选描述。
		if info == nil || info.Name == "" {
			continue
		}
		// allowedNames 是业务层挑选出的本轮工具白名单。
		if h.allowedNames[info.Name] {
			result = append(result, info)
		}
	}
	return result
}

// WrapInvokableToolCall 包装普通函数工具调用。
func (h *ToolMetricsHandler) WrapInvokableToolCall(
	ctx context.Context,
	endpoint einoadk.InvokableToolCallEndpoint,
	tCtx *einoadk.ToolContext,
) (einoadk.InvokableToolCallEndpoint, error) {
	return func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
		startedAt := time.Now()
		output, err := endpoint(ctx, argumentsInJSON, opts...)
		name := toolContextName(tCtx)
		call := callback.ToolCall{
			Type:     "function",
			Name:     name,
			Title:    h.toolTitle(name),
			Status:   "success",
			Input:    argumentsInJSON,
			Output:   output,
			Duration: time.Since(startedAt),
		}
		// 工具调用失败时，把错误转成稳定 JSON 返回给模型，而不是让 ADK 中断整轮对话。
		if err != nil {
			call.Status = "error"
			call.Error = err.Error()
			call.Output = MarshalToolError(err.Error())
			recordToolCall(ctx, call)
			return call.Output, nil
		}
		// 空输出会让模型难以判断工具是否成功，统一使用空 JSON 对象表示成功但无数据。
		if call.Output == "" {
			call.Output = "{}"
		}
		recordToolCall(ctx, call)
		return call.Output, nil
	}, nil
}

// toolTitle 返回工具调用展示标题。
func (h *ToolMetricsHandler) toolTitle(name string) string {
	if h != nil && h.titleResolver != nil {
		return h.titleResolver(name)
	}
	return name
}

// recordToolCall 将工具调用写入统一记录器。
func recordToolCall(ctx context.Context, call callback.ToolCall) {
	recorder := callback.FromContext(ctx)
	// 没有 recorder 时说明当前调用不需要展示统计，工具结果仍正常返回给模型。
	if recorder == nil {
		return
	}
	recorder.RecordTool(call)
}

// toolContextName 读取 ADK 工具上下文中的工具名。
func toolContextName(tCtx *einoadk.ToolContext) string {
	// 工具上下文缺失时无法定位工具名，返回空字符串让 recorder 自行过滤。
	if tCtx == nil {
		return ""
	}
	return tCtx.Name
}
