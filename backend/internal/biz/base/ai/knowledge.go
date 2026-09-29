// Package ai 提供 AI 知识库（RAG）领域层：知识库与文档存储、切片、向量化、
// 语义检索与聊天上下文组装。知识库与切片数据落在 data.databases.rag 命名数据源
// （PostgreSQL），三张表的模型、查询与仓储由 gorm-gen 生成在 internal/data/gen/rag；
// embedding 以 TEXT 列存 JSON 向量，余弦检索在应用侧计算；
// embedding 模型与供应商配置仍读取 MySQL 主库。
package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	ragdata "github.com/liujitcn/kratos-admin/backend/internal/data/gen/rag/data"
	"github.com/liujitcn/kratos-kit/ai/model"
	"github.com/liujitcn/kratos-kit/database/gorm"
	"github.com/sashabaranov/go-openai"
)

// 切片策略：按字符切分，块大小与重叠量与上游 RAG 实践对齐。
const (
	chunkSize      = 512
	chunkOverlap   = 50
	defaultTopK    = 5
	embeddingRetry = 2
)

// KnowledgeEngine 提供知识库存储、切片、向量化、检索与聊天上下文组装能力。
type KnowledgeEngine struct {
	data           *ragdata.Data
	knowledgeRepo  *ragdata.AiKnowledgeRepository
	docRepo        *ragdata.AiKnowledgeDocRepository
	chunkRepo      *ragdata.AiKnowledgeChunkRepository
	aiProviderRepo *data.AiProviderRepository
	aiModelRepo    *data.AiModelRepository
	available      bool
}

// NewKnowledgeEngine 创建知识库引擎；rag 数据源未配置时引擎降级为不可用，
// 相关方法统一返回未配置错误，不阻断服务启动。
func NewKnowledgeEngine(databases map[string]*gorm.Client, base *ragdata.Data, aiProviderRepo *data.AiProviderRepository, aiModelRepo *data.AiModelRepository) *KnowledgeEngine {
	client := databases[ragdata.RagGormClientName]
	if client == nil || client.DB == nil {
		log.Warn("AI知识库（rag）数据源未配置，知识库能力不可用")
		return &KnowledgeEngine{aiProviderRepo: aiProviderRepo, aiModelRepo: aiModelRepo}
	}
	return &KnowledgeEngine{
		data:           base,
		knowledgeRepo:  ragdata.NewAiKnowledgeRepository(base),
		docRepo:        ragdata.NewAiKnowledgeDocRepository(base),
		chunkRepo:      ragdata.NewAiKnowledgeChunkRepository(base),
		aiProviderRepo: aiProviderRepo,
		aiModelRepo:    aiModelRepo,
		available:      true,
	}
}

// IngestText 将文本入库：创建文档记录（登记原始文件路径与hash）→ 切片 → embedding → 批量写切片。
// 处理失败时文档状态置为失败并保留失败原因，原始文件留在对象存储供重试。
func (e *KnowledgeEngine) IngestText(ctx context.Context, knowledgeBaseID int64, name, content, filePath, fileHash string, createdBy int64) (*Doc, error) {
	if e == nil {
		return nil, fmt.Errorf("AI知识库引擎未初始化")
	}
	if err := e.require(); err != nil {
		return nil, err
	}
	knowledge, err := e.GetKnowledge(ctx, knowledgeBaseID)
	if err != nil {
		return nil, fmt.Errorf("AI知识库不存在: %w", err)
	}
	provider, aiModel, err := e.loadEmbeddingModel(ctx, knowledge.ModelID)
	if err != nil {
		return nil, err
	}

	doc := &Doc{
		KnowledgeBaseID: knowledgeBaseID,
		Name:            name,
		FilePath:        filePath,
		FileHash:        fileHash,
		Status:          DocStatusReady,
		CreatedBy:       createdBy,
	}
	if err = e.CreateDoc(ctx, doc); err != nil {
		return nil, fmt.Errorf("创建知识库文档失败: %w", err)
	}

	if err = e.embedAndStore(ctx, provider, aiModel, knowledge, doc.ID, content); err != nil {
		return nil, err
	}
	return doc, nil
}

