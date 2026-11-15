package dto

import "github.com/liujitcn/kratos-admin/backend/pkg/bi"

// ConfiguredSourceSchema 表示可配置数据源的字段语义和物理列映射。
type ConfiguredSourceSchema struct {
	// ModelRevision 标识查询结构及字段元数据修订。
	ModelRevision int64 `json:"model_revision"`
	// BaseTable 是事实表的结构标识。
	BaseTable string `json:"baseTable"`
	// Tables 是当前语义模型选中的表。
	Tables []ConfiguredSourceTable `json:"tables"`
	// Relations 是从事实表向维度表展开的多对一关系。
	Relations []ConfiguredSourceRelation `json:"relations"`
	// Dimensions 是可供分组和过滤的维度。
	Dimensions []ConfiguredSourceDimension `json:"dimensions"`
	// Measures 是可供聚合的指标。
	Measures []ConfiguredSourceMeasure `json:"measures"`
}

// ConfiguredSourceTable 表示模型画布中的一张表。
type ConfiguredSourceTable struct {
	// Fields 是发现时确认的物理字段类型，供纯编译和公式检查使用。
	Fields map[string]bi.FieldType `json:"fields"`
	// Key 是 Schema 和表名组成的结构标识。
	Key string `json:"key"`
	// SchemaName 是数据库 Schema。
	SchemaName string `json:"schemaName"`
	// Name 是数据库表或视图名称。
	Name string `json:"name"`
	// TenantColumn 是该表用于租户隔离的字段，为空表示连接级隔离或共享表。
	TenantColumn string `json:"tenantColumn"`
	// X 是画布横坐标。
	X int32 `json:"x"`
	// Y 是画布纵坐标。
	Y int32 `json:"y"`
}

// ConfiguredSourceRelation 表示事实表到维度表的一条多对一关联。
type ConfiguredSourceRelation struct {
	// LeftTable 是已连接表。
	LeftTable string `json:"leftTable"`
	// LeftColumn 是已连接表字段。
	LeftColumn string `json:"leftColumn"`
	// RightTable 是待连接维度表。
	RightTable string `json:"rightTable"`
	// RightColumn 是维度表唯一字段。
	RightColumn string `json:"rightColumn"`
	// JoinType 是 inner 或 left。
	JoinType string `json:"joinType"`
}

// ConfiguredSourceDimension 表示一个维度字段映射。
type ConfiguredSourceDimension struct {
	// Code 是对外语义编码。
	Code string `json:"code"`
	// Name 是展示名称。
	Name string `json:"name"`
	// Type 是字段类型。
	Type bi.FieldType `json:"type"`
	// Filterable 表示是否允许过滤。
	Filterable bool `json:"filterable"`
	// Table 是字段所属表结构标识。
	Table string `json:"table"`
	// Column 是数据库物理列名。
	Column string `json:"column"`
}

// ConfiguredSourceMeasure 表示一个指标字段映射。
type ConfiguredSourceMeasure struct {
	// Code 是对外语义编码。
	Code string `json:"code"`
	// Name 是展示名称。
	Name string `json:"name"`
	// Type 是结果类型。
	Type bi.FieldType `json:"type"`
	// Expression 是唯一固定聚合公式。
	Expression string `json:"expression"`
	// Unit 是公共展示单位。
	Unit string `json:"unit"`
	// Revision 取最近一次定义变更后的数据源版本。
	Revision int64 `json:"revision"`
}

// Descriptor 将配置语义转换为统一 BI 数据源描述。
func (s ConfiguredSourceSchema) Descriptor(code, name, description string, version, maxRows int32) bi.SourceDescriptor {
	dimensions := make([]bi.FieldDescriptor, 0, len(s.Dimensions))
	for _, item := range s.Dimensions {
		dimensions = append(dimensions, bi.FieldDescriptor{Code: item.Code, Name: item.Name, Type: item.Type, Filterable: item.Filterable})
	}
	measures := make([]bi.MeasureDescriptor, 0, len(s.Measures))
	for _, item := range s.Measures {
		measures = append(measures, bi.MeasureDescriptor{Code: item.Code, Name: item.Name, Type: item.Type, Unit: item.Unit, Revision: item.Revision})
	}
	return bi.SourceDescriptor{Code: code, Name: name, Description: description, Version: version, Dimensions: dimensions, Measures: measures, MaxRows: maxRows}
}
