import type { BiQuerySpec, BiResultColumn, BiSourceDescriptor, BiTimeRangeSpec } from "./rpc/bi/admin/v1/bi_query";

/** 查询文案复用管理端当前语言。 */
type Translate = (key: string, params?: Record<string, string>) => string;

/** 日期输入只处理日历日期，避免浏览器时区改变首尾日期。 */
export function shiftCalendarDate(value: string, days: number): string {
  const date = new Date(`${value}T00:00:00Z`);
  date.setUTCDate(date.getUTCDate() + days);
  return date.toISOString().slice(0, 10);
}

/** 将查询边界显示为用户选择的期间，精确到时刻的上限明确标出不包含。 */
export function describeBiTimeRange(range: BiTimeRangeSpec, t: Translate): string {
  if (range.end === "now") {
    if (range.start === "start_of_year") return t("bi.refine.year_to_date");
    const days = /^now-(7|30)d$/.exec(range.start)?.[1];
    if (days) return t(`bi.refine.last_${days}`);
  }
  if (!range.end_exclusive) return `${range.start} — ${range.end}`;
  if (/^\d{4}-\d{2}-\d{2}$/.test(range.start) && /^\d{4}-\d{2}-\d{2}$/.test(range.end)) {
    return `${range.start} — ${shiftCalendarDate(range.end, -1)}`;
  }
  return t("bi.refine.range_before", { start: range.start, end: range.end });
}

/** 以查询记录中实际保存的指标与分组生成摘要，供历史回读与建卡使用。 */
export function describeBiQuery(query: BiQuerySpec, t: Translate, source?: BiSourceDescriptor, columns: BiResultColumn[] = []) {
  const columnName = (code: string) => columns.find(column => column.code === code)?.name;
  const fieldName = (code: string) => source?.dimensions.find(item => item.code === code)?.name || columnName(code) || code;
  const dimensions = query.dimensions.map(code => {
    const group = query.grouping_options?.find(option => option.field === code);
    if (group?.hierarchy_level) return t(`bi.refine.level.${group.hierarchy_level}`);
    const name = fieldName(code);
    return group?.time_grain ? `${name}（${t(`bi.refine.grain.${group.time_grain}`)}）` : name;
  }).join(" × ") || t("bi.card.ungrouped");
  const measures = query.measures.map(({ code }) => source?.measures.find(item => item.code === code)?.name || columnName(code) || code).join("、");
  const range = query.time_range;
  const period = range ? `${fieldName(range.field)} · ${describeBiTimeRange(range, t)}` : t("bi.refine.all_dates");
  return { dimensions, measures, period, title: measures ? `${measures} · ${dimensions}` : t("bi.query.result") };
}
