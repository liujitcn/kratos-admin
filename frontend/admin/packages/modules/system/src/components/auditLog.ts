import type { ColumnProps, EnumProps, SearchProps } from "@liujitcn/kratos-admin-core/components/ProTable/interface";

/** 根据枚举值生成表格筛选项。 */
export function createAuditEnumOptions(items: Array<[number, string]>): EnumProps[] {
  return items.map(([value, label]) => ({ value, label }));
}

/** 根据枚举值查找展示文案。 */
export function auditEnumLabel(options: EnumProps[], value: unknown): string {
  return options.find(item => item.value === value)?.label ?? String(value ?? "--");
}

/** 创建审计日志时间范围筛选项。 */
export function auditDateSearch(t: (key: string) => string): SearchProps {
  return {
    el: "date-picker",
    props: {
      type: "daterange",
      editable: false,
      class: "!w-[240px]",
      rangeSeparator: "~",
      startPlaceholder: t("common.placeholder.start_date"),
      endPlaceholder: t("common.placeholder.end_date"),
      valueFormat: "YYYY-MM-DD"
    }
  };
}

/** 创建审计日志详情按钮列。 */
export function auditDetailColumn(label: string, onClick: (id: number) => void): ColumnProps {
  return {
    prop: "detailAction",
    label,
    width: 100,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label,
        type: "primary",
        link: true,
        onClick: scope => onClick(Number(scope.row.id))
      }
    ]
  };
}
