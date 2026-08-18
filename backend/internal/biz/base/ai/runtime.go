package ai

import (
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	"github.com/liujitcn/kratos-admin/backend/pkg/agent"
)

// Runtime 是面向 Admin 业务的 AI 运行时适配器。
//
// 通用模型和工具执行能力由 pkg/agent 提供；本层只额外维护 Admin 的固定流程注册表。
type Runtime struct {
	*agent.Runtime
	fixedFlows fixedFlowRegistry
}

// NewRuntime 创建 Admin AI 运行时。
func NewRuntime(
	client *model.ResponsesClient,
	checker ToolAccessChecker,
	adminTools AdminTools,
	appTools AppTools,
) *Runtime {
	return &Runtime{
		Runtime: agent.NewRuntime(agent.RuntimeConfig{
			Client:     client,
			Checker:    checker,
			AdminTools: adminTools,
			AppTools:   appTools,
		}),
		fixedFlows: fixedFlowRegistry{flowNames: make(map[string]struct{})},
	}
}
