import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { test } from "node:test";
import { fileURLToPath } from "node:url";
import { runInNewContext } from "node:vm";
import ts from "typescript";
import * as vue from "vue";
import { buildCardChart, parseCardChart } from "../src/card-schema.ts";
import * as biResponse from "../src/bi-response.ts";

const require = createRequire(import.meta.url);
const { parse, compileScript } = require("vue/compiler-sfc");

function setupComponent(name, props, mocks, emit = () => undefined) {
  const filename = fileURLToPath(new URL(`../src/components/${name}.vue`, import.meta.url));
  const { descriptor } = parse(readFileSync(filename, "utf8"), { filename });
  const script = compileScript(descriptor, { id: name, genDefaultAs: "component" });
  const module = { exports: {} };
  runInNewContext(
    ts.transpileModule(`${script.content}\nmodule.exports = component;`, {
      compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 }
    }).outputText,
    {
      module,
      exports: module.exports,
      Error,
      require: name => {
        if (name.endsWith(".vue")) return {};
        assert.ok(Object.hasOwn(mocks, name), `未配置测试依赖: ${name}`);
        return mocks[name];
      }
    }
  );
  return module.exports.setup(props, { expose: () => undefined, emit });
}

test("工具响应省略空维度时，建卡弹窗按指标卡打开并保留来源记录", () => {
  let open;
  const query = { source_code: "test", measures: [{ code: "count", aggregation: "count" }], limit: 100 };
  const state = setupComponent(
    "BiCardFormDialog",
    {
      modelValue: false,
      sources: [],
      initialQuery: query,
      initialResult: { query_record_id: 7 },
      initialTitle: "用户总数"
    },
    {
      vue: {
        ...vue,
        watch: (_source, callback) => {
          open ??= callback;
        }
      },
      "@liujitcn/kratos-admin-core": { t: key => key },
      "../api/bi/admin/v1/bi_card": {},
      "../api/bi/admin/v1/bi_query": {},
      "../card-schema": { parseCardChart },
      "../bi-response": biResponse
    }
  );
  assert.doesNotThrow(() => open(true));
  assert.equal(state.form.chart_type, "metric");
  assert.equal(state.form.name, "用户总数");
  assert.equal(state.sourceRecordID.value, 7);
  assert.deepEqual(JSON.parse(JSON.stringify(state.baseQuery.value)), {
    ...query,
    dimensions: [],
    filters: [],
    orders: []
  });
  assert.equal(query.dimensions, undefined);
});

test("助手建卡读取来源记录，并补齐工具结果中省略的集合", async () => {
  let open;
  const guide = { query_record_id: 7, query: { source_code: "test", measures: [{ code: "count", aggregation: "count" }] } };
  const state = setupComponent(
    "../views/query/components/BiCardGuideDialog",
    { modelValue: false, guide },
    {
      vue: {
        ...vue,
        watch: (_source, callback) => {
          open = callback;
        }
      },
      "../../../api/bi/admin/v1/bi_query": {
        defBiQueryService: { ListBiSources: async () => ({ sources: [] }) }
      },
      "../../../api/bi/admin/v1/bi_query_record": {
        defBiQueryRecordService: {
          GetBiQuery: async request => {
            assert.equal(request.id, 7);
            return { result: { query_record_id: 7 } };
          }
        }
      },
      "../../../bi-response": biResponse
    }
  );
  await open([true, guide], undefined, () => undefined);
  assert.equal(state.visible.value, true);
  assert.deepEqual(JSON.parse(JSON.stringify(state.result.value)), {
    query_record_id: 7,
    columns: [],
    rows: [],
    truncated: false,
    elapsed_ms: 0
  });
});

test("关闭助手建卡后，迟到的来源响应不再打开弹窗", async () => {
  let open, cancel;
  const pending = Promise.withResolvers();
  const guide = { query_record_id: 7 };
  const state = setupComponent(
    "../views/query/components/BiCardGuideDialog",
    { modelValue: false, guide },
    {
      vue: {
        ...vue,
        watch: (_source, callback) => {
          open = callback;
        }
      },
      "../../../api/bi/admin/v1/bi_query": {
        defBiQueryService: { ListBiSources: async () => ({ sources: [] }) }
      },
      "../../../api/bi/admin/v1/bi_query_record": {
        defBiQueryRecordService: { GetBiQuery: () => pending.promise }
      },
      "../../../bi-response": biResponse
    }
  );
  const opening = open([true, guide], undefined, callback => {
    cancel = callback;
  });
  await Promise.resolve();
  cancel();
  await open([false, guide], undefined, () => undefined);
  pending.resolve({ result: { query_record_id: 7 } });
  await opening;
  assert.equal(state.visible.value, false);
  assert.equal(state.result.value, undefined);
});

