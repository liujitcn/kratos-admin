package workflow

import "testing"

func TestEvalCondition(t *testing.T) {
	cases := []struct {
		name      string
		condition string
		formData  string
		want      bool
		wantErr   bool
	}{
		{"数值比较命中", "amount > 1000", `{"amount": 2000}`, true, false},
		{"数值比较未命中", "amount > 1000", `{"amount": 100}`, false, false},
		{"字符串相等", "type == 'urgent'", `{"type": "urgent"}`, true, false},
		{"字段缺失", "days > 3", `{}`, false, true},
		{"表达式语法错", "amount >", `{"amount": 1}`, false, true},
		{"结果非布尔", "amount", `{"amount": 1}`, false, true},
		{"表单数据非法JSON", "true", `not-json`, false, true},
		{"空表达式恒假", "", `{"amount": 1}`, false, false},
	}
	for _, c := range cases {
		got, err := EvalCondition(c.condition, c.formData)
		if c.wantErr {
			if err == nil {
				t.Fatalf("%s: 期望求值失败", c.name)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: 求值失败: %v", c.name, err)
		}
		if got != c.want {
			t.Fatalf("%s: 结果期望 %v 实际 %v", c.name, c.want, got)
		}
	}
}
