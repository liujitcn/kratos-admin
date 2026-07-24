import service from "@/utils/request";
import {
  type BaseJobForm,
  type BaseJobLog,
  type BaseJobService,
  type CreateBaseJobRequest,
  type DeleteBaseJobRequest,
  type ExecuteBaseJobRequest,
  type GetBaseJobLogRequest,
  type GetBaseJobRequest,
  type PageBaseJobLogRequest,
  type PageBaseJobLogResponse,
  type PageBaseJobRequest,
  type PageBaseJobResponse,
  type SetBaseJobStatusRequest,
  type StartBaseJobRequest,
  type StopBaseJobRequest,
  type UpdateBaseJobRequest
} from "@/rpc/system/admin/v1/base_job";
import type { Empty } from "@/rpc/google/protobuf/empty";

const BASE_JOB_URL = "/v1/admin/base/job";
const BASE_JOB_LOG_URL = "/v1/admin/base/job-log";

/** Admin定时任务服务 */
export class BaseJobServiceImpl implements BaseJobService {
  /** 查询定时任务分页列表 */
  PageBaseJob(request: PageBaseJobRequest): Promise<PageBaseJobResponse> {
    return service<PageBaseJobRequest, PageBaseJobResponse>({
      url: `${BASE_JOB_URL}`,
      method: "get",
      params: request
    });
  }

  /** 查询定时任务 */
  GetBaseJob(request: GetBaseJobRequest): Promise<BaseJobForm> {
    return service<GetBaseJobRequest, BaseJobForm>({
      url: `${BASE_JOB_URL}/${request.id}`,
      method: "get"
    });
  }

  /** 创建定时任务 */
  CreateBaseJob(request: CreateBaseJobRequest): Promise<Empty> {
    return service<BaseJobForm | undefined, Empty>({
      url: `${BASE_JOB_URL}`,
      method: "post",
      data: request.base_job
    });
  }

  /** 更新定时任务 */
  UpdateBaseJob(request: UpdateBaseJobRequest): Promise<Empty> {
    return service<BaseJobForm | undefined, Empty>({
      url: `${BASE_JOB_URL}/${request.base_job?.id ?? ""}`,
      method: "put",
      data: request.base_job
    });
  }

  /** 删除定时任务 */
  DeleteBaseJob(request: DeleteBaseJobRequest): Promise<Empty> {
    return service<DeleteBaseJobRequest, Empty>({
      url: `${BASE_JOB_URL}/${request.id}`,
      method: "delete"
    });
  }

  /** 设置状态 */
  SetBaseJobStatus(request: SetBaseJobStatusRequest): Promise<Empty> {
    return service<SetBaseJobStatusRequest, Empty>({
      url: `${BASE_JOB_URL}/${request.id}/status`,
      method: "put",
      data: request
    });
  }

  /** 启动任务 */
  StartBaseJob(request: StartBaseJobRequest): Promise<Empty> {
    return service<StartBaseJobRequest, Empty>({
      url: `${BASE_JOB_URL}/${request.id}/running`,
      method: "put",
      data: request
    });
  }

  /** 停止任务 */
  StopBaseJob(request: StopBaseJobRequest): Promise<Empty> {
    return service<StopBaseJobRequest, Empty>({
      url: `${BASE_JOB_URL}/${request.id}/running`,
      method: "delete",
      data: request
    });
  }

  /** 执行任务 */
  ExecuteBaseJob(request: ExecuteBaseJobRequest): Promise<Empty> {
    return service<ExecuteBaseJobRequest, Empty>({
      url: `${BASE_JOB_URL}/${request.id}/execution`,
      method: "post",
      data: request
    });
  }

  /** 查询定时任务日志分页列表 */
  PageBaseJobLog(request: PageBaseJobLogRequest): Promise<PageBaseJobLogResponse> {
    return service<PageBaseJobLogRequest, PageBaseJobLogResponse>({
      url: `${BASE_JOB_LOG_URL}`,
      method: "get",
      params: request
    });
  }

  /** 查询定时任务日志 */
  GetBaseJobLog(request: GetBaseJobLogRequest): Promise<BaseJobLog> {
    return service<GetBaseJobLogRequest, BaseJobLog>({
      url: `${BASE_JOB_LOG_URL}/${request.id}`,
      method: "get",
      params: request
    });
  }
}

export const defBaseJobService = new BaseJobServiceImpl();
