import { create } from 'zustand'
import { defConfigService } from '../api/base/config'
import { BaseConfigSite } from '../rpc/base/v1/enum'

const REQUIRED_APP_CONFIGS = [
  { key: 'serviceProtocol', name: '服务条款' },
  { key: 'privacyProtocol', name: '隐私协议' },
  { key: 'captchaType', name: '登录验证码类型' },
] as const

/** 应用配置状态及操作。 */
export interface SettingStoreState {
  data?: Map<string, string>
  getData: (key: string) => string | undefined
  loadData: () => Promise<void>
}

let loading: Promise<void> | undefined

/** 应用配置 Zustand Store。 */
export const useSettingStore = create<SettingStoreState>((set, get) => ({
  data: undefined,
  getData(key) {
    return get().data?.get(key)
  },
  async loadData() {
    if (loading) return loading
    loading = (async () => {
      const response = await defConfigService.GetConfig({ site: BaseConfigSite.APP })
      const nextData = new Map(response.configs.map((item) => [item.key, item.value]))
      const missing = REQUIRED_APP_CONFIGS.filter(({ key }) => !nextData.get(key))
      if (missing.length) {
        throw new Error(`移动端配置缺失：${missing.map(({ name }) => name).join('、')}`)
      }
      set({ data: nextData })
    })()
    try {
      await loading
    } finally {
      loading = undefined
    }
  },
}))
