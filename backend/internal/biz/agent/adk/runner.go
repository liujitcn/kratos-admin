package adk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/dynamictool/toolsearch"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/callback"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/message"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/middleware"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	agenttool "github.com/liujitcn/kratos-admin/backend/internal/biz/agent/tool"
)

// defaultMaxIterations 限制单轮 Agent 的模型与工具循环次数，避免工具调用链异常时拖垮请求。
// 聚合网关的 tool_search 元工具检索会占用多个前置轮次，工具执行后仍需总结轮，
// 上限过低会在工具结果就绪后耗尽轮次导致整轮失败。
const defaultMaxIterations = 8

// maxCheckpointInputBytes 为快照重复保存上下文及模型、工具输出预留空间。
const maxCheckpointInputBytes = 28 * 1024 * 1024

// Runner 基于 Eino ADK ChatModelAgent 运行 AgenticMessage 对话。
type Runner struct {
	checkPointStore compose.CheckPointStore
	model           model.AgenticModel
	name            string
	description     string
	serverTools     bool
}

// Config 表示 ADK Runner 初始化配置。
type Config struct {
	// CheckPointStore 保存人工确认中断时的运行状态。
	CheckPointStore compose.CheckPointStore
	// Model 当前 Agent 使用的模型。
	Model model.AgenticModel
	// Name Agent 名称。
	Name string
	// Description Agent 能力描述。
	Description string
	// ServerTools 是否注入 Responses 服务端工具（联网搜索等），仅 Responses 协议支持。
	ServerTools bool
}

// Request 表示单轮 ADK 执行输入。
type Request struct {
	// CheckPointID 是由业务层分配的运行编号。
	CheckPointID string
	// Decisions 只允许受信业务入口传入定向审批决定。
	Decisions map[string]any
	// Messages 已构造好的会话消息。
	Messages []*message.AgenticMessage
	// Tools 当前终端可执行的工具。
	Tools []tool.BaseTool
	// ToolInfos 本轮允许暴露给模型的工具定义。
	ToolInfos []*schema.ToolInfo
	// DynamicTools 使用框架动态发现，工具节点由中间件装入工具。
	DynamicTools bool
	// ResolveToolInfos 每次模型调用前重新读取有效权限。
	ResolveToolInfos func(context.Context) []*schema.ToolInfo
	// Recorder 调用记录器。
	Recorder *callback.Recorder
	// LocalizeMessage 按请求语言生成用户可见的错误文案。
	LocalizeMessage func(context.Context, string, map[string]any, string) string
	// ModelOptions 是传给底层 Eino 模型的请求选项。
	ModelOptions []model.Option
	// Stream 表示本次调用是否处于 SSE 流式场景，控制文本增量透传；模型实际调用
	// 方式由 ModelStreaming 独立控制，禁用流式的供应商仍可整段透传文本。
	Stream bool
	// ModelStreaming 表示是否启用模型流式调用（ADK EnableStreaming）。供应商被
	// 禁用流式时应为 false，此时由非流式事件兜底整段透传文本。
	ModelStreaming bool
	// OnDelta 用户可见文本增量回调。
	OnDelta func(string)
}

// Result 表示 ADK 执行后的最终输出。
type Result struct {
	// Pending 是需要用户明确决策的工具调用。
	Pending []agenttool.ApprovalCall
	// Message 最终助手消息。
	Message *message.AgenticMessage
	// Token 累计 token 消耗。
	Token callback.TokenUsage
}

// NewRunner 创建 ADK Runner。
func NewRunner(config Config) *Runner {
	return &Runner{
		checkPointStore: config.CheckPointStore,
		model:           config.Model,
		name:            config.Name,
		description:     config.Description,
		serverTools:     config.ServerTools,
	}
}

