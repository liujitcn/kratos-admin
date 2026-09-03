import type { ColumnProps, EnumProps, SearchProps } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import { defBaseLogService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_log";
import type { BaseLogTraceItem } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_log";

/** 根据枚举值生成表格筛选项。 */
export function createLogEnumOptions(items: Array<[number, string]>): EnumProps[] {
  return items.map(([value, label]) => ({ value, label }));
}

/** 根据枚举值查找展示文案。 */
export function logEnumLabel(options: EnumProps[], value: unknown, fallback = ""): string {
  return options.find(item => item.value === value)?.label ?? (fallback || String(value ?? ""));
}

/** 格式化日志 HTTP 状态码并补充状态语义。 */
export function formatLogStatusCode(value: unknown, t: (key: string) => string): string {
  const code = Number(value);
  if (!Number.isInteger(code)) return t("common.value.unknown");

  let statusKey = "system.base.log.status.other";
  if (code >= 200 && code < 300) statusKey = "system.base.log.status.success";
  else if (code >= 400 && code < 500) statusKey = "system.base.log.status.client_error";
  else if (code >= 500 && code < 600) statusKey = "system.base.log.status.server_error";
  return `${code} (${t(statusKey)})`;
}

/** 格式化日志耗时，零值表示未采集。 */
export function formatLogDuration(value: unknown, t: (key: string) => string): string {
  const duration = Number(value);
  if (!Number.isFinite(duration) || duration <= 0) return t("common.value.none");
  return `${duration} ms`;
}

/** 格式化日志字节数，保留零字节结果。 */
export function formatLogBytes(value: unknown, t: (key: string) => string): string {
  const size = Number(value);
  if (!Number.isFinite(size) || size < 0) return t("common.value.none");
  return `${size} B`;
}

/** 格式化日志数量并补充记录单位。 */
export function formatLogCount(value: unknown, t: (key: string) => string): string {
  const count = Number(value);
  if (!Number.isFinite(count) || count < 0) return t("common.value.none");
  return `${count} ${t("system.base.log.unit.records")}`;
}

/** 创建审计日志时间范围筛选项。 */
export function logDateSearch(t: (key: string) => string): SearchProps {
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
export function logDetailColumn(label: string, onClick: (id: string) => void): ColumnProps {
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
        onClick: scope => onClick(String(scope.row.id))
      }
    ]
  };
}

/** 查询同一请求或链路关联的审计时间线。 */
export async function requestLogTrace(requestID: string, traceID: string): Promise<BaseLogTraceItem[]> {
  const response = await defBaseLogService.GetBaseLogTrace({ request_id: requestID, trace_id: traceID });
  return response.items;
}
