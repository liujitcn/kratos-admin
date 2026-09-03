import service from "@/utils/request";
import type { SelectOptionResponse } from "@/rpc/common/v1/common";
import type { OptionBaseTenantRequest } from "@/rpc/system/admin/v1/base_tenant";

const BASE_TENANT_URL = "/v1/admin/base/tenant";

/** BaseTenantServiceImpl 管理端运行壳租户服务。 */
export class BaseTenantServiceImpl {
  /** 查询租户选项。 */
  OptionBaseTenant(request: OptionBaseTenantRequest): Promise<SelectOptionResponse> {
    return service<OptionBaseTenantRequest, SelectOptionResponse>({
      url: `${BASE_TENANT_URL}/option`,
      method: "get",
      params: request
    });
  }
}

/** defBaseTenantService 管理端运行壳租户服务实例。 */
export const defBaseTenantService = new BaseTenantServiceImpl();
