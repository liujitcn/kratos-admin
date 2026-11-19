package admin

import (
	"os"
	"strings"
	"testing"
)

// TestBaseUserSummaryRouteRegisteredBeforeIDRoute 验证 summary 静态路由先于动态 ID 路由注册。
func TestBaseUserSummaryRouteRegisteredBeforeIDRoute(t *testing.T) {
	path := "../../../../../api/gen/go/system/admin/v1/base_user_http.pb.go"
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	summary := strings.Index(text, `r.Handle("GET", "/api/v1/admin/base/user/summary"`)
	byID := strings.Index(text, `r.Handle("GET", "/api/v1/admin/base/user/{id}"`)
	if summary < 0 || byID < 0 {
		t.Fatalf("生成路由缺少 summary 或 id 路由: summary=%d id=%d", summary, byID)
	}
	if summary > byID {
		t.Fatalf("summary 路由必须先于动态 id 路由注册: summary=%d id=%d", summary, byID)
	}
}
