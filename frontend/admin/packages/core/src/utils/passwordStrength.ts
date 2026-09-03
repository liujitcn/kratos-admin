/** 密码强度等级。 */
export type PasswordStrengthLevel = "empty" | "low" | "medium" | "high";

/** 密码强度分析结果。 */
export interface PasswordStrengthResult {
  /** 命中的复杂度条件数量。 */
  ruleScore: number;
  /** 强度条得分。 */
  strengthScore: number;
  /** 强度等级。 */
  level: PasswordStrengthLevel;
}

/**
 * 计算密码强度结果，供表单展示使用；最终规则由服务端密码策略校验。
 *
 * @param password 当前密码
 * @returns 密码强度分析结果
 */
export function getPasswordStrength(password?: string): PasswordStrengthResult {
  if (!password) {
    return {
      ruleScore: 0,
      strengthScore: 0,
      level: "empty"
    };
  }

  let ruleScore = 0;
  if (/[a-z]/.test(password)) ruleScore += 1;
  if (/[A-Z]/.test(password)) ruleScore += 1;
  if (/\d/.test(password)) ruleScore += 1;
  if (/[^A-Za-z0-9]/.test(password)) ruleScore += 1;

  if (ruleScore >= 4) {
    return {
      ruleScore,
      strengthScore: 3,
      level: "high"
    };
  }
  if (ruleScore === 3) {
    return {
      ruleScore,
      strengthScore: 2,
      level: "medium"
    };
  }
  return {
    ruleScore,
    strengthScore: 1,
    level: "low"
  };
}
