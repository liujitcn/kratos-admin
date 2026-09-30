package biz

import (
	"testing"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-kit/ai/model"
	"google.golang.org/protobuf/types/known/structpb"
)

// TestAiProviderFormNeverReturnsApiKey 验证供应商表单只返回密钥已配置状态。
func TestAiProviderFormNeverReturnsApiKey(t *testing.T) {
	config, err := structpb.NewStruct(map[string]any{"organization": "org-a"})
	if err != nil {
		t.Fatal(err)
	}
	form, err := toAiProviderForm(&models.AiProvider{ID: 1, APIKey: "stored-secret", Config: `{"organization":"org-a"}`})
	if err != nil {
		t.Fatal(err)
	}
	if form.GetApiKey() != nil || !form.GetApiKeyConfigured() || form.GetConfig().GetFields()["organization"].GetStringValue() != config.GetFields()["organization"].GetStringValue() {
		t.Fatalf("form exposed the stored key or lost configured state: %+v", form)
	}
}

// TestAiProviderEntityPreservesApiKey 验证更新时密钥留空会保留已保存密钥。
func TestAiProviderEntityPreservesApiKey(t *testing.T) {
	old := &models.AiProvider{Provider: aiProviderOpenAICompatible, APIKey: "stored-secret"}
	form := &adminv1.AiProviderForm{
		Provider: aiProviderOpenAICompatible,
		Status:   commonv1.Status_STATUS_DISABLE,
	}
	entity, err := aiProviderEntity(form, old)
	if err != nil {
		t.Fatal(err)
	}
	if entity.APIKey != old.APIKey {
		t.Fatal("empty update key should preserve the saved key")
	}
	form.Provider = aiProviderOllama
	entity, err = aiProviderEntity(form, old)
	if err != nil {
		t.Fatal(err)
	}
	if entity.APIKey != old.APIKey {
		t.Fatal("provider changes should not silently clear the saved API key")
	}
}

// TestValidateAiProviderRequiresKeyWhenOpenAICompatibleEnabled 验证启用OpenAI兼容Provider必须提供密钥。
func TestValidateAiProviderRequiresKeyWhenOpenAICompatibleEnabled(t *testing.T) {
	provider := &models.AiProvider{
		Provider: aiProviderOpenAICompatible,
		Name:     "test-provider",
		BaseURL:  "https://example.com/v1",
		Config:   "{}",
		Status:   int32(commonv1.Status_STATUS_ENABLE),
	}
	if err := validateAiProvider(provider); err == nil {
		t.Fatal("enabled OpenAI-compatible provider without API key should be rejected")
	}
	provider.APIKey = "configured-secret"
	if err := validateAiProvider(provider); err != nil {
		t.Fatalf("configured OpenAI-compatible provider should be valid: %v", err)
	}
}

// TestChatModelConfigBuildsClientSettings 验证聊天模型配置能生成正确的客户端参数。
func TestChatModelConfigBuildsClientSettings(t *testing.T) {
	provider := &models.AiProvider{
		Provider: aiProviderOllama,
		Name:     "local-ollama",
		BaseURL:  "http://localhost:11434/v1",
		Config:   "{}",
	}
	item := &models.AiModel{
		ModelName: "llama3",
		Category:  int32(aiModelCategoryChat),
		Config:    `{"api_type":"CHAT_COMPLETIONS","temperature":0.2,"max_tokens":1024,"timeout_seconds":60,"max_retries":2}`,
	}
	modelConfig, err := chatModelConfig(provider, item)
	if err != nil {
		t.Fatal(err)
	}
	if modelConfig.Provider != model.ProviderOllama || modelConfig.ResolvedAPIKey() != "ollama" || modelConfig.BaseURL != provider.BaseURL {
		t.Fatalf("unexpected Ollama model configuration: %+v", modelConfig)
	}
	if modelConfig.ModelName != "llama3" || modelConfig.Temperature != 0.2 || modelConfig.MaxTokens != 1024 {
		t.Fatalf("unexpected chat model settings: %+v", modelConfig)
	}
}

// TestValidateAiModelRejectsInvalidCategoryConfig 验证模型分类与分类配置的校验边界。
func TestValidateAiModelRejectsInvalidCategoryConfig(t *testing.T) {
	base := models.AiModel{
		ModelName: "model-a",
		Category:  int32(aiModelCategoryChat),
		Config:    `{"api_type":"CHAT_COMPLETIONS"}`,
		Status:    int32(commonv1.Status_STATUS_ENABLE),
	}
	if err := validateAiModel(&base); err != nil {
		t.Fatalf("valid chat model should pass: %v", err)
	}
	badCategory := base
	badCategory.Category = 99
	if err := validateAiModel(&badCategory); err == nil {
		t.Fatal("unknown category should be rejected")
	}
	badConfig := base
	badConfig.Config = `{"api_type":"UNSUPPORTED"}`
	if err := validateAiModel(&badConfig); err == nil {
		t.Fatal("invalid chat api_type should be rejected")
	}
	unknownField := base
	unknownField.Config = `{"api_type":"CHAT_COMPLETIONS","api_key":"secret"}`
	if err := validateAiModel(&unknownField); err == nil {
		t.Fatal("unknown chat config fields should be rejected")
	}
	embedding := base
	embedding.Category = int32(aiModelCategoryEmbedding)
	embedding.Config = `{"dimensions":1024}`
	if err := validateAiModel(&embedding); err != nil {
		t.Fatalf("valid embedding model should pass: %v", err)
	}
	badDims := embedding
	badDims.Config = `{"dimensions":10}`
	if err := validateAiModel(&badDims); err == nil {
		t.Fatal("embedding dimensions below 64 should be rejected")
	}
}
