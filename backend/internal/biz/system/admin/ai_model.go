package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	modelconfig "github.com/liujitcn/kratos-kit/ai/model"
	"github.com/sashabaranov/go-openai"
)

// AI模型分类标识，取值对应 system.admin.v1 的 AiModelCategory 枚举。
const (
	aiModelCategoryChat      = adminv1.AiModelCategory_AI_MODEL_CATEGORY_CHAT
	aiModelCategoryEmbedding = adminv1.AiModelCategory_AI_MODEL_CATEGORY_EMBEDDING
	aiModelCategoryRerank    = adminv1.AiModelCategory_AI_MODEL_CATEGORY_RERANK
	aiModelCategoryImage     = adminv1.AiModelCategory_AI_MODEL_CATEGORY_IMAGE
	aiModelCategoryVideo     = adminv1.AiModelCategory_AI_MODEL_CATEGORY_VIDEO
	aiModelCategoryAudio     = adminv1.AiModelCategory_AI_MODEL_CATEGORY_AUDIO
)

// aiModelCategories 汇总全部合法的模型分类。
var aiModelCategories = []adminv1.AiModelCategory{aiModelCategoryChat, aiModelCategoryEmbedding, aiModelCategoryRerank, aiModelCategoryImage, aiModelCategoryVideo, aiModelCategoryAudio}

// AiModelCase 维护AI模型持久化并管理运行时聊天模型快照；模型随AI供应商表单一并保存。
type AiModelCase struct {
	*biz.BaseCase
	*data.AiModelRepository
	tx             data.Transaction
	aiProviderRepo *data.AiProviderRepository
	modelRegistry  *model.Registry
}

// NewAiModelCase 创建AI模型管理业务实例。
func NewAiModelCase(baseCase *biz.BaseCase, tx data.Transaction, repo *data.AiModelRepository, aiProviderRepo *data.AiProviderRepository, modelRegistry *model.Registry) *AiModelCase {
	return &AiModelCase{BaseCase: baseCase, tx: tx, AiModelRepository: repo, aiProviderRepo: aiProviderRepo, modelRegistry: modelRegistry}
}

// RefreshModels 从数据库重建已启用的聊天模型客户端快照。
// 单条脏数据只记录日志并跳过该模型，避免整批刷新失败或阻断服务启动。
func (c *AiModelCase) RefreshModels(ctx context.Context) error {
	modelQuery := c.Query(ctx).AiModel
	items, err := c.List(ctx,
		repository.Where(modelQuery.Status.Eq(int32(commonv1.Status_STATUS_ENABLE))),
		repository.Where(modelQuery.Category.Eq(int32(aiModelCategoryChat))),
		repository.Order(modelQuery.ProviderID.Asc()),
		repository.Order(modelQuery.Sort.Asc()),
		repository.Order(modelQuery.ID.Asc()),
	)
	if err != nil {
		return err
	}
	providerQuery := c.aiProviderRepo.Query(ctx).AiProvider
	providers, err := c.aiProviderRepo.List(ctx,
		repository.Where(providerQuery.Status.Eq(int32(commonv1.Status_STATUS_ENABLE))),
		repository.Order(providerQuery.Sort.Asc()),
		repository.Order(providerQuery.ID.Asc()),
	)
	if err != nil {
		return err
	}
	providerMap := make(map[int64]*models.AiProvider, len(providers))
	for _, provider := range providers {
		providerMap[provider.ID] = provider
	}
	values := make([]model.ProviderModel, 0, len(items))
	for _, item := range items {
		provider := providerMap[item.ProviderID]
		if provider == nil {
			log.Error("AI模型所属供应商不存在或未启用，本次刷新跳过该模型", "model_id", item.ID, "model_name", item.ModelName, "provider_id", item.ProviderID)
			continue
		}
		var modelConfig *modelconfig.ModelConfig
		modelConfig, err = chatModelConfig(provider, item)
		if err != nil {
			log.Error("构造AI模型配置失败，本次刷新跳过该模型", "model_id", item.ID, "model_name", item.ModelName, "error", err)
			continue
		}
		values = append(values, model.ProviderModel{ProviderID: provider.ID, ProviderName: provider.Name, DisplayName: item.DisplayName, Config: modelConfig})
	}
	c.modelRegistry.Replace(values)
	return nil
}

