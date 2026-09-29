package biz

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/rag/admin/v1"
	sysadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	"gorm.io/gorm"
)

// aiModelCategoryEmbedding 表示 embedding 模型分类，取值对应 system.admin.v1 的 AiModelCategory 枚举。
const aiModelCategoryEmbedding = int32(sysadminv1.AiModelCategory_AI_MODEL_CATEGORY_EMBEDDING)

// AiKnowledgeCase 管理AI知识库。
type AiKnowledgeCase struct {
	*biz.BaseCase
	engine          *ai.KnowledgeEngine
	aiModelRepo     *data.AiModelRepository
	aiProviderRepo  *data.AiProviderRepository
	knowledgeMapper *mapper.CopierMapper[adminv1.AiKnowledge, ai.Knowledge]
	formMapper      *mapper.CopierMapper[adminv1.AiKnowledgeForm, ai.Knowledge]
}

// NewAiKnowledgeCase 创建AI知识库管理业务实例。
func NewAiKnowledgeCase(baseCase *biz.BaseCase, engine *ai.KnowledgeEngine, aiModelRepo *data.AiModelRepository, aiProviderRepo *data.AiProviderRepository) *AiKnowledgeCase {
	return &AiKnowledgeCase{
		BaseCase:        baseCase,
		engine:          engine,
		aiModelRepo:     aiModelRepo,
		aiProviderRepo:  aiProviderRepo,
		knowledgeMapper: mapper.NewCopierMapper[adminv1.AiKnowledge, ai.Knowledge](),
		formMapper:      mapper.NewCopierMapper[adminv1.AiKnowledgeForm, ai.Knowledge](),
	}
}

// PageAiKnowledge 分页查询AI知识库。
func (c *AiKnowledgeCase) PageAiKnowledge(ctx context.Context, req *adminv1.PageAiKnowledgeRequest) (*adminv1.PageAiKnowledgeResponse, error) {
	items, total, err := c.engine.ListKnowledge(ctx, req.GetName(), req.GetPageNum(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	modelInfos := c.modelInfos(ctx, items)
	records := make([]*adminv1.AiKnowledge, 0, len(items))
	for _, item := range items {
		record := c.knowledgeMapper.ToDTO(item)
		record.ProviderName = modelInfos[item.ModelID].providerName
		record.ModelName = modelInfos[item.ModelID].modelName
		records = append(records, record)
	}
	return &adminv1.PageAiKnowledgeResponse{Knowledges: records, Total: int32(total)}, nil
}

// GetAiKnowledge 查询AI知识库表单。
func (c *AiKnowledgeCase) GetAiKnowledge(ctx context.Context, id int64) (*adminv1.AiKnowledgeForm, error) {
	item, err := c.engine.GetKnowledge(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("AI知识库不存在")
		}
		return nil, err
	}
	return c.formMapper.ToDTO(item), nil
}

// CreateAiKnowledge 创建AI知识库。
func (c *AiKnowledgeCase) CreateAiKnowledge(ctx context.Context, form *adminv1.AiKnowledgeForm) error {
	dimensions, err := c.validateEmbeddingModel(ctx, form.GetModelId())
	if err != nil {
		return err
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	item := c.formMapper.ToEntity(form)
	item.Name = strings.TrimSpace(item.Name)
	item.Description = strings.TrimSpace(item.Description)
	// 向量维度在创建时快照，后续更换模型不会影响已入库切片。
	item.EmbeddingDimensions = dimensions
	// 创建人与更新人由后端注入，不信任表单。
	item.CreatedBy = authInfo.UserId
	item.UpdatedBy = authInfo.UserId
	if err = c.engine.CreateKnowledge(ctx, item); err != nil {
		return fmt.Errorf("创建AI知识库失败: %w", err)
	}
	return nil
}

// UpdateAiKnowledge 更新AI知识库。
func (c *AiKnowledgeCase) UpdateAiKnowledge(ctx context.Context, form *adminv1.AiKnowledgeForm) error {
	oldItem, err := c.engine.GetKnowledge(ctx, form.GetId())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errorsx.ResourceNotFound("AI知识库不存在")
		}
		return err
	}
	if oldItem.ModelID != form.GetModelId() {
		if _, err = c.validateEmbeddingModel(ctx, form.GetModelId()); err != nil {
			return err
		}
		// 向量化模型变更会使已入库切片失效，存在文档时禁止修改。
		if c.hasDocs(ctx, form.GetId()) {
			return errorsx.InvalidArgument("知识库下已有文档，请先删除全部文档再修改向量化模型")
		}
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	item := c.formMapper.ToEntity(form)
	// 更新目标以数据库记录为准，忽略表单中的 ID。
	item.ID = oldItem.ID
	item.Name = strings.TrimSpace(item.Name)
	item.Description = strings.TrimSpace(item.Description)
	// 维度快照保持创建时的值，避免历史切片与检索维度漂移。
	item.EmbeddingDimensions = oldItem.EmbeddingDimensions
	item.UpdatedBy = authInfo.UserId
	if err = c.engine.UpdateKnowledge(ctx, item); err != nil {
		return fmt.Errorf("更新AI知识库失败: %w", err)
	}
	return nil
}

// DeleteAiKnowledge 删除AI知识库，级联删除文档与切片。
func (c *AiKnowledgeCase) DeleteAiKnowledge(ctx context.Context, id int64) error {
	if _, err := c.engine.GetKnowledge(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errorsx.ResourceNotFound("AI知识库不存在")
		}
		return err
	}
	return c.engine.DeleteKnowledge(ctx, id)
}

