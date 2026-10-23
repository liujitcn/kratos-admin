import service from "@liujitcn/kratos-admin-core/request";
import type {
  BaseScriptLog,
  BaseScriptLogService,
  GetBaseScriptLogRequest,
  PageBaseScriptLogRequest,
  PageBaseScriptLogResponse
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_script_log";

const BASE_SCRIPT_LOG_URL = "/v1/admin/base/script-log";

/** Admin脚本执行日志服务。 */
export class BaseScriptLogServiceImpl implements BaseScriptLogService {
  /** 查询脚本执行日志分页列表。 */
  PageBaseScriptLog(request: PageBaseScriptLogRequest): Promise<PageBaseScriptLogResponse> {
    return service<PageBaseScriptLogRequest, PageBaseScriptLogResponse>({ url: BASE_SCRIPT_LOG_URL, method: "get", params: request });
  }

  /** 查询脚本执行日志详情。 */
  GetBaseScriptLog(request: GetBaseScriptLogRequest): Promise<BaseScriptLog> {
    return service<GetBaseScriptLogRequest, BaseScriptLog>({ url: `${BASE_SCRIPT_LOG_URL}/${request.id}`, method: "get" });
  }
}

export const defBaseScriptLogService = new BaseScriptLogServiceImpl();
