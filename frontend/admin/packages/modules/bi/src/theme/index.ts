/**
 * BI 主题 token：亮/暗两套图表配色，对齐 EP 语义色与 go-wind 设计语言暗色底。
 * 主题在编译期写进引擎 option（backgroundColor 恒为 transparent，宿主背景透出）。
 */

export interface BiResolvedTheme {
  mode: "light" | "dark";
  /** 系列色盘（10 色） */
  palette: string[];
  /** 主文本（标题/图例） */
  text: string;
  /** 次文本（轴标签/饼标签） */
  subText: string;
  /** 轴线 */
  axisLine: string;
  /** 分隔线 */
  splitLine: string;
  /** tooltip 底色 */
  tooltipBg: string;
}

const PALETTE = [
  "#409eff",
  "#67c23a",
  "#e6a23c",
  "#f56c6c",
  "#909399",
  "#36cfc9",
  "#722ed1",
  "#fa8c16",
  "#13c2c2",
  "#eb2f96",
];

export const BI_THEME_LIGHT: BiResolvedTheme = {
  mode: "light",
  palette: PALETTE,
  text: "#1f2937",
  subText: "#6b7280",
  axisLine: "#e5e7eb",
  splitLine: "#f3f4f6",
  tooltipBg: "#ffffff",
};

export const BI_THEME_DARK: BiResolvedTheme = {
  mode: "dark",
  palette: PALETTE,
  // 对齐 EP 暗色文本阶（--el-text-color-* 的暗色值）
  text: "#e5eaf3",
  subText: "#a3a6ad",
  axisLine: "#4c4d4f",
  splitLine: "#414243",
  tooltipBg: "#1d1e1f",
};

export function resolveTheme(mode: "light" | "dark"): BiResolvedTheme {
  return mode === "dark" ? BI_THEME_DARK : BI_THEME_LIGHT;
}

// ============ 宿主无关的暗色探测（渲染器缺省跟随） ============

/** 非响应式读取当前是否暗色：html.dark class 优先，其次系统偏好 */
export function detectDark(): boolean {
  if (typeof document !== "undefined" && document.documentElement.classList.contains("dark")) {
    return true;
  }
  if (typeof window !== "undefined" && window.matchMedia?.("(prefers-color-scheme: dark)").matches) {
    return true;
  }
  return false;
}

/**
 * 监听暗色翻转（模块级共享一份 MutationObserver + matchMedia listener）。
 * 返回解绑函数。渲染器各端订阅它实现缺省主题跟随。
 */
export function watchDark(cb: (dark: boolean) => void): () => void {
  if (typeof document === "undefined") return () => undefined;
  let bound = false;
  const subs = new Set<(dark: boolean) => void>();
  let observer: MutationObserver | null = null;
  let mq: MediaQueryList | null = null;

  const sync = () => {
    const dark = detectDark();
    subs.forEach((s) => s(dark));
  };

  const ensure = () => {
    if (bound) return;
    bound = true;
    observer = new MutationObserver(sync);
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] });
    if (typeof window !== "undefined" && window.matchMedia) {
      mq = window.matchMedia("(prefers-color-scheme: dark)");
      mq.addEventListener("change", sync);
    }
  };

  ensure();
  subs.add(cb);
  cb(detectDark());

  return () => {
    subs.delete(cb);
    // 观察器常驻（模块级单例），不逐订阅者拆除
  };
}
