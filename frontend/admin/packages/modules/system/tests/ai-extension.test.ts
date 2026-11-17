import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import { test } from "node:test";
import { runInNewContext } from "node:vm";
import ts from "typescript";

for (const order of ["flow-first", "bi-first"]) {
  test(`AI 流程卡片与 BI 建卡在 ${order} 注册顺序下同时可用`, () => {
    const { core, ai } = createRegistry();
    const flowBlocks = {};
    const biCardDialog = {};
    const flow = Object.freeze({ flowBlocks });
    const bi = Object.freeze({ biCardDialog });
    const modules = [
      { name: "flow", extensions: { [ai.ADMIN_AI_EXTENSION]: flow } },
      { name: "bi", extensions: { [ai.ADMIN_AI_EXTENSION]: bi } }
    ];
    core.registerAdminModules(order === "flow-first" ? modules : modules.reverse());
    core.registerAdminModule({ name: "unrelated", extensions: { other: {} } });

    const extension = ai.getAdminAiExtension();
    assert.equal(extension.flowBlocks, flowBlocks);
    assert.equal(extension.biCardDialog, biCardDialog);
    assert.deepEqual(Object.keys(flow), ["flowBlocks"]);
    assert.deepEqual(Object.keys(bi), ["biCardDialog"]);
  });
}

test("AI 扩展同一字段由后注册模块覆盖，同时保留其他字段", () => {
  const { core, ai } = createRegistry();
  const original = {};
  const replacement = {};
  const biCardDialog = {};
  core.registerAdminModules([
    { name: "first", extensions: { [ai.ADMIN_AI_EXTENSION]: { flowBlocks: original, biCardDialog } } },
    { name: "override", extensions: { [ai.ADMIN_AI_EXTENSION]: { flowBlocks: replacement } } }
  ]);

  const extension = ai.getAdminAiExtension();
  assert.equal(extension.flowBlocks, replacement);
  assert.equal(extension.biCardDialog, biCardDialog);
});

test("AI 流程渲染器按模块组合，并兼容旧版单组件扩展", () => {
  const { core, ai } = createRegistry();
  const legacy = {};
  const orders = {};
  const dashboard = {};
  core.registerAdminModules([
    { name: "legacy", extensions: { [ai.ADMIN_AI_EXTENSION]: { flowBlocks: legacy } } },
    {
      name: "bi",
      extensions: {
        [ai.ADMIN_AI_EXTENSION]: {
          flowBlockRenderers: [{ key: "orders", component: orders, types: ["order"] }]
        }
      }
    },
    {
      name: "dashboard",
      extensions: {
        [ai.ADMIN_AI_EXTENSION]: {
          flowBlockRenderers: [{ key: "summary", component: dashboard, types: ["summary"] }]
        }
      }
    }
  ]);

  const renderers = ai.getAdminAiFlowBlockRenderers();
  assert.equal(renderers.map(item => item.key).join(","), "legacy:flowBlocks,bi:orders,dashboard:summary");
  assert.equal(renderers[0].component, legacy);
  assert.equal(renderers[1].types?.join(","), "order");
  assert.equal(renderers[2].component, dashboard);
  assert.equal(ai.getAdminAiExtension().flowBlockRenderers?.length, 3);
});

test("未提供 AI 扩展时返回空能力集合", () => {
  const { core, ai } = createRegistry();
  core.registerAdminModule({ name: "system" });
  assert.equal(Object.keys(ai.getAdminAiExtension()).length, 0);
});

/** 为每个用例创建真实模块注册表，避免注册状态相互污染。 */
function createRegistry() {
  const runtime = loadModule("../../core/src/runtime-messages.ts");
  const core = loadModule("../../core/src/modules/index.ts", { "../runtime-messages.js": runtime });
  const ai = loadModule("src/ai.ts", { "@liujitcn/kratos-admin-core": core });
  return { core, ai };
}

/** 在内存中加载实际 TypeScript 源码，复用真实注册逻辑并避免加载 UI 包入口。 */
function loadModule(path: string, imports: Record<string, unknown> = {}) {
  const source = readFileSync(join(process.cwd(), path), "utf8");
  const exports: Record<string, any> = {};
  runInNewContext(ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS } }).outputText, {
    exports,
    process,
    require: (name: string) => {
      assert.ok(Object.hasOwn(imports, name), `未配置源码依赖: ${name}`);
      return imports[name];
    }
  });
  return exports;
}
