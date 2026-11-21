package query

import (
	"fmt"

	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
	"github.com/liujitcn/kratos-core/errorsx"
)

// ValidateSpec 校验并归一化固定查询规则，可变的数据源行数上限由执行定义约束。
func ValidateSpec(descriptor bi.SourceDescriptor, spec bi.QuerySpec) (bi.QuerySpec, error) {
	if len(spec.Dimensions) > 2 || len(spec.Measures) == 0 || len(spec.Measures) > 5 || len(spec.Filters) > 10 || len(spec.Orders) > 2 {
		return bi.QuerySpec{}, errorsx.InvalidArgument("BI 查询维度、指标或过滤条件超出限制")
	}
	if spec.Limit <= 0 {
		spec.Limit = 100
	}
	spec.Limit = min(spec.Limit, 1000)
	dimensions := make(map[string]bi.FieldDescriptor, len(descriptor.Dimensions))
	for _, item := range descriptor.Dimensions {
		dimensions[item.Code] = item
	}
	for _, code := range spec.Dimensions {
		if _, ok := dimensions[code]; !ok {
			return bi.QuerySpec{}, errorsx.WithMessageKey(errorsx.InvalidArgument(fmt.Sprintf("BI 维度不存在: %s", code)), "bi.query.error.dimension_missing", map[string]string{"Code": code})
		}
	}
	measures := make(map[string]bi.MeasureDescriptor, len(descriptor.Measures))
	for _, item := range descriptor.Measures {
		measures[item.Code] = item
	}
	for _, item := range spec.Measures {
		_, ok := measures[item.Code]
		if !ok {
			return bi.QuerySpec{}, errorsx.WithMessageKey(errorsx.InvalidArgument(fmt.Sprintf("BI 指标不可用: %s", item.Code)), "bi.query.error.measure_unavailable", map[string]string{"Code": item.Code})
		}
	}
	for _, item := range spec.Filters {
		field, ok := dimensions[item.Field]
		if !ok || !field.Filterable || len(item.Values) == 0 {
			return bi.QuerySpec{}, errorsx.WithMessageKey(errorsx.InvalidArgument(fmt.Sprintf("BI 过滤字段不可用: %s", item.Field)), "bi.query.error.filter_field_unavailable", map[string]string{"Field": item.Field})
		}
		if ((item.Operator == "eq" || item.Operator == "contains") && len(item.Values) != 1) || (item.Operator == "between" && len(item.Values) != 2) {
			return bi.QuerySpec{}, errorsx.InvalidArgument("BI 过滤操作符和值数量不匹配")
		}
		if item.Operator != "eq" && item.Operator != "in" && item.Operator != "between" && item.Operator != "contains" {
			return bi.QuerySpec{}, errorsx.InvalidArgument("BI 过滤操作符不支持")
		}
	}
	if spec.TimeRange != nil {
		field, ok := dimensions[spec.TimeRange.Field]
		if !ok || field.Type != bi.FieldTypeDate || !field.Filterable || spec.TimeRange.Start == "" || spec.TimeRange.End == "" {
			return bi.QuerySpec{}, errorsx.InvalidArgument("BI 时间范围不可用")
		}
	}
	selected := make(map[string]struct{}, len(spec.Dimensions)+len(spec.Measures))
	for _, code := range spec.Dimensions {
		if _, exists := selected[code]; exists {
			return bi.QuerySpec{}, errorsx.InvalidArgument("BI 查询字段不能重复")
		}
		selected[code] = struct{}{}
	}
	for _, item := range spec.Measures {
		if _, exists := selected[item.Code]; exists {
			return bi.QuerySpec{}, errorsx.InvalidArgument("BI 查询字段不能重复")
		}
		selected[item.Code] = struct{}{}
	}
	for _, item := range spec.Orders {
		if _, ok := selected[item.Field]; !ok || (item.Direction != "asc" && item.Direction != "desc") {
			return bi.QuerySpec{}, errorsx.InvalidArgument("BI 排序字段或方向不可用")
		}
	}
	return spec, nil
}