test("助手建卡来源加载失败时同步关闭外层状态", async () => {
  let open;
  const events = [];
  const guide = { query_record_id: 7 };
  const state = setupComponent(
    "../views/query/components/BiCardGuideDialog",
    { modelValue: false, guide },
    {
      vue: {
        ...vue,
        watch: (_source, callback) => {
          open = callback;
        }
      },
      "../../../api/bi/admin/v1/bi_query": {
        defBiQueryService: {
          ListBiSources: async () => {
            throw new Error("offline");
          }
        }
      },
      "../../../api/bi/admin/v1/bi_query_record": {},
      "../../../bi-response": biResponse
    },
    (...event) => events.push(event)
  );
  await open([true, guide], undefined, () => undefined);
  assert.equal(state.visible.value, false);
  assert.deepEqual(events, [["update:modelValue", false]]);
});

test("缩略图可见后才按发布版本取数，错误可重试，卸载后不回填", async () => {
  let intersect,
    unmount,
    stopped = 0,
    calls = 0;
  const pending = Promise.withResolvers();
  const dashboard = { id: 7, published_version: 3 };
  const schema = { schemaVersion: 3, name: "published", cards: [{ id: "first" }, { id: "second" }, { id: "third" }] };
  const response = { schema_json: JSON.stringify(schema), cards: [{ item_id: "first", result: { rows: [{ values: ["42"] }] } }] };
  const state = setupComponent(
    "BiWorkbenchPreview",
    { dashboard },
    {
      vue: {
        ...vue,
        onBeforeUnmount: callback => {
          unmount = callback;
        }
      },
      "@vueuse/core": {
        useIntersectionObserver: (_host, callback) => {
          intersect = callback;
          return {
            stop: () => {
              stopped++;
            }
          };
        }
      },
      "@liujitcn/kratos-admin-core": { t: key => key },
      "../bi-dashboard-schema": {
        parseDashboardLayout: JSON.parse,
        dashboardResults: data => new Map(data.cards.map(item => [item.item_id, item.result]))
      },
      "../api/bi/admin/v1/bi_dashboard": {
        defBiDashboardService: {
          GetBiDashboardData: request => {
            assert.equal(request.id, 7);
            assert.equal(request.version, 3);
            calls++;
            if (calls === 1) return Promise.reject(new Error("offline"));
            return calls === 2 ? Promise.resolve(response) : pending.promise;
          }
        }
      }
    }
  );
  assert.equal(calls, 0);
  intersect([{ isIntersecting: false }]);
  assert.equal(calls, 0);
  intersect([{ isIntersecting: true }]);
  await new Promise(resolve => setImmediate(resolve));
  assert.equal(stopped, 1);
  assert.equal(state.loading.value, false);
  assert.equal(state.error.value, "offline");
  await state.load();
  assert.equal(state.error.value, "");
  assert.equal(state.layout.value.cards.length, 3);
  assert.equal(state.results.value.get("first").rows[0].values[0], "42");
  const previous = state.layout.value;
  const loading = state.load();
  unmount();
  pending.resolve({ schema_json: JSON.stringify({ ...schema, name: "late" }), cards: [] });
  await loading;
  assert.equal(state.layout.value, previous);
});

test("缩略图保留真实数据和图表类型，隐藏密集控件且不改变完整卡片", () => {
  const result = {
    columns: [
      { code: "day", type: "string" },
      { code: "amount", type: "number" }
    ],
    rows: [{ values: ["Monday", "42"] }, { values: ["Tuesday", "81"] }]
  };
  for (const chart_type of ["bar", "line", "area", "scatter", "heatmap", "pie", "donut", "metric"]) {
    const card = {
      id: 1,
      chart_type,
      chart_json: JSON.stringify({ encoding: {}, style: { showLegend: true, showLabel: true, dataZoom: true } }),
      query: { dimensions: ["day"], measures: [{ code: "amount" }] }
    };
    const mocks = {
      vue,
      "@liujitcn/kratos-admin-core": { t: key => key },
      "../card-schema": { buildCardChart },
      "../env": { useBiTheme: () => vue.ref({}) }
    };
    const original = setupComponent("BiCardVisualization", { card, result }, mocks).data.value;
    const preview = setupComponent("BiCardVisualization", { card, result, compact: true }, mocks).data.value;
    assert.deepEqual(preview.rows, original.rows);
    assert.deepEqual(preview.metrics, original.metrics);
    assert.equal(preview.chart.type, chart_type);
    assert.equal(preview.chart.style.showLegend, false);
    assert.equal(preview.chart.style.dataZoom, false);
    assert.equal(original.chart.options, undefined);
    assert.equal(original.chart.style.showLegend, true);
    assert.equal(original.chart.style.dataZoom, true);
    if (["bar", "line", "area", "scatter", "heatmap"].includes(chart_type)) {
      assert.equal(preview.chart.options.xAxis.show, false);
      assert.equal(preview.chart.options.yAxis.show, false);
    }
  }
});
