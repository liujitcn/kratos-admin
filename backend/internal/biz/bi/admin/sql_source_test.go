package admin

import (
	"database/sql"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
)

// TestDiscoverSourceTablesComments 验证注释、无注释和视图元数据均保留原始标识及唯一键。
func TestDiscoverSourceTablesComments(t *testing.T) {
	for _, driver := range []string{"mysql", "tidb", "doris"} {
		t.Run(driver, func(t *testing.T) {
			db, err := sql.Open("sqlite3", ":memory:")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err = db.Close(); err != nil {
					t.Error(err)
				}
			})
			db.SetMaxOpenConns(1)
			for _, statement := range []string{
				"ATTACH DATABASE ':memory:' AS information_schema",
				"CREATE TABLE information_schema.tables (table_schema TEXT, table_name TEXT, table_type TEXT, table_comment TEXT)",
				"CREATE TABLE information_schema.columns (table_schema TEXT, table_name TEXT, column_name TEXT, data_type TEXT, is_nullable TEXT, column_comment TEXT, ordinal_position INTEGER)",
				"CREATE TABLE information_schema.table_constraints (table_schema TEXT, constraint_schema TEXT, table_name TEXT, constraint_name TEXT, constraint_type TEXT)",
				"CREATE TABLE information_schema.key_column_usage (constraint_schema TEXT, table_name TEXT, constraint_name TEXT, column_name TEXT, ordinal_position INTEGER)",
				"INSERT INTO information_schema.tables VALUES ('app', 'base_user', 'BASE TABLE', '用户'), ('app', 'v_user', 'VIEW', NULL)",
				"INSERT INTO information_schema.columns VALUES ('app', 'base_user', 'id', 'bigint', 'NO', NULL, 1), ('app', 'base_user', 'dept_id', 'bigint', 'YES', '所属部门', 2), ('app', 'v_user', 'id', 'bigint', 'NO', '', 1)",
				"INSERT INTO information_schema.table_constraints VALUES ('app', 'app', 'base_user', 'PRIMARY', 'PRIMARY KEY')",
				"INSERT INTO information_schema.key_column_usage VALUES ('app', 'base_user', 'PRIMARY', 'id', 1)",
			} {
				if _, err = db.Exec(statement); err != nil { //nolint:forbidigo // 仅在隔离内存库创建固定元数据测试夹具。
					t.Fatal(err)
				}
			}
			var tables []*adminv1.BiSourceSchemaTable
			tables, err = discoverSourceTables(t.Context(), &configuredDatabase{db: db, driver: driver}, &models.BiSource{SchemaName: "app"})
			if err != nil {
				t.Fatal(err)
			}
			if len(tables) != 2 || tables[0].Key != "app.base_user" || tables[0].Comment != "用户" || tables[1].TableType != "VIEW" || tables[1].Comment != "" {
				t.Fatalf("表注释或原始标识不正确: %v", tables)
			}
			columns := tables[0].Columns
			if len(columns) != 2 || columns[0].Name != "id" || columns[0].Comment != "" || !columns[0].PrimaryKey || !columns[0].Unique || columns[1].Name != "dept_id" || columns[1].Comment != "所属部门" || !columns[1].Nullable {
				t.Fatalf("字段注释或约束不正确: %v", columns)
			}
		})
	}
}

// TestConfiguredQueryRejectsUnmappedProject 验证未配置项目隔离的模型不能扩大项目授权。
func TestConfiguredQueryRejectsUnmappedProject(t *testing.T) {
	item := &models.BiSource{MaxRows: 100}
	schema := testConfiguredSchema()
	spec := bi.QuerySpec{Measures: []bi.MeasureSpec{{Code: "amount", Aggregation: "sum"}}, Limit: 20}
	for _, driver := range []string{"mysql", "postgres", "tidb", "doris"} {
		query, args, _, err := buildConfiguredQuery(driver, item, schema, bi.Scope{TenantID: 7, ProjectID: 42}, spec)
		if err == nil || query != "" || len(args) != 0 {
			t.Fatalf("%s 将项目查询扩大为租户查询: query=%q args=%v err=%v", driver, query, args, err)
		}
	}
}

