package admin

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/query"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
	modelconfig "github.com/liujitcn/kratos-kit/ai/model"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/database/gorm"
	"github.com/liujitcn/kratos-kit/database/gorm/driver"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
)

// TestBiGroupingWorkflow 用真实数据库核对保存模型、问数、调整、历史、建卡、刷新与发布。
func TestBiGroupingWorkflow(t *testing.T) {
	for _, dialect := range []string{"mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			dsn := os.Getenv("BI_GROUPING_TEST_" + strings.ToUpper(dialect) + "_DSN")
			if dsn == "" {
				t.Skip("设置 BI_GROUPING_TEST_MYSQL_DSN 或 BI_GROUPING_TEST_POSTGRES_DSN 指向空的测试库")
			}
			if driver.Opens[dialect] == nil {
				if dialect == "mysql" {
					driver.Opens[dialect] = mysql.Open
				} else {
					driver.Opens[dialect] = postgres.Open
				}
				t.Cleanup(func() { delete(driver.Opens, dialect) })
			}
			client, cleanup, err := gorm.NewGormClient(&configv1.Data_Database{Driver: dialect, Source: dsn}, gorm.WithName("grouping_test"))
			if err != nil {
				t.Fatal("无法打开 BI 测试库")
			}
			t.Cleanup(cleanup)
			var db *sql.DB
			db, err = client.DB.DB()
			if err != nil {
				t.Fatal(err)
			}
			// 强制使用同一连接，执行后核对宿主连接没有残留报表时区。
			db.SetMaxOpenConns(1)
			var schema, originalZone string
			schema, originalZone = seedBiGroupingTestData(t, db, dialect)
			cards, dashboards := newBiWorkflowTestCases(t)
			cards.GormClients["grouping_test"] = client
			queries := cards.biQueryCase
			sources := queries.biSourceCase
			sources.authorizer = modelTestAuthorizer{}
			identity := &authData.UserTokenPayload{TenantId: 1, TenantCode: gorm.DefaultTenantCode, UserId: 10, DataScope: gorm.DataScopeAll}
			ctx := engine.ContextWithAuthClaims(t.Context(), identity.MakeAuthClaims())
			source := &models.BiSource{TenantID: 1, Code: "grouping_test", Name: "测试用户", SourceName: "grouping_test", Driver: dialect, SchemaName: schema, Status: 1, Version: 1, MaxRows: 100, SchemaJSON: "{}"}
			if err = sources.sourceRepo.Create(ctx, source); err != nil {
				t.Fatal(err)
			}
			users, departments := sourceTableKey(schema, "bi_test_users"), sourceTableKey(schema, "bi_test_departments")
			var saved *adminv1.BiSource
			saved, err = sources.SaveBiSourceModel(ctx, &adminv1.SaveBiSourceModelRequest{Id: source.ID, ExpectedSourceVersion: 1, Model: &adminv1.BiSourceModel{
				BaseTable: users, ReportTimezone: "Asia/Shanghai",
				Tables:    []*adminv1.BiSourceModelTable{{Key: users, TenantColumn: "tenant_id"}, {Key: departments, TenantColumn: "tenant_id"}},
				Relations: []*adminv1.BiSourceRelation{{LeftTable: users, LeftColumn: "dept_id", RightTable: departments, RightColumn: "id", JoinType: "left"}},
				Dimensions: []*adminv1.BiSourceDimension{
					{Code: "created", Name: "创建时间", Type: "date", Filterable: true, Table: users, Column: "created_at"},
					{Code: "dept", Name: "部门", Type: "string", Filterable: true, Table: departments, Column: "name", Hierarchy: &adminv1.BiSourceHierarchy{ParentColumn: "parent_id", RootId: "0"}},
				},
				Measures: []*adminv1.BiSourceMeasure{{Code: "users", Name: "新增用户数", Type: "number", Expression: "COUNT(id)"}},
			}})
			if err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				var body json.RawMessage
				decodeErr := json.NewDecoder(req.Body).Decode(&body)
				if decodeErr != nil || !strings.Contains(string(body), "timeGrains") || !strings.Contains(string(body), "hierarchyLevels") {
					t.Error("问数未收到日期粒度和部门层级目录", decodeErr)
				}
				w.Header().Set("Content-Type", "application/json")
				const answer = `{"query":{"source_code":"grouping_test","dimensions":["created","dept"],"grouping_options":[{"field":"created","time_grain":"month"},{"field":"dept","hierarchy_level":2}],"measures":[{"code":"users"}],"time_range":{"field":"created","start":"2026-01-01","end":"2026-04-01","end_exclusive":true},"orders":[{"field":"created","direction":"asc"},{"field":"dept","direction":"asc"}]},"chart_type":"line","title":"各二级部门每月新增用户数"}`
				encodeErr := json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": answer}, "finish_reason": "stop"}}})
				if encodeErr != nil {
					t.Error(encodeErr)
				}
			}))
			t.Cleanup(server.Close)
			queries.modelStore = model.NewRegistry()
			queries.modelStore.Replace([]model.ProviderModel{{ProviderID: 1, Config: &modelconfig.ModelConfig{Provider: modelconfig.ProviderOpenAICompatible, ModelName: "test", APIKey: "test", BaseURL: server.URL}}})
			var asked *adminv1.AskBiQueryResponse
			asked, err = queries.AskBiQuery(ctx, &adminv1.AskBiQueryRequest{Question: "2026年一季度各二级部门每月新增多少用户", ProviderId: 1, ModelName: "test"})
			if err != nil {
				t.Fatal(err)
			}
			assertBiGroupingRows(t, asked.Result, map[string]string{
				"2026-01-01|研发 / 平台 (#12)": "2", "2026-01-01|研发 / 平台 (#14)": "1", "2026-01-01|研发 / 平台 (#17)": "1",
				"2026-01-01|研发 / 直属 (#11)": "1", "2026-01-01|未归类": "2", "2026-02-01|研发 / 平台 (#14)": "1", "2026-02-01|未归类": "1",
			})
			refine := &adminv1.RefineBiQueryRequest{
				QueryRecordId: asked.Result.QueryRecordId, Dimensions: asked.Query.Dimensions,
				TimeRange:       &adminv1.BiTimeRangeSpec{Field: "created", Start: "2026-02-01", End: "2026-03-01", EndExclusive: true},
				GroupingOptions: []*adminv1.BiGroupingOption{{Field: "created", TimeGrain: "month"}, {Field: "dept", HierarchyLevel: 1}},
			}
			identity.UserId = 11
			if _, err = queries.RefineBiQuery(engine.ContextWithAuthClaims(t.Context(), identity.MakeAuthClaims()), refine); err == nil {
				t.Fatal("其他账号可以调整本人历史查询")
			}
			var refined *adminv1.AskBiQueryResponse
			refined, err = queries.RefineBiQuery(ctx, refine)
			if err != nil {
				t.Fatal(err)
			}
			if asked.Result.QueryRecordId == refined.Result.QueryRecordId {
				t.Fatal("调整覆盖了原始问数记录")
			}
			want := map[string]string{"2026-02-01|研发 (#11)": "1", "2026-02-01|未归类": "1"}
			assertBiGroupingRows(t, refined.Result, want)
			records := NewBiQueryRecordCase(cards.BaseCase, queries.biQueryRepo, cards.cardRepo, data.NewBaseUserRepository(sources.tx.(*data.Data)))
			var history *adminv1.BiQueryRecord
			history, err = records.GetBiQuery(ctx, &adminv1.GetBiQueryRequest{Id: refined.Result.QueryRecordId})
			if err != nil || !biQuerySpecMatches(history.GetQueryJson(), fromProtoQuerySpec(refined.Query)) {
				t.Fatal("历史未保存调整后的查询计划", err)
			}
			assertBiGroupingRows(t, history.GetResult(), want)
			var card *adminv1.BiCard
			card, err = cards.SaveBiCard(ctx, &adminv1.SaveBiCardRequest{ExpectedSourceVersion: saved.Version, Card: &adminv1.BiCardForm{Name: "一级部门每月新增", Query: refined.Query, QueryRecordId: history.GetId(), ChartType: "line"}})
			if err != nil {
				t.Fatal(err)
			}
			var result *adminv1.BiQueryResult
			result, err = cards.GetBiCardData(ctx, &adminv1.GetBiCardDataRequest{Id: card.Id})
			if err != nil {
				t.Fatal(err)
			}
			assertBiGroupingRows(t, result, want)
			var dashboard *adminv1.BiDashboard
			layout := fmt.Sprintf(`{"schemaVersion":4,"name":"测试仪表盘","cards":[{"id":"monthly","card_id":%d,"x":0,"y":0,"w":6,"h":4}]}`, card.Id)
			dashboard, err = dashboards.SaveBiDashboard(ctx, &adminv1.SaveBiDashboardRequest{Dashboard: &adminv1.BiDashboard{Code: "grouping_test", Name: "测试仪表盘", SchemaJson: layout}})
			if err != nil {
				t.Fatal(err)
			}
			dashboard, err = dashboards.PublishBiDashboard(ctx, &adminv1.PublishBiDashboardRequest{Id: dashboard.Id, Version: dashboard.DraftVersion})
			if err != nil {
				t.Fatal(err)
			}
			if _, err = db.ExecContext(ctx, "INSERT INTO bi_test_users VALUES (13,1,13,'2026-02-20 10:00:00')"); err != nil { //nolint:forbidigo // 固定测试数据。
				t.Fatal(err)
			}
			var published *adminv1.BiDashboardDataResponse
			published, err = dashboards.GetBiDashboardData(ctx, &adminv1.GetBiDashboardDataRequest{Id: dashboard.Id})
			if err != nil || len(published.GetCards()) != 1 {
				t.Fatal("仪表盘没有读出已发布的卡片", err)
			}
			want["2026-02-01|研发 (#11)"] = "2"
			assertBiGroupingRows(t, published.Cards[0].Result, want)
			t.Run("回拨边界", func(t *testing.T) {
				assertBiTimestampFallback(t, ctx, db, dialect, schema)
			})
			if zone := readBiGroupingTestZone(t, db, dialect); zone != originalZone {
				t.Fatalf("宿主连接时区从 %q 变为 %q", originalZone, zone)
			}
		})
	}
}

