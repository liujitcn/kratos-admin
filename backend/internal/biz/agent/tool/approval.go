package tool

import (
	"context"
	"encoding/gob"
	"fmt"

	"github.com/cloudwego/eino/compose"
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

// guardedTool 将权限和人工确认绑定到实际工具执行入口。
type guardedTool struct {
	Invokable
	prepare        PrepareCall
	allowInterrupt bool
}

// init 注册 checkpoint 中使用的审批状态类型。
func init() {
	gob.Register(ApprovalInfo{})
}

// NewGuardedTool 包装工具；直接调用不支持持久化审批时拒绝执行写操作。
func NewGuardedTool(value Invokable, prepare PrepareCall, allowInterrupt bool) Invokable {
	return &guardedTool{Invokable: value, prepare: prepare, allowInterrupt: allowInterrupt}
}

// InvokableRun 校验最新权限与冻结参数，只有指定中断点的明确确认才允许写入。
func (t *guardedTool) InvokableRun(ctx context.Context, arguments string, options ...Option) (string, error) {
	var requiresApproval bool
	var err error
	arguments, requiresApproval, err = t.prepare(ctx, arguments)
	if err != nil {
		return "", err
	}
	isTarget, hasData, decision := compose.GetResumeContext[ApprovalDecision](ctx)
	_, hasState, frozen := compose.GetInterruptState[ApprovalInfo](ctx)
	if !requiresApproval && !isTarget && !hasState {
		return t.Invokable.InvokableRun(ctx, arguments, options...)
	}
	if !t.allowInterrupt {
		return "", fmt.Errorf("tool requires human approval through an agent run")
	}
	var info *Info
	info, err = t.Info(ctx)
	if err != nil {
		return "", err
	}
	approval := ApprovalInfo{Name: info.Name, Title: Title(info), Arguments: arguments}
	if !isTarget || !hasData {
		return "", compose.StatefulInterrupt(ctx, approval, approval)
	}
	if !hasState || frozen != approval || decision.ApprovalInfo != frozen {
		return "", fmt.Errorf("approval does not match the pending tool call")
	}
	if !decision.Approved {
		return `{"denied":true,"message":"The user rejected this tool call. Do not retry it."}`, nil
	}
	return t.Invokable.InvokableRun(ctx, arguments, options...)
}
