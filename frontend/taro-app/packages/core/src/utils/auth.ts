import Taro from '@tarojs/taro'

const ACCESS_TOKEN_KEY = 'access_token'
const ACCESS_TOKEN_EXPIRES_IN = 'expiresIn'
const REFRESH_TOKEN_KEY = 'refresh_token'
const TOKEN_REFRESH_THRESHOLD = 5 * 60 * 1000

/** 读取访问令牌。 */
export function getToken(): string {
  return Taro.getStorageSync<string>(ACCESS_TOKEN_KEY) || ''
}

/** 保存访问令牌。 */
export function setToken(token: string): void {
  Taro.setStorageSync(ACCESS_TOKEN_KEY, token)
}

/** 读取访问令牌过期时间戳。 */
export function getTokenExpiresIn(): number {
  return Number(Taro.getStorageSync<string>(ACCESS_TOKEN_EXPIRES_IN) || '')
}

/** 按后端返回的秒数保存访问令牌过期时间。 */
export function setTokenExpiresIn(expiresIn: number): void {
  Taro.setStorageSync(ACCESS_TOKEN_EXPIRES_IN, String(Date.now() + expiresIn * 1000))
}

/** 读取刷新令牌。 */
export function getRefreshToken(): string {
  return Taro.getStorageSync<string>(REFRESH_TOKEN_KEY) || ''
}

/** 保存刷新令牌。 */
export function setRefreshToken(token: string): void {
  Taro.setStorageSync(REFRESH_TOKEN_KEY, token)
}

/** 清理本地认证令牌。 */
export function clearToken(): void {
  Taro.removeStorageSync(ACCESS_TOKEN_KEY)
  Taro.removeStorageSync(REFRESH_TOKEN_KEY)
  Taro.removeStorageSync(ACCESS_TOKEN_EXPIRES_IN)
}

/** 判断本地访问令牌是否仍在有效期内。 */
export function hasValidToken(): boolean {
  const expiresIn = getTokenExpiresIn()
  return Boolean(getToken() && expiresIn && expiresIn > Date.now())
}

/** 判断访问令牌是否即将过期，供请求层提前刷新。 */
export function shouldRefreshToken(): boolean {
  const expiresIn = getTokenExpiresIn()
  return Boolean(getToken() && expiresIn && expiresIn - Date.now() <= TOKEN_REFRESH_THRESHOLD)
}
