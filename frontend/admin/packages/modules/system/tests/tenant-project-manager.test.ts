import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import test from "node:test";
import type { BaseTenantProject } from "../src/rpc/system/admin/v1/base_tenant_project.js";
import { arrangeTenantProjectColumns } from "../src/components/tenant-project/tenant-project-manager-data.js";
import { mergeTenantProjectExtraData, tenantProjectKey } from "../src/components/tenant-project/tenant-project-manager-data.js";

const project: BaseTenantProject = {
  id: 101,
  tenant_id: 1,
  code: "demo",
  name: "Demo",
  status: 1,
  sort: 1,
  remark: "",
  created_at: "",
  updated_at: ""
};

test("tenantProjectKey uses tenant and project ids", () => {
  assert.equal(tenantProjectKey(1, 101), "1:101");
});

test("mergeTenantProjectExtraData preserves public project fields", () => {
  const merged = mergeTenantProjectExtraData(project, {
    sync_status: "ready",
    name: "external-name",
    tenant_id: 99,
    id: 999
  });

  assert.equal(merged.sync_status, "ready");
  assert.equal(merged.name, "Demo");
  assert.equal(merged.tenant_id, 1);
  assert.equal(merged.id, 101);
});

test("arrangeTenantProjectColumns inserts slot columns after the configured field", () => {
  const columns = arrangeTenantProjectColumns(
    [{ prop: "name" }, { prop: "code" }, { prop: "remark" }],
    [
      { prop: "address", after: "name" },
      { prop: "owner", after: "code" }
    ]
  );

  assert.deepEqual(
    columns.map(column => column.prop),
    ["name", "address", "code", "owner", "remark"]
  );
});

test("项目状态切换成功后由开关直接更新当前行", async () => {
  const source = await readFile(join(process.cwd(), "src/components/tenant-project/TenantProjectManager.vue"), "utf8");
  const statusBlock = source.match(/async function handleBeforeSetStatus\([\s\S]*?\n\}/)?.[0];

  assert.ok(statusBlock, "缺少项目状态切换方法");
  assert.match(statusBlock, /await defBaseTenantProjectService\.SetBaseTenantProjectStatus/);
  assert.doesNotMatch(statusBlock, /refreshTable\(/);
  assert.match(statusBlock, /return true;/);
});