// seedBiGroupingTestData 用 UTC 构造月界、小数秒、同名部门、直属、孤儿和跨租户样本。
func seedBiGroupingTestData(t *testing.T, db *sql.DB, driver string) (string, string) {
	t.Helper()
	ctx := t.Context()
	schemaSQL, setZone, timeType := "SELECT DATABASE()", "SET SESSION time_zone = '+00:00'", "TIMESTAMP(6)"
	if driver == "postgres" {
		schemaSQL, setZone, timeType = "SELECT current_schema()", "SET TIME ZONE 'UTC'", "TIMESTAMPTZ(6)"
	}
	var schema string
	err := db.QueryRowContext(ctx, schemaSQL).Scan(&schema) //nolint:forbidigo // 仅查询测试数据库的 schema。
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		setZone,
		"CREATE TABLE bi_test_departments (id BIGINT PRIMARY KEY, tenant_id BIGINT NOT NULL, parent_id BIGINT, name VARCHAR(60) NOT NULL)",
		"CREATE TABLE bi_test_users (id BIGINT PRIMARY KEY, tenant_id BIGINT NOT NULL, dept_id BIGINT, created_at " + timeType + " NOT NULL)",
		"INSERT INTO bi_test_departments VALUES (11,1,0,'研发'),(12,1,11,'平台'),(13,1,12,'客户端'),(14,1,11,'平台'),(15,1,90,'孤儿'),(16,1,0,'研发'),(17,1,16,'平台'),(21,2,0,'其他租户'),(22,2,21,'其他租户二级')",
		`INSERT INTO bi_test_users VALUES (1,1,13,'2025-12-31 16:00:00'),(2,1,13,'2026-01-31 15:59:59.999999'),(3,1,14,'2026-01-31 16:00:00'),
		(4,1,11,'2026-01-10 00:00:00'),(5,1,15,'2026-01-10 00:00:00'),(6,1,NULL,'2026-01-10 00:00:00'),(7,1,22,'2026-02-10 00:00:00'),
		(8,2,22,'2026-01-10 00:00:00'),(9,1,13,'2025-12-31 15:59:59.999999'),(10,1,13,'2026-03-31 16:00:00'),(11,1,14,'2026-01-10 00:00:00'),(12,1,17,'2026-01-10 00:00:00'),
		(14,1,13,'2026-11-01 05:30:00.123456'),(15,1,13,'2026-11-01 06:30:00.123456'),(16,1,13,'2026-11-01 05:30:00.123455')`,
	} {
		if _, err = db.ExecContext(ctx, statement); err != nil { //nolint:forbidigo // 固定测试建表与数据。
			t.Fatal(err)
		}
		if table, ok := strings.CutPrefix(statement, "CREATE TABLE "); ok {
			table, _, _ = strings.Cut(table, " ")
			t.Cleanup(func() {
				if _, cleanupErr := db.ExecContext(context.WithoutCancel(ctx), "DROP TABLE "+table); cleanupErr != nil { //nolint:forbidigo // 只清理本次成功创建的测试表。
					t.Error(cleanupErr)
				}
			})
		}
	}
	return schema, readBiGroupingTestZone(t, db, driver)
}

