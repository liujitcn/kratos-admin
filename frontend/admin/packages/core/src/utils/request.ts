import axios, { type InternalAxiosRequestConfig, type AxiosResponse, type AxiosError } from "axios";
import qs from "qs";
import { ElMessage, ElMessageBox } from "element-plus";
import router from "@/routers";
import { LOGIN_URL } from "@/config";
import pinia from "@/stores";
import { useUserStore } from "@/stores/modules/user";

const apiBasePath = import.meta.env.VITE_APP_BASE_API || "";
const apiTargetUrl = import.meta.env.VITE_API_URL || import.meta.env.VITE_APP_API_URL || "";
export const requestBaseURL = `${apiTargetUrl}${apiBasePath}`;
const SESSION_URL = "/v1/base/session";
const TOKEN_URL = "/v1/base/token";
const CAPTCHA_URL = "/v1/base/captcha";
const CONFIG_URL = "/v1/base/config";
const PASSWORD_PUBLIC_KEY_URL = "/v1/base/password-public-key";
const OAUTH_PROVIDER_URL = "/v1/base/oauth/provider";
const OAUTH_AUTHORIZATION_URL = "/v1/base/oauth/authorization";
const OAUTH_TICKET_URL = "/v1/base/oauth/ticket";
const LEGACY_AUTH_URL = "/auth";
const LEGACY_REFRESH_TOKEN_URL = `${LEGACY_AUTH_URL}/token`;
const LEGACY_CAPTCHA_URL = "/login/captcha";
// 认证公共接口不携带旧 token，同时保留旧路径兼容迁移期间的灰度访问。
const NO_AUTH_URL_SET = new Set([
  SESSION_URL,
  TOKEN_URL,
  CAPTCHA_URL,
  CONFIG_URL,
  PASSWORD_PUBLIC_KEY_URL,
  OAUTH_PROVIDER_URL,
  OAUTH_AUTHORIZATION_URL,
  OAUTH_TICKET_URL,
  LEGACY_AUTH_URL,
  LEGACY_CAPTCHA_URL,
  LEGACY_REFRESH_TOKEN_URL
]);
const AUTH_EXPIRED_EXCLUDED_URL_SET = new Set([
  SESSION_URL,
  TOKEN_URL,
  CAPTCHA_URL,
  CONFIG_URL,
  PASSWORD_PUBLIC_KEY_URL,
  OAUTH_PROVIDER_URL,
  OAUTH_AUTHORIZATION_URL,
  OAUTH_TICKET_URL,
  LEGACY_AUTH_URL,
  LEGACY_REFRESH_TOKEN_URL,
  LEGACY_CAPTCHA_URL
]);

/** 支持自动重试的 Axios 请求配置。 */
type RetryableRequestConfig = InternalAxiosRequestConfig & {
  /** 标记当前请求已经因认证失败重试过，避免刷新失败时递归重放。 */
  _authRetried?: boolean;
};

// 创建 axios 实例
const service = axios.create({
  baseURL: requestBaseURL,
  timeout: 50000,
  headers: { "Content-Type": "application/json;charset=utf-8" },
  paramsSerializer: params => qs.stringify(params)
});

// 刷新令牌请求使用独立实例，避免与主请求拦截器互相递归。
const refreshService = axios.create({
  baseURL: requestBaseURL,
  timeout: 50000,
  headers: { "Content-Type": "application/json;charset=utf-8" }
});

/** 获取用户状态仓库 */
function getUserStore() {
  return useUserStore(pinia);
}

/** 判断当前请求是否需要跳过认证头 */
function shouldSkipAuth(config: InternalAxiosRequestConfig) {
  if (config.headers.Authorization === "no-auth") return true;

  const requestUrl = config.url ?? "";
  return NO_AUTH_URL_SET.has(requestUrl);
}

/** 判断当前请求是否不应触发登录失效弹窗 */
function shouldSkipAuthExpiredPrompt(config?: InternalAxiosRequestConfig) {
  if (!config) return false;

  const requestUrl = config.url ?? "";
  return AUTH_EXPIRED_EXCLUDED_URL_SET.has(requestUrl);
}

/** 判断当前请求是否需要静默掉错误提示。 */
function shouldSkipErrorMessage(config?: InternalAxiosRequestConfig) {
  if (!config) return false;

  const requestUrl = config.url ?? "";
  const requestMethod = String(config.method ?? "").toLowerCase();
  return (requestUrl === SESSION_URL || requestUrl === LEGACY_AUTH_URL) && requestMethod === "delete";
}

/** 读取访问令牌过期时间 */
function getTokenExpiresAt() {
  return getUserStore().tokenExpiresAt;
}

/** 判断当前访问令牌是否仍在有效期内。 */
export function hasValidAccessToken() {
  const userStore = getUserStore();
  return Boolean(userStore.token.trim() && userStore.tokenExpiresAt > Date.now());
}

