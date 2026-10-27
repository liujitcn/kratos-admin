import service from "@liujitcn/kratos-admin-core/request";
import {
  type DeleteWorkflowDelegationRequest,
  type PageWorkflowDelegationRequest,
  type PageWorkflowDelegationResponse,
  type SetWorkflowDelegationRequest,
  type WorkflowDelegationForm,
  type WorkflowDelegationService
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/workflow_delegation";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";

const WORKFLOW_DELEGATION_URL = "/v1/admin/workflow/delegation";

/** Admin工作流审批委托服务。 */
export class WorkflowDelegationServiceImpl implements WorkflowDelegationService {
  /** 查询审批委托分页列表。 */
  PageWorkflowDelegation(request: PageWorkflowDelegationRequest): Promise<PageWorkflowDelegationResponse> {
    return service<PageWorkflowDelegationRequest, PageWorkflowDelegationResponse>({
      url: WORKFLOW_DELEGATION_URL,
      method: "get",
      params: request
    });
  }

  /** 设置我的审批委托。 */
  SetWorkflowDelegation(request: SetWorkflowDelegationRequest): Promise<Empty> {
    return service<WorkflowDelegationForm | undefined, Empty>({
      url: WORKFLOW_DELEGATION_URL,
      method: "post",
      data: request.delegation
    });
  }

  /** 删除审批委托。 */
  DeleteWorkflowDelegation(request: DeleteWorkflowDelegationRequest): Promise<Empty> {
    return service<DeleteWorkflowDelegationRequest, Empty>({
      url: `${WORKFLOW_DELEGATION_URL}/${request.id}`,
      method: "delete"
    });
  }
}

export const defWorkflowDelegationService = new WorkflowDelegationServiceImpl();
