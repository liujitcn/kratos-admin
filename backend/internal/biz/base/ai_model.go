package biz

import (
	"context"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	ai "github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-core/biz"
)

// AiModelCase 提供 AI Provider 模型选项查询能力。
type AiModelCase struct {
	*biz.BaseCase
	aiRuntime *ai.Runtime
}

// NewAiModelCase 创建 AI Provider 模型选项业务实例。
func NewAiModelCase(baseCase *biz.BaseCase, aiRuntime *ai.Runtime) *AiModelCase {
	return &AiModelCase{BaseCase: baseCase, aiRuntime: aiRuntime}
}

// ListAiProviderModelOptions 查询已启用的 Provider 及其可用模型。
func (c *AiModelCase) ListAiProviderModelOptions(context.Context) (*basev1.ListAiProviderModelOptionsResponse, error) {
	providers := make([]*basev1.AiProviderModelOption, 0)
	providerIndexes := make(map[int64]int)
	if c == nil || c.aiRuntime == nil {
		return &basev1.ListAiProviderModelOptionsResponse{Providers: providers}, nil
	}
	for _, option := range c.aiRuntime.ModelOptions() {
		index, exists := providerIndexes[option.ProviderID]
		if !exists {
			index = len(providers)
			providerIndexes[option.ProviderID] = index
			providers = append(providers, &basev1.AiProviderModelOption{ProviderId: option.ProviderID, Name: option.ProviderName})
		}
		providers[index].Models = append(providers[index].Models, option.ModelName)
	}
	return &basev1.ListAiProviderModelOptionsResponse{Providers: providers}, nil
}
