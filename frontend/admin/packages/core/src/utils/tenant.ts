import { defBaseTenantService } from "@/api/system/base_tenant";

/** 默认租户编码。 */
export const DEFAULT_TENANT_CODE = "0000";

/** 读取租户列表筛选选项。 */
export async function requestTenantOptions() {
  const response = await defBaseTenantService.OptionBaseTenant({ keyword: "" });
  return { data: response.list ?? [] };
}
