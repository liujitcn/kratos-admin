package biz

import (
	"context"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-core/biz"
)

// AiKnowledgeCase 提供 AI 知识库选项查询能力。
type AiKnowledgeCase struct {
	*biz.BaseCase
	engine *ai.KnowledgeEngine
}

// NewAiKnowledgeCase 创建 AI 知识库选项业务实例。
func NewAiKnowledgeCase(baseCase *biz.BaseCase, engine *ai.KnowledgeEngine) *AiKnowledgeCase {
	return &AiKnowledgeCase{BaseCase: baseCase, engine: engine}
}

// ListAiKnowledgeOptions 查询可用知识库选项；数据源未配置时返回空列表。
func (c *AiKnowledgeCase) ListAiKnowledgeOptions(ctx context.Context) (*basev1.ListAiKnowledgeOptionsResponse, error) {
	options := make([]*basev1.AiKnowledgeOption, 0)
	if c == nil || !c.engine.Available() {
		return &basev1.ListAiKnowledgeOptionsResponse{Options: options}, nil
	}
	items, err := c.engine.ListAllKnowledge(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		options = append(options, &basev1.AiKnowledgeOption{
			Id:       item.ID,
			Name:     item.Name,
			DocCount: int32(item.DocCount),
		})
	}
	return &basev1.ListAiKnowledgeOptionsResponse{Options: options}, nil
}
