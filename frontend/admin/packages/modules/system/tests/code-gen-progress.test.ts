import assert from "node:assert/strict";
import { test } from "node:test";
import type { CodeGenTask } from "../src/rpc/system/admin/v1/code_gen";
import { normalizeCodeGenTask } from "../src/utils/code-gen-progress.js";

test("生成初期省略已完成计数时显示零进度", () => {
  const task = normalizeCodeGenTask({ total_steps: 7, tables: [{ table_id: 1, total_steps: 7 }] } as CodeGenTask);
  assert.equal(Math.round((task.completed_steps / task.total_steps) * 100), 0);
  assert.equal(task.tables[0].completed_steps, 0);
});

test("步骤尚未注册时补齐空集合与零值，保留后续真实进度", () => {
  const initial = normalizeCodeGenTask({} as CodeGenTask);
  assert.equal(initial.total_steps, 0);
  assert.equal(initial.completed_steps, 0);
  assert.deepEqual(initial.tables, []);
  const running = normalizeCodeGenTask({ total_steps: 7, completed_steps: 3, tables: [{ total_steps: 7, completed_steps: 3 }] } as CodeGenTask);
  assert.equal(running.completed_steps, 3);
  assert.equal(running.tables[0].completed_steps, 3);
});
