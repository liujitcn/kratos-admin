package ai

import (
	"context"
	"sync"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	coremodule "github.com/liujitcn/kratos-core/pkg/module"
)

// FixedFlowProvider 是宿主 AI 固定流程契约的内部实现别名。
type FixedFlowProvider = coremodule.AIFixedFlowProvider

// fixedFlowRegistry 聚合当前组合根显式启用的固定流程提供者。
type fixedFlowRegistry struct {
	mu        sync.RWMutex
	providers []coremodule.AIFixedFlowProvider
}

// GenerateFixedFlowReply 由已启用模块依次尝试处理固定流程请求。
func (r *Runtime) GenerateFixedFlowReply(ctx context.Context, terminal int32, content string, action *basev1.AiAction) (*Response, bool, error) {
	coreAction := toCoreAIAction(action)
	for _, provider := range r.fixedFlowProviders() {
		reply, handled, err := provider.GenerateFixedFlowReply(ctx, r, terminal, content, coreAction)
		if handled || err != nil {
			return reply, handled, err
		}
	}
	return nil, false, nil
}

// IsFixedFlowEntryAction 判断动作是否为已启用模块声明的固定流程入口。
func (r *Runtime) IsFixedFlowEntryAction(terminal int32, flow string, actionType string) bool {
	for _, provider := range r.fixedFlowProviders() {
		if provider.IsFixedFlowEntryAction(terminal, flow, actionType) {
			return true
		}
	}
	return false
}

// FixedFlowShortcuts 汇总当前终端可展示的模块快捷入口。
func (r *Runtime) FixedFlowShortcuts(terminal int32, enabledTools map[string]bool) []*basev1.AiShortcut {
	shortcuts := make([]*basev1.AiShortcut, 0)
	for _, provider := range r.fixedFlowProviders() {
		for _, shortcut := range provider.FixedFlowShortcuts(terminal, enabledTools) {
			shortcuts = append(shortcuts, toProtoAIShortcut(shortcut))
		}
	}
	return shortcuts
}

// fixedFlowProviders 返回注册表快照，避免调用模块代码时持有锁。
func (r *Runtime) fixedFlowProviders() []coremodule.AIFixedFlowProvider {
	if r == nil {
		return nil
	}
	r.fixedFlows.mu.RLock()
	providers := append([]FixedFlowProvider(nil), r.fixedFlows.providers...)
	r.fixedFlows.mu.RUnlock()
	return providers
}

// toCoreAIAction 将接口层动作转换为 Core 的模块契约。
func toCoreAIAction(action *basev1.AiAction) *coremodule.AIAction {
	if action == nil {
		return nil
	}
	return &coremodule.AIAction{
		Flow:            action.GetFlow(),
		Step:            action.GetStep(),
		Type:            action.GetType(),
		PayloadJSON:     action.GetPayloadJson(),
		SourceMessageID: action.GetSourceMessageId(),
		ActionID:        action.GetActionId(),
		FlowVersion:     action.GetFlowVersion(),
	}
}

// toProtoAIShortcut 将 Core 模块快捷入口转换为接口层对象。
func toProtoAIShortcut(shortcut *coremodule.AIShortcut) *basev1.AiShortcut {
	if shortcut == nil {
		return nil
	}
	result := &basev1.AiShortcut{
		Key:           shortcut.Key,
		Title:         shortcut.Title,
		Prompt:        shortcut.Prompt,
		RequiredTools: append([]string(nil), shortcut.RequiredTools...),
		Sort:          shortcut.Sort,
		Group:         shortcut.Group,
	}
	if shortcut.Action != nil {
		result.Action = &basev1.AiShortcutAction{
			Flow:        shortcut.Action.Flow,
			Step:        shortcut.Action.Step,
			Type:        shortcut.Action.Type,
			PayloadJson: shortcut.Action.PayloadJSON,
		}
	}
	return result
}
