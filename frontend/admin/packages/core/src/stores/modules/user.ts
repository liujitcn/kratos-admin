import { defineStore } from "pinia";
import { defAuthService } from "@/api/system/auth";
import { defLoginService } from "@/api/base/login";
import type { LoginRequest } from "@/rpc/base/v1/login";
import type { UserInfoForm } from "@/rpc/system/admin/v1/auth";
import { UserState } from "@/stores/interface";
import piniaPersistConfig from "@/stores/helper/persist";
import { useDictStoreHook } from "@/stores/modules/dict";

const defaultUserInfo: UserInfoForm = {
  user_name: "",
  nick_name: "",
  phone: "",
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
    refreshToken: "",
    tokenType: "",
    tokenExpiresAt: 0,
    userInfo: defaultUserInfo
  }),
  getters: {},
  actions: {
    /** 设置访问令牌 */
    setToken(token: string) {
      this.token = token;
    },
    /** 设置刷新令牌 */
    setRefreshToken(refreshToken: string) {
      this.refreshToken = refreshToken;
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
    updateTokenAuth(accessToken: string, refreshToken: string, tokenType: string, expiresIn?: number) {
      const tokenPrefix = tokenType ? `${tokenType} ` : "";
      const expiresAt = expiresIn ? Date.now() + expiresIn * 1000 : 0;

      this.setToken(`${tokenPrefix}${accessToken}`.trim());
      this.setRefreshToken(refreshToken ?? "");
      this.setTokenType(tokenType ?? "");
      this.setTokenExpiresAt(expiresAt);
    },
    /** 判断当前访问令牌是否仍然可用 */
    isAuthenticated() {
      return Boolean(this.token.trim() && this.tokenExpiresAt > Date.now());
    },
    /** 登录 */
    async login(loginRequest: LoginRequest) {
      const data = await defLoginService.Login(loginRequest);
      this.updateTokenAuth(data.access_token, data.refresh_token ?? "", data.token_type ?? "", data.expires_in);
    },
    /** 刷新认证令牌 */
    async refreshAccessToken() {
      if (!this.refreshToken) {
        return Promise.reject(new Error("refresh token 不存在"));
      }

      const data = await defLoginService.RefreshToken({ refresh_token: this.refreshToken });
      this.updateTokenAuth(data.access_token, data.refresh_token ?? this.refreshToken, data.token_type ?? "", data.expires_in);
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
      this.setToken("");
      this.setRefreshToken("");
      this.setTokenType("");
      this.setTokenExpiresAt(0);
      this.setUserInfo({ ...defaultUserInfo });
    },
    /** 退出登录 */
    async logout() {
      try {
        await defLoginService.Logout({});
      } catch {
        // 退出接口返回异常时，前端仍需清理本地登录态，避免用户卡在当前会话。
      } finally {
        this.clearAuthData();
      }
    }
  },
  persist: piniaPersistConfig("admin-user")
});
