import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { test } from "node:test";

test("审计日志详情按钮不能把 64 位日志ID转换为 JavaScript number", async () => {
  const source = await readFile(join(process.cwd(), "src/components/log.ts"), "utf8");

  assert.match(source, /onClick: scope => onClick\(String\(scope\.row\.id\)\)/);
  assert.doesNotMatch(source, /Number\(scope\.row\.id\)/);
});