// Run 执行单轮 Agent。
func (r *Runner) Run(ctx context.Context, request Request) (*Result, error) {
	// ADK Runner 必须持有模型实例，否则无法创建 ChatModelAgent。
	if r == nil || r.model == nil {
		return nil, fmt.Errorf("adk runner model is not configured")
	}
	var err error
	if r.checkPointStore != nil && request.CheckPointID != "" && request.Decisions == nil {
		var input []byte
		input, err = json.Marshal(request.Messages)
		if err != nil {
			return nil, fmt.Errorf("encode agent checkpoint input: %w", err)
		}
		if len(input) > maxCheckpointInputBytes {
			hint := "Conversation input including encoded attachments exceeds 28 MiB. Reduce attachments or start a new conversation."
			if request.LocalizeMessage != nil {
				hint = request.LocalizeMessage(ctx, "system.ai.chat.error.context_too_large", nil, hint)
			}
			return nil, errors.New(hint)
		}
	}
	recorder := request.Recorder
	// 调用方未传 recorder 时仍创建一份，保证模型与工具统计逻辑可以统一写入 context。
	if recorder == nil {
		recorder = &callback.Recorder{}
	}
	ctx = callback.WithRecorder(ctx, recorder)
	var agent *adk.TypedChatModelAgent[*schema.AgenticMessage]
	agent, err = r.newAgent(ctx, request)
	if err != nil {
		return nil, err
	}
	runner := adk.NewTypedRunner(adk.TypedRunnerConfig[*schema.AgenticMessage]{
		Agent:           agent,
		CheckPointStore: r.checkPointStore,
		// 流式输出经模型层的 normalizeAgenticStream 归一化（混合块重排为顺序流式标记），
		// ADK 内部合并不会再因混合块序列失败；工具事件由 Runner 统一消费透传。
		EnableStreaming: request.ModelStreaming,
	})
	runOptions := []adk.AgentRunOption{adk.WithCallbacks(callback.NewHandler())}
	if len(request.ModelOptions) > 0 {
		runOptions = append(runOptions, adk.WithChatModelOptions(request.ModelOptions))
	}
	var iter *adk.AsyncIterator[*adk.TypedAgentEvent[*schema.AgenticMessage]]
	if request.Decisions != nil {
		iter, err = runner.ResumeWithParams(ctx, request.CheckPointID, &adk.ResumeParams{Targets: request.Decisions}, runOptions...)
		if err != nil {
			return nil, err
		}
	} else {
		runOptions = append(runOptions, adk.WithCheckPointID(request.CheckPointID))
		iter = runner.Run(ctx, request.Messages, runOptions...)
	}
	var messageValue *schema.AgenticMessage
	var pending []agenttool.ApprovalCall
	messageValue, pending, err = consumeEvents(ctx, iter, request.OnDelta, request.Stream)
	if err != nil {
		return nil, err
	}
	for _, call := range pending {
		callback.EmitActivity(ctx, callback.Activity{
			ID:        call.InterruptID,
			Phase:     "tool",
			Status:    "waiting_approval",
			ToolName:  call.Name,
			ToolTitle: call.Title,
		})
	}
	// ADK 事件全部消费后仍没有助手消息，说明模型没有产出可落库的最终回复。
	if messageValue == nil && len(pending) == 0 {
		return nil, fmt.Errorf("ai ai response is empty")
	}
	token := recorder.TotalToken()
	// 部分模型或中间件链路不会触发统计回调，此时直接从最终消息上兜底提取 token。
	if token == (callback.TokenUsage{}) && messageValue != nil {
		token = callback.TokenFromMessage(messageValue)
	}
	return &Result{Message: messageValue, Token: token, Pending: pending}, nil
}

