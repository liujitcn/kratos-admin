package query

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
	"github.com/liujitcn/kratos-core/errorsx"
)

// Compile 冻结参数化 SQL、受控参数来源和公共指标修订。
func Compile(driver string, maxRows int32, model dto.ConfiguredSourceSchema, spec bi.QuerySpec) (*dto.CardExecution, error) {
	_, err := ValidateSpec(model.Descriptor(spec.SourceCode, "", "", 0, maxRows), spec)
	if err != nil {
		return nil, err
	}
	var sql string
	var parameters []dto.ExecutionParameter
	var columns []bi.ResultColumn
	sql, parameters, columns, err = compileSQL(driver, maxRows, model, spec)
	if err != nil {
		return nil, err
	}
	execution := &dto.CardExecution{Version: 1, Limit: configuredSourceLimit(maxRows, spec.Limit), Kind: "configured_sql", Driver: driver, ModelRevision: model.ModelRevision, SQL: sql, Parameters: parameters, Columns: columns, QueryHash: SpecHash(spec)}
	for _, column := range columns {
		if column.Role == "measure" {
			execution.Metrics = append(execution.Metrics, dto.MetricRevision{Code: column.Code, Revision: column.Revision})
		}
	}
	if err = Seal(execution); err != nil {
		return nil, err
	}
	return execution, nil
}

// AdapterExecution 冻结代码注册源的描述版本和有效上限，查询规格保持不变。
func AdapterExecution(descriptor bi.SourceDescriptor, spec bi.QuerySpec) (*dto.CardExecution, error) {
	validated, err := ValidateSpec(descriptor, spec)
	if err != nil {
		return nil, err
	}
	execution := &dto.CardExecution{Version: 1, Limit: configuredSourceLimit(descriptor.MaxRows, validated.Limit), Kind: "adapter", AdapterVersion: descriptor.Version, QueryHash: SpecHash(spec)}
	for _, code := range spec.Dimensions {
		for _, field := range descriptor.Dimensions {
			if field.Code == code {
				execution.Columns = append(execution.Columns, bi.ResultColumn{Code: code, Name: field.Name, Type: field.Type, Role: "dimension"})
			}
		}
	}
	for _, selected := range spec.Measures {
		found := false
		for _, metric := range descriptor.Measures {
			if metric.Code == selected.Code {
				found = true
				execution.Columns = append(execution.Columns, bi.ResultColumn{Code: metric.Code, Name: metric.Name, Type: metric.Type, Role: "measure", Unit: metric.Unit, Revision: metric.Revision})
				execution.Metrics = append(execution.Metrics, dto.MetricRevision{Code: metric.Code, Revision: metric.Revision})
			}
		}
		if !found {
			return nil, errorsx.InvalidArgument("Adapter 指标不存在")
		}
	}
	err = Seal(execution)
	return execution, err
}

// Seal 计算不包含身份、执行时间及自身哈希的执行摘要。
func Seal(execution *dto.CardExecution) error {
	execution.Hash = ""
	payload, err := json.Marshal(execution)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(payload)
	execution.Hash = hex.EncodeToString(digest[:])
	return nil
}

// SpecHash 对完整语义配置生成稳定摘要，空集合统一序列化。
func SpecHash(spec bi.QuerySpec) string {
	if spec.Dimensions == nil {
		spec.Dimensions = []string{}
	}
	if spec.Measures == nil {
		spec.Measures = []bi.MeasureSpec{}
	}
	if spec.Filters == nil {
		spec.Filters = []bi.FilterSpec{}
	}
	if spec.Orders == nil {
		spec.Orders = []bi.OrderSpec{}
	}
	spec.Filters = append([]bi.FilterSpec(nil), spec.Filters...)
	for index := range spec.Filters {
		if spec.Filters[index].Values == nil {
			spec.Filters[index].Values = []string{}
		}
	}
	payload, err := json.Marshal(spec)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}

// TimeField 选择卡片已配置的唯一日期轴，不为非日期卡片引入筛选。
func TimeField(descriptor bi.SourceDescriptor, spec bi.QuerySpec) string {
	for _, dimension := range descriptor.Dimensions {
		if dimension.Type != bi.FieldTypeDate || !dimension.Filterable {
			continue
		}
		if spec.TimeRange != nil && spec.TimeRange.Field == dimension.Code {
			return dimension.Code
		}
	}
	if spec.TimeRange != nil {
		return ""
	}
	for _, code := range spec.Dimensions {
		for _, dimension := range descriptor.Dimensions {
			if dimension.Code == code && dimension.Type == bi.FieldTypeDate && dimension.Filterable {
				return code
			}
		}
	}
	return ""
}