// TestConfiguredSourceTruncation 验证超出上限时截断，恰好达到上限时不会误报。
func TestConfiguredSourceTruncation(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		closeErr := db.Close()
		if closeErr != nil {
			t.Error(closeErr)
		}
	})
	for _, statement := range []string{
		"ATTACH DATABASE ':memory:' AS analytics",
		"CREATE TABLE analytics.orders (tenant_id INTEGER, created_at TEXT, amount INTEGER)",
		"INSERT INTO analytics.orders VALUES (7,'2026-11-01',10),(7,'2026-11-02',20),(7,'2026-11-03',30),(7,'2026-11-04',40),(8,'2026-11-05',999)",
	} {
		if _, err = db.Exec(statement); err != nil { //nolint:forbidigo // 仅创建固定内存测试数据。
			t.Fatal(err)
		}
	}
	for _, test := range []struct {
		maxRows, limit int32
		rows           int
		truncated      bool
	}{{100, 2, 2, true}, {4, 4, 4, false}, {4, 0, 4, false}, {3, 100, 3, true}} {
		var result bi.QueryResult
		result, err = executeConfiguredSource(t.Context(), &configuredDatabase{db: db, driver: "mysql"}, &models.BiSource{MaxRows: test.maxRows}, testConfiguredSchema(), bi.Scope{TenantID: 7}, bi.QuerySpec{Dimensions: []string{"day"}, Measures: []bi.MeasureSpec{{Code: "amount", Aggregation: "sum"}}, Limit: test.limit})
		if err != nil || len(result.Rows) != test.rows || result.Truncated != test.truncated {
			t.Fatalf("上限%d/请求%d: result=%v err=%v", test.maxRows, test.limit, result, err)
		}
	}
}

// TestBuildConfiguredQueryMySQL 验证租户条件及额外一行的截断探测查询。
func TestBuildConfiguredQueryMySQL(t *testing.T) {
	item := &models.BiSource{MaxRows: 100}
	schema := testConfiguredSchema()
	spec := bi.QuerySpec{Dimensions: []string{"day"}, Measures: []bi.MeasureSpec{{Code: "amount", Aggregation: "sum"}}, Filters: []bi.FilterSpec{{Field: "day", Operator: "eq", Values: []string{"2026-10-30"}}}, Limit: 20}
	query, args, columns, err := buildConfiguredQuery("mysql", item, schema, bi.Scope{TenantID: 7}, spec)
	if err != nil {
		t.Fatalf("buildConfiguredQuery() error = %v", err)
	}
	if !strings.Contains(query, "`analytics`.`orders` AS `t0`") || !strings.Contains(query, "`t0`.`tenant_id` = ?") || !strings.Contains(query, "LIMIT 21") {
		t.Fatalf("unexpected query: %s", query)
	}
	if len(args) != 2 || args[0] != int64(7) || args[1] != "2026-10-30" || len(columns) != 2 {
		t.Fatalf("unexpected args or columns: %#v %#v", args, columns)
	}
}

func TestBuildConfiguredQueryPostgresUsesQualifiedIdentifiers(t *testing.T) {
	item := &models.BiSource{MaxRows: 100}
	schema := testConfiguredSchema()
	schema.Tables[0].SchemaName = "public"
	schema.Tables[0].Name = "orders"
	schema.Tables[0].TenantColumn = ""
	schema.Dimensions[0].Column = "amount"
	schema.Measures[0].Column = "amount"
	spec := bi.QuerySpec{Dimensions: []string{"day"}, Measures: []bi.MeasureSpec{{Code: "amount", Aggregation: "sum"}}, Filters: []bi.FilterSpec{{Field: "day", Operator: "eq", Values: []string{"10"}}}}
	query, _, _, err := buildConfiguredQuery("postgres", item, schema, bi.Scope{}, spec)
	if err != nil {
		t.Fatalf("buildConfiguredQuery() error = %v", err)
	}
	if !strings.Contains(query, `"public"."orders" AS "t0"`) || !strings.Contains(query, `"t0"."amount" = $1`) {
		t.Fatalf("unexpected query: %s", query)
	}
}

