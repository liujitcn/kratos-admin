import { defBaseTenantService } from "@/api/system/admin/v1/base_tenant";
import type { SelectOptionResponse_Option } from "@/rpc/common/v1/common";

/** 默认租户编码。 */
export const DEFAULT_TENANT_CODE = "0000";

const tenantOptions: SelectOptionResponse_Option[] = [];
let tenantOptionsLoaded = false;
let tenantOptionsRequest: Promise<SelectOptionResponse_Option[]> | undefined;
let tenantOptionsVersion = 0;

/** 读取共享租户选项，同一时间只发起一次请求。 */
export async function loadSharedTenantOptions() {
  if (tenantOptionsLoaded) return tenantOptions;
  if (!tenantOptionsRequest) {
    const requestVersion = tenantOptionsVersion;
    tenantOptionsRequest = defBaseTenantService.OptionBaseTenant({ keyword: "" })
      .then(response => {
        if (requestVersion !== tenantOptionsVersion) return tenantOptions;
        tenantOptions.splice(0, tenantOptions.length, ...(response.list ?? []));
        tenantOptionsLoaded = true;
        return tenantOptions;
      })
      .finally(() => {
        tenantOptionsRequest = undefined;
      });
  }
  return tenantOptionsRequest;
}

/** 清除共享租户选项，租户数据变更后由下一次读取重新加载。 */
export function invalidateTenantOptions() {
  tenantOptionsVersion += 1;
  tenantOptions.splice(0, tenantOptions.length);
  tenantOptionsLoaded = false;
  tenantOptionsRequest = undefined;
}

/** 读取租户列表筛选选项。 */
export async function requestTenantOptions() {
  return { data: await loadSharedTenantOptions() };
}