// ListAiKnowledgeModels 查询已启用的embedding模型选项。
func (c *AiKnowledgeCase) ListAiKnowledgeModels(ctx context.Context) (*adminv1.ListAiKnowledgeModelsResponse, error) {
	query := c.aiModelRepo.Query(ctx).AiModel
	items, err := c.aiModelRepo.List(ctx,
		repository.Where(query.Status.Eq(int32(commonv1.Status_STATUS_ENABLE))),
		repository.Where(query.Category.Eq(aiModelCategoryEmbedding)),
		repository.Order(query.Sort.Asc()),
		repository.Order(query.ID.Asc()),
	)
	if err != nil {
		return nil, fmt.Errorf("读取embedding模型失败: %w", err)
	}
	providerNames, err := c.providerNames(ctx, items)
	if err != nil {
		return nil, err
	}
	records := make([]*adminv1.AiKnowledgeModel, 0, len(items))
	for _, item := range items {
		records = append(records, &adminv1.AiKnowledgeModel{
			Id:           item.ID,
			ProviderName: providerNames[item.ProviderID],
			ModelName:    item.ModelName,
			DisplayName:  item.DisplayName,
		})
	}
	return &adminv1.ListAiKnowledgeModelsResponse{Models: records}, nil
}

// modelInfo 汇总知识库展示所需的模型来源信息。
type modelInfo struct {
	providerName string
	modelName    string
}

// modelInfos 批量读取知识库引用的embedding模型展示信息。
func (c *AiKnowledgeCase) modelInfos(ctx context.Context, items []*ai.Knowledge) map[int64]modelInfo {
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ModelID)
	}
	infos := make(map[int64]modelInfo, len(ids))
	if len(ids) == 0 {
		return infos
	}
	query := c.aiModelRepo.Query(ctx).AiModel
	models, err := c.aiModelRepo.List(ctx, repository.Where(query.ID.In(ids...)))
	if err != nil {
		return infos
	}
	providerNames, err := c.providerNames(ctx, models)
	if err != nil {
		return infos
	}
	for _, item := range models {
		infos[item.ID] = modelInfo{providerName: providerNames[item.ProviderID], modelName: item.ModelName}
	}
	return infos
}

// providerNames 批量读取供应商名称。
func (c *AiKnowledgeCase) providerNames(ctx context.Context, items []*models.AiModel) (map[int64]string, error) {
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ProviderID)
	}
	names := make(map[int64]string, len(ids))
	if len(ids) == 0 {
		return names, nil
	}
	query := c.aiProviderRepo.Query(ctx).AiProvider
	providers, err := c.aiProviderRepo.List(ctx, repository.Where(query.ID.In(ids...)))
	if err != nil {
		return names, fmt.Errorf("读取AI供应商失败: %w", err)
	}
	for _, provider := range providers {
		names[provider.ID] = provider.Name
	}
	return names, nil
}

// hasDocs 判断知识库下是否存在文档。
func (c *AiKnowledgeCase) hasDocs(ctx context.Context, knowledgeBaseID int64) bool {
	_, total, err := c.engine.ListDocs(ctx, knowledgeBaseID, 1, 1)
	return err == nil && total > 0
}

// validateEmbeddingModel 校验embedding模型存在、分类正确、模型与供应商均启用，
// 并返回创建知识库时应快照的向量维度。
func (c *AiKnowledgeCase) validateEmbeddingModel(ctx context.Context, modelID int64) (int32, error) {
	query := c.aiModelRepo.Query(ctx).AiModel
	items, err := c.aiModelRepo.List(ctx,
		repository.Where(query.ID.Eq(modelID)),
		repository.Limit(1),
	)
	if err != nil {
		return 0, fmt.Errorf("读取embedding模型失败: %w", err)
	}
	if len(items) == 0 {
		return 0, errorsx.InvalidArgument("embedding模型不存在")
	}
	item := items[0]
	if item.Category != aiModelCategoryEmbedding {
		return 0, errorsx.InvalidArgument("所选模型不是embedding模型")
	}
	if item.Status != int32(commonv1.Status_STATUS_ENABLE) {
		return 0, errorsx.InvalidArgument("embedding模型未启用")
	}
	providerQuery := c.aiProviderRepo.Query(ctx).AiProvider
	providers, err := c.aiProviderRepo.List(ctx,
		repository.Where(providerQuery.ID.Eq(item.ProviderID)),
		repository.Limit(1),
	)
	if err != nil {
		return 0, fmt.Errorf("读取AI供应商失败: %w", err)
	}
	if len(providers) == 0 {
		return 0, errorsx.InvalidArgument("embedding模型所属供应商不存在")
	}
	if providers[0].Status != int32(commonv1.Status_STATUS_ENABLE) {
		return 0, errorsx.InvalidArgument("embedding模型所属供应商未启用")
	}
	config, err := ai.ParseEmbeddingModelConfig(item.Config)
	if err != nil {
		return 0, errorsx.InvalidArgument("embedding模型配置无效").WithCause(err)
	}
	if config.Dimensions > 0 {
		return config.Dimensions, nil
	}
	return ai.DefaultEmbeddingDimensions, nil
}
