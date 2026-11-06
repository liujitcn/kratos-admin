import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { test } from "node:test";
import { fileURLToPath } from "node:url";
import { runInNewContext } from "node:vm";
import ts from "typescript";
import * as vue from "vue";

const require = createRequire(import.meta.url);
const { parse, compileScript } = require("vue/compiler-sfc");
const filename = fileURLToPath(new URL("../src/views/query/index.vue", import.meta.url));

for (const action of ["ask", "selectRecord"]) {
  test(`${action}未完成时离开缓存页会结束加载，迟到响应不能覆盖新请求`, async () => {
    let deactivate;
    const pending = Promise.withResolvers();
    let calls = 0;
    const result = { title: "new result", result: { query_record_id: 1 } };
    const request = () => ++calls === 1 ? pending.promise : Promise.resolve(result);
    const mocks = {
      vue: { ...vue, onMounted: () => undefined, onBeforeUnmount: () => undefined, onDeactivated: callback => { deactivate = callback; } },
      "vue-router": { useRoute: () => ({ query: {} }), useRouter: () => ({}) },
      "@liujitcn/kratos-admin-core": { t: key => key, useSpeechRecognition: () => ({ loading: vue.ref(false), stop: () => undefined }) },
      "@liujitcn/kratos-admin-system/components/AiModelSelector/modelSelection": {},
      "@liujitcn/kratos-admin-system/api/base/v1/ai_provider": {},
      "../../bi-response": {},
      "../../api/bi/admin/v1/bi_query": { defBiQueryService: { AskBiQuery: request } },
      "../../api/bi/admin/v1/bi_query_record": { defBiQueryRecordService: { GetBiQuery: request, PageBiQuery: async () => ({ queries: [] }) } }
    };
    const { descriptor } = parse(readFileSync(filename, "utf8"), { filename });
    const script = compileScript(descriptor, { id: "bi-query", genDefaultAs: "component" });
    const module = { exports: {} };
    runInNewContext(ts.transpileModule(`${script.content}\nmodule.exports = component;`, {
      compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 }
    }).outputText, {
      module, exports: module.exports, setTimeout, clearTimeout, Promotion: {}, Microphone: {}, Loading: {},
      require: name => {
        if (name.endsWith(".vue")) return {};
        assert.ok(Object.hasOwn(mocks, name), `未配置测试依赖: ${name}`);
        return mocks[name];
      }
    });
    const state = module.exports.setup({}, { expose: () => undefined });
    state.loadingModels.value = false;
    state.question.value = "question";
    const first = state[action](1);
    assert.equal(state.loading.value, true);
    deactivate();
    assert.equal(state.loading.value, false);
    state.question.value = "new question";
    await state.ask();
    assert.equal(state.response.value.title, "new result");
    pending.resolve({ title: "old result" });
    await first;
    assert.equal(state.loading.value, false);
    assert.equal(state.response.value.title, "new result");
    assert.equal(calls, 2);
  });
}
