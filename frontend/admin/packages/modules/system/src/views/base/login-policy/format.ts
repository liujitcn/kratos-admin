/** formatLoginPolicyList 将登录策略列表格式化为表单文本。 */
export function formatLoginPolicyList(value: string[] | undefined, separator: string): string {
  return (value ?? []).join(separator);
}
