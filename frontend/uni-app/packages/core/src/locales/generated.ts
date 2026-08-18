/* 此文件由 scripts/sync_locales.py 生成，请勿手工修改。 */
import localeZhCn from './zh-CN.json'
import localeEnUs from './en-US.json'
import localeJaJp from './ja-JP.json'
import localeZhTw from './zh-TW.json'

export const LOCALE_MESSAGES = {
  'zh-CN': localeZhCn,
  'en-US': localeEnUs,
  'ja-JP': localeJaJp,
  'zh-TW': localeZhTw,
} as const satisfies Record<string, Record<string, string>>

export type GeneratedLocale = keyof typeof LOCALE_MESSAGES
export const DEFAULT_LOCALE: GeneratedLocale = 'zh-CN'
export const SUPPORTED_LOCALES = Object.keys(LOCALE_MESSAGES) as GeneratedLocale[]
