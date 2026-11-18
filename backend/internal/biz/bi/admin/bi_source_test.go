package admin

import (
	"database/sql"
	"testing"
	"time"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
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
