import Taro, { getCurrentPages } from '@tarojs/taro'
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
import { getLocaleRequestHeaders, t } from '../locales'
import { useThemeStore } from '../stores/theme'

const apiBasePath = process.env.VITE_APP_BASE_API || '/api'
const apiTargetUrl = process.env.VITE_APP_API_URL || ''
const normalizedApiBasePath = apiBasePath.startsWith('/') ? apiBasePath : `/${apiBasePath}`
const requestOrigin = process.env.TARO_ENV === 'h5' ? '' : apiTargetUrl.replace(/\/$/, '')
/** 请求基础地址。 */
export const requestBaseURL = `${requestOrigin}${normalizedApiBasePath}`
/** 站点根地址（不含 API 基础路径），供事件流等非 API 端点复用。 */
export const siteBaseURL =
  process.env.TARO_ENV === 'h5'
    ? typeof window !== 'undefined'
      ? window.location.origin
      : ''
    : apiTargetUrl.replace(/\/$/, '')
export const sourceClient = process.env.TARO_ENV === 'weapp' ? 'taro-weapp' : 'taro-h5'

const SESSION_URL = '/v1/base/session'
const REFRESH_TOKEN_URL = '/v1/base/token'
const CAPTCHA_URL = '/v1/base/captcha'
const CONFIG_URL = '/v1/base/config'
const LANGUAGE_URL = '/v1/base/language'
const SECRET_PUBLIC_KEY_URL = '/v1/base/secret-public-key'
const NO_AUTH_URL_SET = new Set([
  SESSION_URL,
  REFRESH_TOKEN_URL,
  CAPTCHA_URL,
  CONFIG_URL,
  LANGUAGE_URL,
  SECRET_PUBLIC_KEY_URL,
])
const AUTH_EXPIRED_EXCLUDED_URL_SET = new Set([
  SESSION_URL,
  CAPTCHA_URL,
  CONFIG_URL,
  LANGUAGE_URL,
  SECRET_PUBLIC_KEY_URL,
])
/** 认证状态被静默清理的事件名。 */
export const AUTH_SILENT_LOGOUT_EVENT = 'auth:silent-logout'

/** 接口认证模式。 */
export type AuthMode = 'none' | 'optional' | 'required'
/** Taro 请求参数扩展。 */
export type HttpRequestOptions = Taro.request.Option & { authMode?: AuthMode }

/** 认证缺失或失效错误：抛出前请求层已弹窗引导重新登录，页面侧无需重复提示。 */
export class AuthPromptError extends Error {
  constructor() {
    super(t('core.auth.session_expired'))
    this.name = 'AuthPromptError'
  }
}

/** 判断错误是否为登录态失效（请求层已弹窗提示，或响应为 401/403）。 */
export function isAuthExpiredError(error: unknown): boolean {
  if (error && typeof error === 'object') {
    const statusCode = (error as { statusCode?: number }).statusCode
    if (statusCode === 401 || statusCode === 403) return true
  }
  return error instanceof AuthPromptError
}

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
  return (
    authErrorCodeSet.has(code) || authErrorCodeSet.has(reason) || authErrorReasonSet.has(reason)
  )
}

/** 识别响应体中的登录态失效（401/UNAUTHENTICATED）；403 属权限问题，不应触发重新登录。 */
function isUnauthenticatedResponse(data: unknown): boolean {
  if (!data || typeof data !== 'object') return false
  const response = data as ErrorData
  const code = response.code === undefined ? '' : String(response.code)
  const reason = response.reason === undefined ? '' : String(response.reason)
  return code === '401' || reason === '401' || reason === 'UNAUTHENTICATED'
}

function resolveAuthMode(options: HttpRequestOptions, url: string): AuthMode {
  if (options.authMode) return options.authMode
  if (options.header?.Authorization === 'no-auth' || NO_AUTH_URL_SET.has(url)) return 'none'
  return 'required'
}

function resolveRequestUrl(url: string): string {
  return /^https?:\/\//.test(url)
    ? url
    : `${requestBaseURL}${url.startsWith('/') ? url : `/${url}`}`
}

/** 发送经过认证、刷新令牌和统一错误处理的 Taro 请求。 */
export async function http<T>(options: HttpRequestOptions): Promise<T> {
  return sendRequest<T>(options, false, false)
}