/** 读取最新可用访问令牌，必要时先串行刷新，供 axios、fetch 与 SSE 共用。 */
export async function getRequestAccessToken(): Promise<string> {
  const userStore = getUserStore();
  if (!userStore.token && userStore.refreshToken) {
    await handleTokenRefresh(false);
  }

  const expiresAt = getTokenExpiresAt();
  const remainingTime = expiresAt - Date.now();
  if (expiresAt && remainingTime <= 5 * 60 * 1000) {
    await handleTokenRefresh(false);
  }

  return getUserStore().token.trim();
}

/** 尝试通过刷新令牌恢复访问令牌，供路由守卫进入页面前调用。 */
export async function ensureAccessToken() {
  if (hasValidAccessToken()) {
    return true;
  }

  const userStore = getUserStore();
  if (!userStore.refreshToken) {
    return false;
  }

  try {
    await handleTokenRefresh(false);
    return hasValidAccessToken();
  } catch {
    userStore.clearAuthData();
    return false;
  }
}

// 防止并发 401 重复弹出认证失效确认框。
let isHandlingAuthExpired = false;

/** 统一处理认证失效 */
export function handleAuthExpired() {
  if (isHandlingAuthExpired) {
    return;
  }

  isHandlingAuthExpired = true;
  ElMessageBox.confirm("登录状态已失效，请重新登录", "提示", {
    confirmButtonText: "重新登录",
    cancelButtonText: "取消",
    type: "warning",
    closeOnClickModal: false,
    closeOnPressEscape: false
  })
    .then(() => {
      const userStore = getUserStore();
      const currentRoute = router.currentRoute.value;
      const redirect = currentRoute.path === LOGIN_URL ? undefined : currentRoute.fullPath;
      userStore.clearAuthData();
      return router.replace({
        path: LOGIN_URL,
        query: redirect ? { redirect } : undefined
      });
    })
    .finally(() => {
      isHandlingAuthExpired = false;
    });
}

// 请求拦截器
service.interceptors.request.use(
  async (config: InternalAxiosRequestConfig) => {
    const skipAuth = shouldSkipAuth(config);
    const accessToken = skipAuth ? "" : await getRequestAccessToken();
    // 登录、验证码、刷新令牌接口不携带旧 token，避免请求头污染。
    if (!skipAuth && accessToken) {
      config.headers.Authorization = accessToken;
    } else {
      delete config.headers.Authorization;
    }
    return config;
  },
  error => Promise.reject(error)
);

// 响应拦截器
service.interceptors.response.use(
  (response: AxiosResponse) => {
    // 如果响应是二进制流，则直接返回，用于下载文件、Excel 导出等
    if (response.config.responseType === "blob") {
      return response;
    }

    const { code, message, reason, metadata } = response.data;
    if (code === undefined || message === undefined || reason === undefined || metadata === undefined) {
      return response.data;
    }

    ElMessage.error(message || "系统出错");
    return Promise.reject(new Error(message || "Error"));
  },
  async (error: AxiosError) => {
    const status = error.response?.status;
    const data = error.response?.data as { code?: string | number; message?: string } | undefined;
    const code = data?.code;
    const message = data?.message;
    const requestConfig = error.config as RetryableRequestConfig | undefined;

    // 业务请求仅在 401 时尝试刷新并重放，403 直接展示后端权限错误。
    if ((status === 401 || code === 401) && !shouldSkipAuthExpiredPrompt(requestConfig)) {
      if (requestConfig && !requestConfig._authRetried && getUserStore().refreshToken) {
        requestConfig._authRetried = true;
        try {
          await handleTokenRefresh(false);
          requestConfig.headers.Authorization = getUserStore().token.trim();
          return service(requestConfig);
        } catch (refreshError) {
          console.log("token 刷新失败", refreshError);
        }
      }
      handleAuthExpired();
    } else if (data) {
      if (!shouldSkipErrorMessage(requestConfig)) {
        if (message) {
          ElMessage.error(message);
        } else {
          ElMessage.error("系统出错");
        }
      }
    } else if (!shouldSkipErrorMessage(requestConfig)) {
      ElMessage.error(error.message || "系统出错");
    }
    return Promise.reject(error.message);
  }
);

export default service;

// 刷新 Token 的锁
let isRefreshing = false;
let refreshPromise: Promise<void> | null = null;

/** 刷新 Token 处理 */
async function handleTokenRefresh(promptOnFailure = true) {
  if (!isRefreshing) {
    isRefreshing = true;
    refreshPromise = refreshAccessToken()
      .catch(error => {
        if (promptOnFailure) {
          console.log("token 刷新失败", error);
          handleAuthExpired();
        }
        throw error;
      })
      .finally(() => {
        isRefreshing = false;
        refreshPromise = null;
      });
  }

  if (refreshPromise) {
    await refreshPromise;
  }
}

/** 调用刷新令牌接口并回写最新认证信息 */
async function refreshAccessToken() {
  const userStore = getUserStore();
  if (!userStore.refreshToken) {
    return Promise.reject(new Error("refresh token 不存在"));
  }

  const response = await refreshService.post(
    TOKEN_URL,
    { refresh_token: userStore.refreshToken },
    { headers: { Authorization: "no-auth" } }
  );
  const data = response.data;
  userStore.updateTokenAuth(
    data.access_token,
    data.refresh_token ?? userStore.refreshToken,
    data.token_type ?? "",
    data.expires_in
  );
}
