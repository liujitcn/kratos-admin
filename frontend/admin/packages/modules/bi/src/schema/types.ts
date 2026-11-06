export type BiRow = Record<string, any>;

export type BiChartType =
  | "bar"
  | "line"
  | "area"
  | "pie"
  | "donut"
  | "scatter"
  | "radar"
  | "gauge"
  | "funnel"
  | "treemap"
  | "sunburst"
  | "sankey"
  | "heatmap"
  /** 引擎适配器注册的扩展图型（如 s2 的 "pivot"、g6 的 "graph"） */
  | (string & {});

export interface BiChartEncoding {
  /** 类目轴 / 饼名 / 雷达指标名字段 */
  dimension?: string;
  /** 值轴 / 饼值 / 雷达值 / 仪表值字段（单度量） */
  measure?: string;
  /** 宽表多度量：每个字段各成一个系列（与 measure/series 互斥，优先级最高） */
  measures?: string[];
  /** 系列拆分字段（长表格式，可选） */
  series?: string;
}

/** 聚合函数：dimension 分组后对 measure 施加；"none" 不聚合 */
export type BiAggFn = "sum" | "avg" | "count" | "max" | "min" | "none";

export type BiFilterOp = "eq" | "ne" | "gt" | "gte" | "lt" | "lte" | "contains" | "in";

export interface BiChartFilter {
  field: string;
  op: BiFilterOp;
  /** eq/ne/比较/contains：标量；in：数组（UI 以逗号分隔输入） */
  value: string | number | (string | number)[];
}

export interface BiChartSort {
  /** 按维度（类目名）或度量（聚合值）排序 */
  by: "dimension" | "measure";
  order: "asc" | "desc";
  /** 取前 N 项（排序后截断；不传 = 全量） */
  topN?: number;
}

export interface BiNumberFormat {
  /** 小数位数（缺省不处理） */
  decimals?: number;
  /** 千分位 */
  thousand?: boolean;
  prefix?: string;
  suffix?: string;
}

export interface BiChartStyle {
  /** 柱状图堆叠 */
  stack?: boolean;
  /** 显式开关图例（缺省：多系列自动显示） */
  showLegend?: boolean;
  /** 系列上的数值标签 */
  showLabel?: boolean;
  /** 数据缩放（区域缩放滑块，柱/线/面/散点） */
  dataZoom?: boolean;
  /** 色盘 key，见 BI_PALETTES */
  palette?: string;
  /** 数值格式化（轴标签/标签/提示） */
  format?: BiNumberFormat;
}

export interface BiChartSchema {
  id: string;
  title?: string;
  datasetId: string;
  /** 图表引擎，默认 "echarts" */
  engine?: string;
  type: BiChartType;
  /** 12 列网格坐标（栅格单位）；x/y 缺省自动流式放置 */
  x?: number;
  y?: number;
  w?: number;
  h?: number;
  encoding: BiChartEncoding;
  /** 聚合函数（dimension 分组 + measure 聚合），缺省 "sum"（设 "none" 关闭） */
  aggregation?: BiAggFn;
  /** 排序 + TopN */
  sort?: BiChartSort;
  /** 图表级过滤器（作用于数据集行） */
  filters?: BiChartFilter[];
  /** 样式选项（堆叠/图例/标签/数据缩放/色盘/数字格式） */
  style?: BiChartStyle;
  /**
   * 下钻路径（维度层级，如 ["category","city"]）：点击图表逐层下钻，
   * 父层级值作为隐式过滤器携带；渲染器提供返回上一层的面包屑。
   */
  drill?: { path: string[] };
  /** 引擎原生透传覆盖（深合并到编译产物） */
  options?: Record<string, any>;
}
