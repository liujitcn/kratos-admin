import service from "@liujitcn/kratos-admin-core/request";
import type {
  BasePermissionLog,
  BasePermissionLogService,
  GetBasePermissionLogRequest,
  PageBasePermissionLogRequest,
  PageBasePermissionLogResponse
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_permission_log";

const BASE_PERMISSION_LOG_URL = "/v1/admin/base/permission-log";

/** Admin 权限日志服务。 */
export class BasePermissionLogServiceImpl implements BasePermissionLogService {
  /** 查询权限日志分页列表。 */
  PageBasePermissionLog(request: PageBasePermissionLogRequest): Promise<PageBasePermissionLogResponse> {
    return service<PageBasePermissionLogRequest, PageBasePermissionLogResponse>({
      url: BASE_PERMISSION_LOG_URL,
      method: "get",
      params: request
    });
  }

  /** 查询权限日志详情。 */
  GetBasePermissionLog(request: GetBasePermissionLogRequest): Promise<BasePermissionLog> {
    return service<GetBasePermissionLogRequest, BasePermissionLog>({
      url: `${BASE_PERMISSION_LOG_URL}/${request.id}`,
      method: "get"
    });
  }
}

/** 默认权限日志服务。 */
export const defBasePermissionLogService = new BasePermissionLogServiceImpl();
