package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/securityconfig"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
	"github.com/liujitcn/kratos-core/biz"
	modelconfig "github.com/liujitcn/kratos-kit/ai/model"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	"github.com/liujitcn/kratos-kit/database/gorm/driver"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// TestBiCardSourceAndIsolation 验证问数只复用智能问数记录，卡片拒绝预览来源和跨账号引用。
func TestBiCardSourceAndIsolation(t *testing.T) {
	c, _ := newBiWorkflowTestCases(t)
	ctx := biCardTestContext(1, 10, 42)
	spec := &adminv1.BiQuerySpec{SourceCode: "test", Measures: []*adminv1.BiMeasureSpec{{Code: "count"}}}
	manual, err := c.biQueryCase.PreviewBiQuery(ctx, &adminv1.PreviewBiQueryRequest{Query: spec})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.SaveBiCard(ctx, &adminv1.SaveBiCardRequest{Card: &adminv1.BiCardForm{Name: "禁止手动预览来源", Query: spec, ChartType: "metric", QueryRecordId: manual.QueryRecordId}}); err == nil {
		t.Fatal("手动预览绕过了智能问数来源限制")
	}
	var asked *adminv1.AskBiQueryResponse
	asked, err = c.biQueryCase.AskBiQuery(ctx, &adminv1.AskBiQueryRequest{Question: "统计用户数"})
	if err != nil {
		t.Fatal(err)
	}
	result := asked.Result
	if err != nil || result.QueryRecordId <= 0 {
		t.Fatalf("查询未生成可引用记录: %v %v", result, err)
	}
	if result.QueryRecordId == manual.QueryRecordId {
		t.Fatal("智能问数复用了不能建卡的普通预览记录")
	}
	var repeated *adminv1.AskBiQueryResponse
	repeated, err = c.biQueryCase.AskBiQuery(ctx, &adminv1.AskBiQueryRequest{Question: "统计用户数"})
	if err != nil {
		t.Fatal(err)
	}
	if repeated.Result.QueryRecordId != result.QueryRecordId {
		t.Fatal("相同智能问数没有复用成功记录")
	}
	form := &adminv1.BiCardForm{Name: "用户数", Query: spec, ChartType: "metric", QueryRecordId: result.QueryRecordId}
	var saved *adminv1.BiCard
	saved, err = c.SaveBiCard(ctx, &adminv1.SaveBiCardRequest{Card: form})
	if err != nil {
		t.Fatal(err)
	}
	if saved.QueryRecordId != result.QueryRecordId || saved.ChartJson == "" {
		t.Fatal("卡片未关联结果或展示配置")
	}
	if _, err = c.SaveBiCard(biCardTestContext(1, 20, 42), &adminv1.SaveBiCardRequest{Card: form}); err == nil {
		t.Fatal("允许引用他人查询记录")
	}
	if _, err = c.GetBiCardData(biCardTestContext(1, 20, 42), &adminv1.GetBiCardDataRequest{Id: saved.Id}); err == nil {
		t.Fatal("私有卡片泄漏")
	}
	if _, err = c.GetBiCardData(biCardTestContext(2, 10, 42), &adminv1.GetBiCardDataRequest{Id: saved.Id}); err == nil {
		t.Fatal("跨租户卡片泄漏")
	}
	form.Query.Limit = 50
	if _, err = c.SaveBiCard(ctx, &adminv1.SaveBiCardRequest{Card: form}); err == nil || !strings.Contains(err.Error(), "重新问数") {
		t.Fatalf("允许伪造来源记录的查询: %v", err)
	}
}