async function sendRequest<T>(
  options: HttpRequestOptions,
  retriedAsAnonymous: boolean,
  authRetried: boolean,
): Promise<T> {
  const requestUrl = String(options.url)
  const authMode = resolveAuthMode(options, requestUrl)
  // 401 时单飞刷新令牌并重放原请求一次，返回 true 表示应重放；仍失败才引导重新登录。
  const retryAfterTokenRefresh = async (): Promise<boolean> => {
    if (authRetried || authMode !== 'required') return false
    try {
      await handleTokenRefresh()
    } catch {
      return false
    }
    return true
  }
  let accessToken = ''
  try {
    accessToken = await getAccessTokenByMode(
      retriedAsAnonymous && authMode === 'optional' ? 'none' : authMode,
    )
  } catch (error) {
    if (authMode === 'optional' && !retriedAsAnonymous) {
      silentClearAuthData()
      return sendRequest<T>(options, true, authRetried)
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
        'source-client': sourceClient,
        ...getLocaleRequestHeaders(),
        ...getLoginAgreementHeaders(),
        ...(accessToken ? { Authorization: accessToken } : {}),
      },
    })
    const responseData = response.data as ErrorData | null | undefined
    if (response.statusCode >= 200 && response.statusCode < 300) {
      if (isUnauthenticatedResponse(responseData)) {
        if (authMode === 'optional' && !retriedAsAnonymous) {
          silentClearAuthData()
          return sendRequest<T>(options, true, authRetried)
        }
        if (await retryAfterTokenRefresh()) {
          return sendRequest<T>(options, retriedAsAnonymous, true)
        }
        handleAuthExpiredByMode(authMode, requestUrl, responseData)
        throw response
      }
      if (isAuthErrorResponse(responseData)) {
        // 403/PERMISSION_DENIED 属权限不足：按普通业务错误提示，不触发重新登录。
        await Taro.showToast({
          icon: 'none',
          title: responseData?.message || t('common.message.request_error'),
        })
        throw response
      }
      return response.data as T
    }
    if (response.statusCode === 401) {
      // 401 先刷新令牌重放一次；仍失效才清理用户信息，跳转到登录页
      if (authMode === 'optional' && !retriedAsAnonymous) {
        silentClearAuthData()
        return sendRequest<T>(options, true, authRetried)
      }
      if (await retryAfterTokenRefresh()) {
        return sendRequest<T>(options, retriedAsAnonymous, true)
      }
      handleAuthExpiredByMode(authMode, requestUrl, responseData)
      throw response
    }
    if (response.statusCode === 403) {
      // 403 属权限不足：按普通业务错误提示，不触发重新登录。
      await Taro.showToast({
        icon: 'none',
        title: responseData?.message || t('common.message.request_error'),
      })
      throw response
    }
    await Taro.showToast({
      icon: 'none',
      title: responseData?.message || t('common.message.request_error'),
    })
    throw response
  } catch (error) {
    if (
      typeof error === 'object' &&
      error &&
      ('statusCode' in error || String((error as { errMsg?: string }).errMsg).includes('abort'))
    ) {
      throw error
    }
    await Taro.showToast({ icon: 'none', title: t('common.message.network_error') })
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
      throw new AuthPromptError()
    }
    return ''
  }
  if (shouldRefreshToken()) {
    try {
      await handleTokenRefresh()
    } catch {
      // 刷新失败已撤销本地认证，交由下方 hasValidToken 分支按当前调用方 authMode 处理。
    }
  }
  if (hasValidToken()) return getToken()
  if (authMode === 'required') {
    await promptRelogin()
    throw new AuthPromptError()
  }
  silentClearAuthData()
  return ''
}

function handleTokenRefresh(): Promise<void> {
  if (refreshTokenPromise) return refreshTokenPromise
  refreshTokenPromise = refreshAccessTokenWithCrossTabLock()
    .catch((error) => {
      // 仅当刷新接口明确判定令牌失效时才撤销本地认证；网络抖动/超时保留令牌下次再试，
      // 避免一次瞬时失败把用户踢回登录页。
      if (isRefreshTokenUnauthorized(error)) {
        silentClearAuthData()
      }
      throw error
    })
    .finally(() => {
      refreshTokenPromise = null
    })
  return refreshTokenPromise
}

/** 供流式请求在 401 后强制续签一次；成功后用 getRequestAccessToken 取新令牌重放。 */
export function refreshSessionToken(): Promise<void> {
  return handleTokenRefresh()
}

type WebLocksManager = {
  request: (name: string, callback: () => Promise<void>) => Promise<void>
}

/** H5 端用 Web Locks 串行化跨标签页的刷新令牌轮换，避免并发刷新在后端轮换式刷新下互相踢下线。 */
async function refreshAccessTokenWithCrossTabLock(): Promise<void> {
  const locks =
    typeof navigator !== 'undefined'
      ? (navigator as unknown as { locks?: WebLocksManager }).locks
      : undefined
  if (locks) {
    await locks.request('kratos-taro-app:refresh-token', () => refreshAccessToken())
    return
  }
  await refreshAccessToken()
}

