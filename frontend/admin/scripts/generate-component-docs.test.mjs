import assert from "node:assert/strict";
import { mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";
import ts from "typescript";
import { extractComponentApis, validateComponentCatalog } from "./generate-component-docs.mjs";

test("npm 发布转换源码别名后，展示和复制的示例代码保持不变", async () => {
  const directory = new URL("../packages/core/src/views/demo/docs/examples/", import.meta.url);
  const source = await readFile(new URL("catalog.ts", directory), "utf8");
  const api = JSON.parse(await readFile(new URL("component-api.json", directory), "utf8"));
  const examples = text => {
    const compiled = ts.transpileModule(text, {
      compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 }
    }).outputText;
    const exports = {};
    new Function("require", "exports", compiled)(() => api, exports);
    return exports.coreComponentDocs.map(({ id, code }) => ({ id, code }));
  };
  const published = examples(source.replaceAll("@/", "../../../../"));
  examples(source).forEach((entry, index) => assert.equal(published[index].code, entry.code, entry.id));
});

test("组件 API 保留导入继承与工具类型、默认值、重载事件、插槽和公开成员", async () => {
  const root = await mkdtemp(join(tmpdir(), "component-api-"));
  try {
    await writeFile(
      join(root, "props.ts"),
      `export interface BaseProps { /** 记录编号。 */ id: string; ignored: boolean }
export interface ImportedProps extends Omit<BaseProps, "ignored"> { count?: number; enabled?: boolean; nullable?: string | null }
export type Props = ImportedProps & Partial<{ label: string }>;
export interface Events {
  /** 保存记录。 */ (event: "save", id: string): void;
  (event: "cancel" | "close"): void;
  (event: "save", id: number, force?: boolean): void;
}`
    );
    const typedFile = join(root, "Typed.vue");
    await writeFile(
      typedFile,
      `<script setup lang="ts">
import type { Props, Events } from "./props";
const props = withDefaults(defineProps<Props>(), { count: 3, label: "示  例" });
defineEmits<Events>();
defineSlots<{ /** 行内容。 */ row(props: { id: string }): unknown }>();
/** 重置记录。 */
function reset(value: string): boolean { return value.length > 0; }
const state: number = 1;
defineExpose({ reset, value: state });
</script>
<template><slot name="row" :id="props.id" /><slot :name="props.id" :count="props.count" /></template>`
    );
    const runtimeFile = join(root, "Runtime.vue");
    await writeFile(
      runtimeFile,
      `<script setup lang="ts">
import type { PropType } from "vue";
defineProps({ title: { type: String, required: true }, values: { type: Array as PropType<string[]>, default: () => [] }, enabled: Boolean });
defineEmits({ change: (value: string | undefined) => value !== undefined, clear: null });
defineModel<number>("page", { default: 1 });
</script>
<template><slot :row="{ id: 'a' }" /><slot name="footer" /></template>`
    );

    const apis = extractComponentApis([typedFile, runtimeFile]);
    const typed = apis.get(typedFile);
    assert.ok(typed, "应为每个组件生成 API");
    assert.deepEqual(typed.props, [
      { name: "count", description: "", type: "number", required: false, default: "3" },
      { name: "enabled", description: "", type: "boolean", required: false },
      { name: "nullable", description: "", type: "string | null", required: false },
      { name: "id", description: "记录编号。", type: "string", required: true },
      { name: "label", description: "", type: "string", required: false, default: '"示  例"' }
    ]);
    assert.deepEqual(
      typed.events.map(({ name, type }) => ({ name, type })),
      [
        { name: "save", type: "((id: string) => void) | ((id: number, force?: boolean) => void)" },
        { name: "cancel", type: "() => void" },
        { name: "close", type: "() => void" }
      ]
    );
    assert.equal(typed.events[0].description, "保存记录。");
    assert.equal(typed.slots.find(slot => slot.name === "row")?.type, "(props: { id: string; }) => unknown");
    assert.ok(typed.slots.some(slot => slot.name === "[props.id]"));
    assert.deepEqual(typed.methods, [
      { name: "reset", description: "重置记录。", type: "(value: string) => boolean" },
      { name: "value", description: "", type: "number" }
    ]);

    const runtime = apis.get(runtimeFile);
    assert.deepEqual(
      runtime.props.map(({ name, type, required, default: value }) => ({ name, type, required, default: value })),
      [
        { name: "title", type: "string", required: true, default: undefined },
        { name: "values", type: "string[]", required: false, default: "() => []" },
        { name: "enabled", type: "boolean", required: false, default: undefined },
        { name: "page", type: "number", required: false, default: "1" }
      ]
    );
    assert.deepEqual(
      runtime.events.map(({ name, type }) => ({ name, type })),
      [
        { name: "change", type: "(value: string | undefined) => void" },
        { name: "clear", type: "(...args: unknown[]) => void" },
        { name: "update:page", type: "(value: number) => void" }
      ]
    );
    assert.deepEqual(
      runtime.slots.map(slot => slot.name),
      ["default", "footer"]
    );
    assert.equal(runtime.slots[0].type, "(scope: { row: unknown }) => unknown");
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});

test("无法解析的 API 类型报告组件文件名而不是生成空 API", async () => {
  const root = await mkdtemp(join(tmpdir(), "component-api-unresolved-"));
  try {
    const file = join(root, "Broken.vue");
    await writeFile(file, '<script setup lang="ts">import type { Missing } from "./missing"; defineProps<Missing>();</script>');
    assert.throws(() => extractComponentApis([file]), /Broken\.vue.*Missing|Broken\.vue.*missing/s);
    const options = join(root, "Options.vue");
    await writeFile(options, '<script lang="ts">export default { props: { title: String } };</script>');
    assert.throws(() => extractComponentApis([options]), /Options\.vue.*script setup/s);
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});

test("公开方法通过子组件引用调用时保留返回值类型并展开公开 ref", async () => {
  const root = await mkdtemp(join(tmpdir(), "component-api-expose-"));
  try {
    const child = join(root, "Child.vue");
    const parent = join(root, "Parent.vue");
    await writeFile(child, '<script setup lang="ts">async function load() { return true; } defineExpose({ load });</script>');
    await writeFile(
      parent,
      `<script setup lang="ts">
import { ref } from "vue";
import Child from "./Child.vue";
const child = ref<InstanceType<typeof Child>>();
const count = ref(1);
type Item = { id: string };
const selection = ref<Item>();
async function run() { return child.value?.load() ?? false; }
defineExpose({ run, count, selection });
</script>`
    );
    const apis = extractComponentApis([child, parent]);
    assert.deepEqual(
      apis.get(parent).methods.map(({ name, type }) => ({ name, type })),
      [
        { name: "run", type: "() => Promise<boolean>" },
        { name: "count", type: "number" },
        { name: "selection", type: "Item | undefined" }
      ]
    );
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});

test("元组事件保留有名参数及空参数", async () => {
  const root = await mkdtemp(join(tmpdir(), "component-api-tuples-"));
  try {
    const file = join(root, "Tuple.vue");
    await writeFile(file, '<script setup lang="ts">defineEmits<{ change: [value: string]; clear: [] }>();</script>');
    assert.deepEqual(
      extractComponentApis([file])
        .get(file)
        .events.map(({ name, type }) => ({ name, type })),
      [
        { name: "change", type: "(value: string) => void" },
        { name: "clear", type: "() => void" }
      ]
    );
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});

test("catalog 检查缺少的文档，不把示例字符串中的伪 file 算作条目", async () => {
  const root = await mkdtemp(join(tmpdir(), "component-catalog-missing-"));
  try {
    const file = join(root, "catalog.ts");
    for (const catalog of [
      `const code = '<A />; file: "B.vue"';
export const coreComponentDocs = [componentDoc("A", "甲组件", "通用", "A.vue", "组件说明", "组件用法", code)];`,
      `const docs = [{ id: "A", label: "甲组件", file: "A.vue", description: "组件说明", usage: "组件用法", code: '<A />; file: "B.vue"' }];
export const systemComponentDocs = docs.map(doc => ({ ...doc, owner: "System" }));`
    ]) {
      await writeFile(file, catalog);
      assert.throws(() => validateComponentCatalog(file, ["A.vue", "B.vue"]), /catalog\.ts.*缺少.*B\.vue/s);
      assert.doesNotThrow(() => validateComponentCatalog(file, ["A.vue"]));
    }
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});

test("catalog 拒绝重复组件、重复 ID 和空用法或代码", async () => {
  const root = await mkdtemp(join(tmpdir(), "component-catalog-invalid-"));
  try {
    const file = join(root, "catalog.ts");
    const entry = '{ id: "A", label: "甲组件", file: "A.vue", description: "组件说明", usage: "组件用法", code: "<A />" }';
    for (const [entries, files, error] of [
      [`${entry}, ${entry}`, ["A.vue"], /catalog\.ts.*重复.*file.*A\.vue/s],
      [`${entry}, ${entry.replace('file: "A.vue"', 'file: "B.vue"')}`, ["A.vue", "B.vue"], /catalog\.ts.*重复.*id.*A/s],
      [entry.replace('usage: "组件用法"', 'usage: "  "'), ["A.vue"], /catalog\.ts.*A\.vue.*usage/s],
      [entry.replace('code: "<A />"', 'code: ""'), ["A.vue"], /catalog\.ts.*A\.vue.*code/s]
    ]) {
      await writeFile(
        file,
        `const docs = [${entries}]; export const biComponentDocs = docs.map(doc => ({ ...doc, owner: "BI" }));`
      );
      assert.throws(() => validateComponentCatalog(file, files), error);
    }
    await writeFile(file, `const docs = [${entry}]; export const biComponentDocs = docs.map(doc => ({ ...doc, owner: "BI" }));`);
    const ids = new Set(["A"]);
    assert.throws(() => validateComponentCatalog(file, ["A.vue"], ids), /catalog\.ts.*重复.*id.*A/s);
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});
