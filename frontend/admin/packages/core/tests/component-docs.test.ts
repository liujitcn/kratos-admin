import assert from "node:assert/strict";
import test from "node:test";
import { reactive, watch } from "vue";
import { ADMIN_COMPONENT_DOCS, getAdminComponentDocs, type AdminComponentDoc } from "../src/demo.js";
import {
  createComponentDocsState,
  filterComponentDocs,
  getComponentDocsPage,
  openComponentDoc,
  returnComponentDoc
} from "../src/views/demo/docs/navigation.js";

/** 构造仅含文档信息的测试组件，不挂载业务能力。 */
function entry(id: string, label: string, owner: AdminComponentDoc["owner"] = "Core"): AdminComponentDoc {
  return {
    id,
    label,
    owner,
    file: `${id}/index.vue`,
    group: "表格与查询",
    description: label,
    usage: label,
    code: `<${id} />`,
    api: { props: [], events: [], slots: [], methods: [] }
  };
}

test("组件文档只聚合启用模块，不要求基础包反向引用业务包", () => {
  const core = entry("ProTable", "高级表格");
  const bi = entry("BiChart", "BI 图表", "BI");
  const modules = [
    { name: "core", extensions: { [ADMIN_COMPONENT_DOCS]: [core] } },
    { name: "bi", extensions: { [ADMIN_COMPONENT_DOCS]: [bi] } }
  ];
  assert.deepEqual(getAdminComponentDocs(modules), [core, bi]);
  assert.deepEqual(getAdminComponentDocs(modules.slice(0, 1)), [core]);
  assert.deepEqual(getAdminComponentDocs([{ name: "unrelated" }]), []);
});

test("组件目录支持中英文搜索及模块筛选，无结果保持空列表", () => {
  const entries = [entry("ProTable", "高级表格"), entry("BiChart", "BI 图表", "BI")];
  assert.deepEqual(
    filterComponentDocs(entries, "protable", "全部").map(item => item.id),
    ["ProTable"]
  );
  assert.deepEqual(
    filterComponentDocs(entries, "表格", "Core").map(item => item.id),
    ["ProTable"]
  );
  assert.deepEqual(
    filterComponentDocs(entries, "", "BI").map(item => item.id),
    ["BiChart"]
  );
  assert.deepEqual(filterComponentDocs(entries, "不存在", "全部"), []);
});

test("71 个组件按每页 4 项完整分页，缩小结果后夹紧页码", () => {
  const entries = Array.from({ length: 71 }, (_, index) => index);
  const pages = Array.from({ length: 18 }, (_, index) => getComponentDocsPage(entries, index + 1, 4));
  assert.deepEqual(
    pages.flatMap(page => page.items),
    entries
  );
  assert.equal(pages.at(-1)?.items.length, 3);
  assert.equal(getComponentDocsPage(entries.slice(0, 2), 18, 4).page, 1);
  assert.deepEqual(getComponentDocsPage([], 18, 4).items, []);
});

test("多级返回逐级恢复组件、筛选、总览页码和 API 页码", () => {
  const state = createComponentDocsState();
  state.query = "表";
  state.overviewPage = 2;
  state.overviewSize = 8;
  openComponentDoc(state, "ProTable");
  state.api = "events";
  state.apiPage = 2;
  openComponentDoc(state, "ColSetting");
  assert.equal(state.api, "props");
  assert.equal(state.apiPage, 1);
  returnComponentDoc(state);
  assert.equal(state.id, "ProTable");
  assert.equal(state.api, "events");
  assert.equal(state.apiPage, 2);
  returnComponentDoc(state);
  assert.equal(state.id, "");
  assert.equal(state.query, "表");
  assert.equal(state.overviewPage, 2);
  assert.equal(state.overviewSize, 8);
  assert.equal(state.history.length, 0);
  returnComponentDoc(state);
  assert.equal(state.id, "");
});

test("点击当前组件不会制造返回循环", () => {
  const state = createComponentDocsState();
  openComponentDoc(state, "ColSetting");
  openComponentDoc(state, "ColSetting");
  assert.equal(state.history.length, 1);
  returnComponentDoc(state);
  assert.equal(state.id, "");
});

test("超过 20 次组件跳转仍能返回原总览筛选和页码", () => {
  const state = createComponentDocsState();
  state.query = "表";
  state.overviewPage = 5;
  for (let index = 0; index < 25; index++) openComponentDoc(state, `Component${index}`);
  for (let index = 0; index < 25; index++) returnComponentDoc(state);
  assert.equal(state.id, "");
  assert.equal(state.query, "表");
  assert.equal(state.overviewPage, 5);
});

test("返回恢复筛选时，同步页码重置不会覆盖来源位置", () => {
  const state = reactive(createComponentDocsState());
  const stopOverview = watch(
    () => [state.query, state.owner, state.overviewSize],
    () => {
      state.overviewPage = 1;
    },
    { flush: "sync" }
  );
  const stopApi = watch(
    () => state.api,
    () => {
      state.apiPage = 1;
    },
    { flush: "sync" }
  );
  state.overviewSize = 8;
  state.overviewPage = 3;
  openComponentDoc(state, "ProTable");
  state.api = "events";
  state.apiPage = 2;
  openComponentDoc(state, "ColSetting");
  state.query = "列设置";
  state.owner = "Core";
  state.overviewSize = 4;
  returnComponentDoc(state);
  assert.equal(state.overviewPage, 3);
  assert.equal(state.overviewSize, 8);
  assert.equal(state.query, "");
  assert.equal(state.owner, "全部");
  assert.equal(state.api, "events");
  assert.equal(state.apiPage, 2);
  stopOverview();
  stopApi();
});
