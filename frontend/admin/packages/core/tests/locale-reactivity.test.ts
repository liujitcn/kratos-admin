import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { dirname, join, resolve } from "node:path";
import test from "node:test";
import { setImmediate } from "node:timers/promises";
import { runInNewContext } from "node:vm";
import ts from "typescript";
import * as vue from "vue";
import { createPinia } from "pinia";

const require = createRequire(import.meta.url);
const { parse, compileScript } = require("vue/compiler-sfc");
const sourceRoot = join(process.cwd(), "src");

/** 执行真实源码，仅替换浏览器和接口依赖。 */
function loadScript(source: string, filename: string, mocks: Record<string, any> = {}): any {
  const module = { exports: {} };
  const loadDependency = (name: string): any => {
    if (Object.hasOwn(mocks, name)) return mocks[name];
    if (name.endsWith(".vue")) return { render: () => null };
    if (name.startsWith("@/") || name.startsWith(".")) {
      const path = name.startsWith("@/") ? join(sourceRoot, name.slice(2)) : resolve(dirname(filename), name);
      return loadScript(readFileSync(`${path}.ts`, "utf8"), `${path}.ts`, mocks);
    }
    return require(name);
  };
  runInNewContext(
    ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } })
      .outputText,
    {
      ...vue,
      module,
      exports: module.exports,
      require: loadDependency,
      console,
      window: { localStorage: { setItem: () => undefined } },
      document: { documentElement: {} }
    }
  );
  return module.exports;
}

/** 创建独立语言运行时，驱动与管理端一致的切换入口。 */
function createLocaleRuntime() {
  const filename = join(sourceRoot, "locales/index.ts");
  const locales = loadScript(readFileSync(filename, "utf8"), filename, {
    "./generated": { SUPPORTED_LOCALES: ["zh-CN", "en-US"], DEFAULT_LOCALE: "zh-CN", DAYJS_LOCALE_MAP: {} },
    "../runtime-messages": { runtimeMessage: (key: string) => key }
  });
  const messages = Object.fromEntries(
    ["zh-CN", "en-US"].map(locale => [locale, JSON.parse(readFileSync(join(sourceRoot, `locales/${locale}.json`), "utf8"))])
  );
  locales.registerLocaleMessages([{ name: "kratos-admin", messages }]);
  return locales;
}

/** 使用真实字典 Store 和可控接口响应验证缓存刷新。 */
function createDictStore(optionBaseDict: () => Promise<any> = async () => ({ base_dicts: [] })) {
  const filename = join(sourceRoot, "stores/modules/dict.ts");
  const { useDictStore } = loadScript(readFileSync(filename, "utf8"), filename, {
    "@/stores": createPinia(),
    "@/stores/helper/persist": () => ({}),
    "@/api/system/admin/v1/base_dict": { defBaseDictService: { OptionBaseDict: optionBaseDict } }
  });
  return useDictStore(createPinia());
}

/** 挂载真实 SFC 的 setup，以轻量渲染器检查依赖更新和组件状态。 */
function mountComponent(path: string, props: Record<string, any>, mocks: Record<string, any>, renderState: (state: any) => void) {
  const filename = join(sourceRoot, path);
  const { descriptor } = parse(readFileSync(filename, "utf8"), { filename });
  const script = compileScript(descriptor, { id: path, genDefaultAs: "component" });
  const component = loadScript(`${script.content}\nmodule.exports = component;`, filename, {
    "element-plus": { ElTable: {} },
    "@element-plus/icons-vue": {},
    "@/utils": { generateUUID: () => "test-table", handleProp: (prop: string) => prop },
    "@/utils/proTable": { resolveTableColumnAlign: () => "left" },
    ...mocks
  });
  component.render = () => {
    renderState((vue.getCurrentInstance() as any).setupState);
    return null;
  };
  const renderer = vue.createRenderer({
    patchProp: () => undefined,
    insert: () => undefined,
    remove: () => undefined,
    createElement: () => ({}),
    createText: () => ({}),
    createComment: () => ({}),
    setText: () => undefined,
    setElementText: () => undefined,
    parentNode: () => null,
    nextSibling: () => null
  });
  const app = renderer.createApp({ setup: () => () => vue.h(component, props) });
  app.mount({});
  return {
    get state(): any {
      return (app._instance!.subTree.component as any).setupState;
    },
    unmount: () => app.unmount()
  };
}

test("切换语言立即更新表头、搜索与行操作文案，并保留列设置和查询状态", async context => {
  const locales = createLocaleRuntime();
  const buildColumns = () => [
    {
      prop: "status",
      label: locales.t("common.field.status"),
      sortable: false,
      search: { el: "select", label: locales.t("common.field.status"), defaultValue: 1 },
      actions: [{ label: locales.t("common.action.edit") }]
    }
  ];
  const props = vue.reactive({ columns: buildColumns(), requestAuto: false });
  let labels: string[] = [];
  const mounted = mountComponent("components/ProTable/index.vue", props, { "@/locales": locales }, state => {
    labels = [
      state.tableColumns[0].label,
      state.searchColumns[0].search.label,
      state.tableColumns[0].actions[0].label,
      state.colSetting[0].label
    ];
  });
  context.after(mounted.unmount);
  const previousLabels = [...labels];
  mounted.state.tableColumns[0].isShow = false;
  mounted.state.tableColumns[0].sortable = true;
  mounted.state.searchParam.status = 2;
  mounted.state.pageable.page_num = 3;
  mounted.state.selectionChange([{ id: 7 }]);
  await locales.setCurrentLocale("en-US");
  props.columns = buildColumns();
  await vue.nextTick();

  assert.notDeepEqual(labels, previousLabels);
  assert.deepEqual(labels, ["Status", "Status", "Edit", "Status"]);
  assert.equal(mounted.state.tableColumns[0].isShow, false);
  assert.equal(mounted.state.tableColumns[0].sortable, true);
  assert.equal(mounted.state.searchParam.status, 2);
  assert.equal(mounted.state.pageable.page_num, 3);
  assert.deepEqual([...mounted.state.selectedListIds], [7]);
  assert.equal(props.columns[0].sortable, false, "列设置不应修改父页面配置");
});

