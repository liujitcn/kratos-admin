import type { UserProfileForm } from '@/rpc/system/app/v1/auth'
import type { CreateOauthSessionRequest, CreateOauthSessionResponse } from '@/rpc/base/v1/oauth'
import type { LoginRequest, LoginResponse } from '@/rpc/base/v1/login'
import { defAuthService } from '@/api/system/auth'
import { defLoginService } from '@/api/base/login'
import { defOauthService } from '@/api/base/oauth'
import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  setToken,
  setRefreshToken,
  getRefreshToken,
  clearToken,
  setTokenExpiresIn,
  hasValidToken,
} from '@/utils/auth'

const AUTH_SILENT_LOGOUT_EVENT = 'auth:silent-logout'
let silentLogoutEventHandler: (() => void) | undefined

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

    /** 保存登录接口返回的认证令牌。 */
    function applyLoginToken(data: LoginResponse | CreateOauthSessionResponse) {
      const { token_type, access_token, refresh_token, expires_in } = data
      setToken(token_type + ' ' + access_token)
      setRefreshToken(refresh_token)
      setTokenExpiresIn(expires_in)
    }

    /**
     * 登录
     *
     * @param request
     * @returns
     */
    function login(request: LoginRequest) {
      return new Promise<void>((resolve, reject) => {
        defLoginService
          .Login(request)
          .then((data) => {
            applyLoginToken(data)
            resolve()
          })
          .catch((error) => {
            reject(error)
          })
      })
    }

    /**
     * 创建三方登录会话
     *
     * @param request
     * @returns
     */
    function createOauthSession(request: CreateOauthSessionRequest) {
      return new Promise<void>((resolve, reject) => {
        defOauthService
          .CreateOauthSession(request)
          .then((data) => {
            applyLoginToken(data)
            resolve()
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
    function clearUserData() {
      return new Promise<void>((resolve) => {
        clearToken()
        userInfo.value = undefined
        resolve()
      })
    }

    /** 静默清理登录态，用于 token 失效后降级为游客，不主动跳登录页。 */
    function silentLogout() {
      clearToken()
      userInfo.value = undefined
      uni.removeStorageSync('user')
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
    }
    uni.$on(AUTH_SILENT_LOGOUT_EVENT, silentLogoutEventHandler)

    return {
      userInfo,
      isAuthenticated,
      getUserProfile,
      login,
      createOauthSession,
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
