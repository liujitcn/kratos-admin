package dto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
)

// BiQueryRecordContext 保存一次查询的提问和仪表盘执行上下文。
type BiQueryRecordContext struct {
	// Kind 标识问数、预览或保存资产执行。
	Kind int32
	// RangeDays 是受控周期覆盖，只影响执行参数。
	RangeDays int32
	// CardID 标识当前执行的卡片。
	CardID int64
	// Execution 冻结本次实际执行定义。
	Execution *CardExecution
	// Preview 冻结候选版本及公共指标差异。
	Preview *CardPreviewContext
	// Question 保存自然语言问题，非问数查询为空。
	Question string
	// ChartType 保存问数模型建议的图表类型。
	ChartType string
	// ParentRecordID 标识该结果从哪个成功记录调整得来。
	ParentRecordID int64
	// DashboardID 标识执行所属仪表盘。
	DashboardID int64
	// DashboardVersion 标识执行的仪表盘版本。
	DashboardVersion int64
	// WidgetID 标识执行的图表组件。
	WidgetID string
	// StartedAt 用于计算整次查询耗时。
	StartedAt time.Time
}

// BiQueryRecordMetadata 保存查询记录的可选展示元数据。
type BiQueryRecordMetadata struct {
	// ChartType 是问数模型建议的图表类型。
	ChartType string `json:"chart_type,omitempty"`
	// ParentRecordID 标识派生问数的直接来源。
	ParentRecordID int64 `json:"parent_record_id,omitempty"`
}

// BiAskFilterValues 表示模型返回的过滤值，统一为查询协议使用的字符串。
type BiAskFilterValues []string

// UnmarshalJSON 接收字符串或数字过滤值，并保留大整数的精度。
func (v *BiAskFilterValues) UnmarshalJSON(data []byte) error {
	var values []any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	err := decoder.Decode(&values)
	if err != nil {
		return fmt.Errorf("解析 BI 过滤值失败: %w", err)
	}
	normalized := make([]string, len(values))
	for index, value := range values {
		switch item := value.(type) {
		case string:
			normalized[index] = item
		case json.Number:
			normalized[index] = item.String()
		default:
			return fmt.Errorf("BI 过滤值[%d]必须是字符串或数字", index)
		}
	}
	*v = normalized
	return nil
}

// AskOutput 模型生成的受控查询计划和图表建议。
type AskOutput struct {
	Query struct {
		SourceCode      string              `json:"source_code"`
		Dimensions      []string            `json:"dimensions"`
		GroupingOptions []bi.GroupingOption `json:"grouping_options"`
		Measures        []AskMeasure        `json:"measures"`
		Filters         []AskFilter         `json:"filters"`
		TimeRange       *AskTimeRange       `json:"time_range"`
		Orders          []AskOrder          `json:"orders"`
		Limit           int32               `json:"limit"`
	} `json:"query"`
	ChartType string `json:"chart_type"`
	Title     string `json:"title"`
	Error     string `json:"error"`
}

// AskMeasure 模型选择的固定公共指标。
type AskMeasure struct {
	Code string `json:"code"`
}

// AskFilter 模型生成的过滤条件。
type AskFilter struct {
	Field    string            `json:"field"`
	Operator string            `json:"operator"`
	Values   BiAskFilterValues `json:"values"`
}

// AskTimeRange 模型生成的时间范围。
type AskTimeRange struct {
	Field        string `json:"field"`
	Start        string `json:"start"`
	End          string `json:"end"`
	EndExclusive bool   `json:"end_exclusive,omitempty"`
}

// AskOrder 模型选择的结果排序规则。
type AskOrder struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}
