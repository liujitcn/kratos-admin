import { unref } from "vue";
import type { ColumnProps } from "./interface";

/** 同步最新列配置，保留未被页面配置覆盖的用户显隐和排序设置。 */
export function syncTableColumns(
  columns: ColumnProps[],
  current: ColumnProps[] = [],
  previous: ColumnProps[] = []
): ColumnProps[] {
  const currentMap = new Map(current.map((column, index) => [column.prop ?? column.type ?? index, column]));
  const previousMap = new Map(previous.map((column, index) => [column.prop ?? column.type ?? index, column]));
  return columns.map((column, index) => {
    const key = column.prop ?? column.type ?? index;
    const existing = currentMap.get(key);
    const source = previousMap.get(key);
    const isShow = unref(column.isShow) ?? true;
    return {
      ...column,
      isShow: existing && source && isShow === (unref(source.isShow) ?? true) ? existing.isShow : isShow,
      sortable: existing && source && column.sortable === source.sortable ? existing.sortable : column.sortable,
      isSetting: column.isSetting ?? true,
      isFilterEnum: column.isFilterEnum ?? true,
      ...(column.search ? { search: { ...column.search } } : {}),
      ...(column._children ? { _children: syncTableColumns(column._children, existing?._children, source?._children) } : {})
    };
  });
}
