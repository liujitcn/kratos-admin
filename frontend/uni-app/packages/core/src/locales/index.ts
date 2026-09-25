/**
 * uni-app 国际化运行时：合并 core 和业务模块语言包，并向 Vue 页面、请求工具暴露统一翻译入口。
 * 各语言 JSON 由模块定义文件导入，不能在 JSON 文件内添加注释。
 */
import { reactive, readonly, ref } from 'vue'
import type { KratosAppModule } from '../module'
import type { I18nCustomItem } from '../rpc/base/v1/config'
import type { OptionLanguageResponse } from '../rpc/base/v1/language'
import {
  DEFAULT_LOCALE as GENERATED_DEFAULT_LOCALE,
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
  /** 本地语言名称。 */
  native_name: string
}

const DEFAULT_LOCALE: SupportedLocale = GENERATED_DEFAULT_LOCALE
const LOCALE_STORAGE_KEY = 'kratos-app:locale'
const defaultLocaleMessages = new Map<SupportedLocale, LocaleMessages>()
const localeMessages = reactive(new Map<SupportedLocale, LocaleMessages>())
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

/** 应用后端语言配置，并在当前语言不可用时回退到接口第一项。 */
export function applyLanguageConfig(response: OptionLanguageResponse): void {
  const options = response.languages.reduce<LocaleOption[]>((items, item) => {
    const languageCode = parseSupportedLocale(item.language_code)
    if (!languageCode || items.some((option) => option.language_code === languageCode)) return items
    items.push({
      language_code: languageCode,
      native_name: item.native_name || languageCode,
    })
    return items
  }, [])
  mutableLanguageOptions.value = options.length ? options : getFallbackLanguageOptions()
  const availableLocales = getSupportedLocales()
  if (!availableLocales.includes(mutableLocaleState.value)) {
    const locale = availableLocales[0] ?? DEFAULT_LOCALE
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
  defaultLocaleMessages.clear()
  SUPPORTED_LOCALES.forEach((locale) => defaultLocaleMessages.set(locale, {}))
  modules.forEach((module) => {
    const expectedKeys = requiredLocaleKeys(module.messages?.[DEFAULT_LOCALE] || {})
    SUPPORTED_LOCALES.forEach((locale) => {
      const messages = module.messages?.[locale]
      if (!messages)
        throw new Error(localeRuntimeMessage('missing_bundle', { module: module.name, locale }))
      const keys = requiredLocaleKeys(messages)
      if (keys.join('\u0000') !== expectedKeys.join('\u0000')) {
        throw new Error(localeRuntimeMessage('key_set_mismatch', { module: module.name, locale }))
      }
      const target = defaultLocaleMessages.get(locale) as LocaleMessages
      Object.keys(messages).forEach((key) => {
        if (!isAllowedLocaleKey(module.name, key)) {
          throw new Error(localeRuntimeMessage('namespace_invalid', { module: module.name, key }))
        }
        if (Object.prototype.hasOwnProperty.call(target, key)) {
          throw new Error(localeRuntimeMessage('duplicate_key', { locale, key }))
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
  applyCustomLocaleMessages([])
}

/** 按语言和已有 key 覆盖本地文案，每次重新应用以清除已删除的自定义配置。 */
export function applyCustomLocaleMessages(customs: I18nCustomItem[]): void {
  SUPPORTED_LOCALES.forEach((locale) => {
    const messages = { ...(defaultLocaleMessages.get(locale) ?? {}) }
    customs.forEach((item) => {
      if (parseSupportedLocale(item.locale) !== locale || !item.key || !item.value) return
      if (!Object.prototype.hasOwnProperty.call(messages, item.key)) return
      messages[item.key] = item.value
    })
    localeMessages.set(locale, messages)
  })
}

/** 校验公共键和当前业务模块专属的语言键命名空间。 */
function isAllowedLocaleKey(moduleName: string, key: string): boolean {
  const packageName = moduleName.split('/').at(-1) ?? moduleName
  const moduleNamespace = packageName.split('-').at(-1) ?? packageName
  return ['common', 'core', 'system', moduleNamespace].some((namespace) =>
    key.startsWith(`${namespace}.`),
  )
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
    throw new Error(localeRuntimeMessage('placeholder_mismatch', { module: moduleName, key }))
  }
}

function localeRuntimeMessage(key: string, params: Record<string, string>): string {
  const locale = getCurrentLocale().toLowerCase().startsWith('zh') ? 'zh-CN' : 'en-US'
  const templates: Record<string, Record<string, string>> = {
    'zh-CN': {
      missing_bundle: '{module} 缺少 {locale} 语言包',
      key_set_mismatch: '{module} 的 {locale} 语言包键集合不一致',
      namespace_invalid: '{module} 的语言键命名空间无效: {key}',
      duplicate_key: '{locale} 语言键重复: {key}',
      placeholder_mismatch: '{module} 的 {key} 占位符集合不一致',
    },
    'en-US': {
      missing_bundle: '{module} is missing the {locale} locale bundle',
      key_set_mismatch: '{module} has inconsistent keys in the {locale} locale bundle',
      namespace_invalid: 'Invalid locale key namespace for {module}: {key}',
      duplicate_key: 'Duplicate locale key in {locale}: {key}',
      placeholder_mismatch: 'Placeholder set mismatch for {module}: {key}',
    },
  }
  return (templates[locale][key] ?? key).replace(
    /\{([A-Za-z0-9_]+)\}/g,
    (_, name: string) => params[name] ?? `{${name}}`,
  )
}

function requiredLocaleKeys(messages: LocaleMessages): string[] {
  return Object.keys(messages)
    .filter((key) => !key.startsWith('common.language.'))
    .sort()
}

function getFallbackLanguageOptions(): LocaleOption[] {
  return SUPPORTED_LOCALES.map((languageCode) => ({
    language_code: languageCode,
    native_name: fallbackNativeLanguageName(languageCode),
  }))
}

function fallbackNativeLanguageName(languageCode: SupportedLocale): string {
  return {
    'zh-CN': '简体中文',
    'zh-TW': '繁體中文',
    'en-US': 'English',
    'ja-JP': '日本語',
  }[languageCode]
}
