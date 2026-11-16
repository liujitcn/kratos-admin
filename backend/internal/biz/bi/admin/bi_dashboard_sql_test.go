package admin

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/go-kratos/kratos/v3/errors"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	"github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

// TestDashboardDisplayDoesNotExposeExecution 验证统一仪表盘转换不能外泄 SQL、参数或业务过滤。
func TestDashboardDisplayDoesNotExposeExecution(t *testing.T) {
	layout := dto.DashboardSchema{SchemaVersion: 4, Name: "展示", Cards: []dto.DashboardCard{{ID: "a", CardID: 1, X: 0, Y: 0, W: 6, H: 4, Name: "销售", ChartType: "metric", Query: &bi.QuerySpec{SourceCode: "private_source", Filters: []bi.FilterSpec{{Field: "private_filter", Values: []string{"secret"}}}}, Execution: &dto.CardExecution{SQL: "SELECT private_formula", Parameters: []dto.ExecutionParameter{{Kind: "scope", Field: "tenant_id"}}, Hash: "private_hash", Columns: []bi.ResultColumn{{Code: "sales", Name: "销售额", Role: "measure", Type: bi.FieldTypeNumber}}}}}}
	payload, err := json.Marshal(layout)
	if err != nil {
		t.Fatal(err)
	}
	result := toBiDashboard(&models.BiDashboard{ID: 1}, string(payload))
	for _, forbidden := range []string{"private_formula", "private_filter", "private_source", "private_hash", "parameters", "execution", "secret"} {
		if strings.Contains(result.SchemaJson, forbidden) {
			t.Fatalf("展示响应泄漏内部字段 %q: %s", forbidden, result.SchemaJson)
		}
	}
	if !strings.Contains(result.SchemaJson, "销售额") {
		t.Fatal("展示字段未保留")
	}
}

