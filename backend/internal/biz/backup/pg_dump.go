package backup

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// PostgresDumpOptions 描述 Go PostgreSQL 导出的内容范围。
type PostgresDumpOptions struct {
	// Schema 是需要导出的模式名称，当前固定使用 public。
	Schema string
	// Table 是可选的数据表名称，为空时导出模式中的全部表。
	Table string
	// Where 是可选的数据筛选条件，仅允许由调用方使用受控值构造。
	Where string
	// IncludeSchema 表示是否导出表结构。
	IncludeSchema bool
	// IncludeData 表示是否导出表数据。
	IncludeData bool
}

// DumpPostgres 使用 Go 生成可由 PostgreSQL 客户端执行的 SQL 导出文件。
//
// 表结构覆盖列、默认值、序列、主键和索引；视图按定义重建。函数、触发器与外键
// 约束不在 Go 导出范围内，需要完整定义时应使用 pg_dump 命令通道。
func DumpPostgres(ctx context.Context, db *sql.DB, options PostgresDumpOptions, target io.Writer) error {
	if db == nil {
		return fmt.Errorf("PostgreSQL 数据库连接为空")
	}
	if target == nil {
		return fmt.Errorf("SQL 导出目标为空")
	}
	if options.Schema == "" {
		options.Schema = "public"
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return fmt.Errorf("开启 PostgreSQL 导出事务失败: %w", err)
	}
	defer tx.Rollback()
	if err = writeSQL(target, "-- kratos-admin Go PostgreSQL dump\n"); err != nil {
		return err
	}
	tables, err := listPostgresDumpTables(ctx, tx, options)
	if err != nil {
		return err
	}
	for _, table := range tables {
		if err = dumpPostgresTable(ctx, tx, options, table, target); err != nil {
			return err
		}
	}
	if err = dumpPostgresViews(ctx, tx, options, target); err != nil {
		return err
	}
	if err = writeSQL(target, "\n"); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("提交 PostgreSQL 导出事务失败: %w", err)
	}
	return nil
}

// dumpPostgresTable 导出单张表的结构和数据。
func dumpPostgresTable(ctx context.Context, tx *sql.Tx, options PostgresDumpOptions, tableName string, target io.Writer) error {
	qualified := qualifiedPostgresName(options.Schema, tableName)
	if options.IncludeSchema {
		sequences, createSQL, err := buildPostgresTableSchema(ctx, tx, options.Schema, tableName)
		if err != nil {
			return err
		}
		for _, sequence := range sequences {
			if err = writeSQL(target, "DROP SEQUENCE IF EXISTS "+sequence+" CASCADE;\n"); err != nil {
				return fmt.Errorf("写入序列删除语句失败: %w", err)
			}
			if err = writeSQL(target, "CREATE SEQUENCE "+sequence+";\n"); err != nil {
				return fmt.Errorf("写入序列创建语句失败: %w", err)
			}
		}
		if err = writeSQL(target, "DROP TABLE IF EXISTS "+qualified+" CASCADE;\n"); err != nil {
			return fmt.Errorf("写入表删除语句失败: %w", err)
		}
		if err = writeSQL(target, createSQL+";\n"); err != nil {
			return fmt.Errorf("写入表 %s 结构创建语句失败: %w", tableName, err)
		}
		indexes, err := listPostgresIndexes(ctx, tx, options.Schema, tableName)
		if err != nil {
			return err
		}
		for _, index := range indexes {
			if err = writeSQL(target, index+";\n"); err != nil {
				return fmt.Errorf("写入表 %s 索引语句失败: %w", tableName, err)
			}
		}
	}
	if !options.IncludeData {
		return nil
	}
	columns, err := listPostgresColumns(ctx, tx, options.Schema, tableName)
	if err != nil {
		return err
	}
	if len(columns) == 0 {
		return nil
	}
	columnNames := make([]string, 0, len(columns))
	for _, column := range columns {
		columnNames = append(columnNames, quotePostgresIdentifier(column))
	}
	query := "SELECT " + strings.Join(columnNames, ", ") + " FROM " + qualified
	if options.Where != "" {
		query += " WHERE " + options.Where
	}
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("查询 PostgreSQL 表 %s 数据失败: %w", tableName, err)
	}
	defer rows.Close()
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return fmt.Errorf("读取 PostgreSQL 表 %s 字段类型失败: %w", tableName, err)
	}
	values := make([]interface{}, len(columns))
	scanTargets := make([]interface{}, len(values))
	for index := range values {
		scanTargets[index] = &values[index]
	}
	for rows.Next() {
		if err = rows.Scan(scanTargets...); err != nil {
			return fmt.Errorf("读取 PostgreSQL 表 %s 数据失败: %w", tableName, err)
		}
		encodedValues := make([]string, len(values))
		for index, value := range values {
			encodedValues[index] = formatPostgresValue(value, columnTypes[index].DatabaseTypeName())
		}
		statement := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);\n", qualified, strings.Join(columnNames, ", "), strings.Join(encodedValues, ", "))
		if err = writeSQL(target, statement); err != nil {
			return fmt.Errorf("写入 PostgreSQL 表 %s 数据失败: %w", tableName, err)
		}
	}
	if err = rows.Err(); err != nil {
		return fmt.Errorf("遍历 PostgreSQL 表 %s 数据失败: %w", tableName, err)
	}
	return nil
}

