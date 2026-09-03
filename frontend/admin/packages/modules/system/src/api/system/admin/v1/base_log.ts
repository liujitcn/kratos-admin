import service from "@liujitcn/kratos-admin-core/request";
import type {
  BaseLogService,
  GetBaseLogTraceRequest,
  GetBaseLogTraceResponse
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_log";

const BASE_LOG_URL = "/v1/admin/base";

/** 公共日志聚合查询服务。 */
export class BaseLogServiceImpl implements BaseLogService {
  /** 查询同一请求或链路关联的审计时间线。 */
  GetBaseLogTrace(request: GetBaseLogTraceRequest): Promise<GetBaseLogTraceResponse> {
    return service<GetBaseLogTraceRequest, GetBaseLogTraceResponse>({
      url: `${BASE_LOG_URL}/log-trace`,
      method: "get",
      params: request
    });
  }
}

/** 默认公共日志聚合查询服务。 */
export const defBaseLogService = new BaseLogServiceImpl();
