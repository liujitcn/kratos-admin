import assert from "node:assert/strict";
import test from "node:test";
import { normalizeDictValue, normalizeDictValues } from "../src/components/Dict/value.js";
import { resolveTableColumnAlign } from "../src/utils/tableColumn.js";

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

test("普通字段不根据字段名或数据推断对齐方式", () => {
  assert.equal(resolveTableColumnAlign({ prop: "created_at" }), "left");
  assert.equal(resolveTableColumnAlign({ prop: "count" }), "left");
  assert.equal(resolveTableColumnAlign({ prop: "created_at", align: "center" }), "center");
  assert.equal(resolveTableColumnAlign({ prop: "count", align: "right" }), "right");
});

test("预置列组件保留默认对齐方式", () => {
  assert.equal(resolveTableColumnAlign({ cellType: "status" }), "center");
  assert.equal(resolveTableColumnAlign({ cellType: "money" }), "right");
  assert.equal(resolveTableColumnAlign({ type: "selection" }), "center");
});
