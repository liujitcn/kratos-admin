import service from "@/utils/request";
import {
  type BaseLog,
  type BaseLogService,
  type GetBaseLogRequest,
  type PageBaseLogRequest,
  type PageBaseLogResponse
} from "@/rpc/system/admin/v1/base_log";

const BASE_LOG_URL = "/v1/admin/base/log";

/** Admin系统日志服务 */
export class BaseLogServiceImpl implements BaseLogService {
  /** 查询系统日志分页列表 */
  PageBaseLog(request: PageBaseLogRequest): Promise<PageBaseLogResponse> {
    return service<PageBaseLogRequest, PageBaseLogResponse>({
      url: `${BASE_LOG_URL}`,
      method: "get",
      params: request
    });
  }

  /** 查询系统日志 */
  GetBaseLog(request: GetBaseLogRequest): Promise<BaseLog> {
    return service<GetBaseLogRequest, BaseLog>({
      url: `${BASE_LOG_URL}/${request.id}`,
      method: "get"
    });
  }
}

export const defBaseLogService = new BaseLogServiceImpl();
