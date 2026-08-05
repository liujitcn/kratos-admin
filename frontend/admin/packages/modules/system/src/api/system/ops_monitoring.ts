import service from "@liujitcn/kratos-admin-core/request";
import type {
  GetOpsAlertsRequest,
  GetOpsEndpointsRequest,
  GetOpsNodesRequest,
  GetOpsRuntimeRequest,
  GetOpsServicesRequest,
  GetOpsStorageRequest,
  GetOpsTrafficRequest,
  OpsAlertsResponse,
  OpsEndpointsResponse,
  OpsNodesResponse,
  OpsRuntime,
  OpsServicesResponse,
  OpsStorageResponse,
  OpsTrafficResponse,
  OpsMonitoringService
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/ops_monitoring";

const OPS_MONITORING_URL = "/v1/admin/ops-monitoring";

/** Admin运维监控服务。 */
export class OpsMonitoringServiceImpl implements OpsMonitoringService {
  /** 查询当前进程运行信息。 */
  GetOpsRuntime(request: GetOpsRuntimeRequest): Promise<OpsRuntime> {
    return service<GetOpsRuntimeRequest, OpsRuntime>({
      url: `${OPS_MONITORING_URL}/runtime`,
      method: "get",
      params: request
    });
  }

  /** 查询请求流量与延迟趋势。 */
  GetOpsTraffic(request: GetOpsTrafficRequest): Promise<OpsTrafficResponse> {
    return service<GetOpsTrafficRequest, OpsTrafficResponse>({
      url: `${OPS_MONITORING_URL}/traffic`,
      method: "get",
      params: request
    });
  }

  /** 查询服务和外部依赖状态。 */
  GetOpsServices(request: GetOpsServicesRequest): Promise<OpsServicesResponse> {
    return service<GetOpsServicesRequest, OpsServicesResponse>({
      url: `${OPS_MONITORING_URL}/services`,
      method: "get",
      params: request
    });
  }

  /** 查询数据库和缓存状态。 */
  GetOpsStorage(request: GetOpsStorageRequest): Promise<OpsStorageResponse> {
    return service<GetOpsStorageRequest, OpsStorageResponse>({
      url: `${OPS_MONITORING_URL}/storage`,
      method: "get",
      params: request
    });
  }

  /** 查询接口请求摘要。 */
  GetOpsEndpoints(request: GetOpsEndpointsRequest): Promise<OpsEndpointsResponse> {
    return service<GetOpsEndpointsRequest, OpsEndpointsResponse>({
      url: `${OPS_MONITORING_URL}/endpoints`,
      method: "get",
      params: request
    });
  }

  /** 查询实例资源状态。 */
  GetOpsNodes(request: GetOpsNodesRequest): Promise<OpsNodesResponse> {
    return service<GetOpsNodesRequest, OpsNodesResponse>({
      url: `${OPS_MONITORING_URL}/nodes`,
      method: "get",
      params: request
    });
  }

  /** 查询窗口内告警事件。 */
  GetOpsAlerts(request: GetOpsAlertsRequest): Promise<OpsAlertsResponse> {
    return service<GetOpsAlertsRequest, OpsAlertsResponse>({
      url: `${OPS_MONITORING_URL}/alerts`,
      method: "get",
      params: request
    });
  }
}

export const defOpsMonitoringService = new OpsMonitoringServiceImpl();