// compileSQL 根据受控模型生成 SQL 与参数来源，不解析执行身份。
func compileSQL(driver string, maxRows int32, schema dto.ConfiguredSourceSchema, spec bi.QuerySpec) (string, []dto.ExecutionParameter, []bi.ResultColumn, error) {
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
	args := make([]dto.ExecutionParameter, 0, len(schema.Relations)+len(spec.Filters)*2+3)
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
				on += " AND " + quoteSourceIdentifier(driver, rightAlias) + "." + quoteSourceIdentifier(driver, rightTable.TenantColumn) + " = " + Placeholder(driver, len(args)+1)
				args = append(args, dto.ExecutionParameter{Position: len(args) + 1, Kind: "scope", Field: "tenant_id"})
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
		resultColumns = append(resultColumns, bi.ResultColumn{Code: dimension.Code, Name: dimension.Name, Type: dimension.Type, Role: "dimension"})
	}
	baseFields := baseTable.Fields
	for _, measureSpec := range spec.Measures {
		measure, exists := measures[measureSpec.Code]
		if !exists || measure.Type != bi.FieldTypeNumber || measure.Revision <= 0 {
			return "", nil, nil, errorsx.InvalidArgument("BI 公共指标不可用: " + measureSpec.Code)
		}
		expression, err := Parse(measure.Expression, baseFields)
		if err != nil {
			return "", nil, nil, errorsx.InvalidArgument("BI 公共指标公式无效: " + measureSpec.Code).WithCause(err)
		}
		selectParts = append(selectParts, expression.SQL(func(name string) string {
			return quoteSourceIdentifier(driver, "t0") + "." + quoteSourceIdentifier(driver, name)
		})+" AS "+quoteSourceIdentifier(driver, measure.Code))
		resultColumns = append(resultColumns, bi.ResultColumn{Code: measure.Code, Name: measure.Name, Type: measure.Type, Role: "measure", Unit: measure.Unit, Revision: measure.Revision})
	}
	if len(selectParts) == 0 {
		return "", nil, nil, errorsx.InvalidArgument("BI 查询至少需要一个维度或指标")
	}
	whereParts := make([]string, 0, len(spec.Filters)+1)
	if baseTable.TenantColumn != "" {
		whereParts = append(whereParts, quoteSourceIdentifier(driver, "t0")+"."+quoteSourceIdentifier(driver, baseTable.TenantColumn)+" = "+Placeholder(driver, len(args)+1))
		args = append(args, dto.ExecutionParameter{Position: len(args) + 1, Kind: "scope", Field: "tenant_id"})
	}
	timeField := TimeField(schema.Descriptor("", "", "", 0, maxRows), spec)
	if timeField != "" {
		dimension := dimensions[timeField]
		alias := tableAliases[dimension.Table]
		if alias == "" {
			return "", nil, nil, errorsx.InvalidArgument("BI 时间维度所属表无效")
		}
		column := quoteSourceIdentifier(driver, alias) + "." + quoteSourceIdentifier(driver, dimension.Column)
		castType := "DATETIME"
		if strings.EqualFold(driver, "postgres") {
			castType = "TIMESTAMP"
		}
		whereParts = append(whereParts, "("+Placeholder(driver, len(args)+1)+" = 0 OR "+column+" BETWEEN CAST("+Placeholder(driver, len(args)+2)+" AS "+castType+") AND CAST("+Placeholder(driver, len(args)+3)+" AS "+castType+"))")
		for _, kind := range []string{"time_enabled", "time_start", "time_end"} {
			args = append(args, dto.ExecutionParameter{Position: len(args) + 1, Kind: kind, Field: timeField})
		}
	}
	for filterIndex, filter := range spec.Filters {
		dimension, ok := dimensions[filter.Field]
		if !ok || !dimension.Filterable || len(filter.Values) == 0 {
			return "", nil, nil, errorsx.InvalidArgument("BI 过滤字段不可用: " + filter.Field)
		}
		alias := tableAliases[dimension.Table]
		if alias == "" {
			return "", nil, nil, errorsx.InvalidArgument("BI 过滤字段所属表无效")
		}
		column := quoteSourceIdentifier(driver, alias) + "." + quoteSourceIdentifier(driver, dimension.Column)
		previousWhere := len(whereParts)
		overridePlaceholder := ""
		if filter.Field == timeField {
			overridePlaceholder = Placeholder(driver, len(args)+1)
			args = append(args, dto.ExecutionParameter{Position: len(args) + 1, Kind: "time_override", Field: timeField})
		}
		switch filter.Operator {
		case "eq":
			whereParts = append(whereParts, column+" = "+Placeholder(driver, len(args)+1))
			args = append(args, dto.ExecutionParameter{Position: len(args) + 1, Kind: "filter", FilterIndex: filterIndex})
		case "in":
			placeholders := make([]string, 0, len(filter.Values))
			for valueIndex := range filter.Values {
				placeholders = append(placeholders, Placeholder(driver, len(args)+1))
				args = append(args, dto.ExecutionParameter{Position: len(args) + 1, Kind: "filter", FilterIndex: filterIndex, ValueIndex: valueIndex})
			}
			whereParts = append(whereParts, column+" IN ("+strings.Join(placeholders, ",")+")")
		case "between":
			if len(filter.Values) < 2 {
				return "", nil, nil, errorsx.InvalidArgument("between过滤条件需要两个值")
			}
			whereParts = append(whereParts, column+" BETWEEN "+Placeholder(driver, len(args)+1)+" AND "+Placeholder(driver, len(args)+2))
			args = append(args, dto.ExecutionParameter{Position: len(args) + 1, Kind: "filter", FilterIndex: filterIndex}, dto.ExecutionParameter{Position: len(args) + 2, Kind: "filter", FilterIndex: filterIndex, ValueIndex: 1})
		case "contains":
			whereParts = append(whereParts, column+" LIKE "+Placeholder(driver, len(args)+1))
			args = append(args, dto.ExecutionParameter{Position: len(args) + 1, Kind: "filter", FilterIndex: filterIndex, Contains: true})
		default:
			return "", nil, nil, errorsx.InvalidArgument("BI 过滤操作符不支持: " + filter.Operator)
		}
		if overridePlaceholder != "" && len(whereParts) > previousWhere {
			whereParts[len(whereParts)-1] = "(" + overridePlaceholder + " = 1 OR " + whereParts[len(whereParts)-1] + ")"
		}
	}
	limit := configuredSourceLimit(maxRows, spec.Limit)
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

