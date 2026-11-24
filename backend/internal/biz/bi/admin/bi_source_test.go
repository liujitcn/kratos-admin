package admin

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/liujitcn/kratos-admin/backend/adapter/kit"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/query"
	"github.com/liujitcn/kratos-admin/backend/internal/data/dbconn"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	"github.com/liujitcn/kratos-kit/sdk"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// sourceTableDataTestRow 提供在线表数据预览的可排序和租户隔离字段。
type sourceTableDataTestRow struct {
	ID       int64 `gorm:"primaryKey"`
	TenantID int64 `gorm:"column:tenant_id"`
	Name     string
	Amount   int64
	Created  time.Time `gorm:"column:created_at"`
}

// TableName 固定在线预览测试表名称。
func (sourceTableDataTestRow) TableName() string { return "source_table_data_test_rows" }

// TestPageSourceTableDataFiltersTenantAndPaginates 验证在线表数据预览的租户过滤、总数、分页和单元格格式化。
func TestPageSourceTableDataFiltersTenantAndPaginates(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:bi_source_table_data?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	var sqlDB *sql.DB
	sqlDB, err = db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err = sqlDB.Close(); err != nil {
			t.Error(err)
		}
	})
	if err = db.AutoMigrate(&sourceTableDataTestRow{}); err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&[]sourceTableDataTestRow{
		{ID: 1, TenantID: 1, Name: "一", Amount: 10, Created: time.Date(2026, 11, 17, 10, 0, 0, 0, time.Local)},
		{ID: 2, TenantID: 1, Name: "二", Amount: 20, Created: time.Date(2026, 11, 17, 11, 0, 0, 0, time.Local)},
		{ID: 3, TenantID: 1, Name: "三", Amount: 30, Created: time.Date(2026, 11, 17, 12, 0, 0, 0, time.Local)},
		{ID: 4, TenantID: 2, Name: "其他租户", Amount: 99, Created: time.Date(2026, 11, 17, 13, 0, 0, 0, time.Local)},
	}).Error; err != nil {
		t.Fatal(err)
	}

	table := &adminv1.BiSourceSchemaTable{
		Key:  "source_table_data_test_rows",
		Name: "source_table_data_test_rows",
		Columns: []*adminv1.BiSourceSchemaColumn{
			{Name: "id", PrimaryKey: true},
			{Name: "tenant_id"},
			{Name: "name"},
			{Name: "amount"},
			{Name: "created_at"},
		},
	}
	result, err := pageSourceTableData(t.Context(), &configuredDatabase{db: sqlDB, driver: "sqlite"}, table, "tenant_id", 1, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 3 {
		t.Fatalf("总数错误: %d", result.Total)
	}
	if got := sourceTableDataColumnNames(result.Columns); len(got) != 5 || got[0] != "id" || got[4] != "created_at" {
		t.Fatalf("字段错误: %v", got)
	}
	if got := sourceTableDataRowValues(result.Rows); len(got) != 1 || len(got[0]) != 5 || got[0][0] != "3" || got[0][2] != "三" || got[0][4] != "2026-11-17 12:00:00" {
		t.Fatalf("分页数据错误: %v", got)
	}

	result, err = pageSourceTableData(t.Context(), &configuredDatabase{db: sqlDB, driver: "sqlite"}, table, "", 1, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 4 {
		t.Fatalf("共享表总数错误: %d", result.Total)
	}
}

// TestFindSourceTableDataTargetRejectsUnknownIdentifiers 验证表名和租户字段只能使用发现结果中的标识符。
func TestFindSourceTableDataTargetRejectsUnknownIdentifiers(t *testing.T) {
	table := &adminv1.BiSourceSchemaTable{Key: "public.orders", Name: "orders", Columns: []*adminv1.BiSourceSchemaColumn{{Name: "id"}, {Name: "tenant_id"}}}
	_, err := findSourceTableDataTarget([]*adminv1.BiSourceSchemaTable{table}, "public.missing", "")
	if err == nil {
		t.Fatal("未知表被接受")
	}
	_, err = findSourceTableDataTarget([]*adminv1.BiSourceSchemaTable{table}, table.Key, "missing")
	if err == nil {
		t.Fatal("未知租户字段被接受")
	}
}

// TestConfiguredSourceCalendarGrouping 验证目录和编译只向实际含日期的字段开放日历粒度。
func TestConfiguredSourceCalendarGrouping(t *testing.T) {
	const table = "public.orders"
	model, err := configuredSourceModel(&adminv1.BiSourceModel{
		BaseTable: table,
		Tables:    []*adminv1.BiSourceModelTable{{Key: table}},
		Dimensions: []*adminv1.BiSourceDimension{
			{Code: "created", Name: "创建日期", Type: "date", Table: table, Column: "created", Filterable: true},
			{Code: "opens", Name: "营业时刻", Type: "date", Table: table, Column: "opens", Filterable: true},
			{Code: "closes", Name: "打烊时刻", Type: "string", Table: table, Column: "closes", Filterable: true},
		},
		Measures: []*adminv1.BiSourceMeasure{{Code: "orders", Name: "订单数", Type: "number", Expression: "COUNT(*)"}},
	}, []*adminv1.BiSourceSchemaTable{{Key: table, Name: "orders", Columns: []*adminv1.BiSourceSchemaColumn{
		{Name: "created", DataType: "timestamp without time zone"},
		{Name: "opens", DataType: "time without time zone"},
		{Name: "closes", DataType: "time with time zone"},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	model.Measures[0].Revision = 1
	if compatibleSourceTypes("time without time zone", "text") {
		t.Fatal("纯时刻可作为文本显示，但不能和文本列建立关联")
	}
	descriptor := model.Descriptor("orders", "", "", 1, 100, "postgres")
	if got := query.TimeField(descriptor, bi.QuerySpec{Dimensions: []string{"opens", "closes", "created"}}); got != "created" {
		t.Fatalf("自动时间轴应选择实际日期列，得到 %q", got)
	}
	for _, field := range descriptor.Dimensions {
		want := field.Code == "created"
		if got := len(field.TimeGrains) > 0; got != want {
			t.Errorf("%s 日历分组能力为 %v，预期 %v", field.Code, got, want)
		}
		spec := bi.QuerySpec{SourceCode: "orders", Dimensions: []string{field.Code}, Measures: []bi.MeasureSpec{{Code: "orders"}}, GroupingOptions: []bi.GroupingOption{{Field: field.Code, TimeGrain: "month"}}}
		if _, err = query.Compile("postgres", 100, model, spec); (err == nil) != want {
			t.Errorf("%s 月度分组编译结果为 %v，预期成功 %v", field.Code, err, want)
		}
	}
}

func sourceTableDataColumnNames(columns []*adminv1.BiSourceSchemaColumn) []string {
	result := make([]string, 0, len(columns))
	for _, column := range columns {
		result = append(result, column.GetName())
	}
	return result
}

func sourceTableDataRowValues(rows []*adminv1.BiSourceTableDataRow) [][]string {
	result := make([][]string, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.GetValues())
	}
	return result
}

// TestConnectionTestPasswordScope 验证测试不能把其他租户或原连接的凭据转发给不同连接。
func TestConnectionTestPasswordScope(t *testing.T) {
	cards, _ := newBiWorkflowTestCases(t)
	c := cards.biQueryCase.biSourceCase
	ctx := biCardTestContext(1, 1, -1)
	item := &models.BiSource{TenantID: 1, Code: "credential_scope", Host: "db.example", Driver: "postgres", Port: 5432, DatabaseName: "demo", Username: "reader", Password: "encrypted"}
	err := c.sourceRepo.Create(ctx, item)
	if err != nil {
		t.Fatal(err)
	}
	config := dbconn.Config{Host: item.Host, Driver: item.Driver, Port: item.Port, DatabaseName: item.DatabaseName, Username: item.Username}
	if _, err = c.ConnectionTestPassword(biCardTestContext(2, 2, -1), item.ID, config); errors.Code(err) != 404 {
		t.Fatalf("跨租户测试应拒绝复用凭据: %v", err)
	}
	config.Host = "other.example"
	if _, err = c.ConnectionTestPassword(ctx, item.ID, config); errors.Code(err) != 400 {
		t.Fatalf("连接目标变更时必须重新填写密码: %v", err)
	}
}

// TestSourcePasswordRejectsStaleVersion 验证只允许与已读取连接参数同版本的凭据。
func TestSourcePasswordRejectsStaleVersion(t *testing.T) {
	cards, _ := newBiWorkflowTestCases(t)
	c := cards.biQueryCase.biSourceCase
	ctx := biCardTestContext(1, 1, -1)
	db := c.GormClients[databaseGorm.DefaultClientName].DB
	err := db.AutoMigrate(&models.BaseRedactRule{}, &models.BaseRedactStoragePolicy{}, &models.BaseRedactOutputPolicy{})
	if err != nil {
		t.Fatal(err)
	}
	for _, record := range []any{
		&models.BaseRedactRule{ID: 11, Code: "encrypt", RuleType: "ENCRYPT", DefaultParams: `{"encrypt":{"algorithm":"AES_GCM"}}`, Status: 1},
		&models.BaseRedactStoragePolicy{ID: 2010, TenantID: 1, SourceName: databaseGorm.DefaultClientName, TableName_: "bi_source", ColumnName: "password", RuleID: 11, RuleParams: "{}", Status: 1},
	} {
		if err = db.Create(record).Error; err != nil {
			t.Fatal(err)
		}
	}
	originalKey := sdk.Runtime.GetKey()
	sdk.Runtime.SetKey(biCredentialTestKey{})
	t.Cleanup(func() { sdk.Runtime.SetKey(originalKey) })
	c.redactResolver, err = kit.NewRedactPolicyResolver(c.GormClients)
	if err != nil {
		t.Fatal(err)
	}
	if err = c.redactResolver.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	item := &models.BiSource{TenantID: 1, Code: "versioned_credentials", Host: "db.example", Password: "fixture-password", Version: 2}
	if err = c.sourceRepo.Create(ctx, item); err != nil {
		t.Fatal(err)
	}
	config := dbconn.Config{Host: item.Host}
	var password string
	password, err = c.ConnectionTestPassword(ctx, item.ID, config)
	if err != nil || password != "fixture-password" {
		t.Fatalf("当前版本应能还原已保存的凭据: %v", err)
	}
	item.Version--
	password, err = c.sourcePassword(ctx, item)
	if errors.Code(err) != 409 || password != "" {
		t.Fatalf("版本不匹配时不得返回凭据: %v", err)
	}
}

// biCredentialTestKey 提供仅用于本地回归测试的固定密钥。
type biCredentialTestKey struct{}

// Derive 返回固定长度的测试密钥。
func (biCredentialTestKey) Derive(context.Context, string) ([]byte, error) {
	return make([]byte, 32), nil
}
