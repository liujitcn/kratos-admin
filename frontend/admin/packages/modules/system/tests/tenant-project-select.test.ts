import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import test from "node:test";

const componentPath = "src/components/tenant-project/TenantProjectSelect.vue";

test("租户项目选择器选中值必须规整为树节点字符串，防止显示 [object Object]", async () => {
  const source = await readFile(join(process.cwd(), componentPath), "utf8");

  assert.match(source, /get: \(\) => resolveSelectedKey\(props\.modelValue\)/);
  assert.match(source, /function resolveSelectedKey/);
  assert.match(source, /typeof nested === "string" \? nested : undefined|if \(typeof nested === "string"\) return nested \|\| undefined;/);
  assert.match(source, /return undefined;/);
});

test("租户项目选择器必须忽略外部 renderAfterExpand，保持选中标签缓存可用", async () => {
  const source = await readFile(join(process.cwd(), componentPath), "utf8");

  assert.match(source, /const \{ renderAfterExpand: _ignored, \.\.\.rest \} = attrs;/);
  assert.match(source, /renderAfterExpand: true/);
  assert.doesNotMatch(source, /v-bind="\$attrs"/);
});
