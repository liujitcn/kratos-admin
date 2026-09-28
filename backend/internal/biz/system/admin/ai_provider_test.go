package biz

import (
	"testing"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-kit/ai/model"
	"google.golang.org/protobuf/types/known/structpb"
)

// TestParseAiProviderModelsSupportsMultipleModels 验证供应商可以维护多个模型配置。
func TestParseAiProviderModelsSupportsMultipleModels(t *testing.T) {
	values, err := parseAiProviderModels(`[
  {"model_name":"model-a","api_type":"CHAT_COMPLETIONS"},
  {"model_name":"model-b","api_type":"RESPONSES","max_tokens":2048}
]`)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 || values[0].ModelName != "model-a" || values[1].ModelName != "model-b" {
		t.Fatalf("unexpected model configuration: %+v", values)
	}
}

// TestParseAiProviderModelsRejectsInvalidEntries 验证重复模型和未知字段会被拒绝。
func TestParseAiProviderModelsRejectsInvalidEntries(t *testing.T) {
	var err error
	_, err = parseAiProviderModels(`[{"model_name":"same","api_type":"CHAT_COMPLETIONS"},{"model_name":"same","api_type":"RESPONSES"}]`)
	if err == nil {
		t.Fatal("duplicate model names should be rejected")
	}
	err = nil
	_, err = parseAiProviderModels(`[{"model_name":"model-a","api_type":"CHAT_COMPLETIONS","api_key":"secret"}]`)
	if err == nil {
		t.Fatal("unknown model fields should be rejected")
	}
}

// TestAiProviderFormNeverReturnsApiKey 验证供应商表单只返回密钥已配置状态。
func TestAiProviderFormNeverReturnsApiKey(t *testing.T) {
	config, err := structpb.NewStruct(map[string]any{"organization": "org-a"})
	if err != nil {
		t.Fatal(err)
	}
	form, err := toAiProviderForm(&models.AiProvider{ID: 1, APIKey: "stored-secret", Models: "[]", Config: `{"organization":"org-a"}`})
	if err != nil {
		t.Fatal(err)
	}
	if form.GetApiKey() != "" || !form.GetApiKeyConfigured() || form.GetConfig().GetFields()["organization"].GetStringValue() != config.GetFields()["organization"].GetStringValue() {
		t.Fatalf("form exposed the stored key or lost configured state: %+v", form)
	}
}

// TestAiProviderEntityPreservesApiKey 验证更新时密钥留空会保留已保存密钥。
func TestAiProviderEntityPreservesApiKey(t *testing.T) {
	old := &models.AiProvider{Provider: aiProviderOpenAICompatible, APIKey: "stored-secret"}
	form := &adminv1.AiProviderForm{
		Provider:   aiProviderOpenAICompatible,
		ModelsJson: `[{"model_name":"model-a","api_type":"CHAT_COMPLETIONS"}]`,
		Status:     commonv1.Status_STATUS_DISABLE,
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
		Models:   `[{"model_name":"model-a","api_type":"CHAT_COMPLETIONS"}]`,
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

// TestOllamaProviderUsesUnifiedBaseURLAndOptionalApiKey 验证Ollama使用统一地址且不强制要求API Key。
func TestOllamaProviderUsesUnifiedBaseURLAndOptionalApiKey(t *testing.T) {
	provider := &models.AiProvider{
		Provider: aiProviderOllama,
		Name:     "local-ollama",
		BaseURL:  "http://localhost:11434/v1",
		Models:   `[{"model_name":"llama3","api_type":"CHAT_COMPLETIONS"}]`,
		Config:   "{}",
		Status:   int32(commonv1.Status_STATUS_ENABLE),
	}
	if err := validateAiProvider(provider); err != nil {
		t.Fatalf("Ollama provider should not require an API key: %v", err)
	}
	modelConfig, err := providerModelConfig(provider, dto.AiProviderModelConfig{ModelName: "llama3", APIType: "CHAT_COMPLETIONS"})
	if err != nil {
		t.Fatal(err)
	}
	if modelConfig.Provider != model.ProviderOllama || modelConfig.ResolvedAPIKey() != "ollama" || modelConfig.BaseURL != provider.BaseURL {
		t.Fatalf("unexpected Ollama model configuration: %+v", modelConfig)
	}
}
