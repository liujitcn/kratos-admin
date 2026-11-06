package admin

import (
	"strings"
	"testing"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
)

func TestConfiguredSourceModelAcceptsFactToUniqueDimension(t *testing.T) {
	model, err := configuredSourceModel(testSourceModel(), testSourceSchema())
	if err != nil {
		t.Fatalf("configuredSourceModel() error = %v", err)
	}
	if model.BaseTable != "sales.orders" || len(model.Tables) != 2 || len(model.Relations) != 1 || model.Measures[0].Table != model.BaseTable {
		t.Fatalf("unexpected model: %#v", model)
	}
}

func TestConfiguredSourceModelRejectsNonUniqueJoinTarget(t *testing.T) {
	model := testSourceModel()
	model.Relations[0].RightColumn = "name"
	if _, err := configuredSourceModel(model, testSourceSchema()); err == nil {
		t.Fatal("expected non-unique join target to be rejected")
	}
}

func TestConfiguredSourceModelRejectsDimensionMeasure(t *testing.T) {
	model := testSourceModel()
	model.Measures[0].Table = "sales.customers"
	if _, err := configuredSourceModel(model, testSourceSchema()); err == nil {
		t.Fatal("expected dimension-table measure to be rejected")
	}
}

func TestMarshalConfiguredSourceModelRejectsOversizedJSON(t *testing.T) {
	if _, err := marshalConfiguredSourceModel(dto.ConfiguredSourceSchema{BaseTable: strings.Repeat("x", 50001)}); err == nil {
		t.Fatal("expected oversized model to be rejected")
	}
}

// testSourceModel 返回事实表关联唯一维度表的请求模型。
func testSourceModel() *adminv1.BiSourceModel {
	return &adminv1.BiSourceModel{
		BaseTable: "sales.orders",
		Tables: []*adminv1.BiSourceModelTable{
			{Key: "sales.orders", TenantColumn: "tenant_id"},
			{Key: "sales.customers", TenantColumn: "tenant_id"},
		},
		Relations:  []*adminv1.BiSourceRelation{{LeftTable: "sales.orders", LeftColumn: "customer_id", RightTable: "sales.customers", RightColumn: "id", JoinType: "left"}},
		Dimensions: []*adminv1.BiSourceDimension{{Code: "customer_name", Name: "客户", Type: string(bi.FieldTypeString), Table: "sales.customers", Column: "name", Filterable: true}},
		Measures:   []*adminv1.BiSourceMeasure{{Code: "revenue", Name: "收入", Type: string(bi.FieldTypeNumber), Table: "sales.orders", Column: "amount", Aggregations: []string{"sum"}}},
	}
}

// testSourceSchema 返回事实表和唯一维度表的数据库结构。
func testSourceSchema() []*adminv1.BiSourceSchemaTable {
	return []*adminv1.BiSourceSchemaTable{
		{
			Key: "sales.orders", SchemaName: "sales", Name: "orders", TableType: "BASE TABLE",
			Columns: []*adminv1.BiSourceSchemaColumn{
				{Name: "tenant_id", DataType: "bigint"},
				{Name: "customer_id", DataType: "bigint"},
				{Name: "amount", DataType: "decimal"},
			},
		},
		{
			Key: "sales.customers", SchemaName: "sales", Name: "customers", TableType: "BASE TABLE",
			Columns: []*adminv1.BiSourceSchemaColumn{
				{Name: "tenant_id", DataType: "bigint"},
				{Name: "id", DataType: "bigint", PrimaryKey: true, Unique: true},
				{Name: "name", DataType: "varchar"},
			},
		},
	}
}
