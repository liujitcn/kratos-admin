import Taro from '@tarojs/taro'
import {
  clearToken,
  getRefreshToken,
  getToken,
  hasValidToken,
  setRefreshToken,
  setToken,
  setTokenExpiresIn,
  shouldRefreshToken,
} from './auth'
import { saveCurrentRoute } from './navigation'

const apiBasePath = process.env.KRATOS_TARO_API_BASE || '/api'
const apiTargetUrl = process.env.KRATOS_TARO_API_URL || ''
const normalizedApiBasePath = apiBasePath.startsWith('/') ? apiBasePath : `/${apiBasePath}`
const requestOrigin = process.env.TARO_ENV === 'h5' ? '' : apiTargetUrl.replace(/\/$/, '')
/** 请求基础地址。 */
export const requestBaseURL = `${requestOrigin}${normalizedApiBasePath}`

const SESSION_URL = '/v1/base/session'
const REFRESH_TOKEN_URL = '/v1/base/token'
const CAPTCHA_URL = '/v1/base/captcha'
const CONFIG_URL = '/v1/base/config'
const PASSWORD_PUBLIC_KEY_URL = '/v1/base/password-public-key'
const NO_AUTH_URL_SET = new Set([
  SESSION_URL,
  REFRESH_TOKEN_URL,
  CAPTCHA_URL,
  CONFIG_URL,
  PASSWORD_PUBLIC_KEY_URL,
  '/auth',
  '/login/captcha',
  '/auth/token',
])
const AUTH_EXPIRED_EXCLUDED_URL_SET = new Set([
  SESSION_URL,
  CAPTCHA_URL,
  CONFIG_URL,
  PASSWORD_PUBLIC_KEY_URL,
  '/auth',
  '/login/captcha',
])
/** 认证状态被静默清理的事件名。 */
export const AUTH_SILENT_LOGOUT_EVENT = 'auth:silent-logout'

/** 接口认证模式。 */
export type AuthMode = 'none' | 'optional' | 'required'
/** Taro 请求参数扩展。 */
export type HttpRequestOptions = Taro.request.Option & { authMode?: AuthMode }

type ErrorData = {
  code?: string | number
  message?: string
  reason?: string | number
}

const authErrorCodeSet = new Set(['401', '403'])
const authErrorReasonSet = new Set(['UNAUTHENTICATED', 'PERMISSION_DENIED'])
let refreshTokenPromise: Promise<void> | null = null
let isPromptingRelogin = false

function isAuthErrorResponse(data: unknown): boolean {
  if (!data || typeof data !== 'object') return false
  const response = data as ErrorData
  const code = response.code === undefined ? '' : String(response.code)
  const reason = response.reason === undefined ? '' : String(response.reason)
  return authErrorCodeSet.has(code) || authErrorCodeSet.has(reason) || authErrorReasonSet.has(reason)
}

function resolveAuthMode(options: HttpRequestOptions, url: string): AuthMode {
  if (options.authMode) return options.authMode
  if (options.header?.Authorization === 'no-auth' || NO_AUTH_URL_SET.has(url)) return 'none'
  return 'required'
}

function resolveRequestUrl(url: string): string {
  return /^https?:\/\//.test(url) ? url : `${requestBaseURL}${url.startsWith('/') ? url : `/${url}`}`
}

/** 发送经过认证、刷新令牌和统一错误处理的 Taro 请求。 */
export async function http<T>(options: HttpRequestOptions): Promise<T> {
  return sendRequest<T>(options, false)
}

async function sendRequest<T>(
  options: HttpRequestOptions,
  retriedAsAnonymous: boolean,
): Promise<T> {
  const requestUrl = String(options.url)
  const authMode = resolveAuthMode(options, requestUrl)
  let accessToken = ''
  try {
    accessToken = await getAccessTokenByMode(
      retriedAsAnonymous && authMode === 'optional' ? 'none' : authMode,
    )
  } catch (error) {
    if (authMode === 'optional' && !retriedAsAnonymous) {
      silentClearAuthData()
      return sendRequest<T>(options, true)
    }
    throw error
  }

  try {
    const response = await Taro.request({
      ...options,
      url: resolveRequestUrl(requestUrl),
      timeout: options.timeout || 10000,
      header: {
        ...options.header,
        'source-client': 'miniapp',
        ...(accessToken ? { Authorization: accessToken } : {}),
      },
    })
    const responseData = response.data as ErrorData
    if (response.statusCode >= 200 && response.statusCode < 300 && !isAuthErrorResponse(responseData)) {
      return response.data as T
    }
    if (
      response.statusCode === 401 ||
      response.statusCode === 403 ||
      isAuthErrorResponse(responseData)
    ) {
      if (authMode === 'optional' && !retriedAsAnonymous) {
        silentClearAuthData()
        return sendRequest<T>(options, true)
      }
      handleAuthExpiredByMode(authMode, requestUrl, responseData)
      throw response
    }
    await Taro.showToast({ icon: 'none', title: responseData.message || '请求错误' })
    throw response
  } catch (error) {
    if (
      typeof error === 'object' &&
      error &&
      ('statusCode' in error || String((error as { errMsg?: string }).errMsg).includes('abort'))
    ) {
      throw error
    }
    await Taro.showToast({ icon: 'none', title: '网络错误，换个网络试试' })
    throw error
  }
}

