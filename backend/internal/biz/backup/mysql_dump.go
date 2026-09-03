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

// MySQLDumpOptions 描述 Go MySQL 导出的内容范围。
type MySQLDumpOptions struct {
	// Database 是需要导出的数据库名称。
	Database string
	// Table 是可选的数据表名称，为空时导出数据库中的全部表。
	Table string
	// Where 是可选的数据筛选条件，仅允许由调用方使用受控值构造。
	Where string
	// IncludeSchema 表示是否导出表结构。
	IncludeSchema bool
	// IncludeData 表示是否导出表数据。
	IncludeData bool
	// IncludeRoutines 表示是否导出存储过程和函数。
	IncludeRoutines bool
	// IncludeEvents 表示是否导出事件。
	IncludeEvents bool
	// IncludeTriggers 表示是否导出触发器。
	IncludeTriggers bool
}

type mysqlDumpTable struct {
	name      string
	tableType string
}

type mysqlDumpColumn struct {
	name     string
	dataType string
}

// DumpMySQL 使用 Go 生成可由 MySQL 客户端执行的 SQL 导出文件。
func DumpMySQL(ctx context.Context, db *sql.DB, options MySQLDumpOptions, target io.Writer) error {
	if db == nil {
		return fmt.Errorf("MySQL 数据库连接为空")
	}
	if target == nil {
		return fmt.Errorf("SQL 导出目标为空")
	}
	if options.Database == "" {
		return fmt.Errorf("SQL 导出数据库名称为空")
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return fmt.Errorf("开启 MySQL 导出事务失败: %w", err)
	}
	defer tx.Rollback()
	if err = writeDumpHeader(target); err != nil {
		return err
	}
	var tables []mysqlDumpTable
	tables, err = listDumpTables(ctx, tx, options)
	if err != nil {
		return err
	}
	for _, table := range tables {
		if err = dumpMySQLTable(ctx, tx, options, table, target); err != nil {
			return err
		}
	}
	if options.IncludeTriggers {
		if err = dumpMySQLTriggers(ctx, tx, options.Database, target); err != nil {
			return err
		}
	}
	if options.IncludeRoutines {
		if err = dumpMySQLRoutines(ctx, tx, options.Database, target); err != nil {
			return err
		}
	}
	if options.IncludeEvents {
		if err = dumpMySQLEvents(ctx, tx, options.Database, target); err != nil {
			return err
		}
	}
	if err = writeDumpFooter(target); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("提交 MySQL 导出事务失败: %w", err)
	}
	return nil
}

// listDumpTables 查询需要导出的表和视图并保持稳定顺序。
func listDumpTables(ctx context.Context, tx *sql.Tx, options MySQLDumpOptions) ([]mysqlDumpTable, error) {
	query := "SELECT TABLE_NAME, TABLE_TYPE FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_TYPE IN ('BASE TABLE', 'VIEW')"
	args := []interface{}{options.Database}
	if options.Table != "" {
		query += " AND TABLE_NAME = ?"
		args = append(args, options.Table)
	}
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询 MySQL 导出表失败: %w", err)
	}
	defer rows.Close()
	tables := make([]mysqlDumpTable, 0)
	for rows.Next() {
		var table mysqlDumpTable
		if err = rows.Scan(&table.name, &table.tableType); err != nil {
			return nil, fmt.Errorf("读取 MySQL 导出表失败: %w", err)
		}
		tables = append(tables, table)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历 MySQL 导出表失败: %w", err)
	}
	sort.SliceStable(tables, func(i, j int) bool {
		if tables[i].tableType != tables[j].tableType {
			return tables[i].tableType == "BASE TABLE"
		}
		return tables[i].name < tables[j].name
	})
	if options.Table != "" && len(tables) == 0 {
		return nil, fmt.Errorf("MySQL 数据表不存在: %s", options.Table)
	}
	return tables, nil
}

