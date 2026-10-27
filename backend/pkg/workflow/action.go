package workflow

import (
	"context"
	"fmt"
)

// ActionNotify 是通过现有站内信模块发送幂等通知的内置动作。
const ActionNotify = "workflow.notify"

// ActionContext 提供业务动作需要核验的租户、单据关联和稳定幂等键。
type ActionContext struct {
	TenantID       int64
	InstanceID     int64
	BusinessID     int64
	BusinessType   string
	Status         int32
	IdempotencyKey string
	Payload        map[string]any
}

// ActionHandler 执行宿主显式注册的业务动作，必须按幂等键避免重复副作用。
type ActionHandler func(context.Context, ActionContext) error

// RegisterAction 注册某业务类型允许调用的脚本动作，重复注册会替换处理器。
func (r *EventRegistry) RegisterAction(businessType, actionType string, handler ActionHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.actions == nil {
		r.actions = make(map[string]map[string]ActionHandler)
	}
	if r.actions[businessType] == nil {
		r.actions[businessType] = make(map[string]ActionHandler)
	}
	r.actions[businessType][actionType] = handler
}

// ExecuteAction 执行白名单内的业务动作并将异常转换为可重试错误。
func (r *EventRegistry) ExecuteAction(ctx context.Context, actionType string, input ActionContext) (err error) {
	r.mu.RLock()
	handler := r.actions[input.BusinessType][actionType]
	r.mu.RUnlock()
	if handler == nil {
		return fmt.Errorf("业务类型 %s 未注册脚本动作 %s", input.BusinessType, actionType)
	}
	defer func() {
		if value := recover(); value != nil {
			err = fmt.Errorf("业务动作异常: %v", value)
		}
	}()
	return handler(ctx, input)
}