// TestBiCardAllChartShapes 验证十三类图表和指标卡的字段边界。
func TestBiCardAllChartShapes(t *testing.T) {
	var err error
	spec := bi.QuerySpec{Dimensions: []string{"group", "target"}, Measures: []bi.MeasureSpec{{Code: "value"}, {Code: "other"}}}
	result := bi.QueryResult{Columns: []bi.ResultColumn{{Code: "group", Type: bi.FieldTypeString}, {Code: "target", Type: bi.FieldTypeString}, {Code: "value", Type: bi.FieldTypeNumber}, {Code: "other", Type: bi.FieldTypeNumber}}, Rows: [][]string{{"A", "B", "10", "20"}}}
	for _, chartType := range []string{"bar", "line", "area", "pie", "donut", "radar", "funnel", "treemap", "sunburst", "sankey", "heatmap", "scatter", "gauge", "metric"} {
		current := spec
		raw := ""
		if chartType == "scatter" {
			raw = `{"encoding":{"dimension":"value","measure":"other"},"style":{}}`
		}
		if chartType == "gauge" || chartType == "metric" {
			current.Dimensions = nil
		}
		_, err = validateCardChart(chartType, raw, current, result)
		if err != nil {
			t.Fatalf("图型 %s 不可保存: %v", chartType, err)
		}
	}
	for _, raw := range []string{`{"encoding":{"measure":"secret"}}`, `{"options":{"url":"https://example.com"}}`} {
		if _, err = validateCardChart("bar", raw, spec, result); err == nil {
			t.Fatalf("接受非法配置: %s", raw)
		}
	}
	if _, err = validateCardChart("scatter", `{"encoding":{"dimension":"group","measure":"value"}}`, spec, result); err == nil {
		t.Fatal("散点图接受非数值横轴")
	}
}

// TestBiCardMetricGridValidation 验证指标网格的当前列、比较列和单位边界。
func TestBiCardMetricGridValidation(t *testing.T) {
	spec := bi.QuerySpec{Measures: []bi.MeasureSpec{{Code: "revenue"}, {Code: "revenue_prev"}, {Code: "orders"}, {Code: "orders_prev"}}}
	result := bi.QueryResult{Columns: []bi.ResultColumn{
		{Code: "revenue", Type: bi.FieldTypeNumber, Unit: "元"},
		{Code: "revenue_prev", Type: bi.FieldTypeNumber, Unit: "元"},
		{Code: "orders", Type: bi.FieldTypeNumber, Unit: "单"},
		{Code: "orders_prev", Type: bi.FieldTypeNumber, Unit: "单"},
		{Code: "status", Type: bi.FieldTypeString},
	}}
	valid, err := validateCardChart("metric_grid", `{"metricGrid":{"items":[{"value":"revenue","compare":"revenue_prev"},{"value":"orders","compare":"orders_prev","compareLabel":"较昨日"}]}}`, spec, result)
	if err != nil || !strings.Contains(valid, `"metricGrid"`) {
		t.Fatalf("合法指标网格未通过: %s %v", valid, err)
	}

	tooMany := `{"metricGrid":{"items":[{"value":"revenue"},{"value":"revenue_prev"},{"value":"orders"},{"value":"orders_prev"},{"value":"revenue"}]}}`
	cases := []struct {
		name   string
		raw    string
		spec   bi.QuerySpec
		result bi.QueryResult
	}{
		{name: "当前字段不存在", raw: `{"metricGrid":{"items":[{"value":"missing"}]}}`, spec: spec, result: result},
		{name: "比较字段不存在", raw: `{"metricGrid":{"items":[{"value":"revenue","compare":"missing"}]}}`, spec: spec, result: result},
		{name: "比较字段不是数值", raw: `{"metricGrid":{"items":[{"value":"revenue","compare":"status"}]}}`, spec: spec, result: result},
		{name: "当前和比较字段相同", raw: `{"metricGrid":{"items":[{"value":"revenue","compare":"revenue"}]}}`, spec: spec, result: result},
		{name: "单位不一致", raw: `{"metricGrid":{"items":[{"value":"revenue","compare":"revenue_prev"}]}}`, spec: spec, result: bi.QueryResult{Columns: []bi.ResultColumn{
			{Code: "revenue", Type: bi.FieldTypeNumber, Unit: "元"},
			{Code: "revenue_prev", Type: bi.FieldTypeNumber, Unit: "美元"},
		}}},
		{name: "超过四个单元", raw: tooMany, spec: spec, result: result},
		{name: "包含分组维度", raw: `{"metricGrid":{"items":[{"value":"revenue"}]}}`, spec: bi.QuerySpec{Dimensions: []string{"day"}, Measures: []bi.MeasureSpec{{Code: "revenue"}}}, result: result},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			if _, err = validateCardChart("metric_grid", item.raw, item.spec, item.result); err == nil {
				t.Fatalf("非法指标网格被接受: %s", item.raw)
			}
		})
	}
}

