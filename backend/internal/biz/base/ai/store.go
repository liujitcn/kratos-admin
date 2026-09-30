package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/rag/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/rag/models"
)

// 文档处理状态，取值对应 rag.admin.v1 的 AiKnowledgeDocStatus 枚举，
// 按存储类型（PG smallint）定义为 int16。
const (
	// DocStatusReady 表示文档切片与向量化全部完成。
	DocStatusReady = int32(adminv1.AiKnowledgeDocStatus_AI_KNOWLEDGE_DOC_STATUS_READY)
	// DocStatusFailed 表示文档处理失败。
	DocStatusFailed = int32(adminv1.AiKnowledgeDocStatus_AI_KNOWLEDGE_DOC_STATUS_FAILED)
)

// DefaultEmbeddingDimensions 是知识库未显式配置维度时的兜底维度。
// 维度本体存放在 ai_knowledge.embedding_dimensions，随所选 embedding 模型而定。
const DefaultEmbeddingDimensions = 1536

// Knowledge AI 知识库表模型（生成类型别名）。
type Knowledge = models.AiKnowledge

// Doc AI 知识库文档表模型（生成类型别名）。
type Doc = models.AiKnowledgeDoc

// Chunk AI 知识库切片表模型（生成类型别名）。
type Chunk = models.AiKnowledgeChunk

// ChunkContent 表示待入库的切片及其向量。
type ChunkContent struct {
	// Index 切片序号。
	Index int
	// Content 切片文本。
	Content string
	// Vector 切片 embedding 向量。
	Vector []float32
}

// ChunkHit 检索命中的切片。
type ChunkHit struct {
	ChunkID int64
	DocID   int64
	DocName string
	Content string
	Score   float64
}

// Option 聊天选择器使用的知识库摘要。
type Option struct {
	ID       int64
	Name     string
	DocCount int
}

// Available 表示 rag 数据源是否已配置、知识库能力是否可用。
func (e *KnowledgeEngine) Available() bool {
	return e != nil && e.available
}

// require 返回知识库能力不可用的统一错误。
func (e *KnowledgeEngine) require() error {
	if !e.Available() {
		return errors.New("AI知识库数据源未配置")
	}
	return nil
}

// ListKnowledge 分页查询未删除的知识库。
func (e *KnowledgeEngine) ListKnowledge(ctx context.Context, name string, pageNum, pageSize int64) ([]*Knowledge, int64, error) {
	if err := e.require(); err != nil {
		return nil, 0, err
	}
	query := e.knowledgeRepo.Query(ctx).AiKnowledge
	opts := make([]repository.QueryOption, 0, 2)
	if name != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+name+"%")))
	}
	opts = append(opts, repository.Order(query.UpdatedAt.Desc(), query.ID.Desc()))
	return e.knowledgeRepo.Page(ctx, pageNum, pageSize, opts...)
}

// ListAllKnowledge 查询全部未删除的知识库摘要（含文档数），供聊天选择器使用。
func (e *KnowledgeEngine) ListAllKnowledge(ctx context.Context) ([]*Option, error) {
	if err := e.require(); err != nil {
		return nil, err
	}
	items, err := e.knowledgeRepo.List(ctx, repository.Order(e.knowledgeRepo.Query(ctx).AiKnowledge.ID.Asc()))
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	counts := make(map[int64]int, len(items))
	if len(ids) > 0 {
		query := e.docRepo.Query(ctx).AiKnowledgeDoc
		rows := make([]struct {
			KnowledgeBaseID int64
			Count           int64
		}, 0, len(ids))
		// 文档数只统计已就绪文档：处理失败的文档不可用于检索，不能计入展示数量。
		if err = query.WithContext(ctx).
			Select(query.KnowledgeBaseID, query.ID.Count().As("count")).
			Where(query.KnowledgeBaseID.In(ids...), query.Status.Eq(DocStatusReady)).
			Group(query.KnowledgeBaseID).
			Scan(&rows); err != nil {
			return nil, err
		}
		for _, row := range rows {
			counts[row.KnowledgeBaseID] = int(row.Count)
		}
	}
	options := make([]*Option, 0, len(items))
	for _, item := range items {
		options = append(options, &Option{ID: item.ID, Name: item.Name, DocCount: counts[item.ID]})
	}
	return options, nil
}

// GetKnowledge 查询单个未删除的知识库。
func (e *KnowledgeEngine) GetKnowledge(ctx context.Context, id int64) (*Knowledge, error) {
	if err := e.require(); err != nil {
		return nil, err
	}
	return e.knowledgeRepo.FindByID(ctx, id)
}

