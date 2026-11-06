/**
 * 命名色盘（style.palette 消费；缺省走主题色盘）。
 */
import type { BiResolvedTheme } from "./index";

export const BI_PALETTES: Record<string, string[]> = {
  default: [
    "#409eff", "#67c23a", "#e6a23c", "#f56c6c", "#909399",
    "#36cfc9", "#722ed1", "#fa8c16", "#13c2c2", "#eb2f96",
  ],
  ocean: [
    "#1976d2", "#26a69a", "#42a5f5", "#26c6da", "#5c6bc0",
    "#00acc1", "#7e57c2", "#0288d1", "#66bb6a", "#455a64",
  ],
  sunset: [
    "#f56c6c", "#e6a23c", "#fa8c16", "#d4380d", "#ff7a45",
    "#c41d7f", "#eb2f96", "#faad14", "#f759ab", "#ad6800",
  ],
  forest: [
    "#67c23a", "#389e0d", "#13c2c2", "#52c41a", "#08979c",
    "#237804", "#36cfc9", "#5b8c00", "#006d75", "#91d300",
  ],
  mono: [
    "#5b6470", "#78828e", "#98a1ab", "#3e454f", "#b9bfc7",
    "#6f7885", "#8d95a0", "#4a525c", "#c8ccd3", "#5b6470",
  ],
};

export function resolvePalette(key: string | undefined, theme: BiResolvedTheme): string[] {
  if (key && BI_PALETTES[key]) return BI_PALETTES[key];
  return theme.palette;
}

export const BI_PALETTE_KEYS = Object.keys(BI_PALETTES);
