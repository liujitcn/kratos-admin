import service from "@liujitcn/kratos-admin-core/request";
import type {
  BaseDashboardDistributionResponse,
  GetBaseDashboardLoginDistributionRequest,
  GetBaseDashboardLoginTrendRequest,
  GetBaseDashboardOperationDistributionRequest,
  GetBaseDashboardOverviewRequest,
  BaseDashboardOverview,
  BaseDashboardService,
  BaseDashboardTrendResponse
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_dashboard";

const BASE_DASHBOARD_URL = "/v1/admin/base/dashboard";

/** Admin首页业务统计服务。 */
export class BaseDashboardServiceImpl implements BaseDashboardService {
  /** 查询首页概览统计。 */
  GetBaseDashboardOverview(request: GetBaseDashboardOverviewRequest): Promise<BaseDashboardOverview> {
    return service<GetBaseDashboardOverviewRequest, BaseDashboardOverview>({
      url: `${BASE_DASHBOARD_URL}/overview`,
      method: "get",
      params: request
    });
  }

  /** 查询登录趋势。 */
  GetBaseDashboardLoginTrend(request: GetBaseDashboardLoginTrendRequest): Promise<BaseDashboardTrendResponse> {
    return service<GetBaseDashboardLoginTrendRequest, BaseDashboardTrendResponse>({
      url: `${BASE_DASHBOARD_URL}/login-trend`,
      method: "get",
      params: request
    });
  }

  /** 查询操作动作分布。 */
  GetBaseDashboardOperationDistribution(request: GetBaseDashboardOperationDistributionRequest): Promise<BaseDashboardDistributionResponse> {
    return service<GetBaseDashboardOperationDistributionRequest, BaseDashboardDistributionResponse>({
      url: `${BASE_DASHBOARD_URL}/operation-distribution`,
      method: "get",
      params: request
    });
  }

  /** 查询登录结果分布。 */
  GetBaseDashboardLoginDistribution(request: GetBaseDashboardLoginDistributionRequest): Promise<BaseDashboardDistributionResponse> {
    return service<GetBaseDashboardLoginDistributionRequest, BaseDashboardDistributionResponse>({
      url: `${BASE_DASHBOARD_URL}/login-distribution`,
      method: "get",
      params: request
    });
  }
}

/** 默认首页业务统计服务实例。 */
export const defBaseDashboardService = new BaseDashboardServiceImpl();
