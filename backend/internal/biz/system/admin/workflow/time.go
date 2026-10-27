package workflow

import "time"

// FormatTime 格式化工作流展示时间，零值返回空串。
func FormatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

// FormatOptionalTime 格式化可空时间，空指针返回空字符串。
func FormatOptionalTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return FormatTime(*t)
}