// listPostgresDumpTables 按名称序查询需要导出的表。
func listPostgresDumpTables(ctx context.Context, tx *sql.Tx, options PostgresDumpOptions) ([]string, error) {
	query := "SELECT table_name FROM information_schema.tables WHERE table_schema = $1 AND table_type = 'BASE TABLE'"
	args := []interface{}{options.Schema}
	if options.Table != "" {
		query += " AND table_name = $2"
		args = append(args, options.Table)
	}
	rows, err := tx.QueryContext(ctx, query+" ORDER BY table_name", args...)
	if err != nil {
		return nil, fmt.Errorf("查询 PostgreSQL 导出表失败: %w", err)
	}
	defer rows.Close()
	tables := make([]string, 0)
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("读取 PostgreSQL 导出表失败: %w", err)
		}
		tables = append(tables, name)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历 PostgreSQL 导出表失败: %w", err)
	}
	if options.Table != "" && len(tables) == 0 {
		return nil, fmt.Errorf("PostgreSQL 数据表不存在: %s", options.Table)
	}
	return tables, nil
}

// dumpPostgresViews 在全部表之后导出视图定义；单表归档导出跳过视图。
func dumpPostgresViews(ctx context.Context, tx *sql.Tx, options PostgresDumpOptions, target io.Writer) error {
	if !options.IncludeSchema || options.Table != "" {
		return nil
	}
	rows, err := tx.QueryContext(ctx, "SELECT viewname, definition FROM pg_views WHERE schemaname = $1 ORDER BY viewname", options.Schema)
	if err != nil {
		return fmt.Errorf("查询 PostgreSQL 视图失败: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var definition string
		if err = rows.Scan(&name, &definition); err != nil {
			return fmt.Errorf("读取 PostgreSQL 视图失败: %w", err)
		}
		view := "CREATE OR REPLACE VIEW " + qualifiedPostgresName(options.Schema, name) + " AS " + strings.TrimSpace(definition)
		if err = writeSQL(target, view+";\n"); err != nil {
			return fmt.Errorf("写入视图 %s 定义失败: %w", name, err)
		}
	}
	return rows.Err()
}

// buildPostgresTableSchema 构建单表建表语句，并返回需要先行创建的序列限定名列表。
func buildPostgresTableSchema(ctx context.Context, tx *sql.Tx, schema, table string) ([]string, string, error) {
	columns, err := listPostgresColumnDefinitions(ctx, tx, schema, table)
	if err != nil {
		return nil, "", err
	}
	if len(columns) == 0 {
		return nil, "", fmt.Errorf("PostgreSQL 表 %s 缺少字段定义", table)
	}
	primaryKey, err := listPostgresPrimaryKeyColumns(ctx, tx, schema, table)
	if err != nil {
		return nil, "", err
	}
	definitions := make([]string, 0, len(columns)+1)
	sequences := make([]string, 0)
	for _, column := range columns {
		definition := quotePostgresIdentifier(column.name) + " " + column.columnType
		if column.columnDefault != "" {
			if sequence := postgresSequenceName(column.columnDefault); sequence != "" {
				sequences = append(sequences, sequence)
			}
			definition += " DEFAULT " + column.columnDefault
		}
		if !column.nullable {
			definition += " NOT NULL"
		}
		definitions = append(definitions, definition)
	}
	if len(primaryKey) > 0 {
		quotedColumns := make([]string, 0, len(primaryKey))
		for _, name := range primaryKey {
			quotedColumns = append(quotedColumns, quotePostgresIdentifier(name))
		}
		definitions = append(definitions, "PRIMARY KEY ("+strings.Join(quotedColumns, ", ")+")")
	}
	createSQL := "CREATE TABLE " + qualifiedPostgresName(schema, table) + " (\n  " + strings.Join(definitions, ",\n  ") + "\n)"
	sort.Strings(sequences)
	return sequences, createSQL, nil
}

type postgresColumnDefinition struct {
	name          string
	columnType    string
	columnDefault string
	nullable      bool
}

// listPostgresColumnDefinitions 查询列的类型、默认值与可空性，类型带长度和精度修饰。
func listPostgresColumnDefinitions(ctx context.Context, tx *sql.Tx, schema, table string) ([]postgresColumnDefinition, error) {
	rows, err := tx.QueryContext(ctx, `SELECT column_name, data_type, COALESCE(column_default, ''),
COALESCE(character_maximum_length, 0), COALESCE(numeric_precision, 0), COALESCE(numeric_scale, 0), is_nullable
FROM information_schema.columns WHERE table_schema = $1 AND table_name = $2 ORDER BY ordinal_position`, schema, table)
	if err != nil {
		return nil, fmt.Errorf("查询 PostgreSQL 表 %s 字段失败: %w", table, err)
	}
	defer rows.Close()
	columns := make([]postgresColumnDefinition, 0)
	for rows.Next() {
		var column postgresColumnDefinition
		var name, dataType, isNullable string
		var length, precision, scale int64
		if err = rows.Scan(&name, &dataType, &column.columnDefault, &length, &precision, &scale, &isNullable); err != nil {
			return nil, fmt.Errorf("读取 PostgreSQL 表 %s 字段失败: %w", table, err)
		}
		column.name = name
		column.columnType = buildPostgresColumnType(dataType, length, precision, scale)
		column.nullable = isNullable == "YES"
		columns = append(columns, column)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历 PostgreSQL 表 %s 字段失败: %w", table, err)
	}
	return columns, nil
}

// buildPostgresColumnType 拼装带长度和精度修饰的列类型。
func buildPostgresColumnType(dataType string, length int64, precision int64, scale int64) string {
	switch dataType {
	case "character varying", "character":
		if length > 0 {
			return dataType + "(" + strconv.FormatInt(length, 10) + ")"
		}
		return dataType
	case "numeric":
		if precision > 0 {
			return "numeric(" + strconv.FormatInt(precision, 10) + "," + strconv.FormatInt(scale, 10) + ")"
		}
		return dataType
	default:
		return dataType
	}
}

// listPostgresPrimaryKeyColumns 按序返回主键列名。
func listPostgresPrimaryKeyColumns(ctx context.Context, tx *sql.Tx, schema, table string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `SELECT kcu.column_name
FROM information_schema.table_constraints tc
JOIN information_schema.key_column_usage kcu
  ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema
WHERE tc.table_schema = $1 AND tc.table_name = $2 AND tc.constraint_type = 'PRIMARY KEY'
ORDER BY kcu.ordinal_position`, schema, table)
	if err != nil {
		return nil, fmt.Errorf("查询 PostgreSQL 表 %s 主键失败: %w", table, err)
	}
	defer rows.Close()
	columns := make([]string, 0)
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("读取 PostgreSQL 表 %s 主键失败: %w", table, err)
		}
		columns = append(columns, name)
	}
	return columns, rows.Err()
}

