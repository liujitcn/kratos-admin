import type { BiCard, BiWorkbench } from "./rpc/bi/admin/v1/bi_card";
import type { BiQueryResult, BiQuerySpec } from "./rpc/bi/admin/v1/bi_query";
import type { PageBiDashboardResponse } from "./rpc/bi/admin/v1/bi_dashboard";

/** 补齐 HTTP JSON 省略的查询集合字段，保留服务端返回的过滤、排序与时间范围。 */
export function normalizeBiQuerySpec(query: BiQuerySpec): BiQuerySpec {
  return {
    ...query,
    dimensions: query.dimensions ?? [],
    measures: query.measures ?? [],
    filters: (query.filters ?? []).map(filter => ({ ...filter, values: filter.values ?? [] })),
    orders: query.orders ?? []
  };
}

/** 将无数据查询的省略字段统一为可渲染的空结果。 */
export function normalizeBiQueryResult(result: BiQueryResult): BiQueryResult {
  return {
    ...result,
    columns: result.columns ?? [],
    rows: (result.rows ?? []).map(row => ({ values: row.values ?? [] })),
    truncated: result.truncated ?? false,
    elapsed_ms: result.elapsed_ms ?? 0,
    query_record_id: result.query_record_id ?? 0
  };
}

/** 在卡片 API 边界补齐查询配置，页面不再重复处理缺省集合。 */
export function normalizeBiCard(card: BiCard): BiCard {
  return { ...card, query: card.query ? normalizeBiQuerySpec(card.query) : undefined };
}

/** 新账号和清空后的工作台使用相同的空集合结构。 */
export function normalizeBiWorkbench(workbench: BiWorkbench): BiWorkbench {
  return {
    dashboard_ids: workbench.dashboard_ids ?? [],
    favorite_ids: workbench.favorite_ids ?? [],
    dashboards: workbench.dashboards ?? []
  };
}

/** 将仪表盘分页响应中的省略集合补成可迭代结构。 */
export function normalizeBiDashboardPage(response: PageBiDashboardResponse): PageBiDashboardResponse {
  return { ...response, dashboards: response.dashboards ?? [], total: response.total ?? 0 };
}
