import { http } from '../../../utils/http'
import type {
  ConfigService,
  GetConfigRequest,
  GetConfigResponse,
  GetI18nCustomRequest,
  GetI18nCustomResponse,
} from '../../../rpc/base/v1/config'

const CONFIG_URL = '/v1/base/config'

/** 系统配置公共服务 */
export class ConfigServiceImpl implements ConfigService {
  /** 获取系统配置 */
  GetConfig(request: GetConfigRequest): Promise<GetConfigResponse> {
    return http<GetConfigResponse>({
      url: `${CONFIG_URL}`,
      method: 'GET',
      authMode: 'none',
      data: request,
      header: { Authorization: 'no-auth' },
    })
  }

  /** 获取当前租户的自定义国际化覆盖项。 */
  GetI18nCustom(request: GetI18nCustomRequest): Promise<GetI18nCustomResponse> {
    return http<GetI18nCustomResponse>({
      url: `${CONFIG_URL}/i18n-custom`,
      method: 'GET',
      data: request,
    })
  }
}

export const defConfigService = new ConfigServiceImpl()
