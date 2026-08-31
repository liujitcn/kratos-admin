import Taro from '@tarojs/taro'
import { create } from 'zustand'
import { defLoginService } from '../api/base/login'
import { defOauthService } from '../api/base/oauth'
import { defAuthService } from '../api/system/auth'
import type { LoginRequest, LoginResponse } from '../rpc/base/v1/login'
import type { VerifyMfaRequest } from '../rpc/base/v1/mfa'
import { defMfaService } from '../api/base/mfa'
import type {
  BindOauthSessionRequest,
  CreateOauthSessionRequest,
  CreateOauthSessionResponse,
} from '../rpc/base/v1/oauth'
import type { UserProfileForm } from '../rpc/system/app/v1/auth'
import {
  clearToken,
  getRefreshToken,
  hasValidToken,
  setRefreshToken,
  setToken,
  setTokenExpiresIn,
} from '../utils/auth'
import { AUTH_SILENT_LOGOUT_EVENT } from '../utils/http'

const USER_STORAGE_KEY = 'user'

/** 用户状态扩展生命周期处理器。 */
export interface UserStoreExtension {
  /** 登录令牌保存完成后执行。 */
  onLogin?: () => void | Promise<void>
  /** 用户主动登出并清理本地状态后执行。 */
  onLogout?: () => void | Promise<void>
  /** 令牌失效并静默清理本地状态后执行。 */
  onSilentLogout?: () => void | Promise<void>
}

/** 用户状态及操作。 */
export interface UserStoreState {
  userInfo?: UserProfileForm
  hydrated: boolean
  hydrate: () => void
  isAuthenticated: () => boolean
  applyLoginToken: (data: LoginResponse | CreateOauthSessionResponse) => Promise<void>
  login: (request: LoginRequest) => Promise<LoginResponse>
  verifyMfa: (request: VerifyMfaRequest) => Promise<LoginResponse>
  createOauthSession: (
    request: CreateOauthSessionRequest,
  ) => Promise<CreateOauthSessionResponse>
  bindOauthSession: (request: BindOauthSessionRequest) => Promise<CreateOauthSessionResponse>
  getUserProfile: () => Promise<UserProfileForm>
  logout: () => Promise<void>
  refreshToken: () => Promise<void>
  clearUserData: () => Promise<void>
  silentLogout: () => void
  ensureAuthenticated: () => boolean
}

const userStoreExtensions = new Set<UserStoreExtension>()
let eventBridgeStarted = false

/** 注册用户 Store 的业务扩展，返回注销函数。 */
export function registerUserStoreExtension(extension: UserStoreExtension): () => void {
  userStoreExtensions.add(extension)
  return () => userStoreExtensions.delete(extension)
}

async function runUserStoreExtensions(event: keyof UserStoreExtension): Promise<void> {
  for (const extension of userStoreExtensions) {
    try {
      await extension[event]?.()
    } catch (error) {
      console.warn(`user store ${event} extension failed`, error)
    }
  }
}

function persistUserInfo(userInfo?: UserProfileForm): void {
  if (userInfo) Taro.setStorageSync(USER_STORAGE_KEY, { userInfo })
  else Taro.removeStorageSync(USER_STORAGE_KEY)
}

/** 用户 Zustand Store。 */
export const useUserStore = create<UserStoreState>((set, get) => ({
  userInfo: undefined,
  hydrated: false,
  hydrate() {
    const cached = Taro.getStorageSync<{ userInfo?: UserProfileForm }>(USER_STORAGE_KEY)
    set({ userInfo: cached?.userInfo, hydrated: true })
  },
  isAuthenticated() {
    return Boolean(get().userInfo && hasValidToken())
  },
  async applyLoginToken(data) {
    setToken(`${data.token_type} ${data.access_token}`)
    setRefreshToken(data.refresh_token)
    setTokenExpiresIn(data.expires_in)
    await runUserStoreExtensions('onLogin')
  },
  async login(request) {
    const response = await defLoginService.Login(request)
    if (response.status === 1 || response.status === 0 || response.status === 4) await get().applyLoginToken(response)
    return response
  },
  async verifyMfa(request) {
    const response = await defMfaService.VerifyMfa(request)
    await get().applyLoginToken(response)
    return response
  },
  async createOauthSession(request) {
    const response = await defOauthService.CreateOauthSession(request)
    if (!response.binding_required && (response.status === 1 || response.status === 0 || response.status === 4)) await get().applyLoginToken(response)
    return response
  },
  async bindOauthSession(request) {
    const response = await defOauthService.BindOauthSession(request)
    if (response.status === 1 || response.status === 0 || response.status === 4) await get().applyLoginToken(response)
    return response
  },
  async getUserProfile() {
    const profile = await defAuthService.GetUserProfile({})
    if (!profile) throw new Error('Verification failed, please Login again.')
    set({ userInfo: profile })
    persistUserInfo(profile)
    return profile
  },
  async logout() {
    await defLoginService.Logout({})
    await get().clearUserData()
  },
  async refreshToken() {
    const response = await defLoginService.RefreshToken({ refresh_token: getRefreshToken() })
    setToken(`${response.token_type} ${response.access_token}`)
    setRefreshToken(response.refresh_token)
    setTokenExpiresIn(response.expires_in)
  },
  async clearUserData() {
    clearToken()
    persistUserInfo()
    set({ userInfo: undefined })
    await runUserStoreExtensions('onLogout')
  },
  silentLogout() {
    clearToken()
    persistUserInfo()
    set({ userInfo: undefined })
    void runUserStoreExtensions('onSilentLogout')
  },
  ensureAuthenticated() {
    if (get().isAuthenticated()) return true
    get().silentLogout()
    return false
  },
}))

/** 启动请求层与用户 Store 之间的静默登出事件桥。 */
export function startUserStoreEventBridge(): void {
  if (eventBridgeStarted) return
  eventBridgeStarted = true
  Taro.eventCenter.on(AUTH_SILENT_LOGOUT_EVENT, () => {
    persistUserInfo()
    useUserStore.setState({ userInfo: undefined })
    void runUserStoreExtensions('onSilentLogout')
  })
}
