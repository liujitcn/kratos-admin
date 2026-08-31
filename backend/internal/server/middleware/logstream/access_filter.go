package logstream

import (
	"context"
	"log/slog"
	"strings"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/go-kratos/kratos/v3/middleware"
)

const runtimeLogOperationPrefix = "/system.admin.v1.RuntimeLogService/"

// NewRuntimeAccessFilteredLogger 创建忽略运行日志浏览接口访问记录的日志器。
func NewRuntimeAccessFilteredLogger(logger *slog.Logger) *slog.Logger {
	if logger == nil {
		return nil
	}
	return slog.New(&runtimeAccessFilter{handler: logger.Handler()})
}

// InstallRuntimeAccessFilter 安装一次运行日志访问过滤器，避免重复包装当前日志器。
func InstallRuntimeAccessFilter() {
	logger := log.Default()
	if logger == nil {
		return
	}
	if _, ok := logger.Handler().(*runtimeAccessFilter); ok {
		return
	}
	log.SetDefault(NewRuntimeAccessFilteredLogger(logger))
}

// RuntimeAccessMiddleware 在运行日志接口执行前安装访问日志过滤器。
func RuntimeAccessMiddleware() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			InstallRuntimeAccessFilter()
			return handler(ctx, req)
		}
	}
}

// runtimeAccessFilter 在不改变原日志格式和输出目标的前提下过滤运行日志查询反馈。
type runtimeAccessFilter struct {
	handler slog.Handler
}

// Enabled 判断底层日志器是否启用指定级别。
func (f *runtimeAccessFilter) Enabled(ctx context.Context, level slog.Level) bool {
	return f != nil && f.handler != nil && f.handler.Enabled(ctx, level)
}

// Handle 过滤运行日志接口访问日志及其异步持久化 SQL 日志。
func (f *runtimeAccessFilter) Handle(ctx context.Context, record slog.Record) error {
	if f == nil || f.handler == nil {
		return nil
	}
	if isRuntimeAccessRecord(record.Message) {
		return nil
	}
	return f.handler.Handle(ctx, record)
}

// WithAttrs 返回带固定属性的过滤日志器。
func (f *runtimeAccessFilter) WithAttrs(attrs []slog.Attr) slog.Handler {
	if f == nil || f.handler == nil {
		return f
	}
	return &runtimeAccessFilter{handler: f.handler.WithAttrs(attrs)}
}

// WithGroup 返回带属性分组的过滤日志器。
func (f *runtimeAccessFilter) WithGroup(name string) slog.Handler {
	if f == nil || f.handler == nil {
		return f
	}
	return &runtimeAccessFilter{handler: f.handler.WithGroup(name)}
}

// isRuntimeAccessRecord 判断日志是否由运行日志浏览请求产生。
func isRuntimeAccessRecord(message string) bool {
	if strings.Contains(message, runtimeLogOperationPrefix) || strings.Contains(message, "/api/v1/admin/runtime-log/") {
		return true
	}
	for _, method := range []string{"ListRuntimeLogFiles", "ReadRuntimeLogFile", "OpenRuntimeConsole", "DownloadRuntimeLogFile"} {
		if strings.Contains(message, method) {
			return true
		}
	}
	return false
}
