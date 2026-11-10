package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
	"github.com/liujitcn/kratos-core/biz"
	modelconfig "github.com/liujitcn/kratos-kit/ai/model"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	"github.com/liujitcn/kratos-kit/database/gorm/driver"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// TestAskBiQueryPersistsRecords 验证新版问数的成功和失败均能从记录页接口读取。
func TestAskBiQueryPersistsRecords(t *testing.T) {
	plan := `{"query":{"source_code":"test","measures":[{"code":"count","aggregation":"count"}]},"chart_type":"metric","title":"用户数量"}`
	tests := []struct {
		name        string
		modelStatus int
		output      string
		queryErr    error
		cancelQuery bool
		wantError   string
	}{
		{name: "成功", modelStatus: http.StatusOK, output: plan},
		{name: "图型建议不匹配", modelStatus: http.StatusOK, output: strings.Replace(plan, `"metric"`, `"pie"`, 1)},
		{name: "未启用模型", wantError: "AI供应商或模型选择无效"},
		{name: "模型调用失败", modelStatus: http.StatusBadRequest, wantError: "BI 问数生成失败"},
		{name: "模型拒绝回答", modelStatus: http.StatusOK, output: `{"error":"问题超出数据范围"}`, wantError: "问题超出数据范围"},
		{name: "查询计划非法", modelStatus: http.StatusOK, output: `{}`, wantError: "BI 查询维度、指标或过滤条件超出限制"},
		{name: "查询执行失败", modelStatus: http.StatusOK, output: plan, queryErr: errors.New("数据源连接失败"), wantError: "执行 BI 查询失败"},
		{name: "查询取消", modelStatus: http.StatusOK, output: plan, queryErr: context.Canceled, cancelQuery: true, wantError: "执行 BI 查询失败"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := newBiQueryRecordTestStore(t)
			ctx := biCardTestContext(1, 10, 42)
			askCtx, cancel := context.WithCancel(ctx)
			t.Cleanup(cancel)
			registry := bi.NewRegistry()
			source := biQueryRecordTestSource{queryErr: test.queryErr}
			if test.cancelQuery {
				source.cancel = cancel
			}
			err := registry.Register(source)
			if err != nil {
				t.Fatal(err)
			}
			modelStore := model.NewRegistry()
			if test.modelStatus != 0 {
				t.Setenv("SECURITY_AI_ALLOWED_CIDRS", "127.0.0.1/32")
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(test.modelStatus)
					if encodeErr := json.NewEncoder(w).Encode(map[string]any{
						"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": test.output}, "finish_reason": "stop"}},
					}); encodeErr != nil {
						t.Error(encodeErr)
					}
				}))
				t.Cleanup(server.Close)
				modelStore.Replace([]model.ProviderModel{{Config: &modelconfig.ModelConfig{Provider: modelconfig.ProviderOpenAICompatible, ModelName: "test", APIKey: "test", BaseURL: server.URL, MaxRetries: 0}}})
			}
			baseCase := &biz.BaseCase{}
			sourceCase := NewBiSourceCase(baseCase, store, data.NewBiSourceRepository(store), registry, nil)
			queryRepo := data.NewBiQueryRepository(store)
			c := NewBiQueryCase(baseCase, sourceCase, modelStore, queryRepo)
			var answer *adminv1.AskBiQueryResponse
			answer, err = c.AskBiQuery(askCtx, &adminv1.AskBiQueryRequest{Question: "统计用户数量"})
			if test.wantError == "" && err != nil || test.wantError != "" && (err == nil || !strings.Contains(err.Error(), test.wantError)) {
				t.Fatalf("问数结果错误: %v, wantError=%q", err, test.wantError)
			}
			if test.wantError == "" && answer.ChartType != "metric" {
				t.Fatal("未按结果结构修正图型建议")
			}
			recordCase := NewBiQueryRecordCase(baseCase, queryRepo, data.NewBaseUserRepository(store))
			var page *adminv1.PageBiQueryResponse
			page, err = recordCase.PageBiQuery(ctx, &adminv1.PageBiQueryRequest{PageNum: 1, PageSize: 10})
			if err != nil {
				t.Fatal(err)
			}
			if page.Total != 1 || len(page.Queries) != 1 {
				t.Fatalf("问数完成后记录页仍为空: %v", page)
			}
			var record *adminv1.BiQueryRecord
			record, err = recordCase.GetBiQuery(ctx, &adminv1.GetBiQueryRequest{Id: page.Queries[0].Id})
			if err != nil {
				t.Fatal(err)
			}
			if record.Question != "统计用户数量" || record.TenantId != 1 || record.UserId != 10 || record.ProjectId != 42 {
				t.Fatalf("问数内容或认证身份未正确保存: %v", record)
			}
			if test.wantError != "" {
				if record.Status != adminv1.BiQueryStatus_BI_QUERY_STATUS_FAILED || !strings.Contains(record.ErrorMsg, test.wantError) {
					t.Fatalf("失败原因未保存: %v", record)
				}
				return
			}
			if record.Status != adminv1.BiQueryStatus_BI_QUERY_STATUS_SUCCESS || record.RowCount != 1 || len(record.GetResult().GetColumns()) != 1 || record.GetResult().Columns[0].Name != "用户数量" || len(record.GetResult().GetRows()) != 1 || record.Result.Rows[0].Values[0] != "3" || record.Result.QueryRecordId != record.Id {
				t.Fatalf("成功结果快照未保存: %v", record)
			}
			if record.SourceCode != "test" || record.QueryJson == "" || len(record.QueryHash) != 64 {
				t.Fatalf("查询计划或数据源未保存: %v", record)
			}
		})
	}
}

