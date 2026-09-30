package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	// maxSecretLength 敏感字段明文的最大长度。
	maxSecretLength            = 1024
	aiProviderOpenAICompatible = "openai_compatible"
	aiProviderOllama           = "ollama"
)

// AiProviderCase 管理AI供应商配置。
type AiProviderCase struct {
	*biz.BaseCase
	*data.AiProviderRepository
	tx          data.Transaction
	aiModelCase *AiModelCase
}

// NewAiProviderCase 创建AI供应商管理业务实例。
func NewAiProviderCase(baseCase *biz.BaseCase, tx data.Transaction, repo *data.AiProviderRepository, aiModelCase *AiModelCase) *AiProviderCase {
	return &AiProviderCase{BaseCase: baseCase, tx: tx, AiProviderRepository: repo, aiModelCase: aiModelCase}
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
	providerIDs := make([]int64, 0, len(providers))
	for _, provider := range providers {
		providerIDs = append(providerIDs, provider.ID)
	}
	modelNames := c.aiModelCase.ProviderModelNames(ctx, providerIDs)
	for _, provider := range providers {
		result = append(result, &adminv1.AiProvider{
			Id:               provider.ID,
			Name:             provider.Name,
			BaseUrl:          provider.BaseURL,
			ApiKeyConfigured: provider.APIKey != "",
			Sort:             provider.Sort,
			Provider:         provider.Provider,
			ModelNames:       modelNames[provider.ID],
			Status:           commonv1.Status(provider.Status),
			CreatedAt:        provider.CreatedAt.Format(time.DateTime),
			UpdatedAt:        provider.UpdatedAt.Format(time.DateTime),
		})
	}
	return &adminv1.PageAiProviderResponse{Providers: result, Total: int32(total)}, nil
}

// ListAiProviderOptions 查询已启用的AI供应商选项。
func (c *AiProviderCase) ListAiProviderOptions(ctx context.Context) (*adminv1.ListAiProviderOptionsResponse, error) {
	query := c.Query(ctx).AiProvider
	providers, err := c.List(ctx,
		repository.Where(query.Status.Eq(int32(commonv1.Status_STATUS_ENABLE))),
		repository.Order(query.Sort.Asc()),
		repository.Order(query.ID.Asc()),
	)
	if err != nil {
		return nil, fmt.Errorf("读取AI供应商失败: %w", err)
	}
	options := make([]*adminv1.AiProviderOption, 0, len(providers))
	for _, provider := range providers {
		options = append(options, &adminv1.AiProviderOption{Id: provider.ID, Name: provider.Name, Provider: provider.Provider})
	}
	return &adminv1.ListAiProviderOptionsResponse{Options: options}, nil
}

// GetAiProvider 查询AI供应商表单（含模型列表）并隐藏密钥原文。
func (c *AiProviderCase) GetAiProvider(ctx context.Context, id int64) (*adminv1.AiProviderForm, error) {
	provider, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	form, err := toAiProviderForm(provider)
	if err != nil {
		return nil, err
	}
	models, err := c.aiModelCase.ListProviderModelForms(ctx, id)
	if err != nil {
		return nil, err
	}
	form.Models = models
	return form, nil
}

// CreateAiProvider 创建供应商并在同一事务保存其模型列表，随后刷新聊天模型快照。
func (c *AiProviderCase) CreateAiProvider(ctx context.Context, req *adminv1.AiProviderForm) error {
	entity, err := aiProviderEntity(req, nil)
	if err != nil {
		return err
	}
	if err = validateAiProvider(entity); err != nil {
		return err
	}
	authInfo, err := c.GetAuthInfo(ctx)
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
		return c.aiModelCase.SyncProviderModels(txCtx, entity, req.GetModels())
	})
	if err != nil {
		return err
	}
	return c.aiModelCase.RefreshModels(ctx)
}

