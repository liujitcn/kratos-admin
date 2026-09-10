import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { test } from "node:test";
import { runInNewContext } from "node:vm";
import ts from "typescript";

/** 驱动真实 Vite 插件的消息通道，验证连续文件更新只刷新一次。 */
test("代码生成期间合并热更新，关闭最后一个生成客户端后刷新一次", async () => {
  const source = await readFile("build/codegen-hmr.ts", "utf8");
  const exports: Record<string, any> = {};
  runInNewContext(ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS } }).outputText, { exports });
  const handlers = new Map<string, Function>();
  const sent: any[] = [];
  const ws = { send: (payload: any) => sent.push(payload), on: (event: string, handler: Function) => handlers.set(event, handler) };
  exports.codegenHmrPlugin().configureServer({ ws });
  const client = { send: () => undefined, socket: { once: () => undefined } };
  handlers.get("admin:codegen-hold")!({}, client);
  ws.send({ type: "update" });
  ws.send({ type: "full-reload" });
  ws.send({ type: "update" });
  assert.equal(sent.length, 0);
  ws.send({ type: "custom", event: "progress" });
  assert.equal(sent.length, 1);
  handlers.get("admin:codegen-release")!({}, client);
  assert.equal(sent.length, 2);
  assert.equal(sent[1].type, "full-reload");
  handlers.get("admin:codegen-release")!({}, client);
  assert.equal(sent.length, 2);
  ws.send({ type: "update" });
  assert.equal(sent.length, 3);
});
