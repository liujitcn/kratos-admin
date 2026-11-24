package admin

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"math/big"
	"strings"
	"time"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/data/dbconn"
	"github.com/liujitcn/kratos-admin/backend/internal/data/dbmeta"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-kit/database/gorm"
	gormio "gorm.io/gorm"
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
			return nil, errorsx.WithMessageKey(errorsx.InvalidArgument(fmt.Sprintf("BI 数据源绑定的数据库客户端不存在: %s", item.SourceName)), "bi.source.error.client_missing", map[string]string{"Name": item.SourceName})
		}
		var db *sql.DB
		db, err = client.DB.DB()
		if err != nil {
			return nil, errorsx.Internal("获取 BI 数据库连接失败").WithCause(err)
		}
		return &configuredDatabase{db: db, driver: client.Driver(), close: func() {}}, nil
	}
	var db *sql.DB
	db, err = dbconn.Open(dbconn.Config{Driver: item.Driver, Host: item.Host, Port: item.Port, DatabaseName: item.DatabaseName, Username: item.Username, SSLMode: item.SslMode}, password)
	if err != nil {
		return nil, errorsx.InvalidArgument("打开 BI 数据源失败").WithCause(err)
	}
	closeDB := func() { _ = db.Close() }
	if err = dbconn.Ping(ctx, db); err != nil {
		closeDB()
		return nil, errorsx.InvalidArgument("连接 BI 数据源失败").WithCause(err)
	}
	return &configuredDatabase{db: db, driver: strings.ToLower(item.Driver), close: closeDB}, nil
}

// Close 释放独立数据源连接，宿主复用连接不执行关闭。
func (d *configuredDatabase) Close() {
	if d != nil && d.close != nil {
		d.close()
	}
}

