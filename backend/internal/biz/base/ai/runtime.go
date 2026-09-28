package ai

import (
	"context"
	"errors"

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
	modelRegistry   *model.Registry
	fixedFlows      fixedFlowRegistry
	localizeMessage func(context.Context, string, map[string]any, string) string
}

// NewRuntime 创建 Admin AI 运行时。
func NewRuntime(
	modelRegistry *model.Registry,
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
			ClientResolver:  modelRegistry,
			Checker:         checker,
			AdminTools:      adminTools,
			AppTools:        appTools,
			LocalizeMessage: localizeMessage,
		}),
		modelRegistry:   modelRegistry,
		fixedFlows:      fixedFlowRegistry{flowNames: make(map[string]struct{})},
		localizeMessage: localizeMessage,
	}
}

// ModelOptions 返回当前启用的供应商与模型选项。
func (r *Runtime) ModelOptions() []model.ModelOption {
	if r == nil || r.modelRegistry == nil {
		return nil
	}
	return r.modelRegistry.ModelOptions()
}

// SelectModel 校验用户选择并在未指定时返回首个可用模型。
func (r *Runtime) SelectModel(providerID int64, modelName string) (model.ModelOption, error) {
	options := r.ModelOptions()
	if len(options) == 0 {
		if providerID == 0 && modelName == "" {
			return model.ModelOption{}, nil
		}
		return model.ModelOption{}, errors.New("AI供应商或模型未启用")
	}
	if providerID == 0 && modelName == "" {
		return options[0], nil
	}
	if providerID <= 0 || modelName == "" {
		return model.ModelOption{}, errors.New("AI供应商和模型选择不完整")
	}
	for _, option := range options {
		if option.ProviderID == providerID && option.ModelName == modelName {
			return option, nil
		}
	}
	return model.ModelOption{}, errors.New("AI供应商或模型未启用")
}

// SelectedModelName 返回供应商和模型对应的展示名称。
func (r *Runtime) SelectedModelName(providerID int64, modelName string) string {
	if r == nil || r.modelRegistry == nil {
		return ""
	}
	client, err := r.modelRegistry.ResolveAssistantClient(providerID, modelName)
	if err != nil {
		return ""
	}
	return client.Name()
}

// LocalizeMessage 按当前请求语言本地化助手面向用户的提示。
func (r *Runtime) LocalizeMessage(ctx context.Context, key string, args map[string]any, fallback string) string {
	if r == nil || r.localizeMessage == nil {
		return fallback
	}
	return r.localizeMessage(ctx, key, args, fallback)
}
