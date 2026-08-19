package logstream

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strconv"
	"strings"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
)

// Handler 将 slog 记录转换为实时控制台日志。
type Handler struct {
	hub    *Hub
	level  slog.Level
	attrs  []slog.Attr
	groups []string
}

// NewHandler 创建运行日志 slog Handler。
func NewHandler(hub *Hub, level slog.Level) *Handler {
	return &Handler{hub: hub, level: level}
}

// Enabled 判断指定日志级别是否需要进入实时控制台。
func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return h != nil && h.hub != nil && level >= h.level
}

// Handle 格式化并非阻塞写入一条实时日志。
func (h *Handler) Handle(_ context.Context, record slog.Record) error {
	if h == nil || h.hub == nil {
		return nil
	}
	timestamp := record.Time.Format("2006-01-02 15:04:05.000")
	level := runtimeLogLevel(record.Level)
	source := runtimeLogSource(record.PC)
	attributes := make([]string, 0, len(h.attrs)+record.NumAttrs())
	for _, attr := range h.attrs {
		appendRuntimeAttr(&attributes, h.groups, attr)
	}
	record.Attrs(func(attr slog.Attr) bool {
		appendRuntimeAttr(&attributes, h.groups, attr)
		return true
	})
	lineParts := []string{timestamp, level}
	if source != "" {
		lineParts = append(lineParts, source)
	}
	if record.Message != "" {
		lineParts = append(lineParts, record.Message)
	}
	lineParts = append(lineParts, attributes...)
	h.hub.Append(&adminv1.RuntimeLogEntry{
		Timestamp: timestamp,
		Level:     level,
		Source:    source,
		Message:   record.Message,
		Line:      strings.Join(lineParts, " "),
	})
	return nil
}

// WithAttrs 创建附加固定属性的运行日志 Handler。
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	cloned := *h
	cloned.attrs = append(append([]slog.Attr(nil), h.attrs...), attrs...)
	cloned.groups = append([]string(nil), h.groups...)
	return &cloned
}

// WithGroup 创建附加属性分组的运行日志 Handler。
func (h *Handler) WithGroup(name string) slog.Handler {
	cloned := *h
	cloned.attrs = append([]slog.Attr(nil), h.attrs...)
	cloned.groups = append(append([]string(nil), h.groups...), name)
	return &cloned
}

// runtimeLogLevel 将 slog 级别收敛为控制台筛选使用的四类级别。
func runtimeLogLevel(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return "ERROR"
	case level >= slog.LevelWarn:
		return "WARN"
	case level >= slog.LevelInfo:
		return "INFO"
	default:
		return "DEBUG"
	}
}

// runtimeLogSource 从 slog 程序计数器解析调用位置。
func runtimeLogSource(pc uintptr) string {
	if pc == 0 {
		return ""
	}
	frame, _ := runtime.CallersFrames([]uintptr{pc}).Next()
	if frame.File == "" {
		return ""
	}
	return fmt.Sprintf("%s:%d", frame.File, frame.Line)
}

// appendRuntimeAttr 展开分组属性并追加可读文本。
func appendRuntimeAttr(target *[]string, groups []string, attr slog.Attr) {
	attr.Value = attr.Value.Resolve()
	if attr.Equal(slog.Attr{}) {
		return
	}
	if attr.Value.Kind() == slog.KindGroup {
		nestedGroups := groups
		if attr.Key != "" {
			nestedGroups = append(append([]string(nil), groups...), attr.Key)
		}
		for _, nested := range attr.Value.Group() {
			appendRuntimeAttr(target, nestedGroups, nested)
		}
		return
	}
	keyParts := append([]string(nil), groups...)
	if attr.Key != "" {
		keyParts = append(keyParts, attr.Key)
	}
	if len(keyParts) == 0 {
		return
	}
	*target = append(*target, strings.Join(keyParts, ".")+"="+formatRuntimeValue(attr.Value))
}

// formatRuntimeValue 返回适合单行日志展示的属性值。
func formatRuntimeValue(value slog.Value) string {
	var text string
	switch value.Kind() {
	case slog.KindString:
		text = value.String()
	case slog.KindBool:
		text = strconv.FormatBool(value.Bool())
	case slog.KindInt64:
		text = strconv.FormatInt(value.Int64(), 10)
	case slog.KindUint64:
		text = strconv.FormatUint(value.Uint64(), 10)
	case slog.KindFloat64:
		text = strconv.FormatFloat(value.Float64(), 'f', -1, 64)
	case slog.KindDuration:
		text = value.Duration().String()
	case slog.KindTime:
		text = value.Time().Format("2006-01-02 15:04:05.000")
	default:
		text = fmt.Sprint(value.Any())
	}
	if strings.ContainsAny(text, " \t\r\n=\"") {
		return strconv.Quote(text)
	}
	return text
}
