import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { test } from "node:test";

function readSource(path: string) {
  return readFile(join(process.cwd(), path), "utf8");
}

test("自定义业务页面沿用统一卡片外观和页面间距", async () => {
  const [elementStyles, loginPolicy, dashboard, session, opsMonitoring] = await Promise.all([
    readSource("../../core/src/styles/element.scss"),
    readSource("src/views/base/login-policy/index.vue"),
    readSource("src/views/base/dashboard/index.vue"),
    readSource("src/views/profile/components/session.vue"),
    readSource("src/views/tool/ops-monitoring/index.vue")
  ]);

  assert.match(elementStyles, /\.el-card\.admin-page-card\s*\{[^}]*border-radius:\s*var\(--admin-page-radius\)[^}]*box-shadow:/s);
  assert.match(elementStyles, /\.el-card\.admin-page-card\.is-always-shadow\s*\{[^}]*box-shadow:/s);

  assert.match(loginPolicy, /<el-card class="admin-page-card">/);
  assert.doesNotMatch(loginPolicy, /\.login-policy-page\s*\{[^}]*padding:/s);

  assert.equal(dashboard.match(/<el-card[^>]*admin-page-card/g)?.length, 4);
  assert.doesNotMatch(dashboard, /\.dashboard-page\s*\{[^}]*padding:/s);
  assert.doesNotMatch(dashboard, /dashboard-header|updatedAt/);

  assert.match(session, /\.session-card\s*\{[^}]*border-radius:\s*var\(--admin-page-radius\)/s);
  assert.match(session, /\.session-card \.el-card__header/);
  assert.match(session, /\.session-card \.el-card__body/);

  assert.doesNotMatch(opsMonitoring, /ops-page-head|ops-page-meta|ops-section-title/);
  assert.match(opsMonitoring, /<section class="ops-storage-grid ops-storage-section"/);
  assert.match(opsMonitoring, /<template #header>\s*<div class="ops-storage-head">/s);
});
