import Taro from '@tarojs/taro'

/** 用户最近一次选择的供应商与模型，仅保存在设备本地。 */
export interface AiModelSelection {
  provider_id: number
  model_name: string
}

const MODEL_SELECTION_STORAGE_KEY = 'kratos-taro-app:ai:model-selection'

/** 读取本地缓存的模型选择，缓存缺失或损坏时返回 undefined。 */
export function loadStoredModelSelection(): AiModelSelection | undefined {
  try {
    const raw = Taro.getStorageSync<unknown>(MODEL_SELECTION_STORAGE_KEY)
    if (!raw) {
      return undefined
    }
    const parsed = typeof raw === 'string' ? JSON.parse(raw) : raw
    const value = parsed as Partial<AiModelSelection>
    if (
      typeof value.provider_id === 'number' &&
      typeof value.model_name === 'string' &&
      value.model_name
    ) {
      return { provider_id: value.provider_id, model_name: value.model_name }
    }
  } catch {
    // 缓存损坏或存储不可用时忽略，回退到默认选择。
  }
  return undefined
}

/** 持久化模型选择，存储不可用时静默失败，选择仍对当前会话生效。 */
export function saveModelSelection(selection: AiModelSelection): void {
  try {
    Taro.setStorageSync(MODEL_SELECTION_STORAGE_KEY, JSON.stringify(selection))
  } catch {
    // 忽略存储失败，不影响当前会话内的选择。
  }
}