// newAgent 创建带项目中间件的 Eino ChatModelAgent。
func (r *Runner) newAgent(ctx context.Context, request Request) (*adk.TypedChatModelAgent[*schema.AgenticMessage], error) {
	var handlers []adk.TypedChatModelAgentMiddleware[*schema.AgenticMessage]
	tools := request.Tools
	infos := append([]*schema.ToolInfo(nil), request.ToolInfos...)
	if request.DynamicTools && len(tools) > 0 {
		search, err := toolsearch.NewTyped[*schema.AgenticMessage](ctx, &toolsearch.Config{DynamicTools: tools, UseModelToolSearch: false})
		if err != nil {
			return nil, err
		}
		handlers = append(handlers, search)
		tools = nil
		infos = append(infos, &schema.ToolInfo{Name: "tool_search"})
	}
	if r.serverTools {
		handlers = append(handlers, middleware.NewResponsesServerToolHandler())
	}
	filter := middleware.NewToolFilterHandler(infos)
	filter.BeforeModel = agenttool.CheckModelExecution
	filter.ResolveToolInfos = request.ResolveToolInfos
	filter.AllowToolSearch = request.DynamicTools && len(request.Tools) > 0
	handlers = append(handlers, filter, middleware.NewToolMetricsHandler(toolTitleResolver(infos), request.LocalizeMessage))
	return adk.NewTypedChatModelAgent(ctx, &adk.TypedChatModelAgentConfig[*schema.AgenticMessage]{
		Name:        r.name,
		Description: r.description,
		Model:       r.model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools:               tools,
				ExecuteSequentially: true,
			},
		},
		Handlers:         handlers,
		ModelRetryConfig: middleware.ModelRetryConfig(middleware.ModelRetryOptions{MaxRetries: 1}),
		MaxIterations:    defaultMaxIterations,
	})
}

// consumeEvents 消费 ADK 事件流，返回本轮最终助手消息。
func consumeEvents(ctx context.Context, iter *adk.AsyncIterator[*adk.TypedAgentEvent[*schema.AgenticMessage]], onDelta func(string), stream bool) (*schema.AgenticMessage, []agenttool.ApprovalCall, error) {
	var finalMessage *schema.AgenticMessage
	var pending []agenttool.ApprovalCall
	var streamText strings.Builder
	for {
		event, ok := iter.Next()
		// 迭代器结束表示 ADK 本轮模型与工具循环已经完成。
		if !ok {
			break
		}
		// ADK 偶发空事件时没有业务信息，直接跳过以继续消费后续事件。
		if event == nil {
			continue
		}
		// 事件携带错误时需要立即终止，避免把不完整回复误当成最终结果。
		if event.Err != nil {
			return nil, nil, event.Err
		}
		if event.Action != nil && event.Action.Interrupted != nil {
			for _, interrupted := range event.Action.Interrupted.InterruptContexts {
				info, ok := interrupted.Info.(agenttool.ApprovalInfo)
				if ok {
					pending = append(pending, agenttool.ApprovalCall{ApprovalInfo: info, InterruptID: interrupted.ID})
				}
			}
			if len(pending) == 0 {
				return nil, nil, fmt.Errorf("unsupported agent interruption")
			}
		}
		// 只处理模型消息输出事件，工具节点等其他事件由 callback 中间件记录。
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}
		messageOutput := event.Output.MessageOutput
		// 流式输出需要逐 chunk 消费，并在消费过程中透传文本增量给 SSE。
		if messageOutput.IsStreaming {
			currentMessage, err := consumeMessageStream(ctx, messageOutput.MessageStream, onDelta, stream, &streamText)
			if err != nil {
				return nil, nil, err
			}
			// 流式片段合并成功后，作为当前可用的最终助手消息候选；空助手事件不能覆盖已有文本。
			if currentMessage != nil && (message.AITextOnly(currentMessage) != "" || finalMessage == nil) {
				finalMessage = currentMessage
			}
			continue
		}
		currentMessage := messageOutput.Message
		// 非消息事件可能只有状态变化，没有可展示内容。
		if currentMessage == nil {
			continue
		}
		emitToolActivities(ctx, currentMessage)
		emitReasoningSummary(ctx, currentMessage, nil)
		// 非流式事件也可能携带助手文本；只有尚未通过 chunk 透传过文本时才推给前端，避免 SSE 重复显示最终消息。
		if stream && onDelta != nil && streamText.Len() == 0 && messageOutput.AgenticRole == schema.AgenticRoleTypeAssistant && !hasToolCall(currentMessage) {
			if text := message.AITextOnly(currentMessage); text != "" {
				streamText.WriteString(text)
				onDelta(text)
			}
		}
		// 只有助手角色可以作为本轮最终回复；工具结果消息仅作为中间上下文。
		if currentMessage.Role == schema.AgenticRoleTypeAssistant && (message.AITextOnly(currentMessage) != "" || finalMessage == nil) {
			finalMessage = currentMessage
		}
	}
	// 个别流式模型只返回 chunk，或最终事件为空，此时用已透传文本拼出最终回复。
	if streamText.Len() > 0 && message.AITextOnly(finalMessage) == "" {
		finalMessage = message.AIText(streamText.String())
	}
	return finalMessage, pending, nil
}

