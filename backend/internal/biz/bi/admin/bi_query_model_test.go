package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/pkg/agent"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
	"github.com/liujitcn/kratos-core/biz"
	modelconfig "github.com/liujitcn/kratos-kit/ai/model"
)

// TestAskBiQueryUsesSelectedModel 验证问数实际调用指定模型、沿用对话模型，并拒绝无效选择。
func TestAskBiQueryUsesSelectedModel(t *testing.T) {
	for _, test := range []struct {
		name         string
		providerID   int64
		modelName    string
		useChatModel bool
		wantClient   int
		wantError    bool
	}{
		{name: "未指定时使用默认模型", wantClient: 0},
		{name: "选择同供应商的另一个模型", providerID: 1, modelName: "model-b", wantClient: 1},
		{name: "按供应商区分同名模型", providerID: 2, modelName: "model-b", wantClient: 2},
		{name: "对话工具沿用本轮模型", providerID: 1, modelName: "model-a", useChatModel: true, wantClient: 2},
		{name: "供应商无效", providerID: 9, modelName: "model-a", wantError: true},
		{name: "模型无效", providerID: 1, modelName: "missing", wantError: true},
		{name: "模型选择不完整", providerID: 1, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls [3]atomic.Int32
			providers := []model.ProviderModel{
				{ProviderID: 1, Config: &modelconfig.ModelConfig{ModelName: "model-a"}},
				{ProviderID: 1, Config: &modelconfig.ModelConfig{ModelName: "model-b"}},
				{ProviderID: 2, Config: &modelconfig.ModelConfig{ModelName: "model-b"}},
			}
			for index, provider := range providers {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
					var payload struct {
						Model string `json:"model"`
					}
					err := json.NewDecoder(request.Body).Decode(&payload)
					if err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					if payload.Model != provider.Config.ModelName {
						t.Errorf("模型请求 = %q，期望 %q", payload.Model, provider.Config.ModelName)
					}
					calls[index].Add(1)
					w.Header().Set("Content-Type", "application/json")
					if err = json.NewEncoder(w).Encode(map[string]any{
						"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": `{"query":{"source_code":"test","measures":[{"code":"count","aggregation":"count"}]},"chart_type":"metric","title":"用户数量"}`}, "finish_reason": "stop"}},
					}); err != nil {
						t.Error(err)
					}
				}))
				t.Cleanup(server.Close)
				provider.Config.Provider = modelconfig.ProviderOpenAICompatible
				provider.Config.APIKey = "test"
				provider.Config.BaseURL = server.URL
			}
			modelStore := model.NewRegistry()
			modelStore.Replace(providers)
			registry := bi.NewRegistry()
			err := registry.Register(biQueryRecordTestSource{})
			if err != nil {
				t.Fatal(err)
			}
			ctx := biCardTestContext(1, 10, 42)
			if test.useChatModel {
				var client *model.AssistantClient
				client, err = modelStore.ResolveAssistantClient(2, "model-b")
				if err != nil {
					t.Fatal(err)
				}
				ctx = agent.ContextWithAssistantClient(ctx, client)
			}
			store := newBiQueryRecordTestStore(t)
			baseCase := &biz.BaseCase{}
			sourceCase := NewBiSourceCase(baseCase, store, data.NewBiSourceRepository(store), registry, nil)
			c := NewBiQueryCase(baseCase, sourceCase, modelStore, data.NewBiQueryRepository(store))
			_, err = c.AskBiQuery(ctx, &adminv1.AskBiQueryRequest{Question: "统计用户数量", ProviderId: test.providerID, ModelName: test.modelName})
			if test.wantError {
				if err == nil || !strings.Contains(err.Error(), "AI供应商或模型选择无效") {
					t.Fatalf("无效选择未被拒绝: %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			for index := range calls {
				want := int32(0)
				if !test.wantError && index == test.wantClient {
					want = 1
				}
				if got := calls[index].Load(); got != want {
					t.Errorf("客户端 %d 调用次数 = %d，期望 %d", index, got, want)
				}
			}
		})
	}
}
