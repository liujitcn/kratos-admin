package admin

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	kratosErrors "github.com/go-kratos/kratos/v3/errors"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
	authnEngine "github.com/liujitcn/kratos-kit/auth/authn/engine"
	"github.com/liujitcn/kratos-kit/auth/authz/engine"
	"github.com/liujitcn/kratos-kit/auth/data"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/gorm"
)

// modelTestAuthorizer 仅替代外部权限决策，其余同步链路使用真实数据库和仓储。
type modelTestAuthorizer struct{ engine.Engine }

// IsAuthorized 授予测试账号公共模型管理权限。
func (modelTestAuthorizer) IsAuthorized(context.Context, engine.Subject, engine.Action, engine.Resource, engine.Project) (bool, error) {
	return true, nil
}

// modelTestOrder 提供可验证租户隔离、分组及退款公式的业务事实。
type modelTestOrder struct {
	ID       int64 `gorm:"primaryKey"`
	TenantID int64
	Shop     string
	Amount   int64
	Refund   int64
}

// TableName 固定事实表名称，避免测试连接的命名策略影响 SQL 断言。
func (*modelTestOrder) TableName() string { return "model_test_orders" }

// TestCardSQLPublicMetricWorkflow 验证多卡同步、来源保持、预览一次消费和当前及历史发布失效。
func TestCardSQLPublicMetricWorkflow(t *testing.T) {
	c, dashboards := newBiWorkflowTestCases(t)
	sourceCase := c.biQueryCase.biSourceCase
	sourceCase.authorizer = modelTestAuthorizer{}
	identity := &data.UserTokenPayload{TenantId: 1, TenantCode: databaseGorm.DefaultTenantCode, UserId: 10, DataScope: databaseGorm.DataScopeAll}
	ctx := authnEngine.ContextWithAuthClaims(t.Context(), identity.MakeAuthClaims())
	db := c.GormClients[databaseGorm.DefaultClientName].DB
	err := db.AutoMigrate(&modelTestOrder{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&[]modelTestOrder{{ID: 1, TenantID: 1, Shop: "A", Amount: 100, Refund: 20}, {ID: 2, TenantID: 2, Shop: "B", Amount: 999, Refund: 0}}).Error; err != nil {
		t.Fatal(err)
	}
	model := dto.ConfiguredSourceSchema{BaseTable: "orders", ModelRevision: 1, Tables: []dto.ConfiguredSourceTable{{Key: "orders", Name: "model_test_orders", TenantColumn: "tenant_id", Fields: map[string]bi.FieldType{"amount": bi.FieldTypeNumber, "refund": bi.FieldTypeNumber, "shop": bi.FieldTypeString}}}, Dimensions: []dto.ConfiguredSourceDimension{{Code: "shop", Name: "门店", Type: bi.FieldTypeString, Table: "orders", Column: "shop", Filterable: true}}, Measures: []dto.ConfiguredSourceMeasure{{Code: "sales", Name: "销售额", Type: bi.FieldTypeNumber, Expression: "SUM(amount)", Unit: "元", Revision: 5}}}
	var payload []byte
	payload, err = json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	source := &models.BiSource{TenantID: 1, Code: "sales", Name: "销售", SourceName: databaseGorm.DefaultClientName, Status: 1, Version: 5, MaxRows: 100, SchemaJSON: string(payload)}
	if err = sourceCase.sourceRepo.Create(ctx, source); err != nil {
		t.Fatal(err)
	}
	scope := bi.Scope{TenantID: 1, UserID: 10, Terminal: "admin"}
	var cards []*adminv1.BiCard
	for _, dimensions := range [][]string{nil, {"shop"}} {
		spec := bi.QuerySpec{SourceCode: "sales", Dimensions: dimensions, Measures: []bi.MeasureSpec{{Code: "sales"}}, Limit: 100}
		var result bi.QueryResult
		result, err = c.biQueryCase.execute(ctx, scope, spec, &dto.BiQueryRecordContext{Question: "销售额"})
		if err != nil {
			t.Fatal(err)
		}
		chartType := "metric"
		if len(dimensions) > 0 {
			chartType = "bar"
		}
		var saved *adminv1.BiCard
		saved, err = c.SaveBiCard(ctx, &adminv1.SaveBiCardRequest{ExpectedSourceVersion: 5, Card: &adminv1.BiCardForm{Name: "销售", Query: toProtoQuerySpec(spec), QueryRecordId: result.QueryRecordID, ChartType: chartType}})
		if err != nil {
			t.Fatal(err)
		}
		cards = append(cards, saved)
	}
	layout := `{"schemaVersion":4,"name":"销售","cards":[{"id":"a","card_id":1,"x":0,"y":0,"w":6,"h":4},{"id":"b","card_id":2,"x":6,"y":0,"w":6,"h":4}]}`
	var dashboard *adminv1.BiDashboard
	dashboard, err = dashboards.SaveBiDashboard(ctx, &adminv1.SaveBiDashboardRequest{Dashboard: &adminv1.BiDashboard{Code: "sales", Name: "销售", SchemaJson: layout}})
	if err != nil {
		t.Fatal(err)
	}
	dashboard, err = dashboards.PublishBiDashboard(ctx, &adminv1.PublishBiDashboardRequest{Id: dashboard.Id, Version: dashboard.DraftVersion})
	if err != nil {
		t.Fatal(err)
	}
	// 浏览器把空集合回填为 []，不应使相同查询被拒绝。
	cards[0].Query.Dimensions = []string{}
	cards[0].Query.Filters = []*adminv1.BiFilterSpec{}
	cards[0].Query.Orders = []*adminv1.BiOrderSpec{}
	changes := []*adminv1.BiMetricChange{{MetricCode: "sales", Name: "销售额", Expression: "SUM(amount-refund)", Unit: "元", ExpectedRevision: 5}}
	var preview *adminv1.PreviewBiCardSqlResponse
	preview, err = c.PreviewBiCardSql(ctx, &adminv1.PreviewBiCardSqlRequest{Id: cards[0].Id, ExpectedCardVersion: 1, ExpectedSourceVersion: 5, QueryRecordId: cards[0].QueryRecordId, Query: cards[0].Query, MeasureChanges: changes})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Result.Rows[0].Values[0] != "80" || preview.Impact.AffectedCards != 2 || preview.Impact.AffectedDashboards != 1 {
		t.Fatalf("预览结果或影响分析错误: %v", preview)
	}
	var before *models.BiSource
	before, err = sourceCase.sourceRepo.FindByID(ctx, source.ID)
	if err != nil || before.Version != 5 {
		t.Fatal("预览提前更新了公共定义", err)
	}
	request := &adminv1.SaveBiCardRequest{Id: cards[0].Id, ExpectedCardVersion: 1, ExpectedSourceVersion: 5, PreviewQueryRecordId: preview.PreviewQueryRecordId, MeasureChanges: changes, Card: &adminv1.BiCardForm{Name: "净额", Query: cards[0].Query, QueryRecordId: cards[0].QueryRecordId, ChartType: "metric"}}
	var saved *adminv1.BiCard
	saved, err = c.SaveBiCard(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if saved.QueryRecordId != cards[0].QueryRecordId || saved.ValidationQueryRecordId != preview.PreviewQueryRecordId || saved.Version != 2 {
		t.Fatal("人工验证破坏了来源或卡片版本")
	}
	if _, err = c.SaveBiCard(ctx, request); kratosErrors.Code(err) != 409 || !strings.Contains(err.Error(), "card_id") {
		t.Fatalf("重复预览未原子拒绝并定位已保存卡片: %v", err)
	}
	var other *models.BiCard
	other, err = c.cardRepo.FindByID(ctx, cards[1].Id)
	if err != nil || other.Version != 2 || other.QueryRecordID != cards[1].QueryRecordId || other.ValidationQueryRecordID != 0 || !strings.Contains(other.QueryJSON, "shop") || !strings.Contains(other.ExecutionJSON, "refund") {
		t.Fatal("关联卡片未同步或被改写业务配置", err)
	}
	for _, version := range []int64{0, dashboard.PublishedVersion} {
		if _, err = dashboards.GetBiDashboardData(ctx, &adminv1.GetBiDashboardDataRequest{Id: dashboard.Id, Version: version}); kratosErrors.Code(err) != 409 {
			t.Fatalf("旧发布口径仍可执行: %v", err)
		}
	}
	// 级联同步后的卡片仍可保存展示配置，无需借用其他卡片的人工预览。
	var details *adminv1.BiCard
	details, err = c.GetBiCard(ctx, &adminv1.GetBiCardRequest{Id: cards[1].Id})
	if err != nil {
		t.Fatal(err)
	}
	if details.ExecutionKind != "configured_sql" || len(details.Columns) != 2 {
		t.Fatal("卡片缺少安全执行元数据")
	}
	_, err = c.SaveBiCard(ctx, &adminv1.SaveBiCardRequest{Id: details.Id, ExpectedCardVersion: details.Version, ExpectedSourceVersion: 6, Card: &adminv1.BiCardForm{Name: "新样式", Query: details.Query, QueryRecordId: details.QueryRecordId, ChartType: "bar"}})
	if err != nil {
		t.Fatal("级联后展示保存失败", err)
	}
	dashboard, err = dashboards.SaveBiDashboard(ctx, &adminv1.SaveBiDashboardRequest{Dashboard: dashboard})
	if err != nil {
		t.Fatal(err)
	}
	dashboard, err = dashboards.PublishBiDashboard(ctx, &adminv1.PublishBiDashboardRequest{Id: dashboard.Id, Version: dashboard.DraftVersion})
	if err != nil {
		t.Fatal("同步卡片无法重新发布", err)
	}
	// 在源和关联卡片已写入后让当前卡片保存失败，验证整个元数据事务回滚。
	changes[0].Expression, changes[0].ExpectedRevision = "SUM(amount-refund-1)", 6
	preview, err = c.PreviewBiCardSql(ctx, &adminv1.PreviewBiCardSqlRequest{Id: saved.Id, ExpectedCardVersion: 2, ExpectedSourceVersion: 6, QueryRecordId: saved.QueryRecordId, Query: saved.Query, MeasureChanges: changes})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Callback().Update().Before("gorm:update").Register("bi_test_failure", func(tx *gorm.DB) {
		if value, ok := tx.Statement.Dest.(*models.BiCard); ok && value.ID == saved.Id {
			tx.AddError(errors.New("模拟卡片写入失败"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	request.ExpectedCardVersion, request.ExpectedSourceVersion, request.PreviewQueryRecordId = 2, 6, preview.PreviewQueryRecordId
	if _, err = c.SaveBiCard(ctx, request); err == nil {
		t.Fatal("模拟写入失败未生效")
	}
	if err = db.Callback().Update().Remove("bi_test_failure"); err != nil {
		t.Fatal(err)
	}
	before, err = sourceCase.sourceRepo.FindByID(ctx, source.ID)
	if err != nil || before.Version != 6 {
		t.Fatal("失败事务保留了公共定义写入", err)
	}
	other, err = c.cardRepo.FindByID(ctx, cards[1].Id)
	if err != nil || other.Version != 3 {
		t.Fatal("失败事务保留了关联卡片写入", err)
	}
}

// TestSourceModelReadPermission 防止普通查询权限读出物理模型及指标公式。
func TestSourceModelReadPermission(t *testing.T) {
	c, _ := newBiWorkflowTestCases(t)
	sourceCase := c.biQueryCase.biSourceCase
	ctx := biCardTestContext(1, 10, -1)
	source := &models.BiSource{TenantID: 1, Code: "private_model", Name: "私有模型", SchemaJSON: `{"baseTable":"secret_table","measures":[{"expression":"SUM(secret)"}]}`, Version: 1}
	err := sourceCase.sourceRepo.Create(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	var result *adminv1.BiSource
	result, err = sourceCase.GetBiSource(ctx, &adminv1.GetBiSourceRequest{Id: source.ID})
	if err != nil || result.SchemaJson != "" {
		t.Fatalf("没有模型管理权仍读取到物理定义: %v %v", result, err)
	}
	if _, err = sourceCase.ListBiSourceTables(ctx, &adminv1.ListBiSourceTablesRequest{Id: source.ID}); kratosErrors.Code(err) != 403 {
		t.Fatalf("发现接口未在业务层限制模型权: %v", err)
	}
}

// TestSourceConnectionChangeInvalidatesModel 验证连接变更销毁旧模型的执行资格，密码不改变结构身份。
func TestSourceConnectionChangeInvalidatesModel(t *testing.T) {
	c, _ := newBiWorkflowTestCases(t)
	sourceCase := c.biQueryCase.biSourceCase
	sourceCase.authorizer = modelTestAuthorizer{}
	identity := &data.UserTokenPayload{TenantId: 1, TenantCode: databaseGorm.DefaultTenantCode, UserId: 10, DataScope: databaseGorm.DataScopeAll}
	ctx := authnEngine.ContextWithAuthClaims(t.Context(), identity.MakeAuthClaims())
	source := &models.BiSource{TenantID: 1, Code: "connection", Name: "连接", SourceName: databaseGorm.DefaultClientName, Version: 2, MaxRows: 100, SchemaJSON: `{"model_revision":8,"baseTable":"orders","tables":[{"key":"orders","name":"orders"}],"measures":[{"code":"sales","expression":"COUNT(*)"}]}`}
	err := sourceCase.sourceRepo.Create(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	sourceCase.GormClients["other"] = sourceCase.GormClients[databaseGorm.DefaultClientName]
	_, err = sourceCase.UpdateBiSource(ctx, &adminv1.UpdateBiSourceRequest{Source: &adminv1.BiSourceForm{Id: source.ID, Code: source.Code, Name: source.Name, SourceName: "other", MaxRows: 100}})
	if err != nil {
		t.Fatal(err)
	}
	var updated *models.BiSource
	updated, err = sourceCase.sourceRepo.FindByID(ctx, source.ID)
	if err != nil {
		t.Fatal(err)
	}
	var model dto.ConfiguredSourceSchema
	if err = json.Unmarshal([]byte(updated.SchemaJSON), &model); err != nil {
		t.Fatal(err)
	}
	if model.ModelRevision <= 8 || hasConfiguredSourceModel(updated.SchemaJSON) {
		t.Fatal("连接变更后旧模型仍可用")
	}
	if _, err = sourceCase.SetBiSourceStatus(ctx, &adminv1.SetBiSourceStatusRequest{Id: source.ID, Status: 1}); err == nil {
		t.Fatal("未重新发现模型就重新启用连接")
	}
}

// TestConfiguredResultType 验证真实驱动结果不能把文本伪装为数值指标，并保留空值。
func TestConfiguredResultType(t *testing.T) {
	c, _ := newBiWorkflowTestCases(t)
	db, err := c.GormClients[databaseGorm.DefaultClientName].DB.DB()
	if err != nil {
		t.Fatal(err)
	}
	execution := &dto.CardExecution{SQL: "SELECT 'wrong' AS amount", Columns: []bi.ResultColumn{{Code: "amount", Type: bi.FieldTypeNumber, Role: "measure"}}}
	if _, err = executeConfiguredDefinition(t.Context(), &configuredDatabase{db: db}, execution, nil, 100); err == nil {
		t.Fatal("文本结果被标记为数值指标")
	}
	execution.SQL = "SELECT NULL AS amount"
	var result bi.QueryResult
	result, err = executeConfiguredDefinition(t.Context(), &configuredDatabase{db: db}, execution, nil, 100)
	if err != nil || result.Rows[0][0] != "" {
		t.Fatal("NULL 未保留为空值", err)
	}
}
