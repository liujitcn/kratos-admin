import service from "@/utils/request";
import type { ConfigService, GetConfigRequest, GetConfigResponse } from "@/rpc/base/v1/config";

const CONFIG_URL = "/v1/base/config";

/** 系统配置公共服务 */
export class ConfigServiceImpl implements ConfigService {
  /** 获取系统配置 */
  GetConfig(request: GetConfigRequest): Promise<GetConfigResponse> {
    return service<GetConfigRequest, GetConfigResponse>({
      url: `${CONFIG_URL}`,
      method: "get",
      params: request,
      headers: { Authorization: "no-auth" }
    });
  }
}

export const defConfigService = new ConfigServiceImpl();
