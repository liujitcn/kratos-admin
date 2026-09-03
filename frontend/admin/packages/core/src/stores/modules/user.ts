import { defineStore } from "pinia";
import { defAuthService } from "@/api/system/admin/v1/auth";
import { defLoginService } from "@/api/base/v1/login";
import { defMfaService } from "@/api/base/v1/mfa";
import type { LoginRequest, LoginResponse } from "@/rpc/base/v1/login";
import type { VerifyMfaRequest } from "@/rpc/base/v1/mfa";
import type { UserInfoForm } from "@/rpc/system/admin/v1/auth";
import { UserState } from "@/stores/interface";
import piniaPersistConfig from "@/stores/helper/persist";
import { useDictStoreHook } from "@/stores/modules/dict";
import { useLockScreenStore } from "@/stores/modules/lockScreen";

const defaultUserInfo: UserInfoForm = {
  user_name: "",
  nick_name: "",
  phone: "",
  email: "",
  id_type: 0,
  id_code: "",
  avatar: "",
  role_code: "",
  role_name: "",
  dept_name: "",
  tenant_code: "",
  tenant_name: ""
};

export const useUserStore = defineStore("admin-user", {
  state: (): UserState => ({
    token: "",
    tokenType: "",
    tokenExpiresAt: 0,
    isLoggingOut: false,
    authVersion: 0,
    authInvalidated: false,
    userInfo: defaultUserInfo
  }),
  getters: {},
  actions: {
    /** 设置访问令牌 */
    setToken(token: string) {
      this.token = token;
    },
    /** 设置令牌类型 */
    setTokenType(tokenType: string) {
      this.tokenType = tokenType;
    },
    /** 设置令牌过期时间戳 */
    setTokenExpiresAt(tokenExpiresAt: number) {
      this.tokenExpiresAt = tokenExpiresAt;
    },
    /** 设置用户信息 */
    setUserInfo(userInfo: UserState["userInfo"]) {
      this.userInfo = userInfo;
    },
    /** 根据接口返回统一更新令牌信息 */
    updateTokenAuth(accessToken: string, tokenType: string, expiresIn?: number) {
      const tokenPrefix = tokenType ? `${tokenType} ` : "";
      const expiresAt = expiresIn ? Date.now() + expiresIn * 1000 : 0;

      this.authInvalidated = false;
      this.setToken(`${tokenPrefix}${accessToken}`.trim());
      this.setTokenType(tokenType ?? "");
      this.setTokenExpiresAt(expiresAt);
    },
    /** 判断当前访问令牌是否仍然可用 */
    isAuthenticated() {
      return Boolean(this.token.trim() && this.tokenExpiresAt > Date.now());
    },
    /** 登录 */
    async login(loginRequest: LoginRequest): Promise<LoginResponse> {
      const data = await defLoginService.Login(loginRequest);
      if (data.status === 1 || data.status === 0 || data.status === 4) {
        this.updateTokenAuth(data.access_token, data.token_type ?? "", data.expires_in);
      }
      return data;
    },
    /** 校验登录阶段的多因素认证并保存正式令牌。 */
    async verifyMfa(request: VerifyMfaRequest): Promise<LoginResponse> {
      const data = await defMfaService.VerifyMfa(request);
      this.updateTokenAuth(data.access_token, data.token_type ?? "", data.expires_in);
      return data;
    },
    /** 通过 HttpOnly Cookie 刷新认证令牌。 */
    async refreshAccessToken() {
      const data = await defLoginService.RefreshToken({});
      this.updateTokenAuth(data.access_token, data.token_type ?? "", data.expires_in);
      return data;
    },
    /** 获取用户信息 */
    async getUserInfo() {
      const data = await defAuthService.GetUserInfo({});
      this.setUserInfo(data);
      return data;
    },
    /** 清理认证数据 */
    clearAuthData() {
      // 清理登录态时同步清空字典缓存，避免切换账号后读到旧字典。
      useDictStoreHook().clearDictionaryCache();
      useLockScreenStore().clearLock();
      this.authVersion += 1;
      this.authInvalidated = true;
      this.setToken("");
      this.setTokenType("");
      this.setTokenExpiresAt(0);
      this.setUserInfo({ ...defaultUserInfo });
    },
    /** 退出登录 */
    async logout() {
      this.isLoggingOut = true;
      try {
        await defLoginService.Logout({});
      } catch {
        // 退出接口返回异常时，前端仍需清理本地登录态，避免用户卡在当前会话。
      } finally {
        this.clearAuthData();
        this.isLoggingOut = false;
      }
    }
  },
  persist: piniaPersistConfig("admin-user-security-v2", ["userInfo"])
});
