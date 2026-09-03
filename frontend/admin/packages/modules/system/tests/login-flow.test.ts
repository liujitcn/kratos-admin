import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { test } from "node:test";

function readSource(path: string) {
  return readFile(join(process.cwd(), path), "utf8");
}

test("强制改密弹窗挂载到 body，遮罩覆盖完整布局", async () => {
  const source = await readSource("src/components/ForcedPasswordDialog.vue");

  assert.match(source, /<ProDialog[\s\S]*append-to-body/);
  assert.match(source, /:show-footer="false"/);
});

test("登录完成后使用统一回跳逻辑，不强制跳转个人信息页", async () => {
  const source = await readSource("../../core/src/views/login/components/LoginForm.vue");

  assert.match(source, /await navigateTo\(router, getLoginRedirectPath\(\)\)/);
  assert.doesNotMatch(source, /navigateTo\(router, ["']\/profile["']/);
});
