/**
 * 仪表盘主题预设（大屏主题库）：
 * - 空键 / "auto"：跟随宿主亮暗（现状行为）
 * - 具名预设：把图表配色模式与卡片/画布外观"钉死"——宿主无论亮暗，
 *   仪表盘都按预设渲染（大屏的核心诉求）。
 * 卡片外观经 CSS 变量 --bi-card-* 下发，两端渲染器同链消费。
 *
 * 结构化 colors 字段与语义色常量合流自 go-wind-iot 大屏（screens/theme.ts）：
 * 主题只作用于舞台配色；语义色（告警红绿/严重级别）不随主题，跨主题语义稳定。
 * schemaJson.theme 持久化完整对象时经 resolveScreenThemeColors 宽松解析：
 * 未知 key / 缺字段逐项回退到预设基线。
 */

import type { BiResolvedTheme } from "./index";
import { resolveTheme } from "./index";

/**
 * 结构化舞台色板（iot ScreenThemeColors 同形）：
 * axis/legend 供图表轴与图例；accent 为高亮色；text 为主文本。
 */
export interface BiStageColors {
  bg: string;
  panelBg: string;
  panelBorder: string;
  accent: string;
  axis: string;
  legend: string;
  text: string;
  series: string[];
}

export interface BiDashboardThemePreset {
  key: string;
  /** 中文名（设计器下拉展示） */
  name: string;
  nameEn: string;
  /** 图表色彩模式：预设钉死，不再跟随宿主亮暗 */
  mode: "light" | "dark";
  palette?: string[];
  /** 仪表盘画布底色 */
  canvasBg?: string;
  cardBg?: string;
  cardBorder?: string;
  cardTitle?: string;
  /** 结构化舞台色板（合流自 iot；轴/图例/强调色等图表细节） */
  colors?: BiStageColors;
}

