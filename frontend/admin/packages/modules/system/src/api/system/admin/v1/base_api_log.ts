import service from "@liujitcn/kratos-admin-core/request";
import type {
  BaseApiLog,
  BaseApiLogService,
  GetBaseApiLogRequest,
  PageBaseApiLogRequest,
  PageBaseApiLogResponse
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_api_log";

const BASE_API_LOG_URL = "/v1/admin/base/api-log";

/** Admin API 访问日志服务。 */
export class BaseApiLogServiceImpl implements BaseApiLogService {
  /** 查询 API 访问日志分页列表。 */
  PageBaseApiLog(request: PageBaseApiLogRequest): Promise<PageBaseApiLogResponse> {
    return service<PageBaseApiLogRequest, PageBaseApiLogResponse>({
      url: BASE_API_LOG_URL,
      method: "get",
      params: request
    });
  }

  /** 查询 API 访问日志详情。 */
  GetBaseApiLog(request: GetBaseApiLogRequest): Promise<BaseApiLog> {
    return service<GetBaseApiLogRequest, BaseApiLog>({
      url: `${BASE_API_LOG_URL}/${request.id}`,
      method: "get"
    });
  }
}

/** 默认 API 访问日志服务。 */
export const defBaseApiLogService = new BaseApiLogServiceImpl();