/** 判断刷新失败是否源于刷新令牌被服务端明确拒绝；其余失败一律保留本地登录态。 */
function isRefreshTokenUnauthorized(error: unknown): boolean {
  if (!error || typeof error !== 'object') return false
  const response = error as { statusCode?: number; data?: unknown }
  if (typeof response.statusCode !== 'number') return false
  return response.statusCode === 401 || isUnauthenticatedResponse(response.data)
}

async function refreshAccessToken(): Promise<void> {
  const refreshToken = getRefreshToken()
  if (!refreshToken) throw new Error('refresh token missing')
  const response = await Taro.request({
    url: resolveRequestUrl(REFRESH_TOKEN_URL),
    method: 'POST',
    data: { refresh_token: refreshToken },
    header: { 'source-client': sourceClient, ...getLocaleRequestHeaders() },
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

/** 会话过期弹窗等待时长：H5 端路由切换会自动隐藏模态且 Promise 永不结算，超时释放避免提示永久卡死。 */
const RELOGIN_PROMPT_TIMEOUT = 60_000

/** 提示会话过期并进入携带本次业务回跳标记的登录页。 */
async function promptRelogin(): Promise<void> {
  // 401 来自旧会话的在途请求（含刚续签却被服务端拒绝的重放）而当前仍是有效登录态时直接忽略，
  // 避免刚登录成功被弹窗打断；请求层已做过刷新重放，这里不再重复续签。
  if (hasValidToken()) {
    return
  }
  const pages = getCurrentPages()
  // 登录页自身的请求失效只清理残留登录态，避免确认后重载登录页丢失已填写的表单。
  if (pages[pages.length - 1]?.route === 'pages/login/login') {
    if (getToken() || getRefreshToken() || Taro.getStorageSync('user')) silentClearAuthData()
    return
  }
  if (isPromptingRelogin) return
  isPromptingRelogin = true
  try {
    const colors = useThemeStore.getState().colors
    const modal = await Promise.race([
      // 按钮文案复用通用确认键：平台对 confirmText 限制 4 个汉字/8 字符，长文案会被整体拒绝。
      Taro.showModal({
        title: t('common.title.notice'),
        content: t('core.auth.session_expired'),
        showCancel: false,
        confirmText: t('common.action.confirm'),
        confirmColor: colors.primary,
      }).catch(() => undefined as undefined),
      new Promise<undefined>((resolve) =>
        setTimeout(() => resolve(undefined), RELOGIN_PROMPT_TIMEOUT),
      ),
    ])
    if (!modal?.confirm) return
    await new Promise((resolve) => setTimeout(resolve, 80))
    silentClearAuthData()
    saveCurrentRoute()
    await Taro.reLaunch({ url: '/pages/login/login?redirect=1' })
  } finally {
    isPromptingRelogin = false
  }
}

function handleAuthExpiredByMode(
  authMode: AuthMode,
  url: string,
  responseData: ErrorData | null | undefined,
): void {
  if (authMode === 'required' && !AUTH_EXPIRED_EXCLUDED_URL_SET.has(url)) {
    void promptRelogin()
    return
  }
  silentClearAuthData()
  if (authMode !== 'optional') {
    void Taro.showToast({
      icon: 'none',
      title: responseData?.message || t('common.message.request_error'),
    })
  }
}

function silentClearAuthData(): void {
  clearToken()
  Taro.removeStorageSync('user')
  Taro.eventCenter.trigger(AUTH_SILENT_LOGOUT_EVENT)
}

/** 当前登录流程主动确认的协议摘要，完成认证后清除。 */
let acceptedAgreementVersion = ''
let agreementAcceptedAt = 0

/** 由已通过现有协议确认的用户事件记录摘要，不改变页面勾选状态。 */
export function setLoginAgreement(version: string) {
  acceptedAgreementVersion = version
  agreementAcceptedAt = Date.now()
}

/** 仅为有效的本次确认提供审计请求头。 */
function getLoginAgreementHeaders(): Record<string, string> {
  if (!/^[a-f0-9]{64}$/.test(acceptedAgreementVersion) || Date.now() - agreementAcceptedAt > 10 * 60 * 1000) return {}
  return { 'X-Agreement-Accepted': 'true', 'X-Agreement-Version': acceptedAgreementVersion, 'X-Agreement-Site': '3' }
}
