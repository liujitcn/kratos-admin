import assert from "node:assert/strict";
import { test } from "node:test";
import { normalizeBiCard, normalizeBiWorkbench, normalizeBiDashboardPage, normalizeBiQueryResult } from "../src/bi-response.ts";
import { buildCardChart } from "../src/card-schema.ts";

test("HTTP省略空集合时，新账号与清空后的工作台仍能渲染", () => {
  assert.deepEqual(normalizeBiWorkbench(JSON.parse("{}")), { dashboard_ids: [], favorite_ids: [], dashboards: [] });
  const result = normalizeBiQueryResult(JSON.parse("{}"));
  assert.deepEqual(result, { columns: [], rows: [], truncated: false, elapsed_ms: 0, query_record_id: 0 });
  const card = normalizeBiCard({ id: 1, chart_type: "metric", query: { source_code: "test" } });
  assert.equal(buildCardChart(card, result).metrics[0].value, "—");
  assert.deepEqual(card.query.dimensions, []);
});

test("HTTP省略仪表盘分页集合时仍可安全遍历", () => {
  assert.deepEqual(normalizeBiDashboardPage({}), { dashboards: [], total: 0 });
});

test("响应补全保留原查询与结果，不改写输入或数值精度", () => {
  const input = { dashboards: [{ id: 2, name: "仪表盘" }], dashboard_ids: [2] };
  const before = structuredClone(input);
  const actual = normalizeBiWorkbench(input);
  assert.deepEqual(actual.dashboard_ids, [2]);
  assert.equal(actual.dashboards[0].name, "仪表盘");
  const card = normalizeBiCard({ id: 2, query: { source_code: "test", filters: [{ field: "status", operator: "is_null" }] } });
  assert.deepEqual(card.query.filters[0].values, []);
  assert.deepEqual(input, before);
  assert.equal(
    normalizeBiQueryResult({ columns: [], rows: [{ values: ["9007199254740993"] }] }).rows[0].values[0],
    "9007199254740993"
  );
});

test("发布快照中的空分组为null时，指标卡仍能渲染真实结果", () => {
  const snapshot = { id: 2, chart_type: "metric", query: { source_code: "test", dimensions: null, measures: [{ code: "count" }] } };
  const card = normalizeBiCard(snapshot);
  const result = normalizeBiQueryResult({ columns: [{ code: "count", type: "number" }], rows: [{ values: ["5"] }] });
  assert.equal(buildCardChart(card, result).metrics[0].value, "5");
  assert.deepEqual(card.query.dimensions, []);
  assert.equal(snapshot.query.dimensions, null);
});
