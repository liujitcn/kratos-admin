package dbmeta

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"testing"
	"time"

	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	"github.com/liujitcn/kratos-kit/auth/data"
	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"
	_ "github.com/liujitcn/kratos-kit/database/gorm/driver/postgres"
)

// TestPostgresReadOnlyMetadata 验证 PostgreSQL 只读会话可读到键，业务行的原生 SQL 隔离仍然生效。
func TestPostgresReadOnlyMetadata(t *testing.T) {
	dsn := os.Getenv("KRATOS_DBMETA_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("设置 KRATOS_DBMETA_TEST_POSTGRES_DSN 指向支持建角色和模式的独立 PostgreSQL 测试库")
	}
	client, cleanup, err := kitgorm.NewGormClient(&configv1.Data_Database{Driver: "postgres", Source: dsn})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	tx := client.DB.WithContext(t.Context()).Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer func() {
		if rollbackErr := tx.Rollback().Error; rollbackErr != nil {
			t.Error(rollbackErr)
		}
	}()
	schema := fmt.Sprintf("dbmeta_%d", time.Now().UnixNano())
	reader := schema + "_reader"
	for _, stmt := range []string{
		"CREATE SCHEMA " + schema,
		"SET LOCAL search_path = " + schema,
		"CREATE ROLE " + reader,
		"CREATE TABLE sales (id varchar(16) PRIMARY KEY, tenant_id bigint NOT NULL, code varchar(16), external_code varchar(16), spare text, notes varchar(60), amount numeric(12,2), UNIQUE (tenant_id, code))",
		"CREATE UNIQUE INDEX sales_external_code ON sales (external_code) INCLUDE (spare)",
		"COMMENT ON COLUMN sales.notes IS '备注'",
		"GRANT USAGE ON SCHEMA " + schema + " TO " + reader,
		"GRANT SELECT ON sales TO " + reader,
		"SET LOCAL ROLE " + reader,
	} {
		if err = tx.Exec(stmt).Error; err != nil { //nolint:forbidigo // 在可回滚的独立测试事务中建立只读角色及元数据夹具。
			t.Fatal(err)
		}
	}
	identity := (&data.UserTokenPayload{UserId: 1, TenantId: 1, TenantCode: "0000", RoleCode: "super"}).MakeAuthClaims()
	ctx := engine.ContextWithAuthClaims(t.Context(), identity)
	var columns []Column
	columns, err = ListColumns(ctx, tx, "postgres", "", []string{"sales"}, true)
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, column := range columns {
		keys = append(keys, column.ColumnKey)
	}
	if !slices.Equal(keys, []string{"PRI", "UNI", "UNI", "", "", "", ""}) {
		t.Fatalf("只读角色的主键和约束类型错误: %v", keys)
	}
	var constraints []ConstraintColumn
	constraints, err = ListConstraintColumns(ctx, tx, schema, []string{"sales"})
	if err != nil {
		t.Fatal(err)
	}
	if len(constraints) != 3 ||
		constraints[0].ColumnName != "id" || constraints[0].ConstraintType != "PRIMARY KEY" ||
		constraints[1].ColumnName != "tenant_id" || constraints[2].ColumnName != "code" ||
		constraints[1].ConstraintName != constraints[2].ConstraintName || constraints[2].ConstraintType != "UNIQUE" {
		t.Fatalf("BI 约束元数据未保留主键和复合约束: %v", constraints)
	}
	var storage []StorageColumn
	storage, err = ListStorageColumns(ctx, tx, "postgres", "sales")
	if err != nil {
		t.Fatal(err)
	}
	var protected []string
	for _, column := range storage {
		if column.IsKey {
			protected = append(protected, column.Name)
		}
	}
	if !slices.Equal(protected, []string{"id", "tenant_id", "code", "external_code"}) {
		t.Fatalf("脱敏保护字段未正确包含主键、复合约束和唯一索引的键列: %v", protected)
	}
	if len(storage) != 7 || storage[5].Name != "notes" || storage[5].Comment != "备注" ||
		storage[5].DataType != "varchar" || storage[5].ColumnType != "character varying(60)" {
		t.Fatalf("脱敏字段类型、注释或顺序错误: %v", storage)
	}
	if err = tx.WithContext(ctx).Exec("SELECT * FROM sales").Error; !errors.Is(err, kitgorm.ErrRawDataIsolationUnsupported) { //nolint:forbidigo // 断言业务表原生 SQL 未被放行。
		t.Fatalf("业务表的原生 SQL 应继续被隔离钩子拒绝，实际: %v", err)
	}
}
