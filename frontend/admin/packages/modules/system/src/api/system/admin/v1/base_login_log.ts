import service from "@liujitcn/kratos-admin-core/request";
import type {
  BaseLoginLog,
  BaseLoginLogService,
  GetBaseLoginLogRequest,
  PageBaseLoginLogRequest,
  PageBaseLoginLogResponse
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_login_log";

const BASE_LOGIN_LOG_URL = "/v1/admin/base/login-log";

/** Admin 登录日志服务。 */
export class BaseLoginLogServiceImpl implements BaseLoginLogService {
  /** 查询登录日志分页列表。 */
  PageBaseLoginLog(request: PageBaseLoginLogRequest): Promise<PageBaseLoginLogResponse> {
    return service<PageBaseLoginLogRequest, PageBaseLoginLogResponse>({
      url: BASE_LOGIN_LOG_URL,
      method: "get",
      params: request
    });
  }

  /** 查询登录日志详情。 */
  GetBaseLoginLog(request: GetBaseLoginLogRequest): Promise<BaseLoginLog> {
    return service<GetBaseLoginLogRequest, BaseLoginLog>({
      url: `${BASE_LOGIN_LOG_URL}/${request.id}`,
      method: "get"
    });
  }
}

/** 默认登录日志服务。 */
export const defBaseLoginLogService = new BaseLoginLogServiceImpl();
