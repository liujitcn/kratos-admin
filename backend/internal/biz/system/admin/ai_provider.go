package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	modelconfig "github.com/liujitcn/kratos-kit/ai/model"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	aiProviderOpenAICompatible = "openai_compatible"
	aiProviderOllama           = "ollama"
)

// AiProviderCase 管理AI供应商配置并维护运行时模型快照。
type AiProviderCase struct {
	*biz.BaseCase
	*data.AiProviderRepository
	tx            data.Transaction
	modelRegistry *model.Registry
}

// NewAiProviderCase 创建AI供应商管理业务实例。
func NewAiProviderCase(baseCase *biz.BaseCase, tx data.Transaction, repo *data.AiProviderRepository, modelRegistry *model.Registry) *AiProviderCase {
	return &AiProviderCase{BaseCase: baseCase, tx: tx, AiProviderRepository: repo, modelRegistry: modelRegistry}
}

// RefreshAiProvider 从数据库重建已启用AI模型客户端快照。
// 单条脏数据只记录日志并跳过该供应商，避免整批刷新失败或阻断服务启动。
func (c *AiProviderCase) RefreshAiProvider(ctx context.Context) error {
	query := c.Query(ctx).AiProvider
	providers, err := c.List(ctx,
		repository.Where(query.Status.Eq(int32(commonv1.Status_STATUS_ENABLE))),
		repository.Order(query.Sort.Asc()),
		repository.Order(query.ID.Asc()),
	)
	if err != nil {
		return err
	}
	values := make([]model.ProviderModel, 0)
	for _, provider := range providers {
		var modelItems []dto.AiProviderModelConfig
		modelItems, err = parseAiProviderModels(provider.Models)
		if err != nil {
			log.Error("解析AI供应商模型配置失败，本次刷新跳过该供应商", "provider_id", provider.ID, "provider_name", provider.Name, "error", err)
			continue
		}
		for _, item := range modelItems {
			var modelConfig *modelconfig.ModelConfig
			modelConfig, err = providerModelConfig(provider, item)
			if err != nil {
				log.Error("构造AI供应商模型配置失败，本次刷新跳过该模型", "provider_id", provider.ID, "model_name", item.ModelName, "error", err)
				continue
			}
			values = append(values, model.ProviderModel{ProviderID: provider.ID, ProviderName: provider.Name, DisplayName: item.DisplayName, Config: modelConfig})
		}
	}
	c.modelRegistry.Replace(values)
	return nil
}

// PageAiProvider 分页查询AI供应商。
func (c *AiProviderCase) PageAiProvider(ctx context.Context, req *adminv1.PageAiProviderRequest) (*adminv1.PageAiProviderResponse, error) {
	query := c.Query(ctx).AiProvider
	opts := []repository.QueryOption{repository.Order(query.Sort.Asc()), repository.Order(query.ID.Asc())}
	if req.GetName() != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetName()+"%")))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	providers, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	result := make([]*adminv1.AiProvider, 0, len(providers))
	for _, provider := range providers {
		var modelItems []dto.AiProviderModelConfig
		modelItems, err = parseAiProviderModels(provider.Models)
		if err != nil {
			// 单条脏数据不应让整个列表接口失败，记录日志后以空模型列表展示。
			log.Error("读取AI供应商模型配置失败，列表跳过该记录", "provider_id", provider.ID, "provider_name", provider.Name, "error", err)
			modelItems = nil
		}
		modelNames := make([]string, 0, len(modelItems))
		for _, item := range modelItems {
			modelNames = append(modelNames, item.ModelName)
		}
		result = append(result, &adminv1.AiProvider{
			Id:               provider.ID,
			Name:             provider.Name,
			BaseUrl:          provider.BaseURL,
			ApiKeyConfigured: provider.APIKey != "",
			ModelNames:       modelNames,
			Sort:             provider.Sort,
			Provider:         provider.Provider,
			Status:           commonv1.Status(provider.Status),
			CreatedAt:        provider.CreatedAt.Format(time.DateTime),
			UpdatedAt:        provider.UpdatedAt.Format(time.DateTime),
		})
	}
	return &adminv1.PageAiProviderResponse{Providers: result, Total: int32(total)}, nil
}

// GetAiProvider 查询AI供应商表单并隐藏密钥原文。
func (c *AiProviderCase) GetAiProvider(ctx context.Context, id int64) (*adminv1.AiProviderForm, error) {
	provider, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toAiProviderForm(provider)
}

