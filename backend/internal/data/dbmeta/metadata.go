package dbmeta

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Table 承载数据库表元数据查询结果。
type Table struct {
	SchemaName   string `gorm:"column:table_schema"`  // 物理 schema
	Type         string `gorm:"column:table_type"`    // 表或视图
	TableName    string `gorm:"column:table_name"`    // 数据库表名
	TableComment string `gorm:"column:table_comment"` // 业务表描述
}

// Column 承载数据库字段元数据查询结果。
type Column struct {
	TableName              string         `gorm:"column:table_name"`               // 数据库表名
	Name                   string         `gorm:"column:column_name"`              // 数据库字段名
	Comment                string         `gorm:"column:column_comment"`           // 数据库字段注释
	DataType               string         `gorm:"column:data_type"`                // 数据库字段类型
	ColumnType             string         `gorm:"column:column_type"`              // 数据库完整字段类型
	ColumnKey              string         `gorm:"column:column_key"`               // 数据库字段索引类型
	IsNullable             string         `gorm:"column:is_nullable"`              // 是否允许为空
	Extra                  string         `gorm:"column:extra"`                    // 数据库字段扩展信息
	OrdinalPosition        int32          `gorm:"column:ordinal_position"`         // 数据库字段顺序
	CharacterMaximumLength sql.NullInt64  `gorm:"column:character_maximum_length"` // 字符字段长度
	NumericPrecision       sql.NullInt64  `gorm:"column:numeric_precision"`        // 数值字段精度
	NumericScale           sql.NullInt64  `gorm:"column:numeric_scale"`            // 数值字段小数位
	ColumnDefault          sql.NullString `gorm:"column:column_default"`           // 数据库字段默认值
}

// TableScope 限制物理表查询范围；空 Schema 使用连接的当前库或模式。
type TableScope struct {
	Schema        string
	Names         []string
	IncludeViews  bool
	RequireColumn string
	Limit         int
}

// Bind 使 BI 的已有连接池可用方言查询元数据，不开新池也不探测连接。
func Bind(pool *sql.DB, driver string) (*gorm.DB, error) {
	cfg := &gorm.Config{DisableAutomaticPing: true}
	switch {
	case MysqlFamilyDriver(driver):
		return gorm.Open(mysql.New(mysql.Config{Conn: pool, SkipInitializeWithVersion: true}), cfg)
	case strings.EqualFold(driver, "postgres"):
		return gorm.Open(postgres.New(postgres.Config{Conn: pool}), cfg)
	default:
		return nil, fmt.Errorf("不支持的元数据查询驱动: %s", driver)
	}
}

// ListTables 查询物理表名、类型和注释。
//
//nolint:forbidigo // information_schema 无生成模型，固定 SQL 片段与参数绑定分离。
func ListTables(ctx context.Context, db *gorm.DB, driver string, scope TableScope) ([]Table, error) {
	// 系统目录没有生成模型，库名、表名及字段条件统一绑定参数。
	q := db.WithContext(ctx).Table("information_schema.tables AS t").
		Select("t.table_schema, t.table_name, t.table_type, COALESCE("+tableCommentExpr(driver, "t.")+", '') AS table_comment").
		Where("t.table_type IN ?", tableTypes(scope.IncludeViews)).
		Order("t.table_name")
	if scope.Schema == "" {
		q = q.Where("t.table_schema = " + CurrentSchemaExpr(driver))
	} else {
		q = q.Where("t.table_schema = ?", scope.Schema)
	}
	if len(scope.Names) > 0 {
		q = q.Where("t.table_name IN ?", scope.Names)
	}
	if scope.RequireColumn != "" {
		q = q.Where("EXISTS (SELECT 1 FROM information_schema.columns c WHERE c.table_schema = t.table_schema AND c.table_name = t.table_name AND c.column_name = ?)", scope.RequireColumn)
	}
	if scope.Limit > 0 {
		q = q.Limit(scope.Limit)
	}
	var tables []Table
	err := q.Find(&tables).Error
	return tables, err
}

