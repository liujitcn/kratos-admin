package automation

import (
	"slices"
	"time"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-core/errorsx"
)

// Preview 使用统一日历算法计算严格晚于指定时间的未来三次以内触发时间。
func Preview(schedule *basev1.AiAutomationSchedule, after time.Time, count int) ([]time.Time, error) {
	if count < 1 || count > 3 || !slices.Contains([]string{"daily", "weekdays", "weekly", "once"}, schedule.GetFrequency()) {
		return nil, errorsx.InvalidArgument("自动化频率不合法")
	}
	if schedule.GetTimezone() == "" || schedule.GetTimezone() == "Local" {
		return nil, errorsx.InvalidArgument("请选择有效的 IANA 时区")
	}
	location, err := time.LoadLocation(schedule.GetTimezone())
	if err != nil {
		return nil, errorsx.InvalidArgument("时区不合法").WithCause(err)
	}
	if schedule.GetFrequency() == "once" {
		var date time.Time
		date, err = time.Parse("2006-01-02T15:04", schedule.GetOnceAt())
		if err != nil {
			return nil, errorsx.InvalidArgument("单次日期时间不合法").WithCause(err)
		}
		candidate := firstLocalTime(location, date.Year(), date.Month(), date.Day(), date.Hour(), date.Minute())
		if candidate.IsZero() || !candidate.After(after) {
			return nil, errorsx.InvalidArgument("单次时间必须存在且晚于当前时间")
		}
		return []time.Time{candidate.UTC()}, nil
	}
	var clock time.Time
	clock, err = time.Parse("15:04", schedule.GetTime())
	if err != nil || clock.Format("15:04") != schedule.GetTime() {
		return nil, errorsx.InvalidArgument("时间必须为 HH:mm")
	}
	days := schedule.GetWeekdays()
	if schedule.GetFrequency() == "weekly" {
		seen := make(map[int32]bool)
		for _, day := range days {
			if day < 1 || day > 7 || seen[day] {
				return nil, errorsx.InvalidArgument("星期必须为不重复的周一至周日")
			}
			seen[day] = true
		}
		if len(days) == 0 {
			return nil, errorsx.InvalidArgument("至少选择一个星期")
		}
	}
	local := after.In(location)
	calendar := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
	result := make([]time.Time, 0, count)
	for day := 0; len(result) < count && day < 32; day++ {
		date := calendar.AddDate(0, 0, day)
		weekday := int32(date.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		if schedule.GetFrequency() == "weekdays" && weekday > 5 || schedule.GetFrequency() == "weekly" && !slices.Contains(days, weekday) {
			continue
		}
		candidate := firstLocalTime(location, date.Year(), date.Month(), date.Day(), clock.Hour(), clock.Minute())
		if !candidate.IsZero() && candidate.After(after) {
			result = append(result, candidate.UTC())
		}
	}
	if len(result) != count {
		return nil, errorsx.InvalidArgument("计划无法生成有效的未来时间")
	}
	return result, nil
}

// firstLocalTime 枚举时区切换前后的偏移，跳过缺失本地时间并固定选择重叠时间的第一次。
func firstLocalTime(location *time.Location, year int, month time.Month, day, hour, minute int) time.Time {
	wall := time.Date(year, month, day, hour, minute, 0, 0, time.UTC)
	anchor := time.Date(year, month, day, 12, 0, 0, 0, location)
	var earliest time.Time
	for _, delta := range []time.Duration{-48 * time.Hour, 0, 48 * time.Hour} {
		_, offset := anchor.Add(delta).Zone()
		candidate := wall.Add(-time.Duration(offset) * time.Second).In(location)
		if candidate.Year() != year || candidate.Month() != month || candidate.Day() != day || candidate.Hour() != hour || candidate.Minute() != minute {
			continue
		}
		if earliest.IsZero() || candidate.Before(earliest) {
			earliest = candidate
		}
	}
	return earliest
}