// CreateKnowledge 创建知识库。
func (e *KnowledgeEngine) CreateKnowledge(ctx context.Context, item *Knowledge) error {
	if err := e.require(); err != nil {
		return err
	}
	return e.knowledgeRepo.Create(ctx, item)
}

// UpdateKnowledge 更新知识库可维护字段，允许清空描述。
func (e *KnowledgeEngine) UpdateKnowledge(ctx context.Context, item *Knowledge) error {
	if err := e.require(); err != nil {
		return err
	}
	query := e.knowledgeRepo.Query(ctx).AiKnowledge
	_, err := query.WithContext(ctx).
		Where(query.ID.Eq(item.ID)).
		Select(query.Name, query.Description, query.ModelID, query.UpdatedBy).
		Updates(map[string]any{
			"name":        item.Name,
			"description": item.Description,
			"model_id":    item.ModelID,
			"updated_by":  item.UpdatedBy,
		})
	return err
}

// DeleteKnowledge 删除知识库：硬删切片、软删文档与知识库。
func (e *KnowledgeEngine) DeleteKnowledge(ctx context.Context, id int64) error {
	if err := e.require(); err != nil {
		return err
	}
	return e.data.Transaction(ctx, func(txCtx context.Context) error {
		docQuery := e.docRepo.Query(txCtx).AiKnowledgeDoc
		docs, err := e.docRepo.List(txCtx, repository.Where(docQuery.KnowledgeBaseID.Eq(id)))
		if err != nil {
			return err
		}
		docIDs := make([]int64, 0, len(docs))
		for _, doc := range docs {
			docIDs = append(docIDs, doc.ID)
		}
		if len(docIDs) > 0 {
			chunkQuery := e.chunkRepo.Query(txCtx).AiKnowledgeChunk
			if err = e.chunkRepo.Delete(txCtx, repository.Where(chunkQuery.DocID.In(docIDs...))); err != nil {
				return err
			}
		}
		if err = e.docRepo.Delete(txCtx, repository.Where(docQuery.KnowledgeBaseID.Eq(id))); err != nil {
			return err
		}
		knowledgeQuery := e.knowledgeRepo.Query(txCtx).AiKnowledge
		return e.knowledgeRepo.Delete(txCtx, repository.Where(knowledgeQuery.ID.Eq(id)))
	})
}

// ListDocs 分页查询知识库下未删除的文档。
func (e *KnowledgeEngine) ListDocs(ctx context.Context, knowledgeBaseID, pageNum, pageSize int64) ([]*Doc, int64, error) {
	if err := e.require(); err != nil {
		return nil, 0, err
	}
	query := e.docRepo.Query(ctx).AiKnowledgeDoc
	opts := []repository.QueryOption{
		repository.Where(query.KnowledgeBaseID.Eq(knowledgeBaseID)),
		repository.Order(query.CreatedAt.Desc(), query.ID.Desc()),
	}
	return e.docRepo.Page(ctx, pageNum, pageSize, opts...)
}

// GetDoc 查询单个未删除的文档。
func (e *KnowledgeEngine) GetDoc(ctx context.Context, id int64) (*Doc, error) {
	if err := e.require(); err != nil {
		return nil, err
	}
	return e.docRepo.FindByID(ctx, id)
}

// CreateDoc 创建文档记录。
func (e *KnowledgeEngine) CreateDoc(ctx context.Context, item *Doc) error {
	if err := e.require(); err != nil {
		return err
	}
	return e.docRepo.Create(ctx, item)
}

// UpdateDocStatus 更新文档切片数与处理状态，支持状态字段清零。
func (e *KnowledgeEngine) UpdateDocStatus(ctx context.Context, id int64, chunkCount int32, status int32, errorMessage string) error {
	if err := e.require(); err != nil {
		return err
	}
	query := e.docRepo.Query(ctx).AiKnowledgeDoc
	_, err := query.WithContext(ctx).
		Where(query.ID.Eq(id)).
		Select(query.ChunkCount, query.Status, query.ErrorMessage).
		Updates(map[string]any{
			"chunk_count":   chunkCount,
			"status":        status,
			"error_message": errorMessage,
		})
	return err
}

// DeleteDoc 删除文档：硬删切片、软删文档。
func (e *KnowledgeEngine) DeleteDoc(ctx context.Context, id int64) error {
	if err := e.require(); err != nil {
		return err
	}
	return e.data.Transaction(ctx, func(txCtx context.Context) error {
		chunkQuery := e.chunkRepo.Query(txCtx).AiKnowledgeChunk
		if err := e.chunkRepo.Delete(txCtx, repository.Where(chunkQuery.DocID.Eq(id))); err != nil {
			return err
		}
		docQuery := e.docRepo.Query(txCtx).AiKnowledgeDoc
		return e.docRepo.Delete(txCtx, repository.Where(docQuery.ID.Eq(id)))
	})
}

