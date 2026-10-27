import service from "@liujitcn/kratos-admin-core/request";
import {
  type RetryWorkflowScriptRequest,
  type GetWorkflowInstanceRequest,
  type GetWorkflowInstanceResponse,
  type PageWorkflowInstanceRequest,
  type PageWorkflowInstanceResponse,
  type SubmitWorkflowInstanceRequest,
  type SubmitWorkflowInstanceResponse,
  type WithdrawWorkflowInstanceRequest,
  type WorkflowInstanceService
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/workflow_instance";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";

const WORKFLOW_INSTANCE_URL = "/v1/admin/workflow/instance";

/** Admin工作流流程实例服务。 */
export class WorkflowInstanceServiceImpl implements WorkflowInstanceService {
  /** 提交审批申请。 */
  SubmitWorkflowInstance(request: SubmitWorkflowInstanceRequest): Promise<SubmitWorkflowInstanceResponse> {
    return service<SubmitWorkflowInstanceRequest, SubmitWorkflowInstanceResponse>({
      url: WORKFLOW_INSTANCE_URL,
      method: "post",
      data: request
    });
  }

  /** 撤回审批申请。 */
  WithdrawWorkflowInstance(request: WithdrawWorkflowInstanceRequest): Promise<Empty> {
    return service<WithdrawWorkflowInstanceRequest, Empty>({
      url: `${WORKFLOW_INSTANCE_URL}/${request.id}/withdraw`,
      method: "put",
      data: request
    });
  }

  /** 重试实例的失败脚本执行。 */
  RetryWorkflowScript(request: RetryWorkflowScriptRequest): Promise<Empty> {
    return service<RetryWorkflowScriptRequest, Empty>({
      url: `${WORKFLOW_INSTANCE_URL}/${request.instance_id}/script-run/${request.id}/retry`,
      method: "put",
      data: request
    });
  }

  /** 查询流程实例分页列表。 */
  PageWorkflowInstance(request: PageWorkflowInstanceRequest): Promise<PageWorkflowInstanceResponse> {
    return service<PageWorkflowInstanceRequest, PageWorkflowInstanceResponse>({
      url: WORKFLOW_INSTANCE_URL,
      method: "get",
      params: request
    });
  }

  /** 查询流程实例详情。 */
  GetWorkflowInstance(request: GetWorkflowInstanceRequest): Promise<GetWorkflowInstanceResponse> {
    return service<GetWorkflowInstanceRequest, GetWorkflowInstanceResponse>({
      url: `${WORKFLOW_INSTANCE_URL}/${request.id}`,
      method: "get"
    });
  }
}

export const defWorkflowInstanceService = new WorkflowInstanceServiceImpl();
