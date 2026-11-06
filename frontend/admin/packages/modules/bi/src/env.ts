/**
 * BI 运行环境注入：主题跟随 + 数据集 resolver + 语言。
 * 缺省零配置：主题自动跟随 html.dark；static 数据集无需 resolver。
 */
import { computed, inject, provide, ref } from "vue";
import type { ComputedRef, InjectionKey } from "vue";
import { getLocaleText, useLocaleStore } from "@liujitcn/kratos-admin-core";
import { BI_MESSAGES_EN, BI_MESSAGES_ZH } from "./bi-messages";
import { resolveTheme, watchDark } from "./bi-core";
import type { BiResolvedTheme } from "./bi-core";

export interface BiEnv {
  /** 宿主主题模式 getter（响应式）；缺省自动跟随 html.dark + 系统偏好 */
  themeMode?: () => "light" | "dark" | "auto";
}

const BI_ENV_KEY: InjectionKey<BiEnv> = Symbol("BI_ENV");

const DEFAULT_ENV: BiEnv = {};

export function provideBiEnv(env: BiEnv) {
  provide(BI_ENV_KEY, env);
}

export function useBiEnv(): BiEnv {
  return inject(BI_ENV_KEY, DEFAULT_ENV);
}

// ============ 主题：模块级单一响应式源 ============
const darkRef = ref(false);
let bound = false;

function ensureDarkBinding() {
  if (bound || typeof document === "undefined") return;
  bound = true;
  watchDark(dark => {
    darkRef.value = dark;
  });
}

/**
 * 当前图表主题（编译期注入 option）。
 * env.themeMode 缺省时跟随 html.dark + 系统偏好；提供时经 computed 追踪其响应式依赖。
 */
export function useBiTheme(): ComputedRef<BiResolvedTheme> {
  const env = useBiEnv();
  ensureDarkBinding();
  return computed(() => {
    const fallback = darkRef.value ? "dark" : "light";
    if (!env.themeMode) return resolveTheme(fallback);
    const mode = env.themeMode();
    return resolveTheme(mode === "auto" ? fallback : mode);
  });
}

/** 从嵌套的 BI 内置语言包读取回退文案。 */
function getBuiltinMessage(messages: Record<string, unknown>, key: string): string {
  const value = key.split(".").reduce<unknown>((current, segment) => {
    if (!current || typeof current !== "object") return undefined;
    return (current as Record<string, unknown>)[segment];
  }, messages);
  return typeof value === "string" ? value : "";
}

/** 对 BI 组件内置文案执行管理端风格的插值。 */
function interpolate(message: string, params: Record<string, unknown>): string {
  return message.replace(/\{([A-Za-z0-9_]+)\}/g, (_, key: string) => String(params[key] ?? `{${key}}`));
}

/** 使用管理端现有 Vue I18n，并以 BI 内置语言包作为缺省回退。 */
export function useBiI18n() {
  const { locale } = useLocaleStore();

  return {
    t(key: string, params: Record<string, unknown> = {}) {
      const currentLocale = locale.value;
      const projectMessage = getLocaleText(currentLocale, key);
      const builtinMessages = currentLocale.toLowerCase().startsWith("zh") ? BI_MESSAGES_ZH : BI_MESSAGES_EN;
      return interpolate(projectMessage || getBuiltinMessage(builtinMessages, key) || key, params);
    }
  };
}
