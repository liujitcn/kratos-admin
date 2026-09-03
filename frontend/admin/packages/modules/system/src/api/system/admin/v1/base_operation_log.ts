import service from "@liujitcn/kratos-admin-core/request";
import type {
  BaseOperationLog,
  BaseOperationLogService,
  GetBaseOperationLogRequest,
  PageBaseOperationLogRequest,
  PageBaseOperationLogResponse
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_operation_log";

const BASE_OPERATION_LOG_URL = "/v1/admin/base/operation-log";

/** Admin 业务操作日志服务。 */
export class BaseOperationLogServiceImpl implements BaseOperationLogService {
  /** 查询业务操作日志分页列表。 */
  PageBaseOperationLog(request: PageBaseOperationLogRequest): Promise<PageBaseOperationLogResponse> {
    return service<PageBaseOperationLogRequest, PageBaseOperationLogResponse>({
      url: BASE_OPERATION_LOG_URL,
      method: "get",
      params: request
    });
  }

  /** 查询业务操作日志详情。 */
  GetBaseOperationLog(request: GetBaseOperationLogRequest): Promise<BaseOperationLog> {
    return service<GetBaseOperationLogRequest, BaseOperationLog>({
      url: `${BASE_OPERATION_LOG_URL}/${request.id}`,
      method: "get"
    });
  }
}

/** 默认业务操作日志服务。 */
export const defBaseOperationLogService = new BaseOperationLogServiceImpl();