// ListProviderModelForms 读取供应商下的模型表单列表，按排序与ID稳定输出。
func (c *AiModelCase) ListProviderModelForms(ctx context.Context, providerID int64) ([]*adminv1.AiProviderModelForm, error) {
	query := c.Query(ctx).AiModel
	items, err := c.List(ctx,
		repository.Where(query.ProviderID.Eq(providerID)),
		repository.Order(query.Sort.Asc()),
		repository.Order(query.ID.Asc()),
	)
	if err != nil {
		return nil, fmt.Errorf("读取AI模型失败: %w", err)
	}
	result := make([]*adminv1.AiProviderModelForm, 0, len(items))
	for _, item := range items {
		config, err := parseConfigStruct(item.Config)
		if err != nil {
			return nil, errorsx.Internal("解析AI模型分类配置失败").WithCause(err)
		}
		result = append(result, &adminv1.AiProviderModelForm{
			Id: item.ID, ModelName: item.ModelName, DisplayName: item.DisplayName,
			Category: adminv1.AiModelCategory(item.Category), Config: config, Sort: item.Sort, Status: commonv1.Status(item.Status),
		})
	}
	return result, nil
}

// ProviderModelNames 批量读取供应商下的模型名称，用于列表展示。
func (c *AiModelCase) ProviderModelNames(ctx context.Context, providerIDs []int64) map[int64][]string {
	names := make(map[int64][]string, len(providerIDs))
	if len(providerIDs) == 0 {
		return names
	}
	query := c.Query(ctx).AiModel
	items, err := c.List(ctx,
		repository.Where(query.ProviderID.In(providerIDs...)),
		repository.Order(query.ProviderID.Asc()),
		repository.Order(query.Sort.Asc()),
		repository.Order(query.ID.Asc()),
	)
	if err != nil {
		log.Error("读取AI模型名称失败", "error", err)
		return names
	}
	for _, item := range items {
		names[item.ProviderID] = append(names[item.ProviderID], item.ModelName)
	}
	return names
}

