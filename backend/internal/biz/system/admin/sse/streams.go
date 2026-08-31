package sse

import "github.com/liujitcn/kratos-core/sse"

// NewStreams 创建 Admin SSE 流集合。
func NewStreams(codegen *Codegen, notification *Notification, opsMonitoring *OpsMonitoring, runtimeConsole *RuntimeConsole) (sse.Streams, func()) {
	return sse.Streams{
			codegen, notification, opsMonitoring, runtimeConsole,
		}, func() {
			if opsMonitoring != nil {
				opsMonitoring.Stop()
			}
			if runtimeConsole != nil {
				runtimeConsole.Stop()
			}
		}
}
