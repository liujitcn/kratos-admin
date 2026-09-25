package ai

import (
	"context"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	"github.com/liujitcn/kratos-admin/backend/pkg/agent"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/resource/i18n"
)

// Runtime 是面向 Admin 业务的 AI 运行时适配器。
//
// 通用模型和工具执行能力由 pkg/agent 提供；本层只额外维护 Admin 的固定流程注册表。
type Runtime struct {
	*agent.Runtime
	fixedFlows      fixedFlowRegistry
	localizeMessage func(context.Context, string, map[string]any, string) string
}

// NewRuntime 创建 Admin AI 运行时。
func NewRuntime(
	client *model.AssistantClient,
	checker ToolAccessChecker,
	adminTools AdminTools,
	appTools AppTools,
	catalog *i18n.I18n,
) *Runtime {
	localizeMessage := func(ctx context.Context, key string, args map[string]any, fallback string) string {
		return catalog.Localize(biz.LocaleFromContext(ctx), "", key, args, fallback)
	}
	return &Runtime{
		Runtime: agent.NewRuntime(agent.RuntimeConfig{
			Client:          client,
			Checker:         checker,
			AdminTools:      adminTools,
			AppTools:        appTools,
			LocalizeMessage: localizeMessage,
		}),
		fixedFlows:      fixedFlowRegistry{flowNames: make(map[string]struct{})},
		localizeMessage: localizeMessage,
	}
}

// LocalizeMessage 按当前请求语言本地化助手面向用户的提示。
func (r *Runtime) LocalizeMessage(ctx context.Context, key string, args map[string]any, fallback string) string {
	if r == nil || r.localizeMessage == nil {
		return fallback
	}
	return r.localizeMessage(ctx, key, args, fallback)
}
