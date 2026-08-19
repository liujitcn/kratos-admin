package admin

import (
	"context"
	"fmt"

	"github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
)

// OpsMonitoringService 提供运维监控查询接口。
type OpsMonitoringService struct {
	adminv1.UnimplementedOpsMonitoringServiceServer
	opsMonitoringCase *biz.OpsMonitoringCase
}

// NewOpsMonitoringService 创建运维监控服务。
func NewOpsMonitoringService(opsMonitoringCase *biz.OpsMonitoringCase) *OpsMonitoringService {
	return &OpsMonitoringService{opsMonitoringCase: opsMonitoringCase}
}

// GetOpsRuntime 查询当前进程运行信息。
func (s *OpsMonitoringService) GetOpsRuntime(ctx context.Context, req *adminv1.GetOpsRuntimeRequest) (*adminv1.OpsRuntime, error) {
	return s.opsMonitoringCase.GetOpsRuntime(ctx, req), nil
}

// GetOpsTraffic 查询请求流量与延迟趋势。
func (s *OpsMonitoringService) GetOpsTraffic(ctx context.Context, req *adminv1.GetOpsTrafficRequest) (*adminv1.OpsTrafficResponse, error) {
	traffic, err := s.opsMonitoringCase.GetOpsTraffic(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("GetOpsTraffic %v", err))
		return nil, wrapOpsMonitoringError(err, "查询流量监控失败")
	}
	return traffic, nil
}

// GetOpsServices 查询服务和外部依赖状态。
func (s *OpsMonitoringService) GetOpsServices(ctx context.Context, req *adminv1.GetOpsServicesRequest) (*adminv1.OpsServicesResponse, error) {
	return s.opsMonitoringCase.GetOpsServices(ctx, req), nil
}

// GetOpsStorage 查询数据库和缓存状态。
func (s *OpsMonitoringService) GetOpsStorage(ctx context.Context, req *adminv1.GetOpsStorageRequest) (*adminv1.OpsStorageResponse, error) {
	return s.opsMonitoringCase.GetOpsStorage(ctx, req), nil
}

// GetOpsEndpoints 查询接口请求摘要。
func (s *OpsMonitoringService) GetOpsEndpoints(ctx context.Context, req *adminv1.GetOpsEndpointsRequest) (*adminv1.OpsEndpointsResponse, error) {
	endpoints, err := s.opsMonitoringCase.GetOpsEndpoints(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("GetOpsEndpoints %v", err))
		return nil, wrapOpsMonitoringError(err, "查询接口监控失败")
	}
	return endpoints, nil
}

// GetOpsNodes 查询实例资源状态。
func (s *OpsMonitoringService) GetOpsNodes(ctx context.Context, req *adminv1.GetOpsNodesRequest) (*adminv1.OpsNodesResponse, error) {
	return s.opsMonitoringCase.GetOpsNodes(ctx, req), nil
}

// GetOpsAlerts 查询窗口内告警事件。
func (s *OpsMonitoringService) GetOpsAlerts(ctx context.Context, req *adminv1.GetOpsAlertsRequest) (*adminv1.OpsAlertsResponse, error) {
	alerts, err := s.opsMonitoringCase.GetOpsAlerts(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("GetOpsAlerts %v", err))
		return nil, wrapOpsMonitoringError(err, "查询告警监控失败")
	}
	return alerts, nil
}

// wrapOpsMonitoringError 统一封装监控查询错误。
func wrapOpsMonitoringError(err error, message string) error {
	return errorsx.WrapInternal(err, message)
}
