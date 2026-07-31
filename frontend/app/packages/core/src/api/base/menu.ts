import { http } from '../../utils/http'

/** 远端移动菜单服务。 */
export class BaseMenuServiceImpl {
  /** 获取当前访问身份可见的扁平移动菜单。 */
  ListBaseMenu(): Promise<unknown> {
    return http<unknown>({
      url: '/v1/app/base/menu',
      method: 'GET',
      authMode: 'optional',
    })
  }
}

/** 默认移动菜单服务。 */
export const defBaseMenuService = new BaseMenuServiceImpl()
