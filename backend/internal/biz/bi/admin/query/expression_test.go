package query

import (
	"strings"
	"testing"

	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
)

// TestExpression 验证聚合公式的封闭语法、字段类型和安全边界。
func TestExpression(t *testing.T) {
	var err error
	fields := map[string]bi.FieldType{"amount": bi.FieldTypeNumber, "refund": bi.FieldTypeNumber, "name": bi.FieldTypeString}
	for _, input := range []string{"sum(amount-refund)", "COUNT(*)", "COUNT(name)", "COALESCE(SUM(amount), 0)", "SUM(amount)/NULLIF(COUNT(*),0)", "AVG(-amount + 0.25)", "MIN(amount) + MAX(refund)"} {
		var expression *Expression
		expression, err = Parse(input, fields)
		if err != nil {
			t.Fatalf("合法公式 %q: %v", input, err)
		}
		var canonical *Expression
		canonical, err = Parse(expression.Canonical(), fields)
		if err != nil || canonical.Canonical() != expression.Canonical() {
			t.Fatalf("规范化不稳定: %q %v", expression.Canonical(), err)
		}
	}
	for _, input := range []string{"amount", "SUM(amount)+refund", "SUM(SUM(amount))", "SUM(name)", "COUNT(amount+1)", "SUM(other)", "SUM(t.amount)", "SUM(amount);DELETE", "SUM(amount)--x", "SUM(amount)/*x*/", "SUM(amount)//2", "SUM(DISTINCT amount)", "CASE WHEN 1 THEN SUM(amount) END", "SLEEP(1)", "SUM(amount) OVER ()", "SUM(0x10)", "SUM(1e3)", "SUM('1')", "@amount", "NULLIF(SUM(amount))", "COALESCE(SUM(amount),'0')", "1+2", strings.Repeat("(", 33) + "SUM(amount)" + strings.Repeat(")", 33), strings.Repeat(" ", 2049)} {
		if _, err = Parse(input, fields); err == nil {
			t.Fatalf("接受非法公式 %q", input)
		}
	}
}

// TestExpressionDivision 验证 PostgreSQL 计数比例使用精确小数除法。
func TestExpressionDivision(t *testing.T) {
	expression, err := Parse("COUNT(*) / NULLIF(COUNT(*),0)", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := expression.SQL(func(name string) string { return `"t0"."` + name + `"` }); got != "((COUNT(*) * 1.0) / NULLIF(COUNT(*), 0))" {
		t.Fatalf("整数除法未提升为精确小数: %s", got)
	}
}

// TestExpressionDepthBoundary 验证32层限制按实际嵌套计算，允许31层括号加一层聚合。
func TestExpressionDepthBoundary(t *testing.T) {
	_, err := Parse(strings.Repeat("(", 31)+"SUM(amount)"+strings.Repeat(")", 31), map[string]bi.FieldType{"amount": bi.FieldTypeNumber})
	if err != nil {
		t.Fatal("合法嵌套被提前拒绝", err)
	}
}
