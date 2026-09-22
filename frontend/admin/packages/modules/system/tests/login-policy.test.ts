import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { test } from "node:test";
import { formatLoginPolicyList } from "../src/views/base/login-policy/format.js";

test("缺失的登录策略列表按空数组格式化", () => {
  assert.equal(formatLoginPolicyList(undefined, "\n"), "");
});

test("登录策略列表按表单指定的分隔符格式化", () => {
  assert.equal(formatLoginPolicyList(["10.0.0.1", "10.0.0.2"], "\n"), "10.0.0.1\n10.0.0.2");
});

test("登录策略列表的两个开关均先确认再保存", async () => {
  const source = await readFile(join(process.cwd(), "src/views/base/login-policy/index.vue"), "utf8");
  const concurrentColumn = source.match(/prop: "allow_concurrent_login"[\s\S]*?\n\s*\},\n\s*\{ prop: "password_min_length"/)?.[0];
  const concurrentBlock = source.match(/async function handleSetConcurrentLogin\([\s\S]*?\n\}/)?.[0];
  const statusBlock = source.match(/async function handleSetStatus\([\s\S]*?\n\}/)?.[0];

  assert.ok(concurrentColumn, "缺少允许同时登录列配置");
  assert.match(concurrentColumn, /beforeChange: scope => handleSetConcurrentLogin/);
  assert.ok(concurrentBlock, "缺少允许同时登录切换方法");
  assert.match(concurrentBlock, /ElMessageBox\.confirm/);
  assert.match(concurrentBlock, /GetBaseLoginPolicy/);
  assert.match(concurrentBlock, /UpdateBaseLoginPolicy/);
  assert.ok(concurrentBlock.indexOf("ElMessageBox.confirm") < concurrentBlock.indexOf("UpdateBaseLoginPolicy"));
  assert.ok(statusBlock, "缺少登录策略状态切换方法");
  assert.match(statusBlock, /ElMessageBox\.confirm/);
  assert.match(statusBlock, /SetBaseLoginPolicyStatus/);
  assert.ok(statusBlock.indexOf("ElMessageBox.confirm") < statusBlock.indexOf("SetBaseLoginPolicyStatus"));
});

test("登录策略单行和批量删除均提取有效ID", async () => {
  const source = await readFile(join(process.cwd(), "src/views/base/login-policy/index.vue"), "utf8");
  const deleteBlock = source.match(/async function handleDelete\([\s\S]*?\n\}/)?.[0];

  assert.ok(deleteBlock, "缺少登录策略删除方法");
  assert.match(deleteBlock, /typeof value === "object"\s*\? \[value\.id\]/);
  assert.match(deleteBlock, /typeof item === "object" \? item\.id : item/);
  assert.match(deleteBlock, /if \(!ids\.length\)/);
  assert.match(deleteBlock, /DeleteBaseLoginPolicy\(\{ id: ids\.join\(","\) \}\)/);
});
