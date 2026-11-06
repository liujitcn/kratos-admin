package dto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

// BiQueryRecordContext 保存一次查询的提问和仪表盘执行上下文。
type BiQueryRecordContext struct {
	// Question 保存自然语言问题，非问数查询为空。
	Question string
	// DashboardID 标识执行所属仪表盘。
	DashboardID int64
	// DashboardVersion 标识执行的仪表盘版本。
	DashboardVersion int64
	// WidgetID 标识执行的图表组件。
	WidgetID string
	// StartedAt 用于计算整次查询耗时。
	StartedAt time.Time
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
		SourceCode string        `json:"source_code"`
		Dimensions []string      `json:"dimensions"`
		Measures   []AskMeasure  `json:"measures"`
		Filters    []AskFilter   `json:"filters"`
		TimeRange  *AskTimeRange `json:"time_range"`
		Orders     []AskOrder    `json:"orders"`
		Limit      int32         `json:"limit"`
	} `json:"query"`
	ChartType string `json:"chart_type"`
	Title     string `json:"title"`
	Error     string `json:"error"`
}

// AskMeasure 模型选择的语义指标和聚合函数。
type AskMeasure struct {
	Code        string `json:"code"`
	Aggregation string `json:"aggregation"`
}

// AskFilter 模型生成的过滤条件。
type AskFilter struct {
	Field    string            `json:"field"`
	Operator string            `json:"operator"`
	Values   BiAskFilterValues `json:"values"`
}

// AskTimeRange 模型生成的时间范围。
type AskTimeRange struct {
	Field string `json:"field"`
	Start string `json:"start"`
	End   string `json:"end"`
}

// AskOrder 模型选择的结果排序规则。
type AskOrder struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}
