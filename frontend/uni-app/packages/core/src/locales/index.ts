/**
 * uni-app 国际化运行时：合并 core 和业务模块语言包，并向 Vue 页面、请求工具暴露统一翻译入口。
 * 各语言 JSON 由模块定义文件导入，不能在 JSON 文件内添加注释。
 */
import { readonly, ref } from 'vue'
import type { KratosAppModule } from '../module'
import type { GetLanguageResponse } from '../rpc/base/v1/language'
import {
  DEFAULT_LOCALE as GENERATED_DEFAULT_LOCALE,
  LOCALE_MESSAGES,
  SUPPORTED_LOCALES as GENERATED_SUPPORTED_LOCALES,
  type GeneratedLocale,
} from './generated'

/** 移动端已打包的语言区域；运行时可切换列表由 base_language 接口决定。 */
export const SUPPORTED_LOCALES = GENERATED_SUPPORTED_LOCALES
/** 移动端支持的语言区域类型。 */
export type SupportedLocale = GeneratedLocale
/** 单个模块的扁平语言包。 */
export type LocaleMessages = Record<string, string>
/** 翻译插值参数。 */
export type LocaleParams = Record<string, string | number>

/** 运行时语言选项。 */
export interface LocaleOption {
  /** 标准语言代码。 */
  language_code: SupportedLocale
  /** 语言名称。 */
  language_name: string
  /** 本地语言名称。 */
  native_name: string
  /** 排序值。 */
  sort: number
}

const DEFAULT_LOCALE: SupportedLocale = GENERATED_DEFAULT_LOCALE
const LOCALE_STORAGE_KEY = 'kratos-app:locale'
const localeMessages = new Map<SupportedLocale, LocaleMessages>()
const localeChangeHandlers = new Set<() => void | Promise<void>>()
const mutableLocaleState = ref<SupportedLocale>(DEFAULT_LOCALE)
const mutableLanguageOptions = ref<LocaleOption[]>([])

/** 响应式当前语言区域。 */
export const localeState = readonly(mutableLocaleState)

/** 规范化语言区域到应用白名单。 */
export function normalizeLocale(value?: string): SupportedLocale {
  return parseSupportedLocale(value) ?? DEFAULT_LOCALE
}

/** 将接口或系统语言代码解析为已打包的语言区域。 */
function parseSupportedLocale(value?: string): SupportedLocale | undefined {
  const normalized = String(value || '')
    .replace('_', '-')
    .toLowerCase()
  if (!normalized) return undefined
  const alias =
    normalized.startsWith('zh-hk') || normalized.startsWith('zh-mo') ? 'zh-tw' : normalized
  const exact = SUPPORTED_LOCALES.find((locale) => locale.toLowerCase() === alias)
  if (exact) return exact
  const language = alias.split('-', 1)[0]
  return SUPPORTED_LOCALES.find((locale) => locale.toLowerCase().split('-', 1)[0] === language)
}

/** 初始化持久化语言偏好。 */
export function initializeLocale(): SupportedLocale {
  const stored = uni.getStorageSync(LOCALE_STORAGE_KEY) as string | undefined
  const systemLanguage = uni.getSystemInfoSync().language
  mutableLocaleState.value = normalizeLocale(stored || systemLanguage)
  if (typeof uni.setLocale === 'function') uni.setLocale(mutableLocaleState.value)
  return mutableLocaleState.value
}

/** 应用后端语言配置，并在当前语言不可用时回退到接口主语言。 */
export function applyLanguageConfig(response: GetLanguageResponse): void {
  const options = response.languages.reduce<LocaleOption[]>((items, item) => {
    const languageCode = parseSupportedLocale(item.language_code)
    if (!languageCode || items.some((option) => option.language_code === languageCode)) return items
    items.push({
      language_code: languageCode,
      language_name: item.language_name || fallbackLanguageName(languageCode),
      native_name: item.native_name || item.language_name || languageCode,
      sort: item.sort,
    })
    return items
  }, [])
  mutableLanguageOptions.value = options.length
    ? options.sort((left, right) => left.sort - right.sort)
    : getFallbackLanguageOptions()
  const availableLocales = getSupportedLocales()
  const primaryLocale = parseSupportedLocale(response.primary_language_code)
  if (!availableLocales.includes(mutableLocaleState.value)) {
    const locale =
      primaryLocale && availableLocales.includes(primaryLocale)
        ? primaryLocale
        : (availableLocales[0] ?? DEFAULT_LOCALE)
    mutableLocaleState.value = locale
    if (typeof uni.setLocale === 'function') uni.setLocale(locale)
  }
}

/** 获取当前接口配置的语言选项。 */
export function getLanguageOptions(): LocaleOption[] {
  return mutableLanguageOptions.value.length
    ? mutableLanguageOptions.value
    : getFallbackLanguageOptions()
}

