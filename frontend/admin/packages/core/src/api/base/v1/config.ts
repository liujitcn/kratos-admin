import service from "@/utils/request";
import type {
  ConfigService,
  GetConfigRequest,
  GetConfigResponse,
  GetI18nCustomRequest,
  GetI18nCustomResponse
} from "@/rpc/base/v1/config";

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

  /** 获取当前租户的自定义国际化覆盖项。 */
  GetI18nCustom(request: GetI18nCustomRequest): Promise<GetI18nCustomResponse> {
    return service<GetI18nCustomRequest, GetI18nCustomResponse>({
      url: `${CONFIG_URL}/i18n-custom`,
      method: "get",
      params: request
    });
  }
}

export const defConfigService = new ConfigServiceImpl();
