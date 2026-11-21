package query

import (
	"strconv"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
	"github.com/liujitcn/kratos-core/errorsx"
)

// compiledGrouping 保存已验证分组的 SQL 表达式及其组织映射连接。
type compiledGrouping struct {
	grain      string
	table      string
	expression string
	keys       string
	join       func(string) string
}

// compileGrouping 将日期粒度和父子层级编译为受控表达式，层级 CTE 的参数先于主查询参数。
func compileGrouping(driver string, schema dto.ConfiguredSourceSchema, tables map[string]dto.ConfiguredSourceTable, spec bi.QuerySpec, args *[]dto.ExecutionParameter) ([]string, map[string]compiledGrouping, error) {
	grouping := make(map[string]compiledGrouping, len(spec.GroupingOptions))
	var ctes []string
	for _, option := range spec.GroupingOptions {
		if option.TimeGrain != "" {
			grouping[option.Field] = compiledGrouping{grain: option.TimeGrain}
			continue
		}
		var dimension dto.ConfiguredSourceDimension
		for _, field := range schema.Dimensions {
			if field.Code == option.Field {
				dimension = field
				break
			}
		}
		var relation dto.ConfiguredSourceRelation
		for _, item := range schema.Relations {
			if item.RightTable == dimension.Table {
				relation = item
				break
			}
		}
		if relation.RightTable == "" || relation.LeftTable != schema.BaseTable || relation.JoinType != "left" {
			return nil, nil, errorsx.InvalidArgument("BI 层级维度必须通过左连接直接关联事实表")
		}
		table := tables[dimension.Table]
		index := strconv.Itoa(len(ctes))
		name := quoteSourceIdentifier(driver, "bi_hierarchy_"+index)
		physicalTable := quoteSourceTable(driver, table.SchemaName, table.Name)
		id := quoteSourceIdentifier(driver, relation.RightColumn)
		parent := quoteSourceIdentifier(driver, dimension.Hierarchy.ParentColumn)
		label := quoteSourceIdentifier(driver, dimension.Column)
		root := "d." + parent + " IS NULL"
		if dimension.Hierarchy.RootID != "" {
			rootID, err := strconv.ParseInt(dimension.Hierarchy.RootID, 10, 64)
			if err != nil || rootID < 0 {
				return nil, nil, errorsx.InvalidArgument("BI 层级根编号必须为非负整数")
			}
			root = "d." + parent + " = " + Placeholder(driver, len(*args)+1) + " AND d." + id + " <> d." + parent
			*args = append(*args, dto.ExecutionParameter{Position: len(*args) + 1, Kind: "hierarchy_root", RootID: rootID})
		}
		anchorTenant, childTenant, labelTenant := "", "", ""
		if table.TenantColumn != "" {
			tenant := quoteSourceIdentifier(driver, table.TenantColumn)
			anchorTenant = " AND d." + tenant + " = " + Placeholder(driver, len(*args)+1)
			*args = append(*args, dto.ExecutionParameter{Position: len(*args) + 1, Kind: "scope", Field: "tenant_id"})
			childTenant = " AND c." + tenant + " = " + Placeholder(driver, len(*args)+1)
			*args = append(*args, dto.ExecutionParameter{Position: len(*args) + 1, Kind: "scope", Field: "tenant_id"})
			labelTenant = tenant
		}
		// shortcut: 最多沿组织树向下遍历128层，更深的节点归入未归类，实际支持更深组织时调整限制。
		ctes = append(ctes, name+" (node_id, root_id, depth, first_id, second_id) AS ("+
			"SELECT d."+id+", d."+parent+", 1, d."+id+", CASE WHEN 1 = 0 THEN d."+id+" END FROM "+physicalTable+" AS d WHERE "+root+anchorTenant+
			" UNION ALL SELECT c."+id+", w.root_id, w.depth + 1, w.first_id, CASE WHEN w.depth = 1 THEN c."+id+" ELSE w.second_id END"+
			" FROM "+physicalTable+" AS c INNER JOIN "+name+" AS w ON c."+parent+" = w.node_id"+
			" WHERE w.depth < 128 AND (w.root_id IS NULL OR c."+id+" <> w.root_id)"+childTenant+")")
		walk, first, second := "bw"+index, "bf"+index, "bs"+index
		firstLabel := "COALESCE(" + first + "." + label + ", CONCAT('#', " + walk + ".first_id))"
		keys := walk + ".first_id"
		expression := "CASE WHEN " + walk + ".first_id IS NULL THEN '未归类' ELSE CONCAT(" + firstLabel + ", ' (#', " + walk + ".first_id, ')') END"
		if option.HierarchyLevel == 2 {
			keys += ", " + walk + ".second_id"
			expression = "CASE WHEN " + walk + ".first_id IS NULL THEN '未归类' WHEN " + walk + ".second_id IS NULL THEN CONCAT(" + firstLabel + ", ' / 直属 (#', " + walk + ".first_id, ')') ELSE CONCAT(" + firstLabel + ", ' / ', COALESCE(" + second + "." + label + ", ''), ' (#', " + walk + ".second_id, ')') END"
		}
		grouping[option.Field] = compiledGrouping{table: dimension.Table, expression: expression, keys: keys, join: func(alias string) string {
			joins := "LEFT JOIN " + name + " AS " + walk + " ON " + walk + ".node_id = " + alias + "." + id +
				" LEFT JOIN " + physicalTable + " AS " + first + " ON " + first + "." + id + " = " + walk + ".first_id"
			if labelTenant != "" {
				joins += " AND " + first + "." + labelTenant + " = " + alias + "." + labelTenant
			}
			if option.HierarchyLevel == 2 {
				joins += " LEFT JOIN " + physicalTable + " AS " + second + " ON " + second + "." + id + " = " + walk + ".second_id"
				if labelTenant != "" {
					joins += " AND " + second + "." + labelTenant + " = " + alias + "." + labelTenant
				}
			}
			return joins
		}}
	}
	return ctes, grouping, nil
}

// dateBucket 生成各自然周期的起始日期，周从星期一开始。
func dateBucket(driver, column, grain string) string {
	if driver == "postgres" {
		return "CAST(DATE_TRUNC('" + grain + "', " + column + ") AS DATE)"
	}
	var format string
	switch grain {
	case "day":
		format = "%Y-%m-%d"
	case "week":
		column = "DATE_SUB(DATE(" + column + "), INTERVAL WEEKDAY(" + column + ") DAY)"
		format = "%Y-%m-%d"
	case "month":
		format = "%Y-%m-01"
	case "quarter":
		return "CAST(CONCAT(YEAR(" + column + "), '-', LPAD((QUARTER(" + column + ") - 1) * 3 + 1, 2, '0'), '-01') AS DATE)"
	case "year":
		format = "%Y-01-01"
	}
	return "CAST(DATE_FORMAT(" + column + ", '" + format + "') AS DATE)"
}
