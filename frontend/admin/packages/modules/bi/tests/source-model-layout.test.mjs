import assert from "node:assert/strict";
import { test } from "node:test";
import { formatSourceModelLayout } from "../src/components/source-model-layout.ts";

function makeModel(keys, pairs, baseTable = keys[0] ?? "") {
  return {
    base_table: baseTable,
    tables: keys.map(key => ({ key, tenant_column: "tenant_id", x: 0, y: 0 })),
    relations: pairs.map(([left, right]) => ({
      left_table: left,
      left_column: `${right}_id`,
      right_table: right,
      right_column: "id",
      join_type: "left"
    })),
    dimensions: [],
    measures: []
  };
}

test("截图中的四条关联分别展开，事实表居中且没有节点挡住连线通道", () => {
  const model = makeModel(
    ["dept", "role", "post", "tenant", "user"],
    [
      ["user", "dept"],
      ["user", "role"],
      ["user", "post"],
      ["user", "tenant"]
    ],
    "user"
  );
  const relations = structuredClone(model.relations);
  formatSourceModelLayout(model);
  const source = model.tables.find(table => table.key === "user");
  const targets = model.tables.filter(table => table !== source);
  assert.ok(targets.every(target => Math.abs(target.x - source.x) > 250));
  assert.equal(targets.filter(target => target.x < source.x).length, 2);
  assert.equal(targets.filter(target => target.x > source.x).length, 2);
  assert.equal(source.y, (targets[0].y + targets.at(-1).y) / 2);
  assert.ok(Math.max(...targets.map(target => target.y)) - Math.min(...targets.map(target => target.y)) <= 280);
  for (const target of targets) {
    for (const other of targets.filter(table => table !== target && table.x === target.x))
      assert.ok(Math.abs(target.y - other.y) >= 230);
  }
  assert.deepEqual(model.relations, relations);
  assert.ok(model.tables.every(table => table.tenant_column === "tenant_id"));
});

test("多级分支按子树分配行，相邻层级之间保留连线空间", () => {
  const model = makeModel(
    ["root", "a", "b", "a1", "a2", "b1"],
    [
      ["root", "a"],
      ["root", "b"],
      ["a", "a1"],
      ["a", "a2"],
      ["b", "b1"]
    ]
  );
  formatSourceModelLayout(model);
  const tables = new Map(model.tables.map(table => [table.key, table]));
  for (const relation of model.relations)
    assert.ok(Math.abs(tables.get(relation.right_table).x - tables.get(relation.left_table).x) > 250);
  assert.equal(tables.get("a").y, (tables.get("a1").y + tables.get("a2").y) / 2);
  assert.equal(tables.get("b").y, tables.get("b1").y);
  for (const [index, table] of model.tables.entries()) {
    for (const other of model.tables.slice(index + 1)) {
      assert.ok(Math.abs(table.x - other.x) >= 250 || Math.abs(table.y - other.y) >= 230);
    }
  }
  const positions = model.tables.map(table => [table.x, table.y]);
  formatSourceModelLayout(model);
  assert.deepEqual(
    model.tables.map(table => [table.x, table.y]),
    positions
  );
});

test("尚未连通的子树和孤立表也分布在事实表两侧", () => {
  const model = makeModel(["leaf", "isolated", "root", "parent"], [["parent", "leaf"]], "root");
  formatSourceModelLayout(model);
  const tables = new Map(model.tables.map(table => [table.key, table]));
  assert.ok(Math.abs(tables.get("root").x - tables.get("parent").x) > 250);
  assert.ok(Math.abs(tables.get("leaf").x - tables.get("parent").x) > 250);
  assert.equal(tables.get("leaf").y, tables.get("parent").y);
  assert.ok(tables.get("isolated").x !== tables.get("root").x);
});

test("空模型、单表、失效关系及循环关系均可有限完成布局", () => {
  for (const model of [
    makeModel([], []),
    makeModel(["root"], []),
    makeModel(
      ["root", "a", "b"],
      [
        ["missing", "a"],
        ["a", "b"],
        ["b", "a"],
        ["root", "root"]
      ]
    )
  ]) {
    formatSourceModelLayout(model);
    assert.ok(model.tables.every(table => Number.isFinite(table.x) && Number.isFinite(table.y)));
    assert.equal(new Set(model.tables.map(table => `${table.x},${table.y}`)).size, model.tables.length);
  }
});
