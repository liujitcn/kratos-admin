package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/liujitcn/go-utils/translator"
	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-kit/ai/model"

	"github.com/sashabaranov/go-openai"
)

// translationRetry 是翻译请求的默认重试次数。
const translationRetry = 2

// translationLanguageNames 将语言代码映射为翻译提示词中的语言名称。
var translationLanguageNames = map[string]string{
	"ar": "Arabic", "de": "German", "en": "English", "es": "Spanish", "fr": "French",
	"it": "Italian", "ja": "Japanese", "ko": "Korean", "pt": "Portuguese", "ru": "Russian",
	"th": "Thai", "vi": "Vietnamese", "zh": "Chinese", "zh-cn": "Chinese",
	"zh-hk": "Traditional Chinese", "zh-tw": "Traditional Chinese",
}

// ModelTranslator 基于已启用的翻译模型分类（AI_MODEL_CATEGORY_TRANSLATION）提供机器翻译。
type ModelTranslator struct {
	aiProviderRepo *data.AiProviderRepository
	aiModelRepo    *data.AiModelRepository
}

// NewModelTranslator 创建翻译模型翻译器。
func NewModelTranslator(aiProviderRepo *data.AiProviderRepository, aiModelRepo *data.AiModelRepository) *ModelTranslator {
	return &ModelTranslator{aiProviderRepo: aiProviderRepo, aiModelRepo: aiModelRepo}
}

// Available 判断当前是否存在已启用的翻译模型。
func (t *ModelTranslator) Available(ctx context.Context) (bool, error) {
	provider, _, err := t.loadTranslationModel(ctx)
	if err != nil {
		return false, err
	}
	return provider != nil, nil
}

// Translate 使用翻译模型生成译文，提示词遵循 Hunyuan-MT 官方模板。
func (t *ModelTranslator) Translate(ctx context.Context, source, sourceLang, targetLang string) (string, error) {
	if strings.TrimSpace(source) == "" {
		return "", nil
	}
	provider, aiModel, err := t.loadTranslationModel(ctx)
	if err != nil {
		return "", err
	}
	if provider == nil {
		return "", errorsx.PermissionDenied("机器翻译功能未启用")
	}
	config, err := ParseTranslationModelConfig(aiModel.Config)
	if err != nil {
		return "", fmt.Errorf("解析翻译模型配置失败: %w", err)
	}
	client, err := model.NewClient(&model.ModelConfig{
		Provider: model.Provider(provider.Provider), ModelName: aiModel.ModelName, APIKey: provider.APIKey,
		BaseURL: provider.BaseURL, TimeoutSeconds: config.TimeoutSeconds, MaxRetries: config.MaxRetries,
	})
	if err != nil {
		return "", fmt.Errorf("创建翻译模型客户端失败: %w", err)
	}
	if config.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(config.TimeoutSeconds)*time.Second)
		defer cancel()
	}
	retries := translationRetry
	if config.MaxRetries > 0 {
		retries = int(config.MaxRetries)
	}
	request := openai.ChatCompletionRequest{
		Model: aiModel.ModelName,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: translationPrompt(targetLang, source)},
		},
	}
	var lastErr error
	for attempt := 0; attempt < retries; attempt++ {
		var response openai.ChatCompletionResponse
		response, lastErr = client.CreateChatCompletion(ctx, request)
		if lastErr == nil {
			if len(response.Choices) == 0 || strings.TrimSpace(response.Choices[0].Message.Content) == "" {
				return "", fmt.Errorf("翻译模型未返回译文")
			}
			return strings.TrimSpace(response.Choices[0].Message.Content), nil
		}
	}
	return "", fmt.Errorf("调用翻译模型失败: %w", lastErr)
}

// loadTranslationModel 读取按排序最靠前的启用翻译模型及其供应商；
// 无启用模型或供应商被停用时返回空记录，表示翻译能力未配置。
func (t *ModelTranslator) loadTranslationModel(ctx context.Context) (*models.AiProvider, *models.AiModel, error) {
	query := t.aiModelRepo.Query(ctx).AiModel
	items, err := t.aiModelRepo.List(ctx,
		repository.Where(query.Category.Eq(int32(adminv1.AiModelCategory_AI_MODEL_CATEGORY_TRANSLATION))),
		repository.Where(query.Status.Eq(int32(commonv1.Status_STATUS_ENABLE))),
		repository.Order(query.Sort.Asc()), repository.Order(query.ID.Asc()),
		repository.Limit(1),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("读取翻译模型失败: %w", err)
	}
	if len(items) == 0 {
		return nil, nil, nil
	}
	providerQuery := t.aiProviderRepo.Query(ctx).AiProvider
	providers, err := t.aiProviderRepo.List(ctx,
		repository.Where(providerQuery.ID.Eq(items[0].ProviderID)),
		repository.Limit(1),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("读取翻译模型供应商失败: %w", err)
	}
	if len(providers) == 0 {
		return nil, nil, fmt.Errorf("翻译模型供应商 %d 不存在", items[0].ProviderID)
	}
	if providers[0].Status != int32(commonv1.Status_STATUS_ENABLE) {
		return nil, nil, nil
	}
	return providers[0], items[0], nil
}

// translationPrompt 按 Hunyuan-MT 官方模板构造翻译指令，未识别的语言代码原样透传。
func translationPrompt(targetLang, source string) string {
	return fmt.Sprintf("Translate the following segment into %s, without additional explanation.\n\n%s",
		translationLanguageName(targetLang), source)
}

// translationLanguageName 返回提示词使用的目标语言名称，优先精确匹配语言代码，再回退语言前缀。
func translationLanguageName(locale string) string {
	lower := strings.ToLower(locale)
	if name, ok := translationLanguageNames[lower]; ok {
		return name
	}
	if prefix, _, _ := strings.Cut(lower, "-"); prefix != lower {
		if name, ok := translationLanguageNames[prefix]; ok {
			return name
		}
	}
	return locale
}

// 确保适配器实现 kit 翻译接口。
var _ translator.Translator = (*ModelTranslator)(nil)
