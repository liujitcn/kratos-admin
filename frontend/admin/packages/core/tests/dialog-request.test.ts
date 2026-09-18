import assert from "node:assert/strict";
import test from "node:test";
import { createDialogRequestController } from "../src/components/Dialog/interface/index.js";

test("弹窗只提交最新一次异步打开请求", async () => {
  const controller = createDialogRequestController();
  const commits: number[] = [];
  let resolveFirst!: (value: number) => void;
  let resolveSecond!: (value: number) => void;
  const first = new Promise<number>(resolve => {
    resolveFirst = resolve;
  });
  const second = new Promise<number>(resolve => {
    resolveSecond = resolve;
  });

  const firstOpen = controller.open({ load: () => first, commit: value => commits.push(value) });
  const secondOpen = controller.open({ load: () => second, commit: value => commits.push(value) });

  resolveFirst(1);
  assert.equal(await firstOpen, false);
  resolveSecond(2);
  assert.equal(await secondOpen, true);
  assert.deepEqual(commits, [2]);
});

test("失效弹窗打开请求不会提交结果", async () => {
  const controller = createDialogRequestController();
  const commits: number[] = [];
  let resolve!: (value: number) => void;
  const pending = new Promise<number>(resolveValue => {
    resolve = resolveValue;
  });

  const opening = controller.open({ load: () => pending, commit: value => commits.push(value) });
  controller.invalidate();
  resolve(1);

  assert.equal(await opening, false);
  assert.deepEqual(commits, []);
});

test("过期打开请求的异常不会影响当前请求", async () => {
  const controller = createDialogRequestController();
  const commits: number[] = [];
  let rejectFirst!: (reason?: unknown) => void;
  let resolveSecond!: (value: number) => void;
  const first = new Promise<number>((_, reject) => {
    rejectFirst = reject;
  });
  const second = new Promise<number>(resolve => {
    resolveSecond = resolve;
  });

  const firstOpen = controller.open({ load: () => first, commit: value => commits.push(value) });
  const secondOpen = controller.open({ load: () => second, commit: value => commits.push(value) });

  rejectFirst(new Error("stale request"));
  assert.equal(await firstOpen, false);
  resolveSecond(2);
  assert.equal(await secondOpen, true);
  assert.deepEqual(commits, [2]);
});
