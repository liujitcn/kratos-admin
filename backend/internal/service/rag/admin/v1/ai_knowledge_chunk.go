package admin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/rag/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/rag/admin"
	"github.com/liujitcn/kratos-core/errorsx"
)

// AiKnowledgeChunkService AI知识库切片检索服务。
type AiKnowledgeChunkService struct {
	adminv1.UnimplementedAiKnowledgeChunkServiceServer
	aiKnowledgeChunkCase *biz.AiKnowledgeChunkCase
}

// NewAiKnowledgeChunkService 创建AI知识库切片检索服务。
func NewAiKnowledgeChunkService(aiKnowledgeChunkCase *biz.AiKnowledgeChunkCase) *AiKnowledgeChunkService {
	return &AiKnowledgeChunkService{aiKnowledgeChunkCase: aiKnowledgeChunkCase}
}

// SearchAiKnowledge 检索测试。
func (s *AiKnowledgeChunkService) SearchAiKnowledge(ctx context.Context, req *adminv1.SearchAiKnowledgeRequest) (*adminv1.SearchAiKnowledgeResponse, error) {
	result, err := s.aiKnowledgeChunkCase.SearchAiKnowledge(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SearchAiKnowledge %v", err))
		return nil, errorsx.WrapInternal(err, "AI知识库检索失败")
	}
	return result, nil
}