// TestBiExecutionsUseQueryRecords 验证预览和仪表盘执行共用查询表，并保留上下文、失败和截断信息。
func TestBiExecutionsUseQueryRecords(t *testing.T) {
	tests := []struct {
		name      string
		dashboard bool
		truncated bool
		queryErr  error
	}{
		{name: "预览"},
		{name: "仪表盘", dashboard: true},
		{name: "结果截断", truncated: true},
		{name: "执行失败", queryErr: errors.New("数据源连接失败")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := newBiQueryRecordTestStore(t)
			ctx := biCardTestContext(1, 10, 42)
			registry := bi.NewRegistry()
			err := registry.Register(biQueryRecordTestSource{queryErr: test.queryErr, truncated: test.truncated})
			if err != nil {
				t.Fatal(err)
			}
			baseCase := &biz.BaseCase{}
			queryRepo := data.NewBiQueryRepository(store)
			sourceCase := NewBiSourceCase(baseCase, store, data.NewBiSourceRepository(store), registry, nil)
			c := NewBiQueryCase(baseCase, sourceCase, nil, queryRepo)
			spec := &adminv1.BiQuerySpec{SourceCode: "test", Measures: []*adminv1.BiMeasureSpec{{Code: "count", Aggregation: "count"}}}
			if test.dashboard {
				_, err = c.execute(ctx, bi.Scope{TenantID: 1, UserID: 10, ProjectID: 42}, fromProtoQuerySpec(spec), &dto.BiQueryRecordContext{DashboardID: 7, DashboardVersion: 2, WidgetID: "users"})
			} else {
				_, err = c.PreviewBiQuery(ctx, &adminv1.PreviewBiQueryRequest{Query: spec})
			}
			if (err != nil) != (test.queryErr != nil) {
				t.Fatalf("执行结果错误: %v", err)
			}
			recordCase := NewBiQueryRecordCase(baseCase, queryRepo, data.NewBaseUserRepository(store))
			var page *adminv1.PageBiQueryResponse
			page, err = recordCase.PageBiQuery(ctx, &adminv1.PageBiQueryRequest{PageNum: 1, PageSize: 10, SourceCode: "test"})
			if err != nil {
				t.Fatal(err)
			}
			if page.Total != 0 {
				t.Fatalf("手动执行不应进入问数历史: %v", page)
			}
			var rows []*models.BiQuery
			rows, err = queryRepo.List(ctx)
			if err != nil || len(rows) != 1 {
				t.Fatalf("执行记录缺失: %v", err)
			}
			var record *adminv1.BiQueryRecord
			record, err = recordCase.GetBiQuery(ctx, &adminv1.GetBiQueryRequest{Id: rows[0].ID})
			if err != nil {
				t.Fatal(err)
			}
			if record.Question != "" || record.SourceCode != "test" || record.ProjectId != 42 || record.QueryJson == "" || len(record.QueryHash) != 64 {
				t.Fatalf("语义查询未完整保存: %v", record)
			}
			if test.dashboard && (record.DashboardId != 7 || record.DashboardVersion != 2 || record.WidgetId != "users") {
				t.Fatalf("仪表盘上下文未保存: %v", record)
			}
			if test.queryErr != nil {
				if record.Status != adminv1.BiQueryStatus_BI_QUERY_STATUS_FAILED || !strings.Contains(record.ErrorMsg, "数据源连接失败") {
					t.Fatalf("执行失败未保存: %v", record)
				}
			} else if record.Status != adminv1.BiQueryStatus_BI_QUERY_STATUS_SUCCESS || (record.Truncated == 1) != test.truncated || len(record.GetResult().GetRows()) != 1 {
				t.Fatalf("执行结果或截断标记未保存: %v", record)
			}
		})
	}
}

