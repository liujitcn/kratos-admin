import service from "@liujitcn/kratos-admin-core/request";
import type {
  BaseJobLog,
  BaseJobLogService,
  GetBaseJobLogRequest,
  PageBaseJobLogRequest,
  PageBaseJobLogResponse
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_job_log";

const BASE_JOB_LOG_URL = "/v1/admin/base/job-log";

/** Admin定时任务日志服务 */
export class BaseJobLogServiceImpl implements BaseJobLogService {
  /** 查询定时任务日志分页列表 */
  PageBaseJobLog(request: PageBaseJobLogRequest): Promise<PageBaseJobLogResponse> {
    return service<PageBaseJobLogRequest, PageBaseJobLogResponse>({
      url: BASE_JOB_LOG_URL,
      method: "get",
      params: request
    });
  }

  /** 查询定时任务日志 */
  GetBaseJobLog(request: GetBaseJobLogRequest): Promise<BaseJobLog> {
    return service<GetBaseJobLogRequest, BaseJobLog>({
      url: `${BASE_JOB_LOG_URL}/${request.id}`,
      method: "get"
    });
  }
}

export const defBaseJobLogService = new BaseJobLogServiceImpl();
