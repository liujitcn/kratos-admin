/* 此文件由 scripts/sync_locales.py 生成，请勿手工修改。 */
import "dayjs/locale/zh-cn";
import "dayjs/locale/en";
import "dayjs/locale/es";
import "dayjs/locale/fr";
import "dayjs/locale/ja";
import "dayjs/locale/ko";
import "dayjs/locale/zh-tw";
import elementLocaleZhCn from "element-plus/es/locale/lang/zh-cn";
import elementLocaleEnUs from "element-plus/es/locale/lang/en";
import elementLocaleEsEs from "element-plus/es/locale/lang/es";
import elementLocaleFrFr from "element-plus/es/locale/lang/fr";
import elementLocaleJaJp from "element-plus/es/locale/lang/ja";
import elementLocaleKoKr from "element-plus/es/locale/lang/ko";
import elementLocaleZhTw from "element-plus/es/locale/lang/zh-tw";

import localeZhCn from "./zh-CN.json";
import localeEnUs from "./en-US.json";
import localeEsEs from "./es-ES.json";
import localeFrFr from "./fr-FR.json";
import localeJaJp from "./ja-JP.json";
import localeKoKr from "./ko-KR.json";
import localeZhTw from "./zh-TW.json";

export const LOCALE_MESSAGES = {
  "zh-CN": localeZhCn,
  "en-US": localeEnUs,
  "es-ES": localeEsEs,
  "fr-FR": localeFrFr,
  "ja-JP": localeJaJp,
  "ko-KR": localeKoKr,
  "zh-TW": localeZhTw,
} as const satisfies Record<string, Record<string, string>>;

export type GeneratedLocale = keyof typeof LOCALE_MESSAGES;
export const DEFAULT_LOCALE: GeneratedLocale = "zh-CN";
export const SUPPORTED_LOCALES = Object.keys(LOCALE_MESSAGES) as GeneratedLocale[];

export const DAYJS_LOCALE_MAP: Record<string, string> = {
  "zh-CN": "zh-cn",
  "en-US": "en",
  "es-ES": "es",
  "fr-FR": "fr",
  "ja-JP": "ja",
  "ko-KR": "ko",
  "zh-TW": "zh-tw",
};

export const ELEMENT_LOCALES = {
  "zh-CN": elementLocaleZhCn,
  "en-US": elementLocaleEnUs,
  "es-ES": elementLocaleEsEs,
  "fr-FR": elementLocaleFrFr,
  "ja-JP": elementLocaleJaJp,
  "ko-KR": elementLocaleKoKr,
  "zh-TW": elementLocaleZhTw,
} as const;
