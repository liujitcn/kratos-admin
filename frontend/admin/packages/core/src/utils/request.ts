import axios, { type InternalAxiosRequestConfig, type AxiosResponse, type AxiosError, type AxiosRequestConfig } from "axios";
import qs from "qs";
import { reactive } from "vue";
import router from "@/routers";
import { LOGIN_URL } from "@/config";
import pinia from "@/stores";
import { useUserStore } from "@/stores/modules/user";
import { getLocaleRequestHeaders, t } from "@/locales";

const apiBasePath = import.meta.env.VITE_APP_BASE_API || "";
const apiTargetUrl = import.meta.env.VITE_API_URL || import.meta.env.VITE_APP_API_URL || "";
export const requestBaseURL = `${apiTargetUrl}${apiBasePath}`;
const SESSION_URL = "/v1/base/session";
const TOKEN_URL = "/v1/base/token";
const REFRESH_EXPIRY_COOKIE = "kratos_refresh_exp";
const REFRESH_TOKEN_TRANSPORT_HEADER = "X-Refresh-Token-Transport";
const REFRESH_TOKEN_TRANSPORT_COOKIE = "cookie";
const PASSWORD_CHANGE_REQUIRED_METADATA_KEY = "password_change_required";
const PASSWORD_CHANGE_STATE_STORAGE_KEY = "admin-password-change-required";
const CAPTCHA_URL = "/v1/base/captcha";
const CONFIG_URL = "/v1/base/config";
const LANGUAGE_URL = "/v1/base/language";
const PASSWORD_PUBLIC_KEY_URL = "/v1/base/password-public-key";
const OAUTH_PROVIDER_URL = "/v1/base/oauth/provider";
const OAUTH_AUTHORIZATION_URL = "/v1/base/oauth/authorization";
const OAUTH_TICKET_URL = "/v1/base/oauth/ticket";
const MFA_VERIFY_URL = "/v1/base/mfa/verify";
const MFA_ENROLLMENT_URL = "/v1/base/mfa/enrollment";
const MFA_ENROLLMENT_CONFIRM_URL = "/v1/base/mfa/enrollment/confirm";
// 认证公共接口不携带访问令牌。
const NO_AUTH_URL_SET = new Set([
  SESSION_URL,
  TOKEN_URL,
  CAPTCHA_URL,
  CONFIG_URL,
  LANGUAGE_URL,
  PASSWORD_PUBLIC_KEY_URL,
  OAUTH_PROVIDER_URL,
  OAUTH_AUTHORIZATION_URL,
  OAUTH_TICKET_URL,
  MFA_VERIFY_URL,
  MFA_ENROLLMENT_URL,
  MFA_ENROLLMENT_CONFIRM_URL
]);
const AUTH_EXPIRED_EXCLUDED_URL_SET = new Set([
  SESSION_URL,
  TOKEN_URL,
  CAPTCHA_URL,
  CONFIG_URL,
  LANGUAGE_URL,
  PASSWORD_PUBLIC_KEY_URL,
  OAUTH_PROVIDER_URL,
  OAUTH_AUTHORIZATION_URL,
  OAUTH_TICKET_URL,
  MFA_VERIFY_URL,
  MFA_ENROLLMENT_URL,
  MFA_ENROLLMENT_CONFIRM_URL
]);

/** 支持自动重试的 Axios 请求配置。 */
type RetryableRequestConfig = InternalAxiosRequestConfig & {
  /** 标记当前请求已经因认证失败重试过，避免刷新失败时递归重放。 */
  _authRetried?: boolean;
};

/** 统一的结构化错误响应。 */
type ErrorResponseData = {
  code?: string | number;
  message?: string;
  reason?: string;
  metadata?: Record<string, string>;
};

/** 本地持久化的强制改密状态，仅包含非敏感的显示信息。 */
interface PersistedPasswordChangeState {
  /** 是否需要强制修改密码。 */
  required: boolean;
  /** 后端返回的提示文案。 */
  message: string;
}

