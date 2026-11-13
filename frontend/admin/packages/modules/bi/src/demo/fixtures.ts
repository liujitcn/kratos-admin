import type { DashboardLayout } from "../bi-dashboard-schema";
import type { CardChartConfig } from "../card-schema";
import type { BiCard } from "../rpc/bi/admin/v1/bi_card";
import type { BiQueryResult, BiQuerySpec } from "../rpc/bi/admin/v1/bi_query";

/** 本地示例使用的语义查询，不提交到查询服务。 */
export const demoQuery: BiQuerySpec = {
  source_code: "demo_sales",
  dimensions: ["region", "channel"],
  measures: [
    { code: "amount", aggregation: "sum" },
    { code: "orders", aggregation: "sum" }
  ],
  filters: [],
  limit: 100,
  time_range: undefined,
  orders: []
};

/** 查询结果的值与 columns 顺序一一对应，数值遵循 RPC 的字符串格式。 */
export const demoResult: BiQueryResult = {
  columns: [
    { code: "region", name: "区域", type: "string" },
    { code: "channel", name: "渠道", type: "string" },
    { code: "amount", name: "服务金额", type: "number" },
    { code: "orders", name: "服务次数", type: "number" }
  ],
  rows: [
    { values: ["华东", "线上", "128", "24"] },
    { values: ["华东", "线下", "96", "18"] },
    { values: ["华南", "线上", "156", "32"] },
    { values: ["华南", "线下", "112", "20"] },
    { values: ["华北", "线上", "104", "22"] },
    { values: ["华北", "线下", "84", "14"] }
  ],
  truncated: false,
  elapsed_ms: 0,
  query_record_id: 0
};

/** 图表编辑器的本地初始字段映射。 */
export const demoChartConfig: CardChartConfig = {
  encoding: { dimension: "region", measure: "amount", series: "channel" },
  style: { showLegend: true, showLabel: false, stack: false, dataZoom: false }
};

/** 未保存的演示卡片，id 为零且关闭自动刷新。 */
export const demoCard: BiCard = {
  id: 0,
  tenant_id: 0,
  project_id: 0,
  owner_id: 0,
  query_record_id: 0,
  name: "区域服务金额",
  source_code: demoQuery.source_code,
  query: demoQuery,
  chart_type: "bar",
  chart_json: JSON.stringify(demoChartConfig),
  refresh_seconds: 0,
  is_shared: false,
  updated_at: "",
  category: "mine",
  can_edit: false
};

/** 无分组指标结果，用于数值卡片。 */
export const demoMetricResult: BiQueryResult = {
  columns: demoResult.columns.filter(column => column.type === "number"),
  rows: [{ values: ["680", "130"] }],
  truncated: false,
  elapsed_ms: 0,
  query_record_id: 0
};

/** 指标卡片沿用原查询字段，使用无分组的汇总结果。 */
export const demoMetricCard: BiCard = {
  ...demoCard,
  name: "服务汇总",
  query: { ...demoQuery, dimensions: [] },
  chart_type: "metric",
  chart_json: JSON.stringify({ encoding: { measures: ["amount", "orders"] }, style: {} })
};

/** 仪表盘发布快照和数据结果使用布局项 id 关联。 */
export const demoLayout: DashboardLayout = {
  schemaVersion: 3,
  name: "本地经营概览",
  cards: [
    {
      id: "regional-sales",
      card_id: 0,
      x: 0,
      y: 0,
      w: 8,
      h: 4,
      name: demoCard.name,
      chart_type: demoCard.chart_type,
      chart_json: demoCard.chart_json,
      query: demoCard.query
    },
    {
      id: "service-metrics",
      card_id: 0,
      x: 8,
      y: 0,
      w: 4,
      h: 4,
      name: demoMetricCard.name,
      chart_type: demoMetricCard.chart_type,
      chart_json: demoMetricCard.chart_json,
      query: demoMetricCard.query
    }
  ]
};

/** 仪表盘只读取本地结果，不按卡片编号重新取数。 */
export const demoResults = new Map<string, BiQueryResult | undefined>([
  ["regional-sales", demoResult],
  ["service-metrics", demoMetricResult]
]);
