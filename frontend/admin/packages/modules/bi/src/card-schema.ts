import type { BiDashboard } from "./rpc/bi/admin/v1/bi_dashboard";
import type { BiChartEncoding, BiChartStyle, BiChartSchema, BiRow } from "./schema/types";
import type { BiCard } from "./rpc/bi/admin/v1/bi_card";
import type { BiQueryResult, BiSourceDescriptor } from "./rpc/bi/admin/v1/bi_query";

/** 将卡片查询结果映射到现有图表引擎，保留分组及全部所选指标。 */
export function buildCardChart(
  card: BiCard,
  result: BiQueryResult,
  compact = false
): { chart: BiChartSchema; rows: BiRow[]; metrics: CardMetric[]; metricOnly: boolean } {
  const config = parseCardChart(card);
  const metricGrid = card.chart_type === "metric_grid";
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
    (["bar", "line", "area"].includes(card.chart_type) ? card.query?.grouping_options?.find(item => item.time_grain)?.field : undefined) ||
    (card.query ? card.query.dimensions[0] : result.columns.find(column => column.code !== measure)?.code);
  const explicitSeries = config.encoding.series;
  const series = card.chart_type === "scatter" ? undefined :
    explicitSeries && explicitSeries !== dimension ? explicitSeries : card.query?.dimensions.find(code => code !== dimension);
  const rows = result.rows.map(row =>
    Object.fromEntries(result.columns.map((column, index) => [column.code, row.values[index] ?? ""]))
  );
  const gridItems: MetricGridItem[] = metricGrid
    ? (config.metricGrid?.items?.length
        ? config.metricGrid.items
        : (card.query?.measures ?? []).slice(0, 4).map(item => ({ value: item.code })))
    : [];
  const measures = metricGrid
    ? gridItems.map(item => item.value)
    : config.encoding.measures?.length
      ? config.encoding.measures
      : card.query?.measures.length
        ? card.query.measures.map(item => item.code)
        : [measure];
  const metricItems: MetricGridItem[] = metricGrid ? gridItems : measures.map(value => ({ value }));
  return {
    chart: {
      id: `card_${card.id}`,
      datasetId: `card_${card.id}`,
      type: card.chart_type,
      ...(compact && {
        compact,
        measureLabels: Object.fromEntries(numericFields.map(({ code, name, unit }) => [code, unit ? `${name || code}（${unit}）` : name || code]))
      }),
      encoding: {
        dimension,
        measure,
        measures: ["bar", "line", "area"].includes(card.chart_type) && measures.length > 1 ? measures : undefined,
        ...config.encoding,
        series
      },
      aggregation: "none",
      style: {
        showLabel: false,
        showLegend: Boolean(series) || measures.length > 1 || card.chart_type === "pie" || card.chart_type === "donut",
        ...config.style,
        ...(compact && { showLabel: false, dataZoom: false })
      }
    },
    rows,
    metrics: metricItems.map(item => {
      const code = item.value;
      const column = result.columns.find(resultColumn => resultColumn.code === code);
      const value = String(rows[0]?.[code] ?? "") || "—";
      const compareValue = item.compare ? String(rows[0]?.[item.compare] ?? "") : undefined;
      const change = metricGrid ? metricChange(value, compareValue) : undefined;
      return {
        code,
        name: column?.name || code,
        value,
        unit: value === "—" ? "" : column?.unit || "",
        ...(metricGrid
          ? {
              ...(compareValue !== undefined && compareValue !== "" ? { compareValue } : {}),
              ...(change
                ? { compareLabel: item.compareLabel || "较昨日", delta: change.delta, trend: change.trend }
                : {})
            }
          : {})
      };
    }),
    metricOnly: card.chart_type === "metric" || metricGrid
  };
}

/** 数据卡片统一保存字段映射与受控样式。 */
export interface CardChartConfig {
  encoding: BiChartEncoding;
  style: Pick<BiChartStyle, "showLabel" | "showLegend" | "stack" | "dataZoom" | "palette">;
  /** 指标网格的当前列、比较列和比较标签。 */
  metricGrid?: { items: MetricGridItem[] };
}

