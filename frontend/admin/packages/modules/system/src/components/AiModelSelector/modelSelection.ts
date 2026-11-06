import type { AiProviderModelOption } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_provider";

/** 用户最近一次选择的供应商与模型，仅保存在浏览器本地。 */
export interface AiModelSelection {
  provider_id: number;
  model_name: string;
}

const MODEL_SELECTION_STORAGE_KEY = "kratos-admin:ai:model-selection";

/** 恢复仍可用的缓存选择，否则使用首个可用模型。 */
export function resolveModelSelection(providers: AiProviderModelOption[]): AiModelSelection {
  const stored = loadStoredModelSelection();
  if (
    stored &&
    providers.some(
      provider =>
        provider.provider_id === stored.provider_id && provider.models.some(model => model.model_name === stored.model_name)
    )
  ) {
    return stored;
  }
  const provider = providers.find(item => item.models.length);
  return { provider_id: provider?.provider_id ?? 0, model_name: provider?.models[0]?.model_name ?? "" };
}

/** 读取本地缓存的模型选择，缓存缺失或损坏时返回 undefined。 */
export function loadStoredModelSelection(): AiModelSelection | undefined {
  try {
    const raw = localStorage.getItem(MODEL_SELECTION_STORAGE_KEY);
    if (!raw) return undefined;
    const parsed = JSON.parse(raw) as Partial<AiModelSelection>;
    if (typeof parsed.provider_id === "number" && typeof parsed.model_name === "string" && parsed.model_name) {
      return { provider_id: parsed.provider_id, model_name: parsed.model_name };
    }
  } catch {
    // 缓存损坏或存储不可用时忽略，回退到默认选择。
  }
  return undefined;
}

/** 持久化模型选择，存储不可用时静默失败，选择仍对当前会话生效。 */
export function saveModelSelection(selection: AiModelSelection): void {
  try {
    localStorage.setItem(MODEL_SELECTION_STORAGE_KEY, JSON.stringify(selection));
  } catch {
    // 忽略存储失败，不影响当前会话内的选择。
  }
}
