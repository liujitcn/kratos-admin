// 工作流引擎对外公共能力：宿主项目经此提交审批、查询单据关联实例状态。
package workflow

import (
	"context"
	"errors"
	"sync/atomic"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
)

// SubmitParams 外部业务模块提交审批申请的参数。
type SubmitParams struct {
	// TenantID 是所属租户编号，0 表示取请求上下文登录租户。
	TenantID int64
	// ApplicantUserID 是申请人用户编号，0 表示取请求上下文登录用户。
	ApplicantUserID int64
	// Code 是流程编码。
	Code string
	// Version 是流程版本。
	Version int32
	// FormData 是申请表单数据 JSON 文本。
	FormData string
	// BusinessType 是业务单据类型编码，用于终结回调分发。
	BusinessType string
	// BusinessID 是业务单据编号。
	BusinessID int64
	// OnCreated 在实例创建提交后、首次推进前保存宿主单据与实例的关联。
	// 使用回调前必须为业务类型注册 BindingVerifier，恢复任务以持久化关联判断是否允许推进。
	// 宿主自行提交回写；返回错误时终结未推进实例且不触发业务回调。
	OnCreated func(ctx context.Context, instanceID int64) error
}

// InstanceStatus 是业务单据关联流程实例的状态摘要。
type InstanceStatus struct {
	// InstanceID 是流程实例编号。
	InstanceID int64
	// Status 是实例状态。
	Status adminv1.WorkflowInstanceStatus
	// EndAt 是终结时间展示文本，未终结为空串。
	EndAt string
}

// Engine 定义外部业务模块执行工作流的能力。
type Engine interface {
	// Submit 创建实例并执行 OnCreated 关联回写，成功后推进并返回流程实例编号。
	Submit(ctx context.Context, params SubmitParams) (int64, error)
	// StatusByBusiness 查询业务单据最新关联实例的状态摘要。
	StatusByBusiness(ctx context.Context, tenantID int64, businessType string, businessID int64) (InstanceStatus, error)
}

// defaultEngineHolder 保存模块装配时注入的引擎实现；包装一层保证 atomic.Value 类型一致。
var defaultEngineHolder atomic.Value

// engineHolder 引擎包装器。
type engineHolder struct {
	engine Engine
}

// SetDefaultEngine 注入默认引擎实现，由 Admin 模块装配时调用。
func SetDefaultEngine(engine Engine) {
	defaultEngineHolder.Store(&engineHolder{engine: engine})
}

// DefaultEngine 返回默认引擎实现；未装配时返回 nil。
func DefaultEngine() Engine {
	if holder, ok := defaultEngineHolder.Load().(*engineHolder); ok {
		return holder.engine
	}
	return nil
}

// errEngineNotReady 引擎未装配时返回的错误。
var errEngineNotReady = errors.New("工作流引擎未初始化")

// Submit 经默认引擎提交审批申请。
func Submit(ctx context.Context, params SubmitParams) (int64, error) {
	engine := DefaultEngine()
	if engine == nil {
		return 0, errEngineNotReady
	}
	return engine.Submit(ctx, params)
}

// StatusByBusiness 经默认引擎查询业务单据关联实例状态。
func StatusByBusiness(ctx context.Context, tenantID int64, businessType string, businessID int64) (InstanceStatus, error) {
	engine := DefaultEngine()
	if engine == nil {
		return InstanceStatus{}, errEngineNotReady
	}
	return engine.StatusByBusiness(ctx, tenantID, businessType, businessID)
}
