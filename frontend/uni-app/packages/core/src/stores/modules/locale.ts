import { defineStore } from 'pinia'
import { computed } from 'vue'
import {
  getCurrentLocale,
  getLanguageOptions,
  getSupportedLocales,
  localeState,
  setCurrentLocale,
  type SupportedLocale,
} from '../../locales'

/** 移动端语言状态。 */
export const useLocaleStore = defineStore('locale', () => {
  const locale = computed(() => localeState.value)

  /** 切换当前语言并持久化。 */
  const setLocale = async (value: SupportedLocale) => {
    await setCurrentLocale(value)
  }

  return {
    locale,
    currentLocale: getCurrentLocale,
    languageOptions: computed(() => getLanguageOptions()),
    supportedLocales: computed(() => getSupportedLocales()),
    setLocale,
  }
})
