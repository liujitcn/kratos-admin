import { hasValidToken } from './auth'

/** 登录前暂存的目标路由。 */
const LAST_ROUTE_KEY = 'lastRoute'
/** 登录页路径。 */
const LOGIN_PAGE = '/pages/login/login'
/** 携带本次回跳标记的登录入口。 */
const LOGIN_REDIRECT_PAGE = `${LOGIN_PAGE}?redirect=1`
/** 首页 tab 页面路径。 */
export const homeTabPage = '/pages/index/index'

/** 路由 query 支持的值类型。 */
type RouteQueryValue = string | number | boolean | null | undefined

/** 构建包含 query 的页面路由。 */
const buildRouteUrl = (path: string, query: Record<string, RouteQueryValue> = {}) => {
  const queryString = Object.entries(query)
    .filter(([, value]) => value !== undefined && value !== null && value !== '')
    .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`)
    .join('&')

  return queryString ? `${path}?${queryString}` : path
}

/** 获取当前页面的完整路由，包含 query，供登录前回跳使用。 */
const getCurrentRouteUrl = () => {
  const pages = getCurrentPages()
  const currentPage = pages[pages.length - 1]
  if (!currentPage?.route) {
    return ''
  }

  let query: Record<string, RouteQueryValue> = {}
  const miniProgramPage = currentPage as { options?: Record<string, string> }
  const routePage = currentPage as {
    $vm?: { $route?: { query?: Record<string, RouteQueryValue> } }
  }

  // #ifdef MP-WEIXIN
  query = miniProgramPage.options || {}
  // #endif

  // #ifdef H5 || APP-PLUS
  query = routePage.$vm?.$route?.query || {}
  // #endif

  return buildRouteUrl(`/${currentPage.route}`, query)
}

/** 保存当前页面路由，登录成功后恢复到用户原来的页面。 */
export const saveCurrentRoute = () => {
  const currentRoute = getCurrentRouteUrl()
  if (!currentRoute || currentRoute.startsWith(LOGIN_PAGE)) {
    return
  }
  uni.setStorageSync(LAST_ROUTE_KEY, currentRoute)
}

/** 保存指定页面路由，供登录成功后继续原访问目标。 */
export const saveLoginRedirectUrl = (url: string) => {
  const normalizedUrl = url.startsWith('/') ? url : `/${url}`
  if (!normalizedUrl.startsWith(LOGIN_PAGE)) {
    uni.setStorageSync(LAST_ROUTE_KEY, normalizedUrl)
  }
}

/** prepareLoginPage 清理直接访问的旧目标，已登录时替换回首页。 */
export const prepareLoginPage = (redirect?: string): boolean => {
  if (redirect !== '1') uni.removeStorageSync(LAST_ROUTE_KEY)
  if (!hasValidToken()) return true
  void returnFromLogin()
  return false
}

/** returnFromLogin 结束本次登录入口并替换到首页，避免返回受保护页面。 */
export const returnFromLogin = async (): Promise<void> => {
  uni.removeStorageSync(LAST_ROUTE_KEY)
  uni.setStorageSync('SwitchTabIndex', true)
  await uni.reLaunch({ url: homeTabPage })
}

/** restoreLoginRedirect 替换到本次访问目标，跳转成功后才清理保存的地址。 */
export const restoreLoginRedirect = async (): Promise<void> => {
  const savedRoute = uni.getStorageSync(LAST_ROUTE_KEY) || ''
  const normalizedRoute = `/${savedRoute.replace(/^\/+/, '')}`
  const targetRoute =
    savedRoute && !normalizedRoute.startsWith(LOGIN_PAGE) ? normalizedRoute : homeTabPage
  if (targetRoute.startsWith(homeTabPage)) uni.setStorageSync('SwitchTabIndex', true)
  await uni.reLaunch({ url: targetRoute })
  if (savedRoute && uni.getStorageSync(LAST_ROUTE_KEY) === savedRoute) {
    uni.removeStorageSync(LAST_ROUTE_KEY)
  }
}

/** 跳转到登录页，并在跳转前记录当前页面。 */
export const navigateToLogin = (redirectUrl?: string) => {
  // 模板直接绑定时会传入点击事件对象，只有字符串才是明确的回跳地址。
  if (typeof redirectUrl === 'string' && redirectUrl) {
    saveLoginRedirectUrl(redirectUrl)
  } else {
    saveCurrentRoute()
  }
  uni.navigateTo({
    url: LOGIN_REDIRECT_PAGE,
    fail: () => {
      uni.reLaunch({ url: LOGIN_REDIRECT_PAGE })
    },
  })
}

/** 跳转登录页并替换当前页面：路由守卫使用，避免反复拦截时在页面栈堆积登录页。 */
export const redirectToLogin = (redirectUrl?: string) => {
  // 模板直接绑定时会传入点击事件对象，只有字符串才是明确的回跳地址。
  if (typeof redirectUrl === 'string' && redirectUrl) {
    saveLoginRedirectUrl(redirectUrl)
  } else {
    saveCurrentRoute()
  }
  uni.redirectTo({
    url: LOGIN_REDIRECT_PAGE,
    fail: () => {
      uni.reLaunch({ url: LOGIN_REDIRECT_PAGE })
    },
  })
}

/** 返回首页。 */
export const switchTabToHome = () => uni.reLaunch({ url: homeTabPage })