// reportConnection 为报表借用单条连接，确保时间分组和筛选使用同一时区，归还前恢复宿主设置。
func (d *configuredDatabase) reportConnection(ctx context.Context, zone string) (*sql.Conn, func(), error) {
	read, set := "SELECT @@session.time_zone", "SET SESSION time_zone = ?"
	if d.driver == "postgres" {
		read, set = "SHOW TimeZone", "SELECT set_config('TimeZone', $1, false)"
	} else if zone == "UTC" {
		zone = "+00:00"
	}
	conn, err := d.db.Conn(ctx)
	if err != nil {
		return nil, nil, err
	}
	var previous string
	if err = conn.QueryRowContext(ctx, read).Scan(&previous); err != nil { //nolint:forbidigo // 读取当前会话设置以便归还连接时恢复。
		_ = conn.Close()
		return nil, nil, err
	}
	if previous == zone {
		return conn, func() { _ = conn.Close() }, nil
	}
	restore := func() {
		restoreCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
		defer cancel()
		if _, restoreErr := conn.ExecContext(restoreCtx, set, previous); restoreErr != nil { //nolint:forbidigo // 恢复固定会话设置，无法恢复时丢弃物理连接。
			_ = conn.Raw(func(any) error { return driver.ErrBadConn }) //nolint:forbidigo // database/sql 的 Raw 通过 ErrBadConn 丢弃连接，不执行 SQL。
		}
		_ = conn.Close()
	}
	if _, err = conn.ExecContext(ctx, set, zone); err != nil { //nolint:forbidigo // 只设置当前连接、时区值通过参数绑定。
		restore()
		return nil, nil, err
	}
	return conn, restore, nil
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
	var db *gormio.DB
	db, err = dbmeta.Bind(database.db, database.driver)
	if err != nil {
		return nil, errorsx.Internal("初始化数据库元数据查询失败").WithCause(err)
	}
	var physicalTables []dbmeta.Table
	physicalTables, err = dbmeta.ListTables(ctx, db, database.driver, dbmeta.TableScope{Schema: schemaName, IncludeViews: true, Limit: 500})
	if err != nil {
		return nil, errorsx.Internal("读取数据库表结构失败").WithCause(err)
	}
	tables := make([]*adminv1.BiSourceSchemaTable, 0, len(physicalTables))
	byName := make(map[string]*adminv1.BiSourceSchemaTable, len(physicalTables))
	tableNames := make([]string, 0, len(physicalTables))
	for _, physical := range physicalTables {
		if !biSourceIdentifierPattern.MatchString(physical.SchemaName) || !biSourceIdentifierPattern.MatchString(physical.TableName) {
			continue
		}
		table := &adminv1.BiSourceSchemaTable{Key: sourceTableKey(physical.SchemaName, physical.TableName), SchemaName: physical.SchemaName, Name: physical.TableName, TableType: physical.Type, Comment: physical.TableComment}
		tables = append(tables, table)
		byName[physical.TableName] = table
		tableNames = append(tableNames, physical.TableName)
	}
	var physicalColumns []dbmeta.Column
	physicalColumns, err = dbmeta.ListColumns(ctx, db, database.driver, schemaName, tableNames, false)
	if err != nil {
		return nil, errorsx.Internal("读取数据库字段结构失败").WithCause(err)
	}
	columnsByName := make(map[string]map[string]*adminv1.BiSourceSchemaColumn)
	for _, physical := range physicalColumns {
		if !biSourceIdentifierPattern.MatchString(physical.Name) {
			continue
		}
		table := byName[physical.TableName]
		column := &adminv1.BiSourceSchemaColumn{Name: physical.Name, DataType: physical.DataType, Nullable: strings.EqualFold(physical.IsNullable, "YES"), Comment: physical.Comment}
		table.Columns = append(table.Columns, column)
		if columnsByName[physical.TableName] == nil {
			columnsByName[physical.TableName] = make(map[string]*adminv1.BiSourceSchemaColumn)
		}
		columnsByName[physical.TableName][physical.Name] = column
	}
	var constraints []dbmeta.ConstraintColumn
	constraints, err = dbmeta.ListConstraintColumns(ctx, db, schemaName, tableNames)
	if err != nil {
		return nil, errorsx.Internal("读取数据库唯一键失败").WithCause(err)
	}
	type constraintKey struct{ table, name, kind string }
	grouped := make(map[constraintKey][]string)
	for _, column := range constraints {
		key := constraintKey{column.TableName, column.ConstraintName, column.ConstraintType}
		grouped[key] = append(grouped[key], column.ColumnName)
	}
	for key, names := range grouped {
		if len(names) != 1 {
			continue
		}
		if column := columnsByName[key.table][names[0]]; column != nil {
			column.Unique = true
			column.PrimaryKey = column.PrimaryKey || key.kind == "PRIMARY KEY"
		}
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
	queryRows := database.db.QueryContext
	zone := execution.ReportTimezone
	if execution.TimeRangeInstant && (database.driver == "mysql" || database.driver == "tidb") {
		zone = "UTC"
	}
	if zone != "" {
		var conn *sql.Conn
		var restore func()
		conn, restore, err = database.reportConnection(ctx, zone)
		if err != nil {
			return bi.QueryResult{}, errorsx.InvalidArgument("BI 数据库不支持此报表时区").WithCause(err)
		}
		defer restore()
		queryRows = conn.QueryContext
	}
	var rows *sql.Rows
	rows, err = queryRows(ctx, queryText, args...) // 查询由白名单字段、固定聚合和参数占位符构造。
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
		// 已保存执行定义可能仍把纯时刻的原始结果标为日期。
		if actual != "" && sourceFieldType(actual) != column.Type && !(column.Type == bi.FieldTypeDate && isSourceClockType(strings.ToLower(actual))) {
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
			if date, ok := value.(time.Time); ok && types[index].DatabaseTypeName() == "DATE" {
				row = append(row, date.Format(time.DateOnly))
				continue
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
