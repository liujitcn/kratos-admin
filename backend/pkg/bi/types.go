package bi

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"sync"
)

// Scope 表示一次 BI 查询的可信访问范围。
type Scope struct {
	// TenantID 表示当前租户。
	TenantID int64
	// ProjectID 表示当前项目，0 表示租户级查询。
	ProjectID int64
	// UserID 表示当前用户。
	UserID int64
	// Terminal 表示调用终端，例如 admin 或 app。
	Terminal string
}

// FieldType 表示 BI 字段类型。
type FieldType string

const (
	// FieldTypeString 表示字符串字段。
	FieldTypeString FieldType = "string"
	// FieldTypeNumber 表示数值字段。
	FieldTypeNumber FieldType = "number"
	// FieldTypeDate 表示日期时间字段。
	FieldTypeDate FieldType = "date"
)

// FieldDescriptor 描述可供 BI 使用的维度或过滤字段。
type FieldDescriptor struct {
	// Code 是字段编码。
	Code string `json:"code"`
	// Name 是展示名称。
	Name string `json:"name"`
	// Type 是字段类型。
	Type FieldType `json:"type"`
	// Filterable 表示字段是否允许过滤。
	Filterable bool `json:"filterable"`
}

// MeasureDescriptor 描述可供 BI 使用的指标。
type MeasureDescriptor struct {
	// Code 是指标编码。
	Code string `json:"code"`
	// Name 是展示名称。
	Name string `json:"name"`
	// Type 是结果类型。
	Type FieldType `json:"type"`
	// Aggregations 是允许的聚合函数。
	Aggregations []string `json:"aggregations"`
}

// SourceDescriptor 描述一个宿主注册的数据源。
type SourceDescriptor struct {
	// Code 是全局唯一的数据源编码。
	Code string `json:"code"`
	// Name 是展示名称。
	Name string `json:"name"`
	// Description 是数据源说明。
	Description string `json:"description"`
	// Version 是数据源语义版本。
	Version int32 `json:"version"`
	// Dimensions 是可用维度。
	Dimensions []FieldDescriptor `json:"dimensions"`
	// Measures 是可用指标。
	Measures []MeasureDescriptor `json:"measures"`
	// MaxRows 是该数据源允许返回的最大行数。
	MaxRows int32 `json:"maxRows"`
}

// MeasureSpec 表示一次查询中的指标和聚合方式。
type MeasureSpec struct {
	// Code 是指标编码。
	Code string `json:"code"`
	// Aggregation 是聚合函数。
	Aggregation string `json:"aggregation"`
}

// FilterSpec 表示一次查询中的字段过滤条件。
type FilterSpec struct {
	// Field 是字段编码。
	Field string `json:"field"`
	// Operator 是过滤操作符。
	Operator string `json:"operator"`
	// Values 是过滤值；范围和 in 条件可以包含多个值。
	Values []string `json:"values"`
}

// TimeRangeSpec 表示一次查询使用的时间范围。
type TimeRangeSpec struct {
	// Field 是时间维度编码。
	Field string `json:"field"`
	// Start 是范围起点。
	Start string `json:"start"`
	// End 是范围终点。
	End string `json:"end"`
}

// OrderSpec 表示一次查询的结果排序规则。
type OrderSpec struct {
	// Field 是已选择的维度或指标编码。
	Field string `json:"field"`
	// Direction 是排序方向，取 asc 或 desc。
	Direction string `json:"direction"`
}

// QuerySpec 表示可执行的语义查询计划。
type QuerySpec struct {
	// SourceCode 是数据源编码。
	SourceCode string `json:"source_code"`
	// Dimensions 是分组维度编码。
	Dimensions []string `json:"dimensions"`
	// Measures 是查询指标。
	Measures []MeasureSpec `json:"measures"`
	// Filters 是业务过滤条件。
	Filters []FilterSpec `json:"filters"`
	// TimeRange 是可选的时间范围。
	TimeRange *TimeRangeSpec `json:"time_range,omitempty"`
	// Orders 是结果排序规则。
	Orders []OrderSpec `json:"orders"`
	// Limit 是结果行数上限。
	Limit int32 `json:"limit"`
}

// ResultColumn 描述查询结果列。
type ResultColumn struct {
	// Code 是结果列编码。
	Code string `json:"code"`
	// Name 是结果列名称。
	Name string `json:"name"`
	// Type 是结果列类型。
	Type FieldType `json:"type"`
}

// QueryResult 是所有 BI 数据源统一返回的结果。
type QueryResult struct {
	// QueryRecordID 标识本次成功执行的来源记录。
	QueryRecordID int64 `json:"query_record_id"`
	// Columns 是结果列。
	Columns []ResultColumn `json:"columns"`
	// Rows 是结果行，每一行按 Columns 顺序排列。
	Rows [][]string `json:"rows"`
	// Truncated 表示结果是否被限制截断。
	Truncated bool `json:"truncated"`
	// ElapsedMS 是查询耗时毫秒数。
	ElapsedMS int64 `json:"elapsedMs"`
}

// SourceAdapter 是宿主业务数据接入 BI 的唯一扩展接口。
type SourceAdapter interface {
	// Code 返回全局唯一的数据源编码。
	Code() string
	// Descriptor 返回当前访问范围可见的数据源描述。
	Descriptor(context.Context, Scope) (SourceDescriptor, error)
	// Execute 执行已经过服务端校验的语义查询。
	Execute(context.Context, Scope, QuerySpec) (QueryResult, error)
}

// Registry 管理宿主注册的数据源 Adapter。
type Registry struct {
	mu       sync.RWMutex
	adapters map[string]SourceAdapter
}

// NewRegistry 创建空的数据源注册表。
func NewRegistry() *Registry {
	return &Registry{adapters: make(map[string]SourceAdapter)}
}

// Register 注册一个数据源 Adapter；同一编码只能注册一次。
func (r *Registry) Register(adapter SourceAdapter) error {
	if adapter == nil || adapter.Code() == "" {
		return fmt.Errorf("BI 数据源 Adapter 不能为空")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.adapters[adapter.Code()]; exists {
		return fmt.Errorf("BI 数据源重复注册: %s", adapter.Code())
	}
	r.adapters[adapter.Code()] = adapter
	return nil
}

// Has 判断指定数据源是否已经由宿主静态注册。
func (r *Registry) Has(code string) bool {
	if r == nil || code == "" {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.adapters[code]
	return exists
}

// List 返回当前访问范围内的全部数据源描述，并按编码排序。
func (r *Registry) List(ctx context.Context, scope Scope) ([]SourceDescriptor, error) {
	r.mu.RLock()
	adapters := make([]SourceAdapter, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		adapters = append(adapters, adapter)
	}
	r.mu.RUnlock()
	slices.SortFunc(adapters, func(left, right SourceAdapter) int { return cmp.Compare(left.Code(), right.Code()) })
	result := make([]SourceDescriptor, 0, len(adapters))
	for _, adapter := range adapters {
		descriptor, err := adapter.Descriptor(ctx, scope)
		if err != nil {
			return nil, err
		}
		result = append(result, descriptor)
	}
	return result, nil
}

// Execute 根据数据源编码执行查询。
func (r *Registry) Execute(ctx context.Context, scope Scope, spec QuerySpec) (QueryResult, error) {
	r.mu.RLock()
	adapter := r.adapters[spec.SourceCode]
	r.mu.RUnlock()
	if adapter == nil {
		return QueryResult{}, fmt.Errorf("BI 数据源不存在: %s", spec.SourceCode)
	}
	return adapter.Execute(ctx, scope, spec)
}