// dumpMySQLTable 导出单张表或视图的结构和数据。
func dumpMySQLTable(ctx context.Context, tx *sql.Tx, options MySQLDumpOptions, table mysqlDumpTable, target io.Writer) error {
	if options.IncludeSchema {
		createSQL, err := showCreateSQL(ctx, tx, options.Database, table.name, table.tableType)
		if err != nil {
			return err
		}
		dropType := "TABLE"
		if table.tableType == "VIEW" {
			dropType = "VIEW"
		}
		if err = writeSQL(target, fmt.Sprintf("DROP %s IF EXISTS %s;\n", dropType, qualifiedName(options.Database, table.name))); err != nil {
			return fmt.Errorf("写入 %s 结构删除语句失败: %w", table.name, err)
		}
		if err = writeSQL(target, createSQL+";\n"); err != nil {
			return fmt.Errorf("写入 %s 结构创建语句失败: %w", table.name, err)
		}
	}
	if options.IncludeData && table.tableType == "BASE TABLE" {
		if err := dumpMySQLRows(ctx, tx, options, table.name, target); err != nil {
			return err
		}
	}
	return nil
}

// dumpMySQLRows 将表数据编码为可执行的 INSERT 语句。
func dumpMySQLRows(ctx context.Context, tx *sql.Tx, options MySQLDumpOptions, tableName string, target io.Writer) error {
	columns, err := listDumpColumns(ctx, tx, options.Database, tableName)
	if err != nil {
		return err
	}
	if len(columns) == 0 {
		return nil
	}
	columnNames := make([]string, 0, len(columns))
	for _, column := range columns {
		columnNames = append(columnNames, quoteIdentifier(column.name))
	}
	query := "SELECT " + strings.Join(columnNames, ", ") + " FROM " + qualifiedName(options.Database, tableName)
	if options.Where != "" {
		query += " WHERE " + options.Where
	}
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("查询 MySQL 表 %s 数据失败: %w", tableName, err)
	}
	defer rows.Close()
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return fmt.Errorf("读取 MySQL 表 %s 字段类型失败: %w", tableName, err)
	}
	values := make([]interface{}, len(columns))
	scanTargets := make([]interface{}, len(columns))
	for index := range values {
		scanTargets[index] = &values[index]
	}
	for rows.Next() {
		if err = rows.Scan(scanTargets...); err != nil {
			return fmt.Errorf("读取 MySQL 表 %s 数据失败: %w", tableName, err)
		}
		encodedValues := make([]string, len(values))
		for index, value := range values {
			encodedValues[index], err = formatMySQLValue(value, columnTypes[index].DatabaseTypeName())
			if err != nil {
				return fmt.Errorf("编码 MySQL 表 %s 字段 %s 失败: %w", tableName, columns[index].name, err)
			}
		}
		statement := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);\n", qualifiedName(options.Database, tableName), strings.Join(columnNames, ", "), strings.Join(encodedValues, ", "))
		if err = writeSQL(target, statement); err != nil {
			return fmt.Errorf("写入 MySQL 表 %s 数据失败: %w", tableName, err)
		}
	}
	if err = rows.Err(); err != nil {
		return fmt.Errorf("遍历 MySQL 表 %s 数据失败: %w", tableName, err)
	}
	return nil
}

// listDumpColumns 查询可写入的字段，排除生成列。
func listDumpColumns(ctx context.Context, tx *sql.Tx, database, table string) ([]mysqlDumpColumn, error) {
	rows, err := tx.QueryContext(ctx, "SELECT COLUMN_NAME, DATA_TYPE FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? AND IS_GENERATED = 'NEVER' ORDER BY ORDINAL_POSITION", database, table)
	if err != nil {
		return nil, fmt.Errorf("查询 MySQL 表 %s 字段失败: %w", table, err)
	}
	defer rows.Close()
	columns := make([]mysqlDumpColumn, 0)
	for rows.Next() {
		var column mysqlDumpColumn
		if err = rows.Scan(&column.name, &column.dataType); err != nil {
			return nil, fmt.Errorf("读取 MySQL 表 %s 字段失败: %w", table, err)
		}
		columns = append(columns, column)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历 MySQL 表 %s 字段失败: %w", table, err)
	}
	return columns, nil
}

