package agent

import (
	"context"
	"fmt"
	"slices"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/adk"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/callback"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/tool"
)

// Resume 沿用原始终端和模型上下文恢复工具调用，并在执行入口重新检查当前权限与工具开关。
func (r *Runtime) Resume(ctx context.Context, checkpointID string, state RunState, calls []ApprovalCall, approved bool) (*Response, error) {
	pool := r.requestToolPool(ctx, state.Terminal)
	if state.ToolRevision != pool.Revision {
		return nil, fmt.Errorf("agent tool configuration changed; start a new request")
	}
	input := RuntimeInput{CheckPointID: checkpointID, Terminal: state.Terminal, ProviderID: state.ProviderID, ModelName: state.ModelName}
	client, err := r.resolveAssistantClient(input)
	if err != nil {
		return nil, err
	}
	if r.checkPointStore == nil || len(calls) == 0 {
		return nil, fmt.Errorf("agent approval state is unavailable")
	}
	ctx = ContextWithTerminal(ctx, input.Terminal)
	ctx = ContextWithAssistantClient(ctx, client)
	allowed := make(map[string]bool, len(state.ToolNames))
	for _, name := range state.ToolNames {
		allowed[name] = true
	}
	infos := r.enabledToolInfos(ctx, input, r.allToolInfos(ctx, input))
	selected := make([]*tool.Info, 0, len(infos))
	for _, info := range infos {
		if allowed[info.Name] {
			selected = append(selected, info)
		}
	}
	decisions := make(map[string]any, len(calls))
	for _, call := range calls {
		decisions[call.InterruptID] = tool.ApprovalDecision{ApprovalInfo: call.ApprovalInfo, Approved: approved}
	}
	for _, call := range calls {
		if !allowed[call.Name] || !tool.HasInfo(selected, call.Name) {
			return nil, fmt.Errorf("pending agent tool is no longer authorized")
		}
	}
	recorder := &callback.Recorder{}
	runner := adk.NewRunner(adk.Config{Model: client.AgenticModel, Name: "admin_ai", ServerTools: state.ServerTools, CheckPointStore: r.checkPointStore})
	var result *adk.Result
	result, err = runner.Run(ctx, adk.Request{CheckPointID: checkpointID, Decisions: decisions, Tools: r.runnerTools(ctx, input, selected), ToolInfos: selected, DynamicTools: true, ResolveToolInfos: func(callCtx context.Context) []*tool.Info { return r.enabledToolInfos(callCtx, input, selected) }, Recorder: recorder, LocalizeMessage: r.localizeMessage})
	if err != nil {
		return nil, err
	}
	return r.responseFromRun(ctx, input, client, selected, !state.ServerTools, result, recorder), nil
}

// responseFromRun 将工具中断与最终回复统一转换为 Runtime 输出。
func (r *Runtime) responseFromRun(ctx context.Context, input RuntimeInput, client *model.AssistantClient, infos []*tool.Info, internalToolMatched bool, result *adk.Result, recorder *callback.Recorder) *Response {
	response := r.buildResponse(result.Message, tokenFromCallback(result.Token), toolsFromRecorder(recorder), client.Name())
	if len(result.Pending) == 0 {
		return response
	}
	names := make([]string, 0, len(infos))
	for _, info := range infos {
		names = append(names, info.Name)
	}
	response.Content = ""
	response.Pending = &PendingApproval{Calls: result.Pending, State: RunState{Terminal: input.Terminal, ProviderID: input.ProviderID, ModelName: input.ModelName, ToolNames: names, ToolRevision: r.requestToolPool(ctx, input.Terminal).Revision, ServerTools: client.SupportsResponsesServerTools() && !internalToolMatched}}
	return response
}

// ValidateApproval 在消费业务审批前检查当前工具修订与有效权限，跨端拒绝不会消耗许可。
func (r *Runtime) ValidateApproval(ctx context.Context, state RunState, calls []ApprovalCall) error {
	pool := r.requestToolPool(ctx, state.Terminal)
	if state.ToolRevision != pool.Revision {
		return fmt.Errorf("agent tool configuration changed")
	}
	input := RuntimeInput{Terminal: state.Terminal, ProviderID: state.ProviderID, ModelName: state.ModelName}
	infos := r.enabledToolInfos(ctx, input, r.allToolInfos(ctx, input))
	for _, call := range calls {
		if !slices.Contains(state.ToolNames, call.Name) || !tool.HasInfo(infos, call.Name) {
			return fmt.Errorf("pending agent tool is no longer authorized")
		}
	}
	return nil
}