export const BI_DASHBOARD_THEMES: Record<string, BiDashboardThemePreset> = {
  "tech-blue": {
    key: "tech-blue",
    name: "科技蓝·大屏",
    nameEn: "Tech Blue",
    mode: "dark",
    palette: [
      "#3ea6ff", "#00e0c6", "#7c8cff", "#ffd166", "#ff6b8a",
      "#00b8d9", "#9d7bff", "#38d982", "#ff9f43", "#5ad8f0",
    ],
    canvasBg: "#050e1f",
    cardBg: "rgba(10, 28, 58, 0.88)",
    cardBorder: "rgba(64, 169, 255, 0.38)",
    cardTitle: "#d6f0ff",
    colors: {
      bg: "#050e1f",
      panelBg: "rgba(10, 28, 58, 0.88)",
      panelBorder: "rgba(64, 169, 255, 0.38)",
      accent: "#3ea6ff",
      axis: "#7fa8cf",
      legend: "#9fd8ff",
      text: "#d6f0ff",
      series: ["#3ea6ff", "#00e0c6", "#7c8cff", "#ffd166", "#ff6b8a", "#00b8d9", "#9d7bff", "#38d982"],
    },
  },
  ocean: {
    key: "ocean",
    name: "深海蓝·大屏",
    nameEn: "Ocean",
    mode: "dark",
    palette: ["#00fdfa", "#07f7a8", "#e3b337", "#f5023d", "#0072ff", "#ff7a45", "#8fc3f5", "#b37feb", "#36cfc9", "#5ad8f0"],
    canvasBg: "#0a1628",
    cardBg: "rgba(8, 35, 63, 0.65)",
    cardBorder: "rgba(0, 168, 255, 0.35)",
    cardTitle: "#cfe6ff",
    colors: {
      bg: "#0a1628",
      panelBg: "rgba(8, 35, 63, 0.65)",
      panelBorder: "rgba(0, 168, 255, 0.35)",
      accent: "#00fdfa",
      axis: "#7fa8cf",
      legend: "#9fd8ff",
      text: "#cfe6ff",
      series: ["#00fdfa", "#07f7a8", "#e3b337", "#f5023d", "#0072ff", "#ff7a45", "#8fc3f5", "#b37feb"],
    },
  },
  forest: {
    key: "forest",
    name: "森林绿·大屏",
    nameEn: "Forest",
    mode: "dark",
    palette: ["#07f7a8", "#00fdfa", "#e3b337", "#f5023d", "#2f9e44", "#94d82d", "#66d9e8", "#b37feb", "#38d982", "#13c2c2"],
    canvasBg: "#071a12",
    cardBg: "rgba(10, 51, 32, 0.65)",
    cardBorder: "rgba(7, 247, 168, 0.35)",
    cardTitle: "#d0f5e4",
    colors: {
      bg: "#071a12",
      panelBg: "rgba(10, 51, 32, 0.65)",
      panelBorder: "rgba(7, 247, 168, 0.35)",
      accent: "#07f7a8",
      axis: "#7fcfa8",
      legend: "#a8f0d0",
      text: "#d0f5e4",
      series: ["#07f7a8", "#00fdfa", "#e3b337", "#f5023d", "#2f9e44", "#94d82d", "#66d9e8", "#b37feb"],
    },
  },
  purple: {
    key: "purple",
    name: "星云紫·大屏",
    nameEn: "Nebula Purple",
    mode: "dark",
    palette: ["#b37feb", "#8c9eff", "#f78fb3", "#ffd166", "#00fdfa", "#07f7a8", "#ff7a45", "#8fc3f5", "#722ed1", "#9d7bff"],
    canvasBg: "#170f2b",
    cardBg: "rgba(42, 27, 74, 0.65)",
    cardBorder: "rgba(179, 127, 235, 0.35)",
    cardTitle: "#e4d9f7",
    colors: {
      bg: "#170f2b",
      panelBg: "rgba(42, 27, 74, 0.65)",
      panelBorder: "rgba(179, 127, 235, 0.35)",
      accent: "#b37feb",
      axis: "#a08cc8",
      legend: "#cbb8ee",
      text: "#e4d9f7",
      series: ["#b37feb", "#8c9eff", "#f78fb3", "#ffd166", "#00fdfa", "#07f7a8", "#ff7a45", "#8fc3f5"],
    },
  },
  dawn: {
    key: "dawn",
    name: "暮光橙·大屏",
    nameEn: "Dawn",
    mode: "dark",
    palette: ["#ffb14d", "#ff7a45", "#f5023d", "#e3b337", "#07f7a8", "#00fdfa", "#8fc3f5", "#b37feb", "#fa8c16", "#eb2f96"],
    canvasBg: "#1f1410",
    cardBg: "rgba(66, 40, 24, 0.65)",
    cardBorder: "rgba(255, 177, 77, 0.35)",
    cardTitle: "#f7e8d8",
    colors: {
      bg: "#1f1410",
      panelBg: "rgba(66, 40, 24, 0.65)",
      panelBorder: "rgba(255, 177, 77, 0.35)",
      accent: "#ffb14d",
      axis: "#cfa88a",
      legend: "#f0d4b8",
      text: "#f7e8d8",
      series: ["#ffb14d", "#ff7a45", "#f5023d", "#e3b337", "#07f7a8", "#00fdfa", "#8fc3f5", "#b37feb"],
    },
  },
  "midnight-violet": {
    key: "midnight-violet",
    name: "午夜紫·大屏",
    nameEn: "Midnight Violet",
    mode: "dark",
    palette: [
      "#9d7bff", "#ff6b8a", "#5ad8f0", "#ffd166", "#7cf29c",
      "#c77dff", "#4cc9f0", "#ff9e6d", "#b892ff", "#6ef3d6",
    ],
    canvasBg: "#0d0821",
    cardBg: "rgba(30, 21, 66, 0.88)",
    cardBorder: "rgba(157, 123, 255, 0.4)",
    cardTitle: "#e8deff",
  },
  "aurora-teal": {
    key: "aurora-teal",
    name: "极光青·大屏",
    nameEn: "Aurora Teal",
    mode: "dark",
    palette: [
      "#00e0c6", "#38d982", "#4cc9f0", "#a7f3d0", "#ffd166",
      "#059fb0", "#7cf29c", "#36cfc9", "#94e3ff", "#2b9348",
    ],
    canvasBg: "#041f1e",
    cardBg: "rgba(6, 51, 48, 0.88)",
    cardBorder: "rgba(0, 224, 198, 0.35)",
    cardTitle: "#d0fff8",
  },
  graphite: {
    key: "graphite",
    name: "石墨灰·大屏",
    nameEn: "Graphite",
    mode: "dark",
    palette: [
      "#8ab4f8", "#a8adb8", "#c7ccd6", "#6f7885", "#dfe3e8",
      "#5f8dd3", "#98a1ab", "#b0b8c2", "#7d8590", "#9aa5b1",
    ],
    canvasBg: "#111318",
    cardBg: "rgba(26, 29, 36, 0.9)",
    cardBorder: "rgba(200, 204, 211, 0.22)",
    cardTitle: "#e8eaed",
  },
  paper: {
    key: "paper",
    name: "清晨·浅色",
    nameEn: "Morning Paper",
    mode: "light",
    palette: [
      "#409eff", "#67c23a", "#e6a23c", "#f56c6c", "#909399",
      "#36cfc9", "#722ed1", "#fa8c16", "#13c2c2", "#eb2f96",
    ],
    canvasBg: "#eef2f7",
    cardBg: "#ffffff",
    cardBorder: "#e4e9f0",
    cardTitle: "#1f2937",
  },
};

