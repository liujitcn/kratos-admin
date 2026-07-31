import Taro, { getCurrentPages } from '@tarojs/taro'

const LAST_ROUTE_KEY = 'lastRoute'
const LOGIN_PAGE = '/pages/login/login'
/** 首页 tab 页面路径。 */
export const homeTabPage = '/pages/index/index'

type RouteQueryValue = string | number | boolean | null | undefined

function buildRouteUrl(path: string, query: Record<string, RouteQueryValue> = {}): string {
  const queryString = Object.entries(query)
    .filter(([, value]) => value !== undefined && value !== null && value !== '')
    .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`)
    .join('&')
  return queryString ? `${path}?${queryString}` : path
}

function getCurrentRouteUrl(): string {
  const pages = getCurrentPages()
  const currentPage = pages[pages.length - 1] as
    | {
        route?: string
        options?: Record<string, RouteQueryValue>
        $taroParams?: Record<string, RouteQueryValue>
      }
    | undefined
  if (!currentPage?.route) return ''
  return buildRouteUrl(
    `/${currentPage.route}`,
    currentPage.options ?? currentPage.$taroParams ?? {},
  )
}

/** 保存当前页面路由，登录成功后恢复到用户原来的页面。 */
export function saveCurrentRoute(): void {
  const currentRoute = getCurrentRouteUrl()
  if (!currentRoute || currentRoute.startsWith(LOGIN_PAGE)) {
    Taro.removeStorageSync(LAST_ROUTE_KEY)
    return
  }
  Taro.setStorageSync(LAST_ROUTE_KEY, currentRoute)
}

/** 保存指定页面路由，供登录成功后继续原访问目标。 */
export function saveLoginRedirectUrl(url: string): void {
  const normalizedUrl = url.startsWith('/') ? url : `/${url}`
  if (!normalizedUrl.startsWith(LOGIN_PAGE)) Taro.setStorageSync(LAST_ROUTE_KEY, normalizedUrl)
}

/** 读取并清理登录成功后的回跳地址。 */
export function consumeLoginRedirectUrl(): string {
  const url = Taro.getStorageSync<string>(LAST_ROUTE_KEY) || ''
  Taro.removeStorageSync(LAST_ROUTE_KEY)
  return url
}

/** 跳转到登录页，并在跳转前记录当前页面。 */
export function navigateToLogin(redirectUrl?: string): void {
  if (typeof redirectUrl === 'string' && redirectUrl) saveLoginRedirectUrl(redirectUrl)
  else saveCurrentRoute()
  void Taro.navigateTo({
    url: LOGIN_PAGE,
    fail: () => {
      void Taro.reLaunch({ url: LOGIN_PAGE })
    },
  })
}

/** 返回首页。 */
export function switchTabToHome(): Promise<TaroGeneral.CallbackResult> {
  return Taro.reLaunch({ url: homeTabPage })
}