// TestAiProviderModels 使用当前表单草稿依次测试全部模型，不保存任何配置。
func (c *AiProviderCase) TestAiProviderModels(ctx context.Context, req *adminv1.TestAiProviderModelsRequest) (*adminv1.TestAiProviderModelsResponse, error) {
	providerForm := req.GetAiProvider()
	var oldProvider *models.AiProvider
	var err error
	if providerForm.GetId() > 0 {
		oldProvider, err = c.FindByID(ctx, providerForm.GetId())
		if err != nil {
			return nil, err
		}
		if providerForm.GetProvider() != oldProvider.Provider {
			return nil, errorsx.Conflict("Provider标识不可修改")
		}
	}
	var entity *models.AiProvider
	entity, err = aiProviderEntity(providerForm, oldProvider)
	if err != nil {
		return nil, err
	}
	if err = validateAiProvider(entity); err != nil {
		return nil, err
	}
	if entity.Provider == aiProviderOpenAICompatible && entity.APIKey == "" {
		return nil, errorsx.InvalidArgument("请配置模型API密钥后再测试")
	}
	var modelItems []dto.AiProviderModelConfig
	modelItems, err = parseAiProviderModels(entity.Models)
	if err != nil {
		return nil, errorsx.InvalidArgument("AI模型JSON格式无效").WithCause(err)
	}
	configs := make([]*modelconfig.ModelConfig, 0, len(modelItems))
	for _, item := range modelItems {
		var modelConfig *modelconfig.ModelConfig
		modelConfig, err = providerModelConfig(entity, item)
		if err != nil {
			return nil, err
		}
		configs = append(configs, modelConfig)
	}
	return &adminv1.TestAiProviderModelsResponse{Results: testAiProviderModels(ctx, configs)}, nil
}

