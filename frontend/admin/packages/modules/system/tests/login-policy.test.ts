import assert from "node:assert/strict";
import { test } from "node:test";
import { formatLoginPolicyList } from "../src/views/base/login-policy/format.js";

test("缺失的登录策略列表按空数组格式化", () => {
  assert.equal(formatLoginPolicyList(undefined, "\n"), "");
});

test("登录策略列表按表单指定的分隔符格式化", () => {
  assert.equal(formatLoginPolicyList(["10.0.0.1", "10.0.0.2"], "\n"), "10.0.0.1\n10.0.0.2");
});
