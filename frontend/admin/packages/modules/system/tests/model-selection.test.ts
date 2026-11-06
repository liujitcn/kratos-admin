import assert from "node:assert/strict";
import { test } from "node:test";
import { resolveModelSelection, saveModelSelection } from "../src/components/AiModelSelector/modelSelection.js";

test("AI 助手与 BI 共用选择缓存，并按供应商区分同名模型", () => {
  const previous = Object.getOwnPropertyDescriptor(globalThis, "localStorage");
  const storage = new Map<string, string>();
  Object.defineProperty(globalThis, "localStorage", {
    configurable: true,
    value: { getItem: (key: string) => storage.get(key), setItem: (key: string, value: string) => storage.set(key, value) }
  });
  try {
    const providers = [
      { provider_id: 1, name: "供应商一", models: [{ model_name: "same-model", display_name: "模型一" }] },
      { provider_id: 2, name: "供应商二", models: [{ model_name: "same-model", display_name: "模型二" }] }
    ];
    assert.deepEqual(resolveModelSelection(providers), { provider_id: 1, model_name: "same-model" });
    saveModelSelection({ provider_id: 2, model_name: "same-model" });
    assert.deepEqual(resolveModelSelection(providers), { provider_id: 2, model_name: "same-model" });
    assert.deepEqual(resolveModelSelection(providers.slice(0, 1)), { provider_id: 1, model_name: "same-model" });
    assert.deepEqual(resolveModelSelection([]), { provider_id: 0, model_name: "" });
    storage.clear();
    assert.deepEqual(resolveModelSelection([{ provider_id: 3, name: "无可用模型", models: [] }, ...providers]), {
      provider_id: 1,
      model_name: "same-model"
    });
    storage.set("kratos-admin:ai:model-selection", "invalid JSON");
    assert.deepEqual(resolveModelSelection(providers), { provider_id: 1, model_name: "same-model" });
  } finally {
    if (previous) Object.defineProperty(globalThis, "localStorage", previous);
    else Reflect.deleteProperty(globalThis, "localStorage");
  }
});
