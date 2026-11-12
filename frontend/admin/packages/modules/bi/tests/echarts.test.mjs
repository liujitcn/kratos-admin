import assert from "node:assert/strict";
import { existsSync } from "node:fs";
import { registerHooks } from "node:module";
import { fileURLToPath } from "node:url";
import { test } from "node:test";
import * as echarts from "echarts/core";
import { SVGRenderer } from "echarts/renderers";

// 使用真实引擎与 SVG SSR，仅替换挂载容器，避免测试依赖浏览器 Canvas。
const coreURL = import.meta.resolve("echarts/core");
const ssrCore = `export * from ${JSON.stringify(coreURL)};
import * as core from ${JSON.stringify(coreURL)};
export const init = () => core.init(null, null, {renderer: 'svg', ssr: true, width: 600, height: 400});`;
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "echarts/core" && context.parentURL?.endsWith("/engine/echarts.ts")) {
      return { url: `data:text/javascript,${encodeURIComponent(ssrCore)}`, shortCircuit: true };
    }
    if (specifier.startsWith(".")) {
      for (const suffix of [".ts", "/index.ts"]) {
        const url = new URL(specifier + suffix, context.parentURL);
        if (existsSync(fileURLToPath(url))) return nextResolve(url.href, context);
      }
    }
    return nextResolve(specifier, context);
  }
});
const { compileEcharts, createEchartsEngine } = await import("../src/engine/echarts.ts");
const { BI_THEME_LIGHT } = await import("../src/theme/index.ts");
const { createDimensionLabelFormatter } = await import("../src/dataset/labels.ts");
const { buildCardChart } = await import("../src/card-schema.ts");
const { dashboardResults, dashboardSnapshotCard } = await import("../src/bi-dashboard-schema.ts");
echarts.use(SVGRenderer);

test("图表更新移除取消的指标、缩放和旧坐标轴", () => {
  const engine = createEchartsEngine();
  const chart = { id: "test", type: "bar", encoding: { dimension: "city", measures: ["amount", "count"] }, style: { dataZoom: true } };
  const rows = [{ city: "A", amount: "10", count: "2" }];
  const handle = engine.mount(null, engine.compile(chart, rows, BI_THEME_LIGHT));
  try {
    handle.update(engine.compile({ ...chart, encoding: { dimension: "city", measure: "amount" }, style: { dataZoom: false } }, rows, BI_THEME_LIGHT));
    const option = handle.__echartsInstance.getOption();
    assert.equal(option.series.length, 1);
    assert.equal(option.dataZoom?.length ?? 0, 0);
    handle.update(engine.compile({ ...chart, type: "pie" }, rows, BI_THEME_LIGHT));
    assert.equal(handle.__echartsInstance.getOption().xAxis?.length ?? 0, 0);
  } finally {
    handle.dispose();
  }
});

test("双维度多指标卡片保留全部查询指标并对齐缺失类别", () => {
  const query = { dimensions: ["city", "status"], measures: [{ code: "amount" }, { code: "count" }] };
  const result = {
    columns: [
      { code: "city", type: "string" }, { code: "status", type: "string" },
      { code: "amount", type: "number" }, { code: "count", type: "number" }
    ],
    rows: [{ values: ["A", "paid", "1", "2"] }, { values: ["A", "pending", "30", "40"] }, { values: ["B", "paid", "5", "6"] }]
  };
  for (const type of ["bar", "line", "area"]) {
    for (const chart_json of ["", JSON.stringify({ encoding: { dimension: "city", series: "status", measure: "amount" }, style: {} })]) {
      const card = buildCardChart({ id: 1, chart_type: type, chart_json, query }, result);
      const option = compileEcharts(card.chart, card.rows, BI_THEME_LIGHT);
      assert.deepEqual(option.series.map(series => series.data), [[1, 5], [2, 6], [30, null], [40, null]], type);
      assert.equal(new Set(option.series.map(series => series.name)).size, 4);
    }
  }
});

