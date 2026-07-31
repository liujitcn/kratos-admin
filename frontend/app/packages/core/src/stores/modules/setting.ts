import { defConfigService } from '../../api/base/config'
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { BaseConfigSite } from '../../rpc/common/v1/enum'

const REQUIRED_APP_CONFIGS = [
  { key: 'serviceProtocol', name: '服务条款' },
  { key: 'privacyProtocol', name: '隐私协议' },
  { key: 'captchaType', name: '登录验证码类型' },
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
        site: BaseConfigSite.APP,
      })
      const nextData = new Map<string, string>()
      res.configs.forEach((item) => {
        nextData.set(item.key, item.value)
      })

      const missingConfigs = REQUIRED_APP_CONFIGS.filter(({ key }) => !nextData.get(key))
      if (missingConfigs.length) {
        throw new Error(`移动端配置缺失：${missingConfigs.map(({ name }) => name).join('、')}`)
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
