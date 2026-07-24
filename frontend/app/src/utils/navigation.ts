/** 登录前暂存的目标路由。 */
const LAST_ROUTE_KEY = 'lastRoute'
/** 登录页路径。 */
const LOGIN_PAGE = '/pages/login/login'
/** 首页 tab 页面路径。 */
export const homeTabPage = '/pages/index/index'

/** 获取当前页面的完整路由，供登录前回跳使用。 */
const getCurrentRouteUrl = () => {
  const pages = getCurrentPages()
  const currentPage = pages[pages.length - 1]
  return currentPage?.route ? `/${currentPage.route}` : ''
}

/** 保存当前页面路由。 */
export const saveCurrentRoute = () => {
  const currentRoute = getCurrentRouteUrl()
  if (!currentRoute || currentRoute.startsWith(LOGIN_PAGE)) {
    uni.removeStorageSync(LAST_ROUTE_KEY)
    return
  }
  uni.setStorageSync(LAST_ROUTE_KEY, currentRoute)
}

/** 保存指定页面路由。 */
export const saveLoginRedirectUrl = (url: string) => {
  const normalizedUrl = url.startsWith('/') ? url : `/${url}`
  if (!normalizedUrl.startsWith(LOGIN_PAGE)) {
    uni.setStorageSync(LAST_ROUTE_KEY, normalizedUrl)
  }
}

/** 跳转到登录页，并在跳转前记录当前页面。 */
export const navigateToLogin = (redirectUrl?: string) => {
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