test("字典缓存替换后已挂载标签立即更新文字和标签类型", async context => {
  const dictStore = createDictStore();
  dictStore.setDictionary({ code: "status", items: [{ value: "1", label: "启用", tag_type: "success" }] });
  let label = "";
  let tagType = "";
  const mounted = mountComponent(
    "components/Dict/DictLabel.vue",
    { code: "status", modelValue: 1 },
    {
      "@/stores/modules/dict": { useDictStore: () => dictStore }
    },
    state => {
      label = state.label;
      tagType = state.tagType;
    }
  );
  context.after(mounted.unmount);
  await setImmediate();
  await vue.nextTick();
  assert.equal(label, "启用");
  dictStore.setDictionary({ code: "status", items: [{ value: "1", label: "Enabled", tag_type: "warning" }] });
  await vue.nextTick();
  assert.equal(label, "Enabled");
  assert.equal(tagType, "warning");
});

test("同语言自定义翻译更新多级表头，页面显式显隐配置仍能覆盖用户设置", async context => {
  const locales = createLocaleRuntime();
  const buildColumns = (isShow = true) => [
    {
      prop: "group",
      label: locales.t("common.field.status"),
      _children: [{ prop: "status", label: locales.t("common.field.status"), isShow }]
    }
  ];
  const props = vue.reactive({ columns: buildColumns(), requestAuto: false });
  const mounted = mountComponent("components/ProTable/index.vue", props, { "@/locales": locales }, () => undefined);
  context.after(mounted.unmount);
  mounted.state.tableColumns[0]._children[0].isShow = false;
  locales.applyCustomLocaleMessages([{ locale: "zh-CN", key: "common.field.status", value: "当前状态" }]);
  props.columns = buildColumns();
  await vue.nextTick();
  assert.equal(mounted.state.tableColumns[0].label, "当前状态");
  assert.equal(mounted.state.flatColumns[0].label, "当前状态");
  assert.equal(mounted.state.flatColumns[0].isShow, false);
  props.columns = buildColumns(false);
  await vue.nextTick();
  props.columns = buildColumns(true);
  await vue.nextTick();
  assert.equal(mounted.state.flatColumns[0].isShow, true);
  assert.equal(props.columns[0]._children[0].isShow, true);
});

test("刷新字典期间保留选中值，响应完成后更新单选与多选文案", async context => {
  let resolveResponse!: (value: any) => void;
  const dictStore = createDictStore(
    () =>
      new Promise(resolveValue => {
        resolveResponse = resolveValue;
      })
  );
  dictStore.setDictionary({
    code: "status",
    items: [
      { value: "1", label: "启用" },
      { value: "2", label: "停用" }
    ]
  });
  const emitted: unknown[] = [];
  const mocks = { "@/stores/modules/dict": { useDictStore: () => dictStore }, "@/locales": createLocaleRuntime() };
  let labels: string[] = [];
  const single = mountComponent(
    "components/Dict/index.vue",
    { code: "status", modelValue: 1, "onUpdate:modelValue": (value: unknown) => emitted.push(value) },
    mocks,
    state => {
      labels = state.options.map((option: any) => option.label);
    }
  );
  const multi = mountComponent(
    "components/Dict/index.vue",
    { code: "status", modelValue: [1, 2], type: "checkbox", "onUpdate:modelValue": (value: unknown) => emitted.push(value) },
    mocks,
    () => undefined
  );
  context.after(single.unmount);
  context.after(multi.unmount);
  await setImmediate();
  await vue.nextTick();
  const refresh = dictStore.updateDictionaryCache();
  await vue.nextTick();
  assert.deepEqual(labels, ["启用", "停用"]);
  resolveResponse({
    base_dicts: [
      {
        code: "status",
        items: [
          { value: "1", label: "Enabled" },
          { value: "2", label: "Disabled" }
        ]
      }
    ]
  });
  await refresh;
  await vue.nextTick();
  assert.deepEqual(labels, ["Enabled", "Disabled"]);
  assert.equal(single.state.selectedSingleValue, 1);
  assert.deepEqual([...multi.state.selectedMultiValue], [1, 2]);
  assert.deepEqual(emitted, []);
});

test("语言切换刷新异步枚举，旧响应不能覆盖新文案，卸载后移除刷新处理器", async context => {
  const locales = createLocaleRuntime();
  const responses: Array<(value: any) => void> = [];
  const mounted = mountComponent(
    "components/ProTable/index.vue",
    {
      columns: [{ prop: "status", enum: () => new Promise(resolveValue => responses.push(resolveValue)) }],
      requestAuto: false
    },
    { "@/locales": locales },
    () => undefined
  );
  context.after(mounted.unmount);
  assert.equal(responses.length, 1);
  const switching = locales.setCurrentLocale("en-US");
  await vue.nextTick();
  assert.equal(responses.length, 2);
  responses[1]([{ value: 1, label: "Enabled" }]);
  await switching;
  responses[0]([{ value: 1, label: "启用" }]);
  await vue.nextTick();
  assert.equal(mounted.state.enumMap.get("status")[0].label, "Enabled");
  mounted.unmount();
  await locales.setCurrentLocale("zh-CN");
  assert.equal(responses.length, 2);
});
