package sse

import (
	"context"
	"sync"
	"time"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/codegen"
	"github.com/liujitcn/kratos-kit/transport/sse"
)

var _ sse.SSEStream = (*OpsMonitoring)(nil)

const (
	opsMonitoringStreamID = "system.admin.ops-monitoring"
	opsMonitoringTraffic  = "ops.traffic"
	opsMonitoringServices = "ops.services"
	opsMonitoringStorage  = "ops.storage"
	opsMonitoringNodes    = "ops.nodes"
)

// OpsMonitoring 描述运维监控的 SSE 流及其实时发布循环。
type OpsMonitoring struct {
	monitoring *biz.OpsMonitoringCase
	publisher  codegen.Publisher
	mu         sync.Mutex
	cancel     context.CancelFunc
}

// NewOpsMonitoring 创建运维监控 SSE 流。
func NewOpsMonitoring(monitoring *biz.OpsMonitoringCase) *OpsMonitoring {
	return &OpsMonitoring{monitoring: monitoring}
}

// SetPublisher 设置运维监控的 SSE 发布能力。
func (o *OpsMonitoring) SetPublisher(publisher codegen.Publisher) {
	if o == nil {
		return
	}
	o.mu.Lock()
	o.publisher = publisher
	o.mu.Unlock()
}

// ID 返回运维监控 SSE 流标识。
func (*OpsMonitoring) ID() string {
	return opsMonitoringStreamID
}

// Resolve 返回固定的运维监控传输流标识。
func (*OpsMonitoring) Resolve(string, int64) (string, error) {
	return opsMonitoringStreamID, nil
}

// Start 启动运维监控实时事件发布循环。
func (o *OpsMonitoring) Start(ctx context.Context) {
	if o == nil || o.monitoring == nil {
		return
	}
	o.mu.Lock()
	publisher := o.publisher
	if publisher == nil || o.cancel != nil {
		o.mu.Unlock()
		return
	}
	streamCtx, cancel := context.WithCancel(ctx)
	o.cancel = cancel
	o.mu.Unlock()
	go o.publish(streamCtx)
}

// Stop 停止运维监控实时事件发布循环。
func (o *OpsMonitoring) Stop() {
	if o == nil {
		return
	}
	o.mu.Lock()
	cancel := o.cancel
	o.cancel = nil
	o.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// publish 按固定间隔发布运维监控卡片数据。
func (o *OpsMonitoring) publish(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			o.publishSnapshot(ctx)
		}
	}
}

// publishSnapshot 发布一次局部监控卡片数据。
func (o *OpsMonitoring) publishSnapshot(ctx context.Context) {
	o.mu.Lock()
	publisher := o.publisher
	o.mu.Unlock()
	if publisher == nil {
		return
	}
	traffic, err := o.monitoring.GetOpsTraffic(ctx, &adminv1.GetOpsTrafficRequest{WindowMinutes: 15})
	if err == nil {
		publisher(ctx, opsMonitoringStreamID, opsMonitoringTraffic, traffic)
	}
	services := o.monitoring.GetOpsServices(ctx, &adminv1.GetOpsServicesRequest{})
	publisher(ctx, opsMonitoringStreamID, opsMonitoringServices, services)
	storage := o.monitoring.GetOpsStorage(ctx, &adminv1.GetOpsStorageRequest{})
	publisher(ctx, opsMonitoringStreamID, opsMonitoringStorage, storage)
	nodes := o.monitoring.GetOpsNodes(ctx, &adminv1.GetOpsNodesRequest{})
	publisher(ctx, opsMonitoringStreamID, opsMonitoringNodes, nodes)
}
