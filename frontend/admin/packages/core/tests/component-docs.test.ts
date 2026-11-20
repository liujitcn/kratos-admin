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

test("组件目录支持跨模块中英文搜索，无结果保持空列表", () => {
  const entries = [entry("ProTable", "高级表格"), entry("BiChart", "BI 图表", "BI")];
  assert.deepEqual(
    filterComponentDocs(entries, "protable").map(item => item.id),
    ["ProTable"]
  );
  assert.deepEqual(
    filterComponentDocs(entries, "表格").map(item => item.id),
    ["ProTable"]
  );
  assert.deepEqual(
    filterComponentDocs(entries, "BI").map(item => item.id),
    ["BiChart"]
  );
  assert.deepEqual(filterComponentDocs(entries, ""), entries);
  assert.deepEqual(filterComponentDocs(entries, "不存在"), []);
});

test("组件 API 按每页 6 项完整分页，缩小结果后夹紧页码", () => {
  const entries = Array.from({ length: 14 }, (_, index) => index);
  const pages = Array.from({ length: 3 }, (_, index) => getComponentDocsPage(entries, index + 1, 6));
  assert.deepEqual(
    pages.flatMap(page => page.items),
    entries
  );
  assert.equal(pages.at(-1)?.items.length, 2);
  assert.equal(getComponentDocsPage(entries.slice(0, 2), 3, 6).page, 1);
  assert.deepEqual(getComponentDocsPage([], 3, 6).items, []);
});

test("默认展示指定组件，首次进入和无历史返回都保持当前文档", () => {
  const state = createComponentDocsState("ProTable");
  assert.equal(state.id, "ProTable");
  assert.equal(state.history.length, 0);
  returnComponentDoc(state);
  assert.equal(state.id, "ProTable");
  assert.deepEqual(createComponentDocsState("").history, []);
});

test("多级返回逐级恢复组件、搜索和 API 页码", () => {
  const state = createComponentDocsState("SearchForm");
  state.query = "表";
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
  assert.equal(state.id, "SearchForm");
  assert.equal(state.query, "表");
  assert.equal(state.history.length, 0);
  returnComponentDoc(state);
  assert.equal(state.id, "SearchForm");
});

test("点击当前组件不会制造返回循环", () => {
  const state = createComponentDocsState("ProTable");
  openComponentDoc(state, "ColSetting");
  openComponentDoc(state, "ColSetting");
  assert.equal(state.history.length, 1);
  returnComponentDoc(state);
  assert.equal(state.id, "ProTable");
});

test("超过 20 次组件跳转仍能返回起始组件和搜索条件", () => {
  const state = createComponentDocsState("ProTable");
  state.query = "表";
  for (let index = 0; index < 25; index++) openComponentDoc(state, `Component${index}`);
  for (let index = 0; index < 25; index++) returnComponentDoc(state);
  assert.equal(state.id, "ProTable");
  assert.equal(state.query, "表");
});

test("返回恢复 API 分项时，同步页码重置不会覆盖来源位置", () => {
  const state = reactive(createComponentDocsState("ProTable"));
  const stopApi = watch(
    () => state.api,
    () => {
      state.apiPage = 1;
    },
    { flush: "sync" }
  );
  state.api = "events";
  state.apiPage = 2;
  openComponentDoc(state, "ColSetting");
  state.query = "列设置";
  returnComponentDoc(state);
  assert.equal(state.query, "");
  assert.equal(state.api, "events");
  assert.equal(state.apiPage, 2);
  stopApi();
});
