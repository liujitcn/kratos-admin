import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { test } from "node:test";

test("用户表单仅在默认租户下要求选择所属租户", async () => {
	const [pageSource, tenantSource] = await Promise.all([
		readFile(join(process.cwd(), "src/views/base/user/index.vue"), "utf8"),
		readFile(join(process.cwd(), "../../core/src/tenant.ts"), "utf8")
	]);

	assert.match(pageSource, /required: isDefaultTenant\.value/);
	assert.match(tenantSource, /visible: \(\) => isDefaultTenant\.value/);
});
