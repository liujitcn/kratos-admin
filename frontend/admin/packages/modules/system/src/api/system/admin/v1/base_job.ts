import service from "@liujitcn/kratos-admin-core/request";
import {
  type BaseJobForm,
  type BaseJobService,
  type CreateBaseJobRequest,
  type DeleteBaseJobRequest,
  type ExecuteBaseJobRequest,
  type GetBaseJobRequest,
  type OptionBaseJobRequest,
  type PageBaseJobRequest,
  type PageBaseJobResponse,
  type SetBaseJobStatusRequest,
  type StartBaseJobRequest,
  type StopBaseJobRequest,
  type UpdateBaseJobRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_job";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";
import type { SelectOptionResponse } from "@liujitcn/kratos-admin-system/rpc/common/v1/common";

const BASE_JOB_URL = "/v1/admin/base/job";

/** Admin定时任务服务 */
export class BaseJobServiceImpl implements BaseJobService {
  /** 查询定时任务下拉选择 */
  OptionBaseJob(request: OptionBaseJobRequest): Promise<SelectOptionResponse> {
    return service<OptionBaseJobRequest, SelectOptionResponse>({
      url: `${BASE_JOB_URL}/option`,
      method: "get",
      params: request
    });
  }

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

}

export const defBaseJobService = new BaseJobServiceImpl();
