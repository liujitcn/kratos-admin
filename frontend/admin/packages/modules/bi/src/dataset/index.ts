import type { BiRow } from "../schema/types";

/** 从行集合推导列名（所有行键的并集，保持首次出现顺序） */
export function getColumns(rows: BiRow[] | undefined): string[] {
  if (!rows?.length) return [];
  const cols: string[] = [];
  const seen = new Set<string>();
  for (const row of rows) {
    for (const key of Object.keys(row ?? {})) {
      if (!seen.has(key)) {
        seen.add(key);
        cols.push(key);
      }
    }
  }
  return cols;
}

/** 数值化：非有限数归 0（图表轴不允许 NaN/undefined） */
export function toNumber(value: unknown): number {
  const n = Number(value);
  return Number.isFinite(n) ? n : 0;
}

/** 去重保序 */
export function unique<T>(list: T[]): T[] {
  return [...new Set(list)];
}
