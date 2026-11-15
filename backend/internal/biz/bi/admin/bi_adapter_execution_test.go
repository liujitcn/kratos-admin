package admin

import (
	"context"
	"testing"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
	"github.com/liujitcn/kratos-core/biz"
)

// TestAdapterExecutionLimit 验证预览与保存执行定义均遵守源上限，并裁剪超量的 Adapter 结果。
func TestAdapterExecutionLimit(t *testing.T) {
	for _, test := range []struct {
		name      string
		maxRows   int32
		requested int32
		want      int32
	}{
		{name: "源上限", maxRows: 10, requested: 1000, want: 10},
		{name: "查询较小", maxRows: 10, requested: 5, want: 5},
		{name: "未声明源上限", requested: 1000, want: 1000},
		{name: "默认查询上限", want: 100},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := newBiQueryRecordTestStore(t)
			registry := bi.NewRegistry()
			var received int32
			err := registry.Register(biLimitedTestSource{maxRows: test.maxRows, received: &received})
			if err != nil {
				t.Fatal(err)
			}
			baseCase := &biz.BaseCase{}
			source := NewBiSourceCase(baseCase, store, data.NewBiSourceRepository(store), registry, nil, data.NewBiCardRepository(store), data.NewBiDashboardRepository(store), data.NewBiDashboardVersionRepository(store), nil)
			c := NewBiQueryCase(baseCase, source, model.NewRegistry(), data.NewBiQueryRepository(store))
			ctx := biCardTestContext(1, 10, 0)
			spec := bi.QuerySpec{SourceCode: "test", Measures: []bi.MeasureSpec{{Code: "count"}}, Limit: test.requested}
			var result *adminv1.BiQueryResult
			result, err = c.PreviewBiQuery(ctx, &adminv1.PreviewBiQueryRequest{Query: toProtoQuerySpec(spec)})
			if err != nil {
				t.Fatal(err)
			}
			if received != test.want || len(result.Rows) != int(min(12, test.want)) {
				t.Errorf("预览未遵守有效上限 %d: Adapter收到=%d, 行数=%d", test.want, received, len(result.Rows))
			}
			var execution *dto.CardExecution
			execution, err = source.compile(ctx, bi.Scope{TenantID: 1, UserID: 10}, spec)
			if err != nil {
				t.Fatal(err)
			}
			if execution.Limit != test.want {
				t.Errorf("保存定义未冻结有效上限: %d, 期望 %d", execution.Limit, test.want)
			}
			var savedResult bi.QueryResult
			savedResult, err = c.execute(ctx, bi.Scope{TenantID: 1, UserID: 10}, spec, &dto.BiQueryRecordContext{Execution: execution})
			if err != nil {
				t.Fatal(err)
			}
			if received != test.want || len(savedResult.Rows) != int(min(12, test.want)) {
				t.Errorf("保存定义执行未遵守有效上限 %d: Adapter收到=%d, 行数=%d", test.want, received, len(savedResult.Rows))
			}
		})
	}
}

// biLimitedTestSource 故意返回超量行，验证调用端和结果端均有上限。
type biLimitedTestSource struct {
	maxRows  int32
	received *int32
}

// Code 返回测试源编码。
func (biLimitedTestSource) Code() string { return "test" }

// Descriptor 声明测试源的行数限制。
func (s biLimitedTestSource) Descriptor(context.Context, bi.Scope) (bi.SourceDescriptor, error) {
	return bi.SourceDescriptor{Code: "test", Version: 1, MaxRows: s.maxRows, Measures: []bi.MeasureDescriptor{{Code: "count", Name: "计数", Type: bi.FieldTypeNumber, Revision: 1}}}, nil
}

// Execute 记录接收上限并模拟未主动裁剪的宿主源。
func (s biLimitedTestSource) Execute(_ context.Context, _ bi.Scope, spec bi.QuerySpec) (bi.QueryResult, error) {
	*s.received = spec.Limit
	rows := make([][]string, 12)
	for index := range rows {
		rows[index] = []string{"1"}
	}
	return bi.QueryResult{Columns: []bi.ResultColumn{{Code: "count", Type: bi.FieldTypeNumber}}, Rows: rows}, nil
}
