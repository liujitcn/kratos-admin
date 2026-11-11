package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
)

// RetrieveStrict 对自动化的显式知识库依赖逐一检索，任何失效均返回错误。
func (e *KnowledgeEngine) RetrieveStrict(ctx context.Context, ids []int64, query string) (string, error) {
	var content strings.Builder
	for _, id := range ids {
		knowledge, err := e.GetKnowledge(ctx, id)
		if err != nil {
			return "", fmt.Errorf("知识库不可用: %w", err)
		}
		var provider *models.AiProvider
		var aiModel *models.AiModel
		provider, aiModel, err = e.loadEmbeddingModel(ctx, knowledge.ModelID)
		if err != nil {
			return "", err
		}
		if provider.Status != 1 || aiModel.Status != 1 {
			return "", fmt.Errorf("知识库模型或供应商未启用")
		}
		var hits []*ChunkHit
		hits, err = e.Search(ctx, id, query, defaultTopK)
		if err != nil {
			return "", fmt.Errorf("知识库检索失败: %w", err)
		}
		fmt.Fprintf(&content, "【知识库：%s】\n", knowledge.Name)
		for _, hit := range hits {
			fmt.Fprintf(&content, "- （%s）%s\n", hit.DocName, hit.Content)
		}
	}
	return content.String(), nil
}
