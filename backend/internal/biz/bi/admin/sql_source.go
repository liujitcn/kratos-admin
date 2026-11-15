package admin

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/query"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

// configuredDatabase 表示一个可执行 BI 查询的数据库连接。
type configuredDatabase struct {
	db     *sql.DB
	driver string
	close  func()
}

// openConfiguredSource 打开配置数据源；绑定宿主客户端时复用现有连接。
func openConfiguredSource(ctx context.Context, clients map[string]*gorm.Client, item *models.BiSource, password string) (*configuredDatabase, error) {
	var err error
	if item.Host == "" {
		client := clients[item.SourceName]
		if client == nil || client.DB == nil {
			return nil, errorsx.InvalidArgument(fmt.Sprintf("BI 数据源绑定的数据库客户端不存在: %s", item.SourceName))
		}
		var db *sql.DB
		db, err = client.DB.DB()
		if err != nil {
			return nil, errorsx.Internal("获取 BI 数据库连接失败").WithCause(err)
		}
		return &configuredDatabase{db: db, driver: client.Driver(), close: func() {}}, nil
	}
	if !supportedSourceDriver(item.Driver) {
		return nil, errorsx.InvalidArgument("数据库驱动不支持")
	}
	var driverName, dsn string
	driverName, dsn, err = sourceDSN(item, password)
	if err != nil {
		return nil, err
	}
	var db *sql.DB
	db, err = sql.Open(driverName, dsn)
	if err != nil {
		return nil, errorsx.InvalidArgument("打开 BI 数据源失败").WithCause(err)
	}
	closeDB := func() { _ = db.Close() }
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	err = db.PingContext(pingCtx)
	cancel()
	if err != nil {
		closeDB()
		return nil, errorsx.InvalidArgument("连接 BI 数据源失败").WithCause(err)
	}
	return &configuredDatabase{db: db, driver: item.Driver, close: closeDB}, nil
}

// sourceDSN 将独立数据源连接字段转换为驱动 DSN。
func sourceDSN(item *models.BiSource, password string) (string, string, error) {
	driverName := strings.ToLower(item.Driver)
	switch driverName {
	case "mysql", "doris", "tidb":
		if item.Host == "" || item.Port <= 0 || item.DatabaseName == "" || item.Username == "" {
			return "", "", errorsx.InvalidArgument("MySQL兼容数据源连接信息不完整")
		}
		// 零值 Config 不允许 mysql_native_password，而 Doris 等数据源默认使用该认证方式，需显式开启。
		dsn := (&mysql.Config{User: item.Username, Passwd: password, Net: "tcp", Addr: item.Host + ":" + strconv.FormatInt(int64(item.Port), 10), DBName: item.DatabaseName, AllowNativePasswords: true, Params: map[string]string{"charset": "utf8mb4", "parseTime": "true", "loc": "Local"}}).FormatDSN()
		return "mysql", dsn, nil
	case "postgres":
		if item.Host == "" || item.Port <= 0 || item.DatabaseName == "" || item.Username == "" {
			return "", "", errorsx.InvalidArgument("PostgreSQL数据源连接信息不完整")
		}
		sslMode := item.SslMode
		if sslMode == "" {
			sslMode = "disable"
		}
		connectionURL := &url.URL{Scheme: "postgres", Host: net.JoinHostPort(item.Host, strconv.FormatInt(int64(item.Port), 10)), Path: "/" + item.DatabaseName, User: url.UserPassword(item.Username, password)}
		params := connectionURL.Query()
		params.Set("sslmode", sslMode)
		connectionURL.RawQuery = params.Encode()
		return "postgres", connectionURL.String(), nil
	default:
		return "", "", errorsx.InvalidArgument("数据库驱动不支持")
	}
}

// Close 释放独立数据源连接，宿主复用连接不执行关闭。
func (d *configuredDatabase) Close() {
	if d != nil && d.close != nil {
		d.close()
	}
}

