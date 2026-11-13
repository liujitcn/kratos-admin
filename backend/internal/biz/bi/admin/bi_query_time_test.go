package admin

import (
	"testing"
	"time"
)

// TestResolveRelativeTime 验证问数时间范围相对表达式的折叠与报错边界。
func TestResolveRelativeTime(t *testing.T) {
	now := time.Date(2026, 11, 13, 15, 30, 0, 0, time.Local)
	cases := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{name: "裸now折叠为当前时间", value: "now", want: "2026-11-13 15:30:00"},
		{name: "按天回退", value: "now-30d", want: "2026-10-14 15:30:00"},
		{name: "按小时回退", value: "now-24h", want: "2026-11-12 15:30:00"},
		{name: "按周回退", value: "now-2w", want: "2026-10-30 15:30:00"},
		{name: "按月回退", value: "now-1m", want: "2026-10-13 15:30:00"},
		{name: "大小写与空格容忍", value: " NOW-3D ", want: "2026-11-10 15:30:00"},
		{name: "绝对时间原样返回", value: "2026-01-01 00:00:00", want: "2026-01-01 00:00:00"},
		{name: "非法单位报错", value: "now-3x", wantErr: true},
		{name: "缺少数值报错", value: "now-d", wantErr: true},
		{name: "正向偏移报错", value: "now+3d", wantErr: true},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			got, err := resolveRelativeTime(item.value, now)
			if item.wantErr {
				if err == nil {
					t.Fatalf("期望报错，实际返回 %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("期望成功，实际报错: %v", err)
			}
			if got != item.want {
				t.Fatalf("期望 %q，实际 %q", item.want, got)
			}
		})
	}
}
