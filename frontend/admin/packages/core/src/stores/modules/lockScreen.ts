import { defineStore } from "pinia";
import type { LockScreenState } from "@/stores/interface";
import piniaPersistConfig from "@/stores/helper/persist";

/** 将锁屏密码转换为不可逆摘要，避免在本地持久化明文密码。 */
async function hashPassword(password: string): Promise<string> {
  const data = new TextEncoder().encode(password);
  const digest = await crypto.subtle.digest("SHA-256", data);
  return Array.from(new Uint8Array(digest), byte => byte.toString(16).padStart(2, "0")).join("");
}

/** 管理端锁屏状态仓库。 */
export const useLockScreenStore = defineStore("admin-lock-screen", {
  state: (): LockScreenState => ({
    isLocked: false,
    passwordHash: "",
    setupVisible: false,
    unlockVisible: false,
    unlockError: false
  }),
  getters: {
    /** 当前是否已配置可用于解锁的密码。 */
    isConfigured: state => Boolean(state.passwordHash)
  },
  actions: {
    /** 打开锁屏密码设置界面。 */
    openSetup() {
      this.setupVisible = true;
      this.unlockVisible = false;
      this.unlockError = false;
    },
    /** 关闭锁屏密码设置界面。 */
    closeSetup() {
      this.setupVisible = false;
    },
    /** 设置锁屏密码并立即进入锁屏状态。 */
    async lock(password: string) {
      this.passwordHash = await hashPassword(password);
      this.isLocked = true;
      this.setupVisible = false;
      this.unlockVisible = false;
      this.unlockError = false;
    },
    /** 打开锁屏解锁输入界面。 */
    openUnlock() {
      this.unlockVisible = true;
      this.unlockError = false;
    },
    /** 返回锁屏首页。 */
    closeUnlock() {
      this.unlockVisible = false;
      this.unlockError = false;
    },
    /** 校验锁屏密码，成功后解除当前锁定。 */
    async unlock(password: string): Promise<boolean> {
      const passwordHash = await hashPassword(password);
      if (!this.passwordHash || passwordHash !== this.passwordHash) {
        this.unlockError = true;
        return false;
      }

      this.isLocked = false;
      this.passwordHash = "";
      this.unlockVisible = false;
      this.unlockError = false;
      return true;
    },
    /** 清理锁屏状态，供退出登录时使用。 */
    clearLock() {
      this.isLocked = false;
      this.passwordHash = "";
      this.setupVisible = false;
      this.unlockVisible = false;
      this.unlockError = false;
    }
  },
  persist: piniaPersistConfig("admin-lock-screen", ["isLocked", "passwordHash"])
});
