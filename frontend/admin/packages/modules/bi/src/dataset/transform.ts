/**
 * 图表级数据变换管线：过滤 → 聚合 → 排序 → TopN。
 * 渲染器在 compile 前调用 transformRows，设计器画布与渲染器共用同一管线。
 * 聚合语义 = 主流 BI：按 dimension 分组，对 measure（或宽表 measures 全部列）施加 agg。
 */

import type {
  BiAggFn,
  BiChartFilter,
  BiChartSchema,
  BiChartSort,
  BiRow,
} from "../schema/types";
import { NON_AGG_TYPES } from "../engine/chart-types";
import { toNumber } from "./index";

/** 单过滤器行判定；值解析：数字串转数，in 接数组 */
function matchFilter(row: BiRow, f: BiChartFilter): boolean {
  const cell = row[f.field];
  switch (f.op) {
    case "eq":
      return String(cell) === String(f.value);
    case "ne":
      return String(cell) !== String(f.value);
    case "gt":
      return toNumber(cell) > toNumber(f.value);
    case "gte":
      return toNumber(cell) >= toNumber(f.value);
    case "lt":
      return toNumber(cell) < toNumber(f.value);
    case "lte":
      return toNumber(cell) <= toNumber(f.value);
    case "contains":
      return String(cell ?? "").includes(String(f.value ?? ""));
    case "in": {
      const list = Array.isArray(f.value) ? f.value : [f.value];
      return list.map(String).includes(String(cell));
    }
    default:
      return true;
  }
}

export function applyFilters(rows: BiRow[], filters: BiChartFilter[] | undefined): BiRow[] {
  if (!filters?.length) return rows;
  return rows.filter((row) => filters.every((f) => matchFilter(row, f)));
}

const AGG_FNS: Record<Exclude<BiAggFn, "none">, (nums: number[]) => number> = {
  sum: (nums) => nums.reduce((a, b) => a + b, 0),
  avg: (nums) => (nums.length ? nums.reduce((a, b) => a + b, 0) / nums.length : 0),
  max: (nums) => (nums.length ? Math.max(...nums) : 0),
  min: (nums) => (nums.length ? Math.min(...nums) : 0),
  // count 不依赖数值：调用方传入占位 1 数组
  count: (nums) => nums.length,
};

/** 是否应执行聚合：给了 dimension + 显式 agg（非 none）；
 *  边列表/矩阵型（sankey/graph/heatmap）天然按行消费，不做维度聚合 */
function shouldAggregate(chart: BiChartSchema): boolean {
  if (NON_AGG_TYPES.has(chart.type)) return false;
  const agg = chart.aggregation ?? "sum";
  const dim = chart.encoding?.dimension;
  if (!dim || agg === "none") return false;
  // 至少要有一个度量对象（count 允许无 measure）
  return !!(chart.encoding?.measure || chart.encoding?.measures?.length || agg === "count");
}

/** 分组键字段：维度 + （pivot 图的列维度也要保留，供 S2 交叉表用） */
function groupFieldsOf(chart: BiChartSchema): string[] {
  const fields = [chart.encoding?.dimension ?? ""];
  if (chart.type === "pivot" && chart.encoding?.series) fields.push(chart.encoding.series);
  return fields.filter(Boolean);
}

/** 分组聚合：输出每（维度组）一行的宽表（保留维度列 + 各度量列） */
export function aggregateRows(
  rows: BiRow[],
  chart: BiChartSchema
): BiRow[] {
  const groupFields = groupFieldsOf(chart);
  const agg: BiAggFn = chart.aggregation ?? "sum";
  const measure = chart.encoding?.measure;
  const measures = chart.encoding?.measures ?? (measure ? [measure] : []);

  const keyOf = (row: BiRow) => groupFields.map((f) => String(row[f] ?? "")).join("\u0001");
  const groups = new Map<string, BiRow[]>();
  for (const row of rows) {
    const key = keyOf(row);
    const bucket = groups.get(key);
    if (bucket) bucket.push(row);
    else groups.set(key, [row]);
  }

  const apply = (nums: number[]): number => {
    switch (agg) {
      case "count":
        return nums.length;
      case "none":
        return 0;
      case "avg":
        return AGG_FNS.avg(nums);
      case "max":
        return AGG_FNS.max(nums);
      case "min":
        return AGG_FNS.min(nums);
      default:
        return AGG_FNS.sum(nums);
    }
  };

  const out: BiRow[] = [];
  for (const bucket of groups.values()) {
    const row: BiRow = {};
    for (const f of groupFields) row[f] = bucket[0]?.[f];
    if (agg === "count" && measures.length === 0) {
      // 纯计数：值列名用 measure 占位或固定 count
      row[measure ?? "count"] = bucket.length;
    }
    for (const m of measures) {
      row[m] = apply(bucket.map((r) => (agg === "count" ? 1 : toNumber(r[m]))));
    }
    // 非分组字段的长表 series 值在聚合后无意义（pivot 的列维度已保留），丢弃
    out.push(row);
  }
  return out;
}

export function sortRows(rows: BiRow[], chart: BiChartSchema): BiRow[] {
  const sort: BiChartSort | undefined = chart.sort;
  if (!sort) return rows;
  const dim = chart.encoding?.dimension ?? "";
  const measure = chart.encoding?.measures?.[0] ?? chart.encoding?.measure ?? "";
  const dir = sort.order === "desc" ? -1 : 1;
  const sorted = [...rows].sort((a, b) => {
    if (sort.by === "measure" && measure) {
      return (toNumber(a[measure]) - toNumber(b[measure])) * dir;
    }
    return String(a[dim] ?? "").localeCompare(String(b[dim] ?? ""), "zh-CN") * dir;
  });
  return sort.topN && sort.topN > 0 ? sorted.slice(0, sort.topN) : sorted;
}

/** 渲染管线：过滤 → 聚合 → 排序/TopN，返回供 compile 消费的行 */
export function transformRows(chart: BiChartSchema, inputRows: BiRow[]): BiRow[] {
  let rows = applyFilters(inputRows, chart.filters);
  if (shouldAggregate(chart)) {
    rows = aggregateRows(rows, chart);
  }
  rows = sortRows(rows, chart);
  return rows;
}

// ============ 字段类型推断（设计器字段选择器提示用） ============

export type BiFieldType = "number" | "date" | "string";

export function inferFieldType(rows: BiRow[] | undefined, field: string): BiFieldType {
  if (!rows?.length) return "string";
  let sawValue = false;
  let allNumber = true;
  let allDate = true;
  for (const row of rows.slice(0, 100)) {
    const v = row?.[field];
    if (v === null || v === undefined || v === "") continue;
    sawValue = true;
    if (allNumber && !Number.isFinite(Number(v))) allNumber = false;
    if (allDate && !(typeof v === "string" && /^[\d-]{8,}$/.test(v) && !Number.isNaN(Date.parse(v)))) {
      allDate = false;
    }
  }
  if (!sawValue) return "string";
  if (allNumber) return "number";
  if (allDate) return "date";
  return "string";
}

// ============ 数值格式化（轴标签/标签/提示共用） ============

export function formatNumber(
  value: number,
  format?: { decimals?: number; thousand?: boolean; prefix?: string; suffix?: string }
): string {
  let n = Number.isFinite(value) ? value : 0;
  if (format?.decimals !== undefined) n = Number(n.toFixed(format.decimals));
  let str = String(n);
  if (format?.thousand) str = str.replace(/\B(?=(\d{3})+(?!\d))/g, ",");
  return (format?.prefix ?? "") + str + (format?.suffix ?? "");
}
