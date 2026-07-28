/** 登录前暂存的目标路由。 */
const LAST_ROUTE_KEY = 'lastRoute'
/** 登录页路径。 */
const LOGIN_PAGE = '/pages/login/login'
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
    uni.removeStorageSync(LAST_ROUTE_KEY)
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

/** 跳转到登录页，并在跳转前记录当前页面。 */
export const navigateToLogin = (redirectUrl?: string) => {
  // 模板直接绑定时会传入点击事件对象，只有字符串才是明确的回跳地址。
  if (typeof redirectUrl === 'string' && redirectUrl) {
    saveLoginRedirectUrl(redirectUrl)
  } else {
    saveCurrentRoute()
  }
  uni.navigateTo({
    url: LOGIN_PAGE,
    fail: () => {
      uni.reLaunch({ url: LOGIN_PAGE })
    },
  })
}

/** 切换到首页 tab。 */
export const switchTabToHome = () => uni.switchTab({ url: homeTabPage })
