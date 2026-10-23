import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import test from "node:test";
import { createSourcePatterns } from "../build/source-patterns.js";

test("Windows npm 源码匹配 Vite 标准化路径及 Vue 子请求", () => {
  const root = "C:\\app\\node_modules\\.pnpm\\core@0.0.39\\node_modules\\@liujitcn\\kratos-admin-core\\dist\\package\\src";
  const [pattern] = createSourcePatterns([root]);
  for (const file of ["utils/request.ts", "module.ts", "views/Login.vue", "views/Login.vue?vue&type=script&lang.ts"]) {
    assert.ok(pattern.test(`${root.replaceAll("\\", "/")}/${file}`), file);
    assert.ok(pattern.test(`${root}\\${file.replaceAll("/", "\\")}`), file);
  }
});

test("POSIX 路径转义特殊字符并限制源码目录和文件类型", () => {
  const [pattern] = createSourcePatterns(["/app/core(1)/src"]);
  assert.ok(pattern.test("/app/core(1)/src/views/index.vue"));
  assert.ok(!pattern.test("/app/core1/src/views/index.vue"));
  assert.ok(!pattern.test("/app/core(1)/src-other/index.ts"));
  assert.ok(!pattern.test("/app/core(1)/src/styles/index.scss"));
  assert.ok(!pattern.test("/other/app/core(1)/src/index.ts"));
});

test("纯表格的团队筛选不复用城市级联树", async () => {
  const source = await readFile(join(process.cwd(), "src/views/demo/table/index.vue"), "utf8");
  const data = await readFile(join(process.cwd(), "src/views/demo/demo-data.ts"), "utf8");
  assert.match(data, /export const demoOwnerGroupTree/);
  assert.match(source, /demoOwnerGroupTree/);
  assert.doesNotMatch(source, /const ownerGroupOptions = demoRegionTree/);
});
