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
  /** 强度文案。 */
  text: string;
  /** 是否达到允许提交的最高强度。 */
  isValid: boolean;
}

/** 密码强度错误提示。 */
export const PASSWORD_STRENGTH_ERROR_MESSAGE = "密码需同时包含大小写字母、数字、特殊字符，且长度不少于 8 位";

/** 密码强度说明文案。 */
export const PASSWORD_STRENGTH_TIP = "需同时包含大小写字母、数字、特殊字符，且长度不少于 8 位，达到最高强度后才可提交。";

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
      text: "未输入",
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
      text: "高",
      isValid: true
    };
  }
  if (ruleScore === 3) {
    return {
      ruleScore,
      strengthScore: 2,
      level: "medium",
      text: "中",
      isValid: false
    };
  }
  return {
    ruleScore,
    strengthScore: 1,
    level: "low",
    text: "低",
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
  return {
    valid: result.isValid,
    message: result.isValid ? "" : PASSWORD_STRENGTH_ERROR_MESSAGE
  };
}