// consumeMessageStream 消费单个模型流并返回合并后的助手消息。
func consumeMessageStream(
	ctx context.Context,
	reader *schema.StreamReader[*schema.AgenticMessage],
	onDelta func(string),
	stream bool,
	streamText *strings.Builder,
) (*schema.AgenticMessage, error) {
	// 没有 reader 表示该事件没有可消费的流内容。
	if reader == nil {
		return nil, nil
	}
	defer reader.Close()
	chunks := make([]*schema.AgenticMessage, 0)
	reasoningText := make(map[int]string)
	answerStarted := false
	// 透传开关在流开始时确定：部分网关先发完整消息事件（已整段透传），后续流式 chunk 不再重复透传；
	// 正常流式网关在本流内逐 chunk 透传，不受此开关影响。
	deltaDisabled := streamText.Len() > 0
	for {
		chunk, err := reader.Recv()
		// io.EOF 是正常流结束信号，不应作为模型调用错误返回。
		if errors.Is(err, io.EOF) {
			break
		}
		// 部分 OpenAI 兼容网关会在 completed 事件后直接断开 SSE，忽略此时的非标准 EOF。
		if errors.Is(err, io.ErrUnexpectedEOF) && hasCompletedOpenAIResponse(chunks) {
			break
		}
		// 真实流错误需要向上返回，由调用层决定是否失败或重试。
		if err != nil {
			return nil, err
		}
		// 空 chunk 不携带文本或工具信息，忽略后继续等待后续片段。
		if chunk == nil {
			continue
		}
		chunks = append(chunks, chunk)
		emitToolActivities(ctx, chunk)
		emitReasoningSummary(ctx, chunk, reasoningText)
		// 只有处于 SSE 模式、chunk 属于助手文本且不包含工具调用时，才向前端透传。
		if !stream || onDelta == nil || deltaDisabled || chunk.Role != schema.AgenticRoleTypeAssistant || hasToolCall(chunk) {
			continue
		}
		text := message.AITextOnly(chunk)
		// 空文本 chunk 可能只携带元数据或服务端工具事件，不能作为用户可见增量。
		if text == "" {
			continue
		}
		if !answerStarted {
			answerStarted = true
			callback.EmitActivity(ctx, callback.Activity{ID: "answer", Phase: "answering", Status: "running"})
		}
		streamText.WriteString(text)
		onDelta(text)
	}
	// 没有收到任何 chunk 时返回 nil，让上层继续等待其他 ADK 事件。
	if len(chunks) == 0 {
		return nil, nil
	}
	return message.Concat(chunks)
}

// emitReasoningSummary 将 Eino 推理摘要转换为可展示活动，原始推理扩展不会进入摘要字段。
func emitReasoningSummary(ctx context.Context, value *schema.AgenticMessage, accumulated map[int]string) {
	for _, part := range message.ReasoningSummaries(value) {
		text := part.Text
		if accumulated != nil {
			accumulated[part.Index] += part.Text
			text = accumulated[part.Index]
		}
		callback.EmitActivity(ctx, callback.Activity{
			ID:      fmt.Sprintf("reasoning:%d", part.Index),
			Phase:   "reasoning",
			Status:  "running",
			Summary: text,
		})
	}
}

