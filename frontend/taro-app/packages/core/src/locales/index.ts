/**
 * Taro 国际化运行时：合并 core 和业务模块语言包，并向 React 页面、请求工具暴露统一翻译入口。
 * 各语言 JSON 由模块定义文件导入，不能在 JSON 文件内添加注释。
 */
import Taro from '@tarojs/taro'
import { create } from 'zustand'
import type { KratosTaroModule } from '../module'
import type { I18nCustomItem } from '../rpc/base/v1/config'
import type { OptionLanguageResponse } from '../rpc/base/v1/language'
import {
  DEFAULT_LOCALE as GENERATED_DEFAULT_LOCALE,
  SUPPORTED_LOCALES as GENERATED_SUPPORTED_LOCALES,
  type GeneratedLocale,
} from './generated'

/** Taro 端已打包的语言区域；运行时可切换列表由 base_language 接口决定。 */
export const SUPPORTED_LOCALES = GENERATED_SUPPORTED_LOCALES
/** Taro 端支持的语言区域类型。 */
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

/** Taro 语言状态。 */
export interface LocaleStoreState {
  locale: SupportedLocale
  languageOptions: LocaleOption[]
  supportedLocales: SupportedLocale[]
}

const DEFAULT_LOCALE: SupportedLocale = GENERATED_DEFAULT_LOCALE
const LOCALE_STORAGE_KEY = 'kratos-app:locale'
const defaultLocaleMessages = new Map<SupportedLocale, LocaleMessages>()
const localeMessages = new Map<SupportedLocale, LocaleMessages>()
const useLocaleMessagesRevision = create<number>(() => 0)
const localeChangeHandlers = new Set<() => void | Promise<void>>()

/** 响应式语言 Zustand Store。 */
export const useLocaleStore = create<LocaleStoreState>(() => ({
  locale: DEFAULT_LOCALE,
  languageOptions: [],
  supportedLocales: [...SUPPORTED_LOCALES],
}))

/** 规范化语言区域到应用白名单。 */
export function normalizeLocale(value?: string): SupportedLocale {
  return parseSupportedLocale(value) ?? DEFAULT_LOCALE
}

/** 将接口或系统语言代码解析为已打包的语言区域。 */
function parseSupportedLocale(value?: string): SupportedLocale | undefined {
  const normalized = String(value || '').replace('_', '-').toLowerCase()
  if (!normalized) return undefined
  const alias = normalized.startsWith('zh-hk') || normalized.startsWith('zh-mo') ? 'zh-tw' : normalized
  const exact = SUPPORTED_LOCALES.find((locale) => locale.toLowerCase() === alias)
  if (exact) return exact
  const language = alias.split('-', 1)[0]
  return SUPPORTED_LOCALES.find((locale) => locale.toLowerCase().split('-', 1)[0] === language)
}

/** 初始化持久化语言偏好。 */
export function initializeLocale(): SupportedLocale {
  const stored = Taro.getStorageSync<string>(LOCALE_STORAGE_KEY)
  const systemLanguage = Taro.getSystemInfoSync().language
  const locale = normalizeLocale(stored || systemLanguage)
  useLocaleStore.setState({ locale, languageOptions: getFallbackLanguageOptions(), supportedLocales: [...SUPPORTED_LOCALES] })
  return locale
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
  const languageOptions = options.length ? options : getFallbackLanguageOptions()
  const supportedLocales = languageOptions.map((item) => item.language_code)
  const currentLocale = getCurrentLocale()
  const locale = supportedLocales.includes(currentLocale)
    ? currentLocale
    : supportedLocales[0] ?? DEFAULT_LOCALE
  useLocaleStore.setState({ locale, languageOptions, supportedLocales })
  if (locale !== currentLocale) Taro.setStorageSync(LOCALE_STORAGE_KEY, locale)
}