test("发布仪表盘的HTTP空结果能渲染空态并保留缺失结果", () => {
  const placed = {
    card_id: 1, chart_type: "bar",
    query: { dimensions: ["city"], measures: [{ code: "count" }] }
  };
  const response = JSON.parse('{"cards":[{"item_id":"empty","result":{"columns":[{"code":"city","type":"string"},{"code":"count","type":"number"}],"query_record_id":7}},{"item_id":"missing"}]}');
  const snapshot = structuredClone(response);
  const results = dashboardResults(response);
  const card = buildCardChart(dashboardSnapshotCard(placed), results.get("empty"));
  assert.deepEqual(card.rows, []);
  assert.equal(results.get("empty").query_record_id, 7);
  assert.equal(results.get("missing"), undefined);
  assert.equal(dashboardResults({}).size, 0);
  assert.deepEqual(response, snapshot);
});

test("散点图将字符串数值映射到两个数值轴", () => {
  const option = compileEcharts(
    { id: "test", type: "scatter", encoding: { dimension: "x", measure: "y" } },
    [{ x: "1", y: "4" }, { x: "2", y: "5" }, { x: "100", y: "6" }],
    BI_THEME_LIGHT
  );
  assert.equal(option.xAxis.type, "value");
  assert.equal(option.yAxis.type, "value");
  assert.deepEqual(option.series[0].data, [[1, 4], [2, 5], [100, 6]]);
});

test("雷达图按唯一类别对齐不同顺序及缺失的分组值", () => {
  const option = compileEcharts(
    { id: "test", type: "radar", encoding: { dimension: "city", series: "status", measure: "amount" } },
    [{ city: "A", status: "paid", amount: "1" }, { city: "B", status: "pending", amount: "4" }, { city: "B", status: "paid", amount: "3" }],
    BI_THEME_LIGHT
  );
  assert.deepEqual(option.radar.indicator.map(item => item.name), ["A", "B"]);
  assert.deepEqual(option.series[0].data.map(item => item.value), [[1, 3], [null, 4]]);
});

test("日期标签按日、月和单日小时缩写，保留非日期分类与跨年原值", () => {
  const daily = createDimensionLabelFormatter(["2025-06-23 00:00:00", "2026-06-23 00:00:00"]);
  assert.equal(daily("2026-06-23 00:00:00"), "06/23");
  assert.equal(daily("2026-06-23T14:30:00+08:00"), "06/23");
  assert.equal(daily("淋浴设备"), "淋浴设备");
  assert.equal(daily(100), "100");
  assert.equal(daily("2026-06"), "06月");
  assert.equal(createDimensionLabelFormatter([], "consume_month")("2026-06-01 00:00:00"), "06月");
  assert.equal(createDimensionLabelFormatter([], "consume_hour")("2026-06-23 00:00:00"), "00时");
  const hourly = createDimensionLabelFormatter(["2026-06-23 00:00:00", "2026-06-23 14:00:00"]);
  assert.equal(hourly("2026-06-23 00:00:00"), "00时");
  assert.equal(hourly("2026-06-23 14:00:00"), "14时");
  assert.equal(createDimensionLabelFormatter(["2026-06-23 00:00:00"])("2026-06-23 00:00:00"), "06/23");
});

test("多日时间维度的同日记录使用带日期的小时标签，避免轴标签重复", () => {
  const format = createDimensionLabelFormatter([
    "2026-11-10 08:00:00",
    "2026-11-10 14:00:00",
    "2026-11-11 09:00:00"
  ]);
  assert.equal(format("2026-11-10 08:00:00"), "11/10 08时");
  assert.equal(format("2026-11-10 14:00:00"), "11/10 14时");
  assert.equal(format("2026-11-11 09:00:00"), "11/11 09时");
});