func TestBuildConfiguredQueryJoinsUniqueDimensionAndScopesBothTables(t *testing.T) {
	item := &models.BiSource{MaxRows: 100}
	schema := testConfiguredSchema()
	schema.Tables = append(schema.Tables, dto.ConfiguredSourceTable{Key: "analytics.customers", SchemaName: "analytics", Name: "customers", TenantColumn: "tenant_id"})
	schema.Relations = []dto.ConfiguredSourceRelation{{LeftTable: "analytics.orders", LeftColumn: "customer_id", RightTable: "analytics.customers", RightColumn: "id", JoinType: "left"}}
	schema.Dimensions = append(schema.Dimensions, dto.ConfiguredSourceDimension{Code: "customer_name", Name: "客户", Type: bi.FieldTypeString, Filterable: true, Table: "analytics.customers", Column: "name"})
	spec := bi.QuerySpec{Dimensions: []string{"customer_name"}, Measures: []bi.MeasureSpec{{Code: "amount", Aggregation: "sum"}}, Limit: 10}
	query, args, _, err := buildConfiguredQuery("mysql", item, schema, bi.Scope{TenantID: 9}, spec)
	if err != nil {
		t.Fatalf("buildConfiguredQuery() error = %v", err)
	}
	if !strings.Contains(query, "LEFT JOIN `analytics`.`customers` AS `t1` ON `t0`.`customer_id` = `t1`.`id` AND `t1`.`tenant_id` = ?") || !strings.Contains(query, "WHERE `t0`.`tenant_id` = ?") || !strings.Contains(query, "GROUP BY `t1`.`name`") {
		t.Fatalf("unexpected query: %s", query)
	}
	if len(args) != 2 || args[0] != int64(9) || args[1] != int64(9) {
		t.Fatalf("tenant arguments are out of order: %#v", args)
	}
}

func TestBuildConfiguredQuerySupportsTimeRangeAndOrder(t *testing.T) {
	item := &models.BiSource{MaxRows: 100}
	schema := testConfiguredSchema()
	schema.Tables[0].TenantColumn = ""
	spec := bi.QuerySpec{
		Dimensions: []string{"day"},
		Measures:   []bi.MeasureSpec{{Code: "amount", Aggregation: "sum"}},
		TimeRange:  &bi.TimeRangeSpec{Field: "day", Start: "2026-10-01", End: "2026-10-31"},
		Orders:     []bi.OrderSpec{{Field: "amount", Direction: "desc"}},
		Limit:      20,
	}
	query, args, _, err := buildConfiguredQuery("mysql", item, schema, bi.Scope{}, spec)
	if err != nil {
		t.Fatalf("buildConfiguredQuery() error = %v", err)
	}
	if !strings.Contains(query, "`t0`.`created_at` BETWEEN ? AND ?") || !strings.Contains(query, "ORDER BY `amount` DESC") {
		t.Fatalf("unexpected query: %s", query)
	}
	if len(args) != 2 || args[0] != "2026-10-01" || args[1] != "2026-10-31" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestBuildConfiguredQueryRejectsUnsafeOrderDirection(t *testing.T) {
	item := &models.BiSource{MaxRows: 100}
	schema := testConfiguredSchema()
	_, _, _, err := buildConfiguredQuery("mysql", item, schema, bi.Scope{}, bi.QuerySpec{
		Measures: []bi.MeasureSpec{{Code: "amount", Aggregation: "sum"}},
		Orders:   []bi.OrderSpec{{Field: "amount", Direction: "desc, id"}},
	})
	if err == nil {
		t.Fatal("expected invalid order direction error")
	}
}

// testConfiguredSchema 返回单事实表的测试模型。
func testConfiguredSchema() dto.ConfiguredSourceSchema {
	return dto.ConfiguredSourceSchema{
		BaseTable:  "analytics.orders",
		Tables:     []dto.ConfiguredSourceTable{{Key: "analytics.orders", SchemaName: "analytics", Name: "orders", TenantColumn: "tenant_id"}},
		Dimensions: []dto.ConfiguredSourceDimension{{Code: "day", Name: "日期", Type: bi.FieldTypeDate, Filterable: true, Table: "analytics.orders", Column: "created_at"}},
		Measures:   []dto.ConfiguredSourceMeasure{{Code: "amount", Name: "金额", Type: bi.FieldTypeNumber, Aggregations: []string{"sum"}, Table: "analytics.orders", Column: "amount"}},
	}
}
