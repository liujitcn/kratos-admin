import type { ColumnProps, TypeProps } from "@/components/ProTable/interface";

/** 表格单元格支持的对齐方式。 */
export type TableAlign = "left" | "center" | "right";

/**
 * 根据列显式配置和组件类型解析单元格对齐方式。
 *
 * 显式 align 始终优先；选择列、状态列和图片列默认居中，金额列默认右对齐，
 * 字典与枚举列默认居中。普通字段不根据字段名或实际数据猜测，默认左对齐。
 */
export function resolveTableColumnAlign(column: ColumnProps): TableAlign {
  if (column.align === "left" || column.align === "center" || column.align === "right") return column.align;

  const centeredTypes: TypeProps[] = ["selection", "radio", "index", "expand", "sort"];
  if (centeredTypes.includes(column.type as TypeProps)) return "center";
  if (column.cellType === "actions" || column.cellType === "status" || column.cellType === "image") return "center";
  if (column.cellType === "money") return "right";
  if (column.dictCode || column.tag || column.enum) return "center";

  return "left";
}
