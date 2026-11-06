import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { test } from "node:test";
import { fileURLToPath } from "node:url";
import { runInNewContext } from "node:vm";
import ts from "typescript";
import * as vue from "vue";
import { flowPlace } from "../src/schema/layout.ts";
import { orderWorkbenchDashboards } from "../src/card-schema.ts";
import * as response from "../src/bi-response.ts";

const require = createRequire(import.meta.url);
const { parse, compileScript } = require("vue/compiler-sfc");

/** 运行实际布局模块与组件脚本，外部接口只在测试边界替换。 */
function load(relativePath, mocks = {}, globals = {}) {
  const filename = fileURLToPath(new URL(relativePath, import.meta.url));
  let source = readFileSync(filename, "utf8");
  if (filename.endsWith(".vue")) {
    const { descriptor } = parse(source, { filename });
    source =
      compileScript(descriptor, { id: "dashboard-test", genDefaultAs: "component" }).content + "\nmodule.exports = component;";
  }
  const module = { exports: {} };
  runInNewContext(
    ts.transpileModule(source, {
      compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 }
    }).outputText,
    {
      module,
      exports: module.exports,
      Error,
      ...Object.fromEntries(
        ["Plus", "Check", "View", "Grid", "EditPen", "Promotion", "Setting", "Delete"].map(name => [
          name,
          {}
        ])
      ),
      ElMessage: { error: () => undefined },
      ...globals,
      require: name => {
        if (name.endsWith(".vue")) return {};
        assert.ok(Object.hasOwn(mocks, name), `未配置测试依赖: ${name}`);
        return mocks[name];
      }
    }
  );
  return module.exports;
}

const schema = load("../src/bi-dashboard-schema.ts", { "./schema/layout": { flowPlace }, "./bi-response": response });

test("仪表盘布局不再持久化固定主题，保留卡片位置和发布快照", () => {
  const cards = [{ id: "card_1", card_id: 1, w: 6, h: 4, x: 0, y: 0, query: { source_code: "sales" } }];
  const layout = schema.parseDashboardLayout(JSON.stringify({ schemaVersion: 3, name: "销售", theme: { key: "dark" }, cards }));
  assert.equal(layout.name, "销售");
  assert.equal(Object.hasOwn(layout, "theme"), false);
  assert.equal(JSON.stringify(layout.cards), JSON.stringify(cards));
});

test("卡片库禁止重复和超限加入，尺寸调整与移除只修改对应引用", () => {
  const props = vue.reactive({ modelValue: { schemaVersion: 3, name: "销售", cards: [] }, cards: [] });
  const component = load("../src/components/BiDesigner.vue", {
    vue,
    "@liujitcn/kratos-admin-core": { t: key => key },
    "../bi-dashboard-schema": schema,
    "../env": { useBiI18n: () => ({ t: key => key }) }
  });
  const state = component.setup(props, {
    expose: () => undefined,
    emit: (_, value) => {
      props.modelValue = value;
    }
  });
  state.add({ id: 1, chart_type: "metric" });
  props.modelValue.cards[0].query = { source_code: "sales", dimensions: ["city"] };
  state.add({ id: 1, chart_type: "metric" });
  assert.equal(props.modelValue.cards.length, 1);
  assert.equal(props.modelValue.cards[0].w, 3);
  state.add({ id: 2, chart_type: "bar" });
  state.resize("card_1", { w: 12, h: 5 });
  assert.equal(props.modelValue.cards[0].h, 5);
  assert.equal(props.modelValue.cards[1].w, 6);
  assert.equal(props.modelValue.cards[1].y, 5);
  assert.equal(props.modelValue.cards[0].query.source_code, "sales");
  state.move(1, 0);
  assert.equal(props.modelValue.cards[0].card_id, 2);
  state.remove("card_1");
  assert.equal(props.modelValue.cards.length, 1);
  assert.equal(state.addedCardIDs.value.has(1), false);
  for (let id = 3; id <= 22; id++) state.add({ id, chart_type: "bar" });
  assert.equal(props.modelValue.cards.length, 20);
  assert.equal(props.modelValue.name, "销售");
});

test("列表编辑同步名称与布局，保留版本和卡片，失败时保留表单", async () => {
  const saved = [];
  let fail = true;
  const component = load("../src/views/manage/index.vue", {
    vue,
    "vue-router": { useRouter: () => ({ push: () => assert.fail("编辑信息不应跳转设计器") }) },
    "@liujitcn/kratos-admin-core": { t: key => key },
    "@liujitcn/kratos-admin-core/table": {},
    "@liujitcn/kratos-admin-core/components/ProTable": {},
    "@liujitcn/kratos-admin-core/stores/runtime": { useAuthStore: () => ({ flatMenuListGet: [] }) },
    "../../bi-dashboard-schema": schema,
    "../../api/bi/admin/v1/bi_card": {},
    "../../api/bi/admin/v1/bi_dashboard": {
      defBiDashboardService: {
        SaveBiDashboard: async value => {
          if (fail) throw new Error("保存失败");
          saved.push(value.dashboard);
        }
      }
    }
  });
  const state = component.setup({}, { expose: () => undefined });
  state.dialog.value = { validate: async () => true, clearValidate: () => undefined };
  const original = {
    id: 5,
    name: "原名称",
    code: "sales",
    description: "说明",
    draft_version: 3,
    published_version: 2,
    schema_json: JSON.stringify({ schemaVersion: 3, name: "原名称", cards: [{ id: "card_1", card_id: 1, w: 6, h: 4 }] })
  };
  state.edit(original);
  state.form.name = "新名称";
  await state.submit();
  assert.equal(state.creating.value, true);
  assert.equal(state.form.name, "新名称");
  fail = false;
  await state.submit();
  assert.equal(state.creating.value, false);
  assert.equal(saved[0].name, "新名称");
  assert.equal(saved[0].draft_version, 3);
  assert.equal(saved[0].published_version, 2);
  assert.equal(JSON.parse(saved[0].schema_json).name, "新名称");
  assert.equal(JSON.parse(saved[0].schema_json).cards[0].card_id, 1);
  assert.equal(original.name, "原名称");
});

