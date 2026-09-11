import assert from "node:assert/strict";
import { mkdtemp, mkdir, writeFile, rm, readFile } from "node:fs/promises";
import { test } from "node:test";
import { runInNewContext } from "node:vm";
import ts from "typescript";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { setTimeout } from "node:timers/promises";
import { createServer } from "vite";
import { codegenHmrPlugin } from "../build/codegen-hmr.js";

/** 驱动真实 Vite 插件的消息通道，验证连续文件更新只刷新一次。 */
test("代码生成期间合并热更新，关闭最后一个生成客户端后刷新一次", async () => {
  const source = await readFile("build/codegen-hmr.ts", "utf8");
  const exports: Record<string, any> = {};
  runInNewContext(ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS } }).outputText, { exports });
  const handlers = new Map<string, Function>();
  const sent: any[] = [];
  const ws = {
    send: (payload: any) => sent.push(payload),
    on: (event: string, handler: Function) => handlers.set(event, handler)
  };
  exports.codegenHmrPlugin().configureServer({ ws, watcher: { add: () => undefined } });
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

/** 验证真实 Vite 能发现宿主目录之外生成的新页面。 */
test("外部业务模块新增多级目录后自动更新页面映射", async () => {
  const root = await mkdtemp(join(tmpdir(), "admin-codegen-"));
  const app = join(root, "app");
  const sourceRoot = join(root, "module/src");
  await mkdir(app, { recursive: true });
  await mkdir(join(sourceRoot, "views/base"), { recursive: true });
  await writeFile(join(sourceRoot, "views/base/index.vue"), "<template>base</template>");
  await writeFile(join(sourceRoot, "module.js"), 'export const views = import.meta.glob("./views/**/*.vue");');
  const server = await createServer({
    configFile: false,
    root: app,
    logLevel: "silent",
    server: { port: 0, fs: { allow: [root] } },
    optimizeDeps: { noDiscovery: true },
    plugins: [codegenHmrPlugin([sourceRoot])]
  });
  try {
    await server.listen();
    const url = `/@fs${join(sourceRoot, "module.js")}`;
    assert.ok(!(await server.transformRequest(url))?.code.includes("./views/tenant/project/index.vue"));
    await setTimeout(300);
    await mkdir(join(sourceRoot, "views/tenant/project"), { recursive: true });
    await writeFile(join(sourceRoot, "views/tenant/project/index.vue"), "<template>project</template>");
    let resolved = false;
    for (let attempt = 0; attempt < 30; attempt++) {
      await setTimeout(100);
      resolved = (await server.transformRequest(url))?.code.includes("./views/tenant/project/index.vue") ?? false;
      if (resolved) break;
    }
    assert.ok(resolved, "新页面未进入映射，动态路由将回退到建设中页面");
  } finally {
    await server.close();
    await rm(root, { recursive: true, force: true });
  }
});
