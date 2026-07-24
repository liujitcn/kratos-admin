package callback

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	einoCallbacks "github.com/cloudwego/eino/callbacks"
	componentsModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	callbackTemplate "github.com/cloudwego/eino/utils/callbacks"

	"github.com/liujitcn/kratos-admin/backend/pkg/agent/eino/message"
)

type recorderContextKey struct{}
type callbackStartedAtKey struct{}

// TokenUsage 表示一次或多次模型调用累计的 token 消耗。
type TokenUsage struct {
	// Input 输入 token 数。
	Input int32
	// Output 输出 token 数。
	Output int32
	// Cache 命中缓存 token 数。
	Cache int32
	// Total 总 token 数。
	Total int32
}

// ToolCall 表示一次函数工具调用记录。
type ToolCall struct {
	// Type 工具类型，例如 function。
	Type string
	// Name 工具名称。
	Name string
	// Title 工具展示名称。
	Title string
	// Status 工具执行状态。
	Status string
	// Input 工具原始入参 JSON。
	Input string
	// Output 工具原始出参 JSON。
	Output string
	// Duration 工具执行耗时。
	Duration time.Duration
	// Error 工具执行错误。
	Error string
}

// ServerTool 表示模型侧服务端工具调用记录。
type ServerTool struct {
	// Name 工具名称。
	Name string
	// Title 工具展示名称。
	Title string
	// Status 工具执行状态。
	Status string
}

// ModelCall 表示一次模型调用记录。
type ModelCall struct {
	// Mode 模型调用模式，例如 generate 或 stream。
	Mode string
	// Duration 模型调用耗时。
	Duration time.Duration
	// Token 本次模型调用 token 消耗。
	Token TokenUsage
	// Error 模型调用错误。
	Error string
}

// Recorder 记录 ADK 模型与工具调用过程，供业务协议层统一汇总。
type Recorder struct {
	mu          sync.Mutex
	modelCalls  []ModelCall
	toolCalls   []ToolCall
	serverTools []ServerTool
}

// NewHandler 创建 Eino 原生 Callback Handler。
//
// Quick Start 的 Callback/Trace 章节要求通过 Eino callback 链路观测组件执行。
// 这里把 AgenticModel 的开始、结束、流式结束和错误事件统一写入 Recorder，
// 让业务层继续复用现有 Token/Tools 协议，同时不再依赖手写模型包装器统计模型调用。
func NewHandler() einoCallbacks.Handler {
	return callbackTemplate.NewHandlerHelper().
		AgenticModel(&callbackTemplate.AgenticModelCallbackHandler{
			OnStart: func(ctx context.Context, _ *einoCallbacks.RunInfo, _ *componentsModel.AgenticCallbackInput) context.Context {
				return context.WithValue(ctx, callbackStartedAtKey{}, time.Now())
			},
			OnEnd: func(ctx context.Context, _ *einoCallbacks.RunInfo, output *componentsModel.AgenticCallbackOutput) context.Context {
				recordAgenticModelOutput(ctx, "generate", output, nil)
				return ctx
			},
			OnEndWithStreamOutput: func(ctx context.Context, _ *einoCallbacks.RunInfo, output *schema.StreamReader[*componentsModel.AgenticCallbackOutput]) context.Context {
				recordAgenticModelStream(ctx, output)
				return ctx
			},
			OnError: func(ctx context.Context, _ *einoCallbacks.RunInfo, err error) context.Context {
				recordAgenticModelOutput(ctx, "model", nil, err)
				return ctx
			},
		}).
		Handler()
}

// WithRecorder 将调用记录器写入 context。
func WithRecorder(ctx context.Context, recorder *Recorder) context.Context {
	if recorder == nil {
		return ctx
	}
	return context.WithValue(ctx, recorderContextKey{}, recorder)
}

// FromContext 从 context 中读取调用记录器。
func FromContext(ctx context.Context) *Recorder {
	if ctx == nil {
		return nil
	}
	recorder, _ := ctx.Value(recorderContextKey{}).(*Recorder)
	return recorder
}

// MergeToken 合并 token 统计。
func MergeToken(left TokenUsage, right TokenUsage) TokenUsage {
	return TokenUsage{
		Input:  left.Input + right.Input,
		Output: left.Output + right.Output,
		Cache:  left.Cache + right.Cache,
		Total:  left.Total + right.Total,
	}
}

// TokenFromMessage 提取模型消息中的 token 消耗。
func TokenFromMessage(value *message.AgenticMessage) TokenUsage {
	usage := message.Usage(value)
	if usage == nil {
		return TokenUsage{}
	}
	return TokenUsage{
		Input:  int32(usage.PromptTokens),
		Output: int32(usage.CompletionTokens),
		Cache:  int32(usage.PromptTokenDetails.CachedTokens),
		Total:  int32(usage.TotalTokens),
	}
}

// RecordModel 记录一次模型调用。
func (r *Recorder) RecordModel(call ModelCall) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.modelCalls = append(r.modelCalls, call)
}

