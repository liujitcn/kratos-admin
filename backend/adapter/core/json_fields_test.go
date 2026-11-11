package core

import "testing"

// TestNormalizeJSONArrayUsesEmptyArray 验证数组型 JSON 空值使用合法空数组。
func TestNormalizeJSONArrayUsesEmptyArray(t *testing.T) {
	for _, value := range []string{"", "null", "[\"prompt\"]"} {
		got := normalizeJSONArray(value)
		want := value
		if value == "" || value == "null" {
			want = "[]"
		}
		if got != want {
			t.Fatalf("输入 %q 归一化为 %q，期望 %q", value, got, want)
		}
	}
}