/** 读取刷新前保存的强制改密状态。 */
function getPersistedPasswordChangeState(): PersistedPasswordChangeState {
  const emptyState = { required: false, message: "" };
  if (typeof window === "undefined") return emptyState;

  try {
    const value = window.localStorage.getItem(PASSWORD_CHANGE_STATE_STORAGE_KEY);
    if (!value) return emptyState;
    const parsed = JSON.parse(value) as unknown;
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) return emptyState;
    const persisted = parsed as { message?: unknown };
    return {
      required: true,
      message: typeof persisted.message === "string" ? persisted.message : ""
    };
  } catch {
    return emptyState;
  }
}

/** 强制改密弹窗状态。 */
export const passwordChangeState = reactive(getPersistedPasswordChangeState());

// 创建 axios 实例
const service = axios.create({
  baseURL: requestBaseURL,
  timeout: 50000,
  headers: { "Content-Type": "application/json;charset=utf-8" },
  paramsSerializer: params => qs.stringify(params),
  withCredentials: true
});

// 刷新令牌请求使用独立实例，避免与主请求拦截器互相递归。
const refreshService = axios.create({
  baseURL: requestBaseURL,
  timeout: 50000,
  headers: { "Content-Type": "application/json;charset=utf-8" },
  withCredentials: true
});

/** 获取用户状态仓库 */
function getUserStore() {
  return useUserStore(pinia);
}

/** 判断当前请求是否需要跳过认证头 */
function shouldSkipAuth(config: InternalAxiosRequestConfig) {
  if (config.headers.Authorization === "no-auth") return true;

  const requestUrl = config.url ?? "";
  const requestMethod = String(config.method ?? "").toLowerCase();
  // 登录请求会显式声明 no-auth，登出请求则必须带访问令牌让服务端清理会话和刷新令牌。
  if (requestUrl === SESSION_URL && requestMethod === "delete") return false;
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
  return requestUrl === SESSION_URL && requestMethod === "delete";
}

/** 判断响应是否要求用户先修改密码。 */
function isPasswordChangeRequired(data?: ErrorResponseData) {
  return data?.metadata?.[PASSWORD_CHANGE_REQUIRED_METADATA_KEY] === "true";
}

/** 判断最近是否已经处理过强制改密响应。 */
function isPasswordChangeRequiredPromptActive() {
  return passwordChangePromptUntil > Date.now();
}

/** 处理强制改密响应，避免并发请求重复提示并打开全局改密弹窗。 */
export function handlePasswordChangeRequired(message: string) {
  if (isPasswordChangeRequiredPromptActive()) return;
  passwordChangePromptUntil = Date.now() + 10 * 1000;
  passwordChangeState.required = true;
  passwordChangeState.message = message;
  persistPasswordChangeState();
}

/** 清理强制改密弹窗状态。 */
export function clearPasswordChangeRequired() {
  passwordChangeState.required = false;
  passwordChangeState.message = "";
  passwordChangePromptUntil = 0;
  persistPasswordChangeState();
}

/** 持久化或删除强制改密状态，兼容浏览器禁用本地存储的场景。 */
function persistPasswordChangeState() {
  if (typeof window === "undefined") return;

  try {
    if (passwordChangeState.required) {
      window.localStorage.setItem(
        PASSWORD_CHANGE_STATE_STORAGE_KEY,
        JSON.stringify({ message: passwordChangeState.message })
      );
    } else {
      window.localStorage.removeItem(PASSWORD_CHANGE_STATE_STORAGE_KEY);
    }
  } catch {
    // 本地存储不可用时仍保留当前页面内的响应式状态。
  }
}

/** 统一展示请求错误，合并并发请求产生的相同提示。 */
function showRequestError(message: string) {
  ElMessage.error({ message, grouping: true });
}

/** 读取访问令牌过期时间 */
function getTokenExpiresAt() {
  return getUserStore().tokenExpiresAt;
}

/** 判断浏览器是否存在非敏感的刷新令牌过期提示。 */
function hasRefreshCookieHint() {
  if (typeof document === "undefined") return false;
  const item = document.cookie.split(";").find(value => value.trim().startsWith(`${REFRESH_EXPIRY_COOKIE}=`));
  if (!item) return false;
  const expiresAt = Number(item.split("=")[1]);
  return Number.isFinite(expiresAt) && expiresAt * 1000 > Date.now();
}

/** 判断当前访问令牌是否仍在有效期内。 */
export function hasValidAccessToken() {
  const userStore = getUserStore();
  return Boolean(userStore.token.trim() && userStore.tokenExpiresAt > Date.now());
}

