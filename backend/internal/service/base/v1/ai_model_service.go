package base

import (
	"context"

	"github.com/go-kratos/kratos/v3/log"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-core/errorsx"
)

// AiModelService 查询 AI Provider 模型选项。
type AiModelService struct {
	basev1.UnimplementedAiModelServiceServer
	aiModelCase *biz.AiModelCase
}

// NewAiModelService 创建 AI Provider 模型选项服务。
func NewAiModelService(aiModelCase *biz.AiModelCase) *AiModelService {
	return &AiModelService{aiModelCase: aiModelCase}
}

// ListAiProviderModelOptions 查询已启用的 Provider 及其模型。
func (s *AiModelService) ListAiProviderModelOptions(ctx context.Context, _ *basev1.ListAiProviderModelOptionsRequest) (*basev1.ListAiProviderModelOptionsResponse, error) {
	response, err := s.aiModelCase.ListAiProviderModelOptions(ctx)
	if err != nil {
		log.Error("ListAiProviderModelOptions", "error", err)
		return nil, errorsx.WrapInternal(err, "查询AI Provider模型选项失败")
	}
	return response, nil
}
