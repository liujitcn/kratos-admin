import assert from "node:assert/strict";
import { test } from "node:test";
import {
  buildCardChart,
  orderWorkbenchDashboards,
  supportsCardPeriod,
  cardRefreshDelay,
  formatCardMetric
} from "../src/card-schema.ts";

test("卡片原样使用服务端聚合结果，指标卡保留大整数和金额精度", () => {
  const result = { columns: [{ code: "amount", type: "number" }], rows: [{ values: ["9007199254740993"] }] };
  const card = { id: 1, chart_type: "metric", query: { dimensions: [], measures: [{ code: "amount" }] } };
  const snapshot = structuredClone(result);
  const actual = buildCardChart(card, result);
  assert.equal(actual.metrics[0].value, "9007199254740993");
  assert.equal(actual.metricOnly, true);
  assert.equal(actual.chart.aggregation, "none");
  assert.deepEqual(result, snapshot);
  assert.equal(buildCardChart(card, { ...result, rows: [{ values: ["128460.30"] }] }).metrics[0].value, "128460.30");
});

test("无分组查询自动展示指标，空数据和缺失值不会显示NaN", () => {
  const card = { id: 1, chart_type: "metric", query: { dimensions: [], measures: [{ code: "count" }] } };
  const actual = buildCardChart(card, { columns: [{ code: "count", type: "number" }], rows: [] });
  assert.equal(actual.metricOnly, true);
  assert.equal(actual.metrics[0].value, "—");
  assert.deepEqual(actual.rows, []);
});

test("按个人配置顺序组装仪表盘入口，忽略已撤回或删除的仪表盘", () => {
  const cards = [{ id: 1 }, { id: 2 }, { id: 3 }];
  assert.deepEqual(orderWorkbenchDashboards([3, 99, 1], cards), [cards[2], cards[0]]);
  assert.deepEqual(orderWorkbenchDashboards([], cards), []);
});

test("多个聚合指标不会把第二个指标误认为分组维度", () => {
  const card = { id: 1, chart_type: "metric", query: { dimensions: [], measures: [{ code: "amount" }, { code: "count" }] } };
  const actual = buildCardChart(card, {
    columns: [
      { code: "amount", type: "number" },
      { code: "count", type: "number" }
    ],
    rows: [{ values: ["128460.30", "600"] }]
  });
  assert.equal(actual.metricOnly, true);
  assert.equal(actual.chart.encoding.dimension, undefined);
  assert.equal(actual.metrics[0].value, "128460.30");
});

test("双维度卡片显式选择单个指标时不补回其他指标", () => {
  const actual = buildCardChart({
    id: 1, chart_type: "bar",
    query: { dimensions: ["city", "status"], measures: [{ code: "amount" }, { code: "count" }] },
    chart_json: JSON.stringify({ encoding: { measures: ["count"] }, style: {} })
  }, {
    columns: [{ code: "city", type: "string" }, { code: "status", type: "string" }, { code: "amount", type: "number" }, { code: "count", type: "number" }],
    rows: [{ values: ["A", "paid", "10", "2"] }]
  });
  assert.deepEqual(actual.chart.encoding.measures, ["count"]);
  assert.deepEqual(actual.metrics.map(item => item.code), ["count"]);
});

test("卡片仅在日期字段可过滤且参与查询时支持周期，无日期分组的时间范围也适用", () => {
  const source = {
    dimensions: [
      { code: "paid_at", type: "date", filterable: true },
      { code: "created_at", type: "date", filterable: false },
      { code: "status", type: "string", filterable: true }
    ]
  };
  assert.equal(supportsCardPeriod({ query: { dimensions: ["paid_at"] } }, source), true);
  assert.equal(supportsCardPeriod({ query: { dimensions: [], time_range: { field: "paid_at" } } }, source), true);
  for (const dimensions of [[], ["status"], ["created_at"]]) {
    assert.equal(supportsCardPeriod({ query: { dimensions } }, source), false);
  }
  assert.equal(supportsCardPeriod({ query: { dimensions: ["paid_at"] } }), false);
  assert.equal(supportsCardPeriod({ query: { dimensions: ["paid_at"], time_range: { field: "missing" } } }, source), false);
});

test("多个日期分组必须指定时间字段，避免自动使用第一个日期", () => {
  const source = {
    dimensions: [
      { code: "created_at", type: "date", filterable: true },
      { code: "paid_at", type: "date", filterable: true }
    ]
  };
  const query = { dimensions: ["created_at", "paid_at"] };
  assert.equal(supportsCardPeriod({ query }, source), false);
  assert.equal(supportsCardPeriod({ query: { ...query, time_range: { field: "paid_at" } } }, source), true);
});

test("自动刷新按间隔或下一次本地02:00执行，跨月后仍保持每日时点", () => {
  assert.equal(cardRefreshDelay(300, new Date(2026, 10, 4, 1, 30)), 300_000);
  assert.equal(cardRefreshDelay(86400, new Date(2026, 10, 4, 1, 30)), 30 * 60_000);
  assert.equal(cardRefreshDelay(86400, new Date(2026, 0, 31, 3)), 23 * 60 * 60_000);
});

test("卡片数字显示保留大整数、小数末尾零和科学记数原值", () => {
  assert.equal(formatCardMetric("9007199254740993"), "9,007,199,254,740,993");
  assert.equal(formatCardMetric("128460.30"), "128,460.30");
  assert.equal(formatCardMetric("-128460.000"), "-128,460.000");
  assert.equal(formatCardMetric("1e20"), "1e20");
});

test("完整展示配置用于桑基、热力、旭日与散点，不丢失第二字段或样式", () => {
  const result = {
    columns: [
      { code: "source", type: "string" },
      { code: "target", type: "string" },
      { code: "value", type: "number" },
      { code: "other", type: "number" }
    ],
    rows: [{ values: ["A", "B", "10", "20"] }]
  };
  for (const type of ["sankey", "heatmap", "sunburst", "scatter"]) {
    const encoding = {
      dimension: type === "scatter" ? "value" : "source",
      series: type === "scatter" ? undefined : "target",
      measure: type === "scatter" ? "other" : "value"
    };
    const actual = buildCardChart(
      {
        id: 2,
        chart_type: type,
        query: { dimensions: ["source", "target"], measures: [{ code: "value" }, { code: "other" }] },
        chart_json: JSON.stringify({ encoding, style: { showLegend: false, showLabel: true } })
      },
      result
    );
    assert.equal(actual.chart.encoding.dimension, encoding.dimension);
    assert.equal(actual.chart.encoding.series, encoding.series);
    assert.equal(actual.chart.encoding.measure, encoding.measure);
    assert.equal(actual.chart.style.showLegend, false);
    assert.equal(actual.chart.style.showLabel, true);
    assert.equal(actual.rows[0].value, "10");
  }
});
