package logstream

import (
	"sync"
)

var (
	runtimeLoggingOnce sync.Once
	runtimeLoggingErr  error
)

// InitializeRuntimeLogging 初始化进程级实时日志采集和运行日志访问过滤。
func InitializeRuntimeLogging() error {
	runtimeLoggingOnce.Do(func() {
		runtimeLoggingErr = StartConsoleCapture(DefaultHub())
		InstallRuntimeAccessFilter()
	})
	return runtimeLoggingErr
}