// emitToolActivities 将模型侧服务端工具和 MCP 工具事件转换为活动，不携带原始参数和结果。
func emitToolActivities(ctx context.Context, value *schema.AgenticMessage) {
	if value == nil {
		return
	}
	for _, block := range value.ContentBlocks {
		if block == nil {
			continue
		}
		if call := block.ServerToolCall; call != nil && call.Name != "" {
			callback.EmitActivity(ctx, callback.Activity{
				ID:        activityID(call.CallID, "server:"+call.Name),
				Phase:     "tool",
				Status:    "running",
				ToolName:  call.Name,
				ToolTitle: call.Name,
			})
		}
		if result := block.ServerToolResult; result != nil && result.Name != "" {
			callback.EmitActivity(ctx, callback.Activity{
				ID:        activityID(result.CallID, "server:"+result.Name),
				Phase:     "tool",
				Status:    "completed",
				ToolName:  result.Name,
				ToolTitle: result.Name,
			})
		}
		if call := block.MCPToolCall; call != nil && call.Name != "" {
			callback.EmitActivity(ctx, callback.Activity{
				ID:        activityID(call.CallID, "mcp:"+call.Name),
				Phase:     "tool",
				Status:    "running",
				ToolName:  call.Name,
				ToolTitle: call.Name,
			})
		}
		if result := block.MCPToolResult; result != nil && result.Name != "" {
			status := "completed"
			if result.Error != nil {
				status = "failed"
			}
			callback.EmitActivity(ctx, callback.Activity{
				ID:        activityID(result.CallID, "mcp:"+result.Name),
				Phase:     "tool",
				Status:    status,
				Summary:   mcpToolError(result.Error),
				ToolName:  result.Name,
				ToolTitle: result.Name,
			})
		}
	}
}

// activityID 返回模型工具活动的稳定编号。
func activityID(callID string, fallback string) string {
	if callID != "" {
		return callID
	}
	return fallback
}

// mcpToolError 返回 MCP 工具失败的用户可见摘要。
func mcpToolError(value *schema.MCPToolCallError) string {
	if value == nil {
		return ""
	}
	return value.Message
}

// hasCompletedOpenAIResponse 判断流中是否已收到 OpenAI Responses 的完成状态。
func hasCompletedOpenAIResponse(chunks []*schema.AgenticMessage) bool {
	for _, chunk := range chunks {
		if chunk == nil || chunk.ResponseMeta == nil || chunk.ResponseMeta.OpenAIExtension == nil {
			continue
		}
		if string(chunk.ResponseMeta.OpenAIExtension.Status) == "completed" {
			return true
		}
	}
	return false
}

// hasToolCall 判断消息中是否包含函数工具调用。
func hasToolCall(value *schema.AgenticMessage) bool {
	return len(message.ToolCalls(value)) > 0
}

// toolTitleResolver 构造工具展示标题解析器。
func toolTitleResolver(infos []*schema.ToolInfo) middleware.ToolTitleResolver {
	infoMap := make(map[string]*schema.ToolInfo, len(infos))
	for _, info := range infos {
		// 过滤无效工具定义，避免空名称污染标题映射。
		if info == nil || info.Name == "" {
			continue
		}
		infoMap[info.Name] = info
	}
	return func(name string) string {
		info := infoMap[name]
		// 未命中工具定义时使用工具名兜底，保证前端工具卡至少可识别来源。
		if info == nil {
			return name
		}
		// 生成工具的 Desc 通常已经是后台可读标题，优先展示它。
		if info.Desc != "" {
			return info.Desc
		}
		return info.Name
	}
}