// listPostgresIndexes 返回表上除主键约束索引外的全部索引定义。
func listPostgresIndexes(ctx context.Context, tx *sql.Tx, schema, table string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, "SELECT indexname, indexdef FROM pg_indexes WHERE schemaname = $1 AND tablename = $2 ORDER BY indexname", schema, table)
	if err != nil {
		return nil, fmt.Errorf("查询 PostgreSQL 表 %s 索引失败: %w", table, err)
	}
	defer rows.Close()
	definitions := make([]string, 0)
	for rows.Next() {
		var name, definition string
		if err = rows.Scan(&name, &definition); err != nil {
			return nil, fmt.Errorf("读取 PostgreSQL 表 %s 索引失败: %w", table, err)
		}
		// 主键约束的支撑索引已由建表语句中的 PRIMARY KEY 创建，跳过避免重复。
		if strings.Contains(name, "_pkey") {
			continue
		}
		definitions = append(definitions, definition)
	}
	return definitions, rows.Err()
}

// listPostgresColumns 查询数据导出使用的列名。
func listPostgresColumns(ctx context.Context, tx *sql.Tx, schema, table string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, "SELECT column_name FROM information_schema.columns WHERE table_schema = $1 AND table_name = $2 AND is_generated = 'NEVER' AND is_updatable = 'YES' ORDER BY ordinal_position", schema, table)
	if err != nil {
		return nil, fmt.Errorf("查询 PostgreSQL 表 %s 字段失败: %w", table, err)
	}
	defer rows.Close()
	columns := make([]string, 0)
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("读取 PostgreSQL 表 %s 字段失败: %w", table, err)
		}
		columns = append(columns, name)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历 PostgreSQL 表 %s 字段失败: %w", table, err)
	}
	return columns, nil
}

