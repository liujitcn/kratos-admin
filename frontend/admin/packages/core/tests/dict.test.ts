import assert from "node:assert/strict";
import test from "node:test";
import { normalizeDictValue, normalizeDictValues } from "../src/components/Dict/value.js";
import { resolveTableColumnAlign } from "../src/utils/proTable.js";

test("字典下拉在重置后不会保留不属于选项的未知值", () => {
  const options = [1, 2, 3] as const;

  assert.equal(normalizeDictValue(0, options), undefined);
  assert.equal(normalizeDictValue(2, options), 2);
  assert.equal(normalizeDictValue(undefined, options), undefined);
});

test("字典多选只保留当前选项中的值", () => {
  const options = ["admin", "user"] as const;
  assert.deepEqual(normalizeDictValues(["admin", "missing"], options), ["admin"]);
});

test("数字字段右对齐，数字字典和枚举文字居中", () => {
  assert.equal(resolveTableColumnAlign({ prop: "count" }), "right");
  assert.equal(resolveTableColumnAlign({ prop: "status", dictCode: "status" }), "center");
  assert.equal(resolveTableColumnAlign({ prop: "status", enum: [{ label: "启用", value: 1 }] }), "center");
});
