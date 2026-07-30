import service from "@/utils/request";
import type { OptionBaseDictRequest, OptionBaseDictResponse } from "@/rpc/system/admin/v1/base_dict";

const BASE_DICT_URL = "/v1/admin/base/dict";

/** BaseDictServiceImpl 管理端运行壳字典服务。 */
export class BaseDictServiceImpl {
  /** 查询全局字典选项。 */
  OptionBaseDict(request: OptionBaseDictRequest): Promise<OptionBaseDictResponse> {
    return service<OptionBaseDictRequest, OptionBaseDictResponse>({
      url: `${BASE_DICT_URL}/option`,
      method: "get",
      params: request
    });
  }
}

/** defBaseDictService 管理端运行壳字典服务实例。 */
export const defBaseDictService = new BaseDictServiceImpl();
