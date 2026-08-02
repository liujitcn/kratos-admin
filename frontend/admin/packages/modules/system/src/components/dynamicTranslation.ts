import { TranslationStatus } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_translation";
import { getEnabledBaseLanguages } from "@liujitcn/kratos-admin-system/api/system/base_language";

/** DynamicTranslationValue 动态资源单语言翻译编辑状态。 */
export interface DynamicTranslationValue {
  /** 语言代码。 */
  locale: string;
  text: string;
  translation_status: TranslationStatus;
  source_changed: boolean;
  source_hash: string;
  translation_provider: string;
  translated_at: string;
  id: number;
  reviewed_by: number;
  reviewed_at: string;
  created_by: number;
  updated_by: number;
  created_at: string;
  updated_at: string;
  deleted_at: number;
}

/** DynamicTranslationRecord 描述三类动态资源翻译共有字段。 */
export interface DynamicTranslationRecord extends Omit<DynamicTranslationValue, "text"> {
  title?: string;
  name?: string;
  label?: string;
}

/** DynamicLanguageOption 动态翻译编辑器显示的语言选项。 */
export interface DynamicLanguageOption {
  /** 语言代码。 */
  value: string;
  /** 语言显示名称。 */
  label: string;
}

/** 返回语言管理中启用且非主语言的翻译语言选项。 */
export function getEditableLanguageOptions(): DynamicLanguageOption[] {
  return getEnabledBaseLanguages()
    .filter(item => !item.is_primary)
    .map(item => ({ value: item.language_code, label: item.native_name || item.language_name || item.language_code }));
}

/** 返回语言管理中的语言显示名称。 */
export function getLanguageLabel(locale: string): string {
  const language = getEnabledBaseLanguages().find(item => item.language_code === locale);
  return language?.native_name || language?.language_name || locale;
}

/** normalizeDynamicTranslations 将后端翻译记录补齐为全部非主语言编辑状态。 */
export function normalizeDynamicTranslations(
  records: DynamicTranslationRecord[] | undefined,
  textField: "title" | "name" | "label",
  locales: string[] = getEditableLanguageOptions().map(item => item.value)
): DynamicTranslationValue[] {
  return locales.map(locale => {
    const record = records?.find(item => item.locale === locale);
    return {
      id: record?.id ?? 0,
      locale,
      text: String(record?.[textField] ?? ""),
      translation_status: record?.translation_status ?? TranslationStatus.TRANSLATION_STATUS_PENDING,
      source_changed: record?.source_changed ?? false,
      source_hash: record?.source_hash ?? "",
      translation_provider: record?.translation_provider ?? "",
      translated_at: record?.translated_at ?? "",
      reviewed_by: record?.reviewed_by ?? 0,
      reviewed_at: record?.reviewed_at ?? "",
      created_by: record?.created_by ?? 0,
      updated_by: record?.updated_by ?? 0,
      created_at: record?.created_at ?? "",
      updated_at: record?.updated_at ?? "",
      deleted_at: record?.deleted_at ?? 0
    };
  });
}

/** serializeDynamicTranslations 将编辑状态转换回后端资源翻译结构。 */
export function serializeDynamicTranslations(
  values: DynamicTranslationValue[],
  textField: "title" | "name" | "label"
): Array<{ locale: string; [key: string]: string }> {
  return values.map(({ locale, text }) => ({ locale, [textField]: text }));
}
