package query

import (
	"strings"
	"testing"

	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
)

// TestNormalizeTimeGrain 验证模型输出的中英文粒度同义归一与未知值透传。
func TestNormalizeTimeGrain(t *testing.T) {
	for input, expected := range map[string]string{
		"month": "month", "月": "month", "每月": "month", "MONTH": "month", " monthly ": "month",
		"day": "day", "日": "day", "天": "day",
		"周": "week", "quarter": "quarter", "季度": "quarter",
		"year": "year", "年": "year", "每年": "year",
		"": "", "months": "months",
	} {
		if actual := normalizeTimeGrain(input); actual != expected {
			t.Fatalf("粒度 %q 归一为 %q，期望 %q", input, actual, expected)
		}
	}
}

// TestValidateSpecNormalizesGroupingGrain 验证分组选项的中文粒度在校验前被归一为目录编码。
func TestValidateSpecNormalizesGroupingGrain(t *testing.T) {
	descriptor := bi.SourceDescriptor{Dimensions: []bi.FieldDescriptor{
		{Code: "created_at", Name: "创建时间", Type: bi.FieldTypeDate, TimeGrains: []string{"day", "week", "month", "quarter", "year"}},
	}, Measures: []bi.MeasureDescriptor{{Code: "user_count", Name: "人员数量", Type: bi.FieldTypeNumber}}}
	spec := bi.QuerySpec{SourceCode: "default", Dimensions: []string{"created_at"}, Measures: []bi.MeasureSpec{{Code: "user_count"}}, GroupingOptions: []bi.GroupingOption{{Field: "created_at", TimeGrain: "月"}}}
	validated, err := ValidateSpec(descriptor, spec)
	if err != nil {
		t.Fatalf("中文粒度应归一通过: %v", err)
	}
	if len(validated.GroupingOptions) != 1 || validated.GroupingOptions[0].TimeGrain != "month" {
		t.Fatalf("分组选项粒度应归一为 month: %+v", validated.GroupingOptions)
	}
	// 未知粒度归一后仍应被拒绝，且报错信息携带请求值供诊断。
	spec.GroupingOptions = []bi.GroupingOption{{Field: "created_at", TimeGrain: "months"}}
	_, err = ValidateSpec(descriptor, spec)
	if err == nil || !strings.Contains(err.Error(), `"months"`) {
		t.Fatalf("未知粒度应拒绝并携带请求值: %v", err)
	}
}
