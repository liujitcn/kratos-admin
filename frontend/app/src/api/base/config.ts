import { http } from '@/utils/http'
import type { ConfigService, GetConfigRequest, GetConfigResponse } from '@/rpc/base/v1/config'

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
}

export const defConfigService = new ConfigServiceImpl()
