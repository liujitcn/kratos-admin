import Taro, { getCurrentPages } from '@tarojs/taro'
import { hasValidToken } from './auth'

const LAST_ROUTE_KEY = 'lastRoute'
const LOGIN_PAGE = '/pages/login/login'
/** 携带本次回跳标记的登录入口。 */
const LOGIN_REDIRECT_PAGE = `${LOGIN_PAGE}?redirect=1`
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

/** 规范化应用内部路由，避免多余的前导斜杠导致 Taro 跳转失败。 */
function normalizeRouteUrl(url: string): string {
  return `/${url.replace(/^\/+/, '')}`
}

/** 获取当前页面及其跨端查询参数。 */
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
    normalizeRouteUrl(currentPage.route),
    { ...currentPage.$taroParams, ...currentPage.options },
  )
}

/** 保存当前页面路由，登录成功后恢复到用户原来的页面。 */
export function saveCurrentRoute(): void {
  const currentRoute = getCurrentRouteUrl()
  if (!currentRoute || currentRoute.startsWith(LOGIN_PAGE)) return
  Taro.setStorageSync(LAST_ROUTE_KEY, currentRoute)
}

/** 保存指定页面路由，供登录成功后继续原访问目标。 */
export function saveLoginRedirectUrl(url: string): void {
  const normalizedUrl = normalizeRouteUrl(url)
  if (!normalizedUrl.startsWith(LOGIN_PAGE)) Taro.setStorageSync(LAST_ROUTE_KEY, normalizedUrl)
}

/** 读取并清理登录成功后的回跳地址。 */
export function consumeLoginRedirectUrl(): string {
  const url = Taro.getStorageSync<string>(LAST_ROUTE_KEY) || ''
  Taro.removeStorageSync(LAST_ROUTE_KEY)
  return url
}

/** prepareLoginPage 清理直接访问的旧目标，已登录时替换回首页。 */
export function prepareLoginPage(redirect?: string): boolean {
  if (redirect !== '1') Taro.removeStorageSync(LAST_ROUTE_KEY)
  if (!hasValidToken()) return true
  void returnFromLogin()
  return false
}

/** returnFromLogin 结束本次登录入口并替换到首页，避免返回受保护页面。 */
export async function returnFromLogin(): Promise<void> {
  Taro.removeStorageSync(LAST_ROUTE_KEY)
  Taro.setStorageSync('SwitchTabIndex', true)
  await Taro.reLaunch({ url: homeTabPage })
}

/** restoreLoginRedirect 替换到本次访问目标，跳转成功后才清理保存的地址。 */
export async function restoreLoginRedirect(): Promise<void> {
  const savedRoute = Taro.getStorageSync<string>(LAST_ROUTE_KEY) || ''
  const normalizedRoute = normalizeRouteUrl(savedRoute)
  const targetRoute =
    savedRoute && !normalizedRoute.startsWith(LOGIN_PAGE) ? normalizedRoute : homeTabPage
  if (targetRoute.startsWith(homeTabPage)) Taro.setStorageSync('SwitchTabIndex', true)
  await Taro.reLaunch({ url: targetRoute })
  if (savedRoute && Taro.getStorageSync<string>(LAST_ROUTE_KEY) === savedRoute) {
    Taro.removeStorageSync(LAST_ROUTE_KEY)
  }
}

/** 跳转到登录页，并在跳转前记录当前页面。 */
export function navigateToLogin(redirectUrl?: string): void {
  if (typeof redirectUrl === 'string' && redirectUrl) saveLoginRedirectUrl(redirectUrl)
  else saveCurrentRoute()
  void Taro.navigateTo({
    url: LOGIN_REDIRECT_PAGE,
    fail: () => {
      void Taro.reLaunch({ url: LOGIN_REDIRECT_PAGE })
    },
  })
}

/** 跳转登录页并替换当前页面：路由守卫使用，避免反复拦截时在页面栈堆积登录页。 */
export function redirectToLogin(redirectUrl?: string): void {
  if (typeof redirectUrl === 'string' && redirectUrl) saveLoginRedirectUrl(redirectUrl)
  else saveCurrentRoute()
  void Taro.redirectTo({
    url: LOGIN_REDIRECT_PAGE,
    fail: () => {
      void Taro.reLaunch({ url: LOGIN_REDIRECT_PAGE })
    },
  })
}

/** 返回首页。 */
export function switchTabToHome(): Promise<TaroGeneral.CallbackResult> {
  return Taro.reLaunch({ url: homeTabPage })
}
