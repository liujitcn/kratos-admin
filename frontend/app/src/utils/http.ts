/**
 * 添加拦截器:
 *   拦截 request 请求
 *   拦截 uploadFile 文件上传
 *
 */

import {
  clearToken,
  getRefreshToken,
  getToken,
  hasValidToken,
  setRefreshToken,
  setToken,
  setTokenExpiresIn,
  shouldRefreshToken,
} from '@/utils/auth'
import { saveCurrentRoute } from '@/utils/navigation'

const apiBasePath = import.meta.env.VITE_APP_BASE_API || '/api'
const apiTargetUrl = import.meta.env.VITE_APP_API_URL || ''
const normalizedApiBasePath = apiBasePath.startsWith('/') ? apiBasePath : `/${apiBasePath}`

/**
 * H5 开发环境优先走同源代理，避免浏览器直接请求后端产生跨域。
 * 其它平台继续使用显式配置的后端地址。
 */
const requestOrigin =
  typeof window !== 'undefined' && window.location?.protocol?.startsWith('http')
    ? ''
    : apiTargetUrl.replace(/\/$/, '')
const baseURL = `${requestOrigin}${normalizedApiBasePath}`
export const requestBaseURL = baseURL
const SESSION_URL = '/v1/base/session'
const REFRESH_TOKEN_URL = '/v1/base/token'
const CAPTCHA_URL = '/v1/base/captcha'
const CONFIG_URL = '/v1/base/config'
const PASSWORD_PUBLIC_KEY_URL = '/v1/base/password-public-key'
const LEGACY_AUTH_URL = '/auth'
const LEGACY_REFRESH_TOKEN_URL = `${LEGACY_AUTH_URL}/token`
const LEGACY_CAPTCHA_URL = '/login/captcha'
// 认证公共接口不携带旧 token，同时保留旧路径兼容迁移期间的灰度访问。
const NO_AUTH_URL_SET = new Set([
  SESSION_URL,
  REFRESH_TOKEN_URL,
  CAPTCHA_URL,
  CONFIG_URL,
  PASSWORD_PUBLIC_KEY_URL,
  LEGACY_AUTH_URL,
  LEGACY_CAPTCHA_URL,
  LEGACY_REFRESH_TOKEN_URL,
])
const AUTH_EXPIRED_EXCLUDED_URL_SET = new Set([
  SESSION_URL,
  CAPTCHA_URL,
  CONFIG_URL,
  PASSWORD_PUBLIC_KEY_URL,
  LEGACY_AUTH_URL,
  LEGACY_CAPTCHA_URL,
])

const AUTH_SILENT_LOGOUT_EVENT = 'auth:silent-logout'

export type AuthMode = 'none' | 'optional' | 'required'

type HttpRequestOptions = UniApp.RequestOptions & {
  /** none: 游客接口；optional: 可选登录；required: 必须登录。 */
  authMode?: AuthMode
}

// 添加拦截器
const httpInterceptor = {
  // 拦截前触发
  invoke(options: UniApp.RequestOptions) {
    const authMode = (options as HttpRequestOptions).authMode
    // 1. 非 http 开头需拼接地址
    if (!options.url.startsWith('http')) {
      options.url = baseURL + options.url
    }
    // 2. 请求超时，普通请求默认 10s；流式等长连接可自行传入更长超时。
    options.timeout = options.timeout || 10000
    // 3. 添加小程序端请求头标识
    options.header = {
      ...options.header,
      'source-client': 'miniapp',
    }
    // 4. 添加 token 请求头标识
    const accessToken = getToken()
    if (authMode !== 'none' && options.header.Authorization !== 'no-auth' && accessToken) {
      options.header.Authorization = accessToken
    } else {
      delete options.header.Authorization
    }
  },
}
uni.addInterceptor('request', httpInterceptor)
uni.addInterceptor('uploadFile', httpInterceptor)

/**
 * 请求函数
 * @param  UniApp.RequestOptions
 * @returns Promise
 *  1. 返回 Promise 对象
 *  2. 获取数据成功
 *    2.1 提取核心数据 res.data
 *    2.2 添加类型，支持泛型
 *  3. 获取数据失败
 *    3.1 401/403错误 -> 清理用户信息，跳转到登录页
 *    3.2 其他错误 -> 根据后端错误信息轻提示
 *    3.3 网络错误 -> 提示用户换网络
 */