// ListColumns 按表和字段顺序读取指定表的元数据；完整模式包含代码生成所需的类型、精度和默认值。
//
//nolint:forbidigo // information_schema 无生成模型，固定 SQL 片段与参数绑定分离。
func ListColumns(ctx context.Context, db *gorm.DB, driver, schema string, tables []string, full bool) ([]Column, error) {
	if len(tables) == 0 {
		return nil, nil
	}
	projection := "table_name, column_name, data_type, is_nullable, COALESCE(column_comment, '') AS column_comment"
	if !MysqlFamilyDriver(driver) {
		projection = "table_name, column_name, data_type, is_nullable, COALESCE(col_description(to_regclass(format('%I.%I', table_schema, table_name)), ordinal_position), '') AS column_comment"
	}
	if full {
		projection = columnMetadataColumns(driver)
	}
	q := db.WithContext(ctx).Table("information_schema.columns").
		Select(projection).
		Where("table_name IN ?", tables).
		Order("table_name, ordinal_position")
	if schema == "" {
		q = q.Where("table_schema = " + CurrentSchemaExpr(driver))
	} else {
		q = q.Where("table_schema = ?", schema)
	}
	var columns []Column
	err := q.Find(&columns).Error
	return columns, err
}

// ConstraintColumn 表示主键或唯一约束中的一个字段。
type ConstraintColumn struct {
	TableName      string
	ConstraintName string
	ConstraintType string
	ColumnName     string
}

// StorageColumn 承载入库脱敏所需的字段类型与键成员信息。
type StorageColumn struct {
	Name       string
	Comment    string
	DataType   string
	ColumnType string
	IsKey      bool // 字段是否属于主键或唯一索引（含复合键）
}

// ListConstraintColumns 查询指定表的主键、唯一约束和按序组成字段，支持 PostgreSQL 只读连接。
//
//nolint:forbidigo // 系统目录无生成模型，查询只绑定请求的模式名、表名。
func ListConstraintColumns(ctx context.Context, db *gorm.DB, schema string, tables []string) ([]ConstraintColumn, error) {
	if len(tables) == 0 {
		return nil, nil
	}
	q := db.WithContext(ctx)
	if db.Dialector.Name() == "postgres" {
		q = q.Table("pg_catalog.pg_constraint AS c").
			Select("t.relname AS table_name, c.conname AS constraint_name, CASE c.contype WHEN 'p' THEN 'PRIMARY KEY' ELSE 'UNIQUE' END AS constraint_type, a.attname AS column_name").
			Joins("JOIN pg_catalog.pg_class t ON t.oid = c.conrelid").
			Joins("JOIN pg_catalog.pg_namespace n ON n.oid = t.relnamespace").
			Joins("JOIN LATERAL unnest(c.conkey) WITH ORDINALITY AS k(attnum, position) ON TRUE").
			Joins("JOIN pg_catalog.pg_attribute a ON a.attrelid = t.oid AND a.attnum = k.attnum").
			Where("n.nspname = ? AND t.relname IN ? AND c.contype IN ('p', 'u')", schema, tables).
			Order("t.relname, c.conname, k.position")
	} else {
		q = q.Table("information_schema.table_constraints AS tc").
			Select("tc.table_name, tc.constraint_name, tc.constraint_type, kcu.column_name").
			Joins("JOIN information_schema.key_column_usage kcu ON tc.constraint_schema = kcu.constraint_schema AND tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema AND tc.table_name = kcu.table_name").
			Where("tc.table_schema = ? AND tc.table_name IN ? AND tc.constraint_type IN ('PRIMARY KEY', 'UNIQUE')", schema, tables).
			Order("tc.table_name, tc.constraint_name, kcu.ordinal_position")
	}
	var columns []ConstraintColumn
	err := q.Find(&columns).Error
	return columns, err
}

