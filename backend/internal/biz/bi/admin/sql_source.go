package admin

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/dto"
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
	if item.Host == "" {
		client := clients[item.SourceName]
		if client == nil || client.DB == nil {
			return nil, errorsx.InvalidArgument(fmt.Sprintf("BI 数据源绑定的数据库客户端不存在: %s", item.SourceName))
		}
		db, err := client.DB.DB()
		if err != nil {
			return nil, errorsx.Internal("获取 BI 数据库连接失败").WithCause(err)
		}
		return &configuredDatabase{db: db, driver: client.Driver(), close: func() {}}, nil
	}
	if !supportedSourceDriver(item.Driver) {
		return nil, errorsx.InvalidArgument("数据库驱动不支持")
	}
	driverName, dsn, err := sourceDSN(item, password)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open(driverName, dsn)
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
	placeholder := sourcePlaceholder(database.driver, 1)
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

// executeConfiguredSource 执行配置数据源的多表语义聚合查询。
func executeConfiguredSource(ctx context.Context, database *configuredDatabase, item *models.BiSource, schema dto.ConfiguredSourceSchema, scope bi.Scope, spec bi.QuerySpec) (bi.QueryResult, error) {
	queryText, args, columns, err := buildConfiguredQuery(database.driver, item, schema, scope, spec)
	if err != nil {
		return bi.QueryResult{}, err
	}
	var rows *sql.Rows
	rows, err = database.db.QueryContext(ctx, queryText, args...) //nolint:forbidigo // 查询由白名单字段、固定聚合和参数占位符构造。
	if err != nil {
		return bi.QueryResult{}, errorsx.Internal("执行配置数据源查询失败").WithCause(err)
	}
	defer func() { _ = rows.Close() }()
	values := make([][]string, 0, item.MaxRows)
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
		for _, value := range cells {
			row = append(row, formatConfiguredValue(value))
		}
		values = append(values, row)
	}
	if err = rows.Err(); err != nil {
		return bi.QueryResult{}, errorsx.Internal("遍历配置数据源结果失败").WithCause(err)
	}
	limit := configuredSourceLimit(item.MaxRows, spec.Limit)
	truncated := int32(len(values)) > limit
	if truncated {
		values = values[:limit]
	}
	return bi.QueryResult{Columns: columns, Rows: values, Truncated: truncated}, nil
}

