import service from "@liujitcn/kratos-admin-core/request";
import { normalizeBiDashboardPage } from "../../../../bi-response";
import type {
  BiDashboard,
  BiDashboardService,
  DeleteBiDashboardRequest,
  DeleteBiDashboardResponse,
  BiDashboardDataResponse,
  GetBiDashboardDataRequest,
  GetBiDashboardRequest,
  PageBiDashboardRequest,
  PageBiDashboardResponse,
  PublishBiDashboardRequest,
  SaveBiDashboardRequest
} from "../../../../rpc/bi/admin/v1/bi_dashboard";

const BI_DASHBOARD_URL = "/v1/admin/bi/dashboard";

/** BI 仪表盘管理服务实现。 */
export class BiDashboardServiceImpl implements BiDashboardService {
  /** 分页查询仪表盘。 */
  PageBiDashboard(request: PageBiDashboardRequest): Promise<PageBiDashboardResponse> {
    return service<PageBiDashboardRequest, PageBiDashboardResponse>({url: BI_DASHBOARD_URL, method: "get", params: request}).then(normalizeBiDashboardPage);
  }

  /** 查询仪表盘详情。 */
  GetBiDashboard(request: GetBiDashboardRequest): Promise<BiDashboard> {
    return service({url: `${BI_DASHBOARD_URL}/${request.id}`, method: "get"});
  }

  /** 保存仪表盘草稿。 */
  SaveBiDashboard(request: SaveBiDashboardRequest): Promise<BiDashboard> {
    return service({url: BI_DASHBOARD_URL, method: "post", data: request});
  }

  /** 发布仪表盘草稿。 */
  PublishBiDashboard(request: PublishBiDashboardRequest): Promise<BiDashboard> {
    return service({url: `${BI_DASHBOARD_URL}/${request.id}/publish`, method: "post", data: request});
  }

  /** 删除仪表盘。 */
  DeleteBiDashboard(request: DeleteBiDashboardRequest): Promise<DeleteBiDashboardResponse> {
    return service({url: `${BI_DASHBOARD_URL}/${request.id}`, method: "delete"});
  }

  /** 执行已发布仪表盘中的图表查询。 */
  GetBiDashboardData(request: GetBiDashboardDataRequest): Promise<BiDashboardDataResponse> {
    return service({url: `${BI_DASHBOARD_URL}/${request.id}/data`, method: "get", params: {version: request.version}});
  }
}

export const defBiDashboardService = new BiDashboardServiceImpl();