/** 获取当前接口配置的语言选项。 */
export function getLanguageOptions(): LocaleOption[] {
  return useLocaleStore.getState().languageOptions.length ? useLocaleStore.getState().languageOptions : getFallbackLanguageOptions()
}

/** 获取当前可切换的语言区域。 */
export function getSupportedLocales(): SupportedLocale[] {
  return useLocaleStore.getState().supportedLocales
}

/** 注册所有模块贡献的语言包并校验语言键集合。 */
export function registerLocaleMessages(modules: KratosTaroModule[]): void {
  defaultLocaleMessages.clear()
  SUPPORTED_LOCALES.forEach((locale) => defaultLocaleMessages.set(locale, {}))
  modules.forEach((module) => {
    const expectedKeys = requiredLocaleKeys(module.messages?.[DEFAULT_LOCALE] || {})
    SUPPORTED_LOCALES.forEach((locale) => {
      const messages = module.messages?.[locale]
      if (!messages) throw new Error(localeRuntimeMessage('missing_bundle', { module: module.name, locale }))
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
        assertLocalePlaceholders(module.name, key, messages[key], module.messages?.[DEFAULT_LOCALE]?.[key] || '')
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
  useLocaleMessagesRevision.setState((revision) => revision + 1)
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
  return useLocaleStore.getState().locale
}

/** 获取请求需要携带的语言头。 */
export function getLocaleRequestHeaders(): Record<'Accept-Language', SupportedLocale> {
  return { 'Accept-Language': getCurrentLocale() }
}

/** 切换当前语言并通知需要刷新动态本地化数据的模块。 */
export async function setCurrentLocale(value: SupportedLocale): Promise<void> {
  const locale = normalizeLocale(value)
  if (!getSupportedLocales().includes(locale)) return
  if (locale === getCurrentLocale()) return
  useLocaleStore.setState({ locale })
  Taro.setStorageSync(LOCALE_STORAGE_KEY, locale)
  for (const handler of localeChangeHandlers) await handler()
}

/** 注册语言切换后的动态数据刷新处理器。 */
export function registerLocaleChangeHandler(handler: () => void | Promise<void>): () => void {
  localeChangeHandlers.add(handler)
  return () => localeChangeHandlers.delete(handler)
}

/** 翻译稳定语言键，缺失时回退中文且不展示裸键。 */
export function t(key: string, params: LocaleParams = {}, locale = getCurrentLocale()): string {
  const message = localeMessages.get(locale)?.[key] || localeMessages.get(DEFAULT_LOCALE)?.[key]
  if (!message) return localeMessages.get(DEFAULT_LOCALE)?.['common.message.unknown'] || ''
  return message.replace(/\{([A-Za-z0-9_]+)\}/g, (_, name: string) => String(params[name] ?? `{${name}}`))
}

/** 在 React 页面中使用响应式国际化能力。 */
export function useI18n() {
  useLocaleMessagesRevision()
  const locale = useLocaleStore((state) => state.locale)
  return {
    locale,
    setLocale: setCurrentLocale,
    t: (key: string, params: LocaleParams = {}) => t(key, params, locale),
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
    'zh-CN': { missing_bundle: '{module} 缺少 {locale} 语言包', key_set_mismatch: '{module} 的 {locale} 语言包键集合不一致', namespace_invalid: '{module} 的语言键命名空间无效: {key}', duplicate_key: '{locale} 语言键重复: {key}', placeholder_mismatch: '{module} 的 {key} 占位符集合不一致' },
    'en-US': { missing_bundle: '{module} is missing the {locale} locale bundle', key_set_mismatch: '{module} has inconsistent keys in the {locale} locale bundle', namespace_invalid: 'Invalid locale key namespace for {module}: {key}', duplicate_key: 'Duplicate locale key in {locale}: {key}', placeholder_mismatch: 'Placeholder set mismatch for {module}: {key}' },
  }
  return (templates[locale][key] ?? key).replace(/\{([A-Za-z0-9_]+)\}/g, (_, name: string) => params[name] ?? `{${name}}`)
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