// buildConfiguredQuery 将语义查询编译为经过白名单校验的只读 SQL。
func buildConfiguredQuery(driver string, item *models.BiSource, schema dto.ConfiguredSourceSchema, scope bi.Scope, spec bi.QuerySpec) (string, []any, []bi.ResultColumn, error) {
	// 当前模型只声明租户隔离列，不能将项目授权扩大为整个租户。
	if scope.ProjectID != 0 {
		return "", nil, nil, errorsx.PermissionDenied("可配置数据源暂不支持项目范围查询")
	}
	tables := make(map[string]dto.ConfiguredSourceTable, len(schema.Tables))
	for _, table := range schema.Tables {
		tables[table.Key] = table
	}
	baseTable, ok := tables[schema.BaseTable]
	if !ok || len(tables) != len(schema.Tables) {
		return "", nil, nil, errorsx.InvalidArgument("BI 数据源表模型无效")
	}
	tableAliases := map[string]string{schema.BaseTable: "t0"}
	joinParts := make([]string, 0, len(schema.Relations))
	args := make([]any, 0, len(schema.Relations)+len(spec.Filters)*2+3)
	remaining := append([]dto.ConfiguredSourceRelation(nil), schema.Relations...)
	for len(remaining) > 0 {
		unresolved := make([]dto.ConfiguredSourceRelation, 0, len(remaining))
		for _, relation := range remaining {
			leftAlias, leftExists := tableAliases[relation.LeftTable]
			rightTable, rightExists := tables[relation.RightTable]
			if !leftExists {
				unresolved = append(unresolved, relation)
				continue
			}
			if !rightExists || tableAliases[relation.RightTable] != "" || (relation.JoinType != "inner" && relation.JoinType != "left") {
				return "", nil, nil, errorsx.InvalidArgument("BI 数据源表关系无效")
			}
			rightAlias := "t" + strconv.Itoa(len(tableAliases))
			tableAliases[relation.RightTable] = rightAlias
			on := quoteSourceIdentifier(driver, leftAlias) + "." + quoteSourceIdentifier(driver, relation.LeftColumn) + " = " + quoteSourceIdentifier(driver, rightAlias) + "." + quoteSourceIdentifier(driver, relation.RightColumn)
			if rightTable.TenantColumn != "" {
				on += " AND " + quoteSourceIdentifier(driver, rightAlias) + "." + quoteSourceIdentifier(driver, rightTable.TenantColumn) + " = " + sourcePlaceholder(driver, len(args)+1)
				args = append(args, scope.TenantID)
			}
			joinType := "INNER JOIN"
			if relation.JoinType == "left" {
				joinType = "LEFT JOIN"
			}
			joinParts = append(joinParts, joinType+" "+quoteSourceTable(driver, rightTable.SchemaName, rightTable.Name)+" AS "+quoteSourceIdentifier(driver, rightAlias)+" ON "+on)
		}
		if len(unresolved) == len(remaining) {
			return "", nil, nil, errorsx.InvalidArgument("BI 数据源表关系未从事实表连通")
		}
		remaining = unresolved
	}
	if len(tableAliases) != len(tables) {
		return "", nil, nil, errorsx.InvalidArgument("BI 数据源表关系未覆盖所有选中表")
	}
	dimensions := make(map[string]dto.ConfiguredSourceDimension, len(schema.Dimensions))
	for _, dimension := range schema.Dimensions {
		dimensions[dimension.Code] = dimension
	}
	measures := make(map[string]dto.ConfiguredSourceMeasure, len(schema.Measures))
	for _, measure := range schema.Measures {
		measures[measure.Code] = measure
	}
	selectParts := make([]string, 0, len(spec.Dimensions)+len(spec.Measures))
	groupParts := make([]string, 0, len(spec.Dimensions))
	resultColumns := make([]bi.ResultColumn, 0, len(spec.Dimensions)+len(spec.Measures))
	for _, code := range spec.Dimensions {
		dimension, ok := dimensions[code]
		alias := tableAliases[dimension.Table]
		if !ok || alias == "" {
			return "", nil, nil, errorsx.InvalidArgument("BI 维度不存在: " + code)
		}
		column := quoteSourceIdentifier(driver, alias) + "." + quoteSourceIdentifier(driver, dimension.Column)
		selectParts = append(selectParts, column+" AS "+quoteSourceIdentifier(driver, dimension.Code))
		groupParts = append(groupParts, column)
		resultColumns = append(resultColumns, bi.ResultColumn{Code: dimension.Code, Name: dimension.Name, Type: dimension.Type})
	}
	for _, measureSpec := range spec.Measures {
		measure, ok := measures[measureSpec.Code]
		if !ok || measure.Table != schema.BaseTable || !slices.Contains(measure.Aggregations, strings.ToLower(measureSpec.Aggregation)) {
			return "", nil, nil, errorsx.InvalidArgument("BI 指标或聚合函数不可用: " + measureSpec.Code)
		}
		aggregation := strings.ToUpper(measureSpec.Aggregation)
		selectParts = append(selectParts, aggregation+"("+quoteSourceIdentifier(driver, "t0")+"."+quoteSourceIdentifier(driver, measure.Column)+") AS "+quoteSourceIdentifier(driver, measure.Code))
		resultColumns = append(resultColumns, bi.ResultColumn{Code: measure.Code, Name: measure.Name, Type: measure.Type})
	}
	if len(selectParts) == 0 {
		return "", nil, nil, errorsx.InvalidArgument("BI 查询至少需要一个维度或指标")
	}
	whereParts := make([]string, 0, len(spec.Filters)+1)
	if baseTable.TenantColumn != "" {
		whereParts = append(whereParts, quoteSourceIdentifier(driver, "t0")+"."+quoteSourceIdentifier(driver, baseTable.TenantColumn)+" = "+sourcePlaceholder(driver, len(args)+1))
		args = append(args, scope.TenantID)
	}
	if spec.TimeRange != nil {
		dimension, ok := dimensions[spec.TimeRange.Field]
		if !ok || !dimension.Filterable || spec.TimeRange.Start == "" || spec.TimeRange.End == "" {
			return "", nil, nil, errorsx.InvalidArgument("BI 时间范围不可用: " + spec.TimeRange.Field)
		}
		alias := tableAliases[dimension.Table]
		if alias == "" {
			return "", nil, nil, errorsx.InvalidArgument("BI 时间维度所属表无效")
		}
		column := quoteSourceIdentifier(driver, alias) + "." + quoteSourceIdentifier(driver, dimension.Column)
		whereParts = append(whereParts, column+" BETWEEN "+sourcePlaceholder(driver, len(args)+1)+" AND "+sourcePlaceholder(driver, len(args)+2))
		args = append(args, spec.TimeRange.Start, spec.TimeRange.End)
	}
	for _, filter := range spec.Filters {
		dimension, ok := dimensions[filter.Field]
		if !ok || !dimension.Filterable || len(filter.Values) == 0 {
			return "", nil, nil, errorsx.InvalidArgument("BI 过滤字段不可用: " + filter.Field)
		}
		alias := tableAliases[dimension.Table]
		if alias == "" {
			return "", nil, nil, errorsx.InvalidArgument("BI 过滤字段所属表无效")
		}
		column := quoteSourceIdentifier(driver, alias) + "." + quoteSourceIdentifier(driver, dimension.Column)
		switch filter.Operator {
		case "eq":
			whereParts = append(whereParts, column+" = "+sourcePlaceholder(driver, len(args)+1))
			args = append(args, filter.Values[0])
		case "in":
			placeholders := make([]string, 0, len(filter.Values))
			for _, value := range filter.Values {
				placeholders = append(placeholders, sourcePlaceholder(driver, len(args)+1))
				args = append(args, value)
			}
			whereParts = append(whereParts, column+" IN ("+strings.Join(placeholders, ",")+")")
		case "between":
			if len(filter.Values) < 2 {
				return "", nil, nil, errorsx.InvalidArgument("between过滤条件需要两个值")
			}
			whereParts = append(whereParts, column+" BETWEEN "+sourcePlaceholder(driver, len(args)+1)+" AND "+sourcePlaceholder(driver, len(args)+2))
			args = append(args, filter.Values[0], filter.Values[1])
		case "contains":
			whereParts = append(whereParts, column+" LIKE "+sourcePlaceholder(driver, len(args)+1))
			args = append(args, "%"+filter.Values[0]+"%")
		default:
			return "", nil, nil, errorsx.InvalidArgument("BI 过滤操作符不支持: " + filter.Operator)
		}
	}
	limit := configuredSourceLimit(item.MaxRows, spec.Limit)
	queryText := "SELECT " + strings.Join(selectParts, ", ") + " FROM " + quoteSourceTable(driver, baseTable.SchemaName, baseTable.Name) + " AS " + quoteSourceIdentifier(driver, "t0")
	if len(joinParts) > 0 {
		queryText += " " + strings.Join(joinParts, " ")
	}
	if len(whereParts) > 0 {
		queryText += " WHERE " + strings.Join(whereParts, " AND ")
	}
	if len(groupParts) > 0 {
		queryText += " GROUP BY " + strings.Join(groupParts, ", ")
	}
	if len(spec.Orders) > 0 {
		selectedDimensions := make(map[string]string, len(spec.Dimensions))
		for _, code := range spec.Dimensions {
			dimension, ok := dimensions[code]
			if ok && tableAliases[dimension.Table] != "" {
				selectedDimensions[code] = quoteSourceIdentifier(driver, tableAliases[dimension.Table]) + "." + quoteSourceIdentifier(driver, dimension.Column)
			}
		}
		selectedMeasures := make(map[string]struct{}, len(spec.Measures))
		for _, measure := range spec.Measures {
			selectedMeasures[measure.Code] = struct{}{}
		}
		orderParts := make([]string, 0, len(spec.Orders))
		for _, order := range spec.Orders {
			if order.Direction != "asc" && order.Direction != "desc" {
				return "", nil, nil, errorsx.InvalidArgument("BI 排序方向不支持: " + order.Direction)
			}
			column, ok := selectedDimensions[order.Field]
			if !ok {
				if _, measureSelected := selectedMeasures[order.Field]; !measureSelected {
					return "", nil, nil, errorsx.InvalidArgument("BI 排序字段必须是已选择的维度或指标: " + order.Field)
				}
				column = quoteSourceIdentifier(driver, order.Field)
			}
			orderParts = append(orderParts, column+" "+strings.ToUpper(order.Direction))
		}
		queryText += " ORDER BY " + strings.Join(orderParts, ", ")
	}
	queryText += " LIMIT " + strconv.FormatInt(int64(limit)+1, 10)
	return queryText, args, resultColumns, nil
}

func quoteSourceTable(driver, schemaName, tableName string) string {
	if schemaName == "" {
		return quoteSourceIdentifier(driver, tableName)
	}
	return quoteSourceIdentifier(driver, schemaName) + "." + quoteSourceIdentifier(driver, tableName)
}

func quoteSourceIdentifier(driver, value string) string {
	if strings.EqualFold(driver, "postgres") {
		return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
	}
	return "`" + strings.ReplaceAll(value, "`", "``") + "`"
}

func sourcePlaceholder(driver string, index int) string {
	if strings.EqualFold(driver, "postgres") {
		return "$" + strconv.Itoa(index)
	}
	return "?"
}

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

// configuredSourceLimit 统一查询上限，额外读取一行仅用于判断是否截断。
func configuredSourceLimit(maxRows, requested int32) int32 {
	if requested > 0 && requested < maxRows {
		return requested
	}
	return maxRows
}
