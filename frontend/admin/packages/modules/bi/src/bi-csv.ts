import type { BiQueryResult } from "./rpc/bi/admin/v1/bi_query";

/** 查询快照导出失败后提供的界面反馈和重试策略。 */
export type BiCsvExportFailure = "denied" | "unavailable" | "retry";

/** 根据查询记录接口的 HTTP 状态和业务原因判断能否重试。 */
export function classifyBiCsvExportFailure(error: unknown): BiCsvExportFailure {
  const response = (error as { response?: { status?: number; data?: { reason?: string } } } | null)?.response;
  const status = response?.status;
  const reason = response?.data?.reason;
  if (status === 403 || reason === "PERMISSION_DENIED") return "denied";
  if (reason === "RESOURCE_NOT_FOUND" || (status && status >= 400 && status < 500 && status !== 408 && status !== 429)) {
    return "unavailable";
  }
  return "retry";
}

/** 转成 CSV，保留列顺序和字符串数值，并防止表格软件把文本当作公式。 */
export function buildBiCsv(result: Pick<BiQueryResult, "columns" | "rows">): string {
  const columns = result.columns ?? [];
  return [
    columns.map(column => csvCell(column.name || column.code || "")),
    ...(result.rows ?? []).map(row => columns.map((column, index) => csvCell(row.values?.[index] ?? "", column.type === "number")))
  ].map(row => row.join(",")).join("\r\n");
}

/** 下载已经取得的查询结果快照。 */
export function downloadBiCsv(result: Pick<BiQueryResult, "columns" | "rows">, name: string) {
  const url = URL.createObjectURL(new Blob(["\ufeff", buildBiCsv(result)], { type: "text/csv;charset=utf-8" }));
  const link = document.createElement("a");
  link.href = url;
  link.download = `${name}.csv`;
  link.click();
  window.setTimeout(() => URL.revokeObjectURL(url), 0);
}

/** 指标列合法的带符号数值保持数值语义，其他公式前缀作为文字导出。 */
function csvCell(value: string, numeric = false): string {
  const formulaPrefix = /^[\t\r\n]|^\s*[=+\-@＝＋－＠]/u;
  const strictNumber = /^[+-]?(?:\d+(?:\.\d*)?|\.\d+)(?:e[+-]?\d+)?$/i;
  const safe = formulaPrefix.test(value) && !(numeric && strictNumber.test(value)) ? `'${value}` : value;
  return `"${safe.replaceAll('"', '""')}"`;
}
