package workflow

import (
	"context"
	"fmt"
)

// BindingVerifier 从宿主持久化存储核验单据已关联指定实例且允许开始审批。
// 仅查询状态，不改变单据；返回 false 时引擎等待后续重试。
type BindingVerifier func(ctx context.Context, tenantID, instanceID, businessID int64) (bool, error)

// RegisterBindingVerifier 注册业务单据关联核验器，用于恢复提交与首次推进之间的中断。
func (r *EventRegistry) RegisterBindingVerifier(businessType string, verifier BindingVerifier) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.bindings == nil {
		r.bindings = make(map[string]BindingVerifier)
	}
	r.bindings[businessType] = verifier
}

// HasBindingVerifier 检查业务类型是否具备持久化关联核验能力。
func (r *EventRegistry) HasBindingVerifier(businessType string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.bindings[businessType] != nil
}

// VerifyBinding 核验单据关联，将宿主异常转换为可记录、可重试的错误。
func (r *EventRegistry) VerifyBinding(ctx context.Context, businessType string, tenantID, instanceID, businessID int64) (confirmed bool, err error) {
	r.mu.RLock()
	verifier := r.bindings[businessType]
	r.mu.RUnlock()
	if verifier == nil {
		return false, fmt.Errorf("业务类型 %s 未注册流程关联核验器", businessType)
	}
	defer func() {
		if value := recover(); value != nil {
			err = fmt.Errorf("流程关联核验异常: %v", value)
		}
	}()
	return verifier(ctx, tenantID, instanceID, businessID)
}
