import { defBaseApiService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_api";
import type { ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import type { BaseApi, OptionBaseApiRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_api";

/** API 穿梭框选项。 */
export type ApiTransferOption = BaseApi & ProFormOption;

/** API 穿梭框目录数据。 */
export interface ApiTransferCatalog {
  /** 原始 API 列表。 */
  apis: BaseApi[];
  /** 穿梭框选项列表。 */
  options: ApiTransferOption[];
}

/** 加载 API 目录并统一生成穿梭框选项。 */
export async function loadApiTransferCatalog(request: OptionBaseApiRequest = {}): Promise<ApiTransferCatalog> {
  const response = await defBaseApiService.OptionBaseApi(request);
  const apis = response.base_apis ?? [];
  const options = apis.map(api => ({
    ...api,
    value: api.operation,
    label: `${api.service_desc || api.service_name}/${api.desc || api.operation} · ${api.method} ${api.path} · ${api.operation}`
  }));

  return { apis, options };
}