// quoteSourceTable 按驱动转义并拼接数据源表名，配置了 Schema 时追加限定前缀。
func quoteSourceTable(driver, schemaName, tableName string) string {
	if schemaName == "" {
		return quoteSourceIdentifier(driver, tableName)
	}
	return quoteSourceIdentifier(driver, schemaName) + "." + quoteSourceIdentifier(driver, tableName)
}

// QuoteSourceTable 按驱动转义并拼接数据源表名。
func QuoteSourceTable(driver, schemaName, tableName string) string {
	return quoteSourceTable(driver, schemaName, tableName)
}

// quoteSourceIdentifier 按驱动转义标识符并加引号，PostgreSQL 使用双引号、MySQL 系使用反引号，
// 是拼接数据源 SQL 时防止标识符注入与关键字冲突的唯一出口。
func quoteSourceIdentifier(driver, value string) string {
	if strings.EqualFold(driver, "postgres") {
		return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
	}
	return "`" + strings.ReplaceAll(value, "`", "``") + "`"
}

// QuoteSourceIdentifier 按驱动转义数据源标识符。
func QuoteSourceIdentifier(driver, value string) string {
	return quoteSourceIdentifier(driver, value)
}

// Placeholder 生成参数占位符：PostgreSQL 使用 $N 序号占位，MySQL 系使用问号占位。
func Placeholder(driver string, index int) string {
	if strings.EqualFold(driver, "postgres") {
		return "$" + strconv.Itoa(index)
	}
	return "?"
}

// configuredSourceLimit 计算有效执行上限，不改变用于身份校验的原查询规格。
func configuredSourceLimit(maxRows, requested int32) int32 {
	if requested <= 0 {
		requested = 100
	}
	if maxRows <= 0 {
		maxRows = 1000
	}
	return min(requested, maxRows, 1000)
}
