package biz

import "testing"

// TestMatchToolTerminalSharesBiQuery 验证应用端只复用 BI 查询工具，不暴露其他管理端能力。
func TestMatchToolTerminalSharesBiQuery(t *testing.T) {
	for _, test := range []struct {
		name string
		want bool
	}{
		{name: "bi_admin_v1_bi_query_service_ask_bi_query", want: true},
		{name: "bi_admin_v1_bi_query_service_list_bi_sources", want: true},
		{name: "bi_admin_v1_bi_query_service_preview_bi_query", want: true},
		{name: "bi_admin_v1_bi_query_record_service_get_bi_query"},
		{name: "bi_admin_v1_bi_source_service_save_bi_source"},
		{name: "system_admin_v1_base_user_service_page_base_user"},
	} {
		if got := matchToolTerminal("app", test.name); got != test.want {
			t.Errorf("应用端工具 %s 可见性 = %v，期望 %v", test.name, got, test.want)
		}
	}
}
