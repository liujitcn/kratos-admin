import Taro from '@tarojs/taro'
import { create } from 'zustand'
import type { KratosTaroModule } from '../module'

/** Taro 端支持的语言区域。 */
export const SUPPORTED_LOCALES = ['zh-CN', 'en-US', 'ja-JP'] as const
/** Taro 端支持的语言区域类型。 */
export type SupportedLocale = (typeof SUPPORTED_LOCALES)[number]
/** 单个模块的扁平语言包。 */
export type LocaleMessages = Record<string, string>
/** 翻译插值参数。 */
export type LocaleParams = Record<string, string | number>

/** Taro 语言状态。 */
export interface LocaleStoreState {
  locale: SupportedLocale
}

const DEFAULT_LOCALE: SupportedLocale = 'zh-CN'
const LOCALE_STORAGE_KEY = 'kratos-app:locale'
const localeMessages = new Map<SupportedLocale, LocaleMessages>()
const localeChangeHandlers = new Set<() => void | Promise<void>>()

/** 响应式语言 Zustand Store。 */
export const useLocaleStore = create<LocaleStoreState>(() => ({ locale: DEFAULT_LOCALE }))

/** 规范化语言区域到应用白名单。 */
export function normalizeLocale(value?: string): SupportedLocale {
  const normalized = String(value || '').replace('_', '-').toLowerCase()
  if (normalized.startsWith('ja')) return 'ja-JP'
  if (normalized.startsWith('en')) return 'en-US'
  return DEFAULT_LOCALE
}

/** 初始化持久化语言偏好。 */
export function initializeLocale(): SupportedLocale {
  const stored = Taro.getStorageSync<string>(LOCALE_STORAGE_KEY)
  const systemLanguage = Taro.getSystemInfoSync().language
  const locale = normalizeLocale(stored || systemLanguage)
  useLocaleStore.setState({ locale })
  return locale
}

/** 注册所有模块贡献的语言包并校验三语键集合。 */
export function registerLocaleMessages(modules: KratosTaroModule[]): void {
  localeMessages.clear()
  SUPPORTED_LOCALES.forEach((locale) => localeMessages.set(locale, {}))
  modules.forEach((module) => {
    const expectedKeys = Object.keys(module.messages?.[DEFAULT_LOCALE] || {}).sort()
    SUPPORTED_LOCALES.forEach((locale) => {
      const messages = module.messages?.[locale]
      if (!messages) throw new Error(`${module.name} 缺少 ${locale} 语言包`)
      const keys = Object.keys(messages).sort()
      if (keys.join('\u0000') !== expectedKeys.join('\u0000')) {
        throw new Error(`${module.name} 的 ${locale} 语言包键集合不一致`)
      }
      const target = localeMessages.get(locale) as LocaleMessages
      keys.forEach((key) => {
        if (!key.startsWith('common.') && !key.startsWith('core.') && !key.startsWith('system.')) {
          throw new Error(`${module.name} 的语言键命名空间无效: ${key}`)
        }
        if (Object.prototype.hasOwnProperty.call(target, key)) {
          throw new Error(`${locale} 语言键重复: ${key}`)
        }
        assertLocalePlaceholders(module.name, key, messages[key], module.messages?.[DEFAULT_LOCALE]?.[key] || '')
        target[key] = messages[key]
      })
    })
  })
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
    throw new Error(`${moduleName} 的 ${key} 占位符集合不一致`)
  }
}
