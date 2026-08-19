package logstream

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
)

var supportedLevels = map[string]struct{}{
	"DEBUG": {},
	"INFO":  {},
	"WARN":  {},
	"ERROR": {},
}

// ParseLine 尽力提取常见文本或 JSON 字段，无法识别的格式仍保留完整日志行。
func ParseLine(rawLine string, truncated bool) *adminv1.RuntimeLogEntry {
	line := stripANSI(strings.ToValidUTF8(rawLine, "\uFFFD"))
	entry := &adminv1.RuntimeLogEntry{Line: line, IsTruncated: truncated}
	if parseJSONLine(entry) {
		return entry
	}
	parseTextLine(entry)
	return entry
}

// NormalizeLevel 将常见日志级别收敛为页面筛选使用的四类值。
func NormalizeLevel(level string) string {
	normalized := strings.ToUpper(strings.Trim(strings.TrimSpace(level), "[](){}<>:;,\"'"))
	if key, value, found := strings.Cut(normalized, "="); found && (key == "LEVEL" || key == "LVL" || key == "SEVERITY") {
		normalized = value
	}
	switch normalized {
	case "TRACE":
		return "DEBUG"
	case "WARNING":
		return "WARN"
	case "DPANIC", "PANIC", "FATAL":
		return "ERROR"
	default:
		return normalized
	}
}

// IsSupportedLevel 判断日志级别是否支持筛选。
func IsSupportedLevel(level string) bool {
	_, ok := supportedLevels[NormalizeLevel(level)]
	return ok
}

// parseJSONLine 从常见 JSON 日志键中提取展示字段。
func parseJSONLine(entry *adminv1.RuntimeLogEntry) bool {
	line := strings.TrimSpace(entry.GetLine())
	if len(line) < 2 || line[0] != '{' || line[len(line)-1] != '}' {
		return false
	}
	fields := make(map[string]any)
	if err := json.Unmarshal([]byte(line), &fields); err != nil {
		return false
	}
	entry.Timestamp = firstJSONText(fields, "timestamp", "time", "ts", "@timestamp")
	entry.Level = NormalizeLevel(firstJSONText(fields, "level", "lvl", "severity"))
	if !IsSupportedLevel(entry.GetLevel()) {
		entry.Level = ""
	}
	entry.Source = firstJSONText(fields, "source", "caller", "file")
	entry.Message = firstJSONText(fields, "message", "msg")
	if entry.GetMessage() == "" {
		entry.Message = entry.GetLine()
	}
	return true
}

// firstJSONText 返回第一个存在且非空的 JSON 字段文本。
func firstJSONText(fields map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := fields[key]
		if !ok || value == nil {
			continue
		}
		text := fmt.Sprint(value)
		if text != "" {
			return text
		}
	}
	return ""
}

// parseTextLine 从常见单行文本日志中提取时间、级别、来源和消息。
func parseTextLine(entry *adminv1.RuntimeLogEntry) {
	fields := strings.Fields(entry.GetLine())
	if len(fields) == 0 {
		return
	}
	offset := parseTextTimestamp(entry, fields)
	if offset < len(fields) {
		level := NormalizeLevel(fields[offset])
		if IsSupportedLevel(level) {
			entry.Level = level
			offset++
		}
	}
	if offset < len(fields) && looksLikeSource(fields[offset]) {
		entry.Source = strings.Trim(fields[offset], "[]()")
		offset++
	}
	if offset < len(fields) {
		entry.Message = strings.Join(fields[offset:], " ")
	} else {
		entry.Message = entry.GetLine()
	}
}

// parseTextTimestamp 识别常见的本地时间或 RFC3339 时间前缀。
func parseTextTimestamp(entry *adminv1.RuntimeLogEntry, fields []string) int {
	if len(fields) >= 2 && len(fields[0]) == len("2006-01-02") && strings.Contains(fields[1], ":") {
		entry.Timestamp = fields[0] + " " + fields[1]
		return 2
	}
	if _, err := time.Parse(time.RFC3339Nano, strings.Trim(fields[0], "[]")); err == nil {
		entry.Timestamp = strings.Trim(fields[0], "[]")
		return 1
	}
	return 0
}

// looksLikeSource 判断字段是否类似带行号的源码位置。
func looksLikeSource(value string) bool {
	trimmed := strings.Trim(value, "[]()")
	index := strings.LastIndexByte(trimmed, ':')
	if index <= 0 || index == len(trimmed)-1 {
		return false
	}
	_, err := strconv.Atoi(trimmed[index+1:])
	return err == nil
}

// stripANSI 移除终端颜色控制码，避免浏览器显示不可读转义字符。
func stripANSI(text string) string {
	if text == "" || !strings.Contains(text, "\x1b[") {
		return text
	}
	var builder strings.Builder
	builder.Grow(len(text))
	inEscape := false
	afterBracket := false
	for i := 0; i < len(text); i++ {
		character := text[i]
		if inEscape {
			if !afterBracket {
				if character == '[' {
					afterBracket = true
				}
				continue
			}
			if character >= '@' && character <= '~' {
				inEscape = false
				afterBracket = false
			}
			continue
		}
		if character == '\x1b' && i+1 < len(text) && text[i+1] == '[' {
			inEscape = true
			afterBracket = false
			continue
		}
		builder.WriteByte(character)
	}
	return builder.String()
}
