/**
 * 管理端国际化运行时：合并 core 和业务模块语言包，并为 Vue I18n、组件和请求层提供统一入口。
 * 各语言 JSON 由对应模块定义文件导入，不能在 JSON 文件内添加注释。
 */
import dayjs from "dayjs";
import "dayjs/locale/en";
import "dayjs/locale/ja";
import "dayjs/locale/zh-cn";
import "dayjs/locale/zh-tw";
import "dayjs/locale/ko";
import "dayjs/locale/fr";
import "dayjs/locale/es";
import { computed, readonly, ref } from "vue";
import { createI18n } from "vue-i18n";
import type { AdminModule } from "@/modules";
import type { GetLanguageResponse } from "@/rpc/base/v1/language";

/** 管理端支持的语言区域。 */
export const SUPPORTED_LOCALES = ["zh-CN", "zh-TW", "en-US", "ja-JP", "ko-KR", "fr-FR", "es-ES"] as const;
/** 管理端支持的语言区域类型。 */
export type SupportedLocale = (typeof SUPPORTED_LOCALES)[number];
/** 单个模块的扁平语言包。 */
export type LocaleMessages = Record<string, string>;
/** 翻译插值参数。 */
export type LocaleParams = Record<string, string | number>;

/** 运行时语言选项。 */
export interface LocaleOption {
  /** 标准语言代码。 */
  language_code: SupportedLocale;
  /** 语言名称。 */
  language_name: string;
  /** 本地语言名称。 */
  native_name: string;
  /** 排序值。 */
  sort: number;
}

export const DEFAULT_LOCALE: SupportedLocale = "zh-CN";
export const LOCALE_STORAGE_KEY = "kratos-admin:locale";

const mutableLocale = ref<SupportedLocale>(DEFAULT_LOCALE);
const mutableLanguageOptions = ref<LocaleOption[]>([]);
const localeChangeHandlers = new Set<() => void | Promise<void>>();

/** 管理端 Vue I18n 实例。 */
export const adminI18n = createI18n({
  legacy: false,
  locale: DEFAULT_LOCALE,
  fallbackLocale: DEFAULT_LOCALE,
  flatJson: true,
  messages: {},
  missingWarn: false,
  fallbackWarn: false
});

/** 规范化任意语言输入到管理端白名单。 */
export function normalizeLocale(value?: string): SupportedLocale {
  const normalized = String(value ?? "")
    .replace("_", "-")
    .toLowerCase();
  if (normalized.startsWith("ja")) return "ja-JP";
  if (normalized.startsWith("en")) return "en-US";
  if (normalized.startsWith("zh-tw") || normalized.startsWith("zh-hk") || normalized.startsWith("zh-mo")) return "zh-TW";
  if (normalized.startsWith("ko")) return "ko-KR";
  if (normalized.startsWith("fr")) return "fr-FR";
  if (normalized.startsWith("es")) return "es-ES";
  return DEFAULT_LOCALE;
}

/** 将接口或系统语言代码解析为已打包的语言区域。 */
function parseSupportedLocale(value?: string): SupportedLocale | undefined {
  const normalized = String(value ?? "").replace("_", "-").toLowerCase();
  if (normalized.startsWith("zh-tw") || normalized.startsWith("zh-hk") || normalized.startsWith("zh-mo")) return "zh-TW";
  if (normalized.startsWith("zh")) return "zh-CN";
  if (normalized.startsWith("en")) return "en-US";
  if (normalized.startsWith("ja")) return "ja-JP";
  if (normalized.startsWith("ko")) return "ko-KR";
  if (normalized.startsWith("fr")) return "fr-FR";
  if (normalized.startsWith("es")) return "es-ES";
  return undefined;
}

/** 注册模块语言包并校验七语键、占位符和命名空间。 */
export function registerLocaleMessages(modules: AdminModule[]): void {
  const merged = new Map<SupportedLocale, LocaleMessages>(SUPPORTED_LOCALES.map(locale => [locale, {}]));

  modules.forEach(module => {
    if (!module.messages) return;
    const expectedKeys = Object.keys(module.messages[DEFAULT_LOCALE] ?? {}).sort();
    SUPPORTED_LOCALES.forEach(locale => {
      const messages = module.messages?.[locale];
      if (!messages) throw new Error(`${module.name} 缺少 ${locale} 语言包`);
      const keys = Object.keys(messages).sort();
      if (keys.join("\u0000") !== expectedKeys.join("\u0000")) {
        throw new Error(`${module.name} 的 ${locale} 语言包键集合不一致`);
      }

      const target = merged.get(locale) as LocaleMessages;
      keys.forEach(key => {
        assertLocaleKeyNamespace(module.name, key);
        if (Object.prototype.hasOwnProperty.call(target, key)) {
          throw new Error(`${locale} 语言键重复: ${key}`);
        }
        assertLocalePlaceholders(module.name, key, messages[key], module.messages?.[DEFAULT_LOCALE]?.[key] ?? "");
        target[key] = messages[key];
      });
    });
  });

  SUPPORTED_LOCALES.forEach(locale => adminI18n.global.setLocaleMessage(locale, merged.get(locale) ?? {}));
}

