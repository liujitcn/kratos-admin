package admin

import (
	"context"
	"fmt"
	"sync"
	"time"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	coreSSE "github.com/liujitcn/kratos-admin/backend/core/pkg/sse"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"

	"github.com/go-kratos/kratos/v3/log"
)

// OpsMonitoringService 提供运维监控查询接口。
type OpsMonitoringService struct {
	systemadminv1.UnimplementedOpsMonitoringServiceServer
	opsMonitoringCase *biz.OpsMonitoringCase
	publisherMu       sync.RWMutex
	publisher         *coreSSE.Publisher
	streamCancel      context.CancelFunc
}

// NewOpsMonitoringService 创建运维监控服务。
func NewOpsMonitoringService(opsMonitoringCase *biz.OpsMonitoringCase) *OpsMonitoringService {
	return &OpsMonitoringService{opsMonitoringCase: opsMonitoringCase}
}

// GetOpsRuntime 查询当前进程运行信息。
func (s *OpsMonitoringService) GetOpsRuntime(ctx context.Context, req *systemadminv1.GetOpsRuntimeRequest) (*systemadminv1.OpsRuntime, error) {
	return s.opsMonitoringCase.GetOpsRuntime(ctx, req), nil
}

// GetOpsTraffic 查询请求流量与延迟趋势。
func (s *OpsMonitoringService) GetOpsTraffic(ctx context.Context, req *systemadminv1.GetOpsTrafficRequest) (*systemadminv1.OpsTrafficResponse, error) {
	traffic, err := s.opsMonitoringCase.GetOpsTraffic(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("GetOpsTraffic %v", err))
		return nil, wrapOpsMonitoringError(err, "查询流量监控失败")
	}
	return traffic, nil
}

// GetOpsServices 查询服务和外部依赖状态。
func (s *OpsMonitoringService) GetOpsServices(ctx context.Context, req *systemadminv1.GetOpsServicesRequest) (*systemadminv1.OpsServicesResponse, error) {
	return s.opsMonitoringCase.GetOpsServices(ctx, req), nil
}

// GetOpsStorage 查询数据库和缓存状态。
func (s *OpsMonitoringService) GetOpsStorage(ctx context.Context, req *systemadminv1.GetOpsStorageRequest) (*systemadminv1.OpsStorageResponse, error) {
	return s.opsMonitoringCase.GetOpsStorage(ctx, req), nil
}

// GetOpsEndpoints 查询接口请求摘要。
func (s *OpsMonitoringService) GetOpsEndpoints(ctx context.Context, req *systemadminv1.GetOpsEndpointsRequest) (*systemadminv1.OpsEndpointsResponse, error) {
	endpoints, err := s.opsMonitoringCase.GetOpsEndpoints(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("GetOpsEndpoints %v", err))
		return nil, wrapOpsMonitoringError(err, "查询接口监控失败")
	}
	return endpoints, nil
}

// GetOpsNodes 查询实例资源状态。
func (s *OpsMonitoringService) GetOpsNodes(ctx context.Context, req *systemadminv1.GetOpsNodesRequest) (*systemadminv1.OpsNodesResponse, error) {
	return s.opsMonitoringCase.GetOpsNodes(ctx, req), nil
}

// GetOpsAlerts 查询窗口内告警事件。
func (s *OpsMonitoringService) GetOpsAlerts(ctx context.Context, req *systemadminv1.GetOpsAlertsRequest) (*systemadminv1.OpsAlertsResponse, error) {
	alerts, err := s.opsMonitoringCase.GetOpsAlerts(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("GetOpsAlerts %v", err))
		return nil, wrapOpsMonitoringError(err, "查询告警监控失败")
	}
	return alerts, nil
}

// SetSSEPublisher 设置运维监控实时事件发布器。
func (s *OpsMonitoringService) SetSSEPublisher(publisher *coreSSE.Publisher) {
	s.publisherMu.Lock()
	s.publisher = publisher
	s.publisherMu.Unlock()
}

// StartOpsMonitoringStream 启动运维监控实时事件发布循环。
func (s *OpsMonitoringService) StartOpsMonitoringStream(ctx context.Context) error {
	s.publisherMu.Lock()
	if s.publisher == nil || s.streamCancel != nil {
		s.publisherMu.Unlock()
		return nil
	}
	streamCtx, cancel := context.WithCancel(ctx)
	s.streamCancel = cancel
	s.publisherMu.Unlock()
	go s.publishOpsMonitoringStream(streamCtx)
	return nil
}

// StopOpsMonitoringStream 停止运维监控实时事件发布循环。
func (s *OpsMonitoringService) StopOpsMonitoringStream(context.Context) error {
	s.publisherMu.Lock()
	cancel := s.streamCancel
	s.streamCancel = nil
	s.publisherMu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

// publishOpsMonitoringStream 定时发布局部监控卡片数据。
func (s *OpsMonitoringService) publishOpsMonitoringStream(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.publishOpsMonitoringSnapshot(ctx)
		}
	}
}

// publishOpsMonitoringSnapshot 发布一次局部监控卡片数据。
func (s *OpsMonitoringService) publishOpsMonitoringSnapshot(ctx context.Context) {
	s.publisherMu.RLock()
	publisher := s.publisher
	s.publisherMu.RUnlock()
	if publisher == nil {
		return
	}
	traffic, err := s.opsMonitoringCase.GetOpsTraffic(ctx, &systemadminv1.GetOpsTrafficRequest{WindowMinutes: 15})
	if err == nil {
		publisher.TryPublishJSON(ctx, OpsMonitoringSSEStreamID, OpsMonitoringSSETraffic, traffic)
	}
	services := s.opsMonitoringCase.GetOpsServices(ctx, &systemadminv1.GetOpsServicesRequest{})
	publisher.TryPublishJSON(ctx, OpsMonitoringSSEStreamID, OpsMonitoringSSEServices, services)
	storage := s.opsMonitoringCase.GetOpsStorage(ctx, &systemadminv1.GetOpsStorageRequest{})
	publisher.TryPublishJSON(ctx, OpsMonitoringSSEStreamID, OpsMonitoringSSEStorage, storage)
	nodes := s.opsMonitoringCase.GetOpsNodes(ctx, &systemadminv1.GetOpsNodesRequest{})
	publisher.TryPublishJSON(ctx, OpsMonitoringSSEStreamID, OpsMonitoringSSENodes, nodes)
}

// wrapOpsMonitoringError 统一封装监控查询错误。
func wrapOpsMonitoringError(err error, message string) error {
	return errorsx.WrapInternal(err, message)
}
