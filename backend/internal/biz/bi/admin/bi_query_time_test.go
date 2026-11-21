package admin

import (
	"slices"
	"testing"
	"time"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
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
		{name: "绝对时间交给日历解析", value: "2026-01-01 00:00:00"},
		{name: "非法单位报错", value: "now-3x", wantErr: true},
		{name: "缺少数值报错", value: "now-d", wantErr: true},
		{name: "正向偏移报错", value: "now+3d", wantErr: true},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			instant, relative, err := resolveRelativeTime(item.value, now)
			if item.wantErr {
				if err == nil {
					t.Fatalf("期望报错，实际返回 %v", instant)
				}
				return
			}
			if err != nil {
				t.Fatalf("期望成功，实际报错: %v", err)
			}
			if relative != (item.want != "") || relative && instant.Format(time.DateTime) != item.want {
				t.Fatalf("期望 %q，实际 %v，相对时间 %v", item.want, instant, relative)
			}
		})
	}
}

// TestBindYearToDate 跨年刷新时按报表时区重新计算同一份保存计划的时间边界。
func TestBindYearToDate(t *testing.T) {
	execution := &dto.CardExecution{ReportTimezone: "Asia/Shanghai", Parameters: []dto.ExecutionParameter{
		{Position: 1, Kind: "time_start"}, {Position: 2, Kind: "time_end"},
	}}
	spec := bi.QuerySpec{TimeRange: &bi.TimeRangeSpec{Field: "created", Start: "start_of_year", End: "now", EndExclusive: true}}
	for _, item := range []struct {
		now  time.Time
		want []any
	}{
		{time.Date(2026, time.December, 31, 15, 59, 59, 0, time.UTC), []any{"2026-01-01 00:00:00", "2026-12-31 23:59:59"}},
		{time.Date(2026, time.December, 31, 16, 0, 1, 0, time.UTC), []any{"2027-01-01 00:00:00", "2027-01-01 00:00:01"}},
	} {
		got, err := bindExecutionParameters(execution, spec, bi.Scope{}, 0, item.now)
		if err != nil || !slices.Equal(got, item.want) {
			t.Fatalf("时间 %s 得到 %v, %v，预期 %v", item.now, got, err, item.want)
		}
	}
}

// TestBindRelativeRangeAcrossDaylightSaving 验证回拨后的两个 01:30 能作为不同的实际时间边界。
func TestBindRelativeRangeAcrossDaylightSaving(t *testing.T) {
	now := time.Date(2026, time.November, 1, 6, 30, 0, 123456000, time.UTC)
	execution := &dto.CardExecution{Driver: "postgres", ReportTimezone: "America/New_York", TimeRangeInstant: true, Parameters: []dto.ExecutionParameter{
		{Position: 1, Kind: "time_start"}, {Position: 2, Kind: "time_end"},
	}}
	spec := bi.QuerySpec{TimeRange: &bi.TimeRangeSpec{Field: "created", Start: "now-1h", End: "now", EndExclusive: true}}
	got, err := bindExecutionParameters(execution, spec, bi.Scope{}, 0, now)
	want := []any{"2026-11-01 05:30:00.123456+00:00", "2026-11-01 06:30:00.123456+00:00"}
	if err != nil || !slices.Equal(got, want) {
		t.Fatalf("回拨区间应保留两个实际时刻，得到 %v, %v", got, err)
	}
	spec.TimeRange.Start, spec.TimeRange.End = "2026-11-01T01:30:00.123456-04:00", "2026-11-01T01:30:00.123456-05:00"
	got, err = bindExecutionParameters(execution, spec, bi.Scope{}, 0, now)
	if err != nil || !slices.Equal(got, want) {
		t.Fatalf("带偏移的固定区间得到 %v, %v，预期 %v", got, err, want)
	}
}

// TestRequireRequestedGrouping 验证月度拆分与订阅周期描述不会混淆。
func TestRequireRequestedGrouping(t *testing.T) {
	for _, question := range []string{
		"开通了每月续费的用户总数是多少",
		"按月付费的用户共有多少人",
		"有多少用户启用了按月查看账单的功能",
		"How many users have a monthly subscription?",
		"How many users pay by month?",
	} {
		if err := requireRequestedGrouping(question, bi.QuerySpec{}); err != nil {
			t.Errorf("总数问法 %q 被拒绝：%v", question, err)
		}
	}
	for _, question := range []string{"按月统计新增用户数", "每月分别新增了多少用户", "今年每个月新增多少用户", "Show a monthly breakdown of sales"} {
		if err := requireRequestedGrouping(question, bi.QuerySpec{}); err == nil {
			t.Errorf("明确的月度拆分 %q 没有要求月度分组", question)
		}
		spec := bi.QuerySpec{GroupingOptions: []bi.GroupingOption{{Field: "created", TimeGrain: "month"}}}
		if err := requireRequestedGrouping(question, spec); err != nil {
			t.Errorf("月度查询 %q 被拒绝：%v", question, err)
		}
	}
}