// dumpMySQLTriggers 导出指定数据库中的触发器。
func dumpMySQLTriggers(ctx context.Context, tx *sql.Tx, database string, target io.Writer) error {
	rows, err := tx.QueryContext(ctx, "SELECT TRIGGER_NAME FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA = ? ORDER BY TRIGGER_NAME", database)
	if err != nil {
		return fmt.Errorf("查询 MySQL 触发器失败: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			return fmt.Errorf("读取 MySQL 触发器失败: %w", err)
		}
		var createSQL string
		createSQL, err = showCreateSQL(ctx, tx, database, name, "TRIGGER")
		if err != nil {
			return err
		}
		if err = writeDelimitedSQL(target, createSQL); err != nil {
			return fmt.Errorf("写入 MySQL 触发器 %s 失败: %w", name, err)
		}
	}
	if err = rows.Err(); err != nil {
		return fmt.Errorf("遍历 MySQL 触发器失败: %w", err)
	}
	return nil
}

// dumpMySQLRoutines 导出指定数据库中的存储过程和函数。
func dumpMySQLRoutines(ctx context.Context, tx *sql.Tx, database string, target io.Writer) error {
	rows, err := tx.QueryContext(ctx, "SELECT ROUTINE_TYPE, ROUTINE_NAME FROM information_schema.ROUTINES WHERE ROUTINE_SCHEMA = ? ORDER BY ROUTINE_TYPE, ROUTINE_NAME", database)
	if err != nil {
		return fmt.Errorf("查询 MySQL 存储程序失败: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var routineType string
		var name string
		if err = rows.Scan(&routineType, &name); err != nil {
			return fmt.Errorf("读取 MySQL 存储程序失败: %w", err)
		}
		var createSQL string
		createSQL, err = showCreateSQL(ctx, tx, database, name, routineType)
		if err != nil {
			return err
		}
		if err = writeDelimitedSQL(target, createSQL); err != nil {
			return fmt.Errorf("写入 MySQL 存储程序 %s %s 失败: %w", routineType, name, err)
		}
	}
	if err = rows.Err(); err != nil {
		return fmt.Errorf("遍历 MySQL 存储程序失败: %w", err)
	}
	return nil
}

// dumpMySQLEvents 导出指定数据库中的事件。
func dumpMySQLEvents(ctx context.Context, tx *sql.Tx, database string, target io.Writer) error {
	rows, err := tx.QueryContext(ctx, "SELECT EVENT_NAME FROM information_schema.EVENTS WHERE EVENT_SCHEMA = ? ORDER BY EVENT_NAME", database)
	if err != nil {
		return fmt.Errorf("查询 MySQL 事件失败: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			return fmt.Errorf("读取 MySQL 事件失败: %w", err)
		}
		var createSQL string
		createSQL, err = showCreateSQL(ctx, tx, database, name, "EVENT")
		if err != nil {
			return err
		}
		if err = writeDelimitedSQL(target, createSQL); err != nil {
			return fmt.Errorf("写入 MySQL 事件 %s 失败: %w", name, err)
		}
	}
	if err = rows.Err(); err != nil {
		return fmt.Errorf("遍历 MySQL 事件失败: %w", err)
	}
	return nil
}

// showCreateSQL 查询指定 MySQL 对象的建表或创建语句。
func showCreateSQL(ctx context.Context, tx *sql.Tx, database, name, objectType string) (string, error) {
	query := "SHOW CREATE TABLE " + qualifiedName(database, name)
	switch objectType {
	case "VIEW":
		query = "SHOW CREATE VIEW " + qualifiedName(database, name)
	case "TRIGGER":
		query = "SHOW CREATE TRIGGER " + qualifiedName(database, name)
	case "PROCEDURE", "FUNCTION":
		query = "SHOW CREATE " + objectType + " " + qualifiedName(database, name)
	case "EVENT":
		query = "SHOW CREATE EVENT " + qualifiedName(database, name)
	}
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return "", fmt.Errorf("查询 MySQL %s %s 定义失败: %w", objectType, name, err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return "", fmt.Errorf("读取 MySQL %s %s 定义字段失败: %w", objectType, name, err)
	}
	if !rows.Next() {
		if err = rows.Err(); err != nil {
			return "", fmt.Errorf("读取 MySQL %s %s 定义失败: %w", objectType, name, err)
		}
		return "", fmt.Errorf("MySQL %s %s 定义为空", objectType, name)
	}
	values := make([]interface{}, len(columns))
	targets := make([]interface{}, len(values))
	for index := range values {
		targets[index] = &values[index]
	}
	if err = rows.Scan(targets...); err != nil {
		return "", fmt.Errorf("读取 MySQL %s %s 定义失败: %w", objectType, name, err)
	}
	for index, column := range columns {
		if strings.Contains(strings.ToLower(column), "create") {
			return stringValue(values[index]), nil
		}
	}
	return "", fmt.Errorf("MySQL %s %s 定义字段缺少创建语句", objectType, name)
}

// formatMySQLValue 将数据库驱动返回值编码为 SQL 字面量。
func formatMySQLValue(value interface{}, dataType string) (string, error) {
	if value == nil {
		return "NULL", nil
	}
	switch item := value.(type) {
	case []byte:
		if isBinaryMySQLType(dataType) {
			return "0x" + hex.EncodeToString(item), nil
		}
		return quoteMySQLString(string(item)), nil
	case string:
		return quoteMySQLString(item), nil
	case time.Time:
		return quoteMySQLString(item.Format("2006-01-02 15:04:05.999999")), nil
	case int64:
		return strconv.FormatInt(item, 10), nil
	case uint64:
		return strconv.FormatUint(item, 10), nil
	case int32:
		return strconv.FormatInt(int64(item), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(item), 10), nil
	case int:
		return strconv.Itoa(item), nil
	case uint:
		return strconv.FormatUint(uint64(item), 10), nil
	case float32:
		return formatMySQLFloat(float64(item))
	case float64:
		return formatMySQLFloat(item)
	case bool:
		if item {
			return "1", nil
		}
		return "0", nil
	default:
		return quoteMySQLString(fmt.Sprint(value)), nil
	}
}

// formatMySQLFloat 将浮点数编码为有限的 SQL 数字。
func formatMySQLFloat(value float64) (string, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "", fmt.Errorf("浮点数值不可编码")
	}
	return strconv.FormatFloat(value, 'g', -1, 64), nil
}