// CreateAiProvider 创建AI供应商并刷新可用模型快照。
func (c *AiProviderCase) CreateAiProvider(ctx context.Context, req *adminv1.AiProviderForm) error {
	entity, err := aiProviderEntity(req, nil)
	if err != nil {
		return err
	}
	if err = validateAiProvider(entity); err != nil {
		return err
	}
	var authInfo *authData.UserTokenPayload
	authInfo, err = c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	entity.CreatedBy, entity.UpdatedBy = authInfo.UserId, authInfo.UserId
	entity.CreatedAt, entity.UpdatedAt = now, now
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		if err = c.Create(txCtx, entity); err != nil {
			if errorsx.IsDuplicateKey(err) {
				return errorsx.UniqueConflict("AI供应商名称重复", "ai_provider", "name", "unique_ai_provider").WithCause(err)
			}
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return c.RefreshAiProvider(ctx)
}

// UpdateAiProvider 更新AI供应商并刷新可用模型快照。
func (c *AiProviderCase) UpdateAiProvider(ctx context.Context, req *adminv1.AiProviderForm) error {
	oldProvider, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	if req.GetProvider() != oldProvider.Provider {
		return errorsx.Conflict("Provider标识不可修改")
	}
	var entity *models.AiProvider
	entity, err = aiProviderEntity(req, oldProvider)
	if err != nil {
		return err
	}
	if err = validateAiProvider(entity); err != nil {
		return err
	}
	var authInfo *authData.UserTokenPayload
	authInfo, err = c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	entity.ID, entity.CreatedBy, entity.CreatedAt = oldProvider.ID, oldProvider.CreatedBy, oldProvider.CreatedAt
	entity.UpdatedBy, entity.UpdatedAt = authInfo.UserId, time.Now()
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		query := c.Query(txCtx).AiProvider
		_, err = query.WithContext(txCtx).
			Where(query.ID.Eq(entity.ID)).
			Select(query.Provider, query.Name, query.BaseURL, query.APIKey, query.Models, query.Config, query.Sort, query.Status, query.UpdatedBy, query.UpdatedAt).
			Updates(entity)
		if err != nil {
			if errorsx.IsDuplicateKey(err) {
				return errorsx.UniqueConflict("AI供应商名称重复", "ai_provider", "name", "unique_ai_provider").WithCause(err)
			}
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return c.RefreshAiProvider(ctx)
}

// DeleteAiProvider 删除AI供应商并刷新可用模型快照。
func (c *AiProviderCase) DeleteAiProvider(ctx context.Context, id int64) error {
	_, err := c.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if err = c.DeleteByIDs(ctx, []int64{id}); err != nil {
		return err
	}
	return c.RefreshAiProvider(ctx)
}

// SetAiProviderStatus 设置AI供应商状态并刷新可用模型快照。
func (c *AiProviderCase) SetAiProviderStatus(ctx context.Context, req *adminv1.SetAiProviderStatusRequest) error {
	if req.GetStatus() != commonv1.Status_STATUS_ENABLE && req.GetStatus() != commonv1.Status_STATUS_DISABLE {
		return errorsx.InvalidArgument("AI供应商状态无效")
	}
	provider, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	if req.GetStatus() == commonv1.Status_STATUS_ENABLE {
		if err = validateAiProvider(provider); err != nil {
			return err
		}
	}
	if provider.Status == int32(req.GetStatus()) {
		return nil
	}
	if err = c.UpdateByID(ctx, &models.AiProvider{ID: provider.ID, Status: int32(req.GetStatus()), UpdatedAt: time.Now()}); err != nil {
		return err
	}
	return c.RefreshAiProvider(ctx)
}

// aiProviderEntity 将表单转换为记录，并在密钥留空时沿用原值。
func aiProviderEntity(req *adminv1.AiProviderForm, old *models.AiProvider) (*models.AiProvider, error) {
	apiKey := req.GetApiKey()
	if apiKey == "" && old != nil {
		apiKey = old.APIKey
	}
	config := "{}"
	if req.GetConfig() != nil {
		encoded, err := json.Marshal(req.GetConfig().AsMap())
		if err != nil {
			return nil, errorsx.InvalidArgument("Provider个性化配置无效").WithCause(err)
		}
		config = string(encoded)
	}
	return &models.AiProvider{
		Provider: req.GetProvider(), Name: req.GetName(),
		BaseURL: req.GetBaseUrl(), APIKey: apiKey, Models: req.GetModelsJson(), Config: config,
		Sort: req.GetSort(), Status: int32(req.GetStatus()),
	}, nil
}

// toAiProviderForm 转换表单并保证API密钥不会回传。
func toAiProviderForm(provider *models.AiProvider) (*adminv1.AiProviderForm, error) {
	config, err := parseAiProviderConfig(provider.Config)
	if err != nil {
		return nil, errorsx.Internal("解析AI Provider个性化配置失败").WithCause(err)
	}
	return &adminv1.AiProviderForm{
		Id: provider.ID, Provider: provider.Provider, Name: provider.Name,
		BaseUrl: provider.BaseURL, ApiKey: "", ApiKeyConfigured: provider.APIKey != "",
		ModelsJson: provider.Models, Config: config, Sort: provider.Sort, Status: commonv1.Status(provider.Status),
	}, nil
}

// parseAiProviderConfig 将数据库中的Provider个性化配置JSON对象转换为Proto Struct。
func parseAiProviderConfig(raw string) (*structpb.Struct, error) {
	values := make(map[string]any)
	var err error
	if raw != "" {
		err = json.Unmarshal([]byte(raw), &values)
		if err != nil {
			return nil, fmt.Errorf("Provider个性化配置必须是JSON对象: %w", err)
		}
	}
	return structpb.NewStruct(values)
}

// parseAiProviderModels 解析并校验供应商中的多模型JSON数组。
func parseAiProviderModels(raw string) ([]dto.AiProviderModelConfig, error) {
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	var values []dto.AiProviderModelConfig
	err := decoder.Decode(&values)
	if err != nil {
		return nil, fmt.Errorf("模型配置必须是JSON数组: %w", err)
	}
	var extra any
	err = decoder.Decode(&extra)
	if !errors.Is(err, io.EOF) {
		return nil, errors.New("模型配置JSON只能包含一个数组")
	}
	if len(values) == 0 {
		return nil, errors.New("至少需要配置一个模型")
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value.ModelName == "" || len(value.ModelName) > 100 {
			return nil, errors.New("模型名称不能为空且不能超过100个字符")
		}
		if len(value.DisplayName) > 100 {
			return nil, fmt.Errorf("模型 %s 的显示名称不能超过100个字符", value.ModelName)
		}
		if _, exists := seen[value.ModelName]; exists {
			return nil, fmt.Errorf("模型名称重复: %s", value.ModelName)
		}
		seen[value.ModelName] = struct{}{}
		if value.APIType != "CHAT_COMPLETIONS" && value.APIType != "RESPONSES" {
			return nil, fmt.Errorf("模型 %s 的 api_type 必须是 CHAT_COMPLETIONS 或 RESPONSES", value.ModelName)
		}
		if value.Temperature < 0 || value.Temperature > 2 || value.MaxTokens < 0 || value.MaxTokens > 200000 || value.TimeoutSeconds < 0 || value.TimeoutSeconds > 600 || value.MaxRetries < 0 || value.MaxRetries > 10 {
			return nil, fmt.Errorf("模型 %s 的请求参数超出范围", value.ModelName)
		}
	}
	return values, nil
}

// validateAiProvider 校验供应商连接信息、模型数组和启用条件。
func validateAiProvider(provider *models.AiProvider) error {
	if provider.Name == "" || len(provider.Name) > 100 {
		return errorsx.InvalidArgument("供应商名称不能为空且不能超过100个字符")
	}
	if provider.Status != int32(commonv1.Status_STATUS_ENABLE) && provider.Status != int32(commonv1.Status_STATUS_DISABLE) {
		return errorsx.InvalidArgument("AI供应商状态无效")
	}
	if provider.Provider != aiProviderOpenAICompatible && provider.Provider != aiProviderOllama {
		return errorsx.InvalidArgument("AI Provider标识无效")
	}
	var err error
	if _, err = parseAiProviderModels(provider.Models); err != nil {
		return errorsx.InvalidArgument("AI模型JSON格式无效").WithCause(err)
	}
	config, err := parseAiProviderConfig(provider.Config)
	if err != nil {
		return errorsx.InvalidArgument("Provider个性化配置JSON格式无效").WithCause(err)
	}
	if organization, exists := config.AsMap()["organization"]; exists {
		if _, ok := organization.(string); !ok {
			return errorsx.InvalidArgument("config.organization必须是字符串")
		}
	}
	var parsedURL *url.URL
	parsedURL, err = url.ParseRequestURI(provider.BaseURL)
	if err != nil {
		return errorsx.InvalidArgument("模型API基础地址必须是完整URL").WithCause(err)
	}
	if (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
		return errorsx.InvalidArgument("模型API基础地址必须是完整HTTP或HTTPS URL")
	}
	if provider.Provider == aiProviderOpenAICompatible && provider.Status == int32(commonv1.Status_STATUS_ENABLE) && provider.APIKey == "" {
		return errorsx.InvalidArgument("启用OpenAI兼容Provider前必须配置API密钥")
	}
	if provider.Sort < 0 {
		return errorsx.InvalidArgument("供应商排序不能小于0")
	}
	return nil
}

// providerModelConfig 将数据库供应商和模型条目组合为模型客户端参数。
func providerModelConfig(provider *models.AiProvider, item dto.AiProviderModelConfig) (*modelconfig.ModelConfig, error) {
	apiType := modelconfig.APITypeChatCompletions
	if item.APIType == "RESPONSES" {
		apiType = modelconfig.APITypeResponses
	}
	if provider.Provider != aiProviderOpenAICompatible && provider.Provider != aiProviderOllama {
		return nil, errors.New("不支持的AI Provider")
	}
	config, err := parseAiProviderConfig(provider.Config)
	if err != nil {
		return nil, err
	}
	organization, _ := config.AsMap()["organization"].(string)
	return &modelconfig.ModelConfig{
		Provider: modelconfig.Provider(provider.Provider), ModelName: item.ModelName, APIKey: provider.APIKey,
		BaseURL: provider.BaseURL, Organization: organization, Temperature: item.Temperature, MaxTokens: item.MaxTokens,
		TimeoutSeconds: item.TimeoutSeconds, MaxRetries: item.MaxRetries, APIType: apiType,
	}, nil
}

// testAiProviderModels 按配置顺序运行单模型测试，并隔离每次请求的时限和输出长度。
func testAiProviderModels(ctx context.Context, configs []*modelconfig.ModelConfig) []*adminv1.AiProviderModelTestResult {
	results := make([]*adminv1.AiProviderModelTestResult, 0, len(configs))
	for _, config := range configs {
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
			ModelName:  testConfig.ModelName,
			Success:    err == nil,
			DurationMs: int32(time.Since(startAt).Milliseconds()),
		}
		if err != nil {
			result.Message = aiProviderTestError(err, testConfig.APIKey)
		}
		results = append(results, result)
	}
	return results
}

// aiProviderTestError 清理并截断模型接口错误，避免把密钥等敏感信息回传给调用方。
func aiProviderTestError(err error, apiKey string) string {
	message := err.Error()
	if apiKey != "" {
		message = strings.ReplaceAll(message, apiKey, "[REDACTED]")
	}
	if runes := []rune(message); len(runes) > 512 {
		return string(runes[:512]) + "..."
	}
	return message
}