test("工作台排序自动保存并保留最新卡片收藏，重复操作被阻止，失败恢复原顺序", async () => {
  const dashboards = [{ id: 1 }, { id: 2 }, { id: 3 }];
  const errors = [];
  const requests = [];
  let favoriteIDs = [8],
    fail = false;
  const pending = Promise.withResolvers();
  const component = load(
    "../src/views/workbench/index.vue",
    {
      vue: { ...vue, onMounted: () => undefined, onActivated: () => undefined },
      "vue-router": { useRouter: () => ({}) },
      "@liujitcn/kratos-admin-core": { t: key => key },
      "../../card-schema": { orderWorkbenchDashboards },
      "../../bi-dashboard-schema": schema,
      "../../api/bi/admin/v1/bi_card": {
        defBiCardService: {
          GetBiWorkbench: async () => ({ dashboard_ids: [3, 1, 2], favorite_ids: favoriteIDs, dashboards }),
          SaveBiWorkbench: async request => {
            requests.push(request);
            if (fail) throw new Error("保存失败");
            await pending.promise;
            return { ...request, dashboards };
          }
        }
      }
    },
    { ElMessage: { success: () => undefined, error: message => errors.push(message) } }
  );
  const state = component.setup({}, { expose: () => undefined });
  await state.load();
  assert.deepEqual(
    Array.from(state.dashboards.value, item => item.id),
    [3, 1, 2]
  );
  await state.move(-1, 0);
  await state.move(0, 3);
  await state.move(1, 1);
  assert.equal(requests.length, 0);
  favoriteIDs = [8, 9];
  const saving = state.move(0, 2);
  assert.equal(state.saving.value, true);
  assert.deepEqual(
    Array.from(state.dashboards.value, item => item.id),
    [1, 2, 3]
  );
  await state.move(2, 0);
  assert.equal(requests.length, 1);
  assert.deepEqual(Array.from(requests[0].favorite_ids), [8, 9]);
  pending.resolve();
  await saving;
  assert.equal(state.saving.value, false);
  assert.deepEqual(
    Array.from(state.dashboards.value, item => item.id),
    [1, 2, 3]
  );
  fail = true;
  await state.move(2, 0);
  assert.equal(state.saving.value, false);
  assert.deepEqual(
    Array.from(state.dashboards.value, item => item.id),
    [1, 2, 3]
  );
  assert.deepEqual(errors, ["保存失败"]);
});

test("管理页收藏开关成功后才切换，保留顺序与卡片收藏，草稿和失败不改状态", async () => {
  const item = { id: 3, published_version: 1 };
  const draft = { id: 4, published_version: 0 };
  const requests = [];
  let config = { dashboard_ids: [2, 1], favorite_ids: [8], dashboards: [] },
    fail = false;
  const component = load(
    "../src/views/manage/index.vue",
    {
      vue,
      "vue-router": { useRouter: () => ({}) },
      "@liujitcn/kratos-admin-core": { t: key => key },
      "@liujitcn/kratos-admin-core/table": { buildPageRequest: params => params },
      "@liujitcn/kratos-admin-core/components/ProTable": {},
      "@liujitcn/kratos-admin-core/stores/runtime": { useAuthStore: () => ({ flatMenuListGet: [] }) },
      "../../bi-dashboard-schema": schema,
      "../../api/bi/admin/v1/bi_dashboard": {
        defBiDashboardService: { PageBiDashboard: async () => ({ dashboards: [item, draft], total: 2 }) }
      },
      "../../api/bi/admin/v1/bi_card": {
        defBiCardService: {
          GetBiWorkbench: async () => config,
          SaveBiWorkbench: async request => {
            requests.push(request);
            if (fail) throw new Error("保存失败");
            config = { ...request, dashboards: [] };
            return config;
          }
        }
      }
    },
    { ElMessage: { success: () => undefined, error: () => undefined } }
  );
  const state = component.setup({}, { expose: () => undefined });
  const table = await state.requestDashboardTable({ page_num: 1 });
  assert.equal(table.data.total, 2);
  const column = state.columns.value.find(column => column.prop === "favorite");
  const status = column.statusProps;
  const row = table.data.list.find(row => row.id === item.id);
  assert.equal(row.favorite, false);
  assert.equal(status.disabled({ row: draft }), true);
  assert.equal(await state.setFavorite(draft, true), false);
  assert.equal(requests.length, 0);
  assert.equal(await status.beforeChange({ row }), true);
  assert.deepEqual(Array.from(config.dashboard_ids), [2, 1, 3]);
  assert.deepEqual(Array.from(requests[0].favorite_ids), [8]);
  assert.equal(await state.setFavorite(item, true), true);
  assert.equal(requests.length, 1);
  config = { dashboard_ids: [1, 3, 2], favorite_ids: [8, 9], dashboards: [] };
  assert.equal(await state.setFavorite({ id: 1, published_version: 1 }, false), true);
  assert.deepEqual(Array.from(config.dashboard_ids), [3, 2]);
  assert.deepEqual(Array.from(requests[1].favorite_ids), [8, 9]);
  fail = true;
  row.favorite = true;
  assert.equal(await status.beforeChange({ row }), false);
  assert.equal(state.savingFavorite.value, false);
  assert.equal(row.favorite, true);
  assert.deepEqual(Array.from(config.dashboard_ids), [3, 2]);
});
