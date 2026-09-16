import service from "@liujitcn/kratos-admin-core/request";
import {
  type BaseTenantProjectGrant,
  type BaseTenantProjectGrantService,
  type CreateBaseTenantProjectGrantRequest,
  type DeleteBaseTenantProjectGrantRequest,
  type GetBaseTenantProjectGrantRequest,
  type PageBaseTenantProjectGrantRequest,
  type PageBaseTenantProjectGrantResponse,
  type UpdateBaseTenantProjectGrantRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_tenant_project_grant";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";

const BASE_TENANT_PROJECT_GRANT_URL = "/v1/admin/base/tenant/project/grant";

/** Admin项目授权服务。 */
export class BaseTenantProjectGrantServiceImpl implements BaseTenantProjectGrantService {
  /** 查询项目授权分页列表。 */
  PageBaseTenantProjectGrant(request: PageBaseTenantProjectGrantRequest): Promise<PageBaseTenantProjectGrantResponse> {
    return service<PageBaseTenantProjectGrantRequest, PageBaseTenantProjectGrantResponse>({
      url: BASE_TENANT_PROJECT_GRANT_URL,
      method: "get",
      params: request
    });
  }

  /** 查询项目授权详情。 */
  GetBaseTenantProjectGrant(request: GetBaseTenantProjectGrantRequest): Promise<BaseTenantProjectGrant> {
    return service<GetBaseTenantProjectGrantRequest, BaseTenantProjectGrant>({
      url: `${BASE_TENANT_PROJECT_GRANT_URL}/${request.tenant_id}/${request.subject_type}/${request.subject_id}`,
      method: "get"
    });
  }

  /** 新增项目授权。 */
  CreateBaseTenantProjectGrant(request: CreateBaseTenantProjectGrantRequest): Promise<Empty> {
    return service<BaseTenantProjectGrant | undefined, Empty>({
      url: BASE_TENANT_PROJECT_GRANT_URL,
      method: "post",
      data: request.base_tenant_project_grant
    });
  }

  /** 更新项目授权的项目范围。 */
  UpdateBaseTenantProjectGrant(request: UpdateBaseTenantProjectGrantRequest): Promise<Empty> {
    const grant = request.base_tenant_project_grant;
    return service<BaseTenantProjectGrant | undefined, Empty>({
      url: `${BASE_TENANT_PROJECT_GRANT_URL}/${grant?.tenant_id ?? 0}/${grant?.subject_type ?? 0}/${grant?.subject_id ?? 0}`,
      method: "put",
      data: grant
    });
  }

  /** 删除项目授权。 */
  DeleteBaseTenantProjectGrant(request: DeleteBaseTenantProjectGrantRequest): Promise<Empty> {
    return service<DeleteBaseTenantProjectGrantRequest, Empty>({
      url: `${BASE_TENANT_PROJECT_GRANT_URL}/${request.tenant_id}/${request.subject_type}/${request.subject_id}`,
      method: "delete"
    });
  }
}

export const defBaseTenantProjectGrantService = new BaseTenantProjectGrantServiceImpl();
