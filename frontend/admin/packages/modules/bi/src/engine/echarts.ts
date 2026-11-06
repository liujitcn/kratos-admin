/**
 * ECharts 引擎适配器（默认内置引擎）。
 * 按需注册图表与组件（素材：go-wind-admin ele 端 plugins/echarts 注册表），
 * 避免 import * as echarts from 'echarts' 的全量引入。
 */
import * as echarts from "echarts/core";
import {
  BarChart,
  FunnelChart,
  GaugeChart,
  HeatmapChart,
  LineChart,
  PieChart,
  RadarChart,
  SankeyChart,
  ScatterChart,
  SunburstChart,
  TreemapChart,
} from "echarts/charts";
import {
  DataZoomComponent,
  VisualMapComponent,
  DataZoomInsideComponent,
  DataZoomSliderComponent,
  DatasetComponent,
  GridComponent,
  LegendComponent,
  TitleComponent,
  TooltipComponent,
  TransformComponent,
} from "echarts/components";
import { LabelLayout, UniversalTransition } from "echarts/features";
import { CanvasRenderer } from "echarts/renderers";

import type {
  BarSeriesOption,
  GaugeSeriesOption,
  LineSeriesOption,
  PieSeriesOption,
  RadarSeriesOption,
  ScatterSeriesOption,
} from "echarts/charts";
import type { ComposeOption } from "echarts/core";
import type {
  GridComponentOption,
  LegendComponentOption,
  TitleComponentOption,
  TooltipComponentOption,
} from "echarts/components";

import { getColumns, toNumber, unique } from "../dataset";
import { createDimensionLabelFormatter } from "../dataset/labels";
import { formatNumber } from "../dataset/transform";
import { resolvePalette } from "../theme/palettes";
import type { BiResolvedTheme } from "../theme";
import type { BiChartSchema, BiRow } from "../schema/types";
import type { BiChartEvent, ChartEngine, ChartHandle } from "./types";
import { registerEngine } from "./types";

let registered = false;

function ensureRegistered() {
  if (registered) return;
  registered = true;
  echarts.use([
    BarChart,
    LineChart,
    PieChart,
    ScatterChart,
    RadarChart,
    GaugeChart,
    FunnelChart,
    TreemapChart,
    SankeyChart,
    HeatmapChart,
    SunburstChart,
    DatasetComponent,
    GridComponent,
    LegendComponent,
    TitleComponent,
    TooltipComponent,
    TransformComponent,
    VisualMapComponent,
    DataZoomComponent,
    DataZoomInsideComponent,
    DataZoomSliderComponent,
    LabelLayout,
    UniversalTransition,
    CanvasRenderer,
  ]);
}

export type BiEChartsOption = ComposeOption<
  | BarSeriesOption
  | LineSeriesOption
  | PieSeriesOption
  | ScatterSeriesOption
  | RadarSeriesOption
  | GaugeSeriesOption
  | TitleComponentOption
  | LegendComponentOption
  | TooltipComponentOption
  | GridComponentOption
>;

/** 轴最大值取整到「好看的刻度」（1/2/5×10^n） */
function niceMax(value: number): number {
  if (value <= 0) return 100;
  const exp = Math.floor(Math.log10(value));
  const base = Math.pow(10, exp);
  const scaled = value / base;
  const nice = scaled <= 1 ? 1 : scaled <= 2 ? 2 : scaled <= 5 ? 5 : 10;
  return nice * base;
}

function axisStyles(theme: BiResolvedTheme) {
  return {
    axisLabel: { color: theme.subText },
    axisLine: { lineStyle: { color: theme.axisLine } },
    axisTick: { lineStyle: { color: theme.axisLine } },
  };
}

function tooltipStyles(theme: BiResolvedTheme) {
  return {
    backgroundColor: theme.tooltipBg,
    borderColor: theme.axisLine,
    textStyle: { color: theme.text },
  };
}

