import { flowPlace } from "./schema/layout";
import { normalizeBiQueryResult, normalizeBiQuerySpec } from "./bi-response";
import type { BiCard } from "./rpc/bi/admin/v1/bi_card";
import type { BiDashboardDataResponse } from "./rpc/bi/admin/v1/bi_dashboard";
import type { BiQuerySpec } from "./rpc/bi/admin/v1/bi_query";

/** 仪表盘只保存名称、卡片引用与栅格位置，外观跟随管理端主题。 */
export interface DashboardLayout {
  schemaVersion: 3;
  name: string;
  cards: DashboardCardLayout[];
}

/** 单张数据卡片的布局与服务端发布快照。 */
export interface DashboardCardLayout {
  id: string;
  card_id: number;
  query_record_id?: number;
  x?: number;
  y?: number;
  w: number;
  h: number;
  name?: string;
  chart_type?: string;
  chart_json?: string;
  query?: BiQuerySpec;
}

/** 读取当前协议的仪表盘布局，不接收旧图表与数据集结构。 */
export function parseDashboardLayout(value: string): DashboardLayout {
  if (!value) return { schemaVersion: 3, name: "", cards: [] };
  const layout = JSON.parse(value) as DashboardLayout;
  if (layout.schemaVersion !== 3 || !Array.isArray(layout.cards)) throw new Error("仪表盘布局格式无效");
  return { schemaVersion: 3, name: layout.name, cards: layout.cards };
}

/** 将发布快照还原成只读数据卡片，保留当次图表配置。 */
export function dashboardSnapshotCard(item: DashboardCardLayout): BiCard {
  return {
    id: item.card_id,
    tenant_id: 0,
    project_id: 0,
    owner_id: 0,
    query_record_id: item.query_record_id || 0,
    name: item.name || "",
    source_code: item.query?.source_code || "",
    query: item.query ? normalizeBiQuerySpec(item.query) : undefined,
    chart_type: item.chart_type || "bar",
    chart_json: item.chart_json || "",
    refresh_seconds: 0,
    is_shared: false,
    updated_at: "",
    category: "shared",
    can_edit: false
  };
}

/** 按发布快照匹配真实查询结果，并补齐 HTTP 省略的空集合。 */
export function dashboardResults(data: BiDashboardDataResponse) {
  return new Map((data.cards ?? []).map(item => [item.item_id, item.result ? normalizeBiQueryResult(item.result) : undefined]));
}

/** 复用现有栅格算法整理卡片位置，避免拖动排序后互相覆盖。 */
export function reflowDashboardCards(cards: DashboardCardLayout[]): DashboardCardLayout[] {
  return flowPlace(
    cards.map(item => ({ ...item, x: undefined as number | undefined, y: undefined as number | undefined })),
    12
  ).map(item => ({ ...item, x: item.x ?? 0, y: item.y ?? 0 }));
}