// assertBiTimestampFallback 核对两个同名时刻分别落在相邻的实时区间内。
func assertBiTimestampFallback(t *testing.T, ctx context.Context, db *sql.DB, driver, schema string) {
	t.Helper()
	const table = "bi_test_users"
	physicalType := "timestamp"
	if driver == "postgres" {
		physicalType = "timestamp with time zone"
	}
	model := dto.ConfiguredSourceSchema{BaseTable: table, ReportTimezone: "America/New_York",
		Tables:     []dto.ConfiguredSourceTable{{Key: table, SchemaName: schema, Name: table, Fields: map[string]bi.FieldType{"id": bi.FieldTypeNumber}}},
		Dimensions: []dto.ConfiguredSourceDimension{{Code: "created", Table: table, Column: "created_at", Type: bi.FieldTypeDate, Filterable: true, DatabaseType: physicalType}},
		Measures:   []dto.ConfiguredSourceMeasure{{Code: "ids", Type: bi.FieldTypeNumber, Expression: "SUM(id)", Revision: 1}},
	}
	spec := bi.QuerySpec{SourceCode: "grouping_test", Dimensions: []string{"created"}, Measures: []bi.MeasureSpec{{Code: "ids"}},
		GroupingOptions: []bi.GroupingOption{{Field: "created", TimeGrain: "day"}},
		TimeRange:       &bi.TimeRangeSpec{Field: "created", Start: "now-1h", End: "now", EndExclusive: true},
	}
	execution, err := query.Compile(driver, 100, model, spec)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		hour int
		id   string
	}{{6, "14"}, {7, "15"}} {
		now := time.Date(2026, time.November, 1, item.hour, 30, 0, 123456000, time.UTC)
		var args []any
		args, err = bindExecutionParameters(execution, spec, bi.Scope{}, 0, now)
		if err != nil {
			t.Fatal(err)
		}
		var result bi.QueryResult
		result, err = executeConfiguredDefinition(ctx, &configuredDatabase{db: db, driver: driver}, execution, args, execution.Limit)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Rows) != 1 || !slices.Equal(result.Rows[0], []string{"2026-11-01", item.id}) {
			t.Fatalf("截止 %s 应只命中记录 %s，得到 %+v", now, item.id, result.Rows)
		}
	}
}

// readBiGroupingTestZone 读取测试连接的会话时区。
func readBiGroupingTestZone(t *testing.T, db *sql.DB, driver string) string {
	t.Helper()
	statement := "SELECT @@session.time_zone"
	if driver == "postgres" {
		statement = "SHOW TimeZone"
	}
	var zone string
	err := db.QueryRowContext(t.Context(), statement).Scan(&zone) //nolint:forbidigo // 只读固定的测试会话属性。
	if err != nil {
		t.Fatal(err)
	}
	return zone
}

// assertBiGroupingRows 校验每一个分组的键和指标，并拒绝重复或截断。
func assertBiGroupingRows(t *testing.T, result *adminv1.BiQueryResult, want map[string]string) {
	t.Helper()
	got := make(map[string]string, len(result.GetRows()))
	for _, row := range result.GetRows() {
		if len(row.Values) != 3 {
			t.Fatalf("结果不是月份、部门和新增用户数: %v", row)
		}
		got[row.Values[0]+"|"+row.Values[1]] = row.Values[2]
	}
	if result.GetTruncated() || len(got) != len(result.GetRows()) || !maps.Equal(got, want) {
		t.Fatalf("分组对账失败：得到 %v，期望 %v", got, want)
	}
}