// ListStorageColumns 只查当前库的系统目录，不读取业务行或跳过数据隔离。
//
//nolint:forbidigo // 系统目录无生成模型，固定方言表达式不拼接请求表名。
func ListStorageColumns(ctx context.Context, db *gorm.DB, driver, table string) ([]StorageColumn, error) {
	q := db.WithContext(ctx).Table("information_schema.columns AS c").
		Where("c.table_name = ?", table).
		Where("c.table_schema = " + CurrentSchemaExpr(driver)).
		Order("c.ordinal_position")
	if MysqlFamilyDriver(driver) {
		q = q.Select(`c.column_name AS name, c.column_comment AS comment, c.data_type, c.column_type,
EXISTS (SELECT 1 FROM information_schema.statistics s WHERE s.table_schema = c.table_schema
	AND s.table_name = c.table_name AND s.column_name = c.column_name AND s.non_unique = 0) AS is_key`)
	} else {
		q = q.Joins("JOIN pg_catalog.pg_attribute a ON a.attrelid = to_regclass(format('%I.%I', c.table_schema, c.table_name)) AND a.attnum = c.ordinal_position").
			Select(`c.column_name AS name, COALESCE(col_description(a.attrelid, a.attnum), '') AS comment,
c.udt_name AS data_type, format_type(a.atttypid, a.atttypmod) AS column_type,
EXISTS (SELECT 1 FROM pg_catalog.pg_index i JOIN LATERAL unnest(i.indkey) WITH ORDINALITY AS k(attnum, position)
	ON k.position <= i.indnkeyatts WHERE i.indrelid = a.attrelid AND i.indisunique AND k.attnum = a.attnum) AS is_key`)
	}
	var columns []StorageColumn
	err := q.Find(&columns).Error
	return columns, err
}

// tableTypes 返回指定业务范围内可见的物理表类型。
func tableTypes(includeViews bool) []string {
	if includeViews {
		return []string{"BASE TABLE", "VIEW"}
	}
	return []string{"BASE TABLE"}
}

// MysqlFamilyDriver 判断驱动是否属于 MySQL 方言系（含复用 MySQL 方言的 Doris 与 TiDB）。
func MysqlFamilyDriver(driverName string) bool {
	switch strings.ToLower(driverName) {
	case "mysql", "doris", "tidb":
		return true
	default:
		return false
	}
}

// CurrentSchemaExpr 按方言返回当前库名（MySQL）或当前模式名（PostgreSQL）表达式。
func CurrentSchemaExpr(driverName string) string {
	if MysqlFamilyDriver(driverName) {
		return "DATABASE()"
	}
	return "current_schema()"
}

// tableCommentExpr 按方言返回读取表注释的表达式，qualifier 为 information_schema.tables 的引用限定（如 "tables."）。
func tableCommentExpr(driverName string, qualifier string) string {
	if MysqlFamilyDriver(driverName) {
		return qualifier + "table_comment"
	}
	// PostgreSQL 表注释存于 pg_description，按表 OID 读取，表不存在或未注释时返回空串。
	return "COALESCE(obj_description(to_regclass(format('%I.%I', " + qualifier + "table_schema, " + qualifier + "table_name)), 'pg_class'), '')"
}

// columnMetadataColumns 按方言返回 information_schema.columns 的字段元数据列表达式，
// 输出列名与 MySQL 对齐（column_comment/column_type/column_key/extra）。
func columnMetadataColumns(driverName string) string {
	if MysqlFamilyDriver(driverName) {
		return "table_name, column_name, column_comment, data_type, column_type, column_key, is_nullable, extra, ordinal_position, character_maximum_length, numeric_precision, numeric_scale, column_default"
	}
	// PostgreSQL 只读角色看不到 information_schema 的约束行，直接查可见字段所属表的约束目录。
	return `table_name,
column_name,
COALESCE(col_description(to_regclass(format('%I.%I', table_schema, table_name)), ordinal_position), '') AS column_comment,
data_type,
CASE
	WHEN data_type IN ('character varying', 'character') AND character_maximum_length IS NOT NULL THEN 'varchar(' || character_maximum_length || ')'
	WHEN data_type = 'character varying' THEN 'varchar'
	WHEN data_type = 'numeric' AND numeric_scale IS NOT NULL THEN 'decimal(' || numeric_precision || ',' || numeric_scale || ')'
	WHEN data_type LIKE 'timestamp%' THEN 'timestamp'
	WHEN data_type LIKE 'time%' THEN 'time'
	ELSE data_type
END AS column_type,
COALESCE((SELECT CASE c.contype WHEN 'p' THEN 'PRI' ELSE 'UNI' END
	FROM pg_catalog.pg_constraint c
	WHERE c.conrelid = to_regclass(format('%I.%I', table_schema, table_name))
		AND c.contype IN ('p', 'u') AND ordinal_position = ANY(c.conkey)
	ORDER BY c.contype LIMIT 1), '') AS column_key,
is_nullable,
CASE WHEN column_default LIKE 'nextval%' OR is_identity = 'YES' THEN 'auto_increment' ELSE '' END AS extra,
ordinal_position, character_maximum_length, numeric_precision, numeric_scale, column_default`
}
