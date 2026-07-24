import { http } from '@/utils/http'
import type { BaseDictForm, BaseDictService } from '@/rpc/system/app/v1/base_dict'

const BASE_DICT_URL = '/v1/app/base/dict'

/** 字典查询请求兼容结构，支持 code 和旧版 value。 */
type GetBaseDictRequestCompat = {
  code?: string
  value?: string
}

/** 字典服务 */
export class BaseDictServiceImpl implements BaseDictService {
  /** 查询单个字典 */
  GetBaseDict(request: GetBaseDictRequestCompat): Promise<BaseDictForm> {
    const code = request.code ?? request.value ?? ''
    return http<BaseDictForm>({
      url: `${BASE_DICT_URL}/${code}`,
      method: 'GET',
    })
  }
}

export const defBaseDictService = new BaseDictServiceImpl()
