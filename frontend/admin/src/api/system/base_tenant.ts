import service from "@/utils/request";
import {
  type BaseTenantForm,
  type BaseTenantService,
  type CreateBaseTenantRequest,
  type DeleteBaseTenantRequest,
  type GetBaseTenantRequest,
  type OptionBaseTenantRequest,
  type PageBaseTenantRequest,
  type PageBaseTenantResponse,
  type SetBaseTenantStatusRequest,
  type UpdateBaseTenantRequest
} from "@/rpc/system/admin/v1/base_tenant";
import type { SelectOptionResponse } from "@/rpc/common/v1/common";
import type { Empty } from "@/rpc/google/protobuf/empty";

const BASE_TENANT_URL = "/v1/admin/base/tenant";

/** Admin租户管理服务 */
export class BaseTenantServiceImpl implements BaseTenantService {
  /** 查询租户下拉选择 */
  OptionBaseTenant(request: OptionBaseTenantRequest): Promise<SelectOptionResponse> {
    return service<OptionBaseTenantRequest, SelectOptionResponse>({
      url: `${BASE_TENANT_URL}/option`,
      method: "get",
      params: request
    });
  }

  /** 查询租户分页列表 */
  PageBaseTenant(request: PageBaseTenantRequest): Promise<PageBaseTenantResponse> {
    return service<PageBaseTenantRequest, PageBaseTenantResponse>({
      url: `${BASE_TENANT_URL}`,
      method: "get",
      params: request
    });
  }

  /** 查询租户 */
  GetBaseTenant(request: GetBaseTenantRequest): Promise<BaseTenantForm> {
    return service<GetBaseTenantRequest, BaseTenantForm>({
      url: `${BASE_TENANT_URL}/${request.id}`,
      method: "get"
    });
  }

  /** 创建租户 */
  CreateBaseTenant(request: CreateBaseTenantRequest): Promise<Empty> {
    return service<BaseTenantForm | undefined, Empty>({
      url: `${BASE_TENANT_URL}`,
      method: "post",
      data: request.base_tenant
    });
  }

  /** 更新租户 */
  UpdateBaseTenant(request: UpdateBaseTenantRequest): Promise<Empty> {
    return service<BaseTenantForm | undefined, Empty>({
      url: `${BASE_TENANT_URL}/${request.base_tenant?.id ?? ""}`,
      method: "put",
      data: request.base_tenant
    });
  }

  /** 删除租户 */
  DeleteBaseTenant(request: DeleteBaseTenantRequest): Promise<Empty> {
    return service<DeleteBaseTenantRequest, Empty>({
      url: `${BASE_TENANT_URL}/${request.id}`,
      method: "delete"
    });
  }

  /** 设置状态 */
  SetBaseTenantStatus(request: SetBaseTenantStatusRequest): Promise<Empty> {
    return service<SetBaseTenantStatusRequest, Empty>({
      url: `${BASE_TENANT_URL}/${request.id}/status`,
      method: "put",
      data: request
    });
  }
}

export const defBaseTenantService = new BaseTenantServiceImpl();
