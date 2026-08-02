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
  /** 是否达到允许提交的最高强度。 */
  isValid: boolean;
}

/**
 * 计算密码强度结果，供表单展示和校验统一复用。
 *
 * @param password 当前密码
 * @returns 密码强度分析结果
 */
export function getPasswordStrength(password?: string): PasswordStrengthResult {
  if (!password) {
    return {
      ruleScore: 0,
      strengthScore: 0,
      level: "empty",
      isValid: false
    };
  }

  let ruleScore = 0;
  if (password.length >= 8) ruleScore += 1;
  if (/[a-z]/.test(password) && /[A-Z]/.test(password)) ruleScore += 1;
  if (/\d/.test(password)) ruleScore += 1;
  if (/[^A-Za-z0-9]/.test(password)) ruleScore += 1;

  if (ruleScore >= 4) {
    return {
      ruleScore,
      strengthScore: 3,
      level: "high",
      isValid: true
    };
  }
  if (ruleScore === 3) {
    return {
      ruleScore,
      strengthScore: 2,
      level: "medium",
      isValid: false
    };
  }
  return {
    ruleScore,
    strengthScore: 1,
    level: "low",
    isValid: false
  };
}

/**
 * 校验密码是否达到最高强度，便于表单规则直接复用。
 *
 * @param password 当前密码
 * @returns 校验结果
 */
export function validatePasswordStrengthValue(password?: string) {
  const result = getPasswordStrength(password);
  return { valid: result.isValid };
}
