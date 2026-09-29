package base

import (
	"context"

	"github.com/go-kratos/kratos/v3/log"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-core/errorsx"
)

// AiKnowledgeService 查询 AI 知识库选项。
type AiKnowledgeService struct {
	basev1.UnimplementedAiKnowledgeServiceServer
	aiKnowledgeCase *biz.AiKnowledgeCase
}

// NewAiKnowledgeService 创建 AI 知识库选项服务。
func NewAiKnowledgeService(aiKnowledgeCase *biz.AiKnowledgeCase) *AiKnowledgeService {
	return &AiKnowledgeService{aiKnowledgeCase: aiKnowledgeCase}
}

// ListAiKnowledgeOptions 查询可用的 AI 知识库选项。
func (s *AiKnowledgeService) ListAiKnowledgeOptions(ctx context.Context, _ *basev1.ListAiKnowledgeOptionsRequest) (*basev1.ListAiKnowledgeOptionsResponse, error) {
	response, err := s.aiKnowledgeCase.ListAiKnowledgeOptions(ctx)
	if err != nil {
		log.Error("ListAiKnowledgeOptions", "error", err)
		return nil, errorsx.WrapInternal(err, "查询AI知识库选项失败")
	}
	return response, nil
}
