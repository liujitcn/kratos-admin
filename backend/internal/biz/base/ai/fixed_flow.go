package ai

import (
	"context"
	"fmt"
	"sync"

	"github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
)

// FixedFlowProvider 提供模块私有的固定流程、入口校验和快捷入口。
type FixedFlowProvider interface {
	// FlowNames 返回该模块声明的稳定流程标识，用于在启动阶段检测冲突。
	FlowNames() []string
	// GenerateFixedFlowReply 尝试生成固定流程回复；handled 为 false 时继续由通用聊天处理。
	GenerateFixedFlowReply(context.Context, *Runtime, int32, string, *basev1.AiAction) (*Response, bool, error)
	// IsFixedFlowEntryAction 判断动作是否为该模块流程的入口。
	IsFixedFlowEntryAction(int32, string, string) bool
	// FixedFlowShortcuts 返回当前终端可展示的模块快捷入口。
	FixedFlowShortcuts(int32, map[string]bool) []*basev1.AiShortcut
}

// fixedFlowRegistry 聚合当前组合根显式启用的固定流程提供者。
type fixedFlowRegistry struct {
	mu        sync.RWMutex
	providers []FixedFlowProvider
	flowNames map[string]struct{}
}

// RegisterFixedFlow 注册模块固定流程提供者，并拒绝重复流程标识。
func (r *Runtime) RegisterFixedFlow(provider FixedFlowProvider) error {
	if r == nil {
		return fmt.Errorf("AI助手运行时未初始化")
	}
	if provider == nil {
		return fmt.Errorf("固定流程提供者不能为空")
	}
	flowNames := provider.FlowNames()
	if len(flowNames) == 0 {
		return fmt.Errorf("固定流程提供者未声明流程")
	}

	r.fixedFlows.mu.Lock()
	defer r.fixedFlows.mu.Unlock()
	if r.fixedFlows.flowNames == nil {
		r.fixedFlows.flowNames = make(map[string]struct{})
	}
	registered := make(map[string]struct{}, len(flowNames))
	for _, name := range flowNames {
		if name == "" {
			return fmt.Errorf("固定流程标识不能为空")
		}
		if _, exists := r.fixedFlows.flowNames[name]; exists {
			return fmt.Errorf("固定流程标识重复: %s", name)
		}
		if _, exists := registered[name]; exists {
			return fmt.Errorf("固定流程标识重复: %s", name)
		}
		registered[name] = struct{}{}
	}
	for name := range registered {
		r.fixedFlows.flowNames[name] = struct{}{}
	}
	r.fixedFlows.providers = append(r.fixedFlows.providers, provider)
	return nil
}

// GenerateFixedFlowReply 由已启用模块依次尝试处理固定流程请求。
func (r *Runtime) GenerateFixedFlowReply(ctx context.Context, terminal int32, content string, action *basev1.AiAction) (*Response, bool, error) {
	for _, provider := range r.fixedFlowProviders() {
		reply, handled, err := provider.GenerateFixedFlowReply(ctx, r, terminal, content, action)
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
		shortcuts = append(shortcuts, provider.FixedFlowShortcuts(terminal, enabledTools)...)
	}
	return shortcuts
}

// fixedFlowProviders 返回注册表快照，避免调用模块代码时持有锁。
func (r *Runtime) fixedFlowProviders() []FixedFlowProvider {
	if r == nil {
		return nil
	}
	r.fixedFlows.mu.RLock()
	providers := append([]FixedFlowProvider(nil), r.fixedFlows.providers...)
	r.fixedFlows.mu.RUnlock()
	return providers
}