/** schema + 行数据 + 主题模式 → ECharts option */
export function compileEcharts(
  chart: BiChartSchema,
  rows: BiRow[],
  theme: BiResolvedTheme
): BiEChartsOption {
  const enc = chart.encoding ?? {};
  const style = chart.style ?? {};
  const cols = getColumns(rows);
  const dimensionLabel = createDimensionLabelFormatter(rows.map(row => row[enc.dimension ?? cols[0] ?? ""]), enc.dimension);
  const seriesLabel = createDimensionLabelFormatter(rows.map(row => row[enc.series ?? cols[1] ?? ""]), enc.series);
  const fmt = style.format;
  const palette = resolvePalette(style.palette, theme);
  const fmtTooltip = fmt ? { valueFormatter: (v: unknown) => formatNumber(toNumber(v), fmt) } : {};
  const base: BiEChartsOption = {
    backgroundColor: "transparent",
    color: palette,
    textStyle: { color: theme.text },
    tooltip: { ...tooltipStyles(theme), ...fmtTooltip },
  };
  // 图例显隐：显式 style.showLegend 优先，否则多系列自动显示
  const multiSeries = !!(enc.series || (enc.measures?.length ?? 0) > 1);
  const showLegend = style.showLegend ?? multiSeries;
  // 引擎原生透传：浅合并每个顶层键（嵌套覆盖用整键替换语义）
  const passthrough = (chart.options ?? {}) as object;
  // y 轴数值格式化
  const yAxisLabel = fmt ? { formatter: (v: number) => formatNumber(v, fmt) } : {};

  const labelOption = style.showLabel ? { label: { show: true, color: theme.subText } } : {};
  const cartesian = (seriesIn: any[], categories: string[], area: boolean): BiEChartsOption => {
    const series = seriesIn.map((sr) => ({
      ...labelOption,
      ...(style.stack ? { stack: "total" } : {}),
      ...sr,
    }));
    const hasLegend = showLegend && (multiSeries || seriesIn.length > 1 || enc.series);
    return ({
    ...base,
    legend: hasLegend
      ? { top: 0, textStyle: { color: theme.subText }, itemWidth: 14, itemHeight: 8, formatter: (name: string) => name.split(" · ").map(seriesLabel).join(" · ") }
      : undefined,
    grid: { left: 8, right: 16, top: hasLegend ? 32 : 20, bottom: style.dataZoom ? 32 : 8, outerBoundsMode: "same", outerBoundsContain: "axisLabel" },
    xAxis: {
      ...(chart.type === "scatter" ? { type: "value" } : { type: "category", data: categories }),
      ...axisStyles(theme),
      ...(chart.type !== "scatter" ? { axisLabel: { color: theme.subText, formatter: dimensionLabel } } : {}),
      splitLine: { show: false },
    },
    yAxis: {
      type: "value",
      ...axisStyles(theme),
      axisLabel: { ...axisStyles(theme).axisLabel, ...yAxisLabel },
      splitLine: { lineStyle: { color: theme.splitLine } },
    },
    ...(style.dataZoom
      ? { dataZoom: [{ type: "inside" }, { type: "slider", height: 16, bottom: 4, ...(chart.type !== "scatter" ? { labelFormatter: (_value: number, text: string) => dimensionLabel(text) } : {}) }] }
      : {}),
    series: series.map((s) => ({
      smooth: true,
      ...(area ? { areaStyle: { opacity: 0.15 } } : {}),
      ...s,
    })),
    ...passthrough,
    } as BiEChartsOption);
  };

  switch (chart.type) {
    case "bar":
    case "line":
    case "area":
    case "scatter": {
      const dim = enc.dimension ?? cols[0] ?? "";
      const measure = enc.measure ?? cols.find((c) => c !== dim) ?? "";
      // ECharts 无 area 系列：面积 = line + areaStyle（cartesian 已按 area 注入）
      const seriesType = chart.type === "area" ? "line" : chart.type;
      const categories = unique(rows.map((r) => String(r[dim] ?? "")));
      if (["line", "area"].includes(chart.type) && !chart.sort && categories.every(value => /^\d{4}-\d{2}(?:-\d{2})?(?:[ T].*)?$/.test(value)))
        categories.sort();

      let series: any[];
      if (chart.type === "scatter") {
        const group = (name: string, subset: BiRow[]) => ({
          name,
          type: "scatter" as const,
          data: subset.map((r) => [toNumber(r[dim]), toNumber(r[measure])]),
          symbolSize: 10,
        });
        series = enc.series
          ? unique(rows.map((r) => String(r[enc.series!]))).map((sn) =>
              group(sn, rows.filter((r) => String(r[enc.series!]) === sn))
            )
          : [group(chart.title ?? measure, rows)];
      } else {
        const seriesNames = enc.series ? unique(rows.map((r) => String(r[enc.series!]))) : [""];
        const measures = enc.measures?.length ? enc.measures : [measure];
        series = seriesNames.flatMap((sn) => measures.map((m) => ({
            name: enc.series ? (measures.length > 1 ? `${sn} · ${m}` : sn) : (enc.measures?.length ? m : chart.title ?? m),
            type: seriesType,
            data: categories.map((c) => {
              const hit = rows.find((r) => String(r[dim]) === c && (!enc.series || String(r[enc.series]) === sn));
              return hit ? toNumber(hit[m]) : null;
            }),
          })));
      }
      return {
        ...cartesian(series, categories, chart.type === "area"),
        ...passthrough,
      } as BiEChartsOption;
    }

    case "pie":
    case "donut": {
      const dim = enc.dimension ?? cols[0] ?? "name";
      const measure = enc.measure ?? cols.find((c) => c !== dim) ?? "value";
      return {
        ...base,
        legend: style.showLegend === false
          ? undefined
          : { bottom: 0, textStyle: { color: theme.subText }, itemWidth: 14, itemHeight: 8, formatter: dimensionLabel },
        series: [
          {
            type: "pie",
            radius: chart.type === "donut" ? ["42%", "68%"] : "68%",
            center: ["50%", "46%"],
            itemStyle: { borderColor: "transparent", borderWidth: 2 },
            label: {
              show: style.showLabel ?? true,
              color: theme.subText,
              formatter: (p: any) => dimensionLabel(p.name) + (fmt ? ` ${formatNumber(toNumber(p.value), fmt)}` : ""),
            },
            data: rows.map((r) => ({
              name: String(r[dim] ?? ""),
              value: toNumber(r[measure]),
            })),
          },
        ],
        ...passthrough,
      } as BiEChartsOption;
    }

    case "radar": {
      const dim = enc.dimension ?? cols[0] ?? "name";
      const measure = enc.measure ?? cols.find((c) => c !== dim) ?? "value";
      const max = niceMax(Math.max(...rows.map((x) => toNumber(x[measure])), 1));
      const categories = unique(rows.map((r) => String(r[dim] ?? "")));
      const indicators = categories.map((name) => ({ name, max }));

      let data: any[];
      if (enc.series) {
        const seriesNames = unique(rows.map((r) => String(r[enc.series!])));
        data = seriesNames.map((sn) => ({
          name: sn,
          value: categories.map((category) => {
            const hit = rows.find((r) => String(r[dim]) === category && String(r[enc.series!]) === sn);
            return hit ? toNumber(hit[measure]) : null;
          }),
        }));
      } else {
        data = [{ name: chart.title ?? measure, value: rows.map((r) => toNumber(r[measure])) }];
      }
      return {
        ...base,
        legend: enc.series
          ? { bottom: 0, textStyle: { color: theme.subText }, itemWidth: 14, itemHeight: 8, formatter: seriesLabel }
          : undefined,
        radar: {
          indicator: indicators,
          center: ["50%", "50%"],
          radius: "62%",
          axisName: { color: theme.subText, formatter: dimensionLabel },
          axisLine: { lineStyle: { color: theme.axisLine } },
          splitLine: { lineStyle: { color: theme.splitLine } },
          splitArea: { show: false },
        },
        series: [{ type: "radar", data, areaStyle: { opacity: 0.12 } }],
        ...passthrough,
      } as BiEChartsOption;
    }

    case "gauge": {
      const measure = enc.measure ?? cols[0] ?? "value";
      const value = toNumber(rows[0]?.[measure]);
      const max = niceMax(Math.max(value, 100));
      return {
        ...base,
        series: [
          {
            type: "gauge",
            min: 0,
            max,
            startAngle: 210,
            endAngle: -30,
            progress: { show: true, width: 12 },
            axisLine: { lineStyle: { width: 12, color: [[1, theme.splitLine]] } },
            axisTick: { show: false },
            splitLine: { length: 6, lineStyle: { color: theme.axisLine } },
            axisLabel: { color: theme.subText, distance: 16, fontSize: 10 },
            pointer: { itemStyle: { color: "auto" } },
            detail: {
              valueAnimation: true,
              formatter: fmt
                ? (v: number) => formatNumber(v, fmt)
                : "{value}",
              color: theme.text,
              fontSize: 22,
              offsetCenter: [0, "62%"],
            },
            data: [{ value }],
          },
        ],
        ...passthrough,
      } as BiEChartsOption;
    }

    case "funnel": {
      const dim = enc.dimension ?? cols[0] ?? "name";
      const measure = enc.measure ?? cols.find((c) => c !== dim) ?? "value";
      // 漏斗要求降序
      const sorted = [...rows]
        .map((r) => ({ name: String(r[dim] ?? ""), value: toNumber(r[measure]) }))
        .sort((a, b) => b.value - a.value);
      return {
        ...base,
        tooltip: { ...tooltipStyles(theme), trigger: "item", ...fmtTooltip },
        series: [
          {
            type: "funnel",
            left: "10%",
            top: 12,
            bottom: 12,
            width: "80%",
            sort: "descending",
            gap: 2,
            label: { show: true, position: "inside", color: "#fff", formatter: (p: any) => dimensionLabel(p.name) + (fmt ? ` ${formatNumber(toNumber(p.value), fmt)}` : "") },
            data: sorted,
          },
        ],
        ...passthrough,
      } as BiEChartsOption;
    }

    case "treemap": {
      const dim = enc.dimension ?? cols[0] ?? "name";
      const measure = enc.measure ?? cols.find((c) => c !== dim) ?? "value";
      return {
        ...base,
        series: [
          {
            type: "treemap",
            left: 8,
            right: 8,
            top: 12,
            bottom: 8,
            roam: false,
            nodeClick: false,
            breadcrumb: { show: false },
            label: { show: true, color: "#fff", formatter: (p: any) => dimensionLabel(p.name) + (fmt ? ` ${formatNumber(toNumber(p.value), fmt)}` : "") },
            data: rows.map((r) => ({
              name: String(r[dim] ?? ""),
              value: toNumber(r[measure]),
            })),
          },
        ],
        ...passthrough,
      } as BiEChartsOption;
    }

    case "sunburst": {
      // 两层旭日：series 分组为内环，维度为外环叶子
      const dim = enc.dimension ?? cols[0] ?? "name";
      const measure = enc.measure ?? cols.find((c) => c !== dim) ?? "value";
      let data: any[];
      if (enc.series) {
        const seriesNames = unique(rows.map((r) => String(r[enc.series!])));
        data = seriesNames.map((sn) => {
          const subset = rows.filter((r) => String(r[enc.series!]) === sn);
          return {
            name: sn,
            children: subset.map((r) => ({
              name: String(r[dim] ?? ""),
              value: toNumber(r[measure]),
            })),
          };
        });
      } else {
        data = rows.map((r) => ({
          name: String(r[dim] ?? ""),
          value: toNumber(r[measure]),
        }));
      }
      return {
        ...base,
        series: [
          {
            type: "sunburst",
            radius: [enc.series ? "18%" : "0%", "82%"],
            label: { color: theme.subText, rotate: "radial", formatter: (p: any) => dimensionLabel(seriesLabel(p.name)) },
            itemStyle: { borderColor: "transparent", borderWidth: 1 },
            data,
          },
        ],
        ...passthrough,
      } as BiEChartsOption;
    }

    case "sankey": {
      // 边列表：dimension=起点，series=终点，measure=流量
      const sourceField = enc.dimension ?? cols[0] ?? "source";
      const targetField = enc.series ?? cols[1] ?? "target";
      const valueField = enc.measure ?? cols[2] ?? "value";
      const nodeIds = new Set<string>();
      const links: any[] = [];
      for (const row of rows) {
        const source = String(row[sourceField] ?? "");
        const target = String(row[targetField] ?? "");
        if (!source || !target) continue;
        nodeIds.add(source);
        nodeIds.add(target);
        links.push({ source, target, value: toNumber(row[valueField]) });
      }
      return {
        ...base,
        series: [
          {
            type: "sankey",
            left: 16,
            right: 16,
            top: 12,
            bottom: 12,
            nodeWidth: 14,
            nodeGap: 10,
            emphasis: { focus: "adjacency" },
            lineStyle: { color: "gradient", curveness: 0.5 },
            label: { color: theme.subText, formatter: (p: any) => dimensionLabel(seriesLabel(p.name)) },
            data: [...nodeIds].map((id) => ({ name: id })),
            links,
          },
        ],
        ...passthrough,
      } as BiEChartsOption;
    }

    case "heatmap": {
      // 矩阵三元组：dimension=x 轴，series=y 轴，measure=值
      const xField = enc.dimension ?? cols[0] ?? "x";
      const yField = enc.series ?? cols[1] ?? "y";
      const valueField = enc.measure ?? cols[2] ?? "value";
      const xCats = unique(rows.map((r) => String(r[xField] ?? "")));
      const yCats = unique(rows.map((r) => String(r[yField] ?? "")));
      const data = rows.map((r) => [
        xCats.indexOf(String(r[xField] ?? "")),
        yCats.indexOf(String(r[yField] ?? "")),
        toNumber(r[valueField]),
      ]);
      const max = Math.max(...rows.map((r) => toNumber(r[valueField])), 1);
      return {
        ...base,
        // 为横向色阶和坐标标签分别留出空间，避免覆盖最后一行数据。
        grid: { left: 8, right: 24, top: 12, bottom: 68, outerBoundsMode: "same", outerBoundsContain: "axisLabel" },
        xAxis: { type: "category", data: xCats, ...axisStyles(theme), axisLabel: { color: theme.subText, formatter: dimensionLabel }, splitArea: { show: true } },
        yAxis: { type: "category", data: yCats, ...axisStyles(theme), axisLabel: { color: theme.subText, formatter: seriesLabel }, splitArea: { show: true } },
        visualMap: {
          min: 0,
          max,
          calculable: true,
          orient: "horizontal",
          left: "center",
          bottom: 0,
          textStyle: { color: theme.subText },
          inRange: { color: theme.palette.slice(0, 5) },
        },
        series: [
          {
            type: "heatmap",
            data,
            label: { show: chart.style?.showLabel ?? true, color: theme.text, fontSize: 10 },
          },
        ],
        ...passthrough,
      } as BiEChartsOption;
    }

    default:
      throw new Error(`[bi] unsupported chart type: ${(chart as any).type}`);
  }
}