/** 获取请求可用访问令牌，供流式请求复用。 */
export async function getRequestAccessToken(authMode: AuthMode = 'required'): Promise<string> {
  return getAccessTokenByMode(authMode)
}

/** 触发登录失效处理，供流式请求复用。 */
export function handleAuthExpired(authMode: AuthMode = 'required'): void {
  if (authMode === 'required') void promptRelogin()
  else silentClearAuthData()
}

async function getAccessTokenByMode(authMode: AuthMode): Promise<string> {
  if (authMode === 'none') return ''
  if (!getToken()) {
    if (authMode === 'required') {
      await promptRelogin()
      throw new Error('auth required')
    }
    return ''
  }
  if (shouldRefreshToken()) await handleTokenRefresh(authMode)
  if (hasValidToken()) return getToken()
  if (authMode === 'required') {
    await promptRelogin()
    throw new Error('auth expired')
  }
  silentClearAuthData()
  return ''
}

function handleTokenRefresh(authMode: AuthMode): Promise<void> {
  if (refreshTokenPromise) return refreshTokenPromise
  refreshTokenPromise = refreshAccessToken()
    .catch(async (error) => {
      if (authMode === 'required') await promptRelogin()
      else silentClearAuthData()
      throw error
    })
    .finally(() => {
      refreshTokenPromise = null
    })
  return refreshTokenPromise
}

async function refreshAccessToken(): Promise<void> {
  const refreshToken = getRefreshToken()
  if (!refreshToken) throw new Error('refresh token missing')
  const response = await Taro.request({
    url: resolveRequestUrl(REFRESH_TOKEN_URL),
    method: 'POST',
    data: { refresh_token: refreshToken },
    header: { 'source-client': 'miniapp' },
  })
  const data = response.data as ErrorData & {
    token_type?: string
    access_token?: string
    refresh_token?: string
    expires_in?: number
  }
  if (
    response.statusCode < 200 ||
    response.statusCode >= 300 ||
    isAuthErrorResponse(data) ||
    !data.token_type ||
    !data.access_token ||
    !data.refresh_token ||
    !data.expires_in
  ) {
    throw response
  }
  setToken(`${data.token_type} ${data.access_token}`)
  setRefreshToken(data.refresh_token)
  setTokenExpiresIn(data.expires_in)
}

async function promptRelogin(): Promise<void> {
  if (isPromptingRelogin) return
  isPromptingRelogin = true
  try {
    const modal = await Taro.showModal({
      title: '提示',
      content: '当前页面已失效，请重新登录',
      showCancel: false,
      confirmText: '重新登录',
    })
    if (!modal.confirm) return
    await new Promise((resolve) => setTimeout(resolve, 80))
    clearToken()
    Taro.removeStorageSync('user')
    saveCurrentRoute()
    await Taro.reLaunch({ url: '/pages/login/login' })
  } finally {
    isPromptingRelogin = false
  }
}

function handleAuthExpiredByMode(
  authMode: AuthMode,
  url: string,
  responseData: ErrorData,
): void {
  if (authMode === 'required' && !AUTH_EXPIRED_EXCLUDED_URL_SET.has(url)) {
    void promptRelogin()
    return
  }
  silentClearAuthData()
  if (authMode !== 'optional') {
    void Taro.showToast({ icon: 'none', title: responseData.message || '请求错误' })
  }
}

function silentClearAuthData(): void {
  clearToken()
  Taro.removeStorageSync('user')
  Taro.eventCenter.trigger(AUTH_SILENT_LOGOUT_EVENT)
}
