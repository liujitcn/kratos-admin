import { http } from '../../../../utils/http'
import type { ListBaseMenuResponse } from '../../../../rpc/system/app/v1/base_menu'

/** 远端移动菜单服务。 */
export class BaseMenuServiceImpl {
  /** 获取扁平移动菜单配置。 */
  ListBaseMenu(): Promise<ListBaseMenuResponse> {
    return http<ListBaseMenuResponse>({
      url: '/v1/app/base/menu',
      method: 'GET',
      authMode: 'optional',
    })
  }
}

/** 默认移动菜单服务。 */
export const defBaseMenuService = new BaseMenuServiceImpl()