test("日期趋势按原始时间排序，短标签不合并跨年分类、改变数值或覆盖显式排序", () => {
  const rows = [
    { date: "2026-06-23 00:00:00", value: "3" },
    { date: "2025-06-23 00:00:00", value: "1" },
    { date: "2026-06-22 00:00:00", value: "2" }
  ];
  const snapshot = structuredClone(rows);
  for (const type of ["bar", "line", "area"]) {
    const chart = { id: "test", type, encoding: { dimension: "date", measure: "value" }, style: { dataZoom: true } };
    const option = compileEcharts(chart, rows, BI_THEME_LIGHT);
    assert.deepEqual(option.xAxis.data, type === "bar" ? rows.map(row => row.date) : [rows[1].date, rows[2].date, rows[0].date]);
    assert.deepEqual(option.series[0].data, type === "bar" ? [3, 1, 2] : [1, 2, 3]);
    assert.equal(option.xAxis.axisLabel.formatter(rows[0].date), "06/23");
    assert.equal(option.dataZoom[1].labelFormatter(0, rows[0].date), "06/23");
    const ordered = compileEcharts({ ...chart, sort: { by: "measure", order: "desc" } }, rows, BI_THEME_LIGHT);
    assert.deepEqual(ordered.xAxis.data, rows.map(row => row.date));
  }
  assert.deepEqual(rows, snapshot);
});

test("真实引擎渲染日、月、小时短标签，保留完整日期供提示使用", () => {
  const engine = createEchartsEngine();
  for (const [dates, labels] of [
    [["2026-06-23 00:00:00", "2026-06-24 00:00:00"], ["06/23", "06/24"]],
    [["2026-06", "2026-07"], ["06月", "07月"]],
    [["2026-06-23 00:00:00", "2026-06-23 14:00:00"], ["00时", "14时"]]
  ]) {
    const chart = { id: "dates", type: "line", encoding: { dimension: "date", measure: "value" } };
    const handle = engine.mount(null, engine.compile(chart, dates.map((date, index) => ({ date, value: index + 1 })), BI_THEME_LIGHT));
    try {
      const svg = handle.__echartsInstance.renderToSVGString();
      for (const label of labels) assert.ok(svg.includes(label), label);
      assert.deepEqual(handle.__echartsInstance.getOption().xAxis[0].data, dates);
    } finally {
      handle.dispose();
    }
  }
  const option = compileEcharts(
    { id: "legend", type: "bar", encoding: { dimension: "name", series: "date", measures: ["value", "count"] } },
    [{ name: "A", date: "2026-06-23 00:00:00", value: 1, count: 2 }],
    BI_THEME_LIGHT
  );
  assert.equal(option.legend.formatter("2026-06-23 00:00:00 · value"), "06/23 · value");
});

test("所有分类图型共用日期短标签，提示数据及桑基节点关联保留完整日期", () => {
  const date = "2026-06-23 00:00:00";
  const rows = [{ date, group: "2026-07-01 00:00:00", value: "10" }];
  for (const type of ["pie", "donut", "radar", "funnel", "treemap", "sunburst", "sankey", "heatmap"]) {
    const option = compileEcharts({ id: "test", type, encoding: { dimension: "date", series: "group", measure: "value" } }, rows, BI_THEME_LIGHT);
    if (["pie", "donut", "funnel", "treemap"].includes(type)) {
      assert.equal(option.series[0].label.formatter({ name: date, value: 10 }), "06/23", type);
      assert.equal(option.series[0].data[0].name, date, type);
    } else if (type === "radar") {
      assert.equal(option.radar.axisName.formatter(date), "06/23");
      assert.equal(option.radar.indicator[0].name, date);
    } else if (type === "heatmap") {
      assert.equal(option.xAxis.axisLabel.formatter(date), "06/23");
      assert.equal(option.yAxis.axisLabel.formatter(rows[0].group), "07/01");
      assert.equal(option.yAxis.data[0], rows[0].group);
    } else {
      assert.equal(option.series[0].label.formatter({ name: date }), "06/23", type);
      if (type === "sankey") assert.equal(option.series[0].links[0].source, date);
      else assert.equal(option.series[0].data[0].children[0].name, date);
    }
  }
});
