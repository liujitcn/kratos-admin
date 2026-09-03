import type { UserProfileForm } from '../../rpc/system/app/v1/auth'
import type {
  BindOauthSessionRequest,
  CreateOauthSessionRequest,
  CreateOauthSessionResponse,
} from '../../rpc/base/v1/oauth'
import type { LoginRequest, LoginResponse } from '../../rpc/base/v1/login'
import type { VerifyMfaRequest } from '../../rpc/base/v1/mfa'
import { defMfaService } from '../../api/base/v1/mfa'
import { defAuthService } from '../../api/system/app/v1/auth'
import { defLoginService } from '../../api/base/v1/login'
import { defOauthService } from '../../api/base/v1/oauth'
import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  setToken,
  setRefreshToken,
  getRefreshToken,
  clearToken,
  setTokenExpiresIn,
  hasValidToken,
} from '../../utils/auth'

const AUTH_SILENT_LOGOUT_EVENT = 'auth:silent-logout'
let silentLogoutEventHandler: (() => void) | undefined

/** 用户状态扩展生命周期处理器。 */
export interface UserStoreExtension {
  /** 登录令牌保存完成后执行。 */
  onLogin?: () => void | Promise<void>
  /** 用户主动登出并清理本地状态后执行。 */
  onLogout?: () => void | Promise<void>
  /** 令牌失效并静默清理本地状态后执行。 */
  onSilentLogout?: () => void | Promise<void>
}

const userStoreExtensions = new Set<UserStoreExtension>()

/** 注册用户 Store 的业务扩展，返回值用于注销本次注册。 */
export function registerUserStoreExtension(extension: UserStoreExtension) {
  userStoreExtensions.add(extension)
  return () => userStoreExtensions.delete(extension)
}

async function runUserStoreExtensions(event: keyof UserStoreExtension) {
  for (const extension of userStoreExtensions) {
    const handler = extension[event]
    if (!handler) {
      continue
    }
    try {
      await handler()
    } catch (error) {
      console.warn(`user store ${event} extension failed`, error)
    }
  }
}

// 定义 Store
export const useUserStore = defineStore(
  'user',
  () => {
    // 会员信息
    const userInfo = ref<UserProfileForm>()

    /** 判断当前本地登录态是否仍然可用。 */
    function isAuthenticated() {
      return Boolean(userInfo.value && hasValidToken())
    }

    /** 保存登录接口返回的认证令牌，并通知已注册的业务扩展。 */
    async function applyLoginToken(data: LoginResponse | CreateOauthSessionResponse) {
      const { token_type, access_token, refresh_token, expires_in } = data
      setToken(token_type + ' ' + access_token)
      setRefreshToken(refresh_token)
      setTokenExpiresIn(expires_in)
      await runUserStoreExtensions('onLogin')
    }

    /**
     * 登录
     *
     * @param request
     * @returns
     */
    function login(request: LoginRequest) {
      return new Promise<LoginResponse>((resolve, reject) => {
        defLoginService
          .Login(request)
          .then(async (data) => {
            if (data.status === 1 || data.status === 0 || data.status === 4)
              await applyLoginToken(data)
            resolve(data)
          })
          .catch((error) => {
            reject(error)
          })
      })
    }

    /** 校验登录阶段的多因素认证并保存正式令牌。 */
    function verifyMfa(request: VerifyMfaRequest) {
      return new Promise<LoginResponse>((resolve, reject) => {
        defMfaService
          .VerifyMfa(request)
          .then(async (data) => {
            await applyLoginToken(data)
            resolve(data)
          })
          .catch((error) => reject(error))
      })
    }

    /**
     * 创建三方登录会话
     *
     * @param request
     * @returns
     */
    function createOauthSession(request: CreateOauthSessionRequest) {
      return new Promise<CreateOauthSessionResponse>((resolve, reject) => {
        defOauthService
          .CreateOauthSession(request)
          .then(async (data) => {
            if (data.binding_required) {
              resolve(data)
              return
            }
            if (data.status === 1 || data.status === 0 || data.status === 4)
              await applyLoginToken(data)
            resolve(data)
          })
          .catch((error) => {
            reject(error)
          })
      })
    }

    /** 绑定已有账号并创建三方登录会话。 */
    function bindOauthSession(request: BindOauthSessionRequest) {
      return new Promise<CreateOauthSessionResponse>((resolve, reject) => {
        defOauthService
          .BindOauthSession(request)
          .then(async (data) => {
            if (data.status === 1 || data.status === 0 || data.status === 4)
              await applyLoginToken(data)
            resolve(data)
          })
          .catch((error) => {
            reject(error)
          })
      })
    }

    /**
     * 获取用户资料
     */
    function getUserProfile() {
      return new Promise<UserProfileForm>((resolve, reject) => {
        defAuthService
          .GetUserProfile({})
          .then((data) => {
            if (!data) {
              reject('Verification failed, please Login again.')
              return
            }
            userInfo.value = data
            resolve(data)
          })
          .catch((error) => {
            reject(error)
          })
      })
    }

    /**
     * 登出
     */
    function logout() {
      return new Promise<void>((resolve, reject) => {
        defLoginService
          .Logout({})
          .then(() => {
            clearUserData().then(() => {
              resolve()
            })
          })
          .catch((error) => {
            reject(error)
          })
      })
    }

    /**
     * 刷新 token
     */
    function refreshToken() {
      const refreshToken = getRefreshToken()
      return new Promise<void>((resolve, reject) => {
        defLoginService
          .RefreshToken({
            refresh_token: refreshToken,
          })
          .then((data) => {
            const { token_type, access_token, refresh_token, expires_in } = data
            setToken(token_type + ' ' + access_token)
            setRefreshToken(refresh_token)
            setTokenExpiresIn(expires_in)
            resolve()
          })
          .catch((error) => {
            console.log(' refreshToken  刷新失败', error)
            reject(error)
          })
      })
    }

    /**
     * 清理用户数据
     *
     * @returns
     */
    async function clearUserData() {
      clearToken()
      userInfo.value = undefined
      await runUserStoreExtensions('onLogout')
    }

    /** 静默清理登录态，用于 token 失效后降级为游客，不主动跳登录页。 */
    function silentLogout() {
      clearToken()
      userInfo.value = undefined
      uni.removeStorageSync('user')
      void runUserStoreExtensions('onSilentLogout')
    }

    /** 确认必须登录的操作是否可继续，不可继续时交给调用方跳登录。 */
    function ensureAuthenticated() {
      if (isAuthenticated()) {
        return true
      }
      silentLogout()
      return false
    }

    if (silentLogoutEventHandler) {
      uni.$off(AUTH_SILENT_LOGOUT_EVENT, silentLogoutEventHandler)
    }
    silentLogoutEventHandler = () => {
      userInfo.value = undefined
      void runUserStoreExtensions('onSilentLogout')
    }
    uni.$on(AUTH_SILENT_LOGOUT_EVENT, silentLogoutEventHandler)

    return {
      userInfo,
      isAuthenticated,
      getUserProfile,
      login,
      verifyMfa,
      createOauthSession,
      bindOauthSession,
      logout,
      clearUserData,
      silentLogout,
      ensureAuthenticated,
      refreshToken,
    }
  },
  {
    // 网页端配置
    // persist: true,
    // 小程序端配置
    persist: {
      storage: {
        getItem(key) {
          return uni.getStorageSync(key)
        },
        setItem(key, value) {
          uni.setStorageSync(key, value)
        },
      },
    },
  },
)