type Data = {
  code?: string | number
  message?: string
  reason?: string | number
}

// 新错误模型下，认证与鉴权只保留两类顶层原因。
const authErrorCodeSet = new Set(['401', '403'])
const authErrorReasonSet = new Set(['UNAUTHENTICATED', 'PERMISSION_DENIED'])

function isAuthErrorResponse(data: unknown) {
  if (!data || typeof data !== 'object') {
    return false
  }

  const response = data as Data
  const code = response.code !== undefined ? String(response.code) : ''
  const reason = response.reason !== undefined ? String(response.reason) : ''

  return (
    authErrorCodeSet.has(code) || authErrorCodeSet.has(reason) || authErrorReasonSet.has(reason)
  )
}

// 当前请求属于公共认证接口时，不应自动附带旧登录态。
function isNoAuthRequest(url: string) {
  return NO_AUTH_URL_SET.has(url)
}

// 登录与验证码接口失败时，只提示本次请求失败，不触发重新登录流程。
function shouldSkipAuthExpiredPrompt(url: string) {
  return AUTH_EXPIRED_EXCLUDED_URL_SET.has(url)
}

function resolveAuthMode(options: HttpRequestOptions, url: string): AuthMode {
  if (options.authMode) {
    return options.authMode
  }
  if (options.header?.Authorization === 'no-auth' || isNoAuthRequest(url)) {
    return 'none'
  }
  return 'required'
}

// 2.2 添加类型，支持泛型
export const http = <T>(options: HttpRequestOptions) => {
  // 1. 返回 Promise 对象
  return new Promise<T>((resolve, reject) => {
    const sendRequest = async (retriedAsAnonymous = false) => {
      try {
        const requestOptions = { ...options, header: { ...options.header } }
        const requestUrl = String(requestOptions.url)
        const authMode = resolveAuthMode(requestOptions, requestUrl)
        let accessToken = ''
        try {
          accessToken = await getAccessTokenByMode(
            retriedAsAnonymous && authMode === 'optional' ? 'none' : authMode,
          )
        } catch (error) {
          if (authMode === 'optional' && !retriedAsAnonymous) {
            silentClearAuthData()
            void sendRequest(true)
            return
          }
          throw error
        }
        if (accessToken) {
          requestOptions.header = {
            ...requestOptions.header,
            Authorization: accessToken,
          }
        } else {
          delete requestOptions.header.Authorization
        }
        uni.request({
          ...requestOptions,
          // 响应成功
          success(res) {
            const responseData = res.data as Data
            // 状态码 2xx， axios 就是这样设计的
            if (res.statusCode >= 200 && res.statusCode < 300) {
              if (isAuthErrorResponse(responseData)) {
                if (authMode === 'optional' && !retriedAsAnonymous) {
                  silentClearAuthData()
                  void sendRequest(true)
                  return
                }
                handleAuthExpiredByMode(authMode, requestUrl, responseData)
                reject(res)
                return
              }
              // 2.1 提取核心数据 res.data
              resolve(res.data as T)
            } else if (res.statusCode === 401 || res.statusCode === 403) {
              // 401/403 错误 -> 清理用户信息，跳转到登录页
              if (authMode === 'optional' && !retriedAsAnonymous) {
                silentClearAuthData()
                void sendRequest(true)
                return
              }
              handleAuthExpiredByMode(authMode, requestUrl, responseData)
              reject(res)
            } else {
              // 其他错误 -> 根据后端错误信息轻提示
              void uni.showToast({
                icon: 'none',
                title: responseData.message || '请求错误',
              })
              reject(res)
            }
          },
          // 响应失败
          fail(err) {
            void uni.showToast({
              icon: 'none',
              title: '网络错误，换个网络试试',
            })
            reject(err)
          },
        })
      } catch (error) {
        reject(error)
      }
    }

    void sendRequest()
  })
}

// 刷新 Token 的锁
let isRefreshing = false
let refreshTokenPromise: Promise<void> | null = null
let isPromptingRelogin = false

/** 获取请求可用访问令牌，供 direct stream 等非 uni.request 场景复用。 */
export async function getRequestAccessToken(authMode: AuthMode = 'required') {
  return getAccessTokenByMode(authMode)
}