/** 获取当前可切换的语言区域。 */
export function getSupportedLocales(): SupportedLocale[] {
  return getLanguageOptions().map((item) => item.language_code)
}

/** 注册所有模块贡献的语言包并校验语言键集合。 */
export function registerLocaleMessages(modules: KratosAppModule[]): void {
  localeMessages.clear()
  SUPPORTED_LOCALES.forEach((locale) => localeMessages.set(locale, {}))
  modules.forEach((module) => {
    const expectedKeys = requiredLocaleKeys(module.messages?.[DEFAULT_LOCALE] || {})
    SUPPORTED_LOCALES.forEach((locale) => {
      const messages = module.messages?.[locale]
      if (!messages) throw new Error(`${module.name} 缺少 ${locale} 语言包`)
      const keys = requiredLocaleKeys(messages)
      if (keys.join('\u0000') !== expectedKeys.join('\u0000')) {
        throw new Error(`${module.name} 的 ${locale} 语言包键集合不一致`)
      }
      const target = localeMessages.get(locale) as LocaleMessages
      Object.keys(messages).forEach((key) => {
        if (!key.startsWith('common.') && !key.startsWith('core.') && !key.startsWith('system.')) {
          throw new Error(`${module.name} 的语言键命名空间无效: ${key}`)
        }
        if (Object.prototype.hasOwnProperty.call(target, key)) {
          throw new Error(`${locale} 语言键重复: ${key}`)
        }
        assertLocalePlaceholders(
          module.name,
          key,
          messages[key],
          module.messages?.[DEFAULT_LOCALE]?.[key] || '',
        )
        target[key] = messages[key]
      })
    })
  })
}

/** 获取当前语言区域。 */
export function getCurrentLocale(): SupportedLocale {
  return mutableLocaleState.value
}

/** 获取请求需要携带的语言头。 */
export function getLocaleRequestHeaders(): Record<'Accept-Language', SupportedLocale> {
  return { 'Accept-Language': getCurrentLocale() }
}

/** 切换当前语言并通知需要刷新动态本地化数据的模块。 */
export async function setCurrentLocale(value: SupportedLocale): Promise<void> {
  const locale = normalizeLocale(value)
  if (!getSupportedLocales().includes(locale)) return
  if (locale === mutableLocaleState.value) return
  mutableLocaleState.value = locale
  uni.setStorageSync(LOCALE_STORAGE_KEY, locale)
  if (typeof uni.setLocale === 'function') uni.setLocale(locale)
  for (const handler of localeChangeHandlers) await handler()
}

/** 注册语言切换后的动态数据刷新处理器。 */
export function registerLocaleChangeHandler(handler: () => void | Promise<void>): () => void {
  localeChangeHandlers.add(handler)
  return () => localeChangeHandlers.delete(handler)
}

/** 翻译稳定语言键，缺失时回退中文且不展示裸键。 */
export function t(key: string, params: LocaleParams = {}): string {
  const message =
    localeMessages.get(getCurrentLocale())?.[key] || localeMessages.get(DEFAULT_LOCALE)?.[key]
  if (!message) return localeMessages.get(DEFAULT_LOCALE)?.['common.message.unknown'] || ''
  return message.replace(/\{([A-Za-z0-9_]+)\}/g, (_, name: string) =>
    String(params[name] ?? `{${name}}`),
  )
}

/** 在 Vue 页面中使用响应式国际化能力。 */
export function useI18n() {
  return {
    locale: localeState,
    setLocale: setCurrentLocale,
    t,
  }
}

function assertLocalePlaceholders(
  moduleName: string,
  key: string,
  message: string,
  sourceMessage: string,
): void {
  const placeholders = (value: string) =>
    [...value.matchAll(/\{([A-Za-z0-9_]+)\}/g)].map((match) => match[1]).sort()
  if (placeholders(message).join('\u0000') !== placeholders(sourceMessage).join('\u0000')) {
    throw new Error(`${moduleName} 的 ${key} 占位符集合不一致`)
  }
}

function requiredLocaleKeys(messages: LocaleMessages): string[] {
  return Object.keys(messages)
    .filter((key) => !key.startsWith('common.language.'))
    .sort()
}

function getFallbackLanguageOptions(): LocaleOption[] {
  return SUPPORTED_LOCALES.map((languageCode, sort) => ({
    language_code: languageCode,
    language_name: fallbackLanguageName(languageCode),
    native_name: languageCode,
    sort,
  }))
}

function fallbackLanguageName(languageCode: SupportedLocale): string {
  const key = `common.language.${languageCode}`
  const messages = LOCALE_MESSAGES as Record<string, Record<string, string>>
  const currentMessage = messages[getCurrentLocale()]?.[key]
  const defaultMessage = messages[DEFAULT_LOCALE]?.[key]
  const anyMessage = Object.values(messages)
    .map((item) => item[key])
    .find(Boolean)
  return currentMessage || defaultMessage || anyMessage || languageCode
}
