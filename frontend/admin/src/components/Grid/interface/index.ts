/** Grid 响应式断点名称。 */
export type BreakPoint = "xs" | "sm" | "md" | "lg" | "xl";

/** 单个断点下的栅格跨度与偏移配置。 */
export type Responsive = {
  span?: number;
  offset?: number;
};
