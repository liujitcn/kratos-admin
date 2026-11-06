/** BI 图表内核：真实结果的字段映射、数据转换、图表引擎、布局与主题。 */

// Schema 与缺省
export * from "./schema/types";
export { flowPlace } from "./schema/layout";

// 数据集解析
export { getColumns, toNumber, unique } from "./dataset";

// 主题
export { BI_THEME_DARK, BI_THEME_LIGHT, detectDark, resolveTheme, watchDark } from "./theme";
export type { BiResolvedTheme } from "./theme";
export { BI_PALETTES, BI_PALETTE_KEYS, resolvePalette } from "./theme/palettes";
export {
  BI_DASHBOARD_THEMES,
  BI_SEMANTIC_COLORS,
  BI_THEME_PRESET_KEYS,
  getDashboardThemePreset,
  resolveDashboardTheme,
  resolveStageColors
} from "./theme/presets";
export type { BiDashboardThemePreset, BiStageColors } from "./theme/presets";

// 数据变换（过滤/聚合/排序/TopN + 字段类型推断 + 数值格式化）
export { applyFilters, aggregateRows, sortRows, transformRows, inferFieldType, formatNumber } from "./dataset/transform";
export type { BiFieldType } from "./dataset/transform";

// 引擎
export { createEchartsEngine, compileEcharts, installEchartsEngine } from "./engine/echarts";
export type { BiEChartsOption } from "./engine/echarts";
export { getEngine, hasEngine, registerEngine } from "./engine/types";
export type { BiChartEvent, ChartEngine, ChartHandle } from "./engine/types";

// 图型注册表（引擎适配器包扩展设计器图型）
export { registerChartType, getChartTypes, getEngineForType } from "./engine/chart-types";

// 文案包（纯数据）
export { BI_MESSAGES_ZH, BI_MESSAGES_EN } from "./bi-messages";

import { installEchartsEngine } from "./engine/echarts";

// 包加载即注册内置引擎
installEchartsEngine();
