import { create } from 'zustand'
import { defConfigService } from '../api/base/v1/config'
import { t } from '../locales'
import { BaseConfigSite } from '../rpc/base/v1/config'

const REQUIRED_APP_CONFIGS = [
  { key: 'serviceProtocol', nameKey: 'core.login.service' },
  { key: 'privacyProtocol', nameKey: 'core.login.privacy' },
  { key: 'captchaType', nameKey: 'core.login.captcha_type_name' },
] as const

/** 应用配置状态及操作。 */
export interface SettingStoreState {
  data?: Map<string, string>
  aiEnabled: boolean
  getData: (key: string) => string | undefined
  loadData: () => Promise<void>
}

let loading: Promise<void> | undefined

/** 应用配置 Zustand Store。 */
export const useSettingStore = create<SettingStoreState>((set, get) => ({
  data: undefined,
  aiEnabled: false,
  getData(key) {
    return get().data?.get(key)
  },
  async loadData() {
    if (loading) return loading
    loading = (async () => {
      let response
      try {
        response = await defConfigService.GetConfig({ site: BaseConfigSite.BASE_CONFIG_SITE_APP })
      } catch {
        set({ aiEnabled: false })
        return
      }
      const nextData = new Map(response.configs.map((item) => [item.key, item.value]))
      const missing = REQUIRED_APP_CONFIGS.filter(({ key }) => !nextData.get(key))
      if (missing.length) {
        throw new Error(
          t('core.config.missing', {
            names: missing.map(({ nameKey }) => t(nameKey)).join(', '),
          }),
        )
      }
      set({ data: nextData, aiEnabled: response.ai_enabled === true })
    })()
    try {
      await loading
    } finally {
      loading = undefined
    }
  },
}))
