package sse

import coreSSE "github.com/liujitcn/kratos-core/sse"

// NewStreams 创建 Admin SSE 流集合。
func NewStreams(codegen *Codegen, opsMonitoring *OpsMonitoring) (coreSSE.Streams, func()) {
	return coreSSE.Streams{
			codegen, opsMonitoring,
		}, func() {
			if opsMonitoring != nil {
				opsMonitoring.Stop()
			}
		}
}
