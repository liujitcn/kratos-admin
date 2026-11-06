package admin

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/dto"

	"github.com/liujitcn/kratos-admin/backend/pkg/agent"
	"github.com/liujitcn/kratos-core/biz"
)

// TestBiQueryScopeTerminal 验证共享查询保留应用端终端，直接 BI 请求使用管理端终端。
func TestBiQueryScopeTerminal(t *testing.T) {
	c := &BiQueryCase{BaseCase: &biz.BaseCase{}}
	for _, terminal := range []string{"app", "admin", ""} {
		ctx := agent.ContextWithTerminal(biCardTestContext(1, 10, 42), terminal)
		scope, err := c.scope(ctx)
		if err != nil {
			t.Fatal(err)
		}
		want := terminal
		if want == "" {
			want = "admin"
		}
		if scope.Terminal != want || scope.TenantID != 1 || scope.ProjectID != 42 || scope.UserID != 10 {
			t.Fatalf("共享查询访问范围错误: %+v", scope)
		}
	}
}

// TestBiAskOutputFilterValues 验证模型过滤值转换及非法结构拒绝行为。
func TestBiAskOutputFilterValues(t *testing.T) {
	tests := []struct {
		name      string
		values    string
		want      []string
		wantError bool
	}{
		{name: "字符串", values: `["0", "管理员"]`, want: []string{"0", "管理员"}},
		{name: "数字", values: `[0, 1.25]`, want: []string{"0", "1.25"}},
		{name: "大整数", values: `[9007199254740993, 9223372036854775807]`, want: []string{"9007199254740993", "9223372036854775807"}},
		{name: "混合值", values: `["管理员", 0]`, want: []string{"管理员", "0"}},
		{name: "对象", values: `[{"value": 0}]`, wantError: true},
		{name: "嵌套数组", values: `[[0]]`, wantError: true},
		{name: "空值", values: `[null]`, wantError: true},
		{name: "布尔值", values: `[true]`, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := "```json\n" + `{"query":{"source_code":"user","dimensions":[],"measures":[{"code":"user_count","aggregation":"count"}],"filters":[{"field":"deleted_at","operator":"eq","values":` + test.values + `}],"orders":[],"limit":100},"chart_type":"bar","title":"当前系统用户总数"}` + "\n```"
			var output dto.AskOutput
			err := json.Unmarshal([]byte(extractJSON(response)), &output)
			if (err != nil) != test.wantError {
				t.Fatalf("解析查询计划失败: %v, wantError %v", err, test.wantError)
			}
			if err != nil {
				return
			}
			filters := toProtoFilters(output.Query.Filters)
			if len(filters) != 1 || !slices.Equal(filters[0].Values, test.want) {
				t.Fatalf("过滤值 = %v, want %v", filters, test.want)
			}
		})
	}
}