/**
 * 创建 ECharts 引擎实例（幂等注册 echarts 模块）。
 * handle 上带内部约定字段 __echartsInstance 供 bindEvents 取实例。
 */
export function createEchartsEngine(): ChartEngine {
  ensureRegistered();
  return {
    mount(el, option) {
      const instance = echarts.init(el);
      instance.setOption(option as any);
      const handle: ChartHandle & { __echartsInstance?: echarts.ECharts } = {
        __echartsInstance: instance,
        update(next) {
          instance.setOption(next as any, { notMerge: true });
        },
        resize() {
          instance.resize();
        },
        dispose() {
          instance.dispose();
        },
        getDataURL() {
          return instance.getDataURL({ type: "png", pixelRatio: 2, backgroundColor: "transparent" });
        },
      };
      return handle;
    },
    compile(chart, rows, theme) {
      return compileEcharts(chart, rows, theme);
    },
    bindEvents(handle, cb) {
      const inst = (handle as any).__echartsInstance as echarts.ECharts | undefined;
      inst?.on("click", (params: any) => {
        const e: BiChartEvent = {
          chartId: "",
          seriesName: params?.seriesName,
          name: params?.name,
          value: params?.value,
          raw: params,
        };
        cb(e);
      });
    },
  };
}

/** 默认注册（调用方引入本模块后执行一次；重复注册无害） */
export function installEchartsEngine(): void {
  registerEngine("echarts", createEchartsEngine());
}
