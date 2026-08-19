package logstream

import (
	"log/slog"
	"strings"

	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	kitlogger "github.com/liujitcn/kratos-kit/logger"
	zaplogger "github.com/liujitcn/kratos-kit/logger/zap"
)

const runtimeLoggerType kitlogger.Type = "runtime-zap"

// init 注册保留 Zap 输出并附加实时日志 Hub 的日志工厂。
func init() {
	_ = kitlogger.Register(runtimeLoggerType, newRuntimeLogger)
}

// newRuntimeLogger 创建同时写入 Zap 和实时控制台 Hub 的日志实例。
func newRuntimeLogger(cfg *configv1.Logger) (*slog.Logger, error) {
	baseLogger, err := zaplogger.NewLogger(cfg)
	if err != nil || baseLogger == nil {
		return baseLogger, err
	}
	return kitlogger.NewMultiLogger(baseLogger.Handler(), NewHandler(DefaultHub(), configuredLevel(cfg.GetZap().GetLevel()))), nil
}

// configuredLevel 将 Zap 配置级别转换为 slog 最低级别。
func configuredLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error", "dpanic", "panic", "fatal":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
