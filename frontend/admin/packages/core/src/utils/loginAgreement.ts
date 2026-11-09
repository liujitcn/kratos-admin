/** 一次登录流程的协议确认，OAuth 跳转和 MFA 继续使用同一次确认。 */
interface PendingAgreement {
  version: string;
  acceptedAt: number;
}

const storageKey = "kratos-admin:pending-agreement";
let pending: PendingAgreement | null = null;

/** 保存用户主动确认的协议摘要，不据此恢复页面勾选状态。 */
export function acceptLoginAgreement(version: string) {
  pending = { version, acceptedAt: Date.now() };
  if (typeof sessionStorage !== "undefined") sessionStorage.setItem(storageKey, JSON.stringify(pending));
}

/** 获取未过期的本次确认，供账号登录、OAuth 回调及 MFA 请求共同携带。 */
export function getLoginAgreementHeaders(): Record<string, string> {
  if (!pending && typeof sessionStorage !== "undefined") {
    try {
      pending = JSON.parse(sessionStorage.getItem(storageKey) || "null") as PendingAgreement | null;
    } catch {
      // 损坏的会话记录不能被视为用户确认。
      sessionStorage.removeItem(storageKey);
    }
  }
  if (!pending || !/^[a-f0-9]{64}$/.test(pending.version) || Date.now() - pending.acceptedAt > 10 * 60 * 1000) return {};
  return { "X-Agreement-Accepted": "true", "X-Agreement-Version": pending.version, "X-Agreement-Site": "2" };
}

/** 登录完成或重置时清除确认，下一次登录仍需主动选择。 */
export function clearLoginAgreement() {
  pending = null;
  if (typeof sessionStorage !== "undefined") sessionStorage.removeItem(storageKey);
}
