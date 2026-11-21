package admin

import (
	"context"
	"encoding/json"
	"regexp"
	"slices"
	"strings"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
	"github.com/liujitcn/kratos-core/errorsx"
)

var (
	biMonthlyGrouping = regexp.MustCompile(`(?i)按(?:照)?(?:月份|月度|月)\s*(?:统计|分组|汇总|拆分|分布|列出)|` +
		`每(?:个)?月\s*(?:分别|有多少|新增了?多少)|月度\s*(?:趋势|分布|分组|汇总)|` +
		`\bmonthly\s+(?:breakdown|trend|grouping)\b|\b(?:group(?:ed)?|break\s+down|aggregate(?:d)?)\s+by\s+month\b`)
	biFirstLevelGrouping  = regexp.MustCompile(`(?i)(?:各|每个|按(?:照)?)(?:第)?(?:一|1)级(?:部门|组织)|(?:一|1)级(?:部门|组织)(?:分组|分布|汇总)|\b(?:group(?:ed)? by|by)\s+(?:level\s*1|first[- ]level)\s+(?:department|org)`)
	biSecondLevelGrouping = regexp.MustCompile(`(?i)(?:各|每个|按(?:照)?)(?:第)?(?:二|2)级(?:部门|组织)|(?:二|2)级(?:部门|组织)(?:分组|分布|汇总)|\b(?:group(?:ed)? by|by)\s+(?:level\s*2|second[- ]level)\s+(?:department|org)`)
)

// RefineBiQuery 保存一次人工调整后的问数结果，可作为独立的建卡来源。
func (c *BiQueryCase) RefineBiQuery(ctx context.Context, req *adminv1.RefineBiQueryRequest) (*adminv1.AskBiQueryResponse, error) {
	scope, err := c.scope(ctx)
	if err != nil {
		return nil, err
	}
	var parent *models.BiQuery
	parent, err = c.findOwnedAskRecord(ctx, scope, req.GetQueryRecordId())
	if err != nil {
		return nil, err
	}
	var spec bi.QuerySpec
	if err = json.Unmarshal([]byte(parent.QueryJSON), &spec); err != nil {
		return nil, errorsx.Internal("读取来源查询失败").WithCause(err)
	}
	spec.Dimensions = req.GetDimensions()
	spec.GroupingOptions = fromProtoGroupingOptions(req.GetGroupingOptions())
	spec.TimeRange = nil
	if date := req.GetTimeRange(); date != nil {
		spec.TimeRange = &bi.TimeRangeSpec{Field: date.GetField(), Start: date.GetStart(), End: date.GetEnd(), EndExclusive: date.GetEndExclusive()}
	}
	spec.Orders = slices.DeleteFunc(spec.Orders, func(order bi.OrderSpec) bool {
		return !slices.Contains(spec.Dimensions, order.Field) && !slices.ContainsFunc(spec.Measures, func(metric bi.MeasureSpec) bool { return metric.Code == order.Field })
	})
	for _, dimension := range spec.Dimensions {
		if !slices.ContainsFunc(spec.Orders, func(order bi.OrderSpec) bool { return order.Field == dimension }) && len(spec.Orders) < 2 {
			spec.Orders = append(spec.Orders, bi.OrderSpec{Field: dimension, Direction: "asc"})
		}
	}
	spec, _, err = c.validate(ctx, scope, spec)
	if err != nil {
		return nil, err
	}
	chartType := "bar"
	if len(spec.Dimensions) == 0 {
		chartType = "metric"
	} else if slices.ContainsFunc(spec.GroupingOptions, func(item bi.GroupingOption) bool { return item.TimeGrain != "" }) {
		chartType = "line"
	}
	var result bi.QueryResult
	result, err = c.execute(ctx, scope, spec, &dto.BiQueryRecordContext{Kind: dto.QueryKindAsk, Question: parent.Question, ParentRecordID: parent.ID, ChartType: chartType})
	if err != nil {
		return nil, err
	}
	return &adminv1.AskBiQueryResponse{Query: toProtoQuerySpec(spec), Result: toProtoQueryResult(result), ChartType: chartType}, nil
}

// requireRequestedGrouping 拒绝明确要求月度或部门层级时返回原始字段的查询计划。
func requireRequestedGrouping(question string, spec bi.QuerySpec) error {
	if biMonthlyGrouping.MatchString(question) && !slices.ContainsFunc(spec.GroupingOptions, func(item bi.GroupingOption) bool { return item.TimeGrain == "month" }) {
		return errorsx.InvalidArgument("问题要求按月汇总，请选择支持月度分组的日期维度")
	}
	for _, requested := range []struct {
		pattern *regexp.Regexp
		level   int32
	}{{biFirstLevelGrouping, 1}, {biSecondLevelGrouping, 2}} {
		if requested.pattern.MatchString(strings.ReplaceAll(question, " ", "")) || requested.pattern.MatchString(question) {
			if !slices.ContainsFunc(spec.GroupingOptions, func(item bi.GroupingOption) bool { return item.HierarchyLevel == requested.level }) {
				return errorsx.InvalidArgument("问题要求按指定部门层级汇总，请检查数据源的部门层级配置")
			}
		}
	}
	return nil
}

// fromProtoGroupingOptions 从接口请求读取日期粒度及层级。
func fromProtoGroupingOptions(items []*adminv1.BiGroupingOption) []bi.GroupingOption {
	options := make([]bi.GroupingOption, 0, len(items))
	for _, item := range items {
		options = append(options, bi.GroupingOption{Field: item.GetField(), TimeGrain: item.GetTimeGrain(), HierarchyLevel: item.GetHierarchyLevel()})
	}
	return options
}

// toProtoGroupingOptions 转换用于接口回显的分组参数。
func toProtoGroupingOptions(items []bi.GroupingOption) []*adminv1.BiGroupingOption {
	options := make([]*adminv1.BiGroupingOption, 0, len(items))
	for _, item := range items {
		options = append(options, &adminv1.BiGroupingOption{Field: item.Field, TimeGrain: item.TimeGrain, HierarchyLevel: item.HierarchyLevel})
	}
	return options
}
