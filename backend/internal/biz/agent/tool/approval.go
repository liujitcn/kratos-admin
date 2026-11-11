package tool

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// ApprovalInfo 是保存在 checkpoint 中的实际工具与冻结参数。
type ApprovalInfo struct {
	// Name 是工具的唯一名称。
	Name string `json:"name"`
	// Title 是展示给用户的操作说明。
	Title string `json:"title"`
	// Arguments 是经过校验和租户补全的冻结参数。
	Arguments string `json:"arguments"`
}

// ApprovalCall 是一个需要用户决策的 Eino 中断点。
type ApprovalCall struct {
	ApprovalInfo
	// InterruptID 是 Eino 定向恢复的中断地址。
	InterruptID string `json:"interrupt_id"`
}

// ApprovalDecision 是由受信业务入口构造的单个工具审批结果。
type ApprovalDecision struct {
	ApprovalInfo
	// Approved 只由受信审批接口决定。
	Approved bool
}

// PrepareCall 在实际调用前重新授权、补全租户并校验参数。
type PrepareCall func(context.Context, string) (string, bool, error)

// guardedBase 统一冻结参数与审批状态，供标准和增强工具共用。
type guardedBase struct {
	Base
	prepare        PrepareCall
	allowInterrupt bool
	info           *Info
}

type guardedTool struct {
	*guardedBase
	value Invokable
}
type guardedEnhancedTool struct {
	*guardedBase
	value tool.EnhancedInvokableTool
}

// init 注册 checkpoint 中使用的审批状态类型。
func init() {
	gob.Register(ApprovalInfo{})
}

// NewGuardedTool 包装标准工具，保留已有标准调用接口。
func NewGuardedTool(value Invokable, prepare PrepareCall, allowInterrupt bool) Invokable {
	return NewGuardedBaseTool(value, prepare, allowInterrupt).(Invokable)
}

// NewGuardedBaseTool 以同一门禁包装标准或增强工具并覆盖有效业务定义。
func NewGuardedBaseTool(value Base, prepare PrepareCall, allowInterrupt bool, infos ...*Info) Base {
	base := &guardedBase{Base: value, prepare: prepare, allowInterrupt: allowInterrupt}
	if len(infos) > 0 {
		base.info = infos[0]
	}
	switch actual := value.(type) {
	case tool.EnhancedInvokableTool:
		return &guardedEnhancedTool{guardedBase: base, value: actual}
	case Invokable:
		return &guardedTool{guardedBase: base, value: actual}
	default:
		return base
	}
}

// Info 返回本轮授权后的有效工具定义。
func (t *guardedBase) Info(ctx context.Context) (*Info, error) {
	if t.info != nil {
		return t.info, nil
	}
	return t.Base.Info(ctx)
}

// InvokableRun 在授权和审批后执行标准工具。
func (t *guardedTool) InvokableRun(ctx context.Context, arguments string, options ...Option) (string, error) {
	prepared, denied, err := t.authorize(ctx, arguments)
	if err != nil {
		return "", err
	}
	if denied {
		return prepared, nil
	}
	var info *Info
	info, err = t.Info(ctx)
	if err != nil {
		return "", err
	}
	if err = beforeExecution(ctx, info.Name); err != nil {
		return "", err
	}
	var output string
	output, err = t.value.InvokableRun(ctx, prepared, options...)
	return output, errors.Join(err, afterExecution(ctx, info.Name, err))
}

// InvokableRun 在授权和审批后执行增强工具，保留全部结构化结果。
func (t *guardedEnhancedTool) InvokableRun(ctx context.Context, arguments *schema.ToolArgument, options ...Option) (*schema.ToolResult, error) {
	prepared, denied, err := t.authorize(ctx, arguments.Text)
	if err != nil {
		return nil, err
	}
	if denied {
		return &schema.ToolResult{Parts: []schema.ToolOutputPart{{Type: schema.ToolPartTypeText, Text: prepared}}}, nil
	}
	var info *Info
	info, err = t.Info(ctx)
	if err != nil {
		return nil, err
	}
	if err = beforeExecution(ctx, info.Name); err != nil {
		return nil, err
	}
	var output *schema.ToolResult
	output, err = t.value.InvokableRun(ctx, &schema.ToolArgument{Text: prepared}, options...)
	return output, errors.Join(err, afterExecution(ctx, info.Name, err))
}

// authorize 校验最新权限与冻结参数，只有指定中断点的明确确认才允许写入。
func (t *guardedBase) authorize(ctx context.Context, arguments string) (string, bool, error) {
	var requiresApproval bool
	var err error
	arguments, requiresApproval, err = t.prepare(ctx, arguments)
	if err != nil {
		return "", false, err
	}
	isTarget, hasData, decision := compose.GetResumeContext[ApprovalDecision](ctx)
	_, hasState, frozen := compose.GetInterruptState[ApprovalInfo](ctx)
	if !requiresApproval && !isTarget && !hasState {
		return arguments, false, nil
	}
	if !t.allowInterrupt {
		return "", false, fmt.Errorf("tool requires human approval through an agent run")
	}
	var info *Info
	info, err = t.Info(ctx)
	if err != nil {
		return "", false, err
	}
	approval := ApprovalInfo{Name: info.Name, Title: Title(info), Arguments: arguments}
	if !isTarget || !hasData {
		return "", false, compose.StatefulInterrupt(ctx, approval, approval)
	}
	if !hasState || frozen != approval || decision.ApprovalInfo != frozen {
		return "", false, fmt.Errorf("approval does not match the pending tool call")
	}
	if !decision.Approved {
		return `{"denied":true,"message":"The user rejected this tool call. Do not retry it."}`, true, nil
	}
	return arguments, false, nil
}
