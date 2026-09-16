import service from "@liujitcn/kratos-admin-core/request";
import {
  type BaseTenantProjectForm,
  type BaseTenantProjectService,
  type CreateBaseTenantProjectRequest,
  type DeleteBaseTenantProjectRequest,
  type GetBaseTenantProjectRequest,
  type OptionBaseTenantProjectRequest,
  type TreeBaseTenantProjectRequest,
  type TreeBaseTenantProjectResponse,
  type PageBaseTenantProjectRequest,
  type PageBaseTenantProjectResponse,
  type SetBaseTenantProjectStatusRequest,
  type UpdateBaseTenantProjectRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_tenant_project";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";
import type { SelectOptionResponse } from "@liujitcn/kratos-admin-system/rpc/common/v1/common";

const BASE_TENANT_PROJECT_URL = "/v1/admin/base/tenant/project";

/** Admin项目服务。 */
export class BaseTenantProjectServiceImpl implements BaseTenantProjectService {
  /** 查询项目下拉选择。 */
  OptionBaseTenantProject(request: OptionBaseTenantProjectRequest): Promise<SelectOptionResponse> {
    return service<OptionBaseTenantProjectRequest, SelectOptionResponse>({
      url: `${BASE_TENANT_PROJECT_URL}/option`,
      method: "get",
      params: request
    });
  }

  /** 查询项目分页列表。 */
  PageBaseTenantProject(request: PageBaseTenantProjectRequest): Promise<PageBaseTenantProjectResponse> {
    return service<PageBaseTenantProjectRequest, PageBaseTenantProjectResponse>({
      url: BASE_TENANT_PROJECT_URL,
      method: "get",
      params: request
    });
  }

  /** 查询当前账号可用的租户项目树。 */
  TreeBaseTenantProject(request: TreeBaseTenantProjectRequest): Promise<TreeBaseTenantProjectResponse> {
    return service<TreeBaseTenantProjectRequest, TreeBaseTenantProjectResponse>({
      url: `${BASE_TENANT_PROJECT_URL}/tree`,
      method: "get",
      params: request
    });
  }

  /** 查询项目。 */
  GetBaseTenantProject(request: GetBaseTenantProjectRequest): Promise<BaseTenantProjectForm> {
    return service<GetBaseTenantProjectRequest, BaseTenantProjectForm>({
      url: `${BASE_TENANT_PROJECT_URL}/${request.id}`,
      method: "get"
    });
  }

  /** 创建项目。 */
  CreateBaseTenantProject(request: CreateBaseTenantProjectRequest): Promise<Empty> {
    return service<BaseTenantProjectForm | undefined, Empty>({
      url: BASE_TENANT_PROJECT_URL,
      method: "post",
      data: request.base_tenant_project
    });
  }

  /** 更新项目。 */
  UpdateBaseTenantProject(request: UpdateBaseTenantProjectRequest): Promise<Empty> {
    return service<BaseTenantProjectForm | undefined, Empty>({
      url: `${BASE_TENANT_PROJECT_URL}/${request.base_tenant_project?.id ?? ""}`,
      method: "put",
      data: request.base_tenant_project
    });
  }

  /** 删除项目。 */
  DeleteBaseTenantProject(request: DeleteBaseTenantProjectRequest): Promise<Empty> {
    return service<DeleteBaseTenantProjectRequest, Empty>({
      url: `${BASE_TENANT_PROJECT_URL}/${request.id}`,
      method: "delete"
    });
  }

  /** 设置项目状态。 */
  SetBaseTenantProjectStatus(request: SetBaseTenantProjectStatusRequest): Promise<Empty> {
    return service<SetBaseTenantProjectStatusRequest, Empty>({
      url: `${BASE_TENANT_PROJECT_URL}/${request.id}/status`,
      method: "put",
      data: request
    });
  }
}

export const defBaseTenantProjectService = new BaseTenantProjectServiceImpl();
