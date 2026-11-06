import type { BiDashboard } from "./rpc/bi/admin/v1/bi_dashboard";
import type { BiChartEncoding, BiChartStyle, BiChartSchema, BiRow } from "./schema/types";
import type { BiCard } from "./rpc/bi/admin/v1/bi_card";
import type { BiQueryResult, BiSourceDescriptor } from "./rpc/bi/admin/v1/bi_query";

/** 将卡片查询结果映射到现有图表引擎，保留分组及全部所选指标。 */
export function buildCardChart(
  card: BiCard,
  result: BiQueryResult
): { chart: BiChartSchema; rows: BiRow[]; metrics: Array<{ code: string; name: string; value: string }>; metricOnly: boolean } {
  const config = parseCardChart(card);
  const numericFields = result.columns.filter(column => column.type === "number");
  const measure =
    config.encoding.measure ||
    (card.chart_type === "scatter" ? numericFields[1]?.code : undefined) ||
    card.query?.measures[0]?.code ||
    result.columns.find(column => column.type === "number")?.code ||
    result.columns[0]?.code ||
    "";
  const dimension =
    config.encoding.dimension ||
    (card.chart_type === "scatter" ? numericFields[0]?.code : undefined) ||
    (card.query ? card.query.dimensions[0] : result.columns.find(column => column.code !== measure)?.code);
  const rows = result.rows.map(row =>
    Object.fromEntries(result.columns.map((column, index) => [column.code, row.values[index] ?? ""]))
  );
  const measures = config.encoding.measures?.length
    ? config.encoding.measures
    : card.query?.measures.length
      ? card.query.measures.map(item => item.code)
      : [measure];
  return {
    chart: {
      id: `card_${card.id}`,
      datasetId: `card_${card.id}`,
      type: card.chart_type,
      encoding: {
        dimension,
        measure,
        measures: ["bar", "line", "area"].includes(card.chart_type) && measures.length > 1 ? measures : undefined,
        series: card.chart_type === "scatter" ? undefined : card.query?.dimensions[1],
        ...config.encoding
      },
      aggregation: "none",
      style: {
        showLabel: false,
        showLegend: measures.length > 1 || card.chart_type === "pie" || card.chart_type === "donut",
        ...config.style
      }
    },
    rows,
    metrics: measures.map(code => ({
      code,
      name: result.columns.find(column => column.code === code)?.name || code,
      value: String(rows[0]?.[code] ?? "—")
    })),
    metricOnly: card.chart_type === "metric"
  };
}

/** 数据卡片统一保存字段映射与受控样式。 */
export interface CardChartConfig {
  encoding: BiChartEncoding;
  style: Pick<BiChartStyle, "showLabel" | "showLegend" | "stack" | "dataZoom" | "palette">;
}

/** 从服务端配置读取图表字段，预览卡片允许使用默认映射。 */
export function parseCardChart(card: Pick<BiCard, "chart_json">): CardChartConfig {
  return card.chart_json
    ? (JSON.parse(card.chart_json) as CardChartConfig)
    : { encoding: {}, style: { showLegend: true, showLabel: false } };
}

/** 仅在卡片明确指定时间字段或只有一个可过滤日期分组时开放周期切换。 */
export function supportsCardPeriod(card: BiCard, source?: BiSourceDescriptor): boolean {
  const fields =
    source?.dimensions.filter(
      dimension =>
        dimension.type === "date" &&
        dimension.filterable &&
        (card.query?.time_range
          ? dimension.code === card.query.time_range.field
          : card.query?.dimensions.includes(dimension.code))
    ) ?? [];
  return fields.length === 1;
}

/** 计算刷新等待时间，86400对应浏览器本地时区的下一次02:00。 */
export function cardRefreshDelay(seconds: number, now = new Date()): number {
  if (seconds !== 86400) return seconds * 1000;
  const next = new Date(now.getTime());
  next.setHours(2, 0, 0, 0);
  if (next.getTime() <= now.getTime()) next.setDate(next.getDate() + 1);
  return next.getTime() - now.getTime();
}

/** 按当前四种界面语言的分组习惯显示数字字符串，不转换浮点数或丢弃小数位。 */
export function formatCardMetric(value: string): string {
  if (!/^[+-]?\d+(?:\.\d+)?$/.test(value)) return value;
  const [integer, fraction] = value.split(".");
  return integer.replace(/\B(?=(\d{3})+(?!\d))/g, ",") + (fraction === undefined ? "" : "." + fraction);
}

/** 保持工作台入口顺序，忽略已删除或撤销权限的仪表盘。 */
export function orderWorkbenchDashboards(ids: number[], dashboards: BiDashboard[]): BiDashboard[] {
  const byID = new Map(dashboards.map(item => [item.id, item]));
  return ids.flatMap(id => {
    const item = byID.get(id);
    return item ? [item] : [];
  });
}
