import service from "@liujitcn/kratos-admin-core/request";
import {
  type TestWorkflowScriptRequest,
  type CreateWorkflowDefinitionRequest,
  type DeleteWorkflowDefinitionRequest,
  type GetWorkflowApplyFormRequest,
  type GetWorkflowApplyFormResponse,
  type GetWorkflowDefinitionRequest,
  type OptionWorkflowDefinitionRequest,
  type PageWorkflowDefinitionRequest,
  type PageWorkflowDefinitionResponse,
  type SetWorkflowDefinitionStatusRequest,
  type UpdateWorkflowDefinitionRequest,
  type WorkflowDefinitionForm,
  type WorkflowDefinitionService
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/workflow_definition";
import type {
  PageBaseScriptRequest,
  PageBaseScriptResponse,
  TestRunBaseScriptResponse
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_script";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";
import type { SelectOptionResponse } from "@liujitcn/kratos-admin-system/rpc/common/v1/common";

const WORKFLOW_DEFINITION_URL = "/v1/admin/workflow/definition";

/** Admin工作流流程定义服务。 */
export class WorkflowDefinitionServiceImpl implements WorkflowDefinitionService {
  /** 查询可用于工作流的启用脚本，不返回源码。 */
  PageWorkflowScript(request: PageBaseScriptRequest): Promise<PageBaseScriptResponse> {
    return service<PageBaseScriptRequest, PageBaseScriptResponse>({
      url: `${WORKFLOW_DEFINITION_URL}/script`,
      method: "get",
      params: request
    });
  }

  /** 试运行已保存的工作流脚本，不接受任意源码。 */
  TestWorkflowScript(request: TestWorkflowScriptRequest): Promise<TestRunBaseScriptResponse> {
    return service<TestWorkflowScriptRequest, TestRunBaseScriptResponse>({
      url: `${WORKFLOW_DEFINITION_URL}/script/test`,
      method: "post",
      data: request
    });
  }

  /** 查询启用流程定义下拉选择。 */
  OptionWorkflowDefinition(request: OptionWorkflowDefinitionRequest): Promise<SelectOptionResponse> {
    return service<OptionWorkflowDefinitionRequest, SelectOptionResponse>({
      url: `${WORKFLOW_DEFINITION_URL}/option`,
      method: "get",
      params: request
    });
  }

  /** 查询启用流程定义的申请表单。 */
  GetWorkflowApplyForm(request: GetWorkflowApplyFormRequest): Promise<GetWorkflowApplyFormResponse> {
    return service<GetWorkflowApplyFormRequest, GetWorkflowApplyFormResponse>({
      url: `${WORKFLOW_DEFINITION_URL}/apply-form`,
      method: "get",
      params: request
    });
  }

  /** 查询流程定义分页列表。 */
  PageWorkflowDefinition(request: PageWorkflowDefinitionRequest): Promise<PageWorkflowDefinitionResponse> {
    return service<PageWorkflowDefinitionRequest, PageWorkflowDefinitionResponse>({
      url: WORKFLOW_DEFINITION_URL,
      method: "get",
      params: request
    });
  }

  /** 查询流程定义。 */
  GetWorkflowDefinition(request: GetWorkflowDefinitionRequest): Promise<WorkflowDefinitionForm> {
    return service<GetWorkflowDefinitionRequest, WorkflowDefinitionForm>({
      url: `${WORKFLOW_DEFINITION_URL}/${request.id}`,
      method: "get"
    });
  }

  /** 创建流程定义。 */
  CreateWorkflowDefinition(request: CreateWorkflowDefinitionRequest): Promise<Empty> {
    return service<WorkflowDefinitionForm | undefined, Empty>({
      url: WORKFLOW_DEFINITION_URL,
      method: "post",
      data: request.definition
    });
  }

  /** 更新流程定义。 */
  UpdateWorkflowDefinition(request: UpdateWorkflowDefinitionRequest): Promise<Empty> {
    return service<WorkflowDefinitionForm | undefined, Empty>({
      url: `${WORKFLOW_DEFINITION_URL}/${request.definition?.id ?? ""}`,
      method: "put",
      data: request.definition
    });
  }

  /** 删除流程定义。 */
  DeleteWorkflowDefinition(request: DeleteWorkflowDefinitionRequest): Promise<Empty> {
    return service<DeleteWorkflowDefinitionRequest, Empty>({
      url: `${WORKFLOW_DEFINITION_URL}/${request.id}`,
      method: "delete"
    });
  }

  /** 设置流程定义状态。 */
  SetWorkflowDefinitionStatus(request: SetWorkflowDefinitionStatusRequest): Promise<Empty> {
    return service<SetWorkflowDefinitionStatusRequest, Empty>({
      url: `${WORKFLOW_DEFINITION_URL}/${request.id}/status`,
      method: "put",
      data: request
    });
  }
}

export const defWorkflowDefinitionService = new WorkflowDefinitionServiceImpl();
