/** 字典组件支持的选项值类型。 */
export type DictOptionValue = string | number;

/**
 * 将不在当前字典选项中的单选值归一化为空值。
 *
 * 字典选项通常异步加载，表单重置可能在选项加载完成后再次写入未知枚举值；
 * 统一归一化可以避免 Element Plus 将无匹配的数字直接渲染出来。
 */
export function normalizeDictValue(value: DictOptionValue | undefined, options: readonly DictOptionValue[]) {
  if (value === undefined || value === null || value === "") return undefined;
  return options.some(option => option === value) ? value : undefined;
}

/** 过滤不在当前字典选项中的多选值。 */
export function normalizeDictValues(values: readonly DictOptionValue[] | undefined, options: readonly DictOptionValue[]) {
  return (values ?? []).filter(value => options.some(option => option === value));
}