// ReplaceChunks 重建文档切片：事务内先删旧切片，再批量写入新切片。
// embedding 向量序列化为 JSON 存入 TEXT 列。
func (e *KnowledgeEngine) ReplaceChunks(ctx context.Context, docID int64, chunks []ChunkContent) error {
	if err := e.require(); err != nil {
		return err
	}
	entities, err := e.buildChunkEntities(docID, chunks)
	if err != nil {
		return err
	}
	return e.data.Transaction(ctx, func(txCtx context.Context) error {
		query := e.chunkRepo.Query(txCtx).AiKnowledgeChunk
		if err := e.chunkRepo.Delete(txCtx, repository.Where(query.DocID.Eq(docID))); err != nil {
			return err
		}
		return e.chunkRepo.BatchCreate(txCtx, entities)
	})
}

// buildChunkEntities 将切片内容转换为带 JSON 向量的存储实体。
func (e *KnowledgeEngine) buildChunkEntities(docID int64, chunks []ChunkContent) ([]*Chunk, error) {
	entities := make([]*Chunk, 0, len(chunks))
	for _, chunk := range chunks {
		vector, err := marshalVector(chunk.Vector)
		if err != nil {
			return nil, err
		}
		entities = append(entities, &Chunk{
			DocID:      docID,
			ChunkIndex: int32(chunk.Index),
			Content:    chunk.Content,
			Embedding:  vector,
		})
	}
	return entities, nil
}

// SearchChunks 在指定知识库内检索与查询向量余弦最相近的切片。
func (e *KnowledgeEngine) SearchChunks(ctx context.Context, knowledgeBaseID int64, vector []float32, topK int) ([]*ChunkHit, error) {
	if err := e.require(); err != nil {
		return nil, err
	}
	docQuery := e.docRepo.Query(ctx).AiKnowledgeDoc
	docs, err := e.docRepo.List(ctx, repository.Where(docQuery.KnowledgeBaseID.Eq(knowledgeBaseID)))
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return []*ChunkHit{}, nil
	}
	docIDs := make([]int64, 0, len(docs))
	docNames := make(map[int64]string, len(docs))
	for _, doc := range docs {
		docIDs = append(docIDs, doc.ID)
		docNames[doc.ID] = doc.Name
	}
	chunkQuery := e.chunkRepo.Query(ctx).AiKnowledgeChunk
	chunks, err := e.chunkRepo.List(ctx,
		repository.Where(chunkQuery.DocID.In(docIDs...)),
		repository.Order(chunkQuery.ID.Asc()),
	)
	if err != nil {
		return nil, err
	}
	hits := make([]*ChunkHit, 0, topK)
	for _, chunk := range chunks {
		embedding, parseErr := unmarshalVector(chunk.Embedding)
		if parseErr != nil {
			log.Warn("知识库切片向量解析失败，跳过该切片", "chunk_id", chunk.ID, "error", parseErr)
			continue
		}
		item := &ChunkHit{
			ChunkID: chunk.ID,
			DocID:   chunk.DocID,
			DocName: docNames[chunk.DocID],
			Content: chunk.Content,
			Score:   cosineSimilarity(vector, embedding),
		}
		if len(hits) < topK {
			hits = append(hits, item)
			if len(hits) == topK {
				sort.Slice(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
			}
			continue
		}
		// 容量已满时替换最小分项并保持有序。
		insertAt := sort.Search(len(hits), func(i int) bool { return hits[i].Score < item.Score })
		if insertAt == 0 {
			continue
		}
		hits = append(hits, nil)
		copy(hits[insertAt+1:], hits[insertAt:])
		hits[insertAt] = item
	}
	return hits, nil
}

// marshalVector 将浮点向量序列化为 JSON 文本。
func marshalVector(vector []float32) (string, error) {
	data, err := json.Marshal(vector)
	if err != nil {
		return "", fmt.Errorf("向量序列化失败: %w", err)
	}
	return string(data), nil
}

// unmarshalVector 解析 JSON 文本为浮点向量。
func unmarshalVector(value string) ([]float32, error) {
	var vector []float32
	if err := json.Unmarshal([]byte(value), &vector); err != nil {
		return nil, fmt.Errorf("向量数据解析失败: %w", err)
	}
	return vector, nil
}

// cosineSimilarity 计算两个向量余弦相似度；维度不等时按较短长度截断。
func cosineSimilarity(a, b []float32) float64 {
	length := len(a)
	if len(b) < length {
		length = len(b)
	}
	if length == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := 0; i < length; i++ {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
