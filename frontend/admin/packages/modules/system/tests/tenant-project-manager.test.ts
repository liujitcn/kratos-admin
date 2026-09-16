import assert from "node:assert/strict";
import test from "node:test";
import type { BaseTenantProject } from "../src/rpc/system/admin/v1/base_tenant_project.js";
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
