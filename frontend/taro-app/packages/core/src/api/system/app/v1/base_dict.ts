import { http } from '../../../../utils/http'
import type { BaseDictForm, BaseDictService, GetBaseDictRequest } from '../../../../rpc/system/app/v1/base_dict'

const BASE_DICT_URL = '/v1/app/base/dict'

/** 字典服务 */
export class BaseDictServiceImpl implements BaseDictService {
  /** 查询单个字典 */
  GetBaseDict(request: GetBaseDictRequest): Promise<BaseDictForm> {
    return http<BaseDictForm>({
      url: `${BASE_DICT_URL}/${request.code}`,
      method: 'GET',
    })
  }
}

export const defBaseDictService = new BaseDictServiceImpl()
