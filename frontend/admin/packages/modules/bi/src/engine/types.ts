/**
 * 图表引擎抽象：任何图表库实现这三个方法即可接入。
 * 主题在编译期注入 option（引擎层不管主题切换，换主题 = 重新 compile + update）。
 */

import type { BiChartSchema } from "../schema/types";
import type { BiRow } from "../schema/types";
import type { BiResolvedTheme } from "../theme";

/** 归一化图表事件（跨引擎联动的基础） */
export interface BiChartEvent {
  chartId: string;
  /** 命中的系列名（饼=系列即数据名；柱线=系列字段值） */
  seriesName?: string;
  /** 命中的类目名（dimension 值） */
  name?: string;
  value?: unknown;
  /** 引擎原始事件参数（逃生口） */
  raw?: unknown;
}

export interface ChartHandle {
  update(option: unknown): void;
  resize(): void;
  dispose(): void;
  /** 导出当前画布为 PNG data URL（引擎不支持时返回 undefined） */
  getDataURL?(): string | undefined;
}

export interface ChartEngine {
  /** 挂载容器并渲染首帧（option 由 compile 产出） */
  mount(el: HTMLElement, option: unknown): ChartHandle;
  /** schema + 行数据 + 主题 → 引擎原生 option */
  compile(chart: BiChartSchema, rows: BiRow[], theme: BiResolvedTheme): unknown;
  /** 订阅归一化点击事件 */
  bindEvents(handle: ChartHandle, cb: (e: BiChartEvent) => void): void;
}

// ============ 注册表 ============

const engines = new Map<string, ChartEngine>();

export function registerEngine(name: string, engine: ChartEngine): void {
  engines.set(name, engine);
}

export function getEngine(name: string | undefined): ChartEngine {
  const key = name || "echarts";
  const engine = engines.get(key);
  if (!engine) {
    throw new Error(
      `[bi] chart engine "${key}" is not registered. Install and register the corresponding adapter package (e.g. registerEngine("${key}", create${key[0]?.toUpperCase() + key.slice(1)}Engine())).`
    );
  }
  return engine;
}

export function hasEngine(name: string | undefined): boolean {
  return engines.has(name || "echarts");
}
