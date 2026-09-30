package biz

// 跨数据库元数据查询辅助：information_schema 的注释、完整类型、索引标记列是 MySQL 专有的，
// PostgreSQL 需映射 pg_catalog 等价实现；输出列名与 MySQL 对齐，扫描结构无需区分方言。

// MysqlFamilyDriver 判断驱动是否属于 MySQL 方言系（含复用 MySQL 方言的 Doris）。
func MysqlFamilyDriver(driverName string) bool {
	return driverName == "mysql" || driverName == "doris"
}

// CurrentSchemaExpr 按方言返回当前库名（MySQL）或当前模式名（PostgreSQL）表达式。
func CurrentSchemaExpr(driverName string) string {
	if MysqlFamilyDriver(driverName) {
		return "DATABASE()"
	}
	return "current_schema()"
}

// TableCommentExpr 按方言返回读取表注释的表达式，qualifier 为 information_schema.tables 的引用限定（如 "tables."）。
func TableCommentExpr(driverName string, qualifier string) string {
	if MysqlFamilyDriver(driverName) {
		return qualifier + "table_comment"
	}
	// PostgreSQL 表注释存于 pg_description，按表 OID 读取，表不存在或未注释时返回空串。
	return "COALESCE(obj_description(to_regclass(format('%I.%I', " + qualifier + "table_schema, " + qualifier + "table_name)), 'pg_class'), '')"
}

// TableMetadataColumns 按方言返回 information_schema.tables 的表名与注释列表达式。
func TableMetadataColumns(driverName string) string {
	if MysqlFamilyDriver(driverName) {
		return "table_name, table_comment"
	}
	return "table_name, " + TableCommentExpr(driverName, "") + " AS table_comment"
}

// ColumnMetadataColumns 按方言返回 information_schema.columns 的字段元数据列表达式，
// 输出列名与 MySQL 对齐（column_comment/column_type/column_key/extra）。
func ColumnMetadataColumns(driverName string) string {
	if MysqlFamilyDriver(driverName) {
		return "table_name, column_name, column_comment, data_type, column_type, column_key, is_nullable, extra, ordinal_position, character_maximum_length, numeric_precision, numeric_scale, column_default"
	}
	// PostgreSQL：注释走 col_description，完整类型按 data_type 合成，主键/唯一经约束视图推断，
	// 序列默认值或身份列视作自增。
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
COALESCE(
	(SELECT 'PRI' FROM information_schema.table_constraints tc JOIN information_schema.key_column_usage ku ON ku.constraint_name = tc.constraint_name AND ku.table_schema = tc.table_schema WHERE tc.table_schema = columns.table_schema AND tc.table_name = columns.table_name AND tc.constraint_type = 'PRIMARY KEY' AND ku.column_name = columns.column_name LIMIT 1),
	(SELECT 'UNI' FROM information_schema.table_constraints tc JOIN information_schema.key_column_usage ku ON ku.constraint_name = tc.constraint_name AND ku.table_schema = tc.table_schema WHERE tc.table_schema = columns.table_schema AND tc.table_name = columns.table_name AND tc.constraint_type = 'UNIQUE' AND ku.column_name = columns.column_name LIMIT 1),
	'') AS column_key,
is_nullable,
CASE WHEN column_default LIKE 'nextval%' OR is_identity = 'YES' THEN 'auto_increment' ELSE '' END AS extra,
ordinal_position, character_maximum_length, numeric_precision, numeric_scale, column_default`
}