/** 指标网格中的一组当前指标与可选比较指标。 */
export interface MetricGridItem {
  /** 当前值所在的数值列。 */
  value: string;
  /** 可选的同次查询比较值列。 */
  compare?: string;
  /** 趋势行显示的比较文案。 */
  compareLabel?: string;
}

/** 卡片渲染使用的指标值和可选趋势信息。 */
interface CardMetric {
  code: string;
  name: string;
  value: string;
  unit?: string;
  compareValue?: string;
  compareLabel?: string;
  delta?: string;
  trend?: "up" | "down" | "flat";
}

/** 计算比较百分比，无法比较时返回空结果。 */
interface MetricDecimal {
  coefficient: bigint;
  scale: number;
}

const metricDecimalPattern = /^([+-]?)(?:(\d+)(?:\.(\d*))?|\.(\d+))(?:[eE]([+-]?\d+))?$/;
const metricDecimalMaxScale = 10_000;

/** 解析结果中的十进制字符串，保留超过 JavaScript 安全整数范围的精度。 */
function parseMetricDecimal(value: string): MetricDecimal | undefined {
  const match = metricDecimalPattern.exec(value);
  if (!match) return undefined;
  const integer = match[2] ?? "";
  const fraction = match[3] ?? match[4] ?? "";
  const exponent = match[5] ? Number(match[5]) : 0;
  if (!Number.isSafeInteger(exponent)) return undefined;
  let scale = fraction.length - exponent;
  if (scale > metricDecimalMaxScale || scale < -metricDecimalMaxScale) return undefined;
  let coefficient = BigInt(`${integer || "0"}${fraction}`);
  if (match[1] === "-") coefficient = -coefficient;
  if (coefficient === 0n) return { coefficient: 0n, scale: 0 };
  if (scale < 0) {
    coefficient *= 10n ** BigInt(-scale);
    scale = 0;
  }
  while (scale > 0 && coefficient % 10n === 0n) {
    coefficient /= 10n;
    scale--;
  }
  return { coefficient, scale };
}

/** 计算十进制定点数的百分比变化，输出两位小数并保留精确方向。 */
function metricChange(value: string, compareValue?: string): { delta: string; trend: "up" | "down" | "flat" } | undefined {
  if (!compareValue) return undefined;
  const current = parseMetricDecimal(value);
  const compare = parseMetricDecimal(compareValue);
  if (!current || !compare || compare.coefficient === 0n) return undefined;
  const scale = Math.max(current.scale, compare.scale);
  const currentAligned = current.coefficient * 10n ** BigInt(scale - current.scale);
  const compareAligned = compare.coefficient * 10n ** BigInt(scale - compare.scale);
  const difference = currentAligned - compareAligned;
  const trend = difference > 0n ? "up" : difference < 0n ? "down" : "flat";
  const numerator = difference < 0n ? -difference : difference;
  const denominator = compareAligned < 0n ? -compareAligned : compareAligned;
  let scaled = (numerator * 10_000n) / denominator;
  if ((numerator * 10_000n) % denominator * 2n >= denominator) scaled++;
  const whole = scaled / 100n;
  const fraction = (scaled % 100n).toString().padStart(2, "0");
  return { delta: `${difference < 0n ? "-" : ""}${whole}.${fraction}`, trend };
}

/** 从服务端配置读取图表字段，预览卡片允许使用默认映射。 */
export function parseCardChart(card: Pick<BiCard, "chart_json">): CardChartConfig {
  if (!card.chart_json) return { encoding: {}, style: { showLegend: true, showLabel: false } };
  const value = JSON.parse(card.chart_json) as Partial<CardChartConfig>;
  return {
    encoding: value.encoding ?? {},
    style: { showLegend: true, showLabel: false, ...value.style },
    metricGrid: value.metricGrid
  };
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
