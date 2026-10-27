// 工作流业务事件注册表，供引擎与业务单据模块解耦联动。
package workflow

import (
	"context"
	"sync"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
)

// BusinessHook 业务单据审批终结回调。status 为实例终态（已通过/已驳回/已撤回）。
// 回调实现须自行校验业务单据与实例的关联（business_id 对应单据的实例 ID 应等于
// instanceID），防止伪造 business 字段触发他人单据副作用；失败在回调内自行记日志。
type BusinessHook func(ctx context.Context, tenantID, instanceID, businessID int64, status adminv1.WorkflowInstanceStatus)

// EventRegistry 按业务单据类型注册的审批终结回调表。业务模块构造时注册，引擎终结时同步回调。
type EventRegistry struct {
	mu       sync.RWMutex
	hooks    map[string]BusinessHook
	actions  map[string]map[string]ActionHandler
	bindings map[string]BindingVerifier
}

// NewEventRegistry 创建审批终结回调表。
func NewEventRegistry() *EventRegistry {
	return &EventRegistry{hooks: make(map[string]BusinessHook)}
}

// defaultRegistry 进程级共享回调表：引擎与外部业务模块（含宿主项目）须经同一实例注册与触发。
var defaultRegistry = NewEventRegistry()

// DefaultEventRegistry 返回进程级共享的审批终结回调表。
// 宿主项目在自身模块初始化时通过它注册业务回调；Admin 引擎终结实例时经同一实例触发。
func DefaultEventRegistry() *EventRegistry {
	return defaultRegistry
}

// Register 注册某业务类型的终结回调，同名类型重复注册以后者覆盖。
func (r *EventRegistry) Register(businessType string, hook BusinessHook) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hooks[businessType] = hook
}

// Fire 触发业务单据的终结回调。businessType 为空或未注册时静默跳过；
// panic 由调用方（引擎侧）recover 隔离，不影响状态机结果。
func (r *EventRegistry) Fire(ctx context.Context, tenantID, instanceID, businessID int64, businessType string, status adminv1.WorkflowInstanceStatus) {
	if businessType == "" || businessID == 0 {
		return
	}
	r.mu.RLock()
	hook, ok := r.hooks[businessType]
	r.mu.RUnlock()
	if !ok {
		return
	}
	hook(ctx, tenantID, instanceID, businessID, status)
}