// UpdateAiProvider 更新供应商并在同一事务同步其模型列表，随后刷新聊天模型快照。
func (c *AiProviderCase) UpdateAiProvider(ctx context.Context, req *adminv1.AiProviderForm) error {
	oldProvider, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	if req.GetProvider() != oldProvider.Provider {
		return errorsx.Conflict("Provider标识不可修改")
	}
	entity, err := aiProviderEntity(req, oldProvider)
	if err != nil {
		return err
	}
	if err = validateAiProvider(entity); err != nil {
		return err
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	entity.ID, entity.CreatedBy, entity.CreatedAt = oldProvider.ID, oldProvider.CreatedBy, oldProvider.CreatedAt
	entity.UpdatedBy, entity.UpdatedAt = authInfo.UserId, time.Now()
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		query := c.Query(txCtx).AiProvider
		_, err = query.WithContext(txCtx).
			Where(query.ID.Eq(entity.ID)).
			Select(query.Provider, query.Name, query.BaseURL, query.APIKey, query.Config, query.Sort, query.Status, query.UpdatedBy, query.UpdatedAt).
			Updates(entity)
		if err != nil {
			if errorsx.IsDuplicateKey(err) {
				return errorsx.UniqueConflict("AI供应商名称重复", "ai_provider", "name", "unique_ai_provider").WithCause(err)
			}
			return err
		}
		return c.aiModelCase.SyncProviderModels(txCtx, entity, req.GetModels())
	})
	if err != nil {
		return err
	}
	return c.aiModelCase.RefreshModels(ctx)
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
	entity, err := aiProviderEntity(providerForm, oldProvider)
	if err != nil {
		return nil, err
	}
	if err = validateAiProvider(entity); err != nil {
		return nil, err
	}
	if entity.Provider == aiProviderOpenAICompatible && entity.APIKey == "" {
		return nil, errorsx.InvalidArgument("请配置模型API密钥后再测试")
	}
	results, err := c.aiModelCase.TestProviderModels(ctx, entity, providerForm.GetModels())
	if err != nil {
		return nil, err
	}
	return &adminv1.TestAiProviderModelsResponse{Results: results}, nil
}

// DeleteAiProvider 删除AI供应商并级联删除其下模型，随后刷新聊天模型快照。
func (c *AiProviderCase) DeleteAiProvider(ctx context.Context, id int64) error {
	if _, err := c.FindByID(ctx, id); err != nil {
		return err
	}
	if err := c.tx.Transaction(ctx, func(txCtx context.Context) error {
		if err := c.DeleteByIDs(txCtx, []int64{id}); err != nil {
			return err
		}
		return c.aiModelCase.DeleteByProvider(txCtx, id)
	}); err != nil {
		return err
	}
	return c.aiModelCase.RefreshModels(ctx)
}

// SetAiProviderStatus 设置AI供应商状态并刷新聊天模型快照。
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
	return c.aiModelCase.RefreshModels(ctx)
}

// aiProviderEntity 将表单转换为记录，并在密钥留空时沿用原值。
func aiProviderEntity(req *adminv1.AiProviderForm, old *models.AiProvider) (*models.AiProvider, error) {
	var apiKey string
	if req.GetApiKey() != nil {
		apiKey = req.GetApiKey().GetText()
		if len(apiKey) > maxSecretLength {
			return nil, errorsx.InvalidArgument("模型密钥不能超过1024个字符")
		}
	}
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
		BaseURL: req.GetBaseUrl(), APIKey: apiKey, Config: config,
		Sort: req.GetSort(), Status: int32(req.GetStatus()),
	}, nil
}

// toAiProviderForm 转换表单并保证API密钥不会回传。
func toAiProviderForm(provider *models.AiProvider) (*adminv1.AiProviderForm, error) {
	config, err := parseConfigStruct(provider.Config)
	if err != nil {
		return nil, errorsx.Internal("解析AI Provider个性化配置失败").WithCause(err)
	}
	return &adminv1.AiProviderForm{
		Id: provider.ID, Provider: provider.Provider, Name: provider.Name,
		BaseUrl: provider.BaseURL, ApiKeyConfigured: provider.APIKey != "",
		Config: config, Sort: provider.Sort, Status: commonv1.Status(provider.Status),
	}, nil
}

// parseConfigStruct 将数据库中的配置JSON对象转换为Proto Struct。
func parseConfigStruct(raw string) (*structpb.Struct, error) {
	values := make(map[string]any)
	var err error
	if raw != "" {
		err = json.Unmarshal([]byte(raw), &values)
		if err != nil {
			return nil, fmt.Errorf("配置必须是JSON对象: %w", err)
		}
	}
	return structpb.NewStruct(values)
}

// validateAiProvider 校验供应商连接信息和启用条件。
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
	config, err := parseConfigStruct(provider.Config)
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