// ReprocessDoc 重新处理失败文档：按重新抽取的原文切片、向量化并替换旧切片。
// 原始文件由调用方从对象存储读取并校验 hash 后抽取文本传入。
func (e *KnowledgeEngine) ReprocessDoc(ctx context.Context, docID int64, text string) error {
	if e == nil {
		return fmt.Errorf("AI知识库引擎未初始化")
	}
	if err := e.require(); err != nil {
		return err
	}
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("文档抽取文本为空，无法重试")
	}
	doc, err := e.GetDoc(ctx, docID)
	if err != nil {
		return fmt.Errorf("知识库文档不存在: %w", err)
	}
	knowledge, err := e.GetKnowledge(ctx, doc.KnowledgeBaseID)
	if err != nil {
		return fmt.Errorf("AI知识库不存在: %w", err)
	}
	provider, aiModel, err := e.loadEmbeddingModel(ctx, knowledge.ModelID)
	if err != nil {
		return err
	}
	return e.embedAndStore(ctx, provider, aiModel, knowledge, doc.ID, text)
}

// embedAndStore 完成切片、逐块向量化与切片写入，并更新文档状态；
// 任一切片失败时文档状态置为失败并保留失败原因。
func (e *KnowledgeEngine) embedAndStore(ctx context.Context, provider *models.AiProvider, aiModel *models.AiModel, knowledge *Knowledge, docID int64, content string) error {
	expectedDims := embeddingDimensions(knowledge)
	chunks := chunkText(content, chunkSize, chunkOverlap)
	chunkContents := make([]ChunkContent, 0, len(chunks))
	for index, chunk := range chunks {
		vector, embedErr := e.embedText(ctx, provider, aiModel, chunk)
		if embedErr != nil {
			_ = e.UpdateDocStatus(ctx, docID, 0, DocStatusFailed, fmt.Sprintf("第%d个切片向量化失败: %v", index+1, embedErr))
			return embedErr
		}
		if len(vector) != expectedDims {
			err := fmt.Errorf("embedding 模型返回维度 %d，与知识库配置维度 %d 不符", len(vector), expectedDims)
			_ = e.UpdateDocStatus(ctx, docID, 0, DocStatusFailed, err.Error())
			return err
		}
		chunkContents = append(chunkContents, ChunkContent{Index: index, Content: chunk, Vector: vector})
	}

	if err := e.ReplaceChunks(ctx, docID, chunkContents); err != nil {
		_ = e.UpdateDocStatus(ctx, docID, 0, DocStatusFailed, fmt.Sprintf("切片写入失败: %v", err))
		return fmt.Errorf("知识库切片写入失败: %w", err)
	}
	if err := e.UpdateDocStatus(ctx, docID, int32(len(chunkContents)), DocStatusReady, ""); err != nil {
		return err
	}
	return nil
}

// Retrieve 检索多个知识库并组装聊天上下文；任一环节失败只记录日志，不阻断对话。
func (e *KnowledgeEngine) Retrieve(ctx context.Context, knowledgeBaseIDs []int64, query string, topK int) string {
	if !e.Available() || len(knowledgeBaseIDs) == 0 || strings.TrimSpace(query) == "" {
		return ""
	}
	if topK <= 0 {
		topK = defaultTopK
	}
	var builder strings.Builder
	matched := 0
	for _, knowledgeBaseID := range knowledgeBaseIDs {
		knowledge, err := e.GetKnowledge(ctx, knowledgeBaseID)
		if err != nil {
			log.Warn("知识库检索跳过：读取知识库失败", "knowledge_base_id", knowledgeBaseID, "error", err)
			continue
		}
		provider, aiModel, err := e.loadEmbeddingModel(ctx, knowledge.ModelID)
		if err != nil {
			log.Warn("知识库检索跳过：embedding模型不可用", "knowledge_base_id", knowledgeBaseID, "error", err)
			continue
		}
		vector, err := e.embedText(ctx, provider, aiModel, query)
		if err != nil {
			log.Warn("知识库检索跳过：查询向量化失败", "knowledge_base_id", knowledgeBaseID, "error", err)
			continue
		}
		hits, err := e.SearchChunks(ctx, knowledgeBaseID, vector, topK)
		if err != nil {
			log.Warn("知识库检索跳过：向量检索失败", "knowledge_base_id", knowledgeBaseID, "error", err)
			continue
		}
		if len(hits) == 0 {
			continue
		}
		if matched > 0 {
			builder.WriteString("\n")
		}
		fmt.Fprintf(&builder, "【知识库：%s】\n", knowledge.Name)
		for _, hit := range hits {
			fmt.Fprintf(&builder, "- （%s）%s\n", hit.DocName, strings.TrimSpace(hit.Content))
		}
		matched++
	}
	if matched == 0 {
		return ""
	}
	return "以下是与用户问题相关的知识库资料，请优先依据这些资料回答；若资料与问题无关或不足，请明确说明并基于通用知识回答。\n\n" + builder.String()
}