// SyncProviderModels 在供应商事务内同步其模型列表：按ID更新已有模型、创建新模型并软删被移除的模型。
func (c *AiModelCase) SyncProviderModels(ctx context.Context, provider *models.AiProvider, forms []*adminv1.AiProviderModelForm) error {
	query := c.Query(ctx).AiModel
	existing, err := c.List(ctx, repository.Where(query.ProviderID.Eq(provider.ID)))
	if err != nil {
		return fmt.Errorf("读取AI模型失败: %w", err)
	}
	existingByID := make(map[int64]*models.AiModel, len(existing))
	for _, item := range existing {
		existingByID[item.ID] = item
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	keepIDs := make([]int64, 0, len(forms))
	seenNames := make(map[string]struct{}, len(forms))
	for _, form := range forms {
		entity, err := aiModelEntity(form)
		if err != nil {
			return err
		}
		if err = validateAiModel(entity); err != nil {
			return err
		}
		if _, duplicated := seenNames[entity.ModelName]; duplicated {
			return errorsx.WithMessageKey(errorsx.InvalidArgument(fmt.Sprintf("AI模型名称重复: %s", entity.ModelName)), "system.admin.base.ai.model.name.duplicated", map[string]string{"Name": entity.ModelName})
		}
		seenNames[entity.ModelName] = struct{}{}
		entity.ProviderID = provider.ID
		if old, exists := existingByID[form.GetId()]; exists && form.GetId() > 0 {
			entity.ID, entity.CreatedBy, entity.CreatedAt = old.ID, old.CreatedBy, old.CreatedAt
			entity.UpdatedBy, entity.UpdatedAt = authInfo.UserId, now
			if err = c.updateModel(ctx, entity); err != nil {
				return err
			}
			keepIDs = append(keepIDs, old.ID)
			continue
		}
		entity.CreatedBy, entity.UpdatedBy = authInfo.UserId, authInfo.UserId
		entity.CreatedAt, entity.UpdatedAt = now, now
		if err = c.Create(ctx, entity); err != nil {
			if errorsx.IsDuplicateKey(err) {
				return errorsx.UniqueConflict("AI模型名称重复", "ai_model", "model_name", "unique_ai_model").WithCause(err)
			}
			return err
		}
		keepIDs = append(keepIDs, entity.ID)
	}
	for id := range existingByID {
		if slices.Contains(keepIDs, id) {
			continue
		}
		if err = c.DeleteByIDs(ctx, []int64{id}); err != nil {
			return err
		}
	}
	return nil
}

// updateModel 按ID整行更新模型，冲突时返回唯一约束错误。
func (c *AiModelCase) updateModel(ctx context.Context, entity *models.AiModel) error {
	query := c.Query(ctx).AiModel
	_, err := query.WithContext(ctx).
		Where(query.ID.Eq(entity.ID)).
		Select(query.ProviderID, query.ModelName, query.DisplayName, query.Category, query.Config, query.Sort, query.Status, query.UpdatedBy, query.UpdatedAt).
		Updates(entity)
	if err != nil {
		if errorsx.IsDuplicateKey(err) {
			return errorsx.UniqueConflict("AI模型名称重复", "ai_model", "model_name", "unique_ai_model").WithCause(err)
		}
		return err
	}
	return nil
}

// TestProviderModels 使用供应商草稿依次测试全部模型，不保存任何配置。
func (c *AiModelCase) TestProviderModels(ctx context.Context, provider *models.AiProvider, forms []*adminv1.AiProviderModelForm) ([]*adminv1.AiProviderModelTestResult, error) {
	results := make([]*adminv1.AiProviderModelTestResult, 0, len(forms))
	for _, form := range forms {
		item, err := aiModelEntity(form)
		if err != nil {
			return nil, err
		}
		if err = validateAiModel(item); err != nil {
			return nil, err
		}
		result := &adminv1.AiProviderModelTestResult{ModelName: item.ModelName}
		switch item.Category {
		case int32(aiModelCategoryChat):
			var modelConfig *modelconfig.ModelConfig
			modelConfig, err = chatModelConfig(provider, item)
			if err != nil {
				return nil, err
			}
			testResult := testChatModel(ctx, modelConfig)
			result.Success, result.DurationMs, result.Message = testResult.Success, testResult.DurationMs, testResult.Message
		case int32(aiModelCategoryEmbedding):
			testResult := testEmbeddingModel(ctx, provider, item)
			result.Success, result.DurationMs, result.Message = testResult.Success, testResult.DurationMs, testResult.Message
		default:
			// 持久化国际化消息标记，由管理端按当前语言渲染。
			result.Message = "__I18N__:system.base.ai_provider.message.test_unsupported_category"
		}
		results = append(results, result)
	}
	return results, nil
}

// DeleteByProvider 级联软删供应商下的全部模型，随供应商删除在同一事务内调用。
func (c *AiModelCase) DeleteByProvider(ctx context.Context, providerID int64) error {
	query := c.Query(ctx).AiModel
	return c.Delete(ctx, repository.Where(query.ProviderID.Eq(providerID)))
}

// aiModelEntity 将模型表单转换为记录。
func aiModelEntity(form *adminv1.AiProviderModelForm) (*models.AiModel, error) {
	config := "{}"
	if form.GetConfig() != nil {
		encoded, err := json.Marshal(form.GetConfig().AsMap())
		if err != nil {
			return nil, errorsx.InvalidArgument("AI模型分类配置无效").WithCause(err)
		}
		config = string(encoded)
	}
	return &models.AiModel{
		ModelName: strings.TrimSpace(form.GetModelName()), DisplayName: strings.TrimSpace(form.GetDisplayName()),
		Category: int32(form.GetCategory()), Config: config, Sort: form.GetSort(), Status: int32(form.GetStatus()),
	}, nil
}

// validateAiModel 校验模型基础字段与分类个性化配置。
func validateAiModel(item *models.AiModel) error {
	if item.ModelName == "" || len(item.ModelName) > 200 {
		return errorsx.InvalidArgument("模型名称不能为空且不能超过200个字符")
	}
	if len(item.DisplayName) > 100 {
		return errorsx.InvalidArgument("显示名称不能超过100个字符")
	}
	category := adminv1.AiModelCategory(item.Category)
	if !slices.Contains(aiModelCategories, category) {
		return errorsx.InvalidArgument("AI模型分类无效")
	}
	if item.Status != int32(commonv1.Status_STATUS_ENABLE) && item.Status != int32(commonv1.Status_STATUS_DISABLE) {
		return errorsx.InvalidArgument("AI模型状态无效")
	}
	if item.Sort < 0 {
		return errorsx.InvalidArgument("模型排序不能小于0")
	}
	var err error
	switch category {
	case aiModelCategoryChat:
		_, err = parseAiModelChatConfig(item.Config)
	case aiModelCategoryEmbedding:
		_, err = validateEmbeddingConfig(item.Config)
	default:
		// 其余分类暂无运行时消费字段，仅要求配置是JSON对象。
		_, err = parseConfigStruct(item.Config)
	}
	if err != nil {
		return errorsx.InvalidArgument("AI模型分类配置无效").WithCause(err)
	}
	return nil
}

// parseAiModelChatConfig 解析并校验聊天模型的分类个性化配置。
func parseAiModelChatConfig(raw string) (*dto.AiModelChatConfig, error) {
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	var value dto.AiModelChatConfig
	err := decoder.Decode(&value)
	if err != nil {
		return nil, fmt.Errorf("聊天模型配置必须是JSON对象: %w", err)
	}
	if value.APIType != "CHAT_COMPLETIONS" && value.APIType != "RESPONSES" {
		return nil, errors.New("api_type 必须是 CHAT_COMPLETIONS 或 RESPONSES")
	}
	if value.Temperature < 0 || value.Temperature > 2 || value.MaxTokens < 0 || value.MaxTokens > 200000 || value.TimeoutSeconds < 0 || value.TimeoutSeconds > 600 || value.MaxRetries < 0 || value.MaxRetries > 10 {
		return nil, errors.New("聊天模型请求参数超出范围")
	}
	return &value, nil
}

// validateEmbeddingConfig 解析并校验embedding模型的分类个性化配置。
func validateEmbeddingConfig(raw string) (*ai.EmbeddingModelConfig, error) {
	value, err := ai.ParseEmbeddingModelConfig(raw)
	if err != nil {
		return nil, err
	}
	if value.Dimensions < 0 || value.Dimensions > 8192 || (value.Dimensions > 0 && value.Dimensions < 64) {
		return nil, errors.New("dimensions 必须在64到8192之间，0表示未声明")
	}
	if value.TimeoutSeconds < 0 || value.TimeoutSeconds > 600 || value.MaxRetries < 0 || value.MaxRetries > 10 {
		return nil, errors.New("embedding模型请求参数超出范围")
	}
	return value, nil
}

// chatModelConfig 将数据库供应商和聊天模型记录组合为模型客户端参数。
func chatModelConfig(provider *models.AiProvider, item *models.AiModel) (*modelconfig.ModelConfig, error) {
	if provider.Provider != aiProviderOpenAICompatible && provider.Provider != aiProviderOllama {
		return nil, errors.New("不支持的AI Provider")
	}
	config, err := parseConfigStruct(provider.Config)
	if err != nil {
		return nil, err
	}
	organization, _ := config.AsMap()["organization"].(string)
	chatConfig, err := parseAiModelChatConfig(item.Config)
	if err != nil {
		return nil, err
	}
	apiType := modelconfig.APITypeChatCompletions
	if chatConfig.APIType == "RESPONSES" {
		apiType = modelconfig.APITypeResponses
	}
	return &modelconfig.ModelConfig{
		Provider: modelconfig.Provider(provider.Provider), ModelName: item.ModelName, APIKey: provider.APIKey,
		BaseURL: provider.BaseURL, Organization: organization, Temperature: chatConfig.Temperature, MaxTokens: chatConfig.MaxTokens,
		TimeoutSeconds: chatConfig.TimeoutSeconds, MaxRetries: chatConfig.MaxRetries, APIType: apiType,
	}, nil
}

// testChatModel 按聊天模型配置运行单次连通性测试，并隔离请求时限和输出长度。
func testChatModel(ctx context.Context, config *modelconfig.ModelConfig) *adminv1.AiProviderModelTestResult {
	testConfig := *config
	if testConfig.MaxTokens == 0 || testConfig.MaxTokens > 16 {
		testConfig.MaxTokens = 16
	}
	testConfig.MaxRetries = 0
	if testConfig.TimeoutSeconds <= 0 || testConfig.TimeoutSeconds > 30 {
		testConfig.TimeoutSeconds = 30
	}
	modelCtx, cancel := context.WithTimeout(ctx, time.Duration(testConfig.TimeoutSeconds)*time.Second)
	startAt := time.Now()
	err := model.TestConnection(modelCtx, &testConfig)
	cancel()
	result := &adminv1.AiProviderModelTestResult{
		Success:    err == nil,
		DurationMs: int32(time.Since(startAt).Milliseconds()),
	}
	if err != nil {
		result.Message = aiTestError(err, testConfig.APIKey)
	}
	return result
}

// testEmbeddingModel 运行embedding模型连通性测试，使用极小输入验证接口可用。
func testEmbeddingModel(ctx context.Context, provider *models.AiProvider, item *models.AiModel) *adminv1.AiProviderModelTestResult {
	result := &adminv1.AiProviderModelTestResult{}
	embeddingConfig, err := ai.ParseEmbeddingModelConfig(item.Config)
	if err != nil {
		result.Message = aiTestError(err, provider.APIKey)
		return result
	}
	timeoutSeconds := embeddingConfig.TimeoutSeconds
	if timeoutSeconds <= 0 || timeoutSeconds > 30 {
		timeoutSeconds = 30
	}
	modelConfig := &modelconfig.ModelConfig{
		Provider: modelconfig.Provider(provider.Provider), ModelName: item.ModelName,
		APIKey: provider.APIKey, BaseURL: provider.BaseURL,
	}
	client, err := modelconfig.NewClient(modelConfig)
	if err != nil {
		result.Message = aiTestError(err, provider.APIKey)
		return result
	}
	modelCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()
	startAt := time.Now()
	request := openai.EmbeddingRequest{Model: openai.EmbeddingModel(item.ModelName), Input: []string{"ping"}}
	_, err = client.CreateEmbeddings(modelCtx, request)
	result.DurationMs = int32(time.Since(startAt).Milliseconds())
	result.Success = err == nil
	if err != nil {
		result.Message = aiTestError(err, provider.APIKey)
	}
	return result
}

// aiTestError 清理并截断模型接口错误，避免把密钥等敏感信息回传给调用方。
func aiTestError(err error, apiKey string) string {
	message := err.Error()
	if apiKey != "" {
		message = strings.ReplaceAll(message, apiKey, "[REDACTED]")
	}
	if runes := []rune(message); len(runes) > 512 {
		return string(runes[:512]) + "..."
	}
	return message
}