// formatPostgresValue 将数据库驱动返回值编码为 SQL 字面量。
func formatPostgresValue(value interface{}, dataType string) string {
	if value == nil {
		return "NULL"
	}
	switch item := value.(type) {
	case []byte:
		return "'\\x" + hex.EncodeToString(item) + "'::bytea"
	case string:
		return quotePostgresString(item)
	case time.Time:
		return "'" + item.Format("2006-01-02 15:04:05.999999999-07:00") + "'"
	case int64:
		return strconv.FormatInt(item, 10)
	case int32:
		return strconv.FormatInt(int64(item), 10)
	case int:
		return strconv.Itoa(item)
	case uint64:
		return strconv.FormatUint(item, 10)
	case uint32:
		return strconv.FormatUint(uint64(item), 10)
	case uint:
		return strconv.FormatUint(uint64(item), 10)
	case float32:
		return formatPostgresFloat(float64(item))
	case float64:
		return formatPostgresFloat(item)
	case bool:
		if item {
			return "true"
		}
		return "false"
	default:
		return quotePostgresString(fmt.Sprint(value))
	}
}

// formatPostgresFloat 将浮点数编码为 PostgreSQL 可解析的数字，特殊值使用字符串字面量。
func formatPostgresFloat(value float64) string {
	switch {
	case math.IsNaN(value):
		return "'NaN'"
	case math.IsInf(value, 1):
		return "'Infinity'"
	case math.IsInf(value, -1):
		return "'-Infinity'"
	default:
		return strconv.FormatFloat(value, 'g', -1, 64)
	}
}

// quotePostgresString 转义并包裹 PostgreSQL 标准模式字符串字面量。
func quotePostgresString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

// quotePostgresIdentifier 转义并包裹 PostgreSQL 标识符。
func quotePostgresIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

// qualifiedPostgresName 返回带双引号的模式限定表名。
func qualifiedPostgresName(schema, table string) string {
	return quotePostgresIdentifier(schema) + "." + quotePostgresIdentifier(table)
}

// postgresSequenceName 从 nextval 默认值中提取序列限定名。
func postgresSequenceName(columnDefault string) string {
	start := strings.Index(columnDefault, "nextval('")
	if start < 0 {
		return ""
	}
	rest := columnDefault[start+len("nextval('"):]
	end := strings.Index(rest, "'")
	if end < 0 {
		return ""
	}
	return rest[:end]
}
