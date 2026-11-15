package query

import (
	"strings"
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
)

// TestCompileParameters 验证身份、业务值和相对时间不写入执行 SQL。
func TestCompileParameters(t *testing.T) {
	model := dto.ConfiguredSourceSchema{BaseTable: "orders", ModelRevision: 3,
		Tables:     []dto.ConfiguredSourceTable{{Key: "orders", Name: "orders", TenantColumn: "tenant_id", Fields: map[string]bi.FieldType{"amount": bi.FieldTypeNumber, "created_at": bi.FieldTypeDate, "shop": bi.FieldTypeString}}},
		Dimensions: []dto.ConfiguredSourceDimension{{Code: "day", Table: "orders", Column: "created_at", Type: bi.FieldTypeDate, Filterable: true}, {Code: "shop", Table: "orders", Column: "shop", Type: bi.FieldTypeString, Filterable: true}},
		Measures:   []dto.ConfiguredSourceMeasure{{Code: "sales", Name: "销售额", Expression: "SUM(amount)", Type: bi.FieldTypeNumber, Revision: 7, Unit: "元"}}}
	spec := bi.QuerySpec{SourceCode: "orders", Dimensions: []string{"day"}, Measures: []bi.MeasureSpec{{Code: "sales"}}, Filters: []bi.FilterSpec{{Field: "shop", Operator: "in", Values: []string{"秘密门店", "店2"}}}, TimeRange: &bi.TimeRangeSpec{Field: "day", Start: "today-7d", End: "today"}, Limit: 10}
	for _, driver := range []string{"mysql", "postgres"} {
		execution, err := Compile(driver, 100, model, spec)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(execution.SQL, "秘密") || strings.Contains(execution.SQL, "today") || len(execution.Parameters) != 6 || execution.Hash == "" {
			t.Fatalf("SQL 未正确保留动态参数: %+v", execution)
		}
		if driver == "postgres" && (!strings.Contains(execution.SQL, `"t0"."tenant_id" = $1`) || !strings.Contains(execution.SQL, "$6")) {
			t.Fatalf("PostgreSQL 参数顺序错误: %s", execution.SQL)
		}
		if execution.Columns[1].Revision != 7 || execution.Columns[1].Unit != "元" || execution.Columns[1].Role != "measure" {
			t.Fatal("未冻结固定指标元数据")
		}
	}
}

// TestSourceLimitDoesNotRewriteQuery 验证源上限只影响实际执行，不改变归一化后的查询规格。
func TestSourceLimitDoesNotRewriteQuery(t *testing.T) {
	model := dto.ConfiguredSourceSchema{BaseTable: "orders", ModelRevision: 1, Tables: []dto.ConfiguredSourceTable{{Key: "orders", Name: "orders"}}, Measures: []dto.ConfiguredSourceMeasure{{Code: "count", Name: "笔数", Expression: "COUNT(*)", Type: bi.FieldTypeNumber, Revision: 1}}}
	for _, test := range []struct {
		name          string
		maxRows       int32
		requested     int32
		wantQuery     int32
		wantExecution int32
	}{
		{name: "降低源上限", maxRows: 10, requested: 100, wantQuery: 100, wantExecution: 10},
		{name: "查询小于源上限", maxRows: 100, requested: 5, wantQuery: 5, wantExecution: 5},
		{name: "默认查询上限", maxRows: 10, wantQuery: 100, wantExecution: 10},
		{name: "固定硬上限", maxRows: 10, requested: 2000, wantQuery: 1000, wantExecution: 10},
		{name: "未声明源上限", requested: 1000, wantQuery: 1000, wantExecution: 1000},
	} {
		t.Run(test.name, func(t *testing.T) {
			descriptor := model.Descriptor("orders", "", "", 1, test.maxRows)
			spec, err := ValidateSpec(descriptor, bi.QuerySpec{SourceCode: "orders", Measures: []bi.MeasureSpec{{Code: "count"}}, Limit: test.requested})
			if err != nil {
				t.Fatal(err)
			}
			if spec.Limit != test.wantQuery {
				t.Errorf("源上限改写了查询: limit=%d，期望%d", spec.Limit, test.wantQuery)
			}
			var configured, adapter *dto.CardExecution
			configured, err = Compile("mysql", test.maxRows, model, spec)
			if err != nil {
				t.Fatal(err)
			}
			adapter, err = AdapterExecution(descriptor, spec)
			if err != nil {
				t.Fatal(err)
			}
			for _, execution := range []*dto.CardExecution{configured, adapter} {
				if execution.Limit != test.wantExecution || execution.QueryHash != SpecHash(spec) {
					t.Errorf("%s 未保持查询身份及有效上限: limit=%d，期望%d", execution.Kind, execution.Limit, test.wantExecution)
				}
			}
		})
	}
}

// TestCompileRejectsLostTimeAxis 防止公共模型同步静默丢弃已配置日期条件。
func TestCompileRejectsLostTimeAxis(t *testing.T) {
	model := dto.ConfiguredSourceSchema{BaseTable: "orders", Tables: []dto.ConfiguredSourceTable{{Key: "orders", Name: "orders", Fields: map[string]bi.FieldType{"amount": bi.FieldTypeNumber}}}, Measures: []dto.ConfiguredSourceMeasure{{Code: "sales", Name: "销售额", Expression: "SUM(amount)", Type: bi.FieldTypeNumber, Revision: 1}}}
	spec := bi.QuerySpec{SourceCode: "sales", Measures: []bi.MeasureSpec{{Code: "sales"}}, TimeRange: &bi.TimeRangeSpec{Field: "day", Start: "today-7d", End: "today"}, Limit: 100}
	_, err := Compile("mysql", 100, model, spec)
	if err == nil {
		t.Fatal("删除日期轴后仍编译成功，原时间过滤被丢弃")
	}
	model.Dimensions = []dto.ConfiguredSourceDimension{{Code: "day", Table: "orders", Column: "amount", Type: bi.FieldTypeNumber, Filterable: true}}
	if _, err = Compile("mysql", 100, model, spec); err == nil {
		t.Fatal("日期类型改为数字后仍可编译")
	}
}
