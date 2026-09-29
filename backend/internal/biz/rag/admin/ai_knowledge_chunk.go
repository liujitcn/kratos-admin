package biz

import (
	"context"
	"errors"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/rag/admin/v1"
	ai "github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	"gorm.io/gorm"
)

// AiKnowledgeChunkCase 提供AI知识库切片检索能力。
type AiKnowledgeChunkCase struct {
	*biz.BaseCase
	engine *ai.KnowledgeEngine
}

// NewAiKnowledgeChunkCase 创建AI知识库切片检索业务实例。
func NewAiKnowledgeChunkCase(baseCase *biz.BaseCase, engine *ai.KnowledgeEngine) *AiKnowledgeChunkCase {
	return &AiKnowledgeChunkCase{BaseCase: baseCase, engine: engine}
}

// SearchAiKnowledge 检索测试：返回与query语义最相近的切片。
func (c *AiKnowledgeChunkCase) SearchAiKnowledge(ctx context.Context, req *adminv1.SearchAiKnowledgeRequest) (*adminv1.SearchAiKnowledgeResponse, error) {
	if _, err := c.engine.GetKnowledge(ctx, req.GetId()); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("AI知识库不存在")
		}
		return nil, err
	}
	hits, err := c.engine.Search(ctx, req.GetId(), req.GetQuery(), int(req.GetTopK()))
	if err != nil {
		return nil, err
	}
	records := make([]*adminv1.AiKnowledgeChunkHit, 0, len(hits))
	for _, hit := range hits {
		records = append(records, &adminv1.AiKnowledgeChunkHit{
			ChunkId: hit.ChunkID,
			DocId:   hit.DocID,
			DocName: hit.DocName,
			Content: hit.Content,
			Score:   float32(hit.Score),
		})
	}
	return &adminv1.SearchAiKnowledgeResponse{Hits: records}, nil
}