// discoverSourceTables 读取已连接数据库中可用于建模的表、字段及其注释。
func discoverSourceTables(ctx context.Context, database *configuredDatabase, item *models.BiSource) ([]*adminv1.BiSourceSchemaTable, error) {
	schemaName := item.SchemaName
	var err error
	if schemaName == "" {
		if strings.EqualFold(database.driver, "postgres") {
			err = database.db.QueryRowContext(ctx, "SELECT current_schema()").Scan(&schemaName) //nolint:forbidigo // 读取当前连接的默认 Schema，不接收用户 SQL。
			if err != nil {
				return nil, errorsx.Internal("读取当前数据库 Schema 失败").WithCause(err)
			}
		} else {
			schemaName = item.DatabaseName
			if schemaName == "" {
				err = database.db.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&schemaName) //nolint:forbidigo // 读取当前连接的默认数据库，不接收用户 SQL。
				if err != nil {
					return nil, errorsx.Internal("读取当前数据库名称失败").WithCause(err)
				}
			}
		}
	}
	if schemaName == "" {
		return nil, errorsx.InvalidArgument("无法确定数据库 Schema，请在连接配置中填写")
	}
	placeholder := query.Placeholder(database.driver, 1)
	tablesQuery := "SELECT table_schema, table_name, table_type, COALESCE(table_comment, '') FROM information_schema.tables WHERE table_schema = " + placeholder + " AND table_type IN ('BASE TABLE', 'VIEW') ORDER BY table_name LIMIT 500"
	if strings.EqualFold(database.driver, "postgres") {
		tablesQuery = "SELECT t.table_schema, t.table_name, t.table_type, COALESCE(pg_catalog.obj_description(c.oid, 'pg_class'), '') FROM information_schema.tables t JOIN pg_catalog.pg_namespace n ON n.nspname = t.table_schema JOIN pg_catalog.pg_class c ON c.relnamespace = n.oid AND c.relname = t.table_name WHERE t.table_schema = " + placeholder + " AND t.table_type IN ('BASE TABLE', 'VIEW') ORDER BY t.table_name LIMIT 500"
	}
	var tableRows *sql.Rows
	tableRows, err = database.db.QueryContext(ctx, tablesQuery, schemaName) //nolint:forbidigo // 仅查询固定数据库元数据，用户值使用参数绑定。
	if err != nil {
		return nil, errorsx.Internal("读取数据库表结构失败").WithCause(err)
	}
	tables := make([]*adminv1.BiSourceSchemaTable, 0)
	byName := make(map[string]*adminv1.BiSourceSchemaTable)
	for tableRows.Next() {
		var tableSchema, name, tableType, comment string
		if err = tableRows.Scan(&tableSchema, &name, &tableType, &comment); err != nil {
			_ = tableRows.Close()
			return nil, errorsx.Internal("读取数据库表结构失败").WithCause(err)
		}
		if !biSourceIdentifierPattern.MatchString(tableSchema) || !biSourceIdentifierPattern.MatchString(name) {
			continue
		}
		table := &adminv1.BiSourceSchemaTable{Key: sourceTableKey(tableSchema, name), SchemaName: tableSchema, Name: name, TableType: tableType, Comment: comment}
		tables = append(tables, table)
		byName[name] = table
	}
	if err = tableRows.Err(); err != nil {
		_ = tableRows.Close()
		return nil, errorsx.Internal("遍历数据库表结构失败").WithCause(err)
	}
	if err = tableRows.Close(); err != nil {
		return nil, errorsx.Internal("关闭数据库表结构查询失败").WithCause(err)
	}
	columnsQuery := "SELECT table_name, column_name, data_type, is_nullable, COALESCE(column_comment, '') FROM information_schema.columns WHERE table_schema = " + placeholder + " ORDER BY table_name, ordinal_position"
	if strings.EqualFold(database.driver, "postgres") {
		columnsQuery = "SELECT t.table_name, t.column_name, t.data_type, t.is_nullable, COALESCE(pg_catalog.col_description(c.oid, t.ordinal_position), '') FROM information_schema.columns t JOIN pg_catalog.pg_namespace n ON n.nspname = t.table_schema JOIN pg_catalog.pg_class c ON c.relnamespace = n.oid AND c.relname = t.table_name WHERE t.table_schema = " + placeholder + " ORDER BY t.table_name, t.ordinal_position"
	}
	var columnRows *sql.Rows
	columnRows, err = database.db.QueryContext(ctx, columnsQuery, schemaName) //nolint:forbidigo // 仅查询固定 information_schema 元数据，用户值使用参数绑定。
	if err != nil {
		return nil, errorsx.Internal("读取数据库字段结构失败").WithCause(err)
	}
	columnsByName := make(map[string]map[string]*adminv1.BiSourceSchemaColumn)
	for columnRows.Next() {
		var tableName, name, dataType, nullable, comment string
		if err = columnRows.Scan(&tableName, &name, &dataType, &nullable, &comment); err != nil {
			_ = columnRows.Close()
			return nil, errorsx.Internal("读取数据库字段结构失败").WithCause(err)
		}
		table := byName[tableName]
		if table == nil || !biSourceIdentifierPattern.MatchString(name) {
			continue
		}
		column := &adminv1.BiSourceSchemaColumn{Name: name, DataType: dataType, Nullable: strings.EqualFold(nullable, "YES"), Comment: comment}
		table.Columns = append(table.Columns, column)
		if columnsByName[tableName] == nil {
			columnsByName[tableName] = make(map[string]*adminv1.BiSourceSchemaColumn)
		}
		columnsByName[tableName][name] = column
	}
	if err = columnRows.Err(); err != nil {
		_ = columnRows.Close()
		return nil, errorsx.Internal("遍历数据库字段结构失败").WithCause(err)
	}
	if err = columnRows.Close(); err != nil {
		return nil, errorsx.Internal("关闭数据库字段查询失败").WithCause(err)
	}
	constraintsQuery := "SELECT tc.table_name, tc.constraint_name, tc.constraint_type, kcu.column_name FROM information_schema.table_constraints tc JOIN information_schema.key_column_usage kcu ON tc.constraint_schema = kcu.constraint_schema AND tc.constraint_name = kcu.constraint_name AND tc.table_name = kcu.table_name WHERE tc.table_schema = " + placeholder + " AND tc.constraint_type IN ('PRIMARY KEY', 'UNIQUE') ORDER BY tc.table_name, tc.constraint_name, kcu.ordinal_position"
	var constraintRows *sql.Rows
	constraintRows, err = database.db.QueryContext(ctx, constraintsQuery, schemaName) //nolint:forbidigo // 仅查询固定 information_schema 元数据，用户值使用参数绑定。
	if err != nil {
		return nil, errorsx.Internal("读取数据库唯一键失败").WithCause(err)
	}
	constraintColumns := make(map[string][]string)
	constraintTypes := make(map[string]string)
	for constraintRows.Next() {
		var tableName, constraintName, constraintType, columnName string
		if err = constraintRows.Scan(&tableName, &constraintName, &constraintType, &columnName); err != nil {
			_ = constraintRows.Close()
			return nil, errorsx.Internal("读取数据库唯一键失败").WithCause(err)
		}
		key := tableName + "." + constraintName
		constraintColumns[key] = append(constraintColumns[key], columnName)
		constraintTypes[key] = constraintType
	}
	if err = constraintRows.Err(); err != nil {
		_ = constraintRows.Close()
		return nil, errorsx.Internal("遍历数据库唯一键失败").WithCause(err)
	}
	if err = constraintRows.Close(); err != nil {
		return nil, errorsx.Internal("关闭数据库唯一键查询失败").WithCause(err)
	}
	for key, names := range constraintColumns {
		if len(names) != 1 {
			continue
		}
		tableName, _, _ := strings.Cut(key, ".")
		column := columnsByName[tableName][names[0]]
		if column == nil {
			continue
		}
		column.Unique = true
		column.PrimaryKey = strings.EqualFold(constraintTypes[key], "PRIMARY KEY")
	}
	return tables, nil
}

