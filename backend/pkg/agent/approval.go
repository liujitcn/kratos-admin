package agent

import (
	"context"
	"fmt"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/adk"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/callback"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/tool"
)

// Resume 沿用原始终端和模型上下文恢复工具调用，并在执行入口重新检查当前权限与工具开关。
func (r *Runtime) Resume(ctx context.Context, checkpointID string, state RunState, calls []ApprovalCall, approved bool) (*Response, error) {
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
	if allowed[agentToolCatalogName] {
		catalog := r.newAgentToolCatalogTool(ctx, state.Terminal, r.allToolInfos(ctx, input), infos, maxModelToolsPerRequest)
		var info *tool.Info
		info, err = catalog.Info(ctx)
		if err != nil {
			return nil, err
		}
		selected = appendCatalogInfo(selected, info)
	}
	decisions := make(map[string]any, len(calls))
	for _, call := range calls {
		decisions[call.InterruptID] = tool.ApprovalDecision{ApprovalInfo: call.ApprovalInfo, Approved: approved}
	}
	recorder := &callback.Recorder{}
	runner := adk.NewRunner(adk.Config{Model: client.AgenticModel, Name: "admin_ai", ServerTools: state.ServerTools, CheckPointStore: r.checkPointStore})
	result, err := runner.Run(ctx, adk.Request{CheckPointID: checkpointID, Decisions: decisions, Tools: r.runnerTools(ctx, input), ToolInfos: selected, Recorder: recorder, LocalizeMessage: r.localizeMessage})
	if err != nil {
		return nil, err
	}
	return r.responseFromRun(input, client, selected, !state.ServerTools, result, recorder), nil
}

// responseFromRun 将工具中断与最终回复统一转换为 Runtime 输出。
func (r *Runtime) responseFromRun(input RuntimeInput, client *model.AssistantClient, infos []*tool.Info, internalToolMatched bool, result *adk.Result, recorder *callback.Recorder) *Response {
	response := r.buildResponse(result.Message, tokenFromCallback(result.Token), toolsFromRecorder(recorder), client.Name())
	if len(result.Pending) == 0 {
		return response
	}
	names := make([]string, 0, len(infos))
	for _, info := range infos {
		names = append(names, info.Name)
	}
	response.Content = ""
	response.Pending = &PendingApproval{Calls: result.Pending, State: RunState{Terminal: input.Terminal, ProviderID: input.ProviderID, ModelName: input.ModelName, ToolNames: names, ServerTools: client.SupportsResponsesServerTools() && !internalToolMatched}}
	return response
}
