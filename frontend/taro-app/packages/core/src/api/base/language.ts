import { http } from '../../utils/http'
import type { GetLanguageRequest, GetLanguageResponse, LanguageService } from '../../rpc/base/v1/language'

const LANGUAGE_URL = '/v1/base/language'

/** 语言公共服务。 */
export class LanguageServiceImpl implements LanguageService {
  /** 查询当前支持的语言和主语言。 */
  GetLanguage(request: GetLanguageRequest): Promise<GetLanguageResponse> {
    return http<GetLanguageResponse>({
      url: LANGUAGE_URL,
      method: 'GET',
      authMode: 'none',
      data: request,
      header: { Authorization: 'no-auth' },
    })
  }
}

/** 主语言公共服务。 */
export const defLanguageService = new LanguageServiceImpl()
