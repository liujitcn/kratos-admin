/* 此文件由 scripts/sync_locales.py 生成，请勿手工修改。 */
import "dayjs/locale/zh-cn";
import "dayjs/locale/en";
import "dayjs/locale/ja";
import "dayjs/locale/zh-tw";
import elementLocaleZhCn from "element-plus/es/locale/lang/zh-cn";
import elementLocaleEnUs from "element-plus/es/locale/lang/en";
import elementLocaleJaJp from "element-plus/es/locale/lang/ja";
import elementLocaleZhTw from "element-plus/es/locale/lang/zh-tw";

import localeZhCn from "./zh-CN.json";
import localeEnUs from "./en-US.json";
import localeJaJp from "./ja-JP.json";
import localeZhTw from "./zh-TW.json";

export const LOCALE_MESSAGES = {
  "zh-CN": localeZhCn,
  "en-US": localeEnUs,
  "ja-JP": localeJaJp,
  "zh-TW": localeZhTw,
} as const satisfies Record<string, Record<string, string>>;

export type GeneratedLocale = keyof typeof LOCALE_MESSAGES;
export const DEFAULT_LOCALE: GeneratedLocale = "zh-CN";
export const SUPPORTED_LOCALES = Object.keys(LOCALE_MESSAGES) as GeneratedLocale[];

export const DAYJS_LOCALE_MAP: Record<string, string> = {
  "zh-CN": "zh-cn",
  "en-US": "en",
  "ja-JP": "ja",
  "zh-TW": "zh-tw",
};

export const ELEMENT_LOCALES = {
  "zh-CN": elementLocaleZhCn,
  "en-US": elementLocaleEnUs,
  "ja-JP": elementLocaleJaJp,
  "zh-TW": elementLocaleZhTw,
} as const;
