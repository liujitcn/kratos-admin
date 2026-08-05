package admin

import coreSSE "github.com/liujitcn/kratos-admin/backend/core/pkg/sse"

const (
	// OpsMonitoringSSEStreamID 表示运维监控实时数据流。
	OpsMonitoringSSEStreamID = "system.admin.ops-monitoring"
	// OpsMonitoringSSETraffic 表示流量监控事件。
	OpsMonitoringSSETraffic = "ops.traffic"
	// OpsMonitoringSSEServices 表示服务状态事件。
	OpsMonitoringSSEServices = "ops.services"
	// OpsMonitoringSSEStorage 表示存储状态事件。
	OpsMonitoringSSEStorage = "ops.storage"
	// OpsMonitoringSSENodes 表示实例资源事件。
	OpsMonitoringSSENodes = "ops.nodes"
)

// OpsMonitoringSSEStream 声明运维监控固定 SSE 流。
type OpsMonitoringSSEStream struct{}

var _ coreSSE.Stream = (*OpsMonitoringSSEStream)(nil)

// NewOpsMonitoringSSEStream 创建运维监控 SSE 流声明。
func NewOpsMonitoringSSEStream() *OpsMonitoringSSEStream {
	return &OpsMonitoringSSEStream{}
}

// ID 返回运维监控 SSE 流标识。
func (*OpsMonitoringSSEStream) ID() string {
	return OpsMonitoringSSEStreamID
}

// Resolve 返回固定的运维监控传输流标识。
func (*OpsMonitoringSSEStream) Resolve(string, int64) (string, error) {
	return OpsMonitoringSSEStreamID, nil
}