// RecordTool 记录一次函数工具调用。
func (r *Recorder) RecordTool(call ToolCall) {
	if r == nil || call.Name == "" {
		return
	}
	if call.Type == "" {
		call.Type = "function"
	}
	if call.Title == "" {
		call.Title = call.Name
	}
	if call.Status == "" {
		call.Status = "success"
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.toolCalls = append(r.toolCalls, call)
}

// RecordServerTools 记录模型侧服务端工具调用。
func (r *Recorder) RecordServerTools(tools []ServerTool) {
	if r == nil || len(tools) == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, tool := range tools {
		if tool.Name == "" {
			continue
		}
		if tool.Title == "" {
			tool.Title = tool.Name
		}
		if tool.Status == "" {
			tool.Status = "success"
		}
		r.serverTools = append(r.serverTools, tool)
	}
}

// ModelCalls 返回模型调用快照。
func (r *Recorder) ModelCalls() []ModelCall {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]ModelCall(nil), r.modelCalls...)
}

// ToolCalls 返回函数工具调用快照。
func (r *Recorder) ToolCalls() []ToolCall {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]ToolCall(nil), r.toolCalls...)
}

// ServerTools 返回服务端工具调用快照。
func (r *Recorder) ServerTools() []ServerTool {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]ServerTool(nil), r.serverTools...)
}

// TotalToken 返回模型调用累计 token。
func (r *Recorder) TotalToken() TokenUsage {
	if r == nil {
		return TokenUsage{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var total TokenUsage
	for _, call := range r.modelCalls {
		total = MergeToken(total, call.Token)
	}
	return total
}

// tokenFromAgenticCallbackOutput 从 Eino AgenticModel callback 输出中提取 token。
func tokenFromAgenticCallbackOutput(output *componentsModel.AgenticCallbackOutput) TokenUsage {
	if output == nil {
		return TokenUsage{}
	}
	if output.TokenUsage != nil {
		return TokenUsage{
			Input:  int32(output.TokenUsage.PromptTokens),
			Output: int32(output.TokenUsage.CompletionTokens),
			Cache:  int32(output.TokenUsage.PromptTokenDetails.CachedTokens),
			Total:  int32(output.TokenUsage.TotalTokens),
		}
	}
	return TokenFromMessage(output.Message)
}

// recordAgenticModelOutput 记录一次非流式模型 callback 输出。
func recordAgenticModelOutput(ctx context.Context, mode string, output *componentsModel.AgenticCallbackOutput, err error) {
	recorder := FromContext(ctx)
	// 调用方未提供 Recorder 时仅跳过统计，不影响 Eino 主链路。
	if recorder == nil {
		return
	}
	call := ModelCall{
		Mode:     mode,
		Duration: durationFromContext(ctx),
		Token:    tokenFromAgenticCallbackOutput(output),
	}
	if err != nil {
		call.Error = err.Error()
	}
	recorder.RecordModel(call)
	if output != nil && output.Message != nil {
		recorder.RecordServerTools(serverToolsFromMessage(output.Message))
	}
}

// recordAgenticModelStream 消费 Eino callback 的流式副本并记录最终统计。
func recordAgenticModelStream(ctx context.Context, output *schema.StreamReader[*componentsModel.AgenticCallbackOutput]) {
	if output == nil {
		recordAgenticModelOutput(ctx, "stream", nil, nil)
		return
	}
	defer output.Close()

	var finalOutput *componentsModel.AgenticCallbackOutput
	var streamErr error
	for {
		chunk, err := output.Recv()
		// io.EOF 表示 callback 流式副本正常结束。
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			streamErr = err
			break
		}
		if chunk == nil {
			continue
		}
		finalOutput = mergeAgenticCallbackOutput(finalOutput, chunk)
	}
	recordAgenticModelOutput(ctx, "stream", finalOutput, streamErr)
}

// mergeAgenticCallbackOutput 合并流式 callback 片段，供 token 和最终消息统计使用。
func mergeAgenticCallbackOutput(left *componentsModel.AgenticCallbackOutput, right *componentsModel.AgenticCallbackOutput) *componentsModel.AgenticCallbackOutput {
	if right == nil {
		return left
	}
	if left == nil {
		return right
	}
	if right.TokenUsage != nil {
		left.TokenUsage = right.TokenUsage
	}
	if right.Message != nil {
		if left.Message == nil {
			left.Message = right.Message
		} else if merged, err := message.Concat([]*message.AgenticMessage{left.Message, right.Message}); err == nil {
			left.Message = merged
		}
	}
	return left
}

// durationFromContext 返回当前 callback 从 OnStart 到当前时刻的耗时。
func durationFromContext(ctx context.Context) time.Duration {
	startedAt, _ := ctx.Value(callbackStartedAtKey{}).(time.Time)
	if startedAt.IsZero() {
		return 0
	}
	return time.Since(startedAt)
}

// serverToolsFromMessage 从模型消息中提取服务端工具调用记录。
func serverToolsFromMessage(value *message.AgenticMessage) []ServerTool {
	serverTools := message.ServerTools(value)
	// 当前消息不含服务端工具事件时不写入记录，避免产生空工具卡。
	if len(serverTools) == 0 {
		return nil
	}
	tools := make([]ServerTool, 0, len(serverTools))
	for _, item := range serverTools {
		tools = append(tools, ServerTool{
			Name:   item.Name,
			Title:  serverToolTitle(item.Name),
			Status: "success",
		})
	}
	return tools
}

// serverToolTitle 返回服务端工具的前端展示标题。
func serverToolTitle(name string) string {
	// OpenAI Responses 的 web_search 工具在前端展示为中文名称，便于用户理解来源。
	if name == "web_search" {
		return "联网搜索"
	}
	return name
}
