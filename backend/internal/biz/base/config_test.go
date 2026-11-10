package biz

import (
	"testing"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-core/biz"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/bootstrap"
)

// TestRuntimeConfigHostVersion 验证版本只来自宿主上下文，不使用数据库配置或底座版本兜底。
func TestRuntimeConfigHostVersion(t *testing.T) {
	for _, scenario := range []struct {
		name    string
		context *bootstrap.Context
		version string
	}{
		{
			name: "cloud-scp",
			context: bootstrap.NewContext(t.Context(), &configv1.AppInfo{
				Project: "cloud", AppId: "scp", Version: "0.0.6",
			}),
			version: "0.0.6",
		},
		{name: "无宿主上下文"},
		{name: "无宿主应用信息", context: new(bootstrap.Context)},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			service := &ConfigCase{BaseCase: &biz.BaseCase{Context: scenario.context}}
			configs := service.appendRuntimeConfig([]*basev1.ConfigItem{{Key: "version", Value: "数据库配置版本"}}, false)
			var version string
			var count int
			for _, item := range configs {
				if item.GetKey() == "version" {
					version = item.GetValue()
					count++
				}
			}
			if count != 1 || version != scenario.version {
				t.Fatalf("版本应取宿主值 %q，实际 %q，共 %d 项", scenario.version, version, count)
			}
		})
	}
}
