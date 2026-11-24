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

test("紧凑卡片保留部门分类和数值提示，不改变放大图配置", () => {
  const query = { dimensions: ["department"], measures: [{ code: "count" }] };
  const result = {
    columns: [{ code: "department", type: "string" }, { code: "count", name: "人员数量", unit: "人", type: "number" }],
    rows: [{ values: ["A部门", "1200"] }, { values: ["B部门", "300"] }]
  };
  const chart_json = JSON.stringify({ encoding: { measures: ["count"] }, style: { showLegend: false, showLabel: true, dataZoom: true } });
  const engine = createEchartsEngine();
  for (const chart_type of ["pie", "donut", "bar", "line", "area"]) {
    const card = { id: 1, chart_type, chart_json, query };
    const compact = buildCardChart(card, result, true);
    const option = engine.compile(compact.chart, compact.rows, BI_THEME_LIGHT);
    const circular = chart_type === "pie" || chart_type === "donut";
    assert.notEqual(option.tooltip.show, false);
    assert.equal(option.tooltip.trigger, circular ? "item" : "axis");
    assert.equal(option.series[0].name, circular ? "人员数量（人）" : "count");
    if (circular) {
      assert.equal(option.legend.type, "scroll");
      assert.deepEqual(option.series[0].data, [{ name: "A部门", value: 1200 }, { name: "B部门", value: 300 }]);
    } else {
      assert.deepEqual(option.xAxis.data, ["A部门", "B部门"]);
      assert.notEqual(option.xAxis.show, false);
      assert.equal(option.dataZoom, undefined);
      assert.match(option.tooltip.formatter([{ name: "A部门", seriesName: "count", value: 1200 }]), /人员数量（人）: 1,200/);
    }
    const instance = echarts.init(null, null, { renderer: "svg", ssr: true, width: 300, height: 116 });
    try {
      instance.setOption(option);
      const svg = instance.renderToSVGString();
      assert.match(svg, /A部门/);
      assert.match(svg, /B部门/);
    } finally {
      instance.dispose();
    }
    const full = buildCardChart(card, result);
    assert.equal(full.chart.style.showLegend, false);
    assert.equal(full.chart.style.showLabel, true);
    assert.equal(full.chart.style.dataZoom, true);
    assert.equal(full.chart.compact, undefined);
  }
});

test("紧凑图表中同名同单位的指标仍可独立切换", () => {
  const card = { id: 1, chart_type: "bar", query: { dimensions: ["department"], measures: [{ code: "a" }, { code: "b" }] } };
  const result = {
    columns: [
      { code: "department", type: "string" },
      { code: "a", name: "金额", unit: "元", type: "number" },
      { code: "b", name: "金额", unit: "元", type: "number" }
    ],
    rows: [{ values: ["A<部门>", "12", "34"] }]
  };
  const { chart, rows } = buildCardChart(card, result, true);
  const engine = createEchartsEngine();
  const option = engine.compile(chart, rows, BI_THEME_LIGHT);
  const names = option.series.map(series => series.name);
  assert.deepEqual(names, ["a", "b"]);
  assert.deepEqual(names.map(option.legend.formatter), ["金额（元）", "金额（元）"]);
  assert.equal(option.tooltip.formatter([{ name: "A<部门>", seriesName: "a", value: 12 }]), "A&lt;部门&gt;<br>金额（元）: 12");
  const handle = engine.mount(null, option);
  try {
    const instance = handle.__echartsInstance;
    instance.dispatchAction({ type: "legendUnSelect", name: names[0] });
    assert.deepEqual(instance.getModel().getCurrentSeriesIndices(), [1]);
  } finally {
    handle.dispose();
  }
});

test("工作台缩略图不展示不可操作的图例、仪表内部标题及重叠刻度", () => {
  createEchartsEngine();
  const chart = { id: "thumbnail", compact: true, encoding: { dimension: "name", measure: "value" }, measureLabels: { value: "人员数量（人）" } };
  const rows = Array.from({ length: 8 }, (_, index) => ({ name: `部门 ${index + 1}`, value: 42 }));
  for (const type of ["pie", "donut", "gauge"]) {
    const compact = compileEcharts({ ...chart, type }, rows, BI_THEME_LIGHT);
    const thumbnail = compileEcharts({ ...chart, type, thumbnail: true }, rows, BI_THEME_LIGHT);
    assert.equal(thumbnail.tooltip.show, false);
    if (type === "gauge") {
      assert.equal(compact.series[0].title.show, true);
      assert.equal(thumbnail.series[0].title.show, false);
      assert.equal(thumbnail.series[0].axisLabel.show, false);
      assert.equal(compact.series[0].axisLabel.show, true);
    } else {
      assert.equal(compact.legend.type, "scroll");
      assert.equal(thumbnail.legend, undefined);
      assert.equal(thumbnail.series[0].bottom, undefined);
      assert.equal(thumbnail.series[0].data.length, 8);
    }
    const instance = echarts.init(null, null, { renderer: "svg", ssr: true, width: 154, height: 70 });
    try {
      instance.setOption(thumbnail);
      const svg = instance.renderToSVGString();
      assert.doesNotMatch(svg, /1\/8|人员数量/);
      if (type === "gauge") {
        assert.match(svg, />42</);
        assert.doesNotMatch(svg, />100</);
      }
    } finally {
      instance.dispose();
    }
  }
});

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

test("月度部门卡片修复重复的轴字段，保留同名部门的独立系列", () => {
  const query = {
    dimensions: ["department", "created_at"],
    measures: [{ code: "count" }],
    grouping_options: [{ field: "created_at", time_grain: "month" }, { field: "department", hierarchy_level: 2 }]
  };
  const result = {
    columns: [{ code: "department", type: "string" }, { code: "created_at", type: "date" }, { code: "count", type: "number" }],
    rows: [
      { values: ["研发 / 平台 (#1)", "2026-01-01", "2"] },
      { values: ["运营 / 平台 (#2)", "2026-01-01", "7"] },
      { values: ["研发 / 平台 (#1)", "2026-02-01", "3"] }
    ]
  };
  for (const chart_json of ["", JSON.stringify({ encoding: { dimension: "created_at", series: "created_at" } })]) {
    const card = buildCardChart({ id: 1, chart_type: "line", query, chart_json }, result);
    const option = compileEcharts(card.chart, card.rows, BI_THEME_LIGHT);
    assert.deepEqual(option.xAxis.data, ["2026-01-01", "2026-02-01"]);
    assert.deepEqual(option.series.map(item => item.name), ["研发 / 平台 (#1)", "运营 / 平台 (#2)"]);
    assert.deepEqual(option.series.map(item => item.data), [[2, 3], [7, null]]);
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
