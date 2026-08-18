/**
 * 管理端国际化运行时：合并 core 和业务模块语言包，并为 Vue I18n、组件和请求层提供统一入口。
 * 各语言 JSON 由对应模块定义文件导入，不能在 JSON 文件内添加注释。
 */
import dayjs from "dayjs";
import { computed, readonly, ref } from "vue";
import { createI18n } from "vue-i18n";
import type { AdminModule } from "@/modules";
import type { OptionLanguageResponse } from "@/rpc/base/v1/language";
import {
  DAYJS_LOCALE_MAP,
  DEFAULT_LOCALE as GENERATED_DEFAULT_LOCALE,
  SUPPORTED_LOCALES as GENERATED_SUPPORTED_LOCALES,
  type GeneratedLocale,
} from "./generated";

/** 管理端已打包的语言区域；运行时可切换列表由 base_language 接口决定。 */
export const SUPPORTED_LOCALES = GENERATED_SUPPORTED_LOCALES;
/** 管理端支持的语言区域类型。 */
export type SupportedLocale = GeneratedLocale;
/** 单个模块的扁平语言包。 */
export type LocaleMessages = Record<string, string>;
/** 翻译插值参数。 */
export type LocaleParams = Record<string, string | number>;

/** 运行时语言选项。 */
export interface LocaleOption {
  /** 标准语言代码。 */
  language_code: SupportedLocale;
  /** 本地语言名称。 */
  native_name: string;
}

export const DEFAULT_LOCALE: SupportedLocale = GENERATED_DEFAULT_LOCALE;
export const LOCALE_STORAGE_KEY = "kratos-admin:locale";

const mutableLocale = ref<SupportedLocale>(DEFAULT_LOCALE);
const mutableLanguageOptions = ref<LocaleOption[]>([]);
const localeChangeHandlers = new Set<() => void | Promise<void>>();

/** 管理端 Vue I18n 实例。 */
export const adminI18n = createI18n({
  legacy: false,
  locale: DEFAULT_LOCALE,
  fallbackLocale: DEFAULT_LOCALE,
  messages: {},
  missingWarn: false,
  fallbackWarn: false
});

/** 规范化任意语言输入到管理端白名单。 */
export function normalizeLocale(value?: string): SupportedLocale {
  return parseSupportedLocale(value) ?? DEFAULT_LOCALE;
}

/** 将接口或系统语言代码解析为已打包的语言区域。 */
function parseSupportedLocale(value?: string): SupportedLocale | undefined {
  const normalized = String(value ?? "").replace("_", "-").toLowerCase();
  if (!normalized) return undefined;
  const alias = normalized.startsWith("zh-hk") || normalized.startsWith("zh-mo") ? "zh-tw" : normalized;
  const exact = SUPPORTED_LOCALES.find(locale => locale.toLowerCase() === alias);
  if (exact) return exact;
  const language = alias.split("-", 1)[0];
  return SUPPORTED_LOCALES.find(locale => locale.toLowerCase().split("-", 1)[0] === language);
}

/** 注册模块语言包并校验语言键、占位符和命名空间。 */
export function registerLocaleMessages(modules: AdminModule[]): void {
  const merged = new Map<SupportedLocale, LocaleMessages>(SUPPORTED_LOCALES.map(locale => [locale, {}]));

  modules.forEach(module => {
    if (!module.messages) return;
    const expectedKeys = requiredLocaleKeys(module.messages[DEFAULT_LOCALE] ?? {});
    SUPPORTED_LOCALES.forEach(locale => {
      const messages = module.messages?.[locale];
      if (!messages) throw new Error(`${module.name} 缺少 ${locale} 语言包`);
      const keys = requiredLocaleKeys(messages);
      if (keys.join("\u0000") !== expectedKeys.join("\u0000")) {
        throw new Error(`${module.name} 的 ${locale} 语言包键集合不一致`);
      }

      const target = merged.get(locale) as LocaleMessages;
      Object.keys(messages).forEach(key => {
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

/** 应用后端语言配置，并在当前语言不可用时回退到接口第一项。 */
export function applyLanguageConfig(response: OptionLanguageResponse): void {
  const options = response.languages.reduce<LocaleOption[]>((items, item) => {
    const languageCode = parseSupportedLocale(item.language_code);
    if (!languageCode || items.some((option) => option.language_code === languageCode)) return items;
    items.push({
      language_code: languageCode,
      native_name: item.native_name || languageCode,
    });
    return items;
  }, []);
  mutableLanguageOptions.value = options.length ? options : getFallbackLanguageOptions();
  const availableLocales = getSupportedLocales();
  if (!availableLocales.includes(mutableLocale.value)) {
    applyLocale(availableLocales[0] ?? DEFAULT_LOCALE);
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
  return SUPPORTED_LOCALES.map(languageCode => {
    return {
      language_code: languageCode,
      native_name: fallbackNativeLanguageName(languageCode),
    };
  });
}

function fallbackNativeLanguageName(languageCode: SupportedLocale): string {
  return {
    "zh-CN": "简体中文",
    "zh-TW": "繁體中文",
    "en-US": "English",
    "ja-JP": "日本語",
  }[languageCode];
}

function applyLocale(locale: SupportedLocale): void {
  mutableLocale.value = locale;
  adminI18n.global.locale.value = locale;
  const dayjsLocale = DAYJS_LOCALE_MAP[locale];
  if (dayjsLocale) dayjs.locale(dayjsLocale);
  document.documentElement.lang = locale;
}

function assertLocaleKeyNamespace(moduleName: string, key: string): void {
  const valid =
    moduleName === "kratos-admin" ? key.startsWith("common.") || key.startsWith("core.") : key.startsWith(`${moduleName}.`);
  if (!valid) throw new Error(`${moduleName} 的语言键命名空间无效: ${key}`);
}

function requiredLocaleKeys(messages: LocaleMessages): string[] {
  return Object.keys(messages)
    .filter(key => !key.startsWith("common.language."))
    .sort();
}

function assertLocalePlaceholders(moduleName: string, key: string, message: string, sourceMessage: string): void {
  const placeholders = (value: string) => [...value.matchAll(/\{([A-Za-z0-9_]+)\}/g)].map(match => match[1]).sort();
  if (placeholders(message).join("\u0000") !== placeholders(sourceMessage).join("\u0000")) {
    throw new Error(`${moduleName} 的 ${key} 占位符集合不一致`);
  }
}