/** 初始化持久化语言偏好并同步日期库。 */
export function initializeLocale(): SupportedLocale {
  const stored = window.localStorage.getItem(LOCALE_STORAGE_KEY) ?? "";
  const preferred = stored || window.navigator.languages?.[0] || window.navigator.language;
  applyLocale(normalizeLocale(preferred));
  return mutableLocale.value;
}

/** 应用后端语言配置，并在当前语言不可用时回退到接口主语言。 */
export function applyLanguageConfig(response: GetLanguageResponse): void {
  const options = response.languages.reduce<LocaleOption[]>((items, item) => {
    const languageCode = parseSupportedLocale(item.language_code);
    if (!languageCode || items.some((option) => option.language_code === languageCode)) return items;
    items.push({
      language_code: languageCode,
      language_name: item.language_name || t(`common.language.${languageCode}`),
      native_name: item.native_name || item.language_name || languageCode,
      sort: item.sort,
    });
    return items;
  }, []);
  mutableLanguageOptions.value = options.length ? options.sort((left, right) => left.sort - right.sort) : getFallbackLanguageOptions();
  const availableLocales = getSupportedLocales();
  const primaryLocale = parseSupportedLocale(response.primary_language_code);
  if (!availableLocales.includes(mutableLocale.value)) {
    applyLocale(primaryLocale && availableLocales.includes(primaryLocale) ? primaryLocale : availableLocales[0] ?? DEFAULT_LOCALE);
  }
}

/** 获取当前接口配置的语言选项。 */
export function getLanguageOptions(): LocaleOption[] {
  return mutableLanguageOptions.value.length ? mutableLanguageOptions.value : getFallbackLanguageOptions();
}

/** 获取当前可切换的语言区域。 */
export function getSupportedLocales(): SupportedLocale[] {
  return getLanguageOptions().map((item) => item.language_code);
}

/** 获取当前规范语言区域。 */
export function getCurrentLocale(): SupportedLocale {
  return mutableLocale.value;
}

/** 获取非 Axios 请求统一使用的语言头。 */
export function getLocaleRequestHeaders(): Record<"Accept-Language", SupportedLocale> {
  return { "Accept-Language": getCurrentLocale() };
}

/** 切换语言并刷新动态本地化数据。 */
export async function setCurrentLocale(value: SupportedLocale): Promise<void> {
  const locale = normalizeLocale(value);
  if (!getSupportedLocales().includes(locale)) return;
  if (locale === mutableLocale.value) return;
  applyLocale(locale);
  window.localStorage.setItem(LOCALE_STORAGE_KEY, locale);
  for (const handler of localeChangeHandlers) await handler();
}

/** 注册语言切换后的动态数据刷新处理器。 */
export function registerLocaleChangeHandler(handler: () => void | Promise<void>): () => void {
  localeChangeHandlers.add(handler);
  return () => localeChangeHandlers.delete(handler);
}

/** 在非组件代码中翻译稳定语言键。 */
export function t(key: string, params: LocaleParams = {}): string {
  return String(adminI18n.global.t(key, params));
}

/** 管理端响应式语言状态。 */
export function useLocaleStore() {
  return {
    locale: readonly(mutableLocale),
    localeIndex: computed(() => Math.max(0, getSupportedLocales().indexOf(mutableLocale.value))),
    languageOptions: computed(() => getLanguageOptions()),
    supportedLocales: computed(() => getSupportedLocales()),
    setLocale: setCurrentLocale,
    t
  };
}

function getFallbackLanguageOptions(): LocaleOption[] {
  return SUPPORTED_LOCALES.map((languageCode, sort) => ({
    language_code: languageCode,
    language_name: t(`common.language.${languageCode}`),
    native_name: languageCode,
    sort,
  }));
}

function applyLocale(locale: SupportedLocale): void {
  mutableLocale.value = locale;
  adminI18n.global.locale.value = locale;
  dayjs.locale({ "zh-CN": "zh-cn", "zh-TW": "zh-tw", "en-US": "en", "ja-JP": "ja", "ko-KR": "ko", "fr-FR": "fr", "es-ES": "es" }[locale]);
  document.documentElement.lang = locale;
}

function assertLocaleKeyNamespace(moduleName: string, key: string): void {
  const valid =
    moduleName === "kratos-admin" ? key.startsWith("common.") || key.startsWith("core.") : key.startsWith(`${moduleName}.`);
  if (!valid) throw new Error(`${moduleName} 的语言键命名空间无效: ${key}`);
}

function assertLocalePlaceholders(moduleName: string, key: string, message: string, sourceMessage: string): void {
  const placeholders = (value: string) => [...value.matchAll(/\{([A-Za-z0-9_]+)\}/g)].map(match => match[1]).sort();
  if (placeholders(message).join("\u0000") !== placeholders(sourceMessage).join("\u0000")) {
    throw new Error(`${moduleName} 的 ${key} 占位符集合不一致`);
  }
}
