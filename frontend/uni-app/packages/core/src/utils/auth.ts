// 访问 token 缓存的 key
const ACCESS_TOKEN_KEY = 'access_token'
// 访问 token 缓存的 key的有效期
const ACCESS_TOKEN_EXPIRES_IN = 'expiresIn'
// 刷新 token 缓存的 key
const REFRESH_TOKEN_KEY = 'refresh_token'

const TOKEN_REFRESH_THRESHOLD = 5 * 60 * 1000

/** 读取本地缓存的访问令牌（含类型前缀）。 */
function getToken(): string {
  return uni.getStorageSync(ACCESS_TOKEN_KEY) || ''
}

/** 保存访问令牌到本地缓存。 */
function setToken(token: string) {
  uni.setStorageSync(ACCESS_TOKEN_KEY, token)
}

/** 读取访问令牌的绝对过期时间戳（毫秒）。 */
function getTokenExpiresIn(): number {
  const s = uni.getStorageSync(ACCESS_TOKEN_EXPIRES_IN) || ''
  return Number(s)
}

/** 把服务端返回的有效期秒数换算为绝对过期时间戳后保存。 */
function setTokenExpiresIn(expires_in: number) {
  // 获取当前时间（毫秒）
  const d = new Date().getTime() + expires_in * 1000
  uni.setStorageSync(ACCESS_TOKEN_EXPIRES_IN, String(d))
}

/** 读取本地缓存的刷新令牌。 */
function getRefreshToken(): string {
  return uni.getStorageSync(REFRESH_TOKEN_KEY) || ''
}

/** 保存刷新令牌到本地缓存。 */
function setRefreshToken(token: string) {
  uni.setStorageSync(REFRESH_TOKEN_KEY, token)
}

/** 清空全部本地登录令牌（访问令牌、刷新令牌、过期时间）。 */
function clearToken() {
  uni.removeStorageSync(ACCESS_TOKEN_KEY)
  uni.removeStorageSync(REFRESH_TOKEN_KEY)
  uni.removeStorageSync(ACCESS_TOKEN_EXPIRES_IN)
}

/** 判断本地访问令牌是否仍在有效期内。 */
function hasValidToken(): boolean {
  const token = getToken()
  const expiresIn = getTokenExpiresIn()
  return Boolean(token && expiresIn && expiresIn > new Date().getTime())
}

/** 判断访问令牌是否即将过期，供请求层提前刷新。 */
function shouldRefreshToken(): boolean {
  const expiresIn = getTokenExpiresIn()
  const remain = expiresIn - new Date().getTime()
  return Boolean(getToken() && expiresIn && remain <= TOKEN_REFRESH_THRESHOLD)
}

export {
  getToken,
  setToken,
  clearToken,
  getRefreshToken,
  setRefreshToken,
  setTokenExpiresIn,
  getTokenExpiresIn,
  hasValidToken,
  shouldRefreshToken,
}
