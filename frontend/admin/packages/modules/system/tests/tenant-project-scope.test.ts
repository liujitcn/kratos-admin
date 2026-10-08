import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import test from "node:test";
import { parseTenantProjectValue } from "../src/components/tenant-project/tenant-project-manager-data.js";

test("解析租户项目树值区分租户节点与项目节点", () => {
  assert.deepEqual(parseTenantProjectValue("tenant:3"), { tenant_id: 3, project_id: 0 });
  assert.deepEqual(parseTenantProjectValue("project:3:9"), { tenant_id: 3, project_id: 9 });
  assert.deepEqual(parseTenantProjectValue(undefined), {});
  assert.deepEqual(parseTenantProjectValue("other:1"), {});
});

test("租户项目文本组件支持仅传项目 ID 并从共享目录解析名称", async () => {
  const source = await readFile(join(process.cwd(), "src/components/tenant-project/TenantProjectText.vue"), "utf8");

  assert.match(source, /projectId = props\.projectId \?\? props\.row\.project_id/);
  assert.match(source, /resolveTenantProjectDisplay\(props\.tenantId \?\? props\.row\.tenant_id, projectId\)/);
  assert.match(source, /displayInfo\?\.projectName/);
  assert.match(source, /displayInfo\?\.tenantName/);
});

test("租户项目范围列与搜索条件使用同一份范围状态", async () => {
  const source = await readFile(join(process.cwd(), "src/components/tenant-project/tenant-project-scope.ts"), "utf8");

  assert.match(source, /render: \(\{ row \}\) => h\(TenantProjectText/);
  assert.match(source, /render: \(scope: SearchRenderScope\) => renderTenantProjectSearch/);
  assert.match(source, /await loadTenantProjectOptions\(isDefaultTenant\.value, force\)/);
  assert.match(source, /delete result\.tenant_project_tree_value/);
});