// TestBiQueryRecordsRespectScope 验证统一后的查询记录和结果快照不能跨租户、项目读取。
func TestBiQueryRecordsRespectScope(t *testing.T) {
	store := newBiQueryRecordTestStore(t)
	queryRepo := data.NewBiQueryRepository(store)
	for _, record := range []*models.BiQuery{
		{ID: 1, TenantID: 1, ProjectID: 42, UserID: 10, Question: "查询用户", SourceCode: "users"},
		{ID: 2, TenantID: 2, ProjectID: 42, UserID: 10, Question: "查询用户", SourceCode: "users"},
		{ID: 3, TenantID: 1, ProjectID: 43, UserID: 10, Question: "查询用户", SourceCode: "users"},
		{ID: 4, TenantID: 1, ProjectID: 42, UserID: 10, Question: "查询日志", SourceCode: "logs"},
	} {
		err := queryRepo.Create(t.Context(), record)
		if err != nil {
			t.Fatal(err)
		}
	}
	c := NewBiQueryRecordCase(&biz.BaseCase{}, queryRepo, data.NewBaseUserRepository(store))
	ctx := biCardTestContext(1, 10, 42)
	page, err := c.PageBiQuery(ctx, &adminv1.PageBiQueryRequest{PageNum: 1, PageSize: 10, SourceCode: "users"})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Queries) != 1 || page.Queries[0].Id != 1 {
		t.Fatalf("查询记录的租户、项目或数据源过滤失效: %v", page)
	}
	for _, id := range []int64{2, 3} {
		if _, err = c.GetBiQuery(ctx, &adminv1.GetBiQueryRequest{Id: id}); err == nil {
			t.Fatalf("允许读取其他租户或项目的记录: %d", id)
		}
	}
	if _, err = c.GetBiQuery(biCardTestContext(1, 20, 42), &adminv1.GetBiQueryRequest{Id: 1}); err == nil {
		t.Fatal("其他用户读到了问数结果快照")
	}

}

// newBiQueryRecordTestStore 创建隔离数据库，通过真实仓储验证问数写入和分页读取。
func newBiQueryRecordTestStore(t *testing.T) *data.Data {
	t.Helper()
	driver.Opens["bi-query-record-test-sqlite"] = sqlite.Open
	modelList := []any{&models.BiQuery{}, &models.BiSource{}, &models.BaseUser{}}
	client, cleanup, err := databaseGorm.NewGormClient(&configv1.Data_Database{Driver: "bi-query-record-test-sqlite", Source: filepath.Join(t.TempDir(), "records.db")}, databaseGorm.WithMigrateModels(modelList...))
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
	return store
}

// biQueryRecordTestSource 提供不依赖外部业务数据库的固定查询结果。
type biQueryRecordTestSource struct {
	queryErr  error
	cancel    context.CancelFunc
	truncated bool
}

// Code 返回测试数据源编码。
func (biQueryRecordTestSource) Code() string { return "test" }

// Descriptor 返回测试数据源支持的计数指标。
func (biQueryRecordTestSource) Descriptor(context.Context, bi.Scope) (bi.SourceDescriptor, error) {
	return bi.SourceDescriptor{Code: "test", Measures: []bi.MeasureDescriptor{{Code: "count", Aggregations: []string{"count"}}}}, nil
}

// Execute 返回测试结果或指定的执行错误。
func (s biQueryRecordTestSource) Execute(context.Context, bi.Scope, bi.QuerySpec) (bi.QueryResult, error) {
	if s.cancel != nil {
		s.cancel()
	}
	return bi.QueryResult{Columns: []bi.ResultColumn{{Code: "count", Name: "用户数量", Type: bi.FieldTypeNumber}}, Rows: [][]string{{"3"}}, Truncated: s.truncated}, s.queryErr
}
