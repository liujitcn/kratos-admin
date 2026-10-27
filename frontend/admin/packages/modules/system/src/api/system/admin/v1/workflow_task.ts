import service from "@liujitcn/kratos-admin-core/request";
import {
  type AuditWorkflowTaskRequest,
  type GetWorkflowTaskRequest,
  type GetWorkflowTaskResponse,
  type PageMyWorkflowTaskRequest,
  type PageMyWorkflowTaskResponse,
  type WorkflowTaskService
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/workflow_task";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";

const WORKFLOW_TASK_URL = "/v1/admin/workflow/task";

/** Admin工作流审批任务服务。 */
export class WorkflowTaskServiceImpl implements WorkflowTaskService {
  /** 查询我的审批任务分页列表。 */
  PageMyWorkflowTask(request: PageMyWorkflowTaskRequest): Promise<PageMyWorkflowTaskResponse> {
    return service<PageMyWorkflowTaskRequest, PageMyWorkflowTaskResponse>({
      url: `${WORKFLOW_TASK_URL}/mine`,
      method: "get",
      params: request
    });
  }

  /** 查询审批任务详情。 */
  GetWorkflowTask(request: GetWorkflowTaskRequest): Promise<GetWorkflowTaskResponse> {
    return service<GetWorkflowTaskRequest, GetWorkflowTaskResponse>({
      url: `${WORKFLOW_TASK_URL}/${request.id}`,
      method: "get"
    });
  }

  /** 审批任务办理。 */
  AuditWorkflowTask(request: AuditWorkflowTaskRequest): Promise<Empty> {
    return service<AuditWorkflowTaskRequest, Empty>({
      url: `${WORKFLOW_TASK_URL}/${request.task_id}/audit`,
      method: "put",
      data: request
    });
  }
}

export const defWorkflowTaskService = new WorkflowTaskServiceImpl();
