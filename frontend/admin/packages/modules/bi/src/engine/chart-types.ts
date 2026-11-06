/**
 * 图型注册表：type → engine 映射。
 * 内置 8 种 echarts 图型；引擎适配器包（bi-engine-s2 等）经
 * registerChartType("pivot", "s2") 扩展设计器的图型选择与引擎解析。
 */

import type { BiChartType } from "../schema/types";

const BUILTIN_CHART_TYPES: BiChartType[] = [
  "bar",
  "line",
  "area",
  "pie",
  "donut",
  "scatter",
  "radar",
  "gauge",
  "funnel",
  "treemap",
  "sunburst",
  "sankey",
  "heatmap",
];

/** 边列表 / 矩阵型图：数据是边或三元组，默认不做维度聚合 */
export const NON_AGG_TYPES = new Set(["sankey", "graph", "heatmap"]);

const typeToEngine = new Map<string, string>();
for (const t of BUILTIN_CHART_TYPES) typeToEngine.set(t, "echarts");

/** 注册新图型（设计器图型网格与 addChart 的引擎解析随之扩展）；重复注册覆盖 */
export function registerChartType(type: string, engine: string): void {
  typeToEngine.set(type, engine);
}

/** 已注册的全部图型（内置 + 扩展，按注册顺序） */
export function getChartTypes(): BiChartType[] {
  return [...typeToEngine.keys()] as BiChartType[];
}

/** 图型对应的引擎名（未注册图型回落 echarts） */
export function getEngineForType(type: string | undefined): string {
  return typeToEngine.get(type ?? "") ?? "echarts";
}
