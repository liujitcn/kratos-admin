import type { BiMetricChange, BiSourceMeasure } from "./rpc/bi/admin/v1/bi_source";

/** 只提交当前定义的变化，预期修订始终取原公共定义。 */
export function metricChanges(original: BiSourceMeasure[], edited: BiSourceMeasure[]): BiMetricChange[] {
  return edited.flatMap(item => {
    const before = original.find(metric => metric.code === item.code);
    if (!before || (before.name === item.name && before.expression === item.expression && before.unit === item.unit)) return [];
    return [{ metric_code: item.code, expression: item.expression, name: item.name, unit: item.unit, expected_revision: before.revision }];
  });
}

/** 刷新冲突后的公共定义时保留本地输入，并采用最新修订重新预览。 */
export function mergeMetricDrafts(current: BiSourceMeasure[], drafts: BiSourceMeasure[]): BiSourceMeasure[] {
  return current.map(item => {
    const draft = drafts.find(metric => metric.code === item.code);
    return draft ? { ...item, expression: draft.expression, name: draft.name, unit: draft.unit } : { ...item };
  });
}

/** 解析一次性预览已应用的冲突，只接受服务端返回的有效卡片编号。 */
export function appliedCardID(cause: unknown): number {
  if (!cause || typeof cause !== "object") return 0;
  const error = cause as { response?: { data?: { code?: number; metadata?: Record<string, string> } } };
  const data = error.response?.data;
  const id = Number(data?.metadata?.card_id ?? 0);
  return data?.code === 409 && Number.isSafeInteger(id) && id > 0 ? id : 0;
}