export const BI_THEME_PRESET_KEYS = Object.keys(BI_DASHBOARD_THEMES);

/**
 * 语义色：不随主题翻转（告警红绿 / 严重级别跨主题语义稳定）。
 * 图表/组件渲染"状态语义"时用这组，配色仍走主题色盘。
 */
export const BI_SEMANTIC_COLORS = {
  danger: "#f5023d",
  warning: "#e3b337",
  success: "#07f7a8",
  info: "#0072ff",
  offline: "#5f88b0",
} as const;

export function getDashboardThemePreset(key: string | undefined): BiDashboardThemePreset | null {
  if (!key || key === "auto") return null;
  return BI_DASHBOARD_THEMES[key] ?? null;
}

/**
 * 舞台色板宽松解析（iot resolveTheme 同语义）：
 * 传入 { key, ...部分颜色覆盖 }，未知 key / 缺字段逐项回退预设基线；
 * series 数组不足 4 色也回退。无主题返回 null（走宿主亮暗链）。
 */
export function resolveStageColors(
  theme?: ({ key?: string } & Partial<BiStageColors>) | null
): BiStageColors | null {
  const base = getDashboardThemePreset(theme?.key)?.colors;
  if (!base) {
    // 无结构化基线的预设（或 auto）：不给舞台色板，走变量链
    return null;
  }
  if (!theme) return base;
  return {
    bg: theme.bg || base.bg,
    panelBg: theme.panelBg || base.panelBg,
    panelBorder: theme.panelBorder || base.panelBorder,
    accent: theme.accent || base.accent,
    axis: theme.axis || base.axis,
    legend: theme.legend || base.legend,
    text: theme.text || base.text,
    series: Array.isArray(theme.series) && theme.series.length >= 4 ? theme.series : base.series,
  };
}

/**
 * 解析生效主题：具名预设以 mode/palette 钉死图表主题；
 * 无预设回落亮暗自动（现状行为）。
 */
export function resolveDashboardTheme(
  key: string | undefined,
  autoIsDark: boolean
): { theme: BiResolvedTheme; preset: BiDashboardThemePreset | null } {
  const preset = getDashboardThemePreset(key);
  if (!preset) {
    return { theme: resolveTheme(autoIsDark ? "dark" : "light"), preset: null };
  }
  const base = resolveTheme(preset.mode);
  return {
    theme: preset.palette ? { ...base, palette: preset.palette } : base,
    preset,
  };
}