// newBiWorkflowTestCases 使用真实仓储验证卡片、仪表盘和工作台闭环。
func newBiWorkflowTestCases(t *testing.T) (*BiCardCase, *BiDashboardCase) {
	t.Helper()
	driver.Opens["bi-workflow-sqlite"] = sqlite.Open
	modelList := []any{&models.BiCard{}, &models.BiWorkbench{}, &models.BiQuery{}, &models.BiDashboard{}, &models.BiDashboardVersion{}, &models.BiSource{}, &models.BaseUser{}}
	client, cleanup, err := databaseGorm.NewGormClient(&configv1.Data_Database{Driver: "bi-workflow-sqlite", Source: filepath.Join(t.TempDir(), "workflow.db")}, databaseGorm.WithMigrateModels(modelList...))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	for _, item := range modelList {
		stmt := &gorm.Statement{DB: client.DB}
		if err = stmt.Parse(item); err != nil {
			t.Fatal(err)
		}
		stmt.Schema.FieldsByName["ID"].DataType = schema.Int
	}
	if err = client.AutoMigrate(modelList...); err != nil {
		t.Fatal(err)
	}
	var store *data.Data
	store, err = data.NewData(map[string]*databaseGorm.Client{databaseGorm.DefaultClientName: client})
	if err != nil {
		t.Fatal(err)
	}
	registry := bi.NewRegistry()
	if err = registry.Register(biQueryRecordTestSource{}); err != nil {
		t.Fatal(err)
	}
	base := &biz.BaseCase{GormClients: map[string]*databaseGorm.Client{databaseGorm.DefaultClientName: client}}
	source := NewBiSourceCase(base, store, data.NewBiSourceRepository(store), registry, nil, data.NewBiCardRepository(store), data.NewBiDashboardRepository(store), data.NewBiDashboardVersionRepository(store), nil)
	securityRuntimeConfig := securityconfig.Current()
	securityRuntimeConfig.AIAllowedCIDRs = []string{"127.0.0.1/32"}
	securityconfig.Set(securityRuntimeConfig)
	t.Cleanup(func() { securityconfig.Set(securityconfig.Default()) })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		encodeErr := json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": `{"query":{"source_code":"test","measures":[{"code":"count"}],"limit":100},"chart_type":"metric","title":"用户数"}`}, "finish_reason": "stop"}}})
		if encodeErr != nil {
			t.Error(encodeErr)
		}
	}))
	t.Cleanup(server.Close)
	modelStore := model.NewRegistry()
	modelStore.Replace([]model.ProviderModel{{ProviderID: 1, Config: &modelconfig.ModelConfig{Provider: modelconfig.ProviderOpenAICompatible, ModelName: "test", APIKey: "test", BaseURL: server.URL}}})
	query := NewBiQueryCase(base, source, modelStore, data.NewBiQueryRepository(store))
	dashboards := NewBiDashboardCase(base, store, data.NewBiDashboardRepository(store), data.NewBiDashboardVersionRepository(store), data.NewBiCardRepository(store), query)
	return NewBiCardCase(base, data.NewBiCardRepository(store), data.NewBiWorkbenchRepository(store), query, dashboards), dashboards
}

// biCardTestContext 构造指定租户、用户和单项目授权的受信上下文。
func biCardTestContext(tenantID, userID, projectID int64) context.Context {
	identity := &authData.UserTokenPayload{TenantId: tenantID, TenantCode: "TEST", UserId: userID, DataScope: databaseGorm.DataScopeAll}
	if projectID >= 0 {
		identity.TenantProjects = []authData.TenantProjectScope{{TenantId: tenantID, ProjectId: []int64{projectID}}}
	}
	return engine.ContextWithAuthClaims(context.Background(), identity.MakeAuthClaims())
}
