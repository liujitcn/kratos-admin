import type { ColumnProps, TypeProps } from "@/components/ProTable/interface";

/** 表格单元格支持的对齐方式。 */
export type TableAlign = "left" | "center" | "right";

/** 获取表格行的列值，支持点号分隔的多级字段。 */
function getTableCellValue(row: Record<string, any>, prop: string) {
  return prop.split(".").reduce((value, key) => (value == null ? undefined : value[key]), row as any);
}

/** 判断值是否为可用于数值对齐的 Number 类型。 */
function isPureNumber(value: unknown) {
  return typeof value === "number" && Number.isFinite(value);
}

/** 判断列字段名是否表示时间值。 */
function isTimeColumn(column: ColumnProps) {
  const prop = column.prop?.toLowerCase();
  return !!prop && /(^|[._])(created|updated|deleted|expires|scheduled|execute|start|end|login|logout|last)[._]?at?$/.test(prop);
}

/** 判断列字段名是否表示数值。 */
function isNumericColumn(column: ColumnProps) {
  const prop = column.prop?.toLowerCase();
  return !!prop && /(^|[._])(id|sort|size|count|total|rows|priority|status_code|latency|duration|process_time|ttl|retention|attempt|qps|rate|number|num|amount|days|minutes|seconds|ms)$/.test(prop);
}

/**
 * 根据列语义和当前表格数据解析单元格对齐方式。
 *
 * 显式 align 始终优先；字典、枚举及预置状态列居中，
 * 金额列和纯数字列右对齐，其余内容左对齐。空表时会先按列语义返回默认值，
 * 数据加载后由响应式表格重新解析。
 */
export function resolveTableColumnAlign(column: ColumnProps, rows: Record<string, any>[] = []): TableAlign {
  if (column.align === "left" || column.align === "center" || column.align === "right") return column.align;

  const centeredTypes: TypeProps[] = ["selection", "radio", "index", "expand", "sort"];
  if (centeredTypes.includes(column.type as TypeProps)) return "center";
  if (column.cellType === "actions" || column.cellType === "status" || column.cellType === "image") return "center";
  if (column.cellType === "money") return "right";
  if (column.dictCode || column.tag) return "center";
  if (column.enum) return "center";
  if (isTimeColumn(column)) return "center";
  if (isNumericColumn(column)) return "right";

  if (column.prop) {
    const values = rows
      .map(row => getTableCellValue(row, column.prop as string))
      .filter(value => value !== undefined && value !== null && value !== "");
    if (values.length && values.every(isPureNumber)) return "right";
  }

  return "left";
}
