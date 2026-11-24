package dbmeta

import (
	"slices"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestMetadataScope 验证表、字段和约束都按请求范围扫描，不混入其他 Schema 或表。
func TestMetadataScope(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()
	fixture := []string{
		"ATTACH DATABASE ':memory:' AS information_schema",
		"CREATE TABLE information_schema.tables (table_schema TEXT, table_name TEXT, table_type TEXT, table_comment TEXT)",
		"CREATE TABLE information_schema.columns (table_schema TEXT, table_name TEXT, column_name TEXT, data_type TEXT, is_nullable TEXT, column_comment TEXT, ordinal_position INT)",
		"CREATE TABLE information_schema.table_constraints (table_schema TEXT, table_name TEXT, constraint_schema TEXT, constraint_name TEXT, constraint_type TEXT)",
		"CREATE TABLE information_schema.key_column_usage (table_schema TEXT, table_name TEXT, constraint_schema TEXT, constraint_name TEXT, column_name TEXT, ordinal_position INT)",
		"INSERT INTO information_schema.tables VALUES ('inventory', 'orders', 'BASE TABLE', '订单'), ('inventory', 'plain', 'BASE TABLE', ''), ('inventory', 'summary', 'VIEW', ''), ('other', 'orders', 'BASE TABLE', '')",
		"INSERT INTO information_schema.columns VALUES ('inventory', 'orders', 'tenant_id', 'int', 'NO', '', 2), ('inventory', 'orders', 'id', 'int', 'NO', '主键', 1), ('inventory', 'plain', 'name', 'varchar', 'YES', '', 1), ('other', 'orders', 'secret', 'varchar', 'YES', '', 1)",
		"INSERT INTO information_schema.table_constraints VALUES ('inventory', 'orders', 'inventory', 'shared', 'UNIQUE')",
		"INSERT INTO information_schema.key_column_usage VALUES ('inventory', 'orders', 'inventory', 'shared', 'id', 1), ('inventory', 'orders', 'inventory', 'shared', 'tenant_id', 2), ('inventory', 'plain', 'inventory', 'shared', 'name', 1)",
	}
	for _, stmt := range fixture {
		if err = db.WithContext(ctx).Exec(stmt).Error; err != nil { //nolint:forbidigo // 构造固定的内存系统目录，不访问实际数据库。
			t.Fatal(err)
		}
	}
	var tables []Table
	tables, err = ListTables(ctx, db, "mysql", TableScope{Schema: "inventory", IncludeViews: true})
	if err != nil || len(tables) != 3 || tables[0].TableComment != "订单" || tables[2].Type != "VIEW" {
		t.Fatalf("表或视图范围错误: %v, %v", tables, err)
	}
	tables, err = ListTables(ctx, db, "mysql", TableScope{Schema: "inventory", RequireColumn: "tenant_id"})
	if err != nil || len(tables) != 1 || tables[0].TableName != "orders" {
		t.Fatalf("未按必须字段筛选: %v, %v", tables, err)
	}
	var columns []Column
	columns, err = ListColumns(ctx, db, "mysql", "inventory", []string{"orders"}, false)
	if err != nil || len(columns) != 2 || columns[0].Name != "id" || columns[1].Name != "tenant_id" {
		t.Fatalf("字段顺序或范围错误: %v, %v", columns, err)
	}
	var keys []ConstraintColumn
	keys, err = ListConstraintColumns(ctx, db, "inventory", []string{"orders"})
	var names []string
	for _, key := range keys {
		names = append(names, key.ColumnName)
	}
	if err != nil || !slices.Equal(names, []string{"id", "tenant_id"}) {
		t.Fatalf("复合约束混入同名约束字段: %v, %v", keys, err)
	}
}