/** 触发登录失效处理，供 direct stream 等非 uni.request 场景复用。 */
export function handleAuthExpired(authMode: AuthMode = 'required') {
  if (authMode === 'required') {
    void promptRelogin()
    return
  }
  silentClearAuthData()
}

// 刷新 Token 处理
function handleTokenRefresh(authMode: AuthMode) {
  if (refreshTokenPromise) {
    return refreshTokenPromise
  }
  isRefreshing = true
  refreshTokenPromise = refreshAccessToken()
    .catch(async (error) => {
      if (authMode === 'required') {
        await promptRelogin()
      } else {
        silentClearAuthData()
      }
      throw error
    })
    .finally(() => {
      isRefreshing = false
      refreshTokenPromise = null
    })
  return refreshTokenPromise
}

async function getAccessTokenByMode(authMode: AuthMode) {
  if (authMode === 'none') {
    return ''
  }

  if (!getToken()) {
    if (authMode === 'required') {
      await promptRelogin()
      throw new Error('auth required')
    }
    return ''
  }

  if (shouldRefreshToken()) {
    await handleTokenRefresh(authMode)
  }

  if (hasValidToken()) {
    return getToken()
  }

  if (authMode === 'required') {
    await promptRelogin()
    throw new Error('auth expired')
  }

  silentClearAuthData()
  return ''
}

async function refreshAccessToken() {
  const refreshToken = getRefreshToken()
  if (!refreshToken) {
    throw new Error('refresh token missing')
  }

  const response = await new Promise<UniApp.RequestSuccessCallbackResult>((resolve, reject) => {
    uni.request({
      url: REFRESH_TOKEN_URL,
      method: 'POST',
      data: { refresh_token: refreshToken },
      header: {
        'source-client': 'miniapp',
      },
      success: resolve,
      fail: reject,
    })
  })

  const responseData = response.data as
    | Data
    | {
        token_type?: string
        access_token?: string
        refresh_token?: string
        expires_in?: number
      }

  if (
    response.statusCode < 200 ||
    response.statusCode >= 300 ||
    isAuthErrorResponse(responseData)
  ) {
    throw response
  }

  const {
    token_type,
    access_token,
    refresh_token: nextRefreshToken,
    expires_in,
  } = responseData as {
    token_type?: string
    access_token?: string
    refresh_token?: string
    expires_in?: number
  }

  if (!token_type || !access_token || !nextRefreshToken || !expires_in) {
    throw new Error('refresh token response invalid')
  }

  setToken(`${token_type} ${access_token}`)
  setRefreshToken(nextRefreshToken)
  setTokenExpiresIn(expires_in)
}

async function promptRelogin() {
  if (isPromptingRelogin) {
    return
  }
  isPromptingRelogin = true
  try {
    const modalRes = await uni.showModal({
      title: '提示',
      content: '当前页面已失效，请重新登录',
      showCancel: false,
      confirmText: '重新登录',
    })

    if (!modalRes.confirm) {
      return
    }

    // 小程序端确认弹窗关闭到页面跳转之间留一个极短缓冲，避免按钮点击后路由不生效。
    await new Promise((resolve) => setTimeout(resolve, 80))
    await clearUserData()
  } finally {
    isPromptingRelogin = false
  }
}

function handleAuthExpiredByMode(authMode: AuthMode, url: string, responseData: Data) {
  if (authMode === 'required' && !shouldSkipAuthExpiredPrompt(url)) {
    void promptRelogin()
    return
  }

  silentClearAuthData()
  if (authMode !== 'optional') {
    void uni.showToast({
      icon: 'none',
      title: responseData.message || '请求错误',
    })
  }
}

function silentClearAuthData() {
  clearToken()
  uni.removeStorageSync('user')
  uni.$emit(AUTH_SILENT_LOGOUT_EVENT)
}

async function clearUserData() {
  clearToken()
  uni.removeStorageSync('user')
  saveCurrentRoute()
  // token 失效时直接重启页面栈，避免小程序历史页残留。
  uni.reLaunch({
    url: '/pages/login/login',
    fail: () => {
      uni.redirectTo({ url: '/pages/login/login' })
    },
  })
}