// TestSourceLimitChangePreservesSavedQuery 验证源上限变化后重新发布、展示保存及公式预览仍使用原查询规格。
func TestSourceLimitChangePreservesSavedQuery(t *testing.T) {
	cards, dashboards := newBiWorkflowTestCases(t)
	sourceCase := cards.biQueryCase.biSourceCase
	sourceCase.authorizer = modelTestAuthorizer{}
	identity := &data.UserTokenPayload{TenantId: 1, TenantCode: gorm.DefaultTenantCode, UserId: 10, DataScope: gorm.DataScopeAll}
	ctx := engine.ContextWithAuthClaims(t.Context(), identity.MakeAuthClaims())
	db := cards.GormClients[gorm.DefaultClientName].DB
	err := db.AutoMigrate(&modelTestOrder{})
	if err != nil {
		t.Fatal(err)
	}
	orders := make([]modelTestOrder, 12)
	for index := range orders {
		orders[index] = modelTestOrder{ID: int64(index + 1), TenantID: 1, Shop: fmt.Sprintf("shop%02d", index)}
	}
	if err = db.Create(&orders).Error; err != nil {
		t.Fatal(err)
	}
	model := dto.ConfiguredSourceSchema{BaseTable: "orders", ModelRevision: 1, Tables: []dto.ConfiguredSourceTable{{Key: "orders", Name: "model_test_orders", TenantColumn: "tenant_id", Fields: map[string]bi.FieldType{"shop": bi.FieldTypeString}}}, Dimensions: []dto.ConfiguredSourceDimension{{Code: "shop", Name: "门店", Type: bi.FieldTypeString, Table: "orders", Column: "shop"}}, Measures: []dto.ConfiguredSourceMeasure{{Code: "count", Name: "笔数", Type: bi.FieldTypeNumber, Expression: "COUNT(*)", Revision: 1}}}
	var payload []byte
	payload, err = json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	source := &models.BiSource{TenantID: 1, Code: "limited", Name: "上限测试", SourceName: gorm.DefaultClientName, Status: 1, Version: 1, MaxRows: 100, SchemaJSON: string(payload)}
	if err = sourceCase.sourceRepo.Create(ctx, source); err != nil {
		t.Fatal(err)
	}
	spec := bi.QuerySpec{SourceCode: source.Code, Dimensions: []string{"shop"}, Measures: []bi.MeasureSpec{{Code: "count"}}, Orders: []bi.OrderSpec{{Field: "shop", Direction: "asc"}}, Limit: 100}
	var initial bi.QueryResult
	initial, err = cards.biQueryCase.execute(ctx, bi.Scope{TenantID: 1, UserID: 10}, spec, &dto.BiQueryRecordContext{Question: "按门店统计笔数"})
	if err != nil {
		t.Fatal(err)
	}
	var card *adminv1.BiCard
	card, err = cards.SaveBiCard(ctx, &adminv1.SaveBiCardRequest{ExpectedSourceVersion: 1, Card: &adminv1.BiCardForm{Name: "门店", Query: toProtoQuerySpec(spec), QueryRecordId: initial.QueryRecordID, ChartType: "bar"}})
	if err != nil {
		t.Fatal(err)
	}
	layout := fmt.Sprintf(`{"schemaVersion":4,"name":"上限测试","cards":[{"id":"a","card_id":%d,"x":0,"y":0,"w":6,"h":4}]}`, card.Id)
	var dashboard *adminv1.BiDashboard
	dashboard, err = dashboards.SaveBiDashboard(ctx, &adminv1.SaveBiDashboardRequest{Dashboard: &adminv1.BiDashboard{Code: "limited", Name: "上限测试", SchemaJson: layout}})
	if err != nil {
		t.Fatal(err)
	}
	dashboard, err = dashboards.PublishBiDashboard(ctx, &adminv1.PublishBiDashboardRequest{Id: dashboard.Id, Version: dashboard.DraftVersion})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name      string
		maxRows   int32
		wantRows  int
		truncated bool
	}{
		{name: "降低上限", maxRows: 10, wantRows: 10, truncated: true},
		{name: "提高上限", maxRows: 20, wantRows: 12},
	} {
		t.Run(test.name, func(t *testing.T) {
			var updated *adminv1.BiSource
			updated, err = sourceCase.UpdateBiSource(ctx, &adminv1.UpdateBiSourceRequest{Source: &adminv1.BiSourceForm{Id: source.ID, Code: source.Code, Name: source.Name, SourceName: source.SourceName, MaxRows: test.maxRows}})
			if err != nil {
				t.Fatal(err)
			}
			if _, err = dashboards.GetBiDashboardData(ctx, &adminv1.GetBiDashboardDataRequest{Id: dashboard.Id}); errors.Code(err) != 409 {
				t.Fatalf("源上限变化后旧发布未失效: %v", err)
			}
			t.Run("重新发布取数", func(t *testing.T) {
				dashboard, err = dashboards.SaveBiDashboard(ctx, &adminv1.SaveBiDashboardRequest{Dashboard: dashboard})
				if err != nil {
					t.Fatal(err)
				}
				dashboard, err = dashboards.PublishBiDashboard(ctx, &adminv1.PublishBiDashboardRequest{Id: dashboard.Id, Version: dashboard.DraftVersion})
				if err != nil {
					t.Fatal(err)
				}
				if dashboard.RequiresRepublish {
					t.Fatal("新发布仍标记为需要重新发布")
				}
				var result *adminv1.BiDashboardDataResponse
				result, err = dashboards.GetBiDashboardData(ctx, &adminv1.GetBiDashboardDataRequest{Id: dashboard.Id})
				if err != nil {
					t.Fatal("重新发布后仍无法取数", err)
				}
				if len(result.Cards) != 1 || len(result.Cards[0].Result.Rows) != test.wantRows || result.Cards[0].Result.Truncated != test.truncated {
					t.Fatal("发布结果未遵守新的有效上限")
				}
			})
			t.Run("保存展示配置", func(t *testing.T) {
				card, err = cards.GetBiCard(ctx, &adminv1.GetBiCardRequest{Id: card.Id})
				if err != nil {
					t.Fatal(err)
				}
				if card.Query.Limit != 100 || card.QueryRecordId != initial.QueryRecordID {
					t.Fatal("源上限变化改写了卡片原查询或来源")
				}
				var saved *adminv1.BiCard
				saved, err = cards.SaveBiCard(ctx, &adminv1.SaveBiCardRequest{Id: card.Id, ExpectedCardVersion: card.Version, ExpectedSourceVersion: updated.Version, Card: &adminv1.BiCardForm{Name: "新样式", Query: card.Query, QueryRecordId: card.QueryRecordId, ChartType: "line"}})
				if err != nil {
					t.Fatal("源上限变化后不能保存展示配置", err)
				}
				card = saved
			})
			t.Run("公式预览", func(t *testing.T) {
				var preview *adminv1.PreviewBiCardSqlResponse
				preview, err = cards.PreviewBiCardSql(ctx, &adminv1.PreviewBiCardSqlRequest{Id: card.Id, ExpectedCardVersion: card.Version, ExpectedSourceVersion: updated.Version, QueryRecordId: initial.QueryRecordID, Query: toProtoQuerySpec(spec), MeasureChanges: []*adminv1.BiMetricChange{{MetricCode: "count", Name: "笔数", Expression: "COUNT(*)*2", ExpectedRevision: 1}}})
				if err != nil {
					t.Fatal("源上限变化后不能预览公式", err)
				}
				if len(preview.Result.Rows) != test.wantRows || preview.Result.Truncated != test.truncated || preview.Result.Rows[0].Values[1] != "2" {
					t.Fatal("预览结果未遵守新的有效上限或公式")
				}
			})
		})
	}
}
