import { http } from '../../../../utils/http'
import type {
  BaseAreaService,
  TreeBaseAreaRequest,
  TreeBaseAreaResponse,
} from '../../../../rpc/system/app/v1/base_area'

const BASE_AREA_URL = '/v1/app/base/area'

/**
 * 行政区域服务实现。
 */
export class BaseAreaServiceImpl implements BaseAreaService {
  /**
   * 查询行政区域树形列表。
   */
  async TreeBaseArea(request: TreeBaseAreaRequest): Promise<TreeBaseAreaResponse> {
    const response = await http<Partial<TreeBaseAreaResponse>>({
      url: `${BASE_AREA_URL}/tree`,
      method: 'GET',
      data: request,
    })

    return {
      ...response,
      areas: response.areas ?? [],
    }
  }
}

/**
 * 默认行政区域服务。
 */
export const defBaseAreaService = new BaseAreaServiceImpl()
