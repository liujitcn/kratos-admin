import service from "@liujitcn/kratos-admin-core/request";
import type {
  BaseDataAccessLog,
  BaseDataAccessLogService,
  GetBaseDataAccessLogRequest,
  PageBaseDataAccessLogRequest,
  PageBaseDataAccessLogResponse
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_data_access_log";

const BASE_DATA_ACCESS_LOG_URL = "/v1/admin/base/data-access-log";

/** Admin 数据访问日志服务。 */
export class BaseDataAccessLogServiceImpl implements BaseDataAccessLogService {
  /** 查询数据访问日志分页列表。 */
  PageBaseDataAccessLog(request: PageBaseDataAccessLogRequest): Promise<PageBaseDataAccessLogResponse> {
    return service<PageBaseDataAccessLogRequest, PageBaseDataAccessLogResponse>({
      url: BASE_DATA_ACCESS_LOG_URL,
      method: "get",
      params: request
    });
  }

  /** 查询数据访问日志详情。 */
  GetBaseDataAccessLog(request: GetBaseDataAccessLogRequest): Promise<BaseDataAccessLog> {
    return service<GetBaseDataAccessLogRequest, BaseDataAccessLog>({
      url: `${BASE_DATA_ACCESS_LOG_URL}/${request.id}`,
      method: "get"
    });
  }
}

/** 默认数据访问日志服务。 */
export const defBaseDataAccessLogService = new BaseDataAccessLogServiceImpl();
