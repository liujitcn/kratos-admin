package adk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	einoadk "github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/liujitcn/kratos-admin/backend/pkg/agent/eino/callback"
	"github.com/liujitcn/kratos-admin/backend/pkg/agent/eino/message"
	einoMiddleware "github.com/liujitcn/kratos-admin/backend/pkg/agent/eino/middleware"
	einoModel "github.com/liujitcn/kratos-admin/backend/pkg/agent/eino/model"
)

// defaultMaxIterations 限制单轮 Agent 的模型与工具循环次数，避免工具调用链异常时拖垮请求。
const defaultMaxIterations = 4

// Runner 基于 Eino ADK ChatModelAgent 运行 AgenticMessage 对话。
type Runner struct {
	model       einoModel.AgenticModel
	name        string
	description string
}

// Config 表示 ADK Runner 初始化配置。
type Config struct {
	// Model 当前 Agent 使用的模型。
	Model einoModel.AgenticModel
	// Name Agent 名称。
	Name string
	// Description Agent 能力描述。
	Description string
}

// Request 表示单轮 ADK 执行输入。
type Request struct {
	// Messages 已构造好的会话消息。
	Messages []*message.AgenticMessage
	// Tools 当前终端可执行的工具。
	Tools []tool.BaseTool
	// ToolInfos 本轮允许暴露给模型的工具定义。
	ToolInfos []*schema.ToolInfo
	// Recorder 调用记录器。
	Recorder *callback.Recorder
	// Stream 是否启用模型流式输出。
	Stream bool
	// OnDelta 用户可见文本增量回调。
	OnDelta func(string)
}

// Result 表示 ADK 执行后的最终输出。
type Result struct {
	// Message 最终助手消息。
	Message *message.AgenticMessage
	// Token 累计 token 消耗。
	Token callback.TokenUsage
}

// NewRunner 创建 ADK Runner。
func NewRunner(config Config) *Runner {
	return &Runner{
		model:       config.Model,
		name:        config.Name,
		description: config.Description,
	}
}

// Run 执行单轮 Agent。
func (r *Runner) Run(ctx context.Context, request Request) (*Result, error) {
	// ADK Runner 必须持有模型实例，否则无法创建 ChatModelAgent。
	if r == nil || r.model == nil {
		return nil, fmt.Errorf("adk runner model is not configured")
	}
	recorder := request.Recorder
	// 调用方未传 recorder 时仍创建一份，保证模型与工具统计逻辑可以统一写入 context。
	if recorder == nil {
		recorder = &callback.Recorder{}
	}
	ctx = callback.WithRecorder(ctx, recorder)
	agent, err := r.newAgent(ctx, request)
	if err != nil {
		return nil, err
	}
	runner := einoadk.NewTypedRunner(einoadk.TypedRunnerConfig[*schema.AgenticMessage]{
		Agent: agent,
		// OpenAI 兼容网关会在同一 Agent 流混合服务端工具块与文本块，当前 Eino 版本无法合并该序列。
		EnableStreaming: false,
	})
	iter := runner.Run(ctx, request.Messages, einoadk.WithCallbacks(callback.NewHandler()))
	var messageValue *schema.AgenticMessage
	messageValue, err = consumeEvents(iter, request.OnDelta, request.Stream)
	if err != nil {
		return nil, err
	}
	// ADK 事件全部消费后仍没有助手消息，说明模型没有产出可落库的最终回复。
	if messageValue == nil {
		return nil, fmt.Errorf("ai ai response is empty")
	}
	token := recorder.TotalToken()
	// 部分模型或中间件链路不会触发统计回调，此时直接从最终消息上兜底提取 token。
	if token == (callback.TokenUsage{}) {
		token = callback.TokenFromMessage(messageValue)
	}
	return &Result{Message: messageValue, Token: token}, nil
}

// newAgent 创建带项目中间件的 Eino ChatModelAgent。
func (r *Runner) newAgent(ctx context.Context, request Request) (*einoadk.TypedChatModelAgent[*schema.AgenticMessage], error) {
	handlers := []einoadk.TypedChatModelAgentMiddleware[*schema.AgenticMessage]{
		einoMiddleware.NewToolFilterHandler(request.ToolInfos),
		einoMiddleware.NewResponsesServerToolHandler(),
		einoMiddleware.NewToolMetricsHandler(toolTitleResolver(request.ToolInfos)),
	}
	return einoadk.NewTypedChatModelAgent(ctx, &einoadk.TypedChatModelAgentConfig[*schema.AgenticMessage]{
		Name:        r.name,
		Description: r.description,
		Model:       r.model,
		ToolsConfig: einoadk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools:               request.Tools,
				ExecuteSequentially: true,
			},
		},
		Handlers:         handlers,
		ModelRetryConfig: einoMiddleware.ModelRetryConfig(einoMiddleware.ModelRetryOptions{MaxRetries: 1}),
		MaxIterations:    defaultMaxIterations,
	})
}

// consumeEvents 消费 ADK 事件流，返回本轮最终助手消息。
func consumeEvents(iter *einoadk.AsyncIterator[*einoadk.TypedAgentEvent[*schema.AgenticMessage]], onDelta func(string), stream bool) (*schema.AgenticMessage, error) {
	var finalMessage *schema.AgenticMessage
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
			return nil, event.Err
		}
		// 只处理模型消息输出事件，工具节点等其他事件由 callback 中间件记录。
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}
		messageOutput := event.Output.MessageOutput
		// 流式输出需要逐 chunk 消费，并在消费过程中透传文本增量给 SSE。
		if messageOutput.IsStreaming {
			currentMessage, err := consumeMessageStream(messageOutput.MessageStream, onDelta, stream, &streamText)
			if err != nil {
				return nil, err
			}
			// 流式片段合并成功后，作为当前可用的最终助手消息候选。
			if currentMessage != nil {
				finalMessage = currentMessage
			}
			continue
		}
		currentMessage := messageOutput.Message
		// 非消息事件可能只有状态变化，没有可展示内容。
		if currentMessage == nil {
			continue
		}
		// 非流式事件也可能携带助手文本；只有尚未通过 chunk 透传过文本时才推给前端，避免 SSE 重复显示最终消息。
		if stream && onDelta != nil && streamText.Len() == 0 && messageOutput.AgenticRole == schema.AgenticRoleTypeAssistant && !hasToolCall(currentMessage) {
			if text := message.AITextOnly(currentMessage); text != "" {
				streamText.WriteString(text)
				onDelta(text)
			}
		}
		// 只有助手角色可以作为本轮最终回复；工具结果消息仅作为中间上下文。
		if currentMessage.Role == schema.AgenticRoleTypeAssistant {
			finalMessage = currentMessage
		}
	}
	// 个别流式模型只返回 chunk，没有最终 message 事件，此时用已透传文本拼出最终回复。
	if finalMessage == nil && streamText.Len() > 0 {
		finalMessage = message.AIText(streamText.String())
	}
	return finalMessage, nil
}

// consumeMessageStream 消费单个模型流并返回合并后的助手消息。
func consumeMessageStream(
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
		// 只有处于 SSE 模式、chunk 属于助手文本且不包含工具调用时，才向前端透传。
		if !stream || onDelta == nil || chunk.Role != schema.AgenticRoleTypeAssistant || hasToolCall(chunk) {
			continue
		}
		text := message.AITextOnly(chunk)
		// 空文本 chunk 可能只携带元数据或服务端工具事件，不能作为用户可见增量。
		if text == "" {
			continue
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
func toolTitleResolver(infos []*schema.ToolInfo) einoMiddleware.ToolTitleResolver {
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