// Search 检索测试：在单个知识库内返回语义最相近的切片。
func (e *KnowledgeEngine) Search(ctx context.Context, knowledgeBaseID int64, query string, topK int) ([]*ChunkHit, error) {
	if err := e.require(); err != nil {
		return nil, err
	}
	if topK <= 0 {
		topK = defaultTopK
	}
	knowledge, err := e.GetKnowledge(ctx, knowledgeBaseID)
	if err != nil {
		return nil, fmt.Errorf("AI知识库不存在: %w", err)
	}
	provider, aiModel, err := e.loadEmbeddingModel(ctx, knowledge.ModelID)
	if err != nil {
		return nil, err
	}
	vector, err := e.embedText(ctx, provider, aiModel, query)
	if err != nil {
		return nil, err
	}
	return e.SearchChunks(ctx, knowledgeBaseID, vector, topK)
}

// loadProvider 读取 AI 模型供应商（MySQL 主库）。
func (e *KnowledgeEngine) loadProvider(ctx context.Context, providerID int64) (*models.AiProvider, error) {
	query := e.aiProviderRepo.Query(ctx).AiProvider
	providers, err := e.aiProviderRepo.List(ctx,
		repository.Where(query.ID.Eq(providerID)),
		repository.Limit(1),
	)
	if err != nil {
		return nil, fmt.Errorf("读取AI供应商失败: %w", err)
	}
	if len(providers) == 0 {
		return nil, fmt.Errorf("AI供应商 %d 不存在", providerID)
	}
	return providers[0], nil
}

// loadEmbeddingModel 读取知识库绑定的embedding模型记录及其供应商（MySQL 主库）。
func (e *KnowledgeEngine) loadEmbeddingModel(ctx context.Context, modelID int64) (*models.AiProvider, *models.AiModel, error) {
	query := e.aiModelRepo.Query(ctx).AiModel
	items, err := e.aiModelRepo.List(ctx,
		repository.Where(query.ID.Eq(modelID)),
		repository.Limit(1),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("读取embedding模型失败: %w", err)
	}
	if len(items) == 0 {
		return nil, nil, fmt.Errorf("embedding模型 %d 不存在", modelID)
	}
	provider, err := e.loadProvider(ctx, items[0].ProviderID)
	if err != nil {
		return nil, nil, err
	}
	return provider, items[0], nil
}

// embedText 调用模型的 embedding 接口向量化文本，超时与重试优先取模型配置。
func (e *KnowledgeEngine) embedText(ctx context.Context, provider *models.AiProvider, aiModel *models.AiModel, text string) ([]float32, error) {
	embeddingConfig, err := ParseEmbeddingModelConfig(aiModel.Config)
	if err != nil {
		return nil, fmt.Errorf("解析embedding模型配置失败: %w", err)
	}
	config := &model.ModelConfig{
		Provider:  model.Provider(provider.Provider),
		ModelName: aiModel.ModelName,
		APIKey:    provider.APIKey,
		BaseURL:   provider.BaseURL,
	}
	client, err := model.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("创建embedding客户端失败: %w", err)
	}

	if embeddingConfig.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(embeddingConfig.TimeoutSeconds)*time.Second)
		defer cancel()
	}

	retries := embeddingRetry
	if embeddingConfig.MaxRetries > 0 {
		retries = int(embeddingConfig.MaxRetries)
	}
	var lastErr error
	request := openai.EmbeddingRequest{Model: openai.EmbeddingModel(aiModel.ModelName), Input: []string{text}}
	for attempt := 0; attempt < retries; attempt++ {
		var response openai.EmbeddingResponse
		response, lastErr = client.CreateEmbeddings(ctx, request)
		if lastErr == nil {
			if len(response.Data) == 0 {
				return nil, fmt.Errorf("embedding 接口返回空结果")
			}
			return response.Data[0].Embedding, nil
		}
		if attempt < retries-1 {
			time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
		}
	}
	return nil, fmt.Errorf("embedding 调用失败: %w", lastErr)
}

// embeddingDimensions 返回知识库配置的向量维度，未配置时回落到默认维度。
func embeddingDimensions(knowledge *Knowledge) int {
	if knowledge != nil && knowledge.EmbeddingDimensions > 0 {
		return int(knowledge.EmbeddingDimensions)
	}
	return DefaultEmbeddingDimensions
}

// chunkText 按字符切分文本，相邻块之间保留重叠。
func chunkText(text string, size, overlap int) []string {
	runes := []rune(text)
	if len(runes) == 0 {
		return nil
	}
	if size <= 0 {
		size = chunkSize
	}
	if overlap < 0 || overlap >= size {
		overlap = 0
	}
	var chunks []string
	for start := 0; start < len(runes); start += size - overlap {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		chunk := strings.TrimSpace(string(runes[start:end]))
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
		if end >= len(runes) {
			break
		}
	}
	return chunks
}