// sourceTableKey 返回数据库 Schema 与表名组成的稳定标识。
func sourceTableKey(schemaName, tableName string) string {
	return schemaName + "." + tableName
}

// executeConfiguredDefinition 执行由封闭编译器生成的参数化只读 SQL，结果保持十进制字符串与空值约定。
func executeConfiguredDefinition(ctx context.Context, database *configuredDatabase, execution *dto.CardExecution, args []any, limit int32) (bi.QueryResult, error) {
	queryText, columns := execution.SQL, execution.Columns
	var err error
	var rows *sql.Rows
	rows, err = database.db.QueryContext(ctx, queryText, args...) //nolint:forbidigo // 查询由白名单字段、固定聚合和参数占位符构造。
	if err != nil {
		return bi.QueryResult{}, errorsx.Internal("执行配置数据源查询失败").WithCause(err)
	}
	defer func() { _ = rows.Close() }()
	var names []string
	names, err = rows.Columns()
	if err != nil {
		return bi.QueryResult{}, errorsx.Internal("读取结果字段失败").WithCause(err)
	}
	if len(names) != len(columns) {
		return bi.QueryResult{}, errorsx.InvalidArgument("查询结果字段与指标定义不一致")
	}
	for index, name := range names {
		if name != columns[index].Code {
			return bi.QueryResult{}, errorsx.InvalidArgument("查询结果字段与指标定义不一致")
		}
	}

	var types []*sql.ColumnType
	types, err = rows.ColumnTypes()
	if err != nil {
		return bi.QueryResult{}, errorsx.Internal("读取结果类型失败").WithCause(err)
	}
	for index, column := range columns {
		actual := types[index].DatabaseTypeName()
		if actual != "" && sourceFieldType(actual) != column.Type {
			return bi.QueryResult{}, errorsx.InvalidArgument("查询结果类型与公共模型不一致")
		}
	}
	values := make([][]string, 0, limit)
	for rows.Next() {
		cells := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for index := range cells {
			pointers[index] = &cells[index]
		}
		if err = rows.Scan(pointers...); err != nil {
			return bi.QueryResult{}, errorsx.Internal("读取配置数据源结果失败").WithCause(err)
		}
		row := make([]string, 0, len(cells))
		for index, value := range cells {
			if columns[index].Type == bi.FieldTypeNumber && value != nil {
				if _, ok := new(big.Rat).SetString(formatConfiguredValue(value)); !ok {
					return bi.QueryResult{}, errorsx.InvalidArgument("查询结果不是有效数值")
				}
			}
			row = append(row, formatConfiguredValue(value))
		}
		values = append(values, row)
	}
	if err = rows.Err(); err != nil {
		return bi.QueryResult{}, errorsx.Internal("遍历配置数据源结果失败").WithCause(err)
	}
	truncated := int32(len(values)) > limit
	if truncated {
		values = values[:limit]
	}
	return bi.QueryResult{Columns: columns, Rows: values, Truncated: truncated}, nil
}

// formatConfiguredValue 将数据库扫描出的任意单元格值格式化为结果字符串，统一时间与二进制类型的展示形式。
func formatConfiguredValue(value any) string {
	switch item := value.(type) {
	case nil:
		return ""
	case []byte:
		return string(item)
	case time.Time:
		return item.Format("2006-01-02 15:04:05")
	default:
		return fmt.Sprint(item)
	}
}