/** 读取最新可用访问令牌，必要时先串行刷新，供 axios、fetch 与 SSE 共用。 */
export async function getRequestAccessToken(): Promise<string> {
  const userStore = getUserStore();
  if (userStore.isLoggingOut) {
    return userStore.token.trim();
  }
  if (userStore.authInvalidated) {
    return "";
  }
  if (!userStore.token && hasRefreshCookieHint()) {
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
  const userStore = getUserStore();
  if (userStore.isLoggingOut || userStore.authInvalidated) {
    return false;
  }
  if (hasValidAccessToken()) {
    return true;
  }

  if (!hasRefreshCookieHint()) {
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
let passwordChangePromptUntil = 0;

/** 统一处理认证失效 */
export function handleAuthExpired() {
  if (isHandlingAuthExpired) {
    return;
  }

  isHandlingAuthExpired = true;
  ElMessageBox.confirm(t("core.auth.session_expired"), t("common.title.notice"), {
    confirmButtonText: t("core.auth.login_again"),
    cancelButtonText: t("common.action.cancel"),
    type: "warning",
    closeOnClickModal: false,
    closeOnPressEscape: false
  })
    .then(() => {
      const userStore = getUserStore();
      const currentRoute = router.currentRoute.value;
      const redirect = currentRoute.path === LOGIN_URL ? undefined : currentRoute.fullPath;
      clearPasswordChangeRequired();
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
    Object.assign(config.headers, getLocaleRequestHeaders());
    config.headers[REFRESH_TOKEN_TRANSPORT_HEADER] = REFRESH_TOKEN_TRANSPORT_COOKIE;
    // 登录、验证码、刷新令牌接口不携带访问令牌，避免请求头污染。
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

    const responseData = response.data as ErrorResponseData;
    const { code, message, reason, metadata } = responseData;
    if (code === undefined || message === undefined || reason === undefined || metadata === undefined) {
      return response.data;
    }

    if (isPasswordChangeRequired(responseData)) {
      handlePasswordChangeRequired(message || t("common.message.system_error"));
    } else {
      showRequestError(message || t("common.message.system_error"));
    }
    return Promise.reject(new Error(message || t("common.message.system_error")));
  },
  async (error: AxiosError) => {
    const status = error.response?.status;
    const data = error.response?.data as ErrorResponseData | undefined;
    const code = data?.code;
    const message = data?.message;
    const requestConfig = error.config as RetryableRequestConfig | undefined;

    // 业务请求仅在 401 时尝试刷新并重放，403 直接展示后端权限错误。
    if ((status === 401 || code === 401) && !shouldSkipAuthExpiredPrompt(requestConfig)) {
      const userStore = getUserStore();
      if (requestConfig && !requestConfig._authRetried && !userStore.isLoggingOut && !userStore.authInvalidated && hasRefreshCookieHint()) {
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
        if (isPasswordChangeRequired(data)) {
          handlePasswordChangeRequired(message || t("common.message.system_error"));
        } else if (message) {
          showRequestError(message);
        } else {
          showRequestError(t("common.message.system_error"));
        }
      }
    } else if (!shouldSkipErrorMessage(requestConfig)) {
      showRequestError(error.message || t("common.message.system_error"));
    }
    return Promise.reject(error.message);
  }
);

/** 以请求体、响应体顺序提供带类型约束的 HTTP 请求函数。 */
function request<Request = unknown, Response = unknown>(config: AxiosRequestConfig<Request>): Promise<Response> {
  return service<Response, Response, Request>(config);
}

export default request;

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
  const authVersion = userStore.authVersion;

  const response = await refreshService.post(
    TOKEN_URL,
    {},
    {
      headers: {
        Authorization: "no-auth",
        [REFRESH_TOKEN_TRANSPORT_HEADER]: REFRESH_TOKEN_TRANSPORT_COOKIE,
        ...getLocaleRequestHeaders()
      }
    }
  );
  const data = response.data;
  if (userStore.authVersion !== authVersion || userStore.isLoggingOut || userStore.authInvalidated) {
    return;
  }
  userStore.updateTokenAuth(data.access_token, data.token_type ?? "", data.expires_in);
}
