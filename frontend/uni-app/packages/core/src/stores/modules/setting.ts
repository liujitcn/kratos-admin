import { defConfigService } from '../../api/base/v1/config'
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { BaseConfigSite } from '../../rpc/base/v1/config'
import { t } from '../../locales'

const REQUIRED_APP_CONFIGS = [
  { key: 'serviceProtocol', nameKey: 'core.protocol.service' },
  { key: 'privacyProtocol', nameKey: 'core.protocol.privacy' },
  { key: 'captchaType', nameKey: 'core.login.captcha_type_name' },
] as const

export const useSettingStore = defineStore('setting', () => {
  const data = ref<Map<string, string>>()
  let loading: Promise<void> | undefined

  /** 读取移动端配置项。 */
  const getData = (key: string): string | undefined => {
    return data.value?.get(key)
  }

  /** 加载并校验移动端必需配置。 */
  const loadData = async () => {
    if (loading) {
      return loading
    }

    const request = (async () => {
      const res = await defConfigService.GetConfig({
        site: BaseConfigSite.BASE_CONFIG_SITE_APP,
      })
      const nextData = new Map<string, string>()
      res.configs.forEach((item) => {
        nextData.set(item.key, item.value)
      })

      const missingConfigs = REQUIRED_APP_CONFIGS.filter(({ key }) => !nextData.get(key))
      if (missingConfigs.length) {
        throw new Error(
          t('core.config.missing', {
            names: missingConfigs.map(({ nameKey }) => t(nameKey)).join(', '),
          }),
        )
      }

      data.value = nextData
    })()
    loading = request

    try {
      await request
    } finally {
      loading = undefined
    }
  }

  return {
    getData,
    loadData,
  }
})