// isBinaryMySQLType 判断字段是否应按二进制字面量导出。
func isBinaryMySQLType(dataType string) bool {
	switch strings.ToUpper(dataType) {
	case "BINARY", "VARBINARY", "TINYBLOB", "BLOB", "MEDIUMBLOB", "LONGBLOB", "GEOMETRY":
		return true
	default:
		return false
	}
}

// quoteMySQLString 转义并包裹 MySQL 字符串字面量。
func quoteMySQLString(value string) string {
	value = strings.NewReplacer("\\", "\\\\", "\x00", "\\0", "\n", "\\n", "\r", "\\r", "\b", "\\b", "\t", "\\t", "\x1a", "\\Z", "'", "\\'").Replace(value)
	return "'" + value + "'"
}

// qualifiedName 返回带反引号的数据表全名。
func qualifiedName(database, table string) string {
	return quoteIdentifier(database) + "." + quoteIdentifier(table)
}

// quoteIdentifier 转义 MySQL 标识符中的反引号。
func quoteIdentifier(value string) string {
	return "`" + strings.ReplaceAll(value, "`", "``") + "`"
}

// writeDumpHeader 写入导出文件头和会话设置。
func writeDumpHeader(target io.Writer) error {
	return writeSQL(target, "-- kratos-admin Go MySQL dump\nSET NAMES utf8mb4;\nSET FOREIGN_KEY_CHECKS=0;\n")
}

// writeDumpFooter 写入导出文件尾和会话恢复设置。
func writeDumpFooter(target io.Writer) error {
	return writeSQL(target, "SET FOREIGN_KEY_CHECKS=1;\n")
}

// writeDelimitedSQL 写入需要自定义分隔符的 MySQL 对象定义。
func writeDelimitedSQL(target io.Writer, statement string) error {
	return writeSQL(target, "DELIMITER ;;\n"+strings.TrimSpace(statement)+";;\nDELIMITER ;\n")
}

// writeSQL 将完整 SQL 文本写入目标流。
func writeSQL(target io.Writer, value string) error {
	written, err := io.WriteString(target, value)
	if err != nil {
		return err
	}
	if written != len(value) {
		return io.ErrShortWrite
	}
	return nil
}

// stringValue 将数据库驱动返回值转换为定义文本。
func stringValue(value interface{}) string {
	switch item := value.(type) {
	case []byte:
		return string(item)
	case string:
		return item
	default:
		return fmt.Sprint(item)
	}
}
